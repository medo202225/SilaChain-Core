package engine

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
		hash := fmt.Sprintf("0xseed-engine-block-%d", i)
		if i == req.Block.Number-1 {
			hash = req.Block.ParentHash
		}
		parentHash := "0xgenesis"
		if i > 1 {
			parentHash = fmt.Sprintf("0xseed-engine-block-%d", i-1)
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

func TestProduceBlock_RunsFullConsensusFlow(t *testing.T) {
	head := blockassembly.Head{
		Number:    15,
		Hash:      "0xhead15",
		StateRoot: "0xstate15",
		BaseFee:   10,
	}

	state := newTestState(head)
	state.SetSenderNonce("alice", 0)
	state.SetSenderNonce("bob", 0)

	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", state.SenderNonce("alice")); err != nil {
		t.Fatalf("set alice sender nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", state.SenderNonce("bob")); err != nil {
		t.Fatalf("set bob sender nonce: %v", err)
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

	eng, err := New(state, pool, 42000)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	result, err := eng.ProduceBlock(blockassembly.PayloadAttributes{
		Timestamp:         111111,
		FeeRecipient:      "SILA_fee_recipient_engine",
		Random:            "SILA_random_engine",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("produce block: %v", err)
	}

	if !result.ImportResult.Imported {
		t.Fatalf("expected imported=true")
	}
	if result.ImportResult.BlockNumber != 16 {
		t.Fatalf("unexpected imported block number: got=%d want=16", result.ImportResult.BlockNumber)
	}
	if result.ImportResult.ParentHash != "0xhead15" {
		t.Fatalf("unexpected parent hash: got=%s want=0xhead15", result.ImportResult.ParentHash)
	}
	if result.ImportResult.TxCount != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", result.ImportResult.TxCount)
	}
	if !result.ForkChoiceResult.Accepted {
		t.Fatalf("expected fork choice accepted=true")
	}
	if !result.ForkChoiceResult.CanonicalChanged {
		t.Fatalf("expected canonical head change")
	}
	if result.CanonicalHead.Number != 16 {
		t.Fatalf("unexpected canonical head number: got=%d want=16", result.CanonicalHead.Number)
	}
	if result.CanonicalHead.Hash != result.ImportResult.BlockHash {
		t.Fatalf("canonical head hash mismatch: got=%s want=%s", result.CanonicalHead.Hash, result.ImportResult.BlockHash)
	}

	newStateHead := state.Head()
	if newStateHead.Number != 16 {
		t.Fatalf("unexpected state head number: got=%d want=16", newStateHead.Number)
	}
	if state.SenderNonce("bob") != 1 {
		t.Fatalf("unexpected bob nonce: got=%d want=1", state.SenderNonce("bob"))
	}
	if state.SenderNonce("alice") != 1 {
		t.Fatalf("unexpected alice nonce: got=%d want=1", state.SenderNonce("alice"))
	}
}

func TestProduceBlock_AdvancesAgainOnNextCycle(t *testing.T) {
	head := blockassembly.Head{
		Number:    1,
		Hash:      "0xhead1",
		StateRoot: "0xstate1",
		BaseFee:   5,
	}

	state := newTestState(head)
	state.SetSenderNonce("alice", 0)
	state.SetSenderNonce("bob", 0)

	pool := txpool.NewPool(5)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set alice sender nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", 0); err != nil {
		t.Fatalf("set bob sender nonce: %v", err)
	}

	if err := pool.Add(txpool.Tx{
		Hash:                 "bob-0",
		From:                 "bob",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         50,
		MaxPriorityFeePerGas: 10,
		Timestamp:            1,
	}); err != nil {
		t.Fatalf("add bob-0: %v", err)
	}

	eng, err := New(state, pool, 30000000)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	first, err := eng.ProduceBlock(blockassembly.PayloadAttributes{
		Timestamp:    100,
		FeeRecipient: "SILA_fee_recipient_cycle1",
		Random:       "SILA_random_cycle1",
	})
	if err != nil {
		t.Fatalf("first produce block: %v", err)
	}

	if first.CanonicalHead.Number != 2 {
		t.Fatalf("unexpected canonical head after first block: got=%d want=2", first.CanonicalHead.Number)
	}

	if err := pool.SetSenderStateNonce("alice", state.SenderNonce("alice")); err != nil {
		t.Fatalf("refresh alice sender nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", state.SenderNonce("bob")); err != nil {
		t.Fatalf("refresh bob sender nonce: %v", err)
	}

	if err := pool.Add(txpool.Tx{
		Hash:                 "alice-0",
		From:                 "alice",
		Nonce:                0,
		GasLimit:             21000,
		MaxFeePerGas:         60,
		MaxPriorityFeePerGas: 12,
		Timestamp:            2,
	}); err != nil {
		t.Fatalf("add alice-0: %v", err)
	}

	second, err := eng.ProduceBlock(blockassembly.PayloadAttributes{
		Timestamp:    200,
		FeeRecipient: "SILA_fee_recipient_cycle2",
		Random:       "SILA_random_cycle2",
	})
	if err != nil {
		t.Fatalf("second produce block: %v", err)
	}

	if second.CanonicalHead.Number != 3 {
		t.Fatalf("unexpected canonical head after second block: got=%d want=3", second.CanonicalHead.Number)
	}
	if state.Head().Number != 3 {
		t.Fatalf("unexpected state head after second block: got=%d want=3", state.Head().Number)
	}
	if state.SenderNonce("alice") != 1 {
		t.Fatalf("unexpected alice nonce after second block: got=%d want=1", state.SenderNonce("alice"))
	}
}
