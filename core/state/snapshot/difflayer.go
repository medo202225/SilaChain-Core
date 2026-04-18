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
	"encoding/binary"
	"fmt"
	"maps"
	"math"
	"math/rand"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	bloomfilter "github.com/holiman/bloomfilter/v2"
	"silachain/common"
	"silachain/core/types"
	"silachain/rlp"
)

var (
	aggregatorMemoryLimit = uint64(4 * 1024 * 1024)
	aggregatorItemLimit   = aggregatorMemoryLimit / 42
	bloomTargetError      = 0.02
	bloomSize             = math.Ceil(float64(aggregatorItemLimit) * math.Log(bloomTargetError) / math.Log(1/math.Pow(2, math.Log(2))))
	bloomFuncs            = math.Round((bloomSize / float64(aggregatorItemLimit)) * math.Log(2))

	bloomAccountHasherOffset = 0
	bloomStorageHasherOffset = 0
)

func init() {
	bloomAccountHasherOffset = rand.Intn(25)
	bloomStorageHasherOffset = rand.Intn(25)
}

// diffLayer represents a collection of modifications made to a state snapshot.
type diffLayer struct {
	origin *diskLayer
	parent snapshot
	memory uint64

	root  common.Hash
	stale atomic.Bool

	accountData map[common.Hash][]byte
	storageData map[common.Hash]map[common.Hash][]byte
	accountList []common.Hash
	storageList map[common.Hash][]common.Hash

	diffed *bloomfilter.Filter

	lock sync.RWMutex
}

// accountBloomHash is used to convert an account hash into a 64 bit mini hash.
func accountBloomHash(h common.Hash) uint64 {
	return binary.BigEndian.Uint64(h[bloomAccountHasherOffset : bloomAccountHasherOffset+8])
}

// storageBloomHash is used to convert an account hash and a storage hash into a 64 bit mini hash.
func storageBloomHash(h0, h1 common.Hash) uint64 {
	return binary.BigEndian.Uint64(h0[bloomStorageHasherOffset:bloomStorageHasherOffset+8]) ^
		binary.BigEndian.Uint64(h1[bloomStorageHasherOffset:bloomStorageHasherOffset+8])
}

// newDiffLayer creates a new diff on top of an existing snapshot.
func newDiffLayer(parent snapshot, root common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer {
	dl := &diffLayer{
		parent:      parent,
		root:        root,
		accountData: accounts,
		storageData: storage,
		storageList: make(map[common.Hash][]common.Hash),
	}
	switch parent := parent.(type) {
	case *diskLayer:
		dl.rebloom(parent)
	case *diffLayer:
		dl.rebloom(parent.origin)
	default:
		panic("unknown parent type")
	}
	for _, blob := range accounts {
		dl.memory += uint64(common.HashLength + len(blob))
		snapshotDirtyAccountWriteMeter.Mark(int64(len(blob)))
	}
	for accountHash, slots := range storage {
		if slots == nil {
			panic(fmt.Sprintf("storage %#x nil", accountHash))
		}
		for _, data := range slots {
			dl.memory += uint64(common.HashLength + len(data))
			snapshotDirtyStorageWriteMeter.Mark(int64(len(data)))
		}
	}
	return dl
}

// rebloom discards the layer's current bloom and rebuilds it from scratch.
func (dl *diffLayer) rebloom(origin *diskLayer) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	defer func(start time.Time) {
		snapshotBloomIndexTimer.Update(time.Since(start))
	}(time.Now())

	dl.origin = origin

	if parent, ok := dl.parent.(*diffLayer); ok {
		parent.lock.RLock()
		dl.diffed, _ = parent.diffed.Copy()
		parent.lock.RUnlock()
	} else {
		dl.diffed, _ = bloomfilter.New(uint64(bloomSize), uint64(bloomFuncs))
	}
	for hash := range dl.accountData {
		dl.diffed.AddHash(accountBloomHash(hash))
	}
	for accountHash, slots := range dl.storageData {
		for storageHash := range slots {
			dl.diffed.AddHash(storageBloomHash(accountHash, storageHash))
		}
	}
	k := float64(dl.diffed.K())
	n := float64(dl.diffed.N())
	m := float64(dl.diffed.M())
	snapshotBloomErrorGauge.Update(math.Pow(1.0-math.Exp((-k)*(n+0.5)/(m-1)), k))
}

// Root returns the root hash for which this snapshot was made.
func (dl *diffLayer) Root() common.Hash {
	return dl.root
}

// Parent returns the subsequent layer of a diff layer.
func (dl *diffLayer) Parent() snapshot {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.parent
}

// Stale return whether this layer has become stale.
func (dl *diffLayer) Stale() bool {
	return dl.stale.Load()
}

