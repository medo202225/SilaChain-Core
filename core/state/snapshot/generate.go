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
	"errors"
	"fmt"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"silachain/common"
	"silachain/common/hexutil"
	"silachain/core/rawdb"
	"silachain/core/types"
	"silachain/ethdb"
	"silachain/log"
	"silachain/rlp"
	"silachain/trie"
	"silachain/triedb"
)

var (
	accountCheckRange = 128
	storageCheckRange = 1024
	errMissingTrie    = errors.New("missing trie")
)

// generateSnapshot regenerates a brand new snapshot based on an existing state database.
func generateSnapshot(diskdb ethdb.KeyValueStore, triedb *triedb.Database, cache int, root common.Hash) *diskLayer {
	var (
		stats     = &generatorStats{start: time.Now()}
		batch     = diskdb.NewBatch()
		genMarker = []byte{}
	)
	rawdb.WriteSnapshotRoot(batch, root)
	journalProgress(batch, genMarker, stats)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write initialized state marker", "err", err)
	}
	base := &diskLayer{
		diskdb:     diskdb,
		triedb:     triedb,
		root:       root,
		cache:      fastcache.New(cache * 1024 * 1024),
		genMarker:  genMarker,
		genPending: make(chan struct{}),
		genAbort:   make(chan chan *generatorStats),
	}
	go base.generate(stats)
	log.Debug("Start snapshot generation", "root", root)
	return base
}

// journalProgress persists the generator stats into the database to resume later.
func journalProgress(db ethdb.KeyValueWriter, marker []byte, stats *generatorStats) {
	entry := journalGenerator{
		Done:   marker == nil,
		Marker: marker,
	}
	if stats != nil {
		entry.Accounts = stats.accounts
		entry.Slots = stats.slots
		entry.Storage = uint64(stats.storage)
	}
	blob, err := rlp.EncodeToBytes(entry)
	if err != nil {
		panic(err)
	}
	var logstr string
	switch {
	case marker == nil:
		logstr = "done"
	case bytes.Equal(marker, []byte{}):
		logstr = "empty"
	case len(marker) == common.HashLength:
		logstr = fmt.Sprintf("%#x", marker)
	default:
		logstr = fmt.Sprintf("%#x:%#x", marker[:common.HashLength], marker[common.HashLength:])
	}
	log.Debug("Journalled generator progress", "progress", logstr)
	rawdb.WriteSnapshotGenerator(db, blob)
}

// proofResult contains the output of range proving.
type proofResult struct {
	keys     [][]byte
	vals     [][]byte
	diskMore bool
	trieMore bool
	proofErr error
	tr       *trie.Trie
}

// valid returns the indicator that range proof is successful or not.
func (result *proofResult) valid() bool {
	return result.proofErr == nil
}

// last returns the last verified element key.
func (result *proofResult) last() []byte {
	var last []byte
	if len(result.keys) > 0 {
		last = result.keys[len(result.keys)-1]
	}
	return last
}

// forEach iterates all the visited elements.
func (result *proofResult) forEach(callback func(key []byte, val []byte) error) error {
	for i := 0; i < len(result.keys); i++ {
		key, val := result.keys[i], result.vals[i]
		if err := callback(key, val); err != nil {
			return err
		}
	}
	return nil
}

