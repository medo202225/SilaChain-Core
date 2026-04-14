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
"math"
"slices"
"sort"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/common/lru"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/log"
)

const (
maxMapsPerBatch   = 32
valuesPerCallback = 1024
cachedRowMappings = 10000
rowsPerBatch      = 1024
)

var (
errChainUpdate = errors.New("rendered section of chain updated")
)

type mapRenderer struct {
f            *FilterMaps
renderBefore uint32
currentMap   *renderedMap
finishedMaps map[uint32]*renderedMap
finished     common.Range[uint32]
iterator     *logIterator
}

type renderedMap struct {
filterMap     filterMap
mapIndex      uint32
lastBlock     uint64
lastBlockId   common.Hash
blockLvPtrs   []uint64
finished      bool
headDelimiter uint64
}

func (r *renderedMap) firstBlock() uint64 {
return r.lastBlock + 1 - uint64(len(r.blockLvPtrs))
}

func (f *FilterMaps) renderMapsBefore(renderBefore uint32) (*mapRenderer, error) {
nextMap, startBlock, startLvPtr, err := f.lastCanonicalMapBoundaryBefore(renderBefore)
if err != nil {
return nil, err
}
if snapshot := f.lastCanonicalSnapshotOfMap(nextMap); snapshot != nil {
return f.renderMapsFromSnapshot(snapshot)
}
if nextMap >= renderBefore {
return nil, nil
}
return f.renderMapsFromMapBoundary(nextMap, renderBefore, startBlock, startLvPtr)
}

func (f *FilterMaps) renderMapsFromSnapshot(cp *renderedMap) (*mapRenderer, error) {
f.testSnapshotUsed = true
iter, err := f.newLogIteratorFromBlockDelimiter(cp.lastBlock, cp.headDelimiter)
if err != nil {
return nil, fmt.Errorf("failed to create log iterator from block delimiter %d: %v", cp.lastBlock, err)
}
return &mapRenderer{
f: f,
currentMap: &renderedMap{
filterMap:   cp.filterMap.fullCopy(),
mapIndex:    cp.mapIndex,
lastBlock:   cp.lastBlock,
blockLvPtrs: slices.Clone(cp.blockLvPtrs),
},
finishedMaps: make(map[uint32]*renderedMap),
finished:     common.NewRange(cp.mapIndex, 0),
renderBefore: math.MaxUint32,
iterator:     iter,
}, nil
}

func (f *FilterMaps) renderMapsFromMapBoundary(firstMap, renderBefore uint32, startBlock, startLvPtr uint64) (*mapRenderer, error) {
iter, err := f.newLogIteratorFromMapBoundary(firstMap, startBlock, startLvPtr)
if err != nil {
return nil, fmt.Errorf("failed to create log iterator from map boundary %d: %v", firstMap, err)
}
return &mapRenderer{
f: f,
currentMap: &renderedMap{
filterMap: f.emptyFilterMap(),
mapIndex:  firstMap,
lastBlock: iter.blockNumber,
},
finishedMaps: make(map[uint32]*renderedMap),
finished:     common.NewRange(firstMap, 0),
renderBefore: renderBefore,
iterator:     iter,
}, nil
}

func (f *FilterMaps) lastCanonicalSnapshotOfMap(mapIndex uint32) *renderedMap {
var best *renderedMap
for _, blockNumber := range f.renderSnapshots.Keys() {
if cp, _ := f.renderSnapshots.Get(blockNumber); cp != nil && blockNumber < f.indexedRange.blocks.AfterLast() &&
blockNumber <= f.indexedView.HeadNumber() && f.indexedView.BlockId(blockNumber) == cp.lastBlockId &&
blockNumber <= f.targetView.HeadNumber() && f.targetView.BlockId(blockNumber) == cp.lastBlockId &&
cp.mapIndex == mapIndex && (best == nil || blockNumber > best.lastBlock) {
best = cp
}
}
return best
}