// Account directly retrieves the account associated with a particular hash.
func (dl *diffLayer) Account(hash common.Hash) (*types.SlimAccount, error) {
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
func (dl *diffLayer) AccountRLP(hash common.Hash) ([]byte, error) {
	dl.lock.RLock()
	if dl.Stale() {
		dl.lock.RUnlock()
		return nil, ErrSnapshotStale
	}
	var origin *diskLayer
	hit := dl.diffed.ContainsHash(accountBloomHash(hash))
	if !hit {
		origin = dl.origin
	}
	dl.lock.RUnlock()

	if origin != nil {
		snapshotBloomAccountMissMeter.Mark(1)
		return origin.AccountRLP(hash)
	}
	return dl.accountRLP(hash, 0)
}

// accountRLP is an internal version of AccountRLP that skips the bloom filter checks.
func (dl *diffLayer) accountRLP(hash common.Hash, depth int) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	if data, ok := dl.accountData[hash]; ok {
		snapshotDirtyAccountHitMeter.Mark(1)
		snapshotDirtyAccountHitDepthHist.Update(int64(depth))
		if n := len(data); n > 0 {
			snapshotDirtyAccountReadMeter.Mark(int64(n))
		} else {
			snapshotDirtyAccountInexMeter.Mark(1)
		}
		snapshotBloomAccountTrueHitMeter.Mark(1)
		return data, nil
	}
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.accountRLP(hash, depth+1)
	}
	snapshotBloomAccountFalseHitMeter.Mark(1)
	return dl.parent.AccountRLP(hash)
}

// Storage directly retrieves the storage data associated with a particular hash.
func (dl *diffLayer) Storage(accountHash, storageHash common.Hash) ([]byte, error) {
	dl.lock.RLock()
	if dl.Stale() {
		dl.lock.RUnlock()
		return nil, ErrSnapshotStale
	}
	var origin *diskLayer
	hit := dl.diffed.ContainsHash(storageBloomHash(accountHash, storageHash))
	if !hit {
		origin = dl.origin
	}
	dl.lock.RUnlock()

	if origin != nil {
		snapshotBloomStorageMissMeter.Mark(1)
		return origin.Storage(accountHash, storageHash)
	}
	return dl.storage(accountHash, storageHash, 0)
}

// storage is an internal version of Storage that skips the bloom filter checks.
func (dl *diffLayer) storage(accountHash, storageHash common.Hash, depth int) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.Stale() {
		return nil, ErrSnapshotStale
	}
	if storage, ok := dl.storageData[accountHash]; ok {
		if data, ok := storage[storageHash]; ok {
			snapshotDirtyStorageHitMeter.Mark(1)
			snapshotDirtyStorageHitDepthHist.Update(int64(depth))
			if n := len(data); n > 0 {
				snapshotDirtyStorageReadMeter.Mark(int64(n))
			} else {
				snapshotDirtyStorageInexMeter.Mark(1)
			}
			snapshotBloomStorageTrueHitMeter.Mark(1)
			return data, nil
		}
	}
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.storage(accountHash, storageHash, depth+1)
	}
	snapshotBloomStorageFalseHitMeter.Mark(1)
	return dl.parent.Storage(accountHash, storageHash)
}

// Update creates a new layer on top of the existing snapshot diff tree.
func (dl *diffLayer) Update(blockRoot common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer {
	return newDiffLayer(dl, blockRoot, accounts, storage)
}

// flatten pushes all data from this point downwards.
func (dl *diffLayer) flatten() snapshot {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		return dl
	}
	parent = parent.flatten().(*diffLayer)

	parent.lock.Lock()
	defer parent.lock.Unlock()

	if parent.stale.Swap(true) {
		panic("parent diff layer is stale")
	}
	for hash, data := range dl.accountData {
		parent.accountData[hash] = data
	}
	for accountHash, storage := range dl.storageData {
		if _, ok := parent.storageData[accountHash]; !ok {
			parent.storageData[accountHash] = storage
			continue
		}
		maps.Copy(parent.storageData[accountHash], storage)
	}
	return &diffLayer{
		parent:      parent.parent,
		origin:      parent.origin,
		root:        dl.root,
		accountData: parent.accountData,
		storageData: parent.storageData,
		storageList: make(map[common.Hash][]common.Hash),
		diffed:      dl.diffed,
		memory:      parent.memory + dl.memory,
	}
}

// AccountList returns a sorted list of all accounts in this diffLayer.
func (dl *diffLayer) AccountList() []common.Hash {
	dl.lock.RLock()
	list := dl.accountList
	dl.lock.RUnlock()

	if list != nil {
		return list
	}
	dl.lock.Lock()
	defer dl.lock.Unlock()

	dl.accountList = slices.SortedFunc(maps.Keys(dl.accountData), common.Hash.Cmp)
	dl.memory += uint64(len(dl.accountList) * common.HashLength)
	return dl.accountList
}

// StorageList returns a sorted list of all storage slot hashes in this diffLayer.
func (dl *diffLayer) StorageList(accountHash common.Hash) []common.Hash {
	dl.lock.RLock()
	if _, ok := dl.storageData[accountHash]; !ok {
		dl.lock.RUnlock()
		return nil
	}
	if list, exist := dl.storageList[accountHash]; exist {
		dl.lock.RUnlock()
		return list
	}
	dl.lock.RUnlock()

	dl.lock.Lock()
	defer dl.lock.Unlock()

	storageList := slices.SortedFunc(maps.Keys(dl.storageData[accountHash]), common.Hash.Cmp)
	dl.storageList[accountHash] = storageList
	dl.memory += uint64(len(storageList)*common.HashLength + common.HashLength)
	return storageList
}
