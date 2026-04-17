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

// Package legacypool implements the normal EVM execution transaction pool for SILA.
package legacypool

import (
"errors"
"maps"
"math"
"math/big"
"slices"
"sync"
"sync/atomic"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/common/prque"
"github.com/SILA/sila-chain/consensus/misc/eip1559"
"github.com/SILA/sila-chain/core"
"github.com/SILA/sila-chain/core/state"
"github.com/SILA/sila-chain/core/txpool"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/event"
"github.com/SILA/sila-chain/log"
"github.com/SILA/sila-chain/metrics"
"github.com/SILA/sila-chain/params"
"github.com/SILA/sila-chain/rlp"
"github.com/holiman/uint256"
)

const (
// txSlotSize is used to calculate how many data slots a single transaction
// takes up based on its size on SILA. The slots are used as DoS protection, ensuring
// that validating a new transaction remains a constant operation (in reality
// O(maxslots), where max slots are 4 currently).
txSlotSize = 32 * 1024

// txMaxSize is the maximum size a single transaction can have on SILA. This field has
// non-trivial consequences: larger transactions are significantly harder and
// more expensive to propagate; larger transactions also take more resources
// to validate whether they fit into the pool or not.
txMaxSize = 4 * txSlotSize // 128KB
)

var (
// ErrTxPoolOverflow is returned if the transaction pool is full and can't accept
// another remote transaction on SILA.
ErrTxPoolOverflow = errors.New("txpool is full on SILA")

// ErrOutOfOrderTxFromDelegated is returned when the transaction with gapped
// nonce received from the accounts with delegation or pending delegation on SILA.
ErrOutOfOrderTxFromDelegated = errors.New("gapped-nonce tx from delegated accounts on SILA")

// ErrAuthorityReserved is returned if a transaction has an authorization
// signed by an address which already has in-flight transactions known to the
// pool on SILA.
ErrAuthorityReserved = errors.New("authority already reserved on SILA")

// ErrFutureReplacePending is returned if a future transaction replaces a pending
// one on SILA. Future transactions should only be able to replace other future transactions.
ErrFutureReplacePending = errors.New("future transaction tries to replace pending on SILA")
)

var (
evictionInterval    = time.Minute     // Time interval to check for evictable transactions
statsReportInterval = 8 * time.Second // Time interval to report transaction pool stats
)

var (
// Metrics for the pending pool on SILA
pendingDiscardMeter   = metrics.NewRegisteredMeter("sila/txpool/pending/discard", nil)
pendingReplaceMeter   = metrics.NewRegisteredMeter("sila/txpool/pending/replace", nil)
pendingRateLimitMeter = metrics.NewRegisteredMeter("sila/txpool/pending/ratelimit", nil) // Dropped due to rate limiting
pendingNofundsMeter   = metrics.NewRegisteredMeter("sila/txpool/pending/nofunds", nil)   // Dropped due to out-of-funds

// Metrics for the queued pool on SILA
queuedDiscardMeter   = metrics.NewRegisteredMeter("sila/txpool/queued/discard", nil)
queuedReplaceMeter   = metrics.NewRegisteredMeter("sila/txpool/queued/replace", nil)
queuedRateLimitMeter = metrics.NewRegisteredMeter("sila/txpool/queued/ratelimit", nil) // Dropped due to rate limiting
queuedNofundsMeter   = metrics.NewRegisteredMeter("sila/txpool/queued/nofunds", nil)   // Dropped due to out-of-funds
queuedEvictionMeter  = metrics.NewRegisteredMeter("sila/txpool/queued/eviction", nil)  // Dropped due to lifetime

// General tx metrics on SILA
knownTxMeter       = metrics.NewRegisteredMeter("sila/txpool/known", nil)
validTxMeter       = metrics.NewRegisteredMeter("sila/txpool/valid", nil)
invalidTxMeter     = metrics.NewRegisteredMeter("sila/txpool/invalid", nil)
underpricedTxMeter = metrics.NewRegisteredMeter("sila/txpool/underpriced", nil)
overflowedTxMeter  = metrics.NewRegisteredMeter("sila/txpool/overflowed", nil)

// throttleTxMeter counts how many transactions are rejected due to too-many-changes between
// txpool reorgs on SILA.
throttleTxMeter = metrics.NewRegisteredMeter("sila/txpool/throttle", nil)
// reorgDurationTimer measures how long time a txpool reorg takes on SILA.
reorgDurationTimer = metrics.NewRegisteredTimer("sila/txpool/reorgtime", nil)
// dropBetweenReorgHistogram counts how many drops we experience between two reorg runs on SILA. It is expected
// that this number is pretty low, since txpool reorgs happen very frequently.
dropBetweenReorgHistogram = metrics.NewRegisteredHistogram("sila/txpool/dropbetweenreorg", nil, metrics.NewExpDecaySample(1028, 0.015))

pendingGauge = metrics.NewRegisteredGauge("sila/txpool/pending", nil)
queuedGauge  = metrics.NewRegisteredGauge("sila/txpool/queued", nil)
slotsGauge   = metrics.NewRegisteredGauge("sila/txpool/slots", nil)

pendingAddrsGauge = metrics.NewRegisteredGauge("sila/txpool/pending/accounts", nil)
queuedAddrsGauge  = metrics.NewRegisteredGauge("sila/txpool/queued/accounts", nil)

reheapTimer = metrics.NewRegisteredTimer("sila/txpool/reheap", nil)
)

