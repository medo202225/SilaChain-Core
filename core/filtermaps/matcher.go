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
"context"
"errors"
"fmt"
"sync"
"sync/atomic"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/common/mclock"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/log"
)

const doRuntimeStats = false

var ErrMatchAll = errors.New("match all patterns not supported")

type MatcherBackend interface {
GetParams() *Params
GetBlockLvPointer(ctx context.Context, blockNumber uint64) (uint64, error)
GetFilterMapRows(ctx context.Context, mapIndices []uint32, rowIndex uint32, baseLayerOnly bool) ([]FilterRow, error)
GetLogByLvIndex(ctx context.Context, lvIndex uint64) (*types.Log, error)
SyncLogIndex(ctx context.Context) (SyncRange, error)
Close()
}

type SyncRange struct {
IndexedView   *ChainView
ValidBlocks   common.Range[uint64]
IndexedBlocks common.Range[uint64]
}

func GetPotentialMatches(ctx context.Context, backend MatcherBackend, firstBlock, lastBlock uint64, addresses []common.Address, topics [][]common.Hash) ([]*types.Log, error) {
params := backend.GetParams()
firstIndex, err := backend.GetBlockLvPointer(ctx, firstBlock)
if err != nil {
return nil, fmt.Errorf("failed to retrieve log value pointer for first block %d: %v", firstBlock, err)
}
lastIndex, err := backend.GetBlockLvPointer(ctx, lastBlock+1)
if err != nil {
return nil, fmt.Errorf("failed to retrieve log value pointer after last block %d: %v", lastBlock, err)
}
if lastIndex > 0 {
lastIndex--
}

matchers := make([]matcher, len(topics)+1)
matchAddress := make(matchAny, len(addresses))
for i, address := range addresses {
matchAddress[i] = &singleMatcher{backend: backend, value: addressValue(address)}
}
matchers[0] = matchAddress
for i, topicList := range topics {
matchTopic := make(matchAny, len(topicList))
for j, topic := range topicList {
matchTopic[j] = &singleMatcher{backend: backend, value: topicValue(topic)}
}
matchers[i+1] = matchTopic
}
matcher := newMatchSequence(params, matchers)

m := &matcherEnv{
ctx:        ctx,
backend:    backend,
params:     params,
matcher:    matcher,
firstIndex: firstIndex,
lastIndex:  lastIndex,
firstMap:   uint32(firstIndex >> params.logValuesPerMap),
lastMap:    uint32(lastIndex >> params.logValuesPerMap),
}

start := time.Now()
res, err := m.process()
matchRequestTimer.Update(time.Since(start))

if doRuntimeStats {
log.Info("Log search finished", "elapsed", time.Since(start))
for i, ma := range matchers {
for j, m := range ma.(matchAny) {
log.Info("Single matcher stats", "matchSequence", i, "matchAny", j)
m.(*singleMatcher).stats.print()
}
}
log.Info("Get log stats")
m.getLogStats.print()
}
return res, err
}

type matcherEnv struct {
getLogStats           runtimeStats
ctx                   context.Context
backend               MatcherBackend
params                *Params
matcher               matcher
firstIndex, lastIndex uint64
firstMap, lastMap     uint32
}

