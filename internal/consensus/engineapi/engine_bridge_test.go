package engineapi

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engine"
	"silachain/internal/consensus/txpool"
)

type bridgeTestState struct {
	head   blockassembly.Head
	nonces map[string]uint64
}

func newBridgeTestState(head blockassembly.Head) *bridgeTestState {
	return &bridgeTestState{
		head:   head,
		nonces: make(map[string]uint64),
	}
}

func (s *bridgeTestState) Head() blockassembly.Head {
	return s.head
}

func (s *bridgeTestState) SetHead(head blockassembly.Head) error {
	s.head = head
	return nil
}

func (s *bridgeTestState) SetSenderNonce(sender string, nonce uint64) error {
	s.nonces[sender] = nonce
	return nil
}

func (s *bridgeTestState) SenderNonce(sender string) uint64 {
	return s.nonces[sender]
}

func TestNewBuilderServiceFromEngine_UsesRealEngineAssemblerAndForkchoiceStore(t *testing.T) {
	head := blockassembly.Head{
		Number:    25,
		Hash:      "0xhead25",
		StateRoot: "0xstate25",
		BaseFee:   10,
	}

	state := newBridgeTestState(head)
	state.SetSenderNonce("alice", 0)
	state.SetSenderNonce("bob", 0)

	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", state.SenderNonce("alice")); err != nil {
		t.Fatalf("set alice pool nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", state.SenderNonce("bob")); err != nil {
		t.Fatalf("set bob pool nonce: %v", err)
	}

	adds := []txpool.Tx{
		{Hash: "alice-0", From: "alice", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 1},
		{Hash: "bob-0", From: "bob", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 100, MaxPriorityFeePerGas: 50, Timestamp: 1},
	}

	for _, tx := range adds {
		if err := pool.Add(tx); err != nil {
			t.Fatalf("add tx %s: %v", tx.Hash, err)
		}
	}

	eng, err := engine.New(state, pool, 42000)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	svc, err := NewBuilderServiceFromEngine(eng)
	if err != nil {
		t.Fatalf("new builder service from engine: %v", err)
	}

	result, err := svc.ForkchoiceUpdatedWithAttributes(ForkchoiceState{
		HeadBlockHash:      "0xhead25",
		SafeBlockHash:      "0xhead25",
		FinalizedBlockHash: "0xhead25",
	}, &blockassembly.PayloadAttributes{
		Timestamp:         1001,
		FeeRecipient:      "SILA_fee_recipient_bridge",
		Random:            "SILA_random_bridge",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("forkchoice updated with attrs: %v", err)
	}

	if result.PayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected payload status: got=%s want=%s", result.PayloadStatus.Status, PayloadStatusValid)
	}
	if result.PayloadID == "" {
		t.Fatalf("expected non-empty payload id")
	}
	if result.CanonicalHead.Hash != "0xhead25" {
		t.Fatalf("unexpected canonical head hash before payload import: got=%s want=0xhead25", result.CanonicalHead.Hash)
	}

	payload, err := svc.GetPayload(result.PayloadID)
	if err != nil {
		t.Fatalf("get payload: %v", err)
	}

	if payload.BlockNumber != 26 {
		t.Fatalf("unexpected payload block number: got=%d want=26", payload.BlockNumber)
	}
	if payload.ParentHash != "0xhead25" {
		t.Fatalf("unexpected payload parent hash: got=%s want=0xhead25", payload.ParentHash)
	}
	if payload.TxCount != 2 {
		t.Fatalf("unexpected tx count: got=%d want=2", payload.TxCount)
	}
	if payload.GasUsed != 42000 {
		t.Fatalf("unexpected gas used: got=%d want=42000", payload.GasUsed)
	}
}

func TestNewBuilderServiceFromEngine_FailsOnNilEngine(t *testing.T) {
	_, err := NewBuilderServiceFromEngine(nil)
	if err == nil {
		t.Fatalf("expected nil engine bridge error")
	}
}
