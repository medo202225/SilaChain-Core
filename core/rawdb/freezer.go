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
"fmt"
"math"
"os"
"path/filepath"
"sync"
"sync/atomic"

"github.com/SILA/sila-chain/ethdb"
"github.com/SILA/sila-chain/log"
"github.com/SILA/sila-chain/metrics"
"github.com/gofrs/flock"
)

var (
errReadOnly          = errors.New("read only")
errUnknownTable      = errors.New("unknown table")
errOutOrderInsertion = errors.New("the append operation is out-order")
errSymlinkDatadir    = errors.New("symbolic link datadir is not supported")
)

const freezerTableSize = 2 * 1000 * 1000 * 1000

// Freezer is an append-only database to store immutable ordered data into flat files.
type Freezer struct {
datadir string
head    atomic.Uint64
tail    atomic.Uint64

writeLock  sync.RWMutex
writeBatch *freezerBatch

readonly     bool
tables       map[string]*freezerTable
instanceLock *flock.Flock
closeOnce    sync.Once
}

// NewFreezer creates a freezer instance for maintaining immutable ordered data.
func NewFreezer(datadir string, namespace string, readonly bool, maxTableSize uint32, tables map[string]freezerTableConfig) (*Freezer, error) {
var (
readMeter  = metrics.NewRegisteredMeter(namespace+"ancient/read", nil)
writeMeter = metrics.NewRegisteredMeter(namespace+"ancient/write", nil)
sizeGauge  = metrics.NewRegisteredGauge(namespace+"ancient/size", nil)
)
if info, err := os.Lstat(datadir); !os.IsNotExist(err) {
if info == nil {
log.Warn("Could not Lstat the database", "path", datadir)
return nil, errors.New("lstat failed")
}
if info.Mode()&os.ModeSymlink != 0 {
log.Warn("Symbolic link ancient database is not supported", "path", datadir)
return nil, errSymlinkDatadir
}
}
flockFile := filepath.Join(datadir, "FLOCK")
if err := os.MkdirAll(filepath.Dir(flockFile), 0755); err != nil {
return nil, err
}
lock := flock.New(flockFile)
tryLock := lock.TryLock
if readonly {
tryLock = lock.TryRLock
}
if locked, err := tryLock(); err != nil {
return nil, err
} else if !locked {
return nil, errors.New("locking failed")
}
freezer := &Freezer{
datadir:      datadir,
readonly:     readonly,
tables:       make(map[string]*freezerTable),
instanceLock: lock,
}

for name, config := range tables {
table, err := newTable(datadir, name, readMeter, writeMeter, sizeGauge, maxTableSize, config, readonly)
if err != nil {
for _, table := range freezer.tables {
table.Close()
}
lock.Unlock()
return nil, err
}
freezer.tables[name] = table
}
var err error
if freezer.readonly {
err = freezer.validate()
} else {
err = freezer.repair()
}
if err != nil {
for _, table := range freezer.tables {
table.Close()
}
lock.Unlock()
return nil, err
}

freezer.writeBatch = newFreezerBatch(freezer)

log.Info("Opened ancient database", "database", datadir, "readonly", readonly)
return freezer, nil
}

// Close terminates the chain freezer, closing all the data files.
func (f *Freezer) Close() error {
f.writeLock.Lock()
defer f.writeLock.Unlock()

var errs []error
f.closeOnce.Do(func() {
for _, table := range f.tables {
if err := table.Close(); err != nil {
errs = append(errs, err)
}
}
if err := f.instanceLock.Unlock(); err != nil {
errs = append(errs, err)
}
})
return errors.Join(errs...)
}

// AncientDatadir returns the path of the ancient store.
func (f *Freezer) AncientDatadir() (string, error) {
return f.datadir, nil
}

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (f *Freezer) Ancient(kind string, number uint64) ([]byte, error) {
if table := f.tables[kind]; table != nil {
return table.Retrieve(number)
}
return nil, errUnknownTable
}

// AncientRange retrieves multiple items in sequence, starting from the index 'start'.
func (f *Freezer) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
if table := f.tables[kind]; table != nil {
return table.RetrieveItems(start, count, maxBytes)
}
return nil, errUnknownTable
}

// AncientBytes retrieves the value segment of the element specified by the id and value offsets.
func (f *Freezer) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
if table := f.tables[kind]; table != nil {
return table.RetrieveBytes(id, offset, length)
}
return nil, errUnknownTable
}

// Ancients returns the length of the frozen items.
func (f *Freezer) Ancients() (uint64, error) {
return f.head.Load(), nil
}

