package miner

import (
	"context"
	"testing"
	"time"
)

func TestPrepareWork(t *testing.T) {
	m := New(nil, Config{
		PendingFeeRecipient: "SILA_fee_recipient_worker",
		GasCeil:             30_000_000,
		Recommit:            2 * time.Second,
	})

	built, stateRoot, err := m.prepareWork(&generateParams{
		timestamp:  123,
		parentHash: "0xparent-worker",
		coinbase:   "SILA_fee_recipient_worker",
		random:     "SILA_random_worker",
		gasLimit:   30_000_000,
		noTxs:      false,
	})
	if err != nil {
		t.Fatalf("prepare work: %v", err)
	}

	if built.ParentHash != "0xparent-worker" {
		t.Fatalf("unexpected parent hash: got=%s want=%s", built.ParentHash, "0xparent-worker")
	}
	if built.BlockNumber != 1 {
		t.Fatalf("unexpected block number: got=%d want=1", built.BlockNumber)
	}
	if stateRoot == "" {
		t.Fatalf("expected non-empty state root")
	}
}

func TestGenerateWork(t *testing.T) {
	m := New(nil, Config{
		PendingFeeRecipient: "SILA_fee_recipient_worker",
		GasCeil:             30_000_000,
		Recommit:            2 * time.Second,
	})

	result := m.generateWork(context.Background(), &generateParams{
		timestamp:  456,
		parentHash: "0xparent-generate",
		coinbase:   "SILA_fee_recipient_worker",
		random:     "SILA_random_generate",
		gasLimit:   30_000_000,
		noTxs:      false,
	})
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.err != nil {
		t.Fatalf("generate work error: %v", result.err)
	}
	if result.result.ParentHash != "0xparent-generate" {
		t.Fatalf("unexpected parent hash: got=%s want=%s", result.result.ParentHash, "0xparent-generate")
	}
	if result.result.StateRoot == "" {
		t.Fatalf("expected non-empty state root")
	}
}

func TestFillTransactions(t *testing.T) {
	m := New(nil, Config{
		PendingFeeRecipient: "SILA_fee_recipient_worker",
		GasCeil:             30_000_000,
		Recommit:            2 * time.Second,
	})

	result, err := m.fillTransactions(context.Background(), nil, &generateParams{
		timestamp:  789,
		parentHash: "0xparent-fill",
		coinbase:   "SILA_fee_recipient_worker",
		random:     "SILA_random_fill",
		gasLimit:   30_000_000,
		noTxs:      false,
	})
	if err != nil {
		t.Fatalf("fill transactions: %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.result.BlockHash == "" {
		t.Fatalf("expected non-empty block hash")
	}
	if result.result.TxCount != 0 {
		t.Fatalf("unexpected tx count: got=%d want=0", result.result.TxCount)
	}
}

func TestFillTransactionsInterrupted(t *testing.T) {
	m := New(nil, Config{
		PendingFeeRecipient: "SILA_fee_recipient_worker",
		GasCeil:             30_000_000,
		Recommit:            2 * time.Second,
	})

	interrupt := make(chan struct{})
	close(interrupt)

	_, err := m.fillTransactions(context.Background(), interrupt, &generateParams{
		timestamp:  999,
		parentHash: "0xparent-interrupt",
		coinbase:   "SILA_fee_recipient_worker",
		random:     "SILA_random_interrupt",
		gasLimit:   30_000_000,
		noTxs:      true,
	})
	if err == nil {
		t.Fatalf("expected interrupt error")
	}
}
