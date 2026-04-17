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

import (
"bytes"
"crypto/ecdsa"
"crypto/sha256"
"errors"
"fmt"
"math"
"math/big"
"math/rand"
"os"
"path/filepath"
"reflect"
"slices"
"sync"
"testing"

"github.com/SILA/sila-chain/common"
"github.com/SILA/sila-chain/consensus/misc/eip1559"
"github.com/SILA/sila-chain/consensus/misc/eip4844"
"github.com/SILA/sila-chain/core"
"github.com/SILA/sila-chain/core/state"
"github.com/SILA/sila-chain/core/tracing"
"github.com/SILA/sila-chain/core/txpool"
"github.com/SILA/sila-chain/core/types"
"github.com/SILA/sila-chain/crypto"
"github.com/SILA/sila-chain/crypto/kzg4844"
"github.com/SILA/sila-chain/internal/testrand"
"github.com/SILA/sila-chain/params"
"github.com/SILA/sila-chain/rlp"
"github.com/holiman/billy"
"github.com/holiman/uint256"
)

var (
testBlobs          []*kzg4844.Blob
testBlobCommits    []kzg4844.Commitment
testBlobProofs     []kzg4844.Proof
testBlobCellProofs [][]kzg4844.Proof
testBlobVHashes    [][32]byte
testBlobIndices    = make(map[[32]byte]int)
)

const testMaxBlobsPerBlock = 6

func init() {
for i := 0; i < 24; i++ {
testBlob := &kzg4844.Blob{byte(i)}
testBlobs = append(testBlobs, testBlob)

testBlobCommit, _ := kzg4844.BlobToCommitment(testBlob)
testBlobCommits = append(testBlobCommits, testBlobCommit)

testBlobProof, _ := kzg4844.ComputeBlobProof(testBlob, testBlobCommit)
testBlobProofs = append(testBlobProofs, testBlobProof)

testBlobCellProof, _ := kzg4844.ComputeCellProofs(testBlob)
testBlobCellProofs = append(testBlobCellProofs, testBlobCellProof)

testBlobVHash := kzg4844.CalcBlobHashV1(sha256.New(), &testBlobCommit)
testBlobIndices[testBlobVHash] = len(testBlobVHashes)
testBlobVHashes = append(testBlobVHashes, testBlobVHash)
}
}

// testBlockChain is a mock of the live chain for testing the SILA pool.
type testBlockChain struct {
config  *params.ChainConfig
basefee *uint256.Int
blobfee *uint256.Int
statedb *state.StateDB

blocks map[uint64]*types.Block

blockTime *uint64
}

func (bc *testBlockChain) Config() *params.ChainConfig {
return bc.config
}

