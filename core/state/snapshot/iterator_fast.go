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
"cmp"
"fmt"
"slices"
"sort"

"github.com/SILA/sila-chain/common"
)

// weightedIterator is an iterator with an assigned weight.
type weightedIterator struct {
it       Iterator
priority int
}

func (it *weightedIterator) Cmp(other *weightedIterator) int {
hashI := it.it.Hash()
hashJ := other.it.Hash()

switch bytes.Compare(hashI[:], hashJ[:]) {
case -1:
return -1
case 1:
return 1
}
return cmp.Compare(it.priority, other.priority)
}

// fastIterator is a more optimized multi-layer iterator.
type fastIterator struct {
tree *Tree
root common.Hash

curAccount []byte
curSlot    []byte

iterators []*weightedIterator
initiated bool
account   bool
fail      error
}

// newFastIterator creates a new hierarchical account or storage iterator.
func newFastIterator(tree *Tree, root common.Hash, account common.Hash, seek common.Hash, accountIterator bool) (*fastIterator, error) {
snap := tree.Snapshot(root)
if snap == nil {
return nil, fmt.Errorf("unknown snapshot: %x", root)
}
fi := &fastIterator{
tree:    tree,
root:    root,
account: accountIterator,
}
current := snap.(snapshot)
for depth := 0; current != nil; depth++ {
if accountIterator {
fi.iterators = append(fi.iterators, &weightedIterator{
it:       current.AccountIterator(seek),
priority: depth,
})
} else {
fi.iterators = append(fi.iterators, &weightedIterator{
it:       current.StorageIterator(account, seek),
priority: depth,
})
}
current = current.Parent()
}
fi.init()
return fi, nil
}

// init walks over all the iterators and resolves any clashes between them.
func (fi *fastIterator) init() {
var positioned = make(map[common.Hash]int)

for i := 0; i < len(fi.iterators); i++ {
it := fi.iterators[i]
for {
if !it.it.Next() {
it.it.Release()
last := len(fi.iterators) - 1

fi.iterators[i] = fi.iterators[last]
fi.iterators[last] = nil
fi.iterators = fi.iterators[:last]

i--
break
}
hash := it.it.Hash()
if other, exist := positioned[hash]; !exist {
positioned[hash] = i
break
} else {
if fi.iterators[other].priority < it.priority {
continue
} else {
it = fi.iterators[other]
fi.iterators[other], fi.iterators[i] = fi.iterators[i], fi.iterators[other]
continue
}
}
}
}
slices.SortFunc(fi.iterators, func(a, b *weightedIterator) int { return a.Cmp(b) })
fi.initiated = false
}

// Next steps the iterator forward one element, returning false if exhausted.
func (fi *fastIterator) Next() bool {
if len(fi.iterators) == 0 {
return false
}
if !fi.initiated {
fi.initiated = true
if fi.account {
fi.curAccount = fi.iterators[0].it.(AccountIterator).Account()
} else {
fi.curSlot = fi.iterators[0].it.(StorageIterator).Slot()
}
if innerErr := fi.iterators[0].it.Error(); innerErr != nil {
fi.fail = innerErr
return false
}
if fi.curAccount != nil || fi.curSlot != nil {
return true
}
}
for {
if !fi.next(0) {
return false
}
if fi.account {
fi.curAccount = fi.iterators[0].it.(AccountIterator).Account()
} else {
fi.curSlot = fi.iterators[0].it.(StorageIterator).Slot()
}
if innerErr := fi.iterators[0].it.Error(); innerErr != nil {
fi.fail = innerErr
return false
}
if fi.curAccount != nil || fi.curSlot != nil {
break
}
}
return true
}

// next handles the next operation internally.
func (fi *fastIterator) next(idx int) bool {
if it := fi.iterators[idx].it; !it.Next() {
it.Release()

fi.iterators = append(fi.iterators[:idx], fi.iterators[idx+1:]...)
return len(fi.iterators) > 0
}
if idx == len(fi.iterators)-1 {
return true
}
var (
cur, next         = fi.iterators[idx], fi.iterators[idx+1]
curHash, nextHash = cur.it.Hash(), next.it.Hash()
)
if diff := bytes.Compare(curHash[:], nextHash[:]); diff < 0 {
return true
} else if diff == 0 && cur.priority < next.priority {
fi.next(idx + 1)
return true
}
clash := -1
index := sort.Search(len(fi.iterators), func(n int) bool {
if n < idx {
return false
}
if n == len(fi.iterators)-1 {
return true
}
nextHash := fi.iterators[n+1].it.Hash()
if diff := bytes.Compare(curHash[:], nextHash[:]); diff < 0 {
return true
} else if diff > 0 {
return false
}
clash = n + 1

return cur.priority < fi.iterators[n+1].priority
})
fi.move(idx, index)
if clash != -1 {
fi.next(clash)
}
return true
}

// move advances an iterator to another position in the list.
func (fi *fastIterator) move(index, newpos int) {
elem := fi.iterators[index]
copy(fi.iterators[index:], fi.iterators[index+1:newpos+1])
fi.iterators[newpos] = elem
}

// Error returns any failure that occurred during iteration.
func (fi *fastIterator) Error() error {
return fi.fail
}

// Hash returns the current key.
func (fi *fastIterator) Hash() common.Hash {
return fi.iterators[0].it.Hash()
}

// Account returns the current account blob.
func (fi *fastIterator) Account() []byte {
return fi.curAccount
}

// Slot returns the current storage slot.
func (fi *fastIterator) Slot() []byte {
return fi.curSlot
}

// Release iterates over all the remaining live layer iterators and releases each.
func (fi *fastIterator) Release() {
for _, it := range fi.iterators {
it.it.Release()
}
fi.iterators = nil
}

// Debug is a convenience helper during testing.
func (fi *fastIterator) Debug() {
for _, it := range fi.iterators {
fmt.Printf("[p=%v v=%v] ", it.priority, it.it.Hash()[0])
}
fmt.Println()
}

// newFastAccountIterator creates a new hierarchical account iterator.
func newFastAccountIterator(tree *Tree, root common.Hash, seek common.Hash) (AccountIterator, error) {
return newFastIterator(tree, root, common.Hash{}, seek, true)
}

// newFastStorageIterator creates a new hierarchical storage iterator.
func newFastStorageIterator(tree *Tree, root common.Hash, account common.Hash, seek common.Hash) (StorageIterator, error) {
return newFastIterator(tree, root, account, seek, false)
}
