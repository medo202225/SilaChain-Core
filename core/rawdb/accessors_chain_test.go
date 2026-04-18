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
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"silachain/common"
	"silachain/core/types"
	"silachain/core/types/bal"
	"silachain/crypto"
	"silachain/crypto/keccak"
	"silachain/params"
	"silachain/rlp"
)

// Tests block header storage and retrieval operations.
func TestHeaderStorage(t *testing.T) {
	db := NewMemoryDatabase()

	header := &types.Header{Number: big.NewInt(42), Extra: []byte("test header")}
	if entry := ReadHeader(db, header.Hash(), header.Number.Uint64()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	WriteHeader(db, header)
	if entry := ReadHeader(db, header.Hash(), header.Number.Uint64()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != header.Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, header)
	}
	if entry := ReadHeaderRLP(db, header.Hash(), header.Number.Uint64()); entry == nil {
		t.Fatalf("Stored header RLP not found")
	} else {
		if hash := crypto.Keccak256Hash(entry); hash != header.Hash() {
			t.Fatalf("Retrieved RLP header mismatch: have %v, want %v", entry, header)
		}
	}
	DeleteHeader(db, header.Hash(), header.Number.Uint64())
	if entry := ReadHeader(db, header.Hash(), header.Number.Uint64()); entry != nil {
		t.Fatalf("Deleted header returned: %v", entry)
	}
}

// Tests block body storage and retrieval operations.
func TestBodyStorage(t *testing.T) {
	db := NewMemoryDatabase()

	body := &types.Body{Uncles: []*types.Header{{Extra: []byte("test header")}}}
	hasher := keccak.NewLegacyKeccak256()
	rlp.Encode(hasher, body)
	hash := common.BytesToHash(hasher.Sum(nil))

	if entry := ReadBody(db, hash, 0); entry != nil {
		t.Fatalf("Non existent body returned: %v", entry)
	}
	WriteBody(db, hash, 0, body)
	if entry := ReadBody(db, hash, 0); entry == nil {
		t.Fatalf("Stored body not found")
	} else if types.DeriveSha(types.Transactions(entry.Transactions), newTestHasher()) != types.DeriveSha(types.Transactions(body.Transactions), newTestHasher()) || types.CalcUncleHash(entry.Uncles) != types.CalcUncleHash(body.Uncles) {
		t.Fatalf("Retrieved body mismatch: have %v, want %v", entry, body)
	}
	if entry := ReadBodyRLP(db, hash, 0); entry == nil {
		t.Fatalf("Stored body RLP not found")
	} else {
		if calc := crypto.Keccak256Hash(entry); calc != hash {
			t.Fatalf("Retrieved RLP body mismatch: have %v, want %v", entry, body)
		}
	}
	DeleteBody(db, hash, 0)
	if entry := ReadBody(db, hash, 0); entry != nil {
		t.Fatalf("Deleted body returned: %v", entry)
	}
}