// Tail returns the number of first stored item in the freezer.
func (f *Freezer) Tail() (uint64, error) {
return f.tail.Load(), nil
}

// AncientSize returns the ancient size of the specified category.
func (f *Freezer) AncientSize(kind string) (uint64, error) {
f.writeLock.RLock()
defer f.writeLock.RUnlock()

if table := f.tables[kind]; table != nil {
return table.size()
}
return 0, errUnknownTable
}

// ReadAncients runs the given read operation while ensuring that no writes take place.
func (f *Freezer) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
f.writeLock.RLock()
defer f.writeLock.RUnlock()

return fn(f)
}

// ModifyAncients runs the given write operation.
func (f *Freezer) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (writeSize int64, err error) {
if f.readonly {
return 0, errReadOnly
}
f.writeLock.Lock()
defer f.writeLock.Unlock()

prevItem := f.head.Load()
defer func() {
if err != nil {
for name, table := range f.tables {
err := table.truncateHead(prevItem)
if err != nil {
log.Error("Freezer table roll-back failed", "table", name, "index", prevItem, "err", err)
}
}
}
}()

f.writeBatch.reset()
if err := fn(f.writeBatch); err != nil {
return 0, err
}
item, writeSize, err := f.writeBatch.commit()
if err != nil {
return 0, err
}
f.head.Store(item)
return writeSize, nil
}

// TruncateHead discards any recent data above the provided threshold number.
func (f *Freezer) TruncateHead(items uint64) (uint64, error) {
if f.readonly {
return 0, errReadOnly
}
f.writeLock.Lock()
defer f.writeLock.Unlock()

oitems := f.head.Load()
if oitems <= items {
return oitems, nil
}
for _, table := range f.tables {
if err := table.truncateHead(items); err != nil {
return 0, err
}
}
f.head.Store(items)
return oitems, nil
}

// TruncateTail discards all data below the specified threshold.
func (f *Freezer) TruncateTail(tail uint64) (uint64, error) {
if f.readonly {
return 0, errReadOnly
}
f.writeLock.Lock()
defer f.writeLock.Unlock()

old := f.tail.Load()
if old >= tail {
return old, nil
}
for _, table := range f.tables {
if table.config.prunable {
if err := table.truncateTail(tail); err != nil {
return 0, err
}
}
}
f.tail.Store(tail)

if f.head.Load() < tail {
f.head.Store(tail)
}
return old, nil
}

// SyncAncient flushes all data tables to disk.
func (f *Freezer) SyncAncient() error {
var errs []error
for _, table := range f.tables {
if err := table.Sync(); err != nil {
errs = append(errs, err)
}
}
if errs != nil {
return fmt.Errorf("%v", errs)
}
return nil
}

// validate checks that every table has the same boundary.
func (f *Freezer) validate() error {
if len(f.tables) == 0 {
return nil
}
var (
head       uint64
prunedTail *uint64
)
for _, table := range f.tables {
head = table.items.Load()
break
}
for kind, table := range f.tables {
if head != table.items.Load() {
return fmt.Errorf("freezer table %s has a differing head: %d != %d", kind, table.items.Load(), head)
}
if !table.config.prunable {
if table.itemHidden.Load() != 0 {
return fmt.Errorf("non-prunable freezer table '%s' has a non-zero tail: %d", kind, table.itemHidden.Load())
}
} else {
if prunedTail == nil {
tmp := table.itemHidden.Load()
prunedTail = &tmp
}
if *prunedTail != table.itemHidden.Load() {
return fmt.Errorf("freezer table %s has differing tail: %d != %d", kind, table.itemHidden.Load(), *prunedTail)
}
}
}

if prunedTail == nil {
tmp := uint64(0)
prunedTail = &tmp
}

f.head.Store(head)
f.tail.Store(*prunedTail)
return nil
}

// repair truncates all data tables to the same length.
func (f *Freezer) repair() error {
var (
head       = uint64(math.MaxUint64)
prunedTail = uint64(0)
)
for _, table := range f.tables {
head = min(head, table.items.Load())
prunedTail = max(prunedTail, table.itemHidden.Load())
}
for kind, table := range f.tables {
if err := table.truncateHead(head); err != nil {
return err
}
if !table.config.prunable {
if table.itemHidden.Load() != 0 {
panic(fmt.Sprintf("non-prunable freezer table %s has non-zero tail: %v", kind, table.itemHidden.Load()))
}
} else {
if err := table.truncateTail(prunedTail); err != nil {
return err
}
}
}

f.head.Store(head)
f.tail.Store(prunedTail)
return nil
}