func (f *FilterMaps) lastCanonicalMapBoundaryBefore(renderBefore uint32) (nextMap uint32, startBlock, startLvPtr uint64, err error) {
if !f.indexedRange.initialized {
return 0, 0, 0, nil
}
mapIndex := renderBefore
for {
var ok bool
if mapIndex, ok = f.lastMapBoundaryBefore(mapIndex); !ok {
return 0, 0, 0, nil
}
lastBlock, lastBlockId, err := f.getLastBlockOfMap(mapIndex)
if err != nil {
return 0, 0, 0, fmt.Errorf("failed to retrieve last block of reverse iterated map %d: %v", mapIndex, err)
}
if (f.indexedRange.headIndexed && mapIndex >= f.indexedRange.maps.Last()) ||
lastBlock >= f.targetView.HeadNumber() || lastBlockId != f.targetView.BlockId(lastBlock) {
continue
}
lvPtr, err := f.getBlockLvPointer(lastBlock)
if err != nil {
return 0, 0, 0, fmt.Errorf("failed to retrieve log value pointer of last canonical boundary block %d: %v", lastBlock, err)
}
return mapIndex + 1, lastBlock, lvPtr, nil
}
}

func (f *FilterMaps) lastMapBoundaryBefore(renderBefore uint32) (uint32, bool) {
if !f.indexedRange.initialized || f.indexedRange.maps.AfterLast() == 0 || renderBefore == 0 {
return 0, false
}
afterLastFullMap := f.indexedRange.maps.AfterLast()
if afterLastFullMap > 0 && f.indexedRange.headIndexed {
afterLastFullMap--
}
firstRendered := min(renderBefore-1, afterLastFullMap)
if firstRendered == 0 {
return 0, false
}
if firstRendered >= f.indexedRange.maps.First() {
return firstRendered - 1, true
}
if firstRendered+f.mapsPerEpoch > f.indexedRange.maps.First() {
firstRendered = min(firstRendered, f.indexedRange.maps.First()-f.mapsPerEpoch+f.indexedRange.tailPartialEpoch)
} else {
firstRendered = (firstRendered >> f.logMapsPerEpoch) << f.logMapsPerEpoch
}
if firstRendered == 0 {
return 0, false
}
return firstRendered - 1, true
}

func (f *FilterMaps) emptyFilterMap() filterMap {
return make(filterMap, f.mapHeight)
}

func (f *FilterMaps) loadHeadSnapshot() error {
fm, err := f.getFilterMap(f.indexedRange.maps.Last())
if err != nil {
return fmt.Errorf("failed to load head snapshot map %d: %v", f.indexedRange.maps.Last(), err)
}
lastBlock, _, err := f.getLastBlockOfMap(f.indexedRange.maps.Last())
if err != nil {
return fmt.Errorf("failed to retrieve last block of head snapshot map %d: %v", f.indexedRange.maps.Last(), err)
}
var firstBlock uint64
if f.indexedRange.maps.AfterLast() > 1 {
prevLastBlock, _, err := f.getLastBlockOfMap(f.indexedRange.maps.Last() - 1)
if err != nil {
return fmt.Errorf("failed to retrieve last block of map %d before head snapshot: %v", f.indexedRange.maps.Last()-1, err)
}
firstBlock = prevLastBlock + 1
}
lvPtrs := make([]uint64, lastBlock+1-firstBlock)
for i := range lvPtrs {
lvPtrs[i], err = f.getBlockLvPointer(firstBlock + uint64(i))
if err != nil {
return fmt.Errorf("failed to retrieve log value pointer of head snapshot block %d: %v", firstBlock+uint64(i), err)
}
}
f.renderSnapshots.Add(f.indexedRange.blocks.Last(), &renderedMap{
filterMap:     fm.fullCopy(),
mapIndex:      f.indexedRange.maps.Last(),
lastBlock:     f.indexedRange.blocks.Last(),
lastBlockId:   f.indexedView.BlockId(f.indexedRange.blocks.Last()),
blockLvPtrs:   lvPtrs,
finished:      true,
headDelimiter: f.indexedRange.headDelimiter,
})
return nil
}

func (r *mapRenderer) makeSnapshot() {
if r.iterator.blockNumber != r.currentMap.lastBlock || r.iterator.chainView != r.f.targetView {
panic("iterator state inconsistent with current rendered map")
}
r.f.renderSnapshots.Add(r.currentMap.lastBlock, &renderedMap{
filterMap:     r.currentMap.filterMap.fastCopy(),
mapIndex:      r.currentMap.mapIndex,
lastBlock:     r.currentMap.lastBlock,
lastBlockId:   r.iterator.chainView.BlockId(r.currentMap.lastBlock),
blockLvPtrs:   r.currentMap.blockLvPtrs,
finished:      true,
headDelimiter: r.iterator.lvIndex,
})
}