// proveRange proves the snapshot segment with particular prefix is "valid".
func (dl *diskLayer) proveRange(ctx *generatorContext, trieId *trie.ID, prefix []byte, kind string, origin []byte, max int, valueConvertFn func([]byte) ([]byte, error)) (*proofResult, error) {
	var (
		keys     [][]byte
		vals     [][]byte
		proof    = rawdb.NewMemoryDatabase()
		diskMore = false
		iter     = ctx.iterator(kind)
		start    = time.Now()
		min      = append(prefix, origin...)
	)
	for iter.Next() {
		key := iter.Key()
		if bytes.Compare(key, min) < 0 {
			return nil, errors.New("invalid iteration position")
		}
		if !bytes.Equal(key[:len(prefix)], prefix) {
			iter.Hold()
			break
		}
		if len(keys) == max {
			iter.Hold()
			diskMore = true
			break
		}
		keys = append(keys, common.CopyBytes(key[len(prefix):]))

		if valueConvertFn == nil {
			vals = append(vals, common.CopyBytes(iter.Value()))
		} else {
			val, err := valueConvertFn(iter.Value())
			if err != nil {
				vals = append(vals, common.CopyBytes(iter.Value()))
				log.Error("Failed to convert account state data", "err", err)
			} else {
				vals = append(vals, val)
			}
		}
	}
	if kind == snapStorage {
		snapStorageSnapReadCounter.Inc(time.Since(start).Nanoseconds())
	} else {
		snapAccountSnapReadCounter.Inc(time.Since(start).Nanoseconds())
	}
	defer func(start time.Time) {
		if kind == snapStorage {
			snapStorageProveCounter.Inc(time.Since(start).Nanoseconds())
		} else {
			snapAccountProveCounter.Inc(time.Since(start).Nanoseconds())
		}
	}(time.Now())

	root := trieId.Root
	if origin == nil && !diskMore {
		stackTr := trie.NewStackTrie(nil)
		for i, key := range keys {
			if err := stackTr.Update(key, vals[i]); err != nil {
				return nil, err
			}
		}
		if gotRoot := stackTr.Hash(); gotRoot != root {
			return &proofResult{
				keys:     keys,
				vals:     vals,
				proofErr: fmt.Errorf("wrong root: have %#x want %#x", gotRoot, root),
			}, nil
		}
		return &proofResult{keys: keys, vals: vals}, nil
	}
	tr, err := trie.New(trieId, dl.triedb)
	if err != nil {
		ctx.stats.Log("Trie missing, state snapshotting paused", dl.root, dl.genMarker)
		return nil, errMissingTrie
	}
	if origin == nil {
		origin = common.Hash{}.Bytes()
	}
	if err := tr.Prove(origin, proof); err != nil {
		log.Debug("Failed to prove range", "kind", kind, "origin", origin, "err", err)
		return &proofResult{
			keys:     keys,
			vals:     vals,
			diskMore: diskMore,
			proofErr: err,
			tr:       tr,
		}, nil
	}
	if len(keys) > 0 {
		if err := tr.Prove(keys[len(keys)-1], proof); err != nil {
			log.Debug("Failed to prove range", "kind", kind, "last", keys[len(keys)-1], "err", err)
			return &proofResult{
				keys:     keys,
				vals:     vals,
				diskMore: diskMore,
				proofErr: err,
				tr:       tr,
			}, nil
		}
	}
	cont, err := trie.VerifyRangeProof(root, origin, keys, vals, proof)
	return &proofResult{
			keys:     keys,
			vals:     vals,
			diskMore: diskMore,
			trieMore: cont,
			proofErr: err,
			tr:       tr},
		nil
}

// onStateCallback is a function that is called by generateRange.
type onStateCallback func(key []byte, val []byte, write bool, delete bool) error

// generateRange generates the state segment with particular prefix.
func (dl *diskLayer) generateRange(ctx *generatorContext, trieId *trie.ID, prefix []byte, kind string, origin []byte, max int, onState onStateCallback, valueConvertFn func([]byte) ([]byte, error)) (bool, []byte, error) {
	result, err := dl.proveRange(ctx, trieId, prefix, kind, origin, max, valueConvertFn)
	if err != nil {
		return false, nil, err
	}
	last := result.last()

	logCtx := []interface{}{"kind", kind, "prefix", hexutil.Encode(prefix)}
	if len(origin) > 0 {
		logCtx = append(logCtx, "origin", hexutil.Encode(origin))
	}
	logger := log.New(logCtx...)

	if result.valid() {
		snapSuccessfulRangeProofMeter.Mark(1)
		logger.Trace("Proved state range", "last", hexutil.Encode(last))

		if err := result.forEach(func(key []byte, val []byte) error { return onState(key, val, false, false) }); err != nil {
			return false, nil, err
		}
		return !result.diskMore && !result.trieMore, last, nil
	}
	logger.Trace("Detected outdated state range", "last", hexutil.Encode(last), "err", result.proofErr)
	snapFailedRangeProofMeter.Mark(1)

	if origin == nil && last == nil {
		meter := snapMissallAccountMeter
		if kind == snapStorage {
			meter = snapMissallStorageMeter
		}
		meter.Mark(1)
	}
	var resolver trie.NodeResolver
	if len(result.keys) > 0 {
		tr := trie.NewEmpty(nil)
		for i, key := range result.keys {
			tr.Update(key, result.vals[i])
		}
		_, nodes := tr.Commit(false)
		hashSet := nodes.HashSet()
		resolver = func(owner common.Hash, path []byte, hash common.Hash) []byte {
			return hashSet[hash]
		}
	}
	tr := result.tr
	if tr == nil {
		tr, err = trie.New(trieId, dl.triedb)
		if err != nil {
			ctx.stats.Log("Trie missing, state snapshotting paused", dl.root, dl.genMarker)
			return false, nil, errMissingTrie
		}
	}
	var (
		trieMore       bool
		kvkeys, kvvals = result.keys, result.vals

		count     = 0
		created   = 0
		updated   = 0
		deleted   = 0
		untouched = 0

		start    = time.Now()
		internal time.Duration
	)
	nodeIt, err := tr.NodeIterator(origin)
	if err != nil {
		return false, nil, err
	}
	nodeIt.AddResolver(resolver)
	iter := trie.NewIterator(nodeIt)

	for iter.Next() {
		if last != nil && bytes.Compare(iter.Key, last) > 0 {
			trieMore = true
			break
		}
		count++
		write := true
		created++
		for len(kvkeys) > 0 {
			if cmp := bytes.Compare(kvkeys[0], iter.Key); cmp < 0 {
				istart := time.Now()
				if err := onState(kvkeys[0], nil, false, true); err != nil {
					return false, nil, err
				}
				kvkeys = kvkeys[1:]
				kvvals = kvvals[1:]
				deleted++
				internal += time.Since(istart)
				continue
			} else if cmp == 0 {
				created--
				if write = !bytes.Equal(kvvals[0], iter.Value); write {
					updated++
				} else {
					untouched++
				}
				kvkeys = kvkeys[1:]
				kvvals = kvvals[1:]
			}
			break
		}
		istart := time.Now()
		if err := onState(iter.Key, iter.Value, write, false); err != nil {
			return false, nil, err
		}
		internal += time.Since(istart)
	}
	if iter.Err != nil {
		log.Error("State snapshotter failed to iterate trie", "err", iter.Err)
		return false, nil, iter.Err
	}
	istart := time.Now()
	for _, key := range kvkeys {
		if err := onState(key, nil, false, true); err != nil {
			return false, nil, err
		}
		deleted += 1
	}
	internal += time.Since(istart)

	if kind == snapStorage {
		snapStorageTrieReadCounter.Inc((time.Since(start) - internal).Nanoseconds())
	} else {
		snapAccountTrieReadCounter.Inc((time.Since(start) - internal).Nanoseconds())
	}
	logger.Debug("Regenerated state range", "root", trieId.Root, "last", hexutil.Encode(last),
		"count", count, "created", created, "updated", updated, "untouched", untouched, "deleted", deleted)

	return !trieMore && !result.diskMore, last, nil
}