func (m *matcherEnv) process() ([]*types.Log, error) {
type task struct {
epochIndex uint32
logs       []*types.Log
err        error
done       chan struct{}
}

taskCh := make(chan *task)
var wg sync.WaitGroup
defer func() {
close(taskCh)
wg.Wait()
}()

worker := func() {
for task := range taskCh {
if task == nil {
break
}
task.logs, task.err = m.processEpoch(task.epochIndex)
close(task.done)
}
wg.Done()
}

for range 4 {
wg.Add(1)
go worker()
}

firstEpoch, lastEpoch := m.firstMap>>m.params.logMapsPerEpoch, m.lastMap>>m.params.logMapsPerEpoch
var logs []*types.Log
startEpoch, waitEpoch := firstEpoch, firstEpoch
tasks := make(map[uint32]*task)
tasks[startEpoch] = &task{epochIndex: startEpoch, done: make(chan struct{})}
for waitEpoch <= lastEpoch {
select {
case taskCh <- tasks[startEpoch]:
startEpoch++
if startEpoch <= lastEpoch {
if tasks[startEpoch] == nil {
tasks[startEpoch] = &task{epochIndex: startEpoch, done: make(chan struct{})}
}
}
case <-tasks[waitEpoch].done:
logs = append(logs, tasks[waitEpoch].logs...)
if err := tasks[waitEpoch].err; err != nil {
if err == ErrMatchAll {
matchAllMeter.Mark(1)
return logs, err
}
return logs, fmt.Errorf("failed to process log index epoch %d: %v", waitEpoch, err)
}
delete(tasks, waitEpoch)
waitEpoch++
if waitEpoch <= lastEpoch {
if tasks[waitEpoch] == nil {
tasks[waitEpoch] = &task{epochIndex: waitEpoch, done: make(chan struct{})}
}
}
}
}
return logs, nil
}

func (m *matcherEnv) processEpoch(epochIndex uint32) ([]*types.Log, error) {
start := time.Now()
var logs []*types.Log
fm, lm := epochIndex<<m.params.logMapsPerEpoch, (epochIndex+1)<<m.params.logMapsPerEpoch-1
if fm < m.firstMap {
fm = m.firstMap
}
if lm > m.lastMap {
lm = m.lastMap
}
mapIndices := make([]uint32, lm+1-fm)
for i := range mapIndices {
mapIndices[i] = fm + uint32(i)
}
matches, err := m.getAllMatches(mapIndices)
if err != nil {
return logs, err
}
var st int
m.getLogStats.setState(&st, stGetLog)
defer m.getLogStats.setState(&st, stNone)
for _, match := range matches {
if match == nil {
return nil, ErrMatchAll
}
mlogs, err := m.getLogsFromMatches(match)
if err != nil {
return logs, err
}
logs = append(logs, mlogs...)
}
m.getLogStats.addAmount(st, int64(len(logs)))
matchEpochTimer.Update(time.Since(start))
return logs, nil
}

func (m *matcherEnv) getLogsFromMatches(matches potentialMatches) ([]*types.Log, error) {
var logs []*types.Log
for _, match := range matches {
if match < m.firstIndex || match > m.lastIndex {
continue
}
log, err := m.backend.GetLogByLvIndex(m.ctx, match)
if err != nil {
return logs, fmt.Errorf("failed to retrieve log at index %d: %v", match, err)
}
if log != nil {
logs = append(logs, log)
}
matchLogLookup.Mark(1)
}
return logs, nil
}

func (m *matcherEnv) getAllMatches(mapIndices []uint32) ([]potentialMatches, error) {
instance := m.matcher.newInstance(mapIndices)
resultsMap := make(map[uint32]potentialMatches)
for layerIndex := uint32(0); len(resultsMap) < len(mapIndices); layerIndex++ {
results, err := instance.getMatchesForLayer(m.ctx, layerIndex)
if err != nil {
return nil, err
}
for _, result := range results {
resultsMap[result.mapIndex] = result.matches
}
}
matches := make([]potentialMatches, len(mapIndices))
for i, mapIndex := range mapIndices {
matches[i] = resultsMap[mapIndex]
}
return matches, nil
}

type matcher interface {
newInstance(mapIndices []uint32) matcherInstance
}

type matcherInstance interface {
getMatchesForLayer(ctx context.Context, layerIndex uint32) ([]matcherResult, error)
dropIndices(mapIndices []uint32)
}

type matcherResult struct {
mapIndex uint32
matches  potentialMatches
}

type singleMatcher struct {
backend MatcherBackend
value   common.Hash
stats   runtimeStats
}

