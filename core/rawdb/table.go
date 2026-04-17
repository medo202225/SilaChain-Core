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
"github.com/SILA/sila-chain/ethdb"
)

// table is a wrapper around a database that prefixes each key access with a pre-configured string.
type table struct {
db     ethdb.Database
prefix string
}

// NewTable returns a database object that prefixes all keys with a given string.
func NewTable(db ethdb.Database, prefix string) ethdb.Database {
return &table{
db:     db,
prefix: prefix,
}
}

// Close is a noop to implement the Database interface.
func (t *table) Close() error {
return nil
}

// Has retrieves if a prefixed version of a key is present in the database.
func (t *table) Has(key []byte) (bool, error) {
return t.db.Has(append([]byte(t.prefix), key...))
}

// Get retrieves the given prefixed key if it's present in the database.
func (t *table) Get(key []byte) ([]byte, error) {
return t.db.Get(append([]byte(t.prefix), key...))
}

// Ancient is a noop passthrough.
func (t *table) Ancient(kind string, number uint64) ([]byte, error) {
return t.db.Ancient(kind, number)
}

// AncientRange is a noop passthrough.
func (t *table) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
return t.db.AncientRange(kind, start, count, maxBytes)
}

// AncientBytes is a noop passthrough.
func (t *table) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
return t.db.AncientBytes(kind, id, offset, length)
}

// Ancients is a noop passthrough.
func (t *table) Ancients() (uint64, error) {
return t.db.Ancients()
}

// Tail is a noop passthrough.
func (t *table) Tail() (uint64, error) {
return t.db.Tail()
}

// AncientSize is a noop passthrough.
func (t *table) AncientSize(kind string) (uint64, error) {
return t.db.AncientSize(kind)
}

// ModifyAncients runs an ancient write operation.
func (t *table) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
return t.db.ModifyAncients(fn)
}

func (t *table) ReadAncients(fn func(reader ethdb.AncientReaderOp) error) (err error) {
return t.db.ReadAncients(fn)
}

// TruncateHead is a noop passthrough.
func (t *table) TruncateHead(items uint64) (uint64, error) {
return t.db.TruncateHead(items)
}

// TruncateTail is a noop passthrough.
func (t *table) TruncateTail(items uint64) (uint64, error) {
return t.db.TruncateTail(items)
}

// SyncAncient is a noop passthrough.
func (t *table) SyncAncient() error {
return t.db.SyncAncient()
}

// AncientDatadir returns the ancient datadir.
func (t *table) AncientDatadir() (string, error) {
return t.db.AncientDatadir()
}

// Put inserts the given value into the database.
func (t *table) Put(key []byte, value []byte) error {
return t.db.Put(append([]byte(t.prefix), key...), value)
}

// Delete removes the given prefixed key from the database.
func (t *table) Delete(key []byte) error {
return t.db.Delete(append([]byte(t.prefix), key...))
}

// DeleteRange deletes all of the keys in the range [start,end).
func (t *table) DeleteRange(start, end []byte) error {
if end == nil {
end = ethdb.MaximumKey
}
return t.db.DeleteRange(append([]byte(t.prefix), start...), append([]byte(t.prefix), end...))
}

// NewIterator creates a binary-alphabetical iterator.
func (t *table) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
innerPrefix := append([]byte(t.prefix), prefix...)
iter := t.db.NewIterator(innerPrefix, start)
return &tableIterator{
iter:   iter,
prefix: t.prefix,
}
}

// Stat returns the statistic data of the database.
func (t *table) Stat() (string, error) {
return t.db.Stat()
}

// Compact flattens the underlying data store for the given key range.
func (t *table) Compact(start []byte, limit []byte) error {
if start == nil {
start = []byte(t.prefix)
} else {
start = append([]byte(t.prefix), start...)
}
if limit == nil {
limit = []byte(t.prefix)
for i := len(limit) - 1; i >= 0; i-- {
limit[i]++
if limit[i] > 0 {
break
}
if i == 0 {
limit = nil
}
}
} else {
limit = append([]byte(t.prefix), limit...)
}
return t.db.Compact(start, limit)
}

// SyncKeyValue ensures that all pending writes are flushed to disk.
func (t *table) SyncKeyValue() error {
return t.db.SyncKeyValue()
}

// NewBatch creates a write-only database.
func (t *table) NewBatch() ethdb.Batch {
return &tableBatch{t.db.NewBatch(), t.prefix}
}

// NewBatchWithSize creates a write-only database batch with pre-allocated buffer.
func (t *table) NewBatchWithSize(size int) ethdb.Batch {
return &tableBatch{t.db.NewBatchWithSize(size), t.prefix}
}

// tableBatch is a wrapper around a database batch.
type tableBatch struct {
batch  ethdb.Batch
prefix string
}

// Put inserts the given value into the batch.
func (b *tableBatch) Put(key, value []byte) error {
return b.batch.Put(append([]byte(b.prefix), key...), value)
}

// Delete inserts a key removal into the batch.
func (b *tableBatch) Delete(key []byte) error {
return b.batch.Delete(append([]byte(b.prefix), key...))
}

// DeleteRange removes all keys in the range [start, end).
func (b *tableBatch) DeleteRange(start, end []byte) error {
if end == nil {
end = ethdb.MaximumKey
}
return b.batch.DeleteRange(append([]byte(b.prefix), start...), append([]byte(b.prefix), end...))
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *tableBatch) ValueSize() int {
return b.batch.ValueSize()
}

// Write flushes any accumulated data to disk.
func (b *tableBatch) Write() error {
return b.batch.Write()
}

// Reset resets the batch for reuse.
func (b *tableBatch) Reset() {
b.batch.Reset()
}

// Close closes the batch.
func (b *tableBatch) Close() {
b.batch.Close()
}

// tableReplayer is a wrapper around a batch replayer.
type tableReplayer struct {
w      ethdb.KeyValueWriter
prefix string
}

// Put implements the interface KeyValueWriter.
func (r *tableReplayer) Put(key []byte, value []byte) error {
trimmed := key[len(r.prefix):]
return r.w.Put(trimmed, value)
}

// Delete implements the interface KeyValueWriter.
func (r *tableReplayer) Delete(key []byte) error {
trimmed := key[len(r.prefix):]
return r.w.Delete(trimmed)
}

// Replay replays the batch contents.
func (b *tableBatch) Replay(w ethdb.KeyValueWriter) error {
return b.batch.Replay(&tableReplayer{w: w, prefix: b.prefix})
}

// tableIterator is a wrapper around a database iterator.
type tableIterator struct {
iter   ethdb.Iterator
prefix string
}

// Next moves the iterator to the next key/value pair.
func (iter *tableIterator) Next() bool {
return iter.iter.Next()
}

// Error returns any accumulated error.
func (iter *tableIterator) Error() error {
return iter.iter.Error()
}

// Key returns the key of the current key/value pair.
func (iter *tableIterator) Key() []byte {
key := iter.iter.Key()
if key == nil {
return nil
}
return key[len(iter.prefix):]
}

// Value returns the value of the current key/value pair.
func (iter *tableIterator) Value() []byte {
return iter.iter.Value()
}

// Release releases associated resources.
func (iter *tableIterator) Release() {
iter.iter.Release()
}
