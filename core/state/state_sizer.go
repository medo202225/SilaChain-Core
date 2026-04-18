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

package state

import (
	"container/heap"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"silachain/common"
	"silachain/core/rawdb"
	"silachain/crypto"
	"silachain/ethdb"
	"silachain/log"
	"silachain/metrics"
	"silachain/triedb"
)

const (
	statEvictThreshold = 128
)

var (
	accountKeySize            = int64(len(rawdb.SnapshotAccountPrefix) + common.HashLength)
	storageKeySize            = int64(len(rawdb.SnapshotStoragePrefix) + common.HashLength*2)
	accountTrienodePrefixSize = int64(len(rawdb.TrieNodeAccountPrefix))
	storageTrienodePrefixSize = int64(len(rawdb.TrieNodeStoragePrefix) + common.HashLength)
	codeKeySize               = int64(len(rawdb.CodePrefix) + common.HashLength)
)

var (
	stateSizeChainHeightGauge           = metrics.NewRegisteredGauge("sila/state/height", nil)
	stateSizeAccountsCountGauge         = metrics.NewRegisteredGauge("sila/state/accounts/count", nil)
	stateSizeAccountsBytesGauge         = metrics.NewRegisteredGauge("sila/state/accounts/bytes", nil)
	stateSizeStoragesCountGauge         = metrics.NewRegisteredGauge("sila/state/storages/count", nil)
	stateSizeStoragesBytesGauge         = metrics.NewRegisteredGauge("sila/state/storages/bytes", nil)
	stateSizeAccountTrieNodesCountGauge = metrics.NewRegisteredGauge("sila/state/trienodes/account/count", nil)
	stateSizeAccountTrieNodesBytesGauge = metrics.NewRegisteredGauge("sila/state/trienodes/account/bytes", nil)
	stateSizeStorageTrieNodesCountGauge = metrics.NewRegisteredGauge("sila/state/trienodes/storage/count", nil)
	stateSizeStorageTrieNodesBytesGauge = metrics.NewRegisteredGauge("sila/state/trienodes/storage/bytes", nil)
	stateSizeContractsCountGauge        = metrics.NewRegisteredGauge("sila/state/contracts/count", nil)
	stateSizeContractsBytesGauge        = metrics.NewRegisteredGauge("sila/state/contracts/bytes", nil)
)

type SizeStats struct {
	StateRoot   common.Hash
	BlockNumber uint64

	Accounts             int64
	AccountBytes         int64
	Storages             int64
	StorageBytes         int64
	AccountTrienodes     int64
	AccountTrienodeBytes int64
	StorageTrienodes     int64
	StorageTrienodeBytes int64
	ContractCodes        int64
	ContractCodeBytes    int64
}

func (s SizeStats) String() string {
	return fmt.Sprintf("SILA Accounts: %d(%s), Storages: %d(%s), AccountTrienodes: %d(%s), StorageTrienodes: %d(%s), Codes: %d(%s)",
		s.Accounts, common.StorageSize(s.AccountBytes),
		s.Storages, common.StorageSize(s.StorageBytes),
		s.AccountTrienodes, common.StorageSize(s.AccountTrienodeBytes),
		s.StorageTrienodes, common.StorageSize(s.StorageTrienodeBytes),
		s.ContractCodes, common.StorageSize(s.ContractCodeBytes),
	)
}

func (s SizeStats) publish() {
	stateSizeChainHeightGauge.Update(int64(s.BlockNumber))
	stateSizeAccountsCountGauge.Update(s.Accounts)
	stateSizeAccountsBytesGauge.Update(s.AccountBytes)
	stateSizeStoragesCountGauge.Update(s.Storages)
	stateSizeStoragesBytesGauge.Update(s.StorageBytes)
	stateSizeAccountTrieNodesCountGauge.Update(s.AccountTrienodes)
	stateSizeAccountTrieNodesBytesGauge.Update(s.AccountTrienodeBytes)
	stateSizeStorageTrieNodesCountGauge.Update(s.StorageTrienodes)
	stateSizeStorageTrieNodesBytesGauge.Update(s.StorageTrienodeBytes)
	stateSizeContractsCountGauge.Update(s.ContractCodes)
	stateSizeContractsBytesGauge.Update(s.ContractCodeBytes)
}

func (s SizeStats) add(diff SizeStats) SizeStats {
	s.StateRoot = diff.StateRoot
	s.BlockNumber = diff.BlockNumber

	s.Accounts += diff.Accounts
	s.AccountBytes += diff.AccountBytes
	s.Storages += diff.Storages
	s.StorageBytes += diff.StorageBytes
	s.AccountTrienodes += diff.AccountTrienodes
	s.AccountTrienodeBytes += diff.AccountTrienodeBytes
	s.StorageTrienodes += diff.StorageTrienodes
	s.StorageTrienodeBytes += diff.StorageTrienodeBytes
	s.ContractCodes += diff.ContractCodes
	s.ContractCodeBytes += diff.ContractCodeBytes
	return s
}