type singleMatcherInstance struct {
*singleMatcher
mapIndices []uint32
filterRows map[uint32][]FilterRow
}

func (m *singleMatcher) newInstance(mapIndices []uint32) matcherInstance {
filterRows := make(map[uint32][]FilterRow)
for _, idx := range mapIndices {
filterRows[idx] = []FilterRow{}
}
copiedIndices := make([]uint32, len(mapIndices))
copy(copiedIndices, mapIndices)
return &singleMatcherInstance{
singleMatcher: m,
mapIndices:    copiedIndices,
filterRows:    filterRows,
}
}

func (m *singleMatcherInstance) getMatchesForLayer(ctx context.Context, layerIndex uint32) (results []matcherResult, err error) {
var st int
m.stats.setState(&st, stOther)
params := m.backend.GetParams()
var ptr int
for len(m.mapIndices) > ptr {
maskedMapIndex := params.maskedMapIndex(m.mapIndices[ptr], layerIndex)
rowIndex := params.rowIndex(m.mapIndices[ptr], layerIndex, m.value)
groupLength := 1
for ptr+groupLength < len(m.mapIndices) && params.maskedMapIndex(m.mapIndices[ptr+groupLength], layerIndex) == maskedMapIndex {
groupLength++
}
if layerIndex == 0 {
m.stats.setState(&st, stFetchFirst)
} else {
m.stats.setState(&st, stFetchMore)
}
groupRows, err := m.backend.GetFilterMapRows(ctx, m.mapIndices[ptr:ptr+groupLength], rowIndex, layerIndex == 0)
if err != nil {
m.stats.setState(&st, stNone)
return nil, fmt.Errorf("failed to retrieve filter map %d row %d: %v", m.mapIndices[ptr], rowIndex, err)
}
m.stats.setState(&st, stOther)
for i := range groupLength {
mapIndex := m.mapIndices[ptr+i]
filterRow := groupRows[i]
filterRows, ok := m.filterRows[mapIndex]
if !ok {
panic("dropped map in mapIndices")
}
if layerIndex == 0 {
matchBaseRowAccessMeter.Mark(1)
matchBaseRowSizeMeter.Mark(int64(len(filterRow)))
} else {
matchExtRowAccessMeter.Mark(1)
matchExtRowSizeMeter.Mark(int64(len(filterRow)))
}
m.stats.addAmount(st, int64(len(filterRow)))
filterRows = append(filterRows, filterRow)
if uint32(len(filterRow)) < params.maxRowLength(layerIndex) {
m.stats.setState(&st, stProcess)
matches := params.potentialMatches(filterRows, mapIndex, m.value)
m.stats.addAmount(st, int64(len(matches)))
results = append(results, matcherResult{
mapIndex: mapIndex,
matches:  matches,
})
m.stats.setState(&st, stOther)
delete(m.filterRows, mapIndex)
} else {
m.filterRows[mapIndex] = filterRows
}
}
ptr += groupLength
}
m.cleanMapIndices()
m.stats.setState(&st, stNone)
return results, nil
}

func (m *singleMatcherInstance) dropIndices(dropIndices []uint32) {
for _, mapIndex := range dropIndices {
delete(m.filterRows, mapIndex)
}
m.cleanMapIndices()
}

func (m *singleMatcherInstance) cleanMapIndices() {
var j int
for i, mapIndex := range m.mapIndices {
if _, ok := m.filterRows[mapIndex]; ok {
if i != j {
m.mapIndices[j] = mapIndex
}
j++
}
}
m.mapIndices = m.mapIndices[:j]
}

type matchAny []matcher

type matchAnyInstance struct {
matchAny
childInstances []matcherInstance
childResults   map[uint32]matchAnyResults
}

type matchAnyResults struct {
matches  []potentialMatches
done     []bool
needMore int
}

