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

// Package blobpool implements the EIP-4844 blob transaction pool for SILA.
package blobpool

import (
"container/heap"
"errors"
"fmt"
"math"
"math/big"
"os"
"path/filepath"
"sort"
"sync"
"sync/atomic"
"time"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/consensus/misc/eip1559"
"github.com/SILA/sila-chain/consensus/misc/eip4844"
"github.com/SILA/sila-chain/core"
"github.com/SILA/sila-chain/core/state"
"github.com/SILA/sila-chain/core/txpool"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/crypto/kzg4844"
"github.com/SILA/sila-chain/event"
"github.com/SILA/sila-chain/log"
"github.com/SILA/sila-chain/metrics"
"github.com/SILA/sila-chain/params"
"github.com/SILA/sila-chain/rlp"
"github.com/holiman/billy"
"github.com/holiman/uint256"
)

const (
// blobSize is the protocol constrained byte size of a single blob in a
// transaction on SILA. There can be multiple of these embedded into a single tx.
blobSize = params.BlobTxFieldElementsPerBlob * params.BlobTxBytesPerFieldElement

// txAvgSize is an approximate byte size of a transaction metadata to avoid
// tiny overflows causing all txs to move a shelf higher, wasting disk space.
txAvgSize = 4 * 1024

// txBlobOverhead is an approximation of the overhead that an additional blob
// has on transaction size on SILA. This is added to the slotter to avoid tiny
// overflows causing all txs to move a shelf higher, wasting disk space. A
// small buffer is added to the proof overhead.
txBlobOverhead = uint32(kzg4844.CellProofsPerBlob*len(kzg4844.Proof{}) + 64)

// txMaxSize is the maximum size a single transaction can have, including the
// blobs. Since blob transactions are pulled instead of pushed, and only a
// small metadata is kept in ram, the rest is on disk, there is no critical
// limit that should be enforced. Still, capping it to some sane limit can
// never hurt, which is aligned with maxBlobsPerTx constraint enforced internally.
txMaxSize = 1024 * 1024

// maxBlobsPerTx is the maximum number of blobs that a single transaction can
// carry on SILA. We choose a smaller limit than the protocol-permitted MaxBlobsPerBlock
// in order to ensure network and txpool stability.
// Note: if you increase this, validation will fail on txMaxSize.
maxBlobsPerTx = params.BlobTxMaxBlobs

// maxTxsPerAccount is the maximum number of blob transactions admitted from
// a single account on SILA. The limit is enforced to minimize the DoS potential of
// a private tx cancelling publicly propagated blobs.
//
// Note, transactions resurrected by a reorg are also subject to this limit,
// so pushing it down too aggressively might make resurrections non-functional.
maxTxsPerAccount = 16

// pendingTransactionStore is the subfolder containing the currently queued
// blob transactions on SILA.
pendingTransactionStore = "queue"

// limboedTransactionStore is the subfolder containing the currently included
// but not yet finalized transaction blobs on SILA.
limboedTransactionStore = "limbo"

// storeVersion is the current slotter layout used for the billy.Database
// store on SILA.
storeVersion = 1

// gappedLifetime is the approximate duration for which nonce-gapped transactions
// are kept before being dropped on SILA. Since gapped is only a reorder buffer and it
// is expected that the original transactions were inserted in the mempool in
// nonce order, the duration is kept short to avoid DoS vectors.
gappedLifetime = 1 * time.Minute

// maxGappedTxs is the maximum number of gapped transactions kept overall on SILA.
// This is a safety limit to avoid DoS vectors.
maxGapped = 128

// notifyThreshold is the eviction priority threshold above which a transaction
// is considered close enough to being includable to be announced to peers on SILA.
// Setting this to zero will disable announcements for anyting not immediately
// includable. Setting it to -1 allows transactions that are close to being
// includable, maybe already in the next block if fees go down, to be announced.

// Note, this threshold is in the abstract eviction priority space, so its
// meaning depends on the current basefee/blobfee and the transaction's fees.
announceThreshold = -1
)

