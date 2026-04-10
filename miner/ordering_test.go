package miner

import (
	"testing"

	"silachain/internal/consensus/txpool"
)

func TestTxByPriceAndTime_HigherFeeFirst(t *testing.T) {
	a := newTxWithMinerFee(txpool.Tx{
		Hash:                 "a",
		From:                 "alice",
		MaxFeePerGas:         20,
		MaxPriorityFeePerGas: 2,
		Timestamp:            2,
	}, 10)

	b := newTxWithMinerFee(txpool.Tx{
		Hash:                 "b",
		From:                 "bob",
		MaxFeePerGas:         30,
		MaxPriorityFeePerGas: 5,
		Timestamp:            1,
	}, 10)

	s := txByPriceAndTime{a, b}
	if !s.Less(1, 0) {
		t.Fatalf("expected tx b to rank before tx a")
	}
}

func TestTxByPriceAndTime_EarlierTimeWinsOnEqualFee(t *testing.T) {
	a := &txWithMinerFee{
		tx:   txpool.Tx{Hash: "a", From: "alice", Timestamp: 2},
		from: "alice",
		fees: 3,
	}
	b := &txWithMinerFee{
		tx:   txpool.Tx{Hash: "b", From: "bob", Timestamp: 1},
		from: "bob",
		fees: 3,
	}

	s := txByPriceAndTime{a, b}
	if !s.Less(1, 0) {
		t.Fatalf("expected earlier tx to rank first on equal fee")
	}
}

func TestTransactionsByPriceAndNonce_PeekShiftPop(t *testing.T) {
	set := newTransactionsByPriceAndNonce(map[string][]txpool.Tx{
		"alice": {
			{Hash: "alice-0", From: "alice", Nonce: 0, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 2},
			{Hash: "alice-1", From: "alice", Nonce: 1, MaxFeePerGas: 25, MaxPriorityFeePerGas: 3, Timestamp: 3},
		},
		"bob": {
			{Hash: "bob-0", From: "bob", Nonce: 0, MaxFeePerGas: 30, MaxPriorityFeePerGas: 5, Timestamp: 1},
		},
	}, 10)

	tx, fee := set.Peek()
	if tx == nil {
		t.Fatalf("expected tx from peek")
	}
	if tx.Hash != "bob-0" {
		t.Fatalf("unexpected first tx: got=%s want=bob-0", tx.Hash)
	}
	if fee == 0 {
		t.Fatalf("expected non-zero fee")
	}

	set.Pop()

	tx, _ = set.Peek()
	if tx == nil || tx.Hash != "alice-0" {
		t.Fatalf("unexpected tx after pop: got=%v want=alice-0", tx)
	}

	set.Shift()

	tx, _ = set.Peek()
	if tx == nil || tx.Hash != "alice-1" {
		t.Fatalf("unexpected tx after shift: got=%v want=alice-1", tx)
	}

	set.Clear()
	if !set.Empty() {
		t.Fatalf("expected empty set after clear")
	}
}