func (m matchAny) newInstance(mapIndices []uint32) matcherInstance {
if len(m) == 1 {
return m[0].newInstance(mapIndices)
}
childResults := make(map[uint32]matchAnyResults)
for _, idx := range mapIndices {
childResults[idx] = matchAnyResults{
matches:  make([]potentialMatches, len(m)),
done:     make([]bool, len(m)),
needMore: len(m),
}
}
childInstances := make([]matcherInstance, len(m))
for i, matcher := range m {
childInstances[i] = matcher.newInstance(mapIndices)
}
return &matchAnyInstance{
matchAny:       m,
childInstances: childInstances,
childResults:   childResults,
}
}

func (m *matchAnyInstance) getMatchesForLayer(ctx context.Context, layerIndex uint32) (mergedResults []matcherResult, err error) {
if len(m.matchAny) == 0 {
mergedResults = make([]matcherResult, len(m.childResults))
var i int
for mapIndex := range m.childResults {
mergedResults[i] = matcherResult{mapIndex: mapIndex, matches: nil}
i++
}
return mergedResults, nil
}
for i, childInstance := range m.childInstances {
results, err := childInstance.getMatchesForLayer(ctx, layerIndex)
if err != nil {
return nil, fmt.Errorf("failed to evaluate child matcher on layer %d: %v", layerIndex, err)
}
for _, result := range results {
mr, ok := m.childResults[result.mapIndex]
if !ok || mr.done[i] {
continue
}
mr.done[i] = true
mr.matches[i] = result.matches
mr.needMore--
if mr.needMore == 0 || result.matches == nil {
mergedResults = append(mergedResults, matcherResult{
mapIndex: result.mapIndex,
matches:  mergeResults(mr.matches),
})
delete(m.childResults, result.mapIndex)
} else {
m.childResults[result.mapIndex] = mr
}
}
}
return mergedResults, nil
}

func (m *matchAnyInstance) dropIndices(dropIndices []uint32) {
for _, childInstance := range m.childInstances {
childInstance.dropIndices(dropIndices)
}
for _, mapIndex := range dropIndices {
delete(m.childResults, mapIndex)
}
}

func mergeResults(results []potentialMatches) potentialMatches {
if len(results) == 0 {
return nil
}
var sumLen int
for _, res := range results {
if res == nil {
return nil
}
sumLen += len(res)
}
merged := make(potentialMatches, 0, sumLen)
for {
best := -1
for i, res := range results {
if len(res) == 0 {
continue
}
if best < 0 || res[0] < results[best][0] {
best = i
}
}
if best < 0 {
return merged
}
if len(merged) == 0 || results[best][0] > merged[len(merged)-1] {
merged = append(merged, results[best][0])
}
results[best] = results[best][1:]
}
}

type matchSequence struct {
params               *Params
base, next           matcher
offset               uint64
statsLock            sync.Mutex
baseStats, nextStats matchOrderStats
}

func (m *matchSequence) newInstance(mapIndices []uint32) matcherInstance {
needMatched := make(map[uint32]struct{})
baseRequested := make(map[uint32]struct{})
nextRequested := make(map[uint32]struct{})
for _, mapIndex := range mapIndices {
needMatched[mapIndex] = struct{}{}
baseRequested[mapIndex] = struct{}{}
nextRequested[mapIndex] = struct{}{}
}
return &matchSequenceInstance{
matchSequence: m,
baseInstance:  m.base.newInstance(mapIndices),
nextInstance:  m.next.newInstance(mapIndices),
needMatched:   needMatched,
baseRequested: baseRequested,
nextRequested: nextRequested,
baseResults:   make(map[uint32]potentialMatches),
nextResults:   make(map[uint32]potentialMatches),
}
}

type matchOrderStats struct {
totalCount, nonEmptyCount, totalCost uint64
}

func (ms *matchOrderStats) add(empty bool, layerIndex uint32) {
if empty && layerIndex != 0 {
return
}
ms.totalCount++
if !empty {
ms.nonEmptyCount++
}
ms.totalCost += uint64(layerIndex + 1)
}

