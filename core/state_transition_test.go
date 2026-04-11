package core

import (
	"testing"

	"silachain/internal/execution/executionstate"
)

func TestStateTransition_ApplyTransaction(t *testing.T) {
	state := executionstate.NewState("0xgenesis")
	state.SetBalance("alice", 1000000000)

	transition := NewStateTransition(state)

	receipt, err := transition.ApplyTransaction(executionstate.PendingTx{
		Hash:     "tx-1",
		From:     "alice",
		To:       "SILA_BLOCK_FEE_SINK",
		Value:    0,
		Nonce:    0,
		Data:     "",
		Fee:      1,
		GasLimit: 21000,
	})
	if err != nil {
		t.Fatalf("apply transaction: %v", err)
	}

	if receipt.TxHash != "tx-1" {
		t.Fatalf("unexpected receipt tx hash: got=%s want=tx-1", receipt.TxHash)
	}
	if !receipt.Success {
		t.Fatalf("expected success receipt")
	}
	if receipt.GasUsed != executionstate.IntrinsicGas(executionstate.PendingTx{
		Hash:     "tx-1",
		From:     "alice",
		To:       "SILA_BLOCK_FEE_SINK",
		Value:    0,
		Nonce:    0,
		Data:     "",
		Fee:      1,
		GasLimit: 21000,
	}) {
		t.Fatalf("unexpected gas used: got=%d", receipt.GasUsed)
	}
	if state.GetNonce("alice") != 1 {
		t.Fatalf("unexpected alice nonce: got=%d want=1", state.GetNonce("alice"))
	}
}
