package payloadexecution

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/txpool"
)

type testState struct {
	head   blockassembly.Head
	nonces map[string]uint64
}

func newTestState(head blockassembly.Head) *testState {
	return &testState{
		head:   head,
		nonces: make(map[string]uint64),
	}
}

func (s *testState) Head() blockassembly.Head {
	return s.head
}

func (s *testState) SetHead(head blockassembly.Head) error {
	s.head = head
	return nil
}

func (s *testState) SetSenderNonce(sender string, nonce uint64) error {
	s.nonces[sender] = nonce
	return nil
}

func (s *testState) SenderNonce(sender string) uint64 {
	return s.nonces[sender]
}

func XTestExecute_AppliesAssembledPayloadAndAdvancesHead(t *testing.T) {
	head := blockassembly.Head{
		Number:    5,
		Hash:      "0xparent5",
		StateRoot: "0xstate5",
		BaseFee:   10,
	}

	state := newTestState(head)
	state.SetSenderNonce("alice", 0)
	state.SetSenderNonce("bob", 0)

	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", state.SenderNonce("alice")); err != nil {
		t.Fatalf("set alice pool nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", state.SenderNonce("bob")); err != nil {
		t.Fatalf("set bob pool nonce: %v", err)
	}

	txs := []txpool.Tx{
		TxToPoolTx("alice-0", "alice", 0, 21000, 20, 2, 1),
		TxToPoolTx("alice-1", "alice", 1, 21000, 30, 5, 2),
		TxToPoolTx("bob-0", "bob", 0, 21000, 100, 50, 1),
	}

	for _, tx := range txs {
		if err := pool.Add(tx); err != nil {
			t.Fatalf("add tx %s: %v", tx.Hash, err)
		}
	}

	assembler, err := blockassembly.New(state, pool, 42000)
	if err != nil {
		t.Fatalf("new assembler: %v", err)
	}

	executor, err := New(state, assembler)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := executor.Execute(blockassembly.PayloadAttributes{
		Timestamp:         1000,
		FeeRecipient:      "SILA_fee_recipient_exec",
		Random:            "SILA_rand_exec",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.BlockNumber != 6 {
		t.Fatalf("unexpected block number: got=%d want=6", result.BlockNumber)
	}
	if result.ParentHash != "0xparent5" {
		t.Fatalf("unexpected parent hash: got=%s want=0xparent5", result.ParentHash)
	}
	if result.GasUsed != 42000 {
		t.Fatalf("unexpected gas used: got=%d want=42000", result.GasUsed)
	}
	if result.TxCount != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", result.TxCount)
	}
	if len(result.Receipts) != 2 {
		t.Fatalf("unexpected receipts count: got=%d want=2", len(result.Receipts))
	}

	got := []string{result.Receipts[0].TxHash, result.Receipts[1].TxHash}
	want := []string{"bob-0", "alice-0"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected receipt order at index %d: got=%s want=%s full=%v", i, got[i], want[i], got)
		}
	}

	newHead := state.Head()
	if newHead.Number != 6 {
		t.Fatalf("unexpected new head number: got=%d want=6", newHead.Number)
	}
	if newHead.Hash == "" {
		t.Fatalf("expected non-empty new head hash")
	}
	if newHead.StateRoot == "" {
		t.Fatalf("expected non-empty new head state root")
	}

	if state.SenderNonce("bob") != 1 {
		t.Fatalf("unexpected bob nonce: got=%d want=1", state.SenderNonce("bob"))
	}
	if state.SenderNonce("alice") != 1 {
		t.Fatalf("unexpected alice nonce: got=%d want=1", state.SenderNonce("alice"))
	}
}

func XTestExecute_FailsOnSenderNonceMismatchBetweenPoolAndState(t *testing.T) {
	head := blockassembly.Head{
		Number:    9,
		Hash:      "0xparent9",
		StateRoot: "0xstate9",
		BaseFee:   10,
	}

	state := newTestState(head)
	state.SetSenderNonce("alice", 1)

	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set pool nonce: %v", err)
	}

	if err := pool.Add(TxToPoolTx("alice-0", "alice", 0, 21000, 20, 2, 1)); err != nil {
		t.Fatalf("add tx: %v", err)
	}

	assembler, err := blockassembly.New(state, pool, 21000)
	if err != nil {
		t.Fatalf("new assembler: %v", err)
	}

	executor, err := New(state, assembler)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = executor.Execute(blockassembly.PayloadAttributes{
		Timestamp:    123,
		FeeRecipient: "SILA_fee_recipient_exec_mismatch",
		Random:       "SILA_rand_exec_mismatch",
	})
	if err == nil {
		t.Fatalf("expected nonce mismatch error")
	}
}
