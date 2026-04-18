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

package filtermaps

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"silachain/common"
	"silachain/common/lru"
	"silachain/core/rawdb"
	"silachain/core/types"
	"silachain/ethdb"
	"silachain/log"
	"silachain/metrics"
)

var (
	mapCountGauge           = metrics.NewRegisteredGauge("filtermaps/maps/count", nil)
	mapLogValueMeter        = metrics.NewRegisteredMeter("filtermaps/maps/logvalues", nil)
	mapBlockMeter           = metrics.NewRegisteredMeter("filtermaps/maps/blocks", nil)
	mapRenderTimer          = metrics.NewRegisteredTimer("filtermaps/maps/rendertime", nil)
	mapWriteTimer           = metrics.NewRegisteredTimer("filtermaps/maps/writetime", nil)
	matchRequestTimer       = metrics.NewRegisteredTimer("filtermaps/match/requesttime", nil)
	matchEpochTimer         = metrics.NewRegisteredTimer("filtermaps/match/epochtime", nil)
	matchBaseRowAccessMeter = metrics.NewRegisteredMeter("filtermaps/match/baserowaccess", nil)
	matchBaseRowSizeMeter   = metrics.NewRegisteredMeter("filtermaps/match/baserowsize", nil)
	matchExtRowAccessMeter  = metrics.NewRegisteredMeter("filtermaps/match/extrowaccess", nil)
	matchExtRowSizeMeter    = metrics.NewRegisteredMeter("filtermaps/match/extrowsize", nil)
	matchLogLookup          = metrics.NewRegisteredMeter("filtermaps/match/loglookup", nil)
	matchAllMeter           = metrics.NewRegisteredMeter("filtermaps/match/matchall", nil)
)

const (
	databaseVersion       = 2
	cachedLastBlocks      = 1000
	cachedLvPointers      = 1000
	cachedFilterMaps      = 3
	cachedRenderSnapshots = 8
)

// FilterMaps is the in-memory representation of the log index structure that is
// responsible for building and updating the index according to the canonical
// chain.
//
// Note that FilterMaps implements the same data structure as proposed in EIP-7745
// without the tree hashing and consensus changes:
// https://eips.ethereum.org/EIPS/eip-7745
type FilterMaps struct {
	disabled   bool
	disabledCh chan struct{}

	closeCh        chan struct{}
	closeWg        sync.WaitGroup
	history        uint64
	hashScheme     bool
	exportFileName string
	Params

	db ethdb.KeyValueStore

	indexLock    sync.RWMutex
	indexedRange filterMapsRange
	indexedView  *ChainView
	hasTempRange bool

	cleanedEpochsBefore uint32

	filterMapCache *lru.Cache[uint32, filterMap]
	lastBlockCache *lru.Cache[uint32, lastBlockOfMap]
	lvPointerCache *lru.Cache[uint64, uint64]

	matchersLock sync.Mutex
	matchers     map[*FilterMapsMatcherBackend]struct{}

	renderSnapshots                                              *lru.Cache[uint64, *renderedMap]
	startedHeadIndex, startedTailIndex, startedTailUnindex       bool
	startedHeadIndexAt, startedTailIndexAt, startedTailUnindexAt time.Time
	loggedHeadIndex, loggedTailIndex                             bool
	lastLogHeadIndex, lastLogTailIndex                           time.Time
	ptrHeadIndex, ptrTailIndex, ptrTailUnindexBlock              uint64
	ptrTailUnindexMap                                            uint32

	targetView            *ChainView
	matcherSyncRequests   []*FilterMapsMatcherBackend
	historyCutoff         uint64
	finalBlock, lastFinal uint64
	lastFinalEpoch        uint32
	stop                  bool
	targetCh              chan targetUpdate
	blockProcessingCh     chan bool
	blockProcessing       bool
	matcherSyncCh         chan *FilterMapsMatcherBackend
	waitIdleCh            chan chan bool
	tailRenderer          *mapRenderer

	testDisableSnapshots, testSnapshotUsed bool
	testProcessEventsHook                  func()
}

type filterMap []FilterRow

func (fm filterMap) fastCopy() filterMap {
	return slices.Clone(fm)
}

