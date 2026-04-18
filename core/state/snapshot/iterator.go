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

package snapshot

import (
	"bytes"
	"fmt"
	"sort"

	"silachain/common"
	"silachain/core/rawdb"
	"silachain/ethdb"
)

// Iterator is an iterator to step over all the accounts or the specific
// storage in a snapshot which may or may not be composed of multiple layers.
type Iterator interface {
	Next() bool
	Error() error
	Hash() common.Hash
	Release()
}

// AccountIterator is an iterator to step over all the accounts in a snapshot.
type AccountIterator interface {
	Iterator
	Account() []byte
}

// StorageIterator is an iterator to step over the specific storage in a snapshot.
type StorageIterator interface {
	Iterator
	Slot() []byte
}

// diffAccountIterator is an account iterator that steps over the accounts
// contained within a single diff layer.
type diffAccountIterator struct {
	curHash common.Hash
	layer   *diffLayer
	keys    []common.Hash
	fail    error
}

// AccountIterator creates an account iterator over a single diff layer.
func (dl *diffLayer) AccountIterator(seek common.Hash) AccountIterator {
	hashes := dl.AccountList()
	index := sort.Search(len(hashes), func(i int) bool {
		return bytes.Compare(seek[:], hashes[i][:]) <= 0
	})
	return &diffAccountIterator{
		layer: dl,
		keys:  hashes[index:],
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diffAccountIterator) Next() bool {
	if it.fail != nil {
		panic(fmt.Sprintf("called Next of failed iterator: %v", it.fail))
	}
	if len(it.keys) == 0 {
		return false
	}
	if it.layer.Stale() {
		it.fail, it.keys = ErrSnapshotStale, nil
		return false
	}
	it.curHash = it.keys[0]
	it.keys = it.keys[1:]
	return true
}

// Error returns any failure that occurred during iteration.
func (it *diffAccountIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the account the iterator is currently at.
func (it *diffAccountIterator) Hash() common.Hash {
	return it.curHash
}

// Account returns the RLP encoded slim account the iterator is currently at.
func (it *diffAccountIterator) Account() []byte {
	it.layer.lock.RLock()
	blob, ok := it.layer.accountData[it.curHash]
	if !ok {
		panic(fmt.Sprintf("iterator referenced non-existent account: %x", it.curHash))
	}
	it.layer.lock.RUnlock()
	if it.layer.Stale() {
		it.fail, it.keys = ErrSnapshotStale, nil
	}
	return blob
}

// Release is a noop for diff account iterators.
func (it *diffAccountIterator) Release() {}

// diskAccountIterator is an account iterator that steps over the live accounts
// contained within a disk layer.
type diskAccountIterator struct {
	layer *diskLayer
	it    ethdb.Iterator
}

// AccountIterator creates an account iterator over a disk layer.
func (dl *diskLayer) AccountIterator(seek common.Hash) AccountIterator {
	pos := common.TrimRightZeroes(seek[:])
	return &diskAccountIterator{
		layer: dl,
		it:    dl.diskdb.NewIterator(rawdb.SnapshotAccountPrefix, pos),
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diskAccountIterator) Next() bool {
	if it.it == nil {
		return false
	}
	for {
		if !it.it.Next() {
			it.it.Release()
			it.it = nil
			return false
		}
		if len(it.it.Key()) == len(rawdb.SnapshotAccountPrefix)+common.HashLength {
			break
		}
	}
	return true
}

// Error returns any failure that occurred during iteration.
func (it *diskAccountIterator) Error() error {
	if it.it == nil {
		return nil
	}
	return it.it.Error()
}

// Hash returns the hash of the account the iterator is currently at.
func (it *diskAccountIterator) Hash() common.Hash {
	return common.BytesToHash(it.it.Key())
}

// Account returns the RLP encoded slim account the iterator is currently at.
func (it *diskAccountIterator) Account() []byte {
	return it.it.Value()
}

// Release releases the database snapshot held during iteration.
func (it *diskAccountIterator) Release() {
	if it.it != nil {
		it.it.Release()
		it.it = nil
	}
}

// diffStorageIterator is a storage iterator that steps over the specific storage
// contained within a single diff layer.
type diffStorageIterator struct {
	curHash common.Hash
	account common.Hash
	layer   *diffLayer
	keys    []common.Hash
	fail    error
}

// StorageIterator creates a storage iterator over a single diff layer.
func (dl *diffLayer) StorageIterator(account common.Hash, seek common.Hash) StorageIterator {
	hashes := dl.StorageList(account)
	index := sort.Search(len(hashes), func(i int) bool {
		return bytes.Compare(seek[:], hashes[i][:]) <= 0
	})
	return &diffStorageIterator{
		layer:   dl,
		account: account,
		keys:    hashes[index:],
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diffStorageIterator) Next() bool {
	if it.fail != nil {
		panic(fmt.Sprintf("called Next of failed iterator: %v", it.fail))
	}
	if len(it.keys) == 0 {
		return false
	}
	if it.layer.Stale() {
		it.fail, it.keys = ErrSnapshotStale, nil
		return false
	}
	it.curHash = it.keys[0]
	it.keys = it.keys[1:]
	return true
}

// Error returns any failure that occurred during iteration.
func (it *diffStorageIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the storage slot the iterator is currently at.
func (it *diffStorageIterator) Hash() common.Hash {
	return it.curHash
}

// Slot returns the raw storage slot value the iterator is currently at.
func (it *diffStorageIterator) Slot() []byte {
	it.layer.lock.RLock()
	storage, ok := it.layer.storageData[it.account]
	if !ok {
		panic(fmt.Sprintf("iterator referenced non-existent account storage: %x", it.account))
	}
	blob, ok := storage[it.curHash]
	if !ok {
		panic(fmt.Sprintf("iterator referenced non-existent storage slot: %x", it.curHash))
	}
	it.layer.lock.RUnlock()
	if it.layer.Stale() {
		it.fail, it.keys = ErrSnapshotStale, nil
	}
	return blob
}

// Release is a noop for diff storage iterators.
func (it *diffStorageIterator) Release() {}

// diskStorageIterator is a storage iterator that steps over the live storage
// contained within a disk layer.
type diskStorageIterator struct {
	layer   *diskLayer
	account common.Hash
	it      ethdb.Iterator
}

// StorageIterator creates a storage iterator over a disk layer.
func (dl *diskLayer) StorageIterator(account common.Hash, seek common.Hash) StorageIterator {
	pos := common.TrimRightZeroes(seek[:])
	return &diskStorageIterator{
		layer:   dl,
		account: account,
		it:      dl.diskdb.NewIterator(append(rawdb.SnapshotStoragePrefix, account.Bytes()...), pos),
	}
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *diskStorageIterator) Next() bool {
	if it.it == nil {
		return false
	}
	for {
		if !it.it.Next() {
			it.it.Release()
			it.it = nil
			return false
		}
		if len(it.it.Key()) == len(rawdb.SnapshotStoragePrefix)+common.HashLength+common.HashLength {
			break
		}
	}
	return true
}

// Error returns any failure that occurred during iteration.
func (it *diskStorageIterator) Error() error {
	if it.it == nil {
		return nil
	}
	return it.it.Error()
}

// Hash returns the hash of the storage slot the iterator is currently at.
func (it *diskStorageIterator) Hash() common.Hash {
	return common.BytesToHash(it.it.Key())
}

// Slot returns the raw storage slot content the iterator is currently at.
func (it *diskStorageIterator) Slot() []byte {
	return it.it.Value()
}

// Release releases the database snapshot held during iteration.
func (it *diskStorageIterator) Release() {
	if it.it != nil {
		it.it.Release()
		it.it = nil
	}
}