// Tests block storage and retrieval operations.
func TestBlockStorage(t *testing.T) {
	db := NewMemoryDatabase()

	block := types.NewBlockWithHeader(&types.Header{
		Extra:       []byte("test block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
	})
	if entry := ReadBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	if entry := ReadHeader(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	if entry := ReadBody(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent body returned: %v", entry)
	}
	WriteBlock(db, block)
	if entry := ReadBlock(db, block.Hash(), block.NumberU64()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
	if entry := ReadHeader(db, block.Hash(), block.NumberU64()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != block.Header().Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, block.Header())
	}
	if entry := ReadBody(db, block.Hash(), block.NumberU64()); entry == nil {
		t.Fatalf("Stored body not found")
	} else if types.DeriveSha(types.Transactions(entry.Transactions), newTestHasher()) != types.DeriveSha(block.Transactions(), newTestHasher()) || types.CalcUncleHash(entry.Uncles) != types.CalcUncleHash(block.Uncles()) {
		t.Fatalf("Retrieved body mismatch: have %v, want %v", entry, block.Body())
	}
	DeleteBlock(db, block.Hash(), block.NumberU64())
	if entry := ReadBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Deleted block returned: %v", entry)
	}
	if entry := ReadHeader(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Deleted header returned: %v", entry)
	}
	if entry := ReadBody(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Deleted body returned: %v", entry)
	}
}

// Tests that partial block contents don'\”t get reassembled into full blocks.
func TestPartialBlockStorage(t *testing.T) {
	db := NewMemoryDatabase()
	block := types.NewBlockWithHeader(&types.Header{
		Extra:       []byte("test block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
	})
	WriteHeader(db, block.Header())
	if entry := ReadBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	DeleteHeader(db, block.Hash(), block.NumberU64())

	WriteBody(db, block.Hash(), block.NumberU64(), block.Body())
	if entry := ReadBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	DeleteBody(db, block.Hash(), block.NumberU64())

	WriteHeader(db, block.Header())
	WriteBody(db, block.Hash(), block.NumberU64(), block.Body())

	if entry := ReadBlock(db, block.Hash(), block.NumberU64()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
}

// Tests block storage and retrieval operations.
func TestBadBlockStorage(t *testing.T) {
	db := NewMemoryDatabase()

	block := types.NewBlockWithHeader(&types.Header{
		Number:      big.NewInt(1),
		Extra:       []byte("bad block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
	})
	if entry := ReadBadBlock(db, block.Hash()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	WriteBadBlock(db, block)
	if entry := ReadBadBlock(db, block.Hash()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
	blockTwo := types.NewBlockWithHeader(&types.Header{
		Number:      big.NewInt(2),
		Extra:       []byte("bad block two"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
	})
	WriteBadBlock(db, blockTwo)

	WriteBadBlock(db, block)
	badBlocks := ReadAllBadBlocks(db)
	if len(badBlocks) != 2 {
		t.Fatalf("Failed to load all bad blocks")
	}

	for _, n := range rand.Perm(100) {
		block := types.NewBlockWithHeader(&types.Header{
			Number:      big.NewInt(int64(n)),
			Extra:       []byte("bad block"),
			UncleHash:   types.EmptyUncleHash,
			TxHash:      types.EmptyTxsHash,
			ReceiptHash: types.EmptyReceiptsHash,
		})
		WriteBadBlock(db, block)
	}
	badBlocks = ReadAllBadBlocks(db)
	if len(badBlocks) != badBlockToKeep {
		t.Fatalf("The number of persised bad blocks in incorrect %d", len(badBlocks))
	}
	for i := 0; i < len(badBlocks)-1; i++ {
		if badBlocks[i].NumberU64() < badBlocks[i+1].NumberU64() {
			t.Fatalf("The bad blocks are not sorted #[%d](%d) < #[%d](%d)", i, badBlocks[i].NumberU64(), i+1, badBlocks[i+1].NumberU64())
		}
	}
}

// Tests that canonical numbers can be mapped to hashes and retrieved.
func TestCanonicalMappingStorage(t *testing.T) {
	db := NewMemoryDatabase()

	hash, number := common.Hash{0: 0xff}, uint64(314)
	if entry := ReadCanonicalHash(db, number); entry != (common.Hash{}) {
		t.Fatalf("Non existent canonical mapping returned: %v", entry)
	}
	WriteCanonicalHash(db, hash, number)
	if entry := ReadCanonicalHash(db, number); entry == (common.Hash{}) {
		t.Fatalf("Stored canonical mapping not found")
	} else if entry != hash {
		t.Fatalf("Retrieved canonical mapping mismatch: have %v, want %v", entry, hash)
	}
	DeleteCanonicalHash(db, number)
	if entry := ReadCanonicalHash(db, number); entry != (common.Hash{}) {
		t.Fatalf("Deleted canonical mapping returned: %v", entry)
	}
}

// Tests that head headers and head blocks can be assigned, individually.
func TestHeadStorage(t *testing.T) {
	db := NewMemoryDatabase()

	blockHead := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block header")})
	blockFull := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block full")})
	blockFast := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block fast")})

	if entry := ReadHeadHeaderHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non head header entry returned: %v", entry)
	}
	if entry := ReadHeadBlockHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non head block entry returned: %v", entry)
	}
	if entry := ReadHeadFastBlockHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non fast head block entry returned: %v", entry)
	}
	WriteHeadHeaderHash(db, blockHead.Hash())
	WriteHeadBlockHash(db, blockFull.Hash())
	WriteHeadFastBlockHash(db, blockFast.Hash())

	if entry := ReadHeadHeaderHash(db); entry != blockHead.Hash() {
		t.Fatalf("Head header hash mismatch: have %v, want %v", entry, blockHead.Hash())
	}
	if entry := ReadHeadBlockHash(db); entry != blockFull.Hash() {
		t.Fatalf("Head block hash mismatch: have %v, want %v", entry, blockFull.Hash())
	}
	if entry := ReadHeadFastBlockHash(db); entry != blockFast.Hash() {
		t.Fatalf("Fast head block hash mismatch: have %v, want %v", entry, blockFast.Hash())
	}
}

// Tests that receipts associated with a single block can be stored and retrieved.
func TestBlockReceiptStorage(t *testing.T) {
	db := NewMemoryDatabase()

	tx1 := types.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(1), 1, big.NewInt(1), nil)
	tx2 := types.NewTransaction(2, common.HexToAddress("0x2"), big.NewInt(2), 2, big.NewInt(2), nil)

	body := &types.Body{Transactions: types.Transactions{tx1, tx2}}

	receipt1 := &types.Receipt{
		Status:            types.ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          tx1.Hash(),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}
	receipt1.Bloom = types.CreateBloom(receipt1)

	receipt2 := &types.Receipt{
		PostState:         common.Hash{2}.Bytes(),
		CumulativeGasUsed: 2,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x22})},
			{Address: common.BytesToAddress([]byte{0x02, 0x22})},
		},
		TxHash:          tx2.Hash(),
		ContractAddress: common.BytesToAddress([]byte{0x02, 0x22, 0x22}),
		GasUsed:         222222,
	}
	receipt2.Bloom = types.CreateBloom(receipt2)
	receipts := []*types.Receipt{receipt1, receipt2}

	hash := common.BytesToHash([]byte{0x03, 0x14})
	if rs := ReadReceipts(db, hash, 0, 0, params.TestChainConfig); len(rs) != 0 {
		t.Fatalf("non existent receipts returned: %v", rs)
	}
	WriteBody(db, hash, 0, body)

	WriteReceipts(db, hash, 0, receipts)
	if rs := ReadReceipts(db, hash, 0, 0, params.TestChainConfig); len(rs) == 0 {
		t.Fatal("no receipts returned")
	} else {
		if err := checkReceiptsRLP(rs, receipts); err != nil {
			t.Fatal(err)
		}
	}
	DeleteBody(db, hash, 0)
	if rs := ReadReceipts(db, hash, 0, 0, params.TestChainConfig); rs != nil {
		t.Fatalf("receipts returned when body was deleted: %v", rs)
	}
	raw := ReadRawReceipts(db, hash, 0)
	for _, r := range raw {
		r.Bloom = types.CreateBloom(r)
	}
	if err := checkReceiptsRLP(raw, receipts); err != nil {
		t.Fatal(err)
	}
	WriteBody(db, hash, 0, body)

	DeleteReceipts(db, hash, 0)
	if rs := ReadReceipts(db, hash, 0, 0, params.TestChainConfig); len(rs) != 0 {
		t.Fatalf("deleted receipts returned: %v", rs)
	}
}