func (fm filterMap) fullCopy() filterMap {
	c := make(filterMap, len(fm))
	for i, row := range fm {
		c[i] = slices.Clone(row)
	}
	return c
}

type FilterRow []uint32

func (a FilterRow) Equal(b FilterRow) bool {
	return slices.Equal(a, b)
}

type filterMapsRange struct {
	initialized      bool
	headIndexed      bool
	headDelimiter    uint64
	maps             common.Range[uint32]
	tailPartialEpoch uint32
	blocks           common.Range[uint64]
}

func (fmr *filterMapsRange) hasIndexedBlocks() bool {
	return fmr.initialized && !fmr.blocks.IsEmpty() && !fmr.maps.IsEmpty()
}

type lastBlockOfMap struct {
	number uint64
	id     common.Hash
}

type Config struct {
	History        uint64
	Disabled       bool
	ExportFileName string
	HashScheme     bool
}

func NewFilterMaps(db ethdb.KeyValueStore, initView *ChainView, historyCutoff, finalBlock uint64, params Params, config Config) (*FilterMaps, error) {
	rs, initialized, err := rawdb.ReadFilterMapsRange(db)
	if err != nil || (initialized && rs.Version != databaseVersion) {
		rs, initialized = rawdb.FilterMapsRange{}, false
		log.Warn("Invalid log index database version; resetting log index")
	}
	if err := params.sanitize(); err != nil {
		return nil, err
	}
	f := &FilterMaps{
		db:                db,
		closeCh:           make(chan struct{}),
		waitIdleCh:        make(chan chan bool),
		targetCh:          make(chan targetUpdate, 1),
		blockProcessingCh: make(chan bool, 1),
		history:           config.History,
		disabled:          config.Disabled,
		hashScheme:        config.HashScheme,
		disabledCh:        make(chan struct{}),
		exportFileName:    config.ExportFileName,
		Params:            params,
		targetView:        initView,
		indexedView:       initView,
		indexedRange: filterMapsRange{
			initialized:      initialized,
			headIndexed:      rs.HeadIndexed,
			headDelimiter:    rs.HeadDelimiter,
			blocks:           common.NewRange(rs.BlocksFirst, rs.BlocksAfterLast-rs.BlocksFirst),
			maps:             common.NewRange(rs.MapsFirst, rs.MapsAfterLast-rs.MapsFirst),
			tailPartialEpoch: rs.TailPartialEpoch,
		},
		cleanedEpochsBefore: max(rs.MapsFirst>>params.logMapsPerEpoch, 1) - 1,
		historyCutoff:       historyCutoff,
		finalBlock:          finalBlock,
		matcherSyncCh:       make(chan *FilterMapsMatcherBackend),
		matchers:            make(map[*FilterMapsMatcherBackend]struct{}),
		filterMapCache:      lru.NewCache[uint32, filterMap](cachedFilterMaps),
		lastBlockCache:      lru.NewCache[uint32, lastBlockOfMap](cachedLastBlocks),
		lvPointerCache:      lru.NewCache[uint64, uint64](cachedLvPointers),
		renderSnapshots:     lru.NewCache[uint64, *renderedMap](cachedRenderSnapshots),
	}
	f.checkRevertRange()

	if f.indexedRange.hasIndexedBlocks() {
		log.Info("Initialized log indexer",
			"firstblock", f.indexedRange.blocks.First(), "lastblock", f.indexedRange.blocks.Last(),
			"firstmap", f.indexedRange.maps.First(), "lastmap", f.indexedRange.maps.Last(),
			"headindexed", f.indexedRange.headIndexed)
	}
	return f, nil
}

func (f *FilterMaps) Start() {
	if !f.testDisableSnapshots && f.indexedRange.hasIndexedBlocks() && f.indexedRange.headIndexed {
		if err := f.loadHeadSnapshot(); err != nil {
			log.Error("Could not load head filter map snapshot", "error", err)
		}
	}
	f.closeWg.Add(2)
	go f.removeBloomBits()
	go f.indexerLoop()
}

func (f *FilterMaps) Stop() {
	close(f.closeCh)
	f.closeWg.Wait()
}

