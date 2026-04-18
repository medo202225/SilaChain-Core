// Copyright 2026 The SILA Authors
// This file is part of the sila-library.
//
// The sila-library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila-library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila-library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/snappy"
	"silachain/common"
	"silachain/log"
	"silachain/metrics"
)

var (
	errClosed       = errors.New("closed")
	errOutOfBounds  = errors.New("out of bounds")
	errNotSupported = errors.New("this operation is not supported")
)

// indexEntry contains the number/id of the file that the data resides in, as well as the offset.
type indexEntry struct {
	filenum uint32
	offset  uint32
}

const indexEntrySize = 6

// unmarshalBinary deserializes binary b into the index entry.
func (i *indexEntry) unmarshalBinary(b []byte) {
	i.filenum = uint32(binary.BigEndian.Uint16(b[:2]))
	i.offset = binary.BigEndian.Uint32(b[2:6])
}

// append adds the encoded entry to the end of b.
func (i *indexEntry) append(b []byte) []byte {
	offset := len(b)
	out := append(b, make([]byte, indexEntrySize)...)
	binary.BigEndian.PutUint16(out[offset:], uint16(i.filenum))
	binary.BigEndian.PutUint32(out[offset+2:], i.offset)
	return out
}

// bounds returns the start- and end- offsets, and the file number.
func (i *indexEntry) bounds(end *indexEntry) (startOffset, endOffset, fileId uint32) {
	if i.filenum != end.filenum {
		return 0, end.offset, end.filenum
	}
	return i.offset, end.offset, end.filenum
}

// freezerTable represents a single chained data table within the freezer.
type freezerTable struct {
	items      atomic.Uint64
	itemOffset atomic.Uint64
	itemHidden atomic.Uint64

	config      freezerTableConfig
	readonly    bool
	maxFileSize uint32
	name        string
	path        string

	head   *os.File
	index  *os.File
	files  map[uint32]*os.File
	headId uint32
	tailId uint32

	metadata    *freezerTableMeta
	uncommitted uint64
	lastSync    time.Time

	headBytes  int64
	readMeter  *metrics.Meter
	writeMeter *metrics.Meter
	sizeGauge  *metrics.Gauge

	logger log.Logger
	lock   sync.RWMutex
}

// newFreezerTable opens the given path as a freezer table.
func newFreezerTable(path, name string, config freezerTableConfig, readonly bool) (*freezerTable, error) {
	return newTable(path, name, metrics.NewInactiveMeter(), metrics.NewInactiveMeter(), metrics.NewGauge(), freezerTableSize, config, readonly)
}

// newTable opens a freezer table, creating the data and index files if they are non-existent.
func newTable(path string, name string, readMeter, writeMeter *metrics.Meter, sizeGauge *metrics.Gauge, maxFilesize uint32, config freezerTableConfig, readonly bool) (*freezerTable, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	var idxName string
	if config.noSnappy {
		idxName = fmt.Sprintf("%s.ridx", name)
	} else {
		idxName = fmt.Sprintf("%s.cidx", name)
	}
	var (
		err   error
		index *os.File
		meta  *os.File
	)
	if readonly {
		index, err = openFreezerFileForReadOnly(filepath.Join(path, idxName))
		if err != nil {
			return nil, err
		}
		meta, err = openFreezerFileForReadOnly(filepath.Join(path, fmt.Sprintf("%s.meta", name)))
		if err != nil {
			return nil, err
		}
	} else {
		index, err = openFreezerFileForAppend(filepath.Join(path, idxName))
		if err != nil {
			return nil, err
		}
		meta, err = openFreezerFileForAppend(filepath.Join(path, fmt.Sprintf("%s.meta", name)))
		if err != nil {
			return nil, err
		}
	}
	metadata, err := newMetadata(meta)
	if err != nil {
		return nil, err
	}
	tab := &freezerTable{
		index:       index,
		metadata:    metadata,
		lastSync:    time.Now(),
		files:       make(map[uint32]*os.File),
		readMeter:   readMeter,
		writeMeter:  writeMeter,
		sizeGauge:   sizeGauge,
		name:        name,
		path:        path,
		logger:      log.New("database", path, "table", name),
		config:      config,
		readonly:    readonly,
		maxFileSize: maxFilesize,
	}
	if err := tab.repair(); err != nil {
		tab.Close()
		return nil, err
	}
	size, err := tab.sizeNolock()
	if err != nil {
		tab.Close()
		return nil, err
	}
	tab.sizeGauge.Inc(int64(size))

	return tab, nil
}