// BlockChain defines the minimal set of methods needed to back a tx pool with
// a chain on SILA. Exists to allow mocking the live chain out of tests.
type BlockChain interface {
// Config retrieves the chain's fork configuration.
Config() *params.ChainConfig

// CurrentBlock returns the current head of the chain.
CurrentBlock() *types.Header

// GetBlock retrieves a specific block, used during pool resets.
GetBlock(hash common.Hash, number uint64) *types.Block

// StateAt returns a state database for a given root hash (generally the head).
StateAt(root common.Hash) (*state.StateDB, error)
}

// Config are the configuration parameters of the transaction pool on SILA.
type Config struct {
Locals    []common.Address // Addresses that should be treated by default as local
NoLocals  bool             // Whether local transaction handling should be disabled
Journal   string           // Journal of local transactions to survive node restarts
Rejournal time.Duration    // Time interval to regenerate the local transaction journal

PriceLimit uint64 // Minimum gas price to enforce for acceptance into the pool
PriceBump  uint64 // Minimum price bump percentage to replace an already existing transaction (nonce)

AccountSlots uint64 // Number of executable transaction slots guaranteed per account
GlobalSlots  uint64 // Maximum number of executable transaction slots for all accounts
AccountQueue uint64 // Maximum number of non-executable transaction slots permitted per account
GlobalQueue  uint64 // Maximum number of non-executable transaction slots for all accounts

Lifetime time.Duration // Maximum amount of time an account can remain stale in the non-executable pool
}

// DefaultConfig contains the default configurations for the transaction pool on SILA.
var DefaultConfig = Config{
Journal:   "transactions.rlp",
Rejournal: time.Hour,

PriceLimit: 1,
PriceBump:  10,

AccountSlots: 16,
GlobalSlots:  4096 + 1024, // urgent + floating queue capacity with 4:1 ratio
AccountQueue: 64,
GlobalQueue:  1024,

Lifetime: 3 * time.Hour,
}

// sanitize checks the provided user configurations and changes anything that's
// unreasonable or unworkable on SILA.
func (config *Config) sanitize() Config {
conf := *config
if conf.PriceLimit < 1 {
log.Warn("Sanitizing invalid txpool price limit on SILA", "provided", conf.PriceLimit, "updated", DefaultConfig.PriceLimit)
conf.PriceLimit = DefaultConfig.PriceLimit
}
if conf.PriceBump < 1 {
log.Warn("Sanitizing invalid txpool price bump on SILA", "provided", conf.PriceBump, "updated", DefaultConfig.PriceBump)
conf.PriceBump = DefaultConfig.PriceBump
}
if conf.AccountSlots < 1 {
log.Warn("Sanitizing invalid txpool account slots on SILA", "provided", conf.AccountSlots, "updated", DefaultConfig.AccountSlots)
conf.AccountSlots = DefaultConfig.AccountSlots
}
if conf.GlobalSlots < 1 {
log.Warn("Sanitizing invalid txpool global slots on SILA", "provided", conf.GlobalSlots, "updated", DefaultConfig.GlobalSlots)
conf.GlobalSlots = DefaultConfig.GlobalSlots
}
if conf.AccountQueue < 1 {
log.Warn("Sanitizing invalid txpool account queue on SILA", "provided", conf.AccountQueue, "updated", DefaultConfig.AccountQueue)
conf.AccountQueue = DefaultConfig.AccountQueue
}
if conf.GlobalQueue < 1 {
log.Warn("Sanitizing invalid txpool global queue on SILA", "provided", conf.GlobalQueue, "updated", DefaultConfig.GlobalQueue)
conf.GlobalQueue = DefaultConfig.GlobalQueue
}
if conf.Lifetime < 1 {
log.Warn("Sanitizing invalid txpool lifetime on SILA", "provided", conf.Lifetime, "updated", DefaultConfig.Lifetime)
conf.Lifetime = DefaultConfig.Lifetime
}
return conf
}