func (f *FilterMaps) checkRevertRange() {
	if f.indexedRange.maps.Count() == 0 {
		return
	}
	lastMap := f.indexedRange.maps.Last()
	lastBlockNumber, lastBlockId, err := f.getLastBlockOfMap(lastMap)
	if err != nil {
		log.Error("Error initializing log index database; resetting log index", "error", err)
		f.reset()
		return
	}
	for lastBlockNumber > f.indexedView.HeadNumber() || f.indexedView.BlockId(lastBlockNumber) != lastBlockId {
		if f.indexedRange.maps.Count() == 1 {
			f.reset()
			return
		}
		lastMap--
		newRange := f.indexedRange
		newRange.maps.SetLast(lastMap)
		lastBlockNumber, lastBlockId, err = f.getLastBlockOfMap(lastMap)
		if err != nil {
			log.Error("Error initializing log index database; resetting log index", "error", err)
			f.reset()
			return
		}
		newRange.blocks.SetAfterLast(lastBlockNumber)
		newRange.headIndexed = false
		newRange.headDelimiter = 0
		f.setRange(f.db, f.indexedView, newRange, false)
	}
}

func (f *FilterMaps) reset() {
	f.indexLock.Lock()
	f.indexedRange = filterMapsRange{}
	f.indexedView = nil
	f.filterMapCache.Purge()
	f.renderSnapshots.Purge()
	f.lastBlockCache.Purge()
	f.lvPointerCache.Purge()
	f.indexLock.Unlock()
	rawdb.DeleteFilterMapsRange(f.db)
	f.safeDeleteWithLogs(rawdb.DeleteFilterMapsDb, "Resetting log index database", f.isShuttingDown)
}

func (f *FilterMaps) isShuttingDown() bool {
	select {
	case <-f.closeCh:
		return true
	default:
		return false
	}
}

func (f *FilterMaps) init() error {
	if err := f.safeDeleteWithLogs(rawdb.DeleteFilterMapsDb, "Resetting log index database", f.isShuttingDown); err != nil {
		return err
	}

	f.indexLock.Lock()
	defer f.indexLock.Unlock()

	var bestIdx, bestLen int
	for idx, checkpointList := range checkpoints {
		min, max := 0, len(checkpointList)
		for min < max {
			mid := (min + max + 1) / 2
			cp := checkpointList[mid-1]
			if cp.BlockNumber <= f.targetView.HeadNumber() && f.targetView.BlockId(cp.BlockNumber) == cp.BlockId {
				min = mid
			} else {
				max = mid - 1
			}
		}
		if max > bestLen {
			bestIdx, bestLen = idx, max
		}
	}
	var initBlockNumber uint64
	if bestLen > 0 {
		initBlockNumber = checkpoints[bestIdx][bestLen-1].BlockNumber
	}
	if initBlockNumber < f.historyCutoff {
		return errors.New("cannot start indexing before history cutoff point")
	}
	batch := f.db.NewBatch()
	for epoch := range bestLen {
		cp := checkpoints[bestIdx][epoch]
		f.storeLastBlockOfMap(batch, f.lastEpochMap(uint32(epoch)), cp.BlockNumber, cp.BlockId)
		f.storeBlockLvPointer(batch, cp.BlockNumber, cp.FirstIndex)
	}
	fmr := filterMapsRange{
		initialized: true,
	}
	if bestLen > 0 {
		cp := checkpoints[bestIdx][bestLen-1]
		fmr.blocks = common.NewRange(cp.BlockNumber+1, 0)
		fmr.maps = common.NewRange(f.firstEpochMap(uint32(bestLen)), 0)
	}
	f.setRange(batch, f.targetView, fmr, false)
	return batch.Write()
}

func (f *FilterMaps) removeBloomBits() {
	f.safeDeleteWithLogs(rawdb.DeleteBloomBitsDb, "Removing old bloom bits database", f.isShuttingDown)
	f.closeWg.Done()
}