func calSizeStats(update *stateUpdate) (SizeStats, error) {
	stats := SizeStats{
		BlockNumber: update.blockNumber,
		StateRoot:   update.root,
	}

	for addr, oldValue := range update.accountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		newValue, exists := update.accounts[addrHash]
		if !exists {
			return SizeStats{}, fmt.Errorf("SILA account %x not found", addr)
		}
		oldLen, newLen := len(oldValue), len(newValue)

		switch {
		case oldLen > 0 && newLen == 0:
			stats.Accounts -= 1
			stats.AccountBytes -= accountKeySize + int64(oldLen)
		case oldLen == 0 && newLen > 0:
			stats.Accounts += 1
			stats.AccountBytes += accountKeySize + int64(newLen)
		default:
			stats.AccountBytes += int64(newLen - oldLen)
		}
	}

	for addr, slots := range update.storagesOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		subset, exists := update.storages[addrHash]
		if !exists {
			return SizeStats{}, fmt.Errorf("SILA storage %x not found", addr)
		}
		for key, oldValue := range slots {
			var (
				exists   bool
				newValue []byte
			)
			if update.rawStorageKey {
				newValue, exists = subset[crypto.Keccak256Hash(key.Bytes())]
			} else {
				newValue, exists = subset[key]
			}
			if !exists {
				return SizeStats{}, fmt.Errorf("SILA storage slot %x-%x not found", addr, key)
			}
			oldLen, newLen := len(oldValue), len(newValue)

			switch {
			case oldLen > 0 && newLen == 0:
				stats.Storages -= 1
				stats.StorageBytes -= storageKeySize + int64(oldLen)
			case oldLen == 0 && newLen > 0:
				stats.Storages += 1
				stats.StorageBytes += storageKeySize + int64(newLen)
			default:
				stats.StorageBytes += int64(newLen - oldLen)
			}
		}
	}

	for owner, subset := range update.nodes.Sets {
		var (
			keyPrefix int64
			isAccount = owner == (common.Hash{})
		)
		if isAccount {
			keyPrefix = accountTrienodePrefixSize
		} else {
			keyPrefix = storageTrienodePrefixSize
		}

		for path, oldNode := range subset.Origins {
			newNode, exists := subset.Nodes[path]
			if !exists {
				return SizeStats{}, fmt.Errorf("SILA node %x-%v not found", owner, path)
			}
			keySize := keyPrefix + int64(len(path))

			switch {
			case len(oldNode) > 0 && len(newNode.Blob) == 0:
				if isAccount {
					stats.AccountTrienodes -= 1
					stats.AccountTrienodeBytes -= keySize + int64(len(oldNode))
				} else {
					stats.StorageTrienodes -= 1
					stats.StorageTrienodeBytes -= keySize + int64(len(oldNode))
				}
			case len(oldNode) == 0 && len(newNode.Blob) > 0:
				if isAccount {
					stats.AccountTrienodes += 1
					stats.AccountTrienodeBytes += keySize + int64(len(newNode.Blob))
				} else {
					stats.StorageTrienodes += 1
					stats.StorageTrienodeBytes += keySize + int64(len(newNode.Blob))
				}
			default:
				if isAccount {
					stats.AccountTrienodeBytes += int64(len(newNode.Blob) - len(oldNode))
				} else {
					stats.StorageTrienodeBytes += int64(len(newNode.Blob) - len(oldNode))
				}
			}
		}
	}

	codeExists := make(map[common.Hash]struct{})
	for _, code := range update.codes {
		if _, ok := codeExists[code.hash]; ok || code.duplicate {
			continue
		}
		stats.ContractCodes += 1
		stats.ContractCodeBytes += codeKeySize + int64(len(code.blob))
		codeExists[code.hash] = struct{}{}
	}
	return stats, nil
}

type stateSizeQuery struct {
	root   *common.Hash
	err    error
	result chan *SizeStats
}

type SizeTracker struct {
	db       ethdb.KeyValueStore
	triedb   *triedb.Database
	abort    chan struct{}
	aborted  chan struct{}
	updateCh chan *stateUpdate
	queryCh  chan *stateSizeQuery
}

func NewSizeTracker(db ethdb.KeyValueStore, triedb *triedb.Database) (*SizeTracker, error) {
	if triedb.Scheme() != rawdb.PathScheme {
		return nil, errors.New("SILA state size tracker is not compatible with hash mode")
	}
	t := &SizeTracker{
		db:       db,
		triedb:   triedb,
		abort:    make(chan struct{}),
		aborted:  make(chan struct{}),
		updateCh: make(chan *stateUpdate),
		queryCh:  make(chan *stateSizeQuery),
	}
	go t.run()
	return t, nil
}

