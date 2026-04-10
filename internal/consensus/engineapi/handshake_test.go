package engineapi

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engine"
	"silachain/internal/consensus/txpool"
)

type handshakeState struct {
	head   blockassembly.Head
	nonces map[string]uint64
}

func newHandshakeState(head blockassembly.Head) *handshakeState {
	return &handshakeState{
		head:   head,
		nonces: make(map[string]uint64),
	}
}

func (s *handshakeState) Head() blockassembly.Head {
	return s.head
}

func (s *handshakeState) SetHead(head blockassembly.Head) error {
	s.head = head
	return nil
}

func (s *handshakeState) SetSenderNonce(sender string, nonce uint64) error {
	s.nonces[sender] = nonce
	return nil
}

func (s *handshakeState) SenderNonce(sender string) uint64 {
	return s.nonces[sender]
}

func TestLocalPayloadHandshake_BuildGetSubmitAndAdvanceCanonicalHead(t *testing.T) {
	head := blockassembly.Head{
		Number:    50,
		Hash:      "0xhead50",
		StateRoot: "0xstate50",
		BaseFee:   10,
	}

	state := newHandshakeState(head)
	state.SetSenderNonce("alice", 0)
	state.SetSenderNonce("bob", 0)

	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set alice state nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", 0); err != nil {
		t.Fatalf("set bob state nonce: %v", err)
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

	buildResult, err := svc.ForkchoiceUpdatedWithAttributes(ForkchoiceState{
		HeadBlockHash:      "0xhead50",
		SafeBlockHash:      "0xhead50",
		FinalizedBlockHash: "0xhead50",
	}, &blockassembly.PayloadAttributes{
		Timestamp:         5001,
		FeeRecipient:      "SILA_fee_recipient_handshake",
		Random:            "SILA_random_handshake",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("forkchoice updated with attrs: %v", err)
	}

	if buildResult.PayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected build payload status: got=%s want=%s", buildResult.PayloadStatus.Status, PayloadStatusValid)
	}
	if buildResult.PayloadID == "" {
		t.Fatalf("expected non-empty payload id")
	}

	payload, err := svc.GetPayload(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get payload: %v", err)
	}

	if payload.BlockNumber != 51 {
		t.Fatalf("unexpected payload block number: got=%d want=51", payload.BlockNumber)
	}
	if payload.ParentHash != "0xhead50" {
		t.Fatalf("unexpected payload parent hash: got=%s want=0xhead50", payload.ParentHash)
	}
	if payload.TxCount != 2 {
		t.Fatalf("unexpected payload tx count: got=%d want=2", payload.TxCount)
	}

	newPayloadStatus, err := svc.NewPayload(PayloadEnvelope{
		BlockNumber: payload.BlockNumber,
		BlockHash:   payload.BlockHash,
		ParentHash:  payload.ParentHash,
		StateRoot:   payload.StateRoot,
	})
	if err != nil {
		t.Fatalf("new payload: %v", err)
	}

	if newPayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected newPayload status: got=%s want=%s", newPayloadStatus.Status, PayloadStatusValid)
	}
	if newPayloadStatus.LatestValidHash != payload.BlockHash {
		t.Fatalf("unexpected latest valid hash: got=%s want=%s", newPayloadStatus.LatestValidHash, payload.BlockHash)
	}

	fcResult, err := svc.ForkchoiceUpdated(ForkchoiceState{
		HeadBlockHash:      payload.BlockHash,
		SafeBlockHash:      payload.BlockHash,
		FinalizedBlockHash: "0xhead50",
	})
	if err != nil {
		t.Fatalf("forkchoice updated final step: %v", err)
	}

	if fcResult.PayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected final forkchoice status: got=%s want=%s", fcResult.PayloadStatus.Status, PayloadStatusValid)
	}
	if fcResult.CanonicalHead.Hash != payload.BlockHash {
		t.Fatalf("unexpected canonical head hash: got=%s want=%s", fcResult.CanonicalHead.Hash, payload.BlockHash)
	}
	if fcResult.CanonicalHead.Number != 51 {
		t.Fatalf("unexpected canonical head number: got=%d want=51", fcResult.CanonicalHead.Number)
	}
}

func TestLocalPayloadHandshake_ReturnsSyncingForUnknownSubmittedParent(t *testing.T) {
	storeHead := blockassembly.Head{
		Number:    3,
		Hash:      "0xhead3",
		StateRoot: "0xstate3",
		BaseFee:   5,
	}

	state := newHandshakeState(storeHead)
	pool := txpool.NewPool(5)

	eng, err := engine.New(state, pool, 30000000)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	svc, err := NewBuilderServiceFromEngine(eng)
	if err != nil {
		t.Fatalf("new builder service from engine: %v", err)
	}

	status, err := svc.NewPayload(PayloadEnvelope{
		BlockNumber: 4,
		BlockHash:   "0xorphan4",
		ParentHash:  "0xmissing-parent",
		StateRoot:   "0xorphan-state",
	})
	if err != nil {
		t.Fatalf("new payload orphan: %v", err)
	}

	if status.Status != PayloadStatusSyncing {
		t.Fatalf("unexpected orphan payload status: got=%s want=%s", status.Status, PayloadStatusSyncing)
	}
}
