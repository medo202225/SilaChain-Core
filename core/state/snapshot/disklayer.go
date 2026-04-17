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
"sync"

"github.com/VictoriaMetrics/fastcache"
"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/core/rawdb"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/ethdb"
"github.com/SILA/sila-chain/rlp"
"github.com/SILA/sila-chain/triedb"
)

// diskLayer is a low level persistent snapshot built on top of a key-value store.
type diskLayer struct {
diskdb ethdb.KeyValueStore
triedb *triedb.Database
cache  *fastcache.Cache

root  common.Hash
stale bool

genMarker  []byte
genPending chan struct{}
genAbort   chan chan *generatorStats

lock sync.RWMutex
}

// Release releases underlying resources.
func (dl *diskLayer) Release() error {
if dl.cache != nil {
dl.cache.Reset()
}
return nil
}

// Root returns root hash for which this snapshot was made.
func (dl *diskLayer) Root() common.Hash {
return dl.root
}

// Parent always returns nil as there's no layer below the disk.
func (dl *diskLayer) Parent() snapshot {
return nil
}

// Stale return whether this layer has become stale.
func (dl *diskLayer) Stale() bool {
dl.lock.RLock()
defer dl.lock.RUnlock()

return dl.stale
}

// markStale sets the stale flag as true.
func (dl *diskLayer) markStale() {
dl.lock.Lock()
defer dl.lock.Unlock()

dl.stale = true
}

// Account directly retrieves the account associated with a particular hash.
func (dl *diskLayer) Account(hash common.Hash) (*types.SlimAccount, error) {
data, err := dl.AccountRLP(hash)
if err != nil {
return nil, err
}
if len(data) == 0 {
return nil, nil
}
account := new(types.SlimAccount)
if err := rlp.DecodeBytes(data, account); err != nil {
panic(err)
}
return account, nil
}

// AccountRLP directly retrieves the account RLP associated with a particular hash.
func (dl *diskLayer) AccountRLP(hash common.Hash) ([]byte, error) {
dl.lock.RLock()
defer dl.lock.RUnlock()

if dl.stale {
return nil, ErrSnapshotStale
}
if dl.genMarker != nil && bytes.Compare(hash[:], dl.genMarker) > 0 {
return nil, ErrNotCoveredYet
}
snapshotDirtyAccountMissMeter.Mark(1)

if blob, found := dl.cache.HasGet(nil, hash[:]); found {
snapshotCleanAccountHitMeter.Mark(1)
snapshotCleanAccountReadMeter.Mark(int64(len(blob)))
return blob, nil
}
blob := rawdb.ReadAccountSnapshot(dl.diskdb, hash)
dl.cache.Set(hash[:], blob)

snapshotCleanAccountMissMeter.Mark(1)
if n := len(blob); n > 0 {
snapshotCleanAccountWriteMeter.Mark(int64(n))
} else {
snapshotCleanAccountInexMeter.Mark(1)
}
return blob, nil
}

// Storage directly retrieves the storage data associated with a particular hash.
func (dl *diskLayer) Storage(accountHash, storageHash common.Hash) ([]byte, error) {
dl.lock.RLock()
defer dl.lock.RUnlock()

if dl.stale {
return nil, ErrSnapshotStale
}
key := append(accountHash[:], storageHash[:]...)

if dl.genMarker != nil && bytes.Compare(key, dl.genMarker) > 0 {
return nil, ErrNotCoveredYet
}
snapshotDirtyStorageMissMeter.Mark(1)

if blob, found := dl.cache.HasGet(nil, key); found {
snapshotCleanStorageHitMeter.Mark(1)
snapshotCleanStorageReadMeter.Mark(int64(len(blob)))
return blob, nil
}
blob := rawdb.ReadStorageSnapshot(dl.diskdb, accountHash, storageHash)
dl.cache.Set(key, blob)

snapshotCleanStorageMissMeter.Mark(1)
if n := len(blob); n > 0 {
snapshotCleanStorageWriteMeter.Mark(int64(n))
} else {
snapshotCleanStorageInexMeter.Mark(1)
}
return blob, nil
}

// Update creates a new layer on top of the existing snapshot diff tree.
func (dl *diskLayer) Update(blockHash common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer {
return newDiffLayer(dl, blockHash, accounts, storage)
}

// stopGeneration aborts the state snapshot generation if it is currently running.
func (dl *diskLayer) stopGeneration() {
dl.lock.RLock()
generating := dl.genMarker != nil
dl.lock.RUnlock()
if !generating {
return
}
if dl.genAbort != nil {
abort := make(chan *generatorStats)
dl.genAbort <- abort
<-abort
}
}