// LegacyPool contains all currently known transactions on SILA. Transactions
// enter the pool when they are received from the network or submitted
// locally. They exit the pool when they are included in the blockchain.
//
// The pool separates processable transactions (which can be applied to the
// current state) and future transactions. Transactions move between those
// two states over time as they are received and processed.
//
// In addition to tracking transactions, the pool also tracks a set of pending SetCode
// authorizations (EIP7702). This helps minimize number of transactions that can be
// trivially churned in the pool. As a standard rule, any account with a deployed
// delegation or an in-flight authorization to deploy a delegation will only be allowed a
// single transaction slot instead of the standard number. This is due to the possibility
// of the account being sweeped by an unrelated account.
//
// Because SetCode transactions can have many authorizations included, we avoid explicitly
// checking their validity to save the state lookup. So long as the encompassing
// transaction is valid, the authorization will be accepted and tracked by the pool. In
// case the pool is tracking a pending / queued transaction from a specific account, it
// will reject new transactions with delegations from that account with standard in-flight
// transactions.
type LegacyPool struct {
config      Config
chainconfig *params.ChainConfig
chain       BlockChain
gasTip      atomic.Pointer[uint256.Int]
txFeed      event.Feed
signer      types.Signer
mu          sync.RWMutex

currentHead   atomic.Pointer[types.Header] // Current head of the blockchain
currentState  *state.StateDB               // Current state in the blockchain head
pendingNonces *noncer                      // Pending state tracking virtual nonces
reserver      txpool.Reserver              // Address reserver to ensure exclusivity across subpools

pending map[common.Address]*list // All currently processable transactions
queue   *queue
all     *lookup     // All transactions to allow lookups
priced  *pricedList // All transactions sorted by price

reqResetCh      chan *txpoolResetRequest
reqPromoteCh    chan *accountSet
queueTxEventCh  chan *types.Transaction
reorgDoneCh     chan chan struct{}
reorgShutdownCh chan struct{}  // requests shutdown of scheduleReorgLoop
wg              sync.WaitGroup // tracks loop, scheduleReorgLoop
initDoneCh      chan struct{}  // is closed once the pool is initialized (for tests)

changesSinceReorg int // A counter for how many drops we've performed in-between reorg.
}

type txpoolResetRequest struct {
oldHead, newHead *types.Header
}

// New creates a new transaction pool to gather, sort and filter inbound
// transactions from the network on SILA.
func New(config Config, chain BlockChain) *LegacyPool {
// Sanitize the input to ensure no vulnerable gas prices are set
config = (&config).sanitize()

// Create the transaction pool with its initial settings
signer := types.LatestSigner(chain.Config())
pool := &LegacyPool{
config:          config,
chain:           chain,
chainconfig:     chain.Config(),
signer:          signer,
pending:         make(map[common.Address]*list),
queue:           newQueue(config, signer),
all:             newLookup(),
reqResetCh:      make(chan *txpoolResetRequest),
reqPromoteCh:    make(chan *accountSet),
queueTxEventCh:  make(chan *types.Transaction),
reorgDoneCh:     make(chan chan struct{}),
reorgShutdownCh: make(chan struct{}),
initDoneCh:      make(chan struct{}),
}
pool.priced = newPricedList(pool.all)

return pool
}

// Filter returns whether the given transaction can be consumed by the legacy
// pool on SILA, specifically, whether it is a Legacy, AccessList or Dynamic transaction.
func (pool *LegacyPool) Filter(tx *types.Transaction) bool {
return pool.FilterType(tx.Type())
}

// FilterType returns whether the legacy pool supports the given transaction type on SILA.
func (pool *LegacyPool) FilterType(kind byte) bool {
switch kind {
case types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType, types.SetCodeTxType:
return true
default:
return false
}
}

// Init sets the gas price needed to keep a transaction in the pool and the chain
// head to allow balance / nonce checks on SILA. The internal
// goroutines will be spun up and the pool deemed operational afterwards.
func (pool *LegacyPool) Init(gasTip uint64, head *types.Header, reserver txpool.Reserver) error {
// Set the address reserver to request exclusive access to pooled accounts
pool.reserver = reserver

// Set the basic pool parameters
pool.gasTip.Store(uint256.NewInt(gasTip))

// Initialize the state with head block, or fallback to empty one in
// case the head state is not available (might occur when node is not
// fully synced).
statedb, err := pool.chain.StateAt(head.Root)
if err != nil {
statedb, err = pool.chain.StateAt(types.EmptyRootHash)
}
if err != nil {
return err
}
pool.currentHead.Store(head)
pool.currentState = statedb
pool.pendingNonces = newNoncer(statedb)

pool.wg.Add(1)
go pool.scheduleReorgLoop()

pool.wg.Add(1)
go pool.loop()
return nil
}