// repair cross-checks the head and the index file and truncates them to be in sync.
func (t *freezerTable) repair() error {
	buffer := make([]byte, indexEntrySize)

	stat, err := t.index.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		if _, err := t.index.Write(buffer); err != nil {
			return err
		}
	}
	if overflow := stat.Size() % indexEntrySize; overflow != 0 {
		if t.readonly {
			return fmt.Errorf("index file(path: %s, name: %s) size is not a multiple of %d", t.path, t.name, indexEntrySize)
		}
		if err := truncateFreezerFile(t.index, stat.Size()-overflow); err != nil {
			return err
		}
	}
	if err := t.repairIndex(); err != nil {
		return err
	}
	if stat, err = t.index.Stat(); err != nil {
		return err
	}
	offsetsSize := stat.Size()

	var (
		firstIndex  indexEntry
		lastIndex   indexEntry
		contentSize int64
		contentExp  int64
		verbose     bool
	)
	t.index.ReadAt(buffer, 0)
	firstIndex.unmarshalBinary(buffer)

	t.tailId = firstIndex.filenum
	t.itemOffset.Store(uint64(firstIndex.offset))

	if t.itemOffset.Load() > t.metadata.virtualTail {
		if err := t.metadata.setVirtualTail(t.itemOffset.Load(), true); err != nil {
			return err
		}
	}
	t.itemHidden.Store(t.metadata.virtualTail)

	if offsetsSize == indexEntrySize {
		lastIndex = indexEntry{filenum: t.tailId, offset: 0}
	} else {
		t.index.ReadAt(buffer, offsetsSize-indexEntrySize)
		lastIndex.unmarshalBinary(buffer)
	}
	if t.readonly {
		t.head, err = t.openFile(lastIndex.filenum, openFreezerFileForReadOnly)
	} else {
		t.head, err = t.openFile(lastIndex.filenum, openFreezerFileForAppend)
	}
	if err != nil {
		return err
	}
	if stat, err = t.head.Stat(); err != nil {
		return err
	}
	contentSize = stat.Size()

	contentExp = int64(lastIndex.offset)
	for contentExp != contentSize {
		if t.readonly {
			return fmt.Errorf("freezer table(path: %s, name: %s, num: %d) is corrupted", t.path, t.name, lastIndex.filenum)
		}
		verbose = true

		if contentExp < contentSize {
			t.logger.Warn("Truncating dangling head", "indexed", contentExp, "stored", contentSize)
			if err := truncateFreezerFile(t.head, contentExp); err != nil {
				return err
			}
			contentSize = contentExp
		}
		if contentExp > contentSize {
			t.logger.Warn("Truncating dangling indexes", "indexes", offsetsSize/indexEntrySize, "indexed", contentExp, "stored", contentSize)

			newOffset := offsetsSize - indexEntrySize
			if err := truncateFreezerFile(t.index, newOffset); err != nil {
				return err
			}
			offsetsSize -= indexEntrySize

			if t.metadata.flushOffset > newOffset {
				if err := t.metadata.setFlushOffset(newOffset, true); err != nil {
					return err
				}
			}
			var newLastIndex indexEntry
			if offsetsSize == indexEntrySize {
				newLastIndex = indexEntry{filenum: t.tailId, offset: 0}
			} else {
				t.index.ReadAt(buffer, offsetsSize-indexEntrySize)
				newLastIndex.unmarshalBinary(buffer)
			}
			if newLastIndex.filenum != lastIndex.filenum {
				t.releaseFile(lastIndex.filenum)
				if t.head, err = t.openFile(newLastIndex.filenum, openFreezerFileForAppend); err != nil {
					return err
				}
				if stat, err = t.head.Stat(); err != nil {
					return err
				}
				contentSize = stat.Size()
			}
			lastIndex = newLastIndex
			contentExp = int64(lastIndex.offset)
		}
	}
	if !t.readonly {
		if err := t.index.Sync(); err != nil {
			return err
		}
		if err := t.head.Sync(); err != nil {
			return err
		}
		if err := t.metadata.file.Sync(); err != nil {
			return err
		}
	}
	t.items.Store(t.itemOffset.Load() + uint64(offsetsSize/indexEntrySize-1))
	t.headBytes = contentSize
	t.headId = lastIndex.filenum

	t.releaseFilesAfter(t.headId, true)
	t.releaseFilesBefore(t.tailId, true)

	if err := t.preopen(); err != nil {
		return err
	}
	if verbose {
		t.logger.Info("Chain freezer table opened", "items", t.items.Load(), "deleted", t.itemOffset.Load(), "hidden", t.itemHidden.Load(), "tailId", t.tailId, "headId", t.headId, "size", t.headBytes)
	} else {
		t.logger.Debug("Chain freezer table opened", "items", t.items.Load(), "size", common.StorageSize(t.headBytes))
	}
	return nil
}

