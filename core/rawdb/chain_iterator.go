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

package rawdb

import (
"encoding/binary"
"runtime"
"sync/atomic"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/common/prque"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/ethdb"
"github.com/SILA/sila-chain/log"
"github.com/SILA/sila-chain/rlp"
)

// InitDatabaseFromFreezer reinitializes an empty database from a previous batch
// of frozen ancient blocks.
func InitDatabaseFromFreezer(db ethdb.Database) {
frozen, err := db.Ancients()
if err != nil || frozen == 0 {
return
}
var (
batch  = db.NewBatch()
start  = time.Now()
logged = start.Add(-7 * time.Second)
hash   common.Hash
)
for i := uint64(0); i < frozen; {
count := uint64(100_000)
if i+count > frozen {
count = frozen - i
}
data, err := db.AncientRange(ChainFreezerHashTable, i, count, 32*count)
if err != nil {
log.Crit("Failed to init database from freezer", "err", err)
}
for j, h := range data {
number := i + uint64(j)
hash = common.BytesToHash(h)
WriteHeaderNumber(batch, hash, number)
if batch.ValueSize() > ethdb.IdealBatchSize {
if err := batch.Write(); err != nil {
log.Crit("Failed to write data to db", "err", err)
}
batch.Reset()
}
}
i += uint64(len(data))
if time.Since(logged) > 8*time.Second {
log.Info("Initializing database from freezer", "total", frozen, "number", i, "hash", hash, "elapsed", common.PrettyDuration(time.Since(start)))
logged = time.Now()
}
}
if err := batch.Write(); err != nil {
log.Crit("Failed to write data to db", "err", err)
}
batch.Reset()

WriteHeadHeaderHash(db, hash)
WriteHeadFastBlockHash(db, hash)
log.Info("Initialized database from freezer", "blocks", frozen, "elapsed", common.PrettyDuration(time.Since(start)))
}

type blockTxHashes struct {
number uint64
hashes []common.Hash
err    error
}

// iterateTransactions iterates over all transactions in the (canon) block numbers given.
func iterateTransactions(db ethdb.Database, from uint64, to uint64, reverse bool, interrupt chan struct{}) chan *blockTxHashes {
type numberRlp struct {
number uint64
rlp    rlp.RawValue
}
if to == from {
return nil
}
threads := to - from
if cpus := runtime.NumCPU(); threads > uint64(cpus) {
threads = uint64(cpus)
}
var (
rlpCh    = make(chan *numberRlp, threads*2)
hashesCh = make(chan *blockTxHashes, threads*2)
)
lookup := func() {
n, end := from, to
if reverse {
n, end = to-1, from-1
}
defer close(rlpCh)
for n != end {
data := ReadCanonicalBodyRLP(db, n, nil)
select {
case rlpCh <- &numberRlp{n, data}:
case <-interrupt:
return
}
if reverse {
n--
} else {
n++
}
}
}
var nThreadsAlive atomic.Int32
nThreadsAlive.Store(int32(threads))
process := func() {
defer func() {
if nThreadsAlive.Add(-1) == 0 {
close(hashesCh)
}
}()
for data := range rlpCh {
var body types.Body
var result *blockTxHashes
if err := rlp.DecodeBytes(data.rlp, &body); err != nil {
log.Warn("Failed to decode block body", "block", data.number, "error", err)
result = &blockTxHashes{
number: data.number,
err:    err,
}
} else {
hashes := make([]common.Hash, len(body.Transactions))
for i, tx := range body.Transactions {
hashes[i] = tx.Hash()
}
result = &blockTxHashes{
hashes: hashes,
number: data.number,
}
}
select {
case hashesCh <- result:
case <-interrupt:
return
}
}
}
go lookup()
for i := 0; i < int(threads); i++ {
go process()
}
return hashesCh
}

// indexTransactions creates txlookup indices of the specified block range.
func indexTransactions(db ethdb.Database, from uint64, to uint64, interrupt chan struct{}, hook func(uint64) bool, report bool) {
if from >= to {
return
}
var (
hashesCh = iterateTransactions(db, from, to, true, interrupt)
batch    = db.NewBatch()
start    = time.Now()
logged   = start.Add(-7 * time.Second)
lastNum  = to
queue    = prque.New[int64, *blockTxHashes](nil)
blocks, txs = 0, 0
)
for chanDelivery := range hashesCh {
queue.Push(chanDelivery, int64(chanDelivery.number))
for !queue.Empty() {
if _, priority := queue.Peek(); priority != int64(lastNum-1) {
break
}
if hook != nil && !hook(lastNum-1) {
break
}
delivery := queue.PopItem()
lastNum = delivery.number
if delivery.err != nil {
log.Warn("Skipping tx indexing for block with missing/corrupt body", "block", delivery.number, "error", delivery.err)
continue
}
WriteTxLookupEntries(batch, delivery.number, delivery.hashes)
blocks++
txs += len(delivery.hashes)
if batch.ValueSize() > ethdb.IdealBatchSize {
WriteTxIndexTail(batch, lastNum)
if err := batch.Write(); err != nil {
log.Crit("Failed writing batch to db", "error", err)
return
}
batch.Reset()
}
if time.Since(logged) > 8*time.Second {
log.Info("Indexing transactions", "blocks", blocks, "txs", txs, "tail", lastNum, "total", to-from, "elapsed", common.PrettyDuration(time.Since(start)))
logged = time.Now()
}
}
}
WriteTxIndexTail(batch, lastNum)
if err := batch.Write(); err != nil {
log.Crit("Failed writing batch to db", "error", err)
return
}
logger := log.Debug
if report {
logger = log.Info
}
select {
case <-interrupt:
logger("Transaction indexing interrupted", "blocks", blocks, "txs", txs, "tail", lastNum, "elapsed", common.PrettyDuration(time.Since(start)))
default:
logger("Indexed transactions", "blocks", blocks, "txs", txs, "tail", lastNum, "elapsed", common.PrettyDuration(time.Since(start)))
}
}