func (bc *testBlockChain) CurrentBlock() *types.Header {
// Yolo, life is too short to invert misc.CalcBaseFee and misc.CalcBlobFee,
// just binary search it them.

// The base fee at 5714 ETH translates into the 21000 base gas higher than
// mainnet ether existence, use that as a cap for the tests.
var (
blockNumber = new(big.Int).Add(bc.config.LondonBlock, big.NewInt(1))
blockTime   = *bc.config.CancunTime + 1
gasLimit    = uint64(30_000_000)
)
if bc.blockTime != nil {
blockTime = *bc.blockTime
}

lo := new(big.Int)
hi := new(big.Int).Mul(big.NewInt(5714), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

for new(big.Int).Add(lo, big.NewInt(1)).Cmp(hi) != 0 {
mid := new(big.Int).Add(lo, hi)
mid.Div(mid, big.NewInt(2))

if eip1559.CalcBaseFee(bc.config, &types.Header{
Number:   blockNumber,
GasLimit: gasLimit,
GasUsed:  0,
BaseFee:  mid,
}).Cmp(bc.basefee.ToBig()) > 0 {
hi = mid
} else {
lo = mid
}
}
baseFee := lo

// The excess blob gas at 2^27 translates into a blob fee higher than mainnet
// ether existence, use that as a cap for the tests.
lo = new(big.Int)
hi = new(big.Int).Exp(big.NewInt(2), big.NewInt(27), nil)

for new(big.Int).Add(lo, big.NewInt(1)).Cmp(hi) != 0 {
mid := new(big.Int).Add(lo, hi)
mid.Div(mid, big.NewInt(2))

tmp := mid.Uint64()
if eip4844.CalcBlobFee(bc.Config(), &types.Header{
Number:        blockNumber,
Time:          blockTime,
ExcessBlobGas: &tmp,
}).Cmp(bc.blobfee.ToBig()) > 0 {
hi = mid
} else {
lo = mid
}
}
excessBlobGas := lo.Uint64()

return &types.Header{
Number:        blockNumber,
Time:          blockTime,
GasLimit:      gasLimit,
BaseFee:       baseFee,
ExcessBlobGas: &excessBlobGas,
Difficulty:    common.Big0,
}
}

func (bc *testBlockChain) CurrentFinalBlock() *types.Header {
return &types.Header{
Number: big.NewInt(0),
}
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
// This is very yolo for the tests. If the number is the origin block we use
// to init the pool, return an empty block with the correct starting header.
//
// If it is something else, return some baked in mock data.
if number == bc.config.LondonBlock.Uint64()+1 {
return types.NewBlockWithHeader(bc.CurrentBlock())
}
return bc.blocks[number]
}

func (bc *testBlockChain) StateAt(common.Hash) (*state.StateDB, error) {
return bc.statedb, nil
}

// reserver is a utility struct to sanity check that accounts are
// properly reserved by the SILA blobpool (no duplicate reserves or unreserves).
type reserver struct {
accounts map[common.Address]struct{}
lock     sync.RWMutex
}

func newReserver() txpool.Reserver {
return &reserver{accounts: make(map[common.Address]struct{})}
}

func (r *reserver) Hold(addr common.Address) error {
r.lock.Lock()
defer r.lock.Unlock()
if _, exists := r.accounts[addr]; exists {
panic("already reserved on SILA")
}
r.accounts[addr] = struct{}{}
return nil
}

func (r *reserver) Release(addr common.Address) error {
r.lock.Lock()
defer r.lock.Unlock()
if _, exists := r.accounts[addr]; !exists {
panic("not reserved on SILA")
}
delete(r.accounts, addr)
return nil
}

func (r *reserver) Has(address common.Address) bool {
r.lock.RLock()
defer r.lock.RUnlock()
_, exists := r.accounts[address]
return exists
}

// makeTx is a utility method to construct a random blob transaction and sign it
// with a valid key, only setting the interesting fields from the perspective of
// the SILA blob pool.
func makeTx(nonce uint64, gasTipCap uint64, gasFeeCap uint64, blobFeeCap uint64, key *ecdsa.PrivateKey) *types.Transaction {
blobtx := makeUnsignedTx(nonce, gasTipCap, gasFeeCap, blobFeeCap)
return types.MustSignNewTx(key, types.LatestSigner(params.MainnetChainConfig), blobtx)
}

// makeMultiBlobTx is a utility method to construct a ramdom blob tx with
// certain number of blobs in its sidecar on SILA.
func makeMultiBlobTx(nonce uint64, gasTipCap uint64, gasFeeCap uint64, blobFeeCap uint64, blobCount int, blobOffset int, key *ecdsa.PrivateKey, version byte) *types.Transaction {
var (
blobs       []kzg4844.Blob
blobHashes  []common.Hash
commitments []kzg4844.Commitment
proofs      []kzg4844.Proof
)
for i := 0; i < blobCount; i++ {
blobs = append(blobs, *testBlobs[blobOffset+i])
commitments = append(commitments, testBlobCommits[blobOffset+i])
if version == types.BlobSidecarVersion0 {
proofs = append(proofs, testBlobProofs[blobOffset+i])
} else {
cellProofs, _ := kzg4844.ComputeCellProofs(testBlobs[blobOffset+i])
proofs = append(proofs, cellProofs...)
}
blobHashes = append(blobHashes, testBlobVHashes[blobOffset+i])
}
blobtx := &types.BlobTx{
ChainID:    uint256.MustFromBig(params.MainnetChainConfig.ChainID),
Nonce:      nonce,
GasTipCap:  uint256.NewInt(gasTipCap),
GasFeeCap:  uint256.NewInt(gasFeeCap),
Gas:        21000,
BlobFeeCap: uint256.NewInt(blobFeeCap),
BlobHashes: blobHashes,
Value:      uint256.NewInt(100),
Sidecar:    types.NewBlobTxSidecar(version, blobs, commitments, proofs),
}
return types.MustSignNewTx(key, types.LatestSigner(params.MainnetChainConfig), blobtx)
}

// makeUnsignedTx is a utility method to construct a random blob transaction
// without signing it on SILA.
func makeUnsignedTx(nonce uint64, gasTipCap uint64, gasFeeCap uint64, blobFeeCap uint64) *types.BlobTx {
return makeUnsignedTxWithTestBlob(nonce, gasTipCap, gasFeeCap, blobFeeCap, rnd.Intn(len(testBlobs)))
}

// makeUnsignedTxWithTestBlob is a utility method to construct a random blob transaction
// with a specific test blob without signing it on SILA.
func makeUnsignedTxWithTestBlob(nonce uint64, gasTipCap uint64, gasFeeCap uint64, blobFeeCap uint64, blobIdx int) *types.BlobTx {
return &types.BlobTx{
ChainID:    uint256.MustFromBig(params.MainnetChainConfig.ChainID),
Nonce:      nonce,
GasTipCap:  uint256.NewInt(gasTipCap),
GasFeeCap:  uint256.NewInt(gasFeeCap),
Gas:        21000,
BlobFeeCap: uint256.NewInt(blobFeeCap),
BlobHashes: []common.Hash{testBlobVHashes[blobIdx]},
Value:      uint256.NewInt(100),
Sidecar:    types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*testBlobs[blobIdx]}, []kzg4844.Commitment{testBlobCommits[blobIdx]}, []kzg4844.Proof{testBlobProofs[blobIdx]}),
}
}