func (f *FilterMaps) safeDeleteWithLogs(deleteFn func(db ethdb.KeyValueStore, hashScheme bool, stopCb func(bool) bool) error, action string, stopCb func() bool) error {
	var (
		start          = time.Now()
		logPrinted     bool
		lastLogPrinted = start
	)
	switch err := deleteFn(f.db, f.hashScheme, func(deleted bool) bool {
		if deleted && (!logPrinted || time.Since(lastLogPrinted) > time.Second*10) {
			log.Info(action+" in progress...", "elapsed", common.PrettyDuration(time.Since(start)))
			logPrinted, lastLogPrinted = true, time.Now()
		}
		return stopCb()
	}); {
	case err == nil:
		if logPrinted {
			log.Info(action+" finished", "elapsed", common.PrettyDuration(time.Since(start)))
		}
		return nil
	case errors.Is(err, rawdb.ErrDeleteRangeInterrupted):
		log.Warn(action+" interrupted", "elapsed", common.PrettyDuration(time.Since(start)))
		return err
	default:
		log.Error(action+" failed", "error", err)
		return err
	}
}

func (f *FilterMaps) setRange(batch ethdb.KeyValueWriter, newView *ChainView, newRange filterMapsRange, isTempRange bool) {
	f.indexedView = newView
	f.indexedRange = newRange
	f.hasTempRange = isTempRange
	f.updateMatchersValidRange()
	if newRange.initialized {
		rs := rawdb.FilterMapsRange{
			Version:          databaseVersion,
			HeadIndexed:      newRange.headIndexed,
			HeadDelimiter:    newRange.headDelimiter,
			BlocksFirst:      newRange.blocks.First(),
			BlocksAfterLast:  newRange.blocks.AfterLast(),
			MapsFirst:        newRange.maps.First(),
			MapsAfterLast:    newRange.maps.AfterLast(),
			TailPartialEpoch: newRange.tailPartialEpoch,
		}
		rawdb.WriteFilterMapsRange(batch, rs)
		if !isTempRange {
			mapCountGauge.Update(int64(newRange.maps.Count() + newRange.tailPartialEpoch))
		}
	} else {
		rawdb.DeleteFilterMapsRange(batch)
		mapCountGauge.Update(0)
	}
}

func (f *FilterMaps) getLogByLvIndex(lvIndex uint64) (*types.Log, error) {
	mapIndex := uint32(lvIndex >> f.logValuesPerMap)
	if !f.indexedRange.maps.Includes(mapIndex) {
		return nil, nil
	}
	lastBlockNumber, _, err := f.getLastBlockOfMap(mapIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve last block of map %d containing searched log value index %d: %v", mapIndex, lvIndex, err)
	}
	var firstBlockNumber uint64
	if mapIndex > 0 {
		firstBlockNumber, _, err = f.getLastBlockOfMap(mapIndex - 1)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve last block of map %d before searched log value index %d: %v", mapIndex, lvIndex, err)
		}
	}
	if firstBlockNumber < f.indexedRange.blocks.First() {
		firstBlockNumber = f.indexedRange.blocks.First()
	}
	for firstBlockNumber < lastBlockNumber {
		midBlockNumber := (firstBlockNumber + lastBlockNumber + 1) / 2
		midLvPointer, err := f.getBlockLvPointer(midBlockNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve log value pointer of block %d while binary searching log value index %d: %v", midBlockNumber, lvIndex, err)
		}
		if lvIndex < midLvPointer {
			lastBlockNumber = midBlockNumber - 1
		} else {
			firstBlockNumber = midBlockNumber
		}
	}
	receipts := f.indexedView.Receipts(firstBlockNumber)
	if receipts == nil {
		return nil, fmt.Errorf("failed to retrieve receipts for block %d containing searched log value index %d: %v", firstBlockNumber, lvIndex, err)
	}
	lvPointer, err := f.getBlockLvPointer(firstBlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log value pointer of block %d containing searched log value index %d: %v", firstBlockNumber, lvIndex, err)
	}
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			l := uint64(len(log.Topics) + 1)
			r := f.valuesPerMap - lvPointer%f.valuesPerMap
			if l > r {
				lvPointer += r
			}
			if lvPointer > lvIndex {
				return nil, nil
			}
			if lvPointer == lvIndex {
				return log, nil
			}
			lvPointer += l
		}
	}
	return nil, nil
}

func (f *FilterMaps) getFilterMap(mapIndex uint32) (filterMap, error) {
	if fm, ok := f.filterMapCache.Get(mapIndex); ok {
		return fm, nil
	}
	fm := make(filterMap, f.mapHeight)
	for rowIndex := range fm {
		rows, err := f.getFilterMapRows([]uint32{mapIndex}, uint32(rowIndex), false)
		if err != nil {
			return nil, fmt.Errorf("failed to load filter map %d from database: %v", mapIndex, err)
		}
		fm[rowIndex] = rows[0]
	}
	f.filterMapCache.Add(mapIndex, fm)
	return fm, nil
}