// IndexTransactions creates txlookup indices of the specified block range.
func IndexTransactions(db ethdb.Database, from uint64, to uint64, interrupt chan struct{}, report bool) {
indexTransactions(db, from, to, interrupt, nil, report)
}

// indexTransactionsForTesting is the internal debug version with an additional hook.
func indexTransactionsForTesting(db ethdb.Database, from uint64, to uint64, interrupt chan struct{}, hook func(uint64) bool) {
indexTransactions(db, from, to, interrupt, hook, false)
}

// unindexTransactions removes txlookup indices of the specified block range.
func unindexTransactions(db ethdb.Database, from uint64, to uint64, interrupt chan struct{}, hook func(uint64) bool, report bool) {
if from >= to {
return
}
var (
hashesCh = iterateTransactions(db, from, to, false, interrupt)
batch    = db.NewBatch()
start    = time.Now()
logged   = start.Add(-7 * time.Second)
nextNum  = from
queue    = prque.New[int64, *blockTxHashes](nil)
blocks, txs = 0, 0
)
for delivery := range hashesCh {
queue.Push(delivery, -int64(delivery.number))
for !queue.Empty() {
if _, priority := queue.Peek(); -priority != int64(nextNum) {
break
}
if hook != nil && !hook(nextNum) {
break
}
delivery := queue.PopItem()
nextNum = delivery.number + 1
if delivery.err != nil {
log.Warn("Skipping tx unindexing for block with missing/corrupt body", "block", delivery.number, "error", delivery.err)
continue
}
DeleteTxLookupEntries(batch, delivery.hashes)
txs += len(delivery.hashes)
blocks++

if blocks%1000 == 0 {
WriteTxIndexTail(batch, nextNum)
if err := batch.Write(); err != nil {
log.Crit("Failed writing batch to db", "error", err)
return
}
batch.Reset()
}
if time.Since(logged) > 8*time.Second {
log.Info("Unindexing transactions", "blocks", blocks, "txs", txs, "total", to-from, "elapsed", common.PrettyDuration(time.Since(start)))
logged = time.Now()
}
}
}
WriteTxIndexTail(batch, nextNum)
if err := batch.Write(); err != nil {
log.Crit("Failed writing batch to db", "error", err)
return
}
logger := log.Debug
if report {
logger = log.Info
}
select {
case <-interrupt:
logger("Transaction unindexing interrupted", "blocks", blocks, "txs", txs, "tail", nextNum, "elapsed", common.PrettyDuration(time.Since(start)))
default:
logger("Unindexed transactions", "blocks", blocks, "txs", txs, "tail", nextNum, "elapsed", common.PrettyDuration(time.Since(start)))
}
}

// UnindexTransactions removes txlookup indices of the specified block range.
func UnindexTransactions(db ethdb.Database, from uint64, to uint64, interrupt chan struct{}, report bool) {
unindexTransactions(db, from, to, interrupt, nil, report)
}

// unindexTransactionsForTesting is the internal debug version with an additional hook.
func unindexTransactionsForTesting(db ethdb.Database, from uint64, to uint64, interrupt chan struct{}, hook func(uint64) bool) {
unindexTransactions(db, from, to, interrupt, hook, false)
}

// PruneTransactionIndex removes all tx index entries below a certain block number.
func PruneTransactionIndex(db ethdb.Database, pruneBlock uint64) {
tail := ReadTxIndexTail(db)
if tail == nil || *tail > pruneBlock {
return
}
var count, removed int
DeleteAllTxLookupEntries(db, func(txhash common.Hash, v []byte) bool {
count++
if count%10000000 == 0 {
log.Info("Pruning tx index", "count", count, "removed", removed)
}
if len(v) > 8 {
log.Error("Skipping legacy tx index entry", "hash", txhash)
return false
}
bn := decodeNumber(v)
if bn < pruneBlock {
removed++
return true
}
return false
})
WriteTxIndexTail(db, pruneBlock)
}

func decodeNumber(b []byte) uint64 {
var numBuffer [8]byte
copy(numBuffer[8-len(b):], b)
return binary.BigEndian.Uint64(numBuffer[:])
}