func (r *mapRenderer) run(stopCb func() bool, writeCb func()) (bool, error) {
for {
if done, err := r.renderCurrentMap(stopCb); !done {
return done, err
}
r.finishedMaps[r.currentMap.mapIndex] = r.currentMap
r.finished.SetLast(r.finished.AfterLast())
if len(r.finishedMaps) >= maxMapsPerBatch || r.f.mapGroupOffset(r.finished.AfterLast()) == 0 {
if err := r.writeFinishedMaps(stopCb); err != nil {
return false, err
}
writeCb()
}
if r.finished.AfterLast() == r.renderBefore || r.iterator.finished {
if err := r.writeFinishedMaps(stopCb); err != nil {
return false, err
}
writeCb()
return true, nil
}
r.currentMap = &renderedMap{
filterMap: r.f.emptyFilterMap(),
mapIndex:  r.finished.AfterLast(),
}
}
}

func (r *mapRenderer) renderCurrentMap(stopCb func() bool) (bool, error) {
var (
totalTime                           time.Duration
logValuesProcessed, blocksProcessed int64
)
start := time.Now()
if !r.iterator.updateChainView(r.f.targetView) {
return false, errChainUpdate
}
var waitCnt int

if r.iterator.lvIndex == 0 {
r.currentMap.blockLvPtrs = []uint64{0}
}
type lvPos struct{ rowIndex, layerIndex uint32 }
rowMappingCache := lru.NewCache[common.Hash, lvPos](cachedRowMappings)
defer rowMappingCache.Purge()

for r.iterator.lvIndex < uint64(r.currentMap.mapIndex+1)<<r.f.logValuesPerMap && !r.iterator.finished {
waitCnt++
if waitCnt >= valuesPerCallback {
totalTime += time.Since(start)
if stopCb() {
return false, nil
}
start = time.Now()
if !r.iterator.updateChainView(r.f.targetView) {
return false, errChainUpdate
}
waitCnt = 0
}
if logValue := r.iterator.getValueHash(); logValue != (common.Hash{}) {
lvp, cached := rowMappingCache.Get(logValue)
if !cached {
lvp = lvPos{rowIndex: r.f.rowIndex(r.currentMap.mapIndex, 0, logValue)}
}
for uint32(len(r.currentMap.filterMap[lvp.rowIndex])) >= r.f.maxRowLength(lvp.layerIndex) {
lvp.layerIndex++
lvp.rowIndex = r.f.rowIndex(r.currentMap.mapIndex, lvp.layerIndex, logValue)
cached = false
}
r.currentMap.filterMap[lvp.rowIndex] = append(r.currentMap.filterMap[lvp.rowIndex], r.f.columnIndex(r.iterator.lvIndex, &logValue))
if !cached {
rowMappingCache.Add(logValue, lvp)
}
}
if err := r.iterator.next(); err != nil {
return false, fmt.Errorf("failed to advance log iterator at %d while rendering map %d: %v", r.iterator.lvIndex, r.currentMap.mapIndex, err)
}
if !r.iterator.skipToBoundary {
logValuesProcessed++
r.currentMap.lastBlock = r.iterator.blockNumber
if r.iterator.blockStart {
blocksProcessed++
r.currentMap.blockLvPtrs = append(r.currentMap.blockLvPtrs, r.iterator.lvIndex)
}
if !r.f.testDisableSnapshots && r.renderBefore >= r.f.indexedRange.maps.AfterLast() &&
(r.iterator.delimiter || r.iterator.finished) {
r.makeSnapshot()
}
}
}
if r.iterator.finished {
r.currentMap.finished = true
r.currentMap.headDelimiter = r.iterator.lvIndex
}
r.currentMap.lastBlockId = r.f.targetView.BlockId(r.currentMap.lastBlock)
totalTime += time.Since(start)
mapRenderTimer.Update(totalTime)
mapLogValueMeter.Mark(logValuesProcessed)
mapBlockMeter.Mark(blocksProcessed)
return true, nil
}

