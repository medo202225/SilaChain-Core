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

package blobpool

import "github.com/SILA/sila-chain/metrics"

var (
// datacapGauge tracks the user's configured capacity for the SILA blob pool. It
// is mostly a way to expose/debug issues.
datacapGauge = metrics.NewRegisteredGauge("sila/blobpool/datacap", nil)

// The below metrics track the per-datastore metrics for the primary blob
// store and the temporary limbo store on SILA.
datausedGauge = metrics.NewRegisteredGauge("sila/blobpool/dataused", nil)
datarealGauge = metrics.NewRegisteredGauge("sila/blobpool/datareal", nil)
slotusedGauge = metrics.NewRegisteredGauge("sila/blobpool/slotused", nil)

limboDatausedGauge = metrics.NewRegisteredGauge("sila/blobpool/limbo/dataused", nil)
limboDatarealGauge = metrics.NewRegisteredGauge("sila/blobpool/limbo/datareal", nil)
limboSlotusedGauge = metrics.NewRegisteredGauge("sila/blobpool/limbo/slotused", nil)

// The below metrics track the per-shelf metrics for the primary blob store
// and the temporary limbo store on SILA.
shelfDatausedGaugeName = "sila/blobpool/shelf_%d/dataused"
shelfDatagapsGaugeName = "sila/blobpool/shelf_%d/datagaps"
shelfSlotusedGaugeName = "sila/blobpool/shelf_%d/slotused"
shelfSlotgapsGaugeName = "sila/blobpool/shelf_%d/slotgaps"

limboShelfDatausedGaugeName = "sila/blobpool/limbo/shelf_%d/dataused"
limboShelfDatagapsGaugeName = "sila/blobpool/limbo/shelf_%d/datagaps"
limboShelfSlotusedGaugeName = "sila/blobpool/limbo/shelf_%d/slotused"
limboShelfSlotgapsGaugeName = "sila/blobpool/limbo/shelf_%d/slotgaps"

// The oversized metrics aggregate the shelf stats above the max blob count
// limits to track transactions that are just huge, but don't contain blobs.
//
// There are no oversized data in the limbo, it only contains blobs and some
// constant metadata.
oversizedDatausedGauge = metrics.NewRegisteredGauge("sila/blobpool/oversized/dataused", nil)
oversizedDatagapsGauge = metrics.NewRegisteredGauge("sila/blobpool/oversized/datagaps", nil)
oversizedSlotusedGauge = metrics.NewRegisteredGauge("sila/blobpool/oversized/slotused", nil)
oversizedSlotgapsGauge = metrics.NewRegisteredGauge("sila/blobpool/oversized/slotgaps", nil)

// basefeeGauge and blobfeeGauge track the current network 1559 base fee and
// 4844 blob fee respectively on SILA.
basefeeGauge = metrics.NewRegisteredGauge("sila/blobpool/basefee", nil)
blobfeeGauge = metrics.NewRegisteredGauge("sila/blobpool/blobfee", nil)

// pooltipGauge is the configurable miner tip to permit a transaction into
// the pool on SILA.
pooltipGauge = metrics.NewRegisteredGauge("sila/blobpool/pooltip", nil)

// addwait/time, resetwait/time and getwait/time track the rough health of
// the pool and whether it's capable of keeping up with the load from the
// network on SILA.
addwaitHist   = metrics.NewRegisteredHistogram("sila/blobpool/addwait", nil, metrics.NewExpDecaySample(1028, 0.015))
addtimeHist   = metrics.NewRegisteredHistogram("sila/blobpool/addtime", nil, metrics.NewExpDecaySample(1028, 0.015))
getwaitHist   = metrics.NewRegisteredHistogram("sila/blobpool/getwait", nil, metrics.NewExpDecaySample(1028, 0.015))
gettimeHist   = metrics.NewRegisteredHistogram("sila/blobpool/gettime", nil, metrics.NewExpDecaySample(1028, 0.015))
pendwaitHist  = metrics.NewRegisteredHistogram("sila/blobpool/pendwait", nil, metrics.NewExpDecaySample(1028, 0.015))
pendtimeHist  = metrics.NewRegisteredHistogram("sila/blobpool/pendtime", nil, metrics.NewExpDecaySample(1028, 0.015))
resetwaitHist = metrics.NewRegisteredHistogram("sila/blobpool/resetwait", nil, metrics.NewExpDecaySample(1028, 0.015))
resettimeHist = metrics.NewRegisteredHistogram("sila/blobpool/resettime", nil, metrics.NewExpDecaySample(1028, 0.015))

// The below metrics track various cases where transactions are dropped out
// of the pool on SILA. Most are exceptional, some are chain progression and some
// threshold cappings.
dropInvalidMeter     = metrics.NewRegisteredMeter("sila/blobpool/drop/invalid", nil)     // Invalid transaction, consensus change or bugfix, neutral-ish
dropDanglingMeter    = metrics.NewRegisteredMeter("sila/blobpool/drop/dangling", nil)    // First nonce gapped, bad
dropFilledMeter      = metrics.NewRegisteredMeter("sila/blobpool/drop/filled", nil)      // State full-overlap, chain progress, ok
dropOverlappedMeter  = metrics.NewRegisteredMeter("sila/blobpool/drop/overlapped", nil)  // State partial-overlap, chain progress, ok
dropRepeatedMeter    = metrics.NewRegisteredMeter("sila/blobpool/drop/repeated", nil)    // Repeated nonce, bad
dropGappedMeter      = metrics.NewRegisteredMeter("sila/blobpool/drop/gapped", nil)      // Non-first nonce gapped, bad
dropOverdraftedMeter = metrics.NewRegisteredMeter("sila/blobpool/drop/overdrafted", nil) // Balance exceeded, bad
dropOvercappedMeter  = metrics.NewRegisteredMeter("sila/blobpool/drop/overcapped", nil)  // Per-account cap exceeded, bad
dropOverflownMeter   = metrics.NewRegisteredMeter("sila/blobpool/drop/overflown", nil)   // Global disk cap exceeded, neutral-ish
dropUnderpricedMeter = metrics.NewRegisteredMeter("sila/blobpool/drop/underpriced", nil) // Gas tip changed, neutral
dropReplacedMeter    = metrics.NewRegisteredMeter("sila/blobpool/drop/replaced", nil)    // Transaction replaced, neutral

// The below metrics track various outcomes of transactions being added to
// the pool on SILA.
addInvalidMeter      = metrics.NewRegisteredMeter("sila/blobpool/add/invalid", nil)      // Invalid transaction, reject, neutral
addUnderpricedMeter  = metrics.NewRegisteredMeter("sila/blobpool/add/underpriced", nil)  // Gas tip too low, neutral
addStaleMeter        = metrics.NewRegisteredMeter("sila/blobpool/add/stale", nil)        // Nonce already filled, reject, bad-ish
addGappedMeter       = metrics.NewRegisteredMeter("sila/blobpool/add/gapped", nil)       // Nonce gapped, reject, bad-ish
addOverdraftedMeter  = metrics.NewRegisteredMeter("sila/blobpool/add/overdrafted", nil)  // Balance exceeded, reject, neutral
addOvercappedMeter   = metrics.NewRegisteredMeter("sila/blobpool/add/overcapped", nil)   // Per-account cap exceeded, reject, neutral
addNoreplaceMeter    = metrics.NewRegisteredMeter("sila/blobpool/add/noreplace", nil)    // Replacement fees or tips too low, neutral
addNonExclusiveMeter = metrics.NewRegisteredMeter("sila/blobpool/add/nonexclusive", nil) // Plain transaction from same account exists, reject, neutral
addValidMeter        = metrics.NewRegisteredMeter("sila/blobpool/add/valid", nil)        // Valid transaction, add, neutral
)