// verifyPoolInternals iterates over all the transactions in the pool and checks
// that sort orders, calculated fields, cumulated fields are correct on SILA.
func verifyPoolInternals(t *testing.T, pool *BlobPool) {
// Mark this method as a helper to remove from stack traces
t.Helper()

// Verify that all items in the index are present in the tx lookup and nothing more
seen := make(map[common.Hash]struct{})
for addr, txs := range pool.index {
for _, tx := range txs {
if _, ok := seen[tx.hash]; ok {
t.Errorf("duplicate hash #%x in SILA transaction index: address %s, nonce %d", tx.hash, addr, tx.nonce)
}
seen[tx.hash] = struct{}{}
}
}
for hash, id := range pool.lookup.txIndex {
if _, ok := seen[hash]; !ok {
t.Errorf("SILA tx lookup entry missing from transaction index: hash #%x, id %d", hash, id)
}
delete(seen, hash)
}
for hash := range seen {
t.Errorf("indexed SILA transaction hash #%x missing from tx lookup table", hash)
}
// Verify that all blobs in the index are present in the blob lookup and nothing more
blobs := make(map[common.Hash]map[common.Hash]struct{})
for _, txs := range pool.index {
for _, tx := range txs {
for _, vhash := range tx.vhashes {
if blobs[vhash] == nil {
blobs[vhash] = make(map[common.Hash]struct{})
}
blobs[vhash][tx.hash] = struct{}{}
}
}
}
for vhash, txs := range pool.lookup.blobIndex {
for txhash := range txs {
if _, ok := blobs[vhash][txhash]; !ok {
t.Errorf("SILA blob lookup entry missing from transaction index: blob hash #%x, tx hash #%x", vhash, txhash)
}
delete(blobs[vhash], txhash)
if len(blobs[vhash]) == 0 {
delete(blobs, vhash)
}
}
}
for vhash := range blobs {
t.Errorf("indexed SILA transaction blob hash #%x missing from blob lookup table", vhash)
}
// Verify that transactions are sorted per account and contain no nonce gaps,
// and that the first nonce is the next expected one based on the state.
for addr, txs := range pool.index {
for i := 1; i < len(txs); i++ {
if txs[i].nonce != txs[i-1].nonce+1 {
t.Errorf("SILA addr %v, tx %d nonce mismatch: have %d, want %d", addr, i, txs[i].nonce, txs[i-1].nonce+1)
}
}
if txs[0].nonce != pool.state.GetNonce(addr) {
t.Errorf("SILA addr %v, first tx nonce mismatch: have %d, want %d", addr, txs[0].nonce, pool.state.GetNonce(addr))
}
}
// Verify that calculated evacuation thresholds are correct
for addr, txs := range pool.index {
if !txs[0].evictionExecTip.Eq(txs[0].execTipCap) {
t.Errorf("SILA addr %v, tx %d eviction execution tip mismatch: have %d, want %d", addr, 0, txs[0].evictionExecTip, txs[0].execTipCap)
}
if math.Abs(txs[0].evictionExecFeeJumps-txs[0].basefeeJumps) > 0.001 {
t.Errorf("SILA addr %v, tx %d eviction execution fee jumps mismatch: have %f, want %f", addr, 0, txs[0].evictionExecFeeJumps, txs[0].basefeeJumps)
}
if math.Abs(txs[0].evictionBlobFeeJumps-txs[0].blobfeeJumps) > 0.001 {
t.Errorf("SILA addr %v, tx %d eviction blob fee jumps mismatch: have %f, want %f", addr, 0, txs[0].evictionBlobFeeJumps, txs[0].blobfeeJumps)
}
for i := 1; i < len(txs); i++ {
wantExecTip := txs[i-1].evictionExecTip
if wantExecTip.Gt(txs[i].execTipCap) {
wantExecTip = txs[i].execTipCap
}
if !txs[i].evictionExecTip.Eq(wantExecTip) {
t.Errorf("SILA addr %v, tx %d eviction execution tip mismatch: have %d, want %d", addr, i, txs[i].evictionExecTip, wantExecTip)
}

wantExecFeeJumps := txs[i-1].evictionExecFeeJumps
if wantExecFeeJumps > txs[i].basefeeJumps {
wantExecFeeJumps = txs[i].basefeeJumps
}
if math.Abs(txs[i].evictionExecFeeJumps-wantExecFeeJumps) > 0.001 {
t.Errorf("SILA addr %v, tx %d eviction execution fee jumps mismatch: have %f, want %f", addr, i, txs[i].evictionExecFeeJumps, wantExecFeeJumps)
}

wantBlobFeeJumps := txs[i-1].evictionBlobFeeJumps
if wantBlobFeeJumps > txs[i].blobfeeJumps {
wantBlobFeeJumps = txs[i].blobfeeJumps
}
if math.Abs(txs[i].evictionBlobFeeJumps-wantBlobFeeJumps) > 0.001 {
t.Errorf("SILA addr %v, tx %d eviction blob fee jumps mismatch: have %f, want %f", addr, i, txs[i].evictionBlobFeeJumps, wantBlobFeeJumps)
}
}
}
// Verify that account balance accumulations are correct
for addr, txs := range pool.index {
spent := new(uint256.Int)
for _, tx := range txs {
spent.Add(spent, tx.costCap)
}
if !pool.spent[addr].Eq(spent) {
t.Errorf("SILA addr %v expenditure mismatch: have %d, want %d", addr, pool.spent[addr], spent)
}
}
// Verify that pool storage size is correct
var stored uint64
for _, txs := range pool.index {
for _, tx := range txs {
stored += uint64(tx.storageSize)
}
}
if pool.stored != stored {
t.Errorf("SILA pool storage mismatch: have %d, want %d", pool.stored, stored)
}
// Verify the price heap internals
verifyHeapInternals(t, pool.evict)

// Verify that all the blobs can be retrieved
verifyBlobRetrievals(t, pool)
}