func (ms *matchOrderStats) mergeStats(add matchOrderStats) {
ms.totalCount += add.totalCount
ms.nonEmptyCount += add.nonEmptyCount
ms.totalCost += add.totalCost
}

func (m *matchSequence) baseFirst() bool {
m.statsLock.Lock()
bf := float64(m.baseStats.totalCost)*float64(m.nextStats.totalCount)+
float64(m.baseStats.nonEmptyCount)*float64(m.nextStats.totalCost) <
float64(m.baseStats.totalCost)*float64(m.nextStats.nonEmptyCount)+
float64(m.nextStats.totalCost)*float64(m.baseStats.totalCount)
m.statsLock.Unlock()
return bf
}

func (m *matchSequence) mergeBaseStats(stats matchOrderStats) {
m.statsLock.Lock()
m.baseStats.mergeStats(stats)
m.statsLock.Unlock()
}

func (m *matchSequence) mergeNextStats(stats matchOrderStats) {
m.statsLock.Lock()
m.nextStats.mergeStats(stats)
m.statsLock.Unlock()
}

func newMatchSequence(params *Params, matchers []matcher) matcher {
if len(matchers) == 0 {
panic("zero length sequence matchers are not allowed")
}
if len(matchers) == 1 {
return matchers[0]
}
return &matchSequence{
params: params,
base:   newMatchSequence(params, matchers[:len(matchers)-1]),
next:   matchers[len(matchers)-1],
offset: uint64(len(matchers) - 1),
}
}

type matchSequenceInstance struct {
*matchSequence
baseInstance, nextInstance                matcherInstance
baseRequested, nextRequested, needMatched map[uint32]struct{}
baseResults, nextResults                  map[uint32]potentialMatches
}

func (m *matchSequenceInstance) getMatchesForLayer(ctx context.Context, layerIndex uint32) (matchedResults []matcherResult, err error) {
baseFirst := m.baseFirst()
if baseFirst {
if err := m.evalBase(ctx, layerIndex); err != nil {
return nil, err
}
}
if err := m.evalNext(ctx, layerIndex); err != nil {
return nil, err
}
if !baseFirst {
if err := m.evalBase(ctx, layerIndex); err != nil {
return nil, err
}
}
for mapIndex := range m.needMatched {
if _, ok := m.baseRequested[mapIndex]; ok {
continue
}
if _, ok := m.nextRequested[mapIndex]; ok {
continue
}
matchedResults = append(matchedResults, matcherResult{
mapIndex: mapIndex,
matches:  m.params.matchResults(mapIndex, m.offset, m.baseResults[mapIndex], m.nextResults[mapIndex]),
})
delete(m.needMatched, mapIndex)
}
return matchedResults, nil
}

func (m *matchSequenceInstance) dropIndices(dropIndices []uint32) {
for _, mapIndex := range dropIndices {
delete(m.needMatched, mapIndex)
}
var dropBase, dropNext []uint32
for _, mapIndex := range dropIndices {
if m.dropBase(mapIndex) {
dropBase = append(dropBase, mapIndex)
}
}
m.baseInstance.dropIndices(dropBase)
for _, mapIndex := range dropIndices {
if m.dropNext(mapIndex) {
dropNext = append(dropNext, mapIndex)
}
}
m.nextInstance.dropIndices(dropNext)
}

func (m *matchSequenceInstance) evalBase(ctx context.Context, layerIndex uint32) error {
results, err := m.baseInstance.getMatchesForLayer(ctx, layerIndex)
if err != nil {
return fmt.Errorf("failed to evaluate base matcher on layer %d: %v", layerIndex, err)
}
var (
dropIndices []uint32
stats       matchOrderStats
)
for _, r := range results {
m.baseResults[r.mapIndex] = r.matches
delete(m.baseRequested, r.mapIndex)
stats.add(r.matches != nil && len(r.matches) == 0, layerIndex)
}
m.mergeBaseStats(stats)
for _, r := range results {
if m.dropNext(r.mapIndex) {
dropIndices = append(dropIndices, r.mapIndex)
}
}
if len(dropIndices) > 0 {
m.nextInstance.dropIndices(dropIndices)
}
return nil
}