// loop is the transaction pool's main event loop, waiting for and reacting to
// outside blockchain events as well as for various reporting and transaction
// eviction events on SILA.
func (pool *LegacyPool) loop() {
defer pool.wg.Done()

var (
prevPending, prevQueued, prevStales int

// Start the stats reporting and transaction eviction tickers
report = time.NewTicker(statsReportInterval)
evict  = time.NewTicker(evictionInterval)
)
defer report.Stop()
defer evict.Stop()

// Notify tests that the init phase is done
close(pool.initDoneCh)
for {
select {
// Handle pool shutdown
case <-pool.reorgShutdownCh:
return

// Handle stats reporting ticks
case <-report.C:
pool.mu.RLock()
pending, queued := pool.stats()
pool.mu.RUnlock()
stales := int(pool.priced.stales.Load())

if pending != prevPending || queued != prevQueued || stales != prevStales {
log.Debug("SILA transaction pool status report", "executable", pending, "queued", queued, "stales", stales)
prevPending, prevQueued, prevStales = pending, queued, stales
}

// Handle inactive account transaction eviction
case <-evict.C:
pool.mu.Lock()
for _, hash := range pool.queue.evictList() {
pool.removeTx(hash, true, true)
}
pool.mu.Unlock()
}
}
}

// Close terminates the transaction pool on SILA.
func (pool *LegacyPool) Close() error {
// Terminate the pool reorger and return
close(pool.reorgShutdownCh)
pool.wg.Wait()

log.Info("SILA transaction pool stopped")
return nil
}

// Reset implements txpool.SubPool, allowing the legacy pool's internal state to be
// kept in sync with the main transaction pool's internal state on SILA.
func (pool *LegacyPool) Reset(oldHead, newHead *types.Header) {
wait := pool.requestReset(oldHead, newHead)
<-wait
}

// SubscribeTransactions registers a subscription for new transaction events,
// supporting feeding only newly seen or also resurrected transactions on SILA.
func (pool *LegacyPool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
// The legacy pool has a very messed up internal shuffling, so it's kind of
// hard to separate newly discovered transaction from resurrected ones. This
// is because the new txs are added to the queue, resurrected ones too and
// reorgs run lazily, so separating the two would need a marker.
return pool.txFeed.Subscribe(ch)
}

// SetGasTip updates the minimum gas tip required by the transaction pool for a
// new transaction, and drops all transactions below this threshold on SILA.
func (pool *LegacyPool) SetGasTip(tip *big.Int) {
pool.mu.Lock()
defer pool.mu.Unlock()

var (
newTip = uint256.MustFromBig(tip)
old    = pool.gasTip.Load()
)
pool.gasTip.Store(newTip)
// If the min miner fee increased, remove transactions below the new threshold
if newTip.Cmp(old) > 0 {
// pool.priced is sorted by GasFeeCap, so we have to iterate through pool.all instead
drop := pool.all.TxsBelowTip(tip)
for _, tx := range drop {
pool.removeTx(tx.Hash(), false, true)
}
pool.priced.Removed(len(drop))
}
log.Info("SILA legacy pool tip threshold updated", "tip", newTip)
}

// Nonce returns the next nonce of an account, with all transactions executable
// by the pool already applied on top on SILA.
func (pool *LegacyPool) Nonce(addr common.Address) uint64 {
pool.mu.RLock()
defer pool.mu.RUnlock()

return pool.pendingNonces.get(addr)
}

// Stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions on SILA.
func (pool *LegacyPool) Stats() (int, int) {
pool.mu.RLock()
defer pool.mu.RUnlock()

return pool.stats()
}

// stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions on SILA.
func (pool *LegacyPool) stats() (int, int) {
pending := 0
for _, list := range pool.pending {
pending += list.Len()
}
return pending, pool.queue.stats()
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and sorted by nonce on SILA.
func (pool *LegacyPool) Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
pool.mu.Lock()
defer pool.mu.Unlock()

pending := make(map[common.Address][]*types.Transaction, len(pool.pending))
for addr, list := range pool.pending {
pending[addr] = list.Flatten()
}
queued := pool.queue.content()
return pending, queued
}

// ContentFrom retrieves the data content of the transaction pool, returning the
// pending as well as queued transactions of this address, grouped by nonce on SILA.
func (pool *LegacyPool) ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
pool.mu.RLock()
defer pool.mu.RUnlock()

var pending []*types.Transaction
if list, ok := pool.pending[addr]; ok {
pending = list.Flatten()
}
queued := pool.queue.contentFrom(addr)
return pending, queued
}