// checkAndFlush checks if an interruption signal is received or the batch size has exceeded the allowance.
func (dl *diskLayer) checkAndFlush(ctx *generatorContext, current []byte) error {
	var abort chan *generatorStats
	select {
	case abort = <-dl.genAbort:
	default:
	}
	if ctx.batch.ValueSize() > ethdb.IdealBatchSize || abort != nil {
		if bytes.Compare(current, dl.genMarker) < 0 {
			log.Error("Snapshot generator went backwards", "current", fmt.Sprintf("%x", current), "genMarker", fmt.Sprintf("%x", dl.genMarker))
		}
		journalProgress(ctx.batch, current, ctx.stats)

		if err := ctx.batch.Write(); err != nil {
			return err
		}
		ctx.batch.Reset()

		dl.lock.Lock()
		dl.genMarker = current
		dl.lock.Unlock()

		if abort != nil {
			ctx.stats.Log("Aborting state snapshot generation", dl.root, current)
			return newAbortErr(abort)
		}
		ctx.reopenIterator(snapAccount)
		ctx.reopenIterator(snapStorage)
	}
	if time.Since(ctx.logged) > 8*time.Second {
		ctx.stats.Log("Generating state snapshot", dl.root, current)
		ctx.logged = time.Now()
	}
	return nil
}

// generateStorages generates the missing storage slots of the specific contract.
func generateStorages(ctx *generatorContext, dl *diskLayer, stateRoot common.Hash, account common.Hash, storageRoot common.Hash, storeMarker []byte) error {
	onStorage := func(key []byte, val []byte, write bool, delete bool) error {
		defer func(start time.Time) {
			snapStorageWriteCounter.Inc(time.Since(start).Nanoseconds())
		}(time.Now())

		if delete {
			rawdb.DeleteStorageSnapshot(ctx.batch, account, common.BytesToHash(key))
			snapWipedStorageMeter.Mark(1)
			return nil
		}
		if write {
			rawdb.WriteStorageSnapshot(ctx.batch, account, common.BytesToHash(key), val)
			snapGeneratedStorageMeter.Mark(1)
		} else {
			snapRecoveredStorageMeter.Mark(1)
		}
		ctx.stats.storage += common.StorageSize(1 + 2*common.HashLength + len(val))
		ctx.stats.slots++

		if err := dl.checkAndFlush(ctx, append(account[:], key...)); err != nil {
			return err
		}
		return nil
	}
	var origin = common.CopyBytes(storeMarker)
	for {
		id := trie.StorageTrieID(stateRoot, account, storageRoot)
		exhausted, last, err := dl.generateRange(ctx, id, append(rawdb.SnapshotStoragePrefix, account.Bytes()...), snapStorage, origin, storageCheckRange, onStorage, nil)
		if err != nil {
			return err
		}
		if exhausted {
			break
		}
		if origin = increaseKey(last); origin == nil {
			break
		}
	}
	return nil
}