func (m *matchSequenceInstance) evalNext(ctx context.Context, layerIndex uint32) error {
results, err := m.nextInstance.getMatchesForLayer(ctx, layerIndex)
if err != nil {
return fmt.Errorf("failed to evaluate next matcher on layer %d: %v", layerIndex, err)
}
var (
dropIndices []uint32
stats       matchOrderStats
)
for _, r := range results {
m.nextResults[r.mapIndex] = r.matches
delete(m.nextRequested, r.mapIndex)
stats.add(r.matches != nil && len(r.matches) == 0, layerIndex)
}
m.mergeNextStats(stats)
for _, r := range results {
if m.dropBase(r.mapIndex) {
dropIndices = append(dropIndices, r.mapIndex)
}
}
if len(dropIndices) > 0 {
m.baseInstance.dropIndices(dropIndices)
}
return nil
}

func (m *matchSequenceInstance) dropBase(mapIndex uint32) bool {
if _, ok := m.baseRequested[mapIndex]; !ok {
return false
}
if _, ok := m.needMatched[mapIndex]; ok {
if next := m.nextResults[mapIndex]; next == nil || len(next) > 0 {
return false
}
}
delete(m.baseRequested, mapIndex)
return true
}

func (m *matchSequenceInstance) dropNext(mapIndex uint32) bool {
if _, ok := m.nextRequested[mapIndex]; !ok {
return false
}
if _, ok := m.needMatched[mapIndex]; ok {
if base := m.baseResults[mapIndex]; base == nil || len(base) > 0 {
return false
}
}
delete(m.nextRequested, mapIndex)
return true
}

func (p *Params) matchResults(mapIndex uint32, offset uint64, baseRes, nextRes potentialMatches) potentialMatches {
if nextRes == nil || (baseRes != nil && len(baseRes) == 0) {
return baseRes
}
if baseRes == nil || len(nextRes) == 0 {
result := make(potentialMatches, 0, len(nextRes))
min := (uint64(mapIndex) << p.logValuesPerMap) + offset
for _, v := range nextRes {
if v >= min {
result = append(result, v-offset)
}
}
return result
}
maxLen := len(baseRes)
if l := len(nextRes); l < maxLen {
maxLen = l
}
matchedRes := make(potentialMatches, 0, maxLen)
for len(nextRes) > 0 && len(baseRes) > 0 {
if nextRes[0] > baseRes[0]+offset {
baseRes = baseRes[1:]
} else if nextRes[0] < baseRes[0]+offset {
nextRes = nextRes[1:]
} else {
matchedRes = append(matchedRes, baseRes[0])
baseRes = baseRes[1:]
nextRes = nextRes[1:]
}
}
return matchedRes
}

type runtimeStats struct {
dt, cnt, amount [stCount]int64
}

const (
stNone = iota
stFetchFirst
stFetchMore
stProcess
stGetLog
stOther
stCount
)

var stNames = []string{"", "fetchFirst", "fetchMore", "process", "getLog", "other"}

func (ts *runtimeStats) setState(state *int, newState int) {
if !doRuntimeStats || newState == *state {
return
}
now := int64(mclock.Now())
atomic.AddInt64(&ts.dt[*state], now)
atomic.AddInt64(&ts.dt[newState], -now)
atomic.AddInt64(&ts.cnt[newState], 1)
*state = newState
}

func (ts *runtimeStats) addAmount(state int, amount int64) {
atomic.AddInt64(&ts.amount[state], amount)
}

func (ts *runtimeStats) print() {
for i := 1; i < stCount; i++ {
log.Info("Matcher stats", "name", stNames[i], "dt", time.Duration(ts.dt[i]), "count", ts.cnt[i], "amount", ts.amount[i])
}
}