func (f *FilterMaps) getFilterMapRows(mapIndices []uint32, rowIndex uint32, baseLayerOnly bool) ([]FilterRow, error) {
	rows := make([]FilterRow, len(mapIndices))
	var ptr int
	for len(mapIndices) > ptr {
		var (
			groupIndex  = f.mapGroupIndex(mapIndices[ptr])
			groupLength = 1
		)
		for ptr+groupLength < len(mapIndices) && f.mapGroupIndex(mapIndices[ptr+groupLength]) == groupIndex {
			groupLength++
		}
		if err := f.getFilterMapRowsOfGroup(rows[ptr:ptr+groupLength], mapIndices[ptr:ptr+groupLength], rowIndex, baseLayerOnly); err != nil {
			return nil, err
		}
		ptr += groupLength
	}
	return rows, nil
}

func (f *FilterMaps) getFilterMapRowsOfGroup(target []FilterRow, mapIndices []uint32, rowIndex uint32, baseLayerOnly bool) error {
	var (
		groupIndex  = f.mapGroupIndex(mapIndices[0])
		mapRowIndex = f.mapRowIndex(groupIndex, rowIndex)
	)
	baseRows, err := rawdb.ReadFilterMapBaseRows(f.db, mapRowIndex, f.baseRowGroupSize, f.logMapWidth)
	if err != nil {
		return fmt.Errorf("failed to retrieve base row group %d of row %d: %v", groupIndex, rowIndex, err)
	}
	for i, mapIndex := range mapIndices {
		if f.mapGroupIndex(mapIndex) != groupIndex {
			return fmt.Errorf("maps are not in the same base row group, index: %d, group: %d", mapIndex, groupIndex)
		}
		row := baseRows[f.mapGroupOffset(mapIndex)]
		if !baseLayerOnly {
			extRow, err := rawdb.ReadFilterMapExtRow(f.db, f.mapRowIndex(mapIndex, rowIndex), f.logMapWidth)
			if err != nil {
				return fmt.Errorf("failed to retrieve filter map %d extended row %d: %v", mapIndex, rowIndex, err)
			}
			row = append(row, extRow...)
		}
		target[i] = row
	}
	return nil
}

func (f *FilterMaps) storeFilterMapRows(batch ethdb.Batch, mapIndices []uint32, rowIndex uint32, rows []FilterRow) error {
	for len(mapIndices) > 0 {
		var (
			pos        = 1
			groupIndex = f.mapGroupIndex(mapIndices[0])
		)
		for pos < len(mapIndices) && f.mapGroupIndex(mapIndices[pos]) == groupIndex {
			pos++
		}
		if err := f.storeFilterMapRowsOfGroup(batch, mapIndices[:pos], rowIndex, rows[:pos]); err != nil {
			return err
		}
		mapIndices, rows = mapIndices[pos:], rows[pos:]
	}
	return nil
}

func (f *FilterMaps) storeFilterMapRowsOfGroup(batch ethdb.Batch, mapIndices []uint32, rowIndex uint32, rows []FilterRow) error {
	var (
		baseRows    [][]uint32
		groupIndex  = f.mapGroupIndex(mapIndices[0])
		mapRowIndex = f.mapRowIndex(groupIndex, rowIndex)
	)
	if uint32(len(mapIndices)) != f.baseRowGroupSize {
		var err error
		baseRows, err = rawdb.ReadFilterMapBaseRows(f.db, mapRowIndex, f.baseRowGroupSize, f.logMapWidth)
		if err != nil {
			return fmt.Errorf("failed to retrieve filter map %d base rows %d for modification: %v", groupIndex, rowIndex, err)
		}
	} else {
		baseRows = make([][]uint32, f.baseRowGroupSize)
	}
	for i, mapIndex := range mapIndices {
		if f.mapGroupIndex(mapIndex) != groupIndex {
			return fmt.Errorf("maps are not in the same base row group, index: %d, group: %d", mapIndex, groupIndex)
		}
		baseRow := []uint32(rows[i])
		var extRow FilterRow
		if uint32(len(rows[i])) > f.baseRowLength {
			extRow = baseRow[f.baseRowLength:]
			baseRow = baseRow[:f.baseRowLength]
		}
		baseRows[f.mapGroupOffset(mapIndex)] = baseRow
		rawdb.WriteFilterMapExtRow(batch, f.mapRowIndex(mapIndex, rowIndex), extRow, f.logMapWidth)
	}
	rawdb.WriteFilterMapBaseRows(batch, mapRowIndex, baseRows, f.logMapWidth)
	return nil
}