// generateAccounts generates the missing snapshot accounts.
func generateAccounts(ctx *generatorContext, dl *diskLayer, accMarker []byte) error {
	onAccount := func(key []byte, val []byte, write bool, delete bool) error {
		account := common.BytesToHash(key)
		ctx.removeStorageBefore(account)

		start := time.Now()
		if delete {
			rawdb.DeleteAccountSnapshot(ctx.batch, account)
			snapWipedAccountMeter.Mark(1)
			snapAccountWriteCounter.Inc(time.Since(start).Nanoseconds())

			ctx.removeStorageAt(account)
			return nil
		}
		var acc types.StateAccount
		if err := rlp.DecodeBytes(val, &acc); err != nil {
			log.Crit("Invalid account encountered during snapshot creation", "err", err)
		}
		if accMarker == nil || !bytes.Equal(account[:], accMarker) {
			dataLen := len(val)
			if !write {
				if bytes.Equal(acc.CodeHash, types.EmptyCodeHash[:]) {
					dataLen -= 32
				}
				if acc.Root == types.EmptyRootHash {
					dataLen -= 32
				}
				snapRecoveredAccountMeter.Mark(1)
			} else {
				data := types.SlimAccountRLP(acc)
				dataLen = len(data)
				rawdb.WriteAccountSnapshot(ctx.batch, account, data)
				snapGeneratedAccountMeter.Mark(1)
			}
			ctx.stats.storage += common.StorageSize(1 + common.HashLength + dataLen)
			ctx.stats.accounts++
		}
		marker := account[:]
		if accMarker != nil && bytes.Equal(marker, accMarker) && len(dl.genMarker) > common.HashLength {
			marker = dl.genMarker[:]
		}
		if err := dl.checkAndFlush(ctx, marker); err != nil {
			return err
		}
		snapAccountWriteCounter.Inc(time.Since(start).Nanoseconds())

		if acc.Root == types.EmptyRootHash {
			ctx.removeStorageAt(account)
		} else {
			var storeMarker []byte
			if accMarker != nil && bytes.Equal(account[:], accMarker) && len(dl.genMarker) > common.HashLength {
				storeMarker = dl.genMarker[common.HashLength:]
			}
			if err := generateStorages(ctx, dl, dl.root, account, acc.Root, storeMarker); err != nil {
				return err
			}
		}
		accMarker = nil
		return nil
	}
	origin := common.CopyBytes(accMarker)
	for {
		id := trie.StateTrieID(dl.root)
		exhausted, last, err := dl.generateRange(ctx, id, rawdb.SnapshotAccountPrefix, snapAccount, origin, accountCheckRange, onAccount, types.FullAccountRLP)
		if err != nil {
			return err
		}
		origin = increaseKey(last)

		if origin == nil || exhausted {
			ctx.removeStorageLeft()
			break
		}
	}
	return nil
}

// generate is a background thread that iterates over the state and storage tries.
func (dl *diskLayer) generate(stats *generatorStats) {
	var (
		accMarker []byte
		abort     chan *generatorStats
	)
	if len(dl.genMarker) > 0 {
		accMarker = dl.genMarker[:common.HashLength]
	}
	stats.Log("Resuming state snapshot generation", dl.root, dl.genMarker)

	ctx := newGeneratorContext(stats, dl.diskdb, accMarker, dl.genMarker)
	defer ctx.close()

	if err := generateAccounts(ctx, dl, accMarker); err != nil {
		if aerr, ok := err.(*abortErr); ok {
			abort = aerr.abort
		}
		if abort == nil {
			abort = <-dl.genAbort
		}
		abort <- stats
		return
	}
	journalProgress(ctx.batch, nil, stats)
	if err := ctx.batch.Write(); err != nil {
		log.Error("Failed to flush batch", "err", err)

		abort = <-dl.genAbort
		abort <- stats
		return
	}
	ctx.batch.Reset()

	log.Info("Generated state snapshot", "accounts", stats.accounts, "slots", stats.slots,
		"storage", stats.storage, "dangling", stats.dangling, "elapsed", common.PrettyDuration(time.Since(stats.start)))

	dl.lock.Lock()
	dl.genMarker = nil
	close(dl.genPending)
	dl.lock.Unlock()

	abort = <-dl.genAbort
	abort <- nil
}

// increaseKey increase the input key by one bit.
func increaseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			return key
		}
	}
	return nil
}

// abortErr wraps an interruption signal.
type abortErr struct {
	abort chan *generatorStats
}

func newAbortErr(abort chan *generatorStats) error {
	return &abortErr{abort: abort}
}

func (err *abortErr) Error() string {
	return "aborted"
}