func (t *freezerTable) repairIndex() error {
	stat, err := t.index.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()

	size, err = t.checkIndex(size)
	if err != nil {
		return err
	}
	if t.metadata.version == freezerTableV1 {
		if t.readonly {
			return nil
		}
		t.logger.Info("Recovering freezer flushOffset for legacy table", "offset", size)
		return t.metadata.setFlushOffset(size, true)
	}

	switch {
	case size == indexEntrySize && t.metadata.flushOffset == 0:
		return t.metadata.setFlushOffset(size, true)

	case size == t.metadata.flushOffset:
		return nil

	case size > t.metadata.flushOffset:
		extraSize := size - t.metadata.flushOffset
		if t.readonly {
			return fmt.Errorf("index file(path: %s, name: %s) contains %d garbage data bytes", t.path, t.name, extraSize)
		}
		t.logger.Warn("Truncating freezer items after flushOffset", "size", extraSize)
		return truncateFreezerFile(t.index, t.metadata.flushOffset)

	default:
		if t.readonly {
			return nil
		}
		t.logger.Warn("Rewinding freezer flushOffset", "old", t.metadata.flushOffset, "new", size)
		return t.metadata.setFlushOffset(size, true)
	}
}

// checkIndex validates the integrity of the index file.
func (t *freezerTable) checkIndex(size int64) (int64, error) {
	_, err := t.index.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}
	fr := bufio.NewReader(t.index)

	var (
		start = time.Now()
		buff  = make([]byte, indexEntrySize)
		prev  indexEntry
		head  indexEntry

		read = func() (indexEntry, error) {
			n, err := io.ReadFull(fr, buff)
			if err != nil {
				return indexEntry{}, err
			}
			if n != indexEntrySize {
				return indexEntry{}, fmt.Errorf("failed to read from index, n: %d", n)
			}
			var entry indexEntry
			entry.unmarshalBinary(buff)
			return entry, nil
		}
		truncate = func(offset int64) (int64, error) {
			if t.readonly {
				return 0, fmt.Errorf("index file is corrupted at %d, size: %d", offset, size)
			}
			if err := truncateFreezerFile(t.index, offset); err != nil {
				return 0, err
			}
			log.Warn("Truncated index file", "offset", offset, "truncated", size-offset)
			return offset, nil
		}
	)
	for offset := int64(0); offset < size; offset += indexEntrySize {
		entry, err := read()
		if err != nil {
			return 0, err
		}
		if offset == 0 {
			head = entry
			continue
		}
		if offset == indexEntrySize {
			if entry.filenum != head.filenum && entry.filenum != head.filenum+1 {
				log.Error("Corrupted index item detected", "earliest", head.filenum, "filenumber", entry.filenum)
				return truncate(offset)
			}
			prev = entry
			continue
		}
		if err := t.checkIndexItems(prev, entry); err != nil {
			log.Error("Corrupted index item detected", "err", err)
			return truncate(offset)
		}
		prev = entry
	}
	_, err = t.index.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	log.Debug("Verified index file", "items", size/indexEntrySize, "elapsed", common.PrettyDuration(time.Since(start)))
	return size, nil
}

