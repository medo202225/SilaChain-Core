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
	"errors"
	"io"
	"math"
	"os"

	"silachain/log"
	"silachain/rlp"
)

const (
	freezerTableV1 = 1
	freezerTableV2 = 2
	freezerVersion = freezerTableV2
)

// freezerTableMeta is a collection of additional properties that describe the freezer table.
type freezerTableMeta struct {
	file        *os.File
	version     uint16
	virtualTail uint64
	flushOffset int64
}

// decodeV1 attempts to decode the metadata structure in v1 format.
func decodeV1(file *os.File) *freezerTableMeta {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil
	}
	type obj struct {
		Version uint16
		Tail    uint64
	}
	var o obj
	if err := rlp.Decode(file, &o); err != nil {
		return nil
	}
	if o.Version != freezerTableV1 {
		return nil
	}
	return &freezerTableMeta{
		file:        file,
		version:     o.Version,
		virtualTail: o.Tail,
	}
}

// decodeV2 attempts to decode the metadata structure in v2 format.
func decodeV2(file *os.File) *freezerTableMeta {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil
	}
	type obj struct {
		Version uint16
		Tail    uint64
		Offset  uint64
	}
	var o obj
	if err := rlp.Decode(file, &o); err != nil {
		return nil
	}
	if o.Version != freezerTableV2 {
		return nil
	}
	if o.Offset > math.MaxInt64 {
		log.Error("Invalid flushOffset %d in freezer metadata", o.Offset, "file", file.Name())
		return nil
	}
	return &freezerTableMeta{
		file:        file,
		version:     freezerTableV2,
		virtualTail: o.Tail,
		flushOffset: int64(o.Offset),
	}
}

// newMetadata initializes the metadata object.
func newMetadata(file *os.File) (*freezerTableMeta, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() == 0 {
		m := &freezerTableMeta{
			file:        file,
			version:     freezerTableV2,
			virtualTail: 0,
			flushOffset: 0,
		}
		if err := m.write(true); err != nil {
			return nil, err
		}
		return m, nil
	}
	if m := decodeV2(file); m != nil {
		return m, nil
	}
	if m := decodeV1(file); m != nil {
		return m, nil
	}
	return nil, errors.New("failed to decode metadata")
}

// setVirtualTail sets the virtual tail and flushes the metadata if sync is true.
func (m *freezerTableMeta) setVirtualTail(tail uint64, sync bool) error {
	m.virtualTail = tail
	return m.write(sync)
}

// setFlushOffset sets the flush offset and flushes the metadata if sync is true.
func (m *freezerTableMeta) setFlushOffset(offset int64, sync bool) error {
	m.flushOffset = offset
	return m.write(sync)
}

// write flushes the content of metadata into file.
func (m *freezerTableMeta) write(sync bool) error {
	type obj struct {
		Version uint16
		Tail    uint64
		Offset  uint64
	}
	var o obj
	o.Version = freezerVersion
	o.Tail = m.virtualTail
	o.Offset = uint64(m.flushOffset)

	_, err := m.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	if err := rlp.Encode(m.file, &o); err != nil {
		return err
	}
	if !sync {
		return nil
	}
	return m.file.Sync()
}