func (r *mapRenderer) writeFinishedMaps(pauseCb func() bool) error {
var totalTime time.Duration
start := time.Now()
if len(r.finishedMaps) == 0 {
return nil
}
r.f.indexLock.Lock()
defer r.f.indexLock.Unlock()

oldRange := r.f.indexedRange
tempRange, err := r.getTempRange()
if err != nil {
return fmt.Errorf("failed to get temporary rendered range: %v", err)
}
newRange, err := r.getUpdatedRange()
if err != nil {
return fmt.Errorf("failed to get updated rendered range: %v", err)
}
renderedView := r.f.targetView

batch := r.f.db.NewBatch()
var writeCnt int
checkWriteCnt := func() {
writeCnt++
if writeCnt == rowsPerBatch {
writeCnt = 0
if err := batch.Write(); err != nil {
log.Crit("Error writing log index update batch", "error", err)
}
r.f.indexLock.Unlock()
totalTime += time.Since(start)
pauseCb()
start = time.Now()
r.f.indexLock.Lock()
batch = r.f.db.NewBatch()
}
}

if tempRange != r.f.indexedRange {
r.f.setRange(batch, r.f.indexedView, tempRange, true)
}
for rowIndex := uint32(0); rowIndex < r.f.mapHeight; rowIndex++ {
var (
mapIndices []uint32
rows       []FilterRow
)
for mapIndex := range r.finished.Iter() {
row := r.finishedMaps[mapIndex].filterMap[rowIndex]
if fm, _ := r.f.filterMapCache.Get(mapIndex); fm != nil && row.Equal(fm[rowIndex]) {
continue
}
mapIndices = append(mapIndices, mapIndex)
rows = append(rows, row)
}
if newRange.maps.AfterLast() == r.finished.AfterLast() {
for mapIndex := r.finished.AfterLast(); mapIndex < oldRange.maps.AfterLast(); mapIndex++ {
if fm, _ := r.f.filterMapCache.Get(mapIndex); fm != nil && len(fm[rowIndex]) == 0 {
continue
}
mapIndices = append(mapIndices, mapIndex)
rows = append(rows, nil)
}
}
if err := r.f.storeFilterMapRows(batch, mapIndices, rowIndex, rows); err != nil {
return fmt.Errorf("failed to store filter maps %v row %d: %v", mapIndices, rowIndex, err)
}
checkWriteCnt()
}
if newRange.maps.AfterLast() == r.finished.AfterLast() {
for mapIndex := range r.finished.Iter() {
r.f.filterMapCache.Add(mapIndex, r.finishedMaps[mapIndex].filterMap)
}
for mapIndex := r.finished.AfterLast(); mapIndex < oldRange.maps.AfterLast(); mapIndex++ {
r.f.filterMapCache.Remove(mapIndex)
}
} else {
for mapIndex := range r.finished.Iter() {
r.f.filterMapCache.Remove(mapIndex)
}
}
var blockNumber uint64
if r.finished.First() > 0 {
lastBlock, _, err := r.f.getLastBlockOfMap(r.finished.First() - 1)
if err != nil {
return fmt.Errorf("failed to get last block of previous map %d: %v", r.finished.First()-1, err)
}
blockNumber = lastBlock + 1
}
for mapIndex := range r.finished.Iter() {
renderedMap := r.finishedMaps[mapIndex]
if blockNumber != renderedMap.firstBlock() {
return fmt.Errorf("non-continuous block numbers in rendered map %d (next expected: %d  first rendered: %d)", mapIndex, blockNumber, renderedMap.firstBlock())
}
r.f.storeLastBlockOfMap(batch, mapIndex, renderedMap.lastBlock, renderedMap.lastBlockId)
checkWriteCnt()
for _, lvPtr := range renderedMap.blockLvPtrs {
r.f.storeBlockLvPointer(batch, blockNumber, lvPtr)
checkWriteCnt()
blockNumber++
}
}
if newRange.maps.AfterLast() == r.finished.AfterLast() {
for mapIndex := r.finished.AfterLast(); mapIndex < oldRange.maps.AfterLast(); mapIndex++ {
r.f.deleteLastBlockOfMap(batch, mapIndex)
checkWriteCnt()
}
for ; blockNumber < oldRange.blocks.AfterLast(); blockNumber++ {
r.f.deleteBlockLvPointer(batch, blockNumber)
checkWriteCnt()
}
}
r.finishedMaps = make(map[uint32]*renderedMap)
r.finished.SetFirst(r.finished.AfterLast())
r.f.setRange(batch, renderedView, newRange, false)
if err := batch.Write(); err != nil {
log.Crit("Error writing log index update batch", "error", err)
}
totalTime += time.Since(start)
mapWriteTimer.Update(totalTime)
return nil
}