func checkReceiptsRLP(have, want types.Receipts) error {
	if len(have) != len(want) {
		return fmt.Errorf("receipts sizes mismatch: have %d, want %d", len(have), len(want))
	}
	for i := 0; i < len(want); i++ {
		rlpHave, err := rlp.EncodeToBytes(have[i])
		if err != nil {
			return err
		}
		rlpWant, err := rlp.EncodeToBytes(want[i])
		if err != nil {
			return err
		}
		if !bytes.Equal(rlpHave, rlpWant) {
			return fmt.Errorf("receipt #%d: receipt mismatch: have %s, want %s", i, hex.EncodeToString(rlpHave), hex.EncodeToString(rlpWant))
		}
	}
	return nil
}

func TestAncientStorage(t *testing.T) {
	frdir := t.TempDir()
	db, err := Open(NewMemoryDatabase(), OpenOptions{Ancient: frdir})
	if err != nil {
		t.Fatalf("failed to create database with ancient backend")
	}
	defer db.Close()

	block := types.NewBlockWithHeader(&types.Header{
		Number:      big.NewInt(0),
		Extra:       []byte("test block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
	})
	hash, number := block.Hash(), block.NumberU64()
	if blob := ReadHeaderRLP(db, hash, number); len(blob) > 0 {
		t.Fatalf("non existent header returned")
	}
	if blob := ReadBodyRLP(db, hash, number); len(blob) > 0 {
		t.Fatalf("non existent body returned")
	}
	if blob := ReadReceiptsRLP(db, hash, number); len(blob) > 0 {
		t.Fatalf("non existent receipts returned")
	}
	if blob := ReadCanonicalReceiptsRLP(db, number, &hash); len(blob) > 0 {
		t.Fatalf("non existent receipts returned")
	}

	WriteAncientBlocks(db, []*types.Block{block}, types.EncodeBlockReceiptLists([]types.Receipts{nil}))

	if blob := ReadHeaderRLP(db, hash, number); len(blob) == 0 {
		t.Fatalf("no header returned")
	}
	if blob := ReadBodyRLP(db, hash, number); len(blob) == 0 {
		t.Fatalf("no body returned")
	}
	if blob := ReadReceiptsRLP(db, hash, number); len(blob) == 0 {
		t.Fatalf("no receipts returned")
	}
	if blob := ReadCanonicalReceiptsRLP(db, number, &hash); len(blob) == 0 {
		t.Fatalf("no receipts returned")
	}

	fakeHash := common.BytesToHash([]byte{0x01, 0x02, 0x03})
	if blob := ReadHeaderRLP(db, fakeHash, number); len(blob) != 0 {
		t.Fatalf("invalid header returned")
	}
	if blob := ReadBodyRLP(db, fakeHash, number); len(blob) != 0 {
		t.Fatalf("invalid body returned")
	}
	if blob := ReadReceiptsRLP(db, fakeHash, number); len(blob) != 0 {
		t.Fatalf("invalid receipts returned")
	}
}

func TestWriteAncientHeaderChain(t *testing.T) {
	db, err := Open(NewMemoryDatabase(), OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatalf("failed to create database with ancient backend")
	}
	defer db.Close()

	var headers []*types.Header
	headers = append(headers, &types.Header{
		Number:      big.NewInt(0),
		Extra:       []byte("test block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
	})
	headers = append(headers, &types.Header{
		Number:      big.NewInt(1),
		Extra:       []byte("test block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
	})
	WriteAncientHeaderChain(db, headers)

	for _, header := range headers {
		if blob := ReadHeaderRLP(db, header.Hash(), header.Number.Uint64()); len(blob) == 0 {
			t.Fatalf("no header returned")
		}
		if h := ReadCanonicalHash(db, header.Number.Uint64()); h != header.Hash() {
			t.Fatalf("no canonical hash returned")
		}
		if blob := ReadBodyRLP(db, header.Hash(), header.Number.Uint64()); len(blob) != 0 {
			t.Fatalf("unexpected body returned")
		}
		if blob := ReadReceiptsRLP(db, header.Hash(), header.Number.Uint64()); len(blob) != 0 {
			t.Fatalf("unexpected receipts returned")
		}
	}
}

func TestHashesInRange(t *testing.T) {
	mkHeader := func(number, seq int) *types.Header {
		h := types.Header{
			Difficulty: new(big.Int),
			Number:     big.NewInt(int64(number)),
			GasLimit:   uint64(seq),
		}
		return &h
	}
	db := NewMemoryDatabase()
	total := 0
	for i := 0; i < 15; i++ {
		for ii := 0; ii < i; ii++ {
			WriteHeader(db, mkHeader(i, ii))
			total++
		}
	}
	if have, want := len(ReadAllHashes(db, 10)), 10; have != want {
		t.Fatalf("Wrong number of hashes read, want %d, got %d", want, have)
	}
	if have, want := len(ReadAllHashes(db, 16)), 0; have != want {
		t.Fatalf("Wrong number of hashes read, want %d, got %d", want, have)
	}
	if have, want := len(ReadAllHashes(db, 1)), 1; have != want {
		t.Fatalf("Wrong number of hashes read, want %d, got %d", want, have)
	}
}

func BenchmarkWriteAncientBlocks(b *testing.B) {
	frdir := b.TempDir()
	db, err := Open(NewMemoryDatabase(), OpenOptions{Ancient: frdir})
	if err != nil {
		b.Fatalf("failed to create database with ancient backend")
	}
	defer db.Close()

	const batchSize = 128
	const blockTxs = 20
	allBlocks := makeTestBlocks(b.N, blockTxs)
	batchReceipts := makeTestReceipts(batchSize, blockTxs)
	b.ResetTimer()

	var totalSize int64
	for i := 0; i < b.N; i += batchSize {
		length := batchSize
		if i+batchSize > b.N {
			length = b.N - i
		}

		blocks := allBlocks[i : i+length]
		receipts := batchReceipts[:length]
		writeSize, err := WriteAncientBlocks(db, blocks, types.EncodeBlockReceiptLists(receipts))
		if err != nil {
			b.Fatal(err)
		}
		totalSize += writeSize
	}

	b.SetBytes(totalSize / int64(b.N))
}

func makeTestBlocks(nblock int, txsPerBlock int) []*types.Block {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	signer := types.LatestSignerForChainID(big.NewInt(8))

	txs := make([]*types.Transaction, txsPerBlock)
	for i := 0; i < len(txs); i++ {
		var err error
		to := common.Address{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
		txs[i], err = types.SignNewTx(key, signer, &types.LegacyTx{
			Nonce:    2,
			GasPrice: big.NewInt(30000),
			Gas:      0x45454545,
			To:       &to,
		})
		if err != nil {
			panic(err)
		}
	}

	blocks := make([]*types.Block, nblock)
	for i := 0; i < nblock; i++ {
		header := &types.Header{
			Number: big.NewInt(int64(i)),
			Extra:  []byte("test block"),
		}
		blocks[i] = types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: txs})
		blocks[i].Hash()
	}
	return blocks
}

func makeTestReceipts(n int, nPerBlock int) []types.Receipts {
	receipts := make([]*types.Receipt, nPerBlock)
	var logs []*types.Log
	for i := 0; i < 5; i++ {
		logs = append(logs, new(types.Log))
	}
	for i := 0; i < len(receipts); i++ {
		receipts[i] = &types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: 0x888888888,
			Logs:              logs,
		}
	}
	allReceipts := make([]types.Receipts, n)
	for i := 0; i < n; i++ {
		allReceipts[i] = receipts
	}
	return allReceipts
}

type fullLogRLP struct {
	Address        common.Address
	Topics         []common.Hash
	Data           []byte
	BlockNumber    uint64
	BlockTimestamp uint64
	TxHash         common.Hash
	TxIndex        uint
	BlockHash      common.Hash
	Index          uint
}

func newFullLogRLP(l *types.Log) *fullLogRLP {
	return &fullLogRLP{
		Address:        l.Address,
		Topics:         l.Topics,
		Data:           l.Data,
		BlockNumber:    l.BlockNumber,
		BlockTimestamp: l.BlockTimestamp,
		TxHash:         l.TxHash,
		TxIndex:        l.TxIndex,
		BlockHash:      l.BlockHash,
		Index:          l.Index,
	}
}

func TestReadLogs(t *testing.T) {
	db := NewMemoryDatabase()

	tx1 := types.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(1), 1, big.NewInt(1), nil)
	tx2 := types.NewTransaction(2, common.HexToAddress("0x2"), big.NewInt(2), 2, big.NewInt(2), nil)

	body := &types.Body{Transactions: types.Transactions{tx1, tx2}}

	receipt1 := &types.Receipt{
		Status:            types.ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          tx1.Hash(),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}
	receipt1.Bloom = types.CreateBloom(receipt1)

	receipt2 := &types.Receipt{
		PostState:         common.Hash{2}.Bytes(),
		CumulativeGasUsed: 2,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x22})},
			{Address: common.BytesToAddress([]byte{0x02, 0x22})},
		},
		TxHash:          tx2.Hash(),
		ContractAddress: common.BytesToAddress([]byte{0x02, 0x22, 0x22}),
		GasUsed:         222222,
	}
	receipt2.Bloom = types.CreateBloom(receipt2)
	receipts := []*types.Receipt{receipt1, receipt2}

	hash := common.BytesToHash([]byte{0x03, 0x14})
	if rs := ReadReceipts(db, hash, 0, 0, params.TestChainConfig); len(rs) != 0 {
		t.Fatalf("non existent receipts returned: %v", rs)
	}
	WriteBody(db, hash, 0, body)

	WriteReceipts(db, hash, 0, receipts)

	logs := ReadLogs(db, hash, 0)
	if len(logs) == 0 {
		t.Fatalf("no logs returned")
	}
	if have, want := len(logs), 2; have != want {
		t.Fatalf("unexpected number of logs returned, have %d want %d", have, want)
	}
	if have, want := len(logs[0]), 2; have != want {
		t.Fatalf("unexpected number of logs[0] returned, have %d want %d", have, want)
	}
	if have, want := len(logs[1]), 2; have != want {
		t.Fatalf("unexpected number of logs[1] returned, have %d want %d", have, want)
	}

	for i, pr := range receipts {
		for j, pl := range pr.Logs {
			rlpHave, err := rlp.EncodeToBytes(newFullLogRLP(logs[i][j]))
			if err != nil {
				t.Fatal(err)
			}
			rlpWant, err := rlp.EncodeToBytes(newFullLogRLP(pl))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(rlpHave, rlpWant) {
				t.Fatalf("receipt #%d: receipt mismatch: have %s, want %s", i, hex.EncodeToString(rlpHave), hex.EncodeToString(rlpWant))
			}
		}
	}
}

