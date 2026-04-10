package miner

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/txpool"
)

func TestBuildPayloadArgs_ID_Deterministic(t *testing.T) {
	args := BuildPayloadArgs{
		ParentHash:   "0xparent1",
		Timestamp:    123456,
		FeeRecipient: "SILA_fee_recipient_001",
		Random:       "SILA_random_001",
		GasLimit:     30000000,
		Version:      1,
	}

	id1 := args.ID()
	id2 := args.ID()

	if id1 != id2 {
		t.Fatalf("expected deterministic payload id: %s != %s", id1.String(), id2.String())
	}
	if id1.String() == "" {
		t.Fatalf("expected non-empty payload id string")
	}
}

func TestNewPayload_Resolve(t *testing.T) {
	args := BuildPayloadArgs{
		ParentHash:   "0xparent7",
		Timestamp:    999,
		FeeRecipient: "SILA_fee_recipient_payload",
		Random:       "SILA_random_payload",
		GasLimit:     30000000,
		Version:      1,
	}

	built := blockassembly.Result{
		ParentNumber:    7,
		BlockNumber:     8,
		ParentHash:      "0xparent7",
		ParentStateRoot: "0xstate7",
		BaseFee:         10,
		GasLimit:        30000000,
		Selection: blockassembly.TransactionSelection{
			Transactions: []txpool.Tx{
				{Hash: "tx1", From: "alice", Nonce: 0, GasLimit: 21000},
				{Hash: "tx2", From: "bob", Nonce: 0, GasLimit: 21000},
			},
			GasUsed: 42000,
		},
	}

	payload, err := NewPayload(args, built, "0xstate8")
	if err != nil {
		t.Fatalf("new payload: %v", err)
	}

	resolved := payload.Resolve()

	if resolved.PayloadID == "" {
		t.Fatalf("expected non-empty payload id")
	}
	if resolved.BlockNumber != 8 {
		t.Fatalf("unexpected block number: got=%d want=8", resolved.BlockNumber)
	}
	if resolved.ParentHash != "0xparent7" {
		t.Fatalf("unexpected parent hash: got=%s want=0xparent7", resolved.ParentHash)
	}
	if resolved.ParentStateRoot != "0xstate7" {
		t.Fatalf("unexpected parent state root: got=%s want=0xstate7", resolved.ParentStateRoot)
	}
	if resolved.StateRoot != "0xstate8" {
		t.Fatalf("unexpected state root: got=%s want=0xstate8", resolved.StateRoot)
	}
	if resolved.GasUsed != 42000 {
		t.Fatalf("unexpected gas used: got=%d want=42000", resolved.GasUsed)
	}
	if resolved.TxCount != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", resolved.TxCount)
	}
}
