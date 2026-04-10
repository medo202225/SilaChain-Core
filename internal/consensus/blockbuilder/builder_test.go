package blockbuilder

import (
	"testing"

	"silachain/internal/consensus/txpool"
)

func TestBuild_SelectsOrderedTransactionsWithinGasLimit(t *testing.T) {
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

	builder, err := New(42000)
	if err != nil {
		t.Fatalf("new builder: %v", err)
	}

	result, err := builder.Build(pool)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(result.Transactions) != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", len(result.Transactions))
	}

	got := []string{result.Transactions[0].Hash, result.Transactions[1].Hash}
	want := []string{"bob-0", "alice-0"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected tx order at index %d: got=%s want=%s full=%v", i, got[i], want[i], got)
		}
	}

	if result.GasUsed != 42000 {
		t.Fatalf("unexpected gas used: got=%d want=42000", result.GasUsed)
	}
}