func TestDeriveLogFields(t *testing.T) {
	to2 := common.HexToAddress("0x2")
	to3 := common.HexToAddress("0x3")
	txs := types.Transactions{
		types.NewTx(&types.LegacyTx{
			Nonce:    1,
			Value:    big.NewInt(1),
			Gas:      1,
			GasPrice: big.NewInt(1),
		}),
		types.NewTx(&types.LegacyTx{
			To:       &to2,
			Nonce:    2,
			Value:    big.NewInt(2),
			Gas:      2,
			GasPrice: big.NewInt(2),
		}),
		types.NewTx(&types.AccessListTx{
			To:       &to3,
			Nonce:    3,
			Value:    big.NewInt(3),
			Gas:      3,
			GasPrice: big.NewInt(3),
		}),
	}
	receipts := []*types.Receipt{
		{
			Logs: []*types.Log{
				{Address: common.BytesToAddress([]byte{0x11})},
				{Address: common.BytesToAddress([]byte{0x01, 0x11})},
			},
		},
		{
			Logs: []*types.Log{
				{Address: common.BytesToAddress([]byte{0x22})},
				{Address: common.BytesToAddress([]byte{0x02, 0x22})},
			},
		},
		{
			Logs: []*types.Log{
				{Address: common.BytesToAddress([]byte{0x33})},
				{Address: common.BytesToAddress([]byte{0x03, 0x33})},
			},
		},
	}

	number := big.NewInt(1)
	hash := common.BytesToHash([]byte{0x03, 0x14})
	types.Receipts(receipts).DeriveFields(params.TestChainConfig, hash, number.Uint64(), 12, big.NewInt(0), big.NewInt(0), txs)

	logIndex := uint(0)
	for i := range receipts {
		for j := range receipts[i].Logs {
			if receipts[i].Logs[j].BlockNumber != number.Uint64() {
				t.Errorf("receipts[%d].Logs[%d].BlockNumber = %d, want %d", i, j, receipts[i].Logs[j].BlockNumber, number.Uint64())
			}
			if receipts[i].Logs[j].BlockHash != hash {
				t.Errorf("receipts[%d].Logs[%d].BlockHash = %s, want %s", i, j, receipts[i].Logs[j].BlockHash.String(), hash.String())
			}
			if receipts[i].Logs[j].BlockTimestamp != 12 {
				t.Errorf("receipts[%d].Logs[%d].BlockTimestamp = %d, want %d", i, j, receipts[i].Logs[j].BlockTimestamp, 12)
			}
			if receipts[i].Logs[j].TxHash != txs[i].Hash() {
				t.Errorf("receipts[%d].Logs[%d].TxHash = %s, want %s", i, j, receipts[i].Logs[j].TxHash.String(), txs[i].Hash().String())
			}
			if receipts[i].Logs[j].TxIndex != uint(i) {
				t.Errorf("receipts[%d].Logs[%d].TransactionIndex = %d, want %d", i, j, receipts[i].Logs[j].TxIndex, i)
			}
			if receipts[i].Logs[j].Index != logIndex {
				t.Errorf("receipts[%d].Logs[%d].Index = %d, want %d", i, j, receipts[i].Logs[j].Index, logIndex)
			}
			logIndex++
		}
	}
}

