package engineapi

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engine"
	"silachain/internal/consensus/txpool"
	"silachain/internal/execution/executionstate"
)

type metadataState struct {
	head   blockassembly.Head
	nonces map[string]uint64
}

func newMetadataState(head blockassembly.Head) *metadataState {
	return &metadataState{
		head:   head,
		nonces: make(map[string]uint64),
	}
}

func (s *metadataState) Head() blockassembly.Head {
	return s.head
}

func (s *metadataState) SetHead(head blockassembly.Head) error {
	s.head = head
	return nil
}

func (s *metadataState) SetSenderNonce(sender string, nonce uint64) error {
	s.nonces[sender] = nonce
	return nil
}

func (s *metadataState) SenderNonce(sender string) uint64 {
	return s.nonces[sender]
}
func (s *metadataState) ExecuteBlock(req executionstate.BlockExecutionRequest) (executionstate.BlockExecutionResult, error) {
	receipts := make([]executionstate.Receipt, 0, len(req.Txs))
	var gasUsed uint64

	for _, tx := range req.Txs {
		intrinsicGas := executionstate.IntrinsicGas(tx)
		gasUsed += intrinsicGas

		s.nonces[tx.From] = tx.Nonce + 1

		receipts = append(receipts, executionstate.Receipt{
			TxHash:          tx.Hash,
			BlockNumber:     req.Block.Number,
			BlockHash:       req.Block.Hash,
			From:            tx.From,
			To:              tx.To,
			GasUsed:         intrinsicGas,
			EffectiveGasFee: intrinsicGas * tx.Fee,
			Success:         true,
		})
	}

	return executionstate.BlockExecutionResult{
		BlockHash:   req.Block.Hash,
		BlockNumber: req.Block.Number,
		StateRoot:   s.head.StateRoot,
		GasUsed:     gasUsed,
		Receipts:    receipts,
	}, nil
}

func TestPayloadMetadata_TracksBuildSubmissionAndCanonicalization(t *testing.T) {
	head := blockassembly.Head{
		Number:    60,
		Hash:      "0xhead60",
		StateRoot: "0xstate60",
		BaseFee:   10,
	}

	state := newMetadataState(head)
	state.SetSenderNonce("alice", 0)
	state.SetSenderNonce("bob", 0)

	pool := txpool.NewPool(10)

	if err := pool.SetSenderStateNonce("alice", 0); err != nil {
		t.Fatalf("set alice sender nonce: %v", err)
	}
	if err := pool.SetSenderStateNonce("bob", 0); err != nil {
		t.Fatalf("set bob sender nonce: %v", err)
	}

	for _, tx := range []txpool.Tx{
		{Hash: "alice-0", From: "alice", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 20, MaxPriorityFeePerGas: 2, Timestamp: 1},
		{Hash: "bob-0", From: "bob", Nonce: 0, GasLimit: 21000, MaxFeePerGas: 100, MaxPriorityFeePerGas: 50, Timestamp: 1},
	} {
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
		HeadBlockHash:      "0xhead60",
		SafeBlockHash:      "0xhead60",
		FinalizedBlockHash: "0xhead60",
	}, &blockassembly.PayloadAttributes{
		Timestamp:         6001,
		FeeRecipient:      "SILA_fee_recipient_metadata",
		Random:            "SILA_random_metadata",
		SuggestedGasLimit: 0,
	})
	if err != nil {
		t.Fatalf("forkchoice updated with attrs: %v", err)
	}

	metaBuilt, err := svc.GetPayloadMetadata(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get built metadata: %v", err)
	}

	if metaBuilt.BuildSequence != 1 {
		t.Fatalf("unexpected build sequence: got=%d want=1", metaBuilt.BuildSequence)
	}
	if metaBuilt.LatestStatus != "BUILT" {
		t.Fatalf("unexpected built status: got=%s want=BUILT", metaBuilt.LatestStatus)
	}
	if metaBuilt.SubmittedToNewPayload {
		t.Fatalf("did not expect submitted=true before newPayload")
	}
	if metaBuilt.Canonical {
		t.Fatalf("did not expect canonical=true before forkchoice update")
	}

	payload, err := svc.GetPayload(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get payload: %v", err)
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

	metaSubmitted, err := svc.GetPayloadMetadata(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get submitted metadata: %v", err)
	}
	if !metaSubmitted.SubmittedToNewPayload {
		t.Fatalf("expected submitted=true after newPayload")
	}
	if metaSubmitted.LatestStatus != PayloadStatusValid {
		t.Fatalf("unexpected submitted status: got=%s want=%s", metaSubmitted.LatestStatus, PayloadStatusValid)
	}
	if metaSubmitted.Canonical {
		t.Fatalf("did not expect canonical=true before final forkchoice")
	}

	fcResult, err := svc.ForkchoiceUpdated(ForkchoiceState{
		HeadBlockHash:      payload.BlockHash,
		SafeBlockHash:      payload.BlockHash,
		FinalizedBlockHash: "0xhead60",
	})
	if err != nil {
		t.Fatalf("forkchoice updated final: %v", err)
	}
	if fcResult.PayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected final forkchoice status: got=%s want=%s", fcResult.PayloadStatus.Status, PayloadStatusValid)
	}

	metaCanonical, err := svc.GetPayloadMetadata(buildResult.PayloadID)
	if err != nil {
		t.Fatalf("get canonical metadata: %v", err)
	}
	if !metaCanonical.Canonical {
		t.Fatalf("expected canonical=true after final forkchoice")
	}
	if metaCanonical.LatestStatus != "CANONICAL" {
		t.Fatalf("unexpected canonical metadata status: got=%s want=CANONICAL", metaCanonical.LatestStatus)
	}
	if metaCanonical.BlockNumber != 61 {
		t.Fatalf("unexpected metadata block number: got=%d want=61", metaCanonical.BlockNumber)
	}
	if metaCanonical.TxCount != 2 {
		t.Fatalf("unexpected metadata tx count: got=%d want=2", metaCanonical.TxCount)
	}
	if metaCanonical.GasUsed != 42000 {
		t.Fatalf("unexpected metadata gas used: got=%d want=42000", metaCanonical.GasUsed)
	}
}

func TestPayloadMetadata_UnknownPayloadIDFails(t *testing.T) {
	head := blockassembly.Head{
		Number:    1,
		Hash:      "0xhead1",
		StateRoot: "0xstate1",
		BaseFee:   1,
	}

	state := newMetadataState(head)
	pool := txpool.NewPool(1)

	eng, err := engine.New(state, pool, 30000000)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	svc, err := NewBuilderServiceFromEngine(eng)
	if err != nil {
		t.Fatalf("new builder service from engine: %v", err)
	}

	_, err = svc.GetPayloadMetadata("missing-payload-id")
	if err == nil {
		t.Fatalf("expected unknown payload metadata error")
	}
}
