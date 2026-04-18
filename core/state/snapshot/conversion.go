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
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"silachain/common"
	"silachain/core/rawdb"
	"silachain/core/types"
	"silachain/ethdb"
	"silachain/log"
	"silachain/rlp"
	"silachain/trie"
)

// trieKV represents a trie key-value pair.
type trieKV struct {
	key   common.Hash
	value []byte
}

type (
	// trieGeneratorFn is the interface of trie generation.
	trieGeneratorFn func(db ethdb.KeyValueWriter, scheme string, owner common.Hash, in chan (trieKV), out chan (common.Hash))

	// leafCallbackFn is the callback invoked at the leaves of the trie.
	leafCallbackFn func(db ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *generateStats) (common.Hash, error)
)

// GenerateTrie takes the whole snapshot tree as the input and regenerates the whole state.
func GenerateTrie(snaptree *Tree, root common.Hash, src ethdb.Database, dst ethdb.KeyValueWriter) error {
	acctIt, err := snaptree.AccountIterator(root, common.Hash{})
	if err != nil {
		return err
	}
	defer acctIt.Release()

	scheme := snaptree.triedb.Scheme()
	got, err := generateTrieRoot(dst, scheme, acctIt, common.Hash{}, stackTrieGenerate, func(dst ethdb.KeyValueWriter, accountHash, codeHash common.Hash, stat *generateStats) (common.Hash, error) {
		if codeHash != types.EmptyCodeHash {
			code := rawdb.ReadCode(src, codeHash)
			if len(code) == 0 {
				return common.Hash{}, errors.New("failed to read contract code")
			}
			rawdb.WriteCode(dst, codeHash, code)
		}
		storageIt, err := snaptree.StorageIterator(root, accountHash, common.Hash{})
		if err != nil {
			return common.Hash{}, err
		}
		defer storageIt.Release()

		hash, err := generateTrieRoot(dst, scheme, storageIt, accountHash, stackTrieGenerate, nil, stat, false)
		if err != nil {
			return common.Hash{}, err
		}
		return hash, nil
	}, newGenerateStats(), true)

	if err != nil {
		return err
	}
	if got != root {
		return fmt.Errorf("state root hash mismatch: got %x, want %x", got, root)
	}
	return nil
}

// generateStats is a collection of statistics gathered by the trie generator.
type generateStats struct {
	head  common.Hash
	start time.Time

	accounts uint64
	slots    uint64

	slotsStart map[common.Hash]time.Time
	slotsHead  map[common.Hash]common.Hash

	lock sync.RWMutex
}

// newGenerateStats creates a new generator stats.
func newGenerateStats() *generateStats {
	return &generateStats{
		slotsStart: make(map[common.Hash]time.Time),
		slotsHead:  make(map[common.Hash]common.Hash),
		start:      time.Now(),
	}
}

// progressAccounts updates the generator stats for the account range.
func (stat *generateStats) progressAccounts(account common.Hash, done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.accounts += done
	stat.head = account
}

// finishAccounts updates the generator stats for the finished account range.
func (stat *generateStats) finishAccounts(done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.accounts += done
}

// progressContract updates the generator stats for a specific in-progress contract.
func (stat *generateStats) progressContract(account common.Hash, slot common.Hash, done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.slots += done
	stat.slotsHead[account] = slot
	if _, ok := stat.slotsStart[account]; !ok {
		stat.slotsStart[account] = time.Now()
	}
}

// finishContract updates the generator stats for a specific just-finished contract.
func (stat *generateStats) finishContract(account common.Hash, done uint64) {
	stat.lock.Lock()
	defer stat.lock.Unlock()

	stat.slots += done
	delete(stat.slotsHead, account)
	delete(stat.slotsStart, account)
}

// report prints the cumulative progress statistic smartly.
func (stat *generateStats) report() {
	stat.lock.RLock()
	defer stat.lock.RUnlock()

	ctx := []interface{}{
		"accounts", stat.accounts,
		"slots", stat.slots,
		"elapsed", common.PrettyDuration(time.Since(stat.start)),
	}
	if stat.accounts > 0 {
		if done := binary.BigEndian.Uint64(stat.head[:8]) / stat.accounts; done > 0 {
			var (
				left = (math.MaxUint64 - binary.BigEndian.Uint64(stat.head[:8])) / stat.accounts
				eta  = common.CalculateETA(done, left, time.Since(stat.start))
			)
			for acc, head := range stat.slotsHead {
				start := stat.slotsStart[acc]
				if done := binary.BigEndian.Uint64(head[:8]); done > 0 {
					left := math.MaxUint64 - binary.BigEndian.Uint64(head[:8])
					if slotETA := common.CalculateETA(done, left, time.Since(start)); eta < slotETA {
						eta = slotETA
					}
				}
			}
			ctx = append(ctx, []interface{}{
				"eta", common.PrettyDuration(eta),
			}...)
		}
	}
	log.Info("Iterating state snapshot", ctx...)
}