func (t *SizeTracker) Stop() {
	close(t.abort)
	<-t.aborted
}

type sizeStatsHeap []SizeStats

func (h sizeStatsHeap) Len() int           { return len(h) }
func (h sizeStatsHeap) Less(i, j int) bool { return h[i].BlockNumber < h[j].BlockNumber }
func (h sizeStatsHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *sizeStatsHeap) Push(x any) {
	*h = append(*h, x.(SizeStats))
}

func (h *sizeStatsHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (t *SizeTracker) run() {
	defer close(t.aborted)

	var last common.Hash
	stats, err := t.init()
	if err != nil {
		return
	}
	h := sizeStatsHeap(slices.Collect(maps.Values(stats)))
	heap.Init(&h)

	for {
		select {
		case u := <-t.updateCh:
			base, found := stats[u.originRoot]
			if !found {
				log.Debug("Ignored the SILA state size without parent", "parent", u.originRoot, "root", u.root, "number", u.blockNumber)
				continue
			}
			diff, err := calSizeStats(u)
			if err != nil {
				continue
			}
			stat := base.add(diff)
			stats[u.root] = stat
			last = u.root

			stat.publish()

			heap.Push(&h, stats[u.root])
			for len(h) > 0 && u.blockNumber-h[0].BlockNumber > statEvictThreshold {
				delete(stats, h[0].StateRoot)
				heap.Pop(&h)
			}
			log.Debug("Update SILA state size", "number", stat.BlockNumber, "root", stat.StateRoot, "stat", stat)

		case r := <-t.queryCh:
			var root common.Hash
			if r.root != nil {
				root = *r.root
			} else {
				root = last
			}
			if s, ok := stats[root]; ok {
				r.result <- &s
			} else {
				r.result <- nil
			}

		case <-t.abort:
			return
		}
	}
}

type buildResult struct {
	stat        SizeStats
	root        common.Hash
	blockNumber uint64
	elapsed     time.Duration
	err         error
}

func (t *SizeTracker) init() (map[common.Hash]SizeStats, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

wait:
	for {
		select {
		case <-ticker.C:
			if t.triedb.SnapshotCompleted() {
				break wait
			}
		case <-t.updateCh:
			continue
		case r := <-t.queryCh:
			r.err = errors.New("SILA state size is not initialized yet")
			r.result <- nil
		case <-t.abort:
			return nil, errors.New("SILA size tracker closed")
		}
	}

	var (
		updates  = make(map[common.Hash]*stateUpdate)
		children = make(map[common.Hash][]common.Hash)
		done     chan buildResult
	)

	for {
		select {
		case u := <-t.updateCh:
			updates[u.root] = u
			children[u.originRoot] = append(children[u.originRoot], u.root)
			log.Debug("Received SILA state update", "root", u.root, "blockNumber", u.blockNumber)

		case r := <-t.queryCh:
			r.err = errors.New("SILA state size is not initialized yet")
			r.result <- nil

		case <-ticker.C:
			if done != nil {
				continue
			}
			root := rawdb.ReadSnapshotRoot(t.db)
			if root == (common.Hash{}) {
				continue
			}
			entry, exists := updates[root]
			if !exists {
				continue
			}
			done = make(chan buildResult)
			go t.build(entry.root, entry.blockNumber, done)
			log.Info("Measuring persistent SILA state size", "root", root.Hex(), "number", entry.blockNumber)

		case result := <-done:
			if result.err != nil {
				return nil, result.err
			}
			var (
				stats = make(map[common.Hash]SizeStats)
				apply func(root common.Hash, stat SizeStats) error
			)
			apply = func(root common.Hash, base SizeStats) error {
				for _, child := range children[root] {
					entry, ok := updates[child]
					if !ok {
						return fmt.Errorf("the SILA state update is not found, %x", child)
					}
					diff, err := calSizeStats(entry)
					if err != nil {
						return err
					}
					stats[child] = base.add(diff)
					if err := apply(child, stats[child]); err != nil {
						return err
					}
				}
				return nil
			}
			if err := apply(result.root, result.stat); err != nil {
				return nil, err
			}

			stats[result.root] = result.stat
			log.Info("Measured persistent SILA state size", "root", result.root, "number", result.blockNumber, "stat", result.stat, "elapsed", common.PrettyDuration(result.elapsed))
			return stats, nil

		case <-t.abort:
			return nil, errors.New("SILA size tracker closed")
		}
	}
}

func (t *SizeTracker) build(root common.Hash, blockNumber uint64, done chan buildResult) {
	var (
		accounts, accountBytes int64
		storages, storageBytes int64
		codes, codeBytes       int64

		accountTrienodes, accountTrienodeBytes int64
		storageTrienodes, storageTrienodeBytes int64

		group errgroup.Group
		start = time.Now()
	)

	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.SnapshotAccountPrefix, "sila_account")
		if err != nil {
			return err
		}
		accounts, accountBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.SnapshotStoragePrefix, "sila_storage")
		if err != nil {
			return err
		}
		storages, storageBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.TrieNodeAccountPrefix, "sila_accountnode")
		if err != nil {
			return err
		}
		accountTrienodes, accountTrienodeBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTableParallel(t.abort, rawdb.TrieNodeStoragePrefix, "sila_storagenode")
		if err != nil {
			return err
		}
		storageTrienodes, storageTrienodeBytes = count, bytes
		return nil
	})

	group.Go(func() error {
		count, bytes, err := t.iterateTable(t.abort, rawdb.CodePrefix, "sila_contractcode")
		if err != nil {
			return err
		}
		codes, codeBytes = count, bytes
		return nil
	})

	if err := group.Wait(); err != nil {
		done <- buildResult{err: err}
	} else {
		stat := SizeStats{
			StateRoot:            root,
			BlockNumber:          blockNumber,
			Accounts:             accounts,
			AccountBytes:         accountBytes,
			Storages:             storages,
			StorageBytes:         storageBytes,
			AccountTrienodes:     accountTrienodes,
			AccountTrienodeBytes: accountTrienodeBytes,
			StorageTrienodes:     storageTrienodes,
			StorageTrienodeBytes: storageTrienodeBytes,
			ContractCodes:        codes,
			ContractCodeBytes:    codeBytes,
		}
		done <- buildResult{
			root:        root,
			blockNumber: blockNumber,
			stat:        stat,
			elapsed:     time.Since(start),
		}
	}
}