func BenchmarkDecodeRLPLogs(b *testing.B) {
	buf, err := os.ReadFile("testdata/stored_receipts.bin")
	if err != nil {
		b.Fatal(err)
	}
	b.Run("ReceiptForStorage", func(b *testing.B) {
		b.ReportAllocs()
		var r []*types.ReceiptForStorage
		for i := 0; i < b.N; i++ {
			if err := rlp.DecodeBytes(buf, &r); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("rlpLogs", func(b *testing.B) {
		b.ReportAllocs()
		var r []*receiptLogs
		for i := 0; i < b.N; i++ {
			if err := rlp.DecodeBytes(buf, &r); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestHeadersRLPStorage(t *testing.T) {
	frdir := t.TempDir()

	db, err := Open(NewMemoryDatabase(), OpenOptions{Ancient: frdir})
	if err != nil {
		t.Fatalf("failed to create database with ancient backend")
	}
	defer db.Close()

	var chain []*types.Block
	var pHash common.Hash
	for i := 0; i < 100; i++ {
		block := types.NewBlockWithHeader(&types.Header{
			Number:      big.NewInt(int64(i)),
			Extra:       []byte("test block"),
			UncleHash:   types.EmptyUncleHash,
			TxHash:      types.EmptyTxsHash,
			ReceiptHash: types.EmptyReceiptsHash,
			ParentHash:  pHash,
		})
		chain = append(chain, block)
		pHash = block.Hash()
	}
	receipts := make([]types.Receipts, 100)
	WriteAncientBlocks(db, chain[:50], types.EncodeBlockReceiptLists(receipts[:50]))
	for i := 50; i < 100; i++ {
		WriteCanonicalHash(db, chain[i].Hash(), chain[i].NumberU64())
		WriteBlock(db, chain[i])
	}
	checkSequence := func(from, amount int) {
		headersRlp := ReadHeaderRange(db, uint64(from), uint64(amount))
		if have, want := len(headersRlp), amount; have != want {
			t.Fatalf("have %d headers, want %d", have, want)
		}
		for i, headerRlp := range headersRlp {
			var header types.Header
			if err := rlp.DecodeBytes(headerRlp, &header); err != nil {
				t.Fatal(err)
			}
			if have, want := header.Number.Uint64(), uint64(from-i); have != want {
				t.Fatalf("wrong number, have %d want %d", have, want)
			}
		}
	}
	checkSequence(99, 20)
	checkSequence(99, 50)
	checkSequence(99, 51)
	checkSequence(99, 52)
	checkSequence(50, 2)
	checkSequence(49, 1)
	checkSequence(49, 50)
	checkSequence(99, 100)
	checkSequence(0, 1)
	checkSequence(1, 1)
	checkSequence(1, 2)
}

func makeTestBAL(t *testing.T) (rlp.RawValue, *bal.BlockAccessList) {
	t.Helper()

	cb := bal.NewConstructionBlockAccessList()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	cb.AccountRead(addr)
	cb.StorageRead(addr, common.BytesToHash([]byte{0x01}))
	cb.StorageWrite(0, addr, common.BytesToHash([]byte{0x02}), common.BytesToHash([]byte{0xaa}))
	cb.BalanceChange(0, addr, uint256.NewInt(100))
	cb.NonceChange(addr, 0, 1)

	var buf bytes.Buffer
	if err := cb.EncodeRLP(&buf); err != nil {
		t.Fatalf("failed to encode BAL: %v", err)
	}
	encoded := buf.Bytes()

	var decoded bal.BlockAccessList
	if err := rlp.DecodeBytes(encoded, &decoded); err != nil {
		t.Fatalf("failed to decode BAL: %v", err)
	}
	return encoded, &decoded
}

// TestBALStorage tests write/read/delete of BALs in the KV store.
func TestBALStorage(t *testing.T) {
	db := NewMemoryDatabase()

	hash := common.BytesToHash([]byte{0x03, 0x14})
	number := uint64(42)

	if HasAccessList(db, hash, number) {
		t.Fatal("BAL found in new database")
	}
	if b := ReadAccessList(db, hash, number); b != nil {
		t.Fatalf("non existent BAL returned: %v", b)
	}

	encoded, testBAL := makeTestBAL(t)
	WriteAccessList(db, hash, number, testBAL)

	if !HasAccessList(db, hash, number) {
		t.Fatal("HasAccessList returned false after write")
	}
	if blob := ReadAccessListRLP(db, hash, number); len(blob) == 0 {
		t.Fatal("ReadAccessListRLP returned empty after write")
	}
	if b := ReadAccessList(db, hash, number); b == nil {
		t.Fatal("ReadAccessList returned nil after write")
	} else if b.Hash() != testBAL.Hash() {
		t.Fatalf("BAL hash mismatch: got %x, want %x", b.Hash(), testBAL.Hash())
	}

	hash2 := common.BytesToHash([]byte{0x03, 0x15})
	WriteAccessListRLP(db, hash2, number, encoded)
	if b := ReadAccessList(db, hash2, number); b == nil {
		t.Fatal("ReadAccessList returned nil after WriteAccessListRLP")
	} else if b.Hash() != testBAL.Hash() {
		t.Fatalf("BAL hash mismatch after WriteAccessListRLP: got %x, want %x", b.Hash(), testBAL.Hash())
	}

	DeleteAccessList(db, hash, number)

	if HasAccessList(db, hash, number) {
		t.Fatal("HasAccessList returned true after delete")
	}
	if b := ReadAccessList(db, hash, number); b != nil {
		t.Fatalf("deleted BAL returned: %v", b)
	}
}