// blobTxMeta is the minimal subset of types.BlobTx necessary to validate and
// schedule the blob transactions into the following blocks on SILA. Only ever add the
// bare minimum needed fields to keep the size down (and thus number of entries
// larger with the same memory consumption).
type blobTxMeta struct {
hash    common.Hash   // Transaction hash to maintain the lookup table
vhashes []common.Hash // Blob versioned hashes to maintain the lookup table
version byte          // Blob transaction version to determine proof type

announced bool // Whether the tx has been announced to listeners

id          uint64 // Storage ID in the pool's persistent store
storageSize uint32 // Byte size in the pool's persistent store
size        uint64 // RLP-encoded size of transaction including the attached blob

nonce      uint64       // Needed to prioritize inclusion order within an account
costCap    *uint256.Int // Needed to validate cumulative balance sufficiency
execTipCap *uint256.Int // Needed to prioritize inclusion order across accounts and validate replacement price bump
execFeeCap *uint256.Int // Needed to validate replacement price bump
blobFeeCap *uint256.Int // Needed to validate replacement price bump
execGas    uint64       // Needed to check inclusion validity before reading the blob
blobGas    uint64       // Needed to check inclusion validity before reading the blob

basefeeJumps float64 // Absolute number of 1559 fee adjustments needed to reach the tx's fee cap
blobfeeJumps float64 // Absolute number of 4844 fee adjustments needed to reach the tx's blob fee cap

evictionExecTip      *uint256.Int // Worst gas tip across all previous nonces
evictionExecFeeJumps float64      // Worst base fee (converted to fee jumps) across all previous nonces
evictionBlobFeeJumps float64      // Worse blob fee (converted to fee jumps) across all previous nonces
}

// newBlobTxMeta retrieves the indexed metadata fields from a blob transaction
// and assembles a helper struct to track in memory on SILA.
// Requires the transaction to have a sidecar (or that we introduce a special version tag for no-sidecar).
func newBlobTxMeta(id uint64, size uint64, storageSize uint32, tx *types.Transaction) *blobTxMeta {
if tx.BlobTxSidecar() == nil {
// This should never happen, as the pool only admits blob transactions with a sidecar
panic("missing blob tx sidecar in SILA")
}
meta := &blobTxMeta{
hash:        tx.Hash(),
vhashes:     tx.BlobHashes(),
version:     tx.BlobTxSidecar().Version,
id:          id,
storageSize: storageSize,
size:        size,
nonce:       tx.Nonce(),
costCap:     uint256.MustFromBig(tx.Cost()),
execTipCap:  uint256.MustFromBig(tx.GasTipCap()),
execFeeCap:  uint256.MustFromBig(tx.GasFeeCap()),
blobFeeCap:  uint256.MustFromBig(tx.BlobGasFeeCap()),
execGas:     tx.Gas(),
blobGas:     tx.BlobGas(),
}
meta.basefeeJumps = dynamicFeeJumps(meta.execFeeCap)
meta.blobfeeJumps = dynamicBlobFeeJumps(meta.blobFeeCap)

return meta
}