// reportDone prints the last log when the whole generation is finished.
func (stat *generateStats) reportDone() {
	stat.lock.RLock()
	defer stat.lock.RUnlock()

	var ctx []interface{}
	ctx = append(ctx, []interface{}{"accounts", stat.accounts}...)
	if stat.slots != 0 {
		ctx = append(ctx, []interface{}{"slots", stat.slots}...)
	}
	ctx = append(ctx, []interface{}{"elapsed", common.PrettyDuration(time.Since(stat.start))}...)
	log.Info("Iterated snapshot", ctx...)
}

// runReport periodically prints the progress information.
func runReport(stats *generateStats, stop chan bool) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			stats.report()
			timer.Reset(time.Second * 8)
		case success := <-stop:
			if success {
				stats.reportDone()
			}
			return
		}
	}
}

// generateTrieRoot generates the trie hash based on the snapshot iterator.
func generateTrieRoot(db ethdb.KeyValueWriter, scheme string, it Iterator, account common.Hash, generatorFn trieGeneratorFn, leafCallback leafCallbackFn, stats *generateStats, report bool) (common.Hash, error) {
	var (
		in      = make(chan trieKV)
		out     = make(chan common.Hash, 1)
		stoplog = make(chan bool, 1)
		wg      sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		generatorFn(db, scheme, account, in, out)
	}()
	if report && stats != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runReport(stats, stoplog)
		}()
	}
	threads := runtime.NumCPU()
	results := make(chan error, threads)
	for i := 0; i < threads; i++ {
		results <- nil
	}
	stop := func(fail error) (common.Hash, error) {
		close(in)
		result := <-out
		for i := 0; i < threads; i++ {
			if err := <-results; err != nil && fail == nil {
				fail = err
			}
		}
		stoplog <- fail == nil

		wg.Wait()
		return result, fail
	}
	var (
		logged    = time.Now()
		processed = uint64(0)
		leaf      trieKV
	)
	for it.Next() {
		if account == (common.Hash{}) {
			var (
				err      error
				fullData []byte
			)
			if leafCallback == nil {
				fullData, err = types.FullAccountRLP(it.(AccountIterator).Account())
				if err != nil {
					return stop(err)
				}
			} else {
				if err := <-results; err != nil {
					results <- nil
					return stop(err)
				}
				account, err := types.FullAccount(it.(AccountIterator).Account())
				if err != nil {
					return stop(err)
				}
				go func(hash common.Hash) {
					subroot, err := leafCallback(db, hash, common.BytesToHash(account.CodeHash), stats)
					if err != nil {
						results <- err
						return
					}
					if account.Root != subroot {
						results <- fmt.Errorf("invalid subroot(path %x), want %x, have %x", hash, account.Root, subroot)
						return
					}
					results <- nil
				}(it.Hash())
				fullData, err = rlp.EncodeToBytes(account)
				if err != nil {
					return stop(err)
				}
			}
			leaf = trieKV{it.Hash(), fullData}
		} else {
			leaf = trieKV{it.Hash(), common.CopyBytes(it.(StorageIterator).Slot())}
		}
		in <- leaf

		processed++
		if time.Since(logged) > 3*time.Second && stats != nil {
			if account == (common.Hash{}) {
				stats.progressAccounts(it.Hash(), processed)
			} else {
				stats.progressContract(account, it.Hash(), processed)
			}
			logged, processed = time.Now(), 0
		}
	}
	if processed > 0 && stats != nil {
		if account == (common.Hash{}) {
			stats.finishAccounts(processed)
		} else {
			stats.finishContract(account, processed)
		}
	}
	return stop(nil)
}

func stackTrieGenerate(db ethdb.KeyValueWriter, scheme string, owner common.Hash, in chan trieKV, out chan common.Hash) {
	var onTrieNode trie.OnTrieNode
	if db != nil {
		onTrieNode = func(path []byte, hash common.Hash, blob []byte) {
			rawdb.WriteTrieNode(db, owner, path, hash, blob, scheme)
		}
	}
	t := trie.NewStackTrie(onTrieNode)
	for leaf := range in {
		t.Update(leaf.key[:], leaf.value)
	}
	out <- t.Hash()
}