// checkIndexItems validates the correctness of two consecutive index items.
func (t *freezerTable) checkIndexItems(a, b indexEntry) error {
	if b.filenum != a.filenum && b.filenum != a.filenum+1 {
		return fmt.Errorf("index items with inconsistent file number, prev: %d, next: %d", a.filenum, b.filenum)
	}
	if b.filenum == a.filenum && b.offset < a.offset {
		return fmt.Errorf("index items with unordered offset, prev: %d, next: %d", a.offset, b.offset)
	}
	if b.filenum == a.filenum+1 && b.offset == 0 {
		return fmt.Errorf("index items with zero offset, file number: %d", b.filenum)
	}
	return nil
}

// preopen opens all files that the freezer will need.
func (t *freezerTable) preopen() (err error) {
	t.releaseFilesAfter(0, false)

	for i := t.tailId; i < t.headId; i++ {
		if _, err = t.openFile(i, openFreezerFileForReadOnly); err != nil {
			return err
		}
	}
	if t.readonly {
		t.head, err = t.openFile(t.headId, openFreezerFileForReadOnly)
	} else {
		t.head, err = t.openFile(t.headId, openFreezerFileForAppend)
	}
	return err
}

// truncateHead discards any recent data above the provided threshold number.
func (t *freezerTable) truncateHead(items uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	existing := t.items.Load()
	if existing <= items {
		return nil
	}
	if items < t.itemHidden.Load() {
		return errors.New("truncation below tail")
	}
	oldSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	log := t.logger.Debug
	if existing > items+1 {
		log = t.logger.Warn
	}
	log("Truncating freezer table", "items", existing, "limit", items)

	length := items - t.itemOffset.Load()
	newOffset := (length + 1) * indexEntrySize
	if err := truncateFreezerFile(t.index, int64(newOffset)); err != nil {
		return err
	}
	if err := t.index.Sync(); err != nil {
		return err
	}
	if t.metadata.flushOffset > int64(newOffset) {
		if err := t.metadata.setFlushOffset(int64(newOffset), true); err != nil {
			return err
		}
	}
	var expected indexEntry
	if length == 0 {
		expected = indexEntry{filenum: t.tailId, offset: 0}
	} else {
		buffer := make([]byte, indexEntrySize)
		if _, err := t.index.ReadAt(buffer, int64(length*indexEntrySize)); err != nil {
			return err
		}
		expected.unmarshalBinary(buffer)
	}
	if expected.filenum != t.headId {
		t.releaseFile(expected.filenum)
		newHead, err := t.openFile(expected.filenum, openFreezerFileForAppend)
		if err != nil {
			return err
		}
		t.releaseFilesAfter(expected.filenum, true)

		t.head = newHead
		t.headId = expected.filenum
	}
	if err := truncateFreezerFile(t.head, int64(expected.offset)); err != nil {
		return err
	}
	if err := t.head.Sync(); err != nil {
		return err
	}
	t.headBytes = int64(expected.offset)
	t.items.Store(items)

	newSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	t.sizeGauge.Dec(int64(oldSize - newSize))
	return nil
}

