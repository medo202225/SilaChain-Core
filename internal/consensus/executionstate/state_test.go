package executionstate

import "testing"

func TestStateImportAndPendingTx(t *testing.T) {
	s := NewState("0xgenesis")

	if s.HeadNumber() != 0 {
		t.Fatalf("unexpected genesis head number: %d", s.HeadNumber())
	}
	if s.HeadHash() != "0xgenesis" {
		t.Fatalf("unexpected genesis head hash: %s", s.HeadHash())
	}

	ok := s.AddPendingTx(PendingTx{
		Hash:  "0xtx1",
		From:  "a",
		To:    "b",
		Value: 10,
		Nonce: 1,
		Fee:   1,
	})
	if !ok {
		t.Fatal("expected pending tx insertion")
	}
	if !s.HasPendingTx("0xtx1") {
		t.Fatal("expected tx in pending pool")
	}
	if s.PendingCount() != 1 {
		t.Fatalf("unexpected pending count: %d", s.PendingCount())
	}

	err := s.ImportBlock(ImportedBlock{
		Number:     1,
		Hash:       "0xblock1",
		ParentHash: "0xgenesis",
		Timestamp:  1,
		TxHashes:   []string{"0xtx1"},
	})
	if err != nil {
		t.Fatalf("ImportBlock failed: %v", err)
	}

	if s.HeadNumber() != 1 {
		t.Fatalf("unexpected head number after import: %d", s.HeadNumber())
	}
	if s.HeadHash() != "0xblock1" {
		t.Fatalf("unexpected head hash after import: %s", s.HeadHash())
	}
}

func TestValidateBlockRejectsBadParent(t *testing.T) {
	err := ValidateBlock("0xgenesis", 0, ImportedBlock{
		Number:     1,
		Hash:       "0xblock1",
		ParentHash: "0xwrong",
		Timestamp:  1,
		TxHashes:   []string{"0xtx1"},
	})
	if err == nil {
		t.Fatal("expected parent mismatch error")
	}
}

func TestValidateBlockRejectsBadNumber(t *testing.T) {
	err := ValidateBlock("0xgenesis", 0, ImportedBlock{
		Number:     3,
		Hash:       "0xblock3",
		ParentHash: "0xgenesis",
		Timestamp:  1,
		TxHashes:   []string{"0xtx1"},
	})
	if err == nil {
		t.Fatal("expected non-sequential number error")
	}
}

func TestValidateBlockRejectsEmptyTxHash(t *testing.T) {
	err := ValidateBlock("0xgenesis", 0, ImportedBlock{
		Number:     1,
		Hash:       "0xblock1",
		ParentHash: "0xgenesis",
		Timestamp:  1,
		TxHashes:   []string{""},
	})
	if err == nil {
		t.Fatal("expected empty tx hash error")
	}
}

func TestApplyTransactionStateTransition(t *testing.T) {
	s := NewState("0xgenesis")
	s.SetBalance("alice", 1000000)

	tx := PendingTx{
		Hash:  "0xtx1",
		From:  "alice",
		To:    "bob",
		Value: 25,
		Nonce: 0,
		Fee:   1,
	}

	if ok := s.AddPendingTx(tx); !ok {
		t.Fatal("expected pending tx insertion")
	}

	if err := s.ApplyTransaction(tx); err != nil {
		t.Fatalf("ApplyTransaction failed: %v", err)
	}

	expectedGas := IntrinsicGas(tx)
	expectedFee := expectedGas * 1
	expectedAlice := uint64(1000000) - 25 - expectedFee

	if s.GetBalance("alice") != expectedAlice {
		t.Fatalf("unexpected alice balance: %d", s.GetBalance("alice"))
	}
	if s.GetBalance("bob") != 25 {
		t.Fatalf("unexpected bob balance: %d", s.GetBalance("bob"))
	}
	if s.GetNonce("alice") != 1 {
		t.Fatalf("unexpected alice nonce: %d", s.GetNonce("alice"))
	}
	if s.HasPendingTx("0xtx1") {
		t.Fatal("expected tx removed from pending pool")
	}
}

func TestApplyTransactionRejectsBadNonce(t *testing.T) {
	s := NewState("0xgenesis")
	s.SetBalance("alice", 1000000)

	tx := PendingTx{
		Hash:  "0xtx1",
		From:  "alice",
		To:    "bob",
		Value: 25,
		Nonce: 1,
		Fee:   1,
	}

	if err := s.ApplyTransaction(tx); err == nil {
		t.Fatal("expected bad nonce rejection")
	}
}

func TestApplyTransactionRejectsInsufficientBalance(t *testing.T) {
	s := NewState("0xgenesis")
	s.SetBalance("alice", 10)

	tx := PendingTx{
		Hash:  "0xtx1",
		From:  "alice",
		To:    "bob",
		Value: 25,
		Nonce: 0,
		Fee:   1,
	}

	if err := s.ApplyTransaction(tx); err == nil {
		t.Fatal("expected insufficient balance rejection")
	}
}

func TestExecuteBlockProducesReceiptsAndGasUsage(t *testing.T) {
	s := NewState("0xgenesis")
	s.SetBalance("alice", 1000000)

	tx := PendingTx{
		Hash:  "0xtx1",
		From:  "alice",
		To:    "bob",
		Value: 25,
		Nonce: 0,
		Fee:   1,
		Data:  "abcd",
	}
	_ = s.AddPendingTx(tx)

	block := ImportedBlock{
		Number:     1,
		Hash:       "0xblock1",
		ParentHash: "0xgenesis",
		Timestamp:  1,
		TxHashes:   []string{"0xtx1"},
	}

	result, err := s.ExecuteBlock(BlockExecutionRequest{
		Block: block,
		Txs:   []PendingTx{tx},
	})
	if err != nil {
		t.Fatalf("ExecuteBlock failed: %v", err)
	}

	if result.BlockNumber != 1 {
		t.Fatalf("unexpected block number: %d", result.BlockNumber)
	}
	if len(result.Receipts) != 1 {
		t.Fatalf("unexpected receipt count: %d", len(result.Receipts))
	}
	if result.GasUsed != IntrinsicGas(tx) {
		t.Fatalf("unexpected gas used: %d", result.GasUsed)
	}

	receipt, ok := s.GetReceipt("0xtx1")
	if !ok {
		t.Fatal("expected stored receipt")
	}
	if !receipt.Success {
		t.Fatal("expected successful receipt")
	}
	if receipt.BlockHash != "0xblock1" {
		t.Fatalf("unexpected receipt block hash: %s", receipt.BlockHash)
	}
	if s.LastBlockGasUsed() != result.GasUsed {
		t.Fatalf("unexpected last block gas: %d", s.LastBlockGasUsed())
	}
}
