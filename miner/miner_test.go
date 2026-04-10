package miner

import (
	"context"
	"testing"
	"time"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/txpool"
)

func TestNew_UsesDefaultConfig(t *testing.T) {
	m := New(nil, Config{})
	cfg := m.Config()

	if cfg.GasCeil != DefaultConfig.GasCeil {
		t.Fatalf("unexpected gas ceil: got=%d want=%d", cfg.GasCeil, DefaultConfig.GasCeil)
	}
	if cfg.Recommit != DefaultConfig.Recommit {
		t.Fatalf("unexpected recommit: got=%s want=%s", cfg.Recommit, DefaultConfig.Recommit)
	}
}

func TestMiner_Setters(t *testing.T) {
	m := New(nil, Config{})

	m.SetPendingFeeRecipient("SILA_fee_recipient_miner")
	m.SetExtra([]byte("sila-extra"))
	m.SetGasCeil(31_000_000)

	cfg := m.Config()

	if cfg.PendingFeeRecipient != "SILA_fee_recipient_miner" {
		t.Fatalf("unexpected recipient: got=%s want=%s", cfg.PendingFeeRecipient, "SILA_fee_recipient_miner")
	}
	if string(cfg.ExtraData) != "sila-extra" {
		t.Fatalf("unexpected extra data: got=%s want=%s", string(cfg.ExtraData), "sila-extra")
	}
	if cfg.GasCeil != 31_000_000 {
		t.Fatalf("unexpected gas ceil: got=%d want=%d", cfg.GasCeil, 31_000_000)
	}
}

func TestMiner_BuildPayload(t *testing.T) {
	m := New(nil, Config{
		PendingFeeRecipient: "SILA_fee_recipient_miner",
		GasCeil:             30_000_000,
		Recommit:            2 * time.Second,
	})

	args := BuildPayloadArgs{
		ParentHash:   "0xparent11",
		Timestamp:    1111,
		FeeRecipient: "SILA_fee_recipient_miner",
		Random:       "SILA_random_miner",
		GasLimit:     30_000_000,
		Version:      1,
	}

	built := blockassembly.Result{
		ParentNumber:    11,
		BlockNumber:     12,
		ParentHash:      "0xparent11",
		ParentStateRoot: "0xstate11",
		BaseFee:         10,
		GasLimit:        30_000_000,
		Selection: blockassembly.TransactionSelection{
			Transactions: []txpool.Tx{
				{Hash: "tx1", From: "alice", Nonce: 0, GasLimit: 21000},
			},
			GasUsed: 21000,
		},
	}

	payload, err := m.BuildPayload(context.Background(), args, built, "0xstate12")
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	resolved := payload.Resolve()

	if resolved.BlockNumber != 12 {
		t.Fatalf("unexpected block number: got=%d want=12", resolved.BlockNumber)
	}
	if resolved.ParentHash != "0xparent11" {
		t.Fatalf("unexpected parent hash: got=%s want=%s", resolved.ParentHash, "0xparent11")
	}
	if resolved.StateRoot != "0xstate12" {
		t.Fatalf("unexpected state root: got=%s want=%s", resolved.StateRoot, "0xstate12")
	}
	if resolved.TxCount != 1 {
		t.Fatalf("unexpected tx count: got=%d want=1", resolved.TxCount)
	}
}
