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

	"silachain/common"
)

// binaryIterator is a simplistic iterator to step over the accounts or storage
// in a snapshot, which may or may not be composed of multiple layers.
type binaryIterator struct {
	a               Iterator
	b               Iterator
	aDone           bool
	bDone           bool
	accountIterator bool
	k               common.Hash
	account         common.Hash
	fail            error
}

// initBinaryAccountIterator creates a simplistic iterator to step over all the
// accounts in a slow, but easily verifiable way.
func (dl *diffLayer) initBinaryAccountIterator(seek common.Hash) Iterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		l := &binaryIterator{
			a:               dl.AccountIterator(seek),
			b:               dl.Parent().AccountIterator(seek),
			accountIterator: true,
		}
		l.aDone = !l.a.Next()
		l.bDone = !l.b.Next()
		return l
	}
	l := &binaryIterator{
		a:               dl.AccountIterator(seek),
		b:               parent.initBinaryAccountIterator(seek),
		accountIterator: true,
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// initBinaryStorageIterator creates a simplistic iterator to step over all the
// storage slots in a slow, but easily verifiable way.
func (dl *diffLayer) initBinaryStorageIterator(account, seek common.Hash) Iterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		l := &binaryIterator{
			a:       dl.StorageIterator(account, seek),
			b:       dl.Parent().StorageIterator(account, seek),
			account: account,
		}
		l.aDone = !l.a.Next()
		l.bDone = !l.b.Next()
		return l
	}
	l := &binaryIterator{
		a:       dl.StorageIterator(account, seek),
		b:       parent.initBinaryStorageIterator(account, seek),
		account: account,
	}
	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

// Next steps the iterator forward one element, returning false if exhausted.
func (it *binaryIterator) Next() bool {
	for {
		if !it.next() {
			return false
		}
		if len(it.Account()) != 0 || len(it.Slot()) != 0 {
			return true
		}
		if it.fail != nil {
			return false
		}
	}
}

func (it *binaryIterator) next() bool {
	if it.aDone && it.bDone {
		return false
	}
	for {
		if it.aDone {
			it.k = it.b.Hash()
			it.bDone = !it.b.Next()
			return true
		}
		if it.bDone {
			it.k = it.a.Hash()
			it.aDone = !it.a.Next()
			return true
		}
		nextA, nextB := it.a.Hash(), it.b.Hash()
		if diff := bytes.Compare(nextA[:], nextB[:]); diff < 0 {
			it.aDone = !it.a.Next()
			it.k = nextA
			return true
		} else if diff == 0 {
			it.aDone = !it.a.Next()
			continue
		}
		it.bDone = !it.b.Next()
		it.k = nextB
		return true
	}
}

// Error returns any failure that occurred during iteration.
func (it *binaryIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the account the iterator is currently at.
func (it *binaryIterator) Hash() common.Hash {
	return it.k
}

// Account returns the RLP encoded slim account the iterator is currently at.
func (it *binaryIterator) Account() []byte {
	if !it.accountIterator {
		return nil
	}
	blob, err := it.a.(*diffAccountIterator).layer.AccountRLP(it.k)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}

// Slot returns the raw storage slot data the iterator is currently at.
func (it *binaryIterator) Slot() []byte {
	if it.accountIterator {
		return nil
	}
	blob, err := it.a.(*diffStorageIterator).layer.Storage(it.account, it.k)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}

// Release recursively releases all the iterators in the stack.
func (it *binaryIterator) Release() {
	it.a.Release()
	if it.b != nil {
		it.b.Release()
	}
}

// newBinaryAccountIterator creates a simplistic account iterator.
func (dl *diffLayer) newBinaryAccountIterator(seek common.Hash) AccountIterator {
	iter := dl.initBinaryAccountIterator(seek)
	return iter.(AccountIterator)
}

// newBinaryStorageIterator creates a simplistic storage iterator.
func (dl *diffLayer) newBinaryStorageIterator(account, seek common.Hash) StorageIterator {
	iter := dl.initBinaryStorageIterator(account, seek)
	return iter.(StorageIterator)
}
