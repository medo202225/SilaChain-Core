package blockassembly

import (
	"testing"

	"silachain/internal/consensus/txpool"
)

type stubStateProvider struct {
	head Head
}

func (s stubStateProvider) Head() Head {
	return s.head
}

func TestAssemble_BuildsNextBlockFromExecutionHead(t *testing.T) {
	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set alice state nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", 0); err != nil {
		t.Fatalf("set bob state nonce: %v", err)
	}

	adds := []txpool.Tx{
		{Hash: "alice-0", From: "alice", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 1},
		{Hash: "alice-1", From: "alice", Nonce: 1, GasLimit: 21000, MaxFeePerGas: 30, MaxPriorityFeePerGas: 5, Timestamp: 2},
		{Hash: "bob-0", From: "bob", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 100, MaxPriorityFeePerGas: 50, Timestamp: 1},
	}

	for _, tx := range adds {
		if err := pool.Add(tx); err != nil {
			t.Fatalf("add tx %s: %v", tx.Hash, err)
		}
	}

	state := stubStateProvider{
		head: Head{
			Number:    7,
			Hash:      "0xabc123",
			StateRoot: "0xdef456",
			BaseFee:   10,
		},
	}

	assembler, err := New(state, pool, 42000)
	if err != nil {
		t.Fatalf("new assembler: %v", err)
	}

	result, err := assembler.Assemble(PayloadAttributes{
		Timestamp:         123456789,
		FeeRecipient:      "SILA_fee_recipient_001",
		Random:            "SILA_random_001",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if result.ParentNumber != 7 {
		t.Fatalf("unexpected parent number: got=%d want=7", result.ParentNumber)
	}
	if result.BlockNumber != 8 {
		t.Fatalf("unexpected block number: got=%d want=8", result.BlockNumber)
	}
	if result.ParentHash != "0xabc123" {
		t.Fatalf("unexpected parent hash: got=%s want=0xabc123", result.ParentHash)
	}
	if result.ParentStateRoot != "0xdef456" {
		t.Fatalf("unexpected parent state root: got=%s want=0xdef456", result.ParentStateRoot)
	}
	if result.BaseFee != 10 {
		t.Fatalf("unexpected base fee: got=%d want=10", result.BaseFee)
	}
	if result.GasLimit != 42000 {
		t.Fatalf("unexpected gas limit: got=%d want=42000", result.GasLimit)
	}
	if result.Selection.GasUsed != 42000 {
		t.Fatalf("unexpected gas used: got=%d want=42000", result.Selection.GasUsed)
	}
	if len(result.Selection.Transactions) != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", len(result.Selection.Transactions))
	}

	got := []string{
		result.Selection.Transactions[0].Hash,
		result.Selection.Transactions[1].Hash,
	}
	want := []string{"bob-0", "alice-0"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected order at index %d: got=%s want=%s full=%v", i, got[i], want[i], got)
		}
	}
}

func TestAssemble_UsesSuggestedGasLimitWhenProvided(t *testing.T) {
	pool := txpool.NewPool(5)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set state nonce: %v", err)
	}
	if err := pool.Add(txpool.Tx{
		Hash:                 "alice-0",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	state := stubStateProvider{
		head: Head{
			Number:    11,
			Hash:      "0xparent",
			StateRoot: "0xstate",
			BaseFee:   5,
		},
	}

	assembler, err := New(state, pool, 30000000)
	if err != nil {
		t.Fatalf("new assembler: %v", err)
	}

	result, err := assembler.Assemble(PayloadAttributes{
		Timestamp:         999,
		FeeRecipient:      "SILA_fee_recipient_002",
		Random:            "SILA_random_002",
		SuggestedGasLimit: 15000000,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if result.BlockNumber != 12 {
		t.Fatalf("unexpected block number: got=%d want=12", result.BlockNumber)
	}
	if result.GasLimit != 15000000 {
		t.Fatalf("unexpected gas limit override: got=%d want=15000000", result.GasLimit)
	}
	if len(result.Selection.Transactions) != 1 {
		t.Fatalf("unexpected tx count: got=%d want=1", len(result.Selection.Transactions))
	}
}