// sizeHidden returns the total data size of hidden items in the freezer table.
func (t *freezerTable) sizeHidden() (uint64, error) {
	hidden, offset := t.itemHidden.Load(), t.itemOffset.Load()
	if hidden <= offset {
		return 0, nil
	}
	indices, err := t.getIndices(hidden-1, 1)
	if err != nil {
		return 0, err
	}
	return uint64(indices[1].offset), nil
}

// truncateTail discards any recent data before the provided threshold number.
func (t *freezerTable) truncateTail(items uint64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.itemHidden.Load() >= items {
		return nil
	}
	if t.items.Load() < items {
		return t.resetTo(items)
	}
	var (
		newTailId uint32
		buffer    = make([]byte, indexEntrySize)
	)
	if t.items.Load() == items {
		newTailId = t.headId
	} else {
		offset := items - t.itemOffset.Load()
		if _, err := t.index.ReadAt(buffer, int64((offset+1)*indexEntrySize)); err != nil {
			return err
		}
		var newTail indexEntry
		newTail.unmarshalBinary(buffer)
		newTailId = newTail.filenum
	}
	oldSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	t.itemHidden.Store(items)

	if err := t.metadata.setVirtualTail(items, false); err != nil {
		return err
	}
	if t.tailId == newTailId {
		return nil
	}
	if t.tailId > newTailId {
		return fmt.Errorf("invalid index, tail-file %d, item-file %d", t.tailId, newTailId)
	}
	if err := t.doSync(); err != nil {
		return err
	}
	var (
		newDeleted = items
		deleted    = t.itemOffset.Load()
	)
	for current := items - 1; current >= deleted; current -= 1 {
		if _, err := t.index.ReadAt(buffer, int64((current-deleted+1)*indexEntrySize)); err != nil {
			return err
		}
		var pre indexEntry
		pre.unmarshalBinary(buffer)
		if pre.filenum != newTailId {
			break
		}
		newDeleted = current
	}
	if err := t.index.Close(); err != nil {
		return err
	}
	err = copyFrom(t.index.Name(), t.index.Name(), indexEntrySize*(newDeleted-deleted+1), func(f *os.File) error {
		tailIndex := indexEntry{
			filenum: newTailId,
			offset:  uint32(newDeleted),
		}
		_, err := f.Write(tailIndex.append(nil))
		return err
	})
	if err != nil {
		return err
	}
	t.index, err = openFreezerFileForAppend(t.index.Name())
	if err != nil {
		return err
	}
	if err := t.index.Sync(); err != nil {
		return err
	}
	t.tailId = newTailId
	t.itemOffset.Store(newDeleted)
	t.releaseFilesBefore(t.tailId, true)

	shorten := indexEntrySize * int64(newDeleted-deleted)
	if t.metadata.flushOffset <= shorten {
		return fmt.Errorf("invalid index flush offset: %d, shorten: %d", t.metadata.flushOffset, shorten)
	}
	if err := t.metadata.setFlushOffset(t.metadata.flushOffset-shorten, true); err != nil {
		return err
	}
	newSize, err := t.sizeNolock()
	if err != nil {
		return err
	}
	t.sizeGauge.Dec(int64(oldSize - newSize))
	return nil
}

// resetTo clears the entire table and sets both the head and tail to the given value.
func (t *freezerTable) resetTo(tail uint64) error {
	err := t.doSync()
	if err != nil {
		return err
	}
	if err := t.index.Close(); err != nil {
		return err
	}
	entry := &indexEntry{
		filenum: t.headId + 1,
		offset:  uint32(tail),
	}
	if err := reset(t.index.Name(), entry.append(nil)); err != nil {
		return err
	}
	if err := t.metadata.setVirtualTail(tail, true); err != nil {
		return err
	}
	if err := t.metadata.setFlushOffset(indexEntrySize, true); err != nil {
		return err
	}
	t.index, err = openFreezerFileForAppend(t.index.Name())
	if err != nil {
		return err
	}

	if err := t.head.Close(); err != nil {
		return err
	}
	t.headId = t.headId + 1
	t.tailId = t.headId
	t.headBytes = 0

	t.head, err = t.openFile(t.headId, openFreezerFileTruncated)
	if err != nil {
		return err
	}
	t.releaseFilesBefore(t.headId, true)

	t.items.Store(tail)
	t.itemOffset.Store(tail)
	t.itemHidden.Store(tail)
	t.sizeGauge.Update(0)

	return nil
}