func (r *mapRenderer) getTempRange() (filterMapsRange, error) {
tempRange := r.f.indexedRange
if err := tempRange.addRenderedRange(r.finished.First(), r.finished.First(), r.renderBefore, r.f.mapsPerEpoch); err != nil {
return filterMapsRange{}, fmt.Errorf("failed to update temporary rendered range: %v", err)
}
if tempRange.maps.First() != r.f.indexedRange.maps.First() {
if tempRange.maps.First() > 0 {
firstBlock, _, err := r.f.getLastBlockOfMap(tempRange.maps.First() - 1)
if err != nil {
return filterMapsRange{}, fmt.Errorf("failed to retrieve last block of map %d before temporary range: %v", tempRange.maps.First()-1, err)
}
tempRange.blocks.SetFirst(firstBlock + 1)
} else {
tempRange.blocks.SetFirst(0)
}
}
if tempRange.maps.AfterLast() != r.f.indexedRange.maps.AfterLast() {
if !tempRange.maps.IsEmpty() {
lastBlock, _, err := r.f.getLastBlockOfMap(tempRange.maps.Last())
if err != nil {
return filterMapsRange{}, fmt.Errorf("failed to retrieve last block of map %d at the end of temporary range: %v", tempRange.maps.Last(), err)
}
tempRange.blocks.SetAfterLast(lastBlock)
} else {
tempRange.blocks.SetAfterLast(0)
}
tempRange.headIndexed = false
tempRange.headDelimiter = 0
}
return tempRange, nil
}

func (r *mapRenderer) getUpdatedRange() (filterMapsRange, error) {
newRange := r.f.indexedRange
if err := newRange.addRenderedRange(r.finished.First(), r.finished.AfterLast(), r.renderBefore, r.f.mapsPerEpoch); err != nil {
return filterMapsRange{}, fmt.Errorf("failed to update rendered range: %v", err)
}
if newRange.maps.First() != r.f.indexedRange.maps.First() {
if newRange.maps.First() > 0 {
firstBlock, _, err := r.f.getLastBlockOfMap(newRange.maps.First() - 1)
if err != nil {
return filterMapsRange{}, fmt.Errorf("failed to retrieve last block of map %d before rendered range: %v", newRange.maps.First()-1, err)
}
newRange.blocks.SetFirst(firstBlock + 1)
} else {
newRange.blocks.SetFirst(0)
}
}
if newRange.maps.AfterLast() == r.finished.AfterLast() {
lm := r.finishedMaps[r.finished.Last()]
newRange.headIndexed = lm.finished
if lm.finished {
newRange.blocks.SetLast(r.f.targetView.HeadNumber())
if lm.lastBlock != r.f.targetView.HeadNumber() {
panic("map rendering finished but last block != head block")
}
newRange.headDelimiter = lm.headDelimiter
} else {
newRange.blocks.SetAfterLast(lm.lastBlock)
newRange.headDelimiter = 0
}
} else {
if lastBlock := r.finishedMaps[r.finished.Last()].lastBlock; !matchViews(r.f.indexedView, r.f.targetView, lastBlock) {
return filterMapsRange{}, errChainUpdate
}
}
return newRange, nil
}

func (fmr *filterMapsRange) addRenderedRange(firstRendered, afterLastRendered, afterLastRemoved, mapsPerEpoch uint32) error {
if !fmr.initialized {
return errors.New("log index not initialized")
}

type endpoint struct {
m uint32
d int
}
endpoints := []endpoint{{fmr.maps.First(), 1}, {fmr.maps.AfterLast(), -1}, {firstRendered, 1}, {afterLastRendered, -101}, {afterLastRemoved, 100}}
if fmr.tailPartialEpoch > 0 {
endpoints = append(endpoints, []endpoint{{fmr.maps.First() - mapsPerEpoch, 1}, {fmr.maps.First() - mapsPerEpoch + fmr.tailPartialEpoch, -1}}...)
}
sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].m < endpoints[j].m })
var (
sum    int
merged []uint32
last   bool
)
for i, e := range endpoints {
sum += e.d
if i < len(endpoints)-1 && endpoints[i+1].m == e.m {
continue
}
if (sum > 0) != last {
merged = append(merged, e.m)
last = !last
}
}

switch len(merged) {
case 0:
fmr.tailPartialEpoch = 0
fmr.maps = common.NewRange(firstRendered, 0)

case 2:
fmr.tailPartialEpoch = 0
fmr.maps = common.NewRange(merged[0], merged[1]-merged[0])

case 4:
if merged[2] != merged[0]+mapsPerEpoch {
return fmt.Errorf("invalid tail partial epoch: %v", merged)
}
fmr.tailPartialEpoch = merged[1] - merged[0]
fmr.maps = common.NewRange(merged[2], merged[3]-merged[2])

default:
return fmt.Errorf("invalid number of rendered sections: %v", merged)
}
return nil
}

