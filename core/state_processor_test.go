package core

import (
	"fmt"
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/txpool"
	"silachain/internal/execution/executionstate"
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

func (s *testState) ExecuteBlock(req executionstate.BlockExecutionRequest) (executionstate.BlockExecutionResult, error) {
	execState := executionstate.NewState("0xgenesis")

	for i := uint64(1); i <= req.Block.Number-1; i++ {
		hash := fmt.Sprintf("0xseed-block-%d", i)
		if i == req.Block.Number-1 {
			hash = req.Block.ParentHash
		}
		parentHash := "0xgenesis"
		if i > 1 {
			parentHash = fmt.Sprintf("0xseed-block-%d", i-1)
		}
		if err := execState.ImportBlock(executionstate.ImportedBlock{
			Number:     i,
			Hash:       hash,
			ParentHash: parentHash,
			Timestamp:  i,
			TxHashes:   nil,
		}); err != nil {
			return executionstate.BlockExecutionResult{}, err
		}
	}

	for sender, nonce := range s.nonces {
		execState.SetBalance(sender, 1000000000)
		for i := uint64(0); i < nonce; i++ {
			seedHash := fmt.Sprintf("seed-%s-%d", sender, i)
			_ = execState.AddPendingTx(executionstate.PendingTx{
				Hash:  seedHash,
				From:  sender,
				To:    "SILA_BLOCK_FEE_SINK",
				Value: 0,
				Nonce: i,
				Fee:   1,
			})
			_ = execState.ApplyTransaction(executionstate.PendingTx{
				Hash:  seedHash,
				From:  sender,
				To:    "SILA_BLOCK_FEE_SINK",
				Value: 0,
				Nonce: i,
				Fee:   1,
			})
		}
	}

	result, err := execState.ExecuteBlock(req)
	if err != nil {
		return executionstate.BlockExecutionResult{}, err
	}

	s.head = blockassembly.Head{
		Number:    result.BlockNumber,
		Hash:      result.BlockHash,
		StateRoot: result.StateRoot,
		BaseFee:   s.head.BaseFee,
	}
	for _, tx := range req.Txs {
		s.nonces[tx.From] = tx.Nonce + 1
	}
	return result, nil
}

func TestExecute_AppliesAssembledPayloadAndAdvancesHead(t *testing.T) {
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

	executor, err := NewStateProcessor(state, assembler)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := executor.Process(blockassembly.PayloadAttributes{
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
	if result.ExecutionStateRoot == "" {
		t.Fatalf("expected non-empty execution state root")
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

func TestExecute_FailsOnSenderNonceMismatchBetweenPoolAndState(t *testing.T) {
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

	executor, err := NewStateProcessor(state, assembler)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = executor.Process(blockassembly.PayloadAttributes{
		Timestamp:    123,
		FeeRecipient: "SILA_fee_recipient_exec_mismatch",
		Random:       "SILA_rand_exec_mismatch",
	})
	if err == nil {
		t.Fatalf("expected nonce mismatch error")
	}
}