// Close closes all opened files.
func (t *freezerTable) Close() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if err := t.doSync(); err != nil {
		return err
	}
	var errs []error
	doClose := func(f *os.File) {
		if err := f.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	doClose(t.index)
	doClose(t.metadata.file)
	for _, f := range t.files {
		doClose(f)
	}
	t.index = nil
	t.head = nil
	t.metadata.file = nil

	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// openFile assumes that the write-lock is held by the caller.
func (t *freezerTable) openFile(num uint32, opener func(string) (*os.File, error)) (f *os.File, err error) {
	var exist bool
	if f, exist = t.files[num]; !exist {
		var name string
		if t.config.noSnappy {
			name = fmt.Sprintf("%s.%04d.rdat", t.name, num)
		} else {
			name = fmt.Sprintf("%s.%04d.cdat", t.name, num)
		}
		f, err = opener(filepath.Join(t.path, name))
		if err != nil {
			return nil, err
		}
		t.files[num] = f
	}
	return f, err
}

// releaseFile closes a file and removes it from the open file cache.
func (t *freezerTable) releaseFile(num uint32) {
	if f, exist := t.files[num]; exist {
		delete(t.files, num)
		f.Close()
	}
}

// releaseFilesAfter closes all open files with a higher number.
func (t *freezerTable) releaseFilesAfter(num uint32, remove bool) {
	for fnum, f := range t.files {
		if fnum > num {
			delete(t.files, fnum)
			f.Close()
			if remove {
				os.Remove(f.Name())
			}
		}
	}
}

// releaseFilesBefore closes all open files with a lower number.
func (t *freezerTable) releaseFilesBefore(num uint32, remove bool) {
	for fnum, f := range t.files {
		if fnum < num {
			delete(t.files, fnum)
			f.Close()
			if remove {
				os.Remove(f.Name())
			}
		}
	}
}

// getIndices returns the index entries for the given from-item, covering 'count' items.
func (t *freezerTable) getIndices(from, count uint64) ([]*indexEntry, error) {
	from = from - t.itemOffset.Load()

	buffer := make([]byte, (count+1)*indexEntrySize)
	if _, err := t.index.ReadAt(buffer, int64(from*indexEntrySize)); err != nil {
		return nil, err
	}
	var (
		indices []*indexEntry
		offset  int
	)
	for i := from; i <= from+count; i++ {
		index := new(indexEntry)
		index.unmarshalBinary(buffer[offset:])
		offset += indexEntrySize
		indices = append(indices, index)
	}
	if from == 0 {
		indices[0].offset = 0
		indices[0].filenum = indices[1].filenum
	}
	return indices, nil
}

// Retrieve looks up the data offset of an item with the given number.
func (t *freezerTable) Retrieve(item uint64) ([]byte, error) {
	items, err := t.RetrieveItems(item, 1, 0)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

// RetrieveItems returns multiple items in sequence.
func (t *freezerTable) RetrieveItems(start, count, maxBytes uint64) ([][]byte, error) {
	diskData, sizes, err := t.retrieveItems(start, count, maxBytes)
	if err != nil {
		return nil, err
	}
	var (
		output     = make([][]byte, 0, count)
		offset     int
		outputSize int
	)
	for i, diskSize := range sizes {
		item := diskData[offset : offset+diskSize]
		offset += diskSize
		decompressedSize := diskSize
		if !t.config.noSnappy {
			decompressedSize, _ = snappy.DecodedLen(item)
		}
		if i > 0 && maxBytes != 0 && uint64(outputSize+decompressedSize) > maxBytes {
			break
		}
		if !t.config.noSnappy {
			data, err := snappy.Decode(nil, item)
			if err != nil {
				return nil, err
			}
			output = append(output, data)
		} else {
			output = append(output, item)
		}
		outputSize += decompressedSize
	}
	return output, nil
}

// retrieveItems reads up to 'count' items from the table.
func (t *freezerTable) retrieveItems(start, count, maxBytes uint64) ([]byte, []int, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if t.index == nil || t.head == nil || t.metadata.file == nil {
		return nil, nil, errClosed
	}
	var (
		items  = t.items.Load()
		hidden = t.itemHidden.Load()
	)
	if items <= start || hidden > start || count == 0 {
		return nil, nil, errOutOfBounds
	}
	if start+count > items {
		count = items - start
	}
	var output []byte
	if maxBytes != 0 {
		output = make([]byte, 0, maxBytes)
	} else {
		output = make([]byte, 0, 1024)
	}
	readData := func(fileId, start uint32, length int) error {
		output = grow(output, length)
		dataFile, exist := t.files[fileId]
		if !exist {
			return fmt.Errorf("missing data file %d", fileId)
		}
		if _, err := dataFile.ReadAt(output[len(output)-length:], int64(start)); err != nil {
			return fmt.Errorf("%w, fileid: %d, start: %d, length: %d", err, fileId, start, length)
		}
		return nil
	}
	indices, err := t.getIndices(start, count)
	if err != nil {
		return nil, nil, err
	}
	var (
		sizes      []int
		totalSize  = 0
		readStart  = indices[0].offset
		unreadSize = 0
	)

	for i, firstIndex := range indices[:len(indices)-1] {
		secondIndex := indices[i+1]
		offset1, offset2, _ := firstIndex.bounds(secondIndex)
		size := int(offset2 - offset1)
		if secondIndex.filenum != firstIndex.filenum {
			if unreadSize > 0 {
				if err := readData(firstIndex.filenum, readStart, unreadSize); err != nil {
					return nil, nil, err
				}
				unreadSize = 0
			}
			readStart = 0
		}
		if i > 0 && uint64(totalSize+size) > maxBytes && maxBytes != 0 {
			if unreadSize > 0 {
				if err := readData(secondIndex.filenum, readStart, unreadSize); err != nil {
					return nil, nil, err
				}
			}
			break
		}
		unreadSize += size
		totalSize += size
		sizes = append(sizes, size)
		if i == len(indices)-2 || (uint64(totalSize) > maxBytes && maxBytes != 0) {
			if err := readData(secondIndex.filenum, readStart, unreadSize); err != nil {
				return nil, nil, err
			}
			break
		}
	}

	t.readMeter.Mark(int64(totalSize))
	return output, sizes, nil
}

// RetrieveBytes retrieves the value segment of the element.
func (t *freezerTable) RetrieveBytes(item, offset, length uint64) ([]byte, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if t.index == nil || t.head == nil || t.metadata.file == nil {
		return nil, errClosed
	}
	items, hidden := t.items.Load(), t.itemHidden.Load()
	if items <= item || hidden > item {
		return nil, errOutOfBounds
	}

	indices, err := t.getIndices(item, 1)
	if err != nil {
		return nil, err
	}
	index0, index1 := indices[0], indices[1]

	itemStart, itemLimit, fileId := index0.bounds(index1)
	itemSize := itemLimit - itemStart

	dataFile, exist := t.files[fileId]
	if !exist {
		return nil, fmt.Errorf("missing data file %d", fileId)
	}

	if t.config.noSnappy {
		if offset > uint64(itemSize) || offset+length > uint64(itemSize) {
			return nil, fmt.Errorf("requested range out of bounds: item size %d, offset %d, length %d", itemSize, offset, length)
		}
		itemStart += uint32(offset)

		buf := make([]byte, length)
		_, err = dataFile.ReadAt(buf, int64(itemStart))
		if err != nil {
			return nil, err
		}
		t.readMeter.Mark(int64(length))
		return buf, nil
	} else {
		buf := make([]byte, itemSize)
		_, err = dataFile.ReadAt(buf, int64(itemStart))
		if err != nil {
			return nil, err
		}
		t.readMeter.Mark(int64(itemSize))

		data, err := snappy.Decode(nil, buf)
		if err != nil {
			return nil, err
		}
		if offset > uint64(len(data)) || offset+length > uint64(len(data)) {
			return nil, fmt.Errorf("requested range out of bounds: item size %d, offset %d, length %d", len(data), offset, length)
		}
		return data[offset : offset+length], nil
	}
}

// size returns the total data size in the freezer table.
func (t *freezerTable) size() (uint64, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.sizeNolock()
}

// sizeNolock returns the total data size in the freezer table.
func (t *freezerTable) sizeNolock() (uint64, error) {
	stat, err := t.index.Stat()
	if err != nil {
		return 0, err
	}
	hidden, err := t.sizeHidden()
	if err != nil {
		return 0, err
	}
	total := uint64(t.maxFileSize)*uint64(t.headId-t.tailId) + uint64(t.headBytes) + uint64(stat.Size()) - hidden
	return total, nil
}

// advanceHead should be called when the current head file would outgrow the file limits.
func (t *freezerTable) advanceHead() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if err := t.doSync(); err != nil {
		return err
	}
	nextID := t.headId + 1
	newHead, err := t.openFile(nextID, openFreezerFileTruncated)
	if err != nil {
		return err
	}
	if err := t.head.Sync(); err != nil {
		return err
	}
	t.releaseFile(t.headId)
	if _, err := t.openFile(t.headId, openFreezerFileForReadOnly); err != nil {
		return err
	}

	t.head = newHead
	t.headBytes = 0
	t.headId = nextID
	return nil
}

// Sync pushes any pending data from memory out to disk.
func (t *freezerTable) Sync() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.doSync()
}