type logIterator struct {
params                                          *Params
chainView                                       *ChainView
blockNumber                                     uint64
receipts                                        types.Receipts
blockStart, delimiter, skipToBoundary, finished bool
txIndex, logIndex, topicIndex                   int
lvIndex                                         uint64
}

var errUnindexedRange = errors.New("unindexed range")

func (f *FilterMaps) newLogIteratorFromBlockDelimiter(blockNumber, lvIndex uint64) (*logIterator, error) {
if blockNumber > f.targetView.HeadNumber() {
return nil, fmt.Errorf("iterator entry point %d after target chain head block %d", blockNumber, f.targetView.HeadNumber())
}
if !f.indexedRange.blocks.Includes(blockNumber) {
return nil, errUnindexedRange
}
finished := blockNumber == f.targetView.HeadNumber()
l := &logIterator{
chainView:   f.targetView,
params:      &f.Params,
blockNumber: blockNumber,
finished:    finished,
delimiter:   !finished,
lvIndex:     lvIndex,
}
l.enforceValidState()
return l, nil
}

func (f *FilterMaps) newLogIteratorFromMapBoundary(mapIndex uint32, startBlock, startLvPtr uint64) (*logIterator, error) {
if startBlock > f.targetView.HeadNumber() {
return nil, fmt.Errorf("iterator entry point %d after target chain head block %d", startBlock, f.targetView.HeadNumber())
}
receipts := f.targetView.RawReceipts(startBlock)
if receipts == nil {
return nil, fmt.Errorf("receipts not found for start block %d", startBlock)
}
l := &logIterator{
chainView:   f.targetView,
params:      &f.Params,
blockNumber: startBlock,
receipts:    receipts,
blockStart:  true,
lvIndex:     startLvPtr,
}
l.enforceValidState()
targetIndex := uint64(mapIndex) << f.logValuesPerMap
if l.lvIndex > targetIndex {
return nil, fmt.Errorf("log value pointer %d of last block of map is after map boundary %d", l.lvIndex, targetIndex)
}
for l.lvIndex < targetIndex {
if l.finished {
return nil, fmt.Errorf("iterator already finished at %d before map boundary target %d", l.lvIndex, targetIndex)
}
if err := l.next(); err != nil {
return nil, fmt.Errorf("failed to advance log iterator at %d before map boundary target %d: %v", l.lvIndex, targetIndex, err)
}
}
return l, nil
}

func (l *logIterator) updateChainView(cv *ChainView) bool {
if !matchViews(cv, l.chainView, l.blockNumber) {
return false
}
l.chainView = cv
return true
}

func (l *logIterator) getValueHash() common.Hash {
if l.delimiter || l.finished || l.skipToBoundary {
return common.Hash{}
}
log := l.receipts[l.txIndex].Logs[l.logIndex]
if l.topicIndex == 0 {
return addressValue(log.Address)
}
return topicValue(log.Topics[l.topicIndex-1])
}

func (l *logIterator) next() error {
if l.skipToBoundary {
l.lvIndex++
if l.lvIndex%l.params.valuesPerMap == 0 {
l.skipToBoundary = false
}
return nil
}
if l.finished {
return nil
}
if l.delimiter {
l.delimiter = false
l.blockNumber++
l.receipts = l.chainView.RawReceipts(l.blockNumber)
if l.receipts == nil {
return fmt.Errorf("receipts not found for block %d", l.blockNumber)
}
l.txIndex, l.logIndex, l.topicIndex, l.blockStart = 0, 0, 0, true
} else {
l.topicIndex++
l.blockStart = false
}
l.lvIndex++
l.enforceValidState()
return nil
}

func (l *logIterator) enforceValidState() {
if l.delimiter || l.finished || l.skipToBoundary {
return
}
for ; l.txIndex < len(l.receipts); l.txIndex++ {
receipt := l.receipts[l.txIndex]
for ; l.logIndex < len(receipt.Logs); l.logIndex++ {
log := receipt.Logs[l.logIndex]
if l.topicIndex == 0 && uint64(len(log.Topics)+1) > l.params.valuesPerMap-l.lvIndex%l.params.valuesPerMap {
l.skipToBoundary = true
return
}
if l.topicIndex <= len(log.Topics) {
return
}
l.topicIndex = 0
}
l.logIndex = 0
}
if l.blockNumber == l.chainView.HeadNumber() {
l.finished = true
} else {
l.delimiter = true
}
}