func (f *FilterMaps) mapRowIndex(mapIndex, rowIndex uint32) uint64 {
	epochIndex, mapSubIndex := mapIndex>>f.logMapsPerEpoch, mapIndex&(f.mapsPerEpoch-1)
	return (uint64(epochIndex)<<f.logMapHeight+uint64(rowIndex))<<f.logMapsPerEpoch + uint64(mapSubIndex)
}

func (f *FilterMaps) getBlockLvPointer(blockNumber uint64) (uint64, error) {
	if lvPointer, ok := f.lvPointerCache.Get(blockNumber); ok {
		return lvPointer, nil
	}
	lvPointer, err := rawdb.ReadBlockLvPointer(f.db, blockNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve log value pointer of block %d: %v", blockNumber, err)
	}
	f.lvPointerCache.Add(blockNumber, lvPointer)
	return lvPointer, nil
}

func (f *FilterMaps) storeBlockLvPointer(batch ethdb.Batch, blockNumber, lvPointer uint64) {
	f.lvPointerCache.Add(blockNumber, lvPointer)
	rawdb.WriteBlockLvPointer(batch, blockNumber, lvPointer)
}

func (f *FilterMaps) deleteBlockLvPointer(batch ethdb.Batch, blockNumber uint64) {
	f.lvPointerCache.Remove(blockNumber)
	rawdb.DeleteBlockLvPointer(batch, blockNumber)
}

func (f *FilterMaps) getLastBlockOfMap(mapIndex uint32) (uint64, common.Hash, error) {
	if lastBlock, ok := f.lastBlockCache.Get(mapIndex); ok {
		return lastBlock.number, lastBlock.id, nil
	}
	number, id, err := rawdb.ReadFilterMapLastBlock(f.db, mapIndex)
	if err != nil {
		return 0, common.Hash{}, fmt.Errorf("failed to retrieve last block of map %d: %v", mapIndex, err)
	}
	f.lastBlockCache.Add(mapIndex, lastBlockOfMap{number: number, id: id})
	return number, id, nil
}

func (f *FilterMaps) storeLastBlockOfMap(batch ethdb.Batch, mapIndex uint32, number uint64, id common.Hash) {
	f.lastBlockCache.Add(mapIndex, lastBlockOfMap{number: number, id: id})
	rawdb.WriteFilterMapLastBlock(batch, mapIndex, number, id)
}

func (f *FilterMaps) deleteLastBlockOfMap(batch ethdb.Batch, mapIndex uint32) {
	f.lastBlockCache.Remove(mapIndex)
	rawdb.DeleteFilterMapLastBlock(batch, mapIndex)
}