// doSync is the internal version of Sync.
func (t *freezerTable) doSync() error {
	if t.readonly {
		return nil
	}
	if t.index == nil || t.head == nil || t.metadata.file == nil {
		return errClosed
	}
	if err := t.index.Sync(); err != nil {
		return err
	}
	if err := t.head.Sync(); err != nil {
		return err
	}
	stat, err := t.index.Stat()
	if err != nil {
		return err
	}
	return t.metadata.setFlushOffset(stat.Size(), true)
}

func (t *freezerTable) dumpIndexStdout(start, stop int64) {
	t.dumpIndex(os.Stdout, start, stop)
}

func (t *freezerTable) dumpIndexString(start, stop int64) string {
	var out bytes.Buffer
	out.WriteString("\n")
	t.dumpIndex(&out, start, stop)
	return out.String()
}

func (t *freezerTable) dumpIndex(w io.Writer, start, stop int64) {
	fmt.Fprintf(w, "Version %d count %d, deleted %d, hidden %d\n",
		t.metadata.version, t.items.Load(), t.itemOffset.Load(), t.itemHidden.Load())

	buf := make([]byte, indexEntrySize)

	fmt.Fprintf(w, "| number | fileno | offset |\n")
	fmt.Fprintf(w, "|--------|--------|--------|\n")

	for i := uint64(start); ; i++ {
		if _, err := t.index.ReadAt(buf, int64((i+1)*indexEntrySize)); err != nil {
			break
		}
		var entry indexEntry
		entry.unmarshalBinary(buf)
		fmt.Fprintf(w, "|  %03d   |  %03d   |  %03d   | \n", i, entry.filenum, entry.offset)
		if stop > 0 && i >= uint64(stop) {
			break
		}
	}
	fmt.Fprintf(w, "|--------------------------|\n")
}