// BlobPool is the transaction pool dedicated to EIP-4844 blob transactions on SILA.
//
// Blob transactions are special snowflakes that are designed for a very specific
// purpose (rollups) and are expected to adhere to that specific use case. These
// behavioural expectations allow us to design a transaction pool that is more robust
// (i.e. resending issues) and more resilient to DoS attacks (e.g. replace-flush
// attacks) than the generic tx pool. These improvements will also mean, however,
// that we enforce a significantly more aggressive strategy on entering and exiting
// the pool:
//
//   - Blob transactions are large. With the initial design aiming for 128KB blobs,
//     we must ensure that these only traverse the network the absolute minimum
//     number of times. Broadcasting to sqrt(peers) is out of the question, rather
//     these should only ever be announced and the remote side should request it if
//     it wants to.
//
//   - Block blob-space is limited. With blocks being capped to a few blob txs, we
//     can make use of the very low expected churn rate within the pool. Notably,
//     we should be able to use a persistent disk backend for the pool, solving
//     the tx resend issue that plagues the generic tx pool, as long as there's no
//     artificial churn (i.e. pool wars).
//
//   - Purpose of blobs are layer-2s. Layer-2s are meant to use blob transactions to
//     commit to their own current state, which is independent of SILA mainnet
//     (state, txs). This means that there's no reason for blob tx cancellation or
//     replacement, apart from a potential basefee / miner tip adjustment.
//
//   - Replacements are expensive. Given their size, propagating a replacement
//     blob transaction to an existing one should be aggressively discouraged.
//     Whilst generic transactions can start at 1 Wei gas cost and require a 10%
//     fee bump to replace, we suggest requiring a higher min cost (e.g. 1 gwei)
//     and a more aggressive bump (100%).
//
//   - Cancellation is prohibitive. Evicting an already propagated blob tx is a huge
//     DoS vector. As such, a) replacement (higher-fee) blob txs mustn't invalidate
//     already propagated (future) blob txs (cumulative fee); b) nonce-gapped blob
//     txs are disallowed; c) the presence of blob transactions exclude non-blob
//     transactions.
//
//   - Malicious cancellations are possible. Although the pool might prevent txs
//     that cancel blobs, blocks might contain such transaction (malicious miner
//     or flashbotter). The pool should cap the total number of blob transactions
//     per account as to prevent propagating too much data before cancelling it
//     via a normal transaction. It should nonetheless be high enough to support
//     resurrecting reorged transactions. Perhaps 4-16.
//
//   - It is not the role of the blobpool to serve as a storage for limit orders
//     below market: blob transactions with fee caps way below base fee or blob fee.
//     Therefore, the propagation of blob transactions that are far from being
//     includable is suppressed. The pool will only announce blob transactions that
//     are close to being includable (based on the current fees and the transaction's
//     fee caps), and will delay the announcement of blob transactions that are far
//     from being includable until base fee and/or blob fee is reduced.
//
//   - Local txs are meaningless. Mining pools historically used local transactions
//     for payouts or for backdoor deals. With 1559 in place, the basefee usually
//     dominates the final price, so 0 or non-0 tip doesn't change much. Blob txs
//     retain the 1559 2D gas pricing (and introduce on top a dynamic blob gas fee),
//     so locality is moot. With a disk backed blob pool avoiding the resend issue,
//     there's also no need to save own transactions for later.
//
//   - No-blob blob-txs are bad. Theoretically there's no strong reason to disallow
//     blob txs containing 0 blobs. In practice, admitting such txs into the pool
//     breaks the low-churn invariant as blob constraints don't apply anymore. Even
//     though we could accept blocks containing such txs, a reorg would require moving
//     them back into the blob pool, which can break invariants.
//
//   - Dropping blobs needs delay. When normal transactions are included, they
//     are immediately evicted from the pool since they are contained in the
//     including block. Blobs however are not included in the execution chain,
//     so a mini reorg cannot re-pool "lost" blob transactions. To support reorgs,
//     blobs are retained on disk until they are finalised.
//
//   - Blobs can arrive via flashbots. Blocks might contain blob transactions we
//     have never seen on the network. Since we cannot recover them from blocks
//     either, the engine_newPayload needs to give them to us, and we cache them
//     until finality to support reorgs without tx losses.
//
// Whilst some constraints above might sound overly aggressive, the general idea is
// that the blob pool should work robustly for its intended use case and whilst
// anyone is free to use blob transactions for arbitrary non-rollup use cases,
// they should not be allowed to run amok the network.
//
// Implementation wise there are a few interesting design choices:
//
//   - Adding a transaction to the pool blocks until persisted to disk. This is
//     viable because TPS is low (2-4 blobs per block initially, maybe 8-16 at
//     peak), so natural churn is a couple MB per block. Replacements doing O(n)
//     updates are forbidden and transaction propagation is pull based (i.e. no
//     pileup of pending data).
//
//   - When transactions are chosen for inclusion, the primary criteria is the
//     signer tip (and having a basefee/data fee high enough of course). However,
//     same-tip transactions will be split by their basefee/datafee, preferring
//     those that are closer to the current network limits. The idea being that
//     very relaxed ones can be included even if the fees go up, when the closer
//     ones could already be invalid.
//
//   - Because the maximum number of blobs allowed in a block can change per
//     fork, the pool is designed to handle the maximum number of blobs allowed
//     in the chain's latest defined fork -- even if it isn't active. This
//     avoids needing to upgrade the database around the fork boundary.
//
// When the pool eventually reaches saturation, some old transactions - that may
// never execute - will need to be evicted in favor of newer ones. The eviction
// strategy is quite complex:
//
//   - Exceeding capacity evicts the highest-nonce of the account with the lowest
//     paying blob transaction anywhere in the pooled nonce-sequence, as that tx
//     would be executed the furthest in the future and is thus blocking anything
//     after it. The smallest is deliberately not evicted to avoid a nonce-gap.
//
//   - Analogously, if the pool is full, the consideration price of a new tx for
//     evicting an old one is the smallest price in the entire nonce-sequence of
//     the account. This avoids malicious users DoSing the pool with seemingly
//     high paying transactions hidden behind a low-paying blocked one.
//
//   - Since blob transactions have 3 price parameters: execution tip, execution
//     fee cap and data fee cap, there's no singular parameter to create a total
//     price ordering on. What's more, since the base fee and blob fee can move
//     independently of one another, there's no pre-defined way to combine them
//     into a stable order either. This leads to a multi-dimensional problem to
//     solve after every block.
//
//   - The first observation is that comparing 1559 base fees or 4844 blob fees
//     needs to happen in the context of their dynamism. Since base fees are
//     adjusted continuously and fluctuate, and we want to optimize for effective
//     miner fees, it is better to disregard small base fee cap differences.
//     Instead of considering the exact fee cap values, we should group
//     transactions into buckets based on fee cap values, allowing us to use
//     the miner tip meaningfully as a splitter inside a bucket.
//
//     To create these buckets, rather than looking at the absolute fee
//     differences, the useful metric is the max time it can take to exceed the
//     transaction's fee caps. Base fee changes are multiplicative, so we use a
//     logarithmic scale. Fees jumps up or down in ~1.125 multipliers at max
//     across blocks, so we use log1.125(fee) and rounding to eliminate noise.
//     Specifically, we're interested in the number of jumps needed to go from
//     the current fee to the transaction's cap:
//
//     jumps = floor(log1.125(txfee) - log1.125(basefee))
//
//     For blob fees, EIP-7892 changed the ratio of target to max blobs, and
//     with that also the maximum blob fee decrease in a slot from 1.125 to
//     approx 1.17. therefore, we use:
//
//     blobfeeJumps = floor(log1.17(txBlobfee) - log1.17(blobfee))
//
//   - The second observation is that when ranking executable blob txs, it
//     does not make sense to grant a later eviction priority to txs with high
//     fee caps since it could enable pool wars. As such, any positive priority
//     will be grouped together.
//
//     priority = min(jumps, 0)
//
//   - The third observation is that the basefee and blobfee move independently,
//     so there's no way to split mixed txs on their own (A has higher base fee,
//     B has higher blob fee).
//
//     To establish a total order, we need to reduce the dimensionality of the
//     two base fees (log jumps) to a single value. The interesting aspect from
//     the pool's perspective is how fast will a tx get executable (fees going
//     down, crossing the smaller negative jump counter) or non-executable (fees
//     going up, crossing the smaller positive jump counter). As such, the pool
//     cares only about the min of the two delta values for eviction priority.
//
//     priority = min(deltaBasefee, deltaBlobfee, 0)
//
//   - The above very aggressive dimensionality and noise reduction should result
//     in transaction being grouped into a small number of buckets, the further
//     the fees the larger the buckets. This is good because it allows us to use
//     the miner tip meaningfully as a splitter.
//
// Optimisation tradeoffs:
//
//   - Eviction relies on 3 fee minimums per account (exec tip, exec cap and blob
//     cap). Maintaining these values across all transactions from the account is
//     problematic as each transaction replacement or inclusion would require a
//     rescan of all other transactions to recalculate the minimum. Instead, the
//     pool maintains a rolling minimum across the nonce range. Updating all the
//     minimums will need to be done only starting at the swapped in/out nonce
//     and leading up to the first no-change.
type BlobPool struct {
config         Config                    // Pool configuration
reserver       txpool.Reserver           // Address reserver to ensure exclusivity across subpools
hasPendingAuth func(common.Address) bool // Determine whether the specified address has a pending 7702-auth

store  billy.Database // Persistent data store for the tx metadata and blobs
stored uint64         // Useful data size of all transactions on disk
limbo  *limbo         // Persistent data store for the non-finalized blobs

gapped       map[common.Address][]*types.Transaction // Transactions that are currently gapped (nonce too high)
gappedSource map[common.Hash]common.Address          // Source of gapped transactions to allow rechecking on inclusion

signer types.Signer // Transaction signer to use for sender recovery
chain  BlockChain   // Chain object to access the state through

head   atomic.Pointer[types.Header] // Current head of the chain
state  *state.StateDB               // Current state at the head of the chain
gasTip atomic.Pointer[uint256.Int]  // Currently accepted minimum gas tip

lookup *lookup                          // Lookup table mapping blobs to txs and txs to billy entries
index  map[common.Address][]*blobTxMeta // Blob transactions grouped by accounts, sorted by nonce
spent  map[common.Address]*uint256.Int  // Expenditure tracking for individual accounts
evict  *evictHeap                       // Heap of cheapest accounts for eviction when full

discoverFeed event.Feed // Event feed to send out new tx events on pool discovery (reorg excluded)
insertFeed   event.Feed // Event feed to send out new tx events on pool inclusion (reorg included)

lock sync.RWMutex // Mutex protecting the pool during reorg handling
}