func (f *FilterMaps) deleteTailEpoch(epoch uint32) (bool, error) {
	f.indexLock.Lock()
	defer f.indexLock.Unlock()

	lastBlock, _, err := f.getLastBlockOfMap(f.lastEpochMap(epoch))
	if err != nil {
		return false, fmt.Errorf("failed to retrieve last block of deleted epoch %d: %v", epoch, err)
	}
	var firstBlock uint64
	firstMap := f.firstEpochMap(epoch)
	if epoch > 0 {
		firstBlock, _, err = f.getLastBlockOfMap(firstMap - 1)
		if err != nil {
			return false, fmt.Errorf("failed to retrieve last block before deleted epoch %d: %v", epoch, err)
		}
		firstBlock++
	}
	var (
		fmr            = f.indexedRange
		firstEpoch     = f.mapEpoch(f.indexedRange.maps.First())
		afterLastEpoch = f.mapEpoch(f.indexedRange.maps.AfterLast() + f.mapsPerEpoch - 1)
	)
	if f.indexedRange.tailPartialEpoch != 0 && firstEpoch > 0 {
		firstEpoch--
	}
	switch {
	case epoch < firstEpoch:
	case epoch == firstEpoch && epoch+1 < afterLastEpoch:
		fmr.tailPartialEpoch = 0
		fmr.maps.SetFirst(f.firstEpochMap(epoch + 1))
		fmr.blocks.SetFirst(lastBlock + 1)
		f.setRange(f.db, f.indexedView, fmr, false)
	default:
		return false, errors.New("invalid tail epoch number")
	}
	deleteFn := func(db ethdb.KeyValueStore, hashScheme bool, stopCb func(bool) bool) error {
		first := f.mapRowIndex(firstMap, 0)
		count := f.mapRowIndex(firstMap+f.mapsPerEpoch, 0) - first
		if err := rawdb.DeleteFilterMapRows(f.db, common.NewRange(first, count), hashScheme, stopCb); err != nil {
			return err
		}
		for mapIndex := firstMap; mapIndex < firstMap+f.mapsPerEpoch; mapIndex++ {
			f.filterMapCache.Remove(mapIndex)
		}
		delMapRange := common.NewRange(firstMap, f.mapsPerEpoch-1)
		if err := rawdb.DeleteFilterMapLastBlocks(f.db, delMapRange, hashScheme, stopCb); err != nil {
			return err
		}
		for mapIndex := firstMap; mapIndex < firstMap+f.mapsPerEpoch-1; mapIndex++ {
			f.lastBlockCache.Remove(mapIndex)
		}
		delBlockRange := common.NewRange(firstBlock, lastBlock-firstBlock)
		if err := rawdb.DeleteBlockLvPointers(f.db, delBlockRange, hashScheme, stopCb); err != nil {
			return err
		}
		for blockNumber := firstBlock; blockNumber < lastBlock; blockNumber++ {
			f.lvPointerCache.Remove(blockNumber)
		}
		return nil
	}
	action := fmt.Sprintf("Deleting tail epoch #%d", epoch)
	stopFn := func() bool {
		f.processEvents()
		return f.stop || !f.targetHeadIndexed()
	}
	if err := f.safeDeleteWithLogs(deleteFn, action, stopFn); err == nil {
		if f.cleanedEpochsBefore == epoch {
			f.cleanedEpochsBefore = epoch + 1
		}
		return true, nil
	} else {
		if f.cleanedEpochsBefore > epoch {
			f.cleanedEpochsBefore = epoch
		}
		if errors.Is(err, rawdb.ErrDeleteRangeInterrupted) {
			return false, nil
		}
		return false, err
	}
}

func (f *FilterMaps) exportCheckpoints() {
	finalLvPtr, err := f.getBlockLvPointer(f.finalBlock + 1)
	if err != nil {
		log.Error("Error fetching log value pointer of finalized block", "block", f.finalBlock, "error", err)
		return
	}
	epochCount := uint32(finalLvPtr >> (f.logValuesPerMap + f.logMapsPerEpoch))
	if epochCount == f.lastFinalEpoch {
		return
	}
	w, err := os.Create(f.exportFileName)
	if err != nil {
		log.Error("Error creating checkpoint export file", "name", f.exportFileName, "error", err)
		return
	}
	defer w.Close()

	log.Info("Exporting log index checkpoints", "epochs", epochCount, "file", f.exportFileName)
	w.WriteString("[\n")
	comma := ","
	for epoch := uint32(0); epoch < epochCount; epoch++ {
		lastBlock, lastBlockId, err := f.getLastBlockOfMap(f.lastEpochMap(epoch))
		if err != nil {
			log.Error("Error fetching last block of epoch", "epoch", epoch, "error", err)
			return
		}
		lvPtr, err := f.getBlockLvPointer(lastBlock)
		if err != nil {
			log.Error("Error fetching log value pointer of last block", "block", lastBlock, "error", err)
			return
		}
		if epoch == epochCount-1 {
			comma = ""
		}
		w.WriteString(fmt.Sprintf("{\"blockNumber\": %d, \"blockId\": \"%#x\", \"firstIndex\": %d}%s\n", lastBlock, lastBlockId, lvPtr, comma))
	}
	w.WriteString("]\n")
	f.lastFinalEpoch = epochCount
}