func (t *SizeTracker) iterateTable(closed chan struct{}, prefix []byte, name string) (int64, int64, error) {
	var (
		start        = time.Now()
		logged       = time.Now()
		count, bytes int64
	)

	iter := t.db.NewIterator(prefix, nil)
	defer iter.Release()

	log.Debug("Iterating SILA state", "category", name)
	for iter.Next() {
		count++
		bytes += int64(len(iter.Key()) + len(iter.Value()))

		if time.Since(logged) > time.Second*8 {
			logged = time.Now()

			select {
			case <-closed:
				log.Debug("SILA state iteration cancelled", "category", name)
				return 0, 0, errors.New("SILA size tracker closed")
			default:
				log.Debug("Iterating SILA state", "category", name, "count", count, "size", common.StorageSize(bytes))
			}
		}
	}
	if err := iter.Error(); err != nil {
		log.Error("SILA iterator error", "category", name, "err", err)
		return 0, 0, err
	}
	log.Debug("Finished SILA state iteration", "category", name, "count", count, "size", common.StorageSize(bytes), "elapsed", common.PrettyDuration(time.Since(start)))
	return count, bytes, nil
}

func (t *SizeTracker) iterateTableParallel(closed chan struct{}, prefix []byte, name string) (int64, int64, error) {
	var (
		totalCount int64
		totalBytes int64

		start   = time.Now()
		workers = runtime.NumCPU()
		group   errgroup.Group
		mu      sync.Mutex
	)
	group.SetLimit(workers)
	log.Debug("Starting parallel SILA state iteration", "category", name, "workers", workers)

	if len(prefix) > 0 {
		if blob, err := t.db.Get(prefix); err == nil && len(blob) > 0 {
			totalCount = 1
			totalBytes = int64(len(prefix) + len(blob))
		}
	}
	for i := 0; i < 256; i++ {
		h := byte(i)
		group.Go(func() error {
			count, bytes, err := t.iterateTable(closed, slices.Concat(prefix, []byte{h}), fmt.Sprintf("%s-%02x", name, h))
			if err != nil {
				return err
			}
			mu.Lock()
			totalCount += count
			totalBytes += bytes
			mu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return 0, 0, err
	}
	log.Debug("Finished parallel SILA state iteration", "category", name, "count", totalCount, "size", common.StorageSize(totalBytes), "elapsed", common.PrettyDuration(time.Since(start)))
	return totalCount, totalBytes, nil
}

func (t *SizeTracker) Notify(update *stateUpdate) {
	if update == nil || update.empty() {
		return
	}
	select {
	case t.updateCh <- update:
	case <-t.abort:
		return
	}
}

func (t *SizeTracker) Query(root *common.Hash) (*SizeStats, error) {
	r := &stateSizeQuery{
		root:   root,
		result: make(chan *SizeStats, 1),
	}
	select {
	case <-t.aborted:
		return nil, errors.New("SILA state sizer has been closed")
	case t.queryCh <- r:
		return <-r.result, r.err
	}
}
