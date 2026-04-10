package engineapi

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/forkchoice"
	"silachain/internal/consensus/txpool"
)

type stubAssembler struct {
	result blockassembly.Result
	err    error
}

func (s *stubAssembler) Assemble(attrs blockassembly.PayloadAttributes) (blockassembly.Result, error) {
	if s.err != nil {
		return blockassembly.Result{}, s.err
	}
	return s.result, nil
}

func TestForkchoiceUpdatedWithAttributes_CreatesPayloadIDAndCachesPayload(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    10,
		Hash:      "0xhead10",
		StateRoot: "0xstate10",
		BaseFee:   10,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	svc, err := NewBuilderService(store, &stubAssembler{
		result: blockassembly.Result{
			ParentNumber:    10,
			BlockNumber:     11,
			ParentHash:      "0xhead10",
			ParentStateRoot: "0xstate10",
			BaseFee:         10,
			GasLimit:        30000000,
			Attributes: blockassembly.PayloadAttributes{
				Timestamp:         111,
				FeeRecipient:      "SILA_fee_recipient_payload",
				Random:            "SILA_random_payload",
				SuggestedGasLimit: 0,
			},
			Selection: blockassembly.TransactionSelection{
				Transactions: []txpool.Tx{
					{Hash: "tx1", From: "alice", Nonce: 0, GasLimit: 21000},
					{Hash: "tx2", From: "bob", Nonce: 0, GasLimit: 21000},
				},
				GasUsed:      42000,
				TotalTipFees: 100,
			},
		},
	})
	if err != nil {
		t.Fatalf("new builder service: %v", err)
	}

	result, err := svc.ForkchoiceUpdatedWithAttributes(ForkchoiceState{
		HeadBlockHash:      "0xhead10",
		SafeBlockHash:      "0xhead10",
		FinalizedBlockHash: "0xhead10",
	}, &blockassembly.PayloadAttributes{
		Timestamp:         111,
		FeeRecipient:      "SILA_fee_recipient_payload",
		Random:            "SILA_random_payload",
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

	payload, err := svc.GetPayload(result.PayloadID)
	if err != nil {
		t.Fatalf("get payload: %v", err)
	}

	if payload.BlockNumber != 11 {
		t.Fatalf("unexpected payload block number: got=%d want=11", payload.BlockNumber)
	}
	if payload.ParentHash != "0xhead10" {
		t.Fatalf("unexpected payload parent hash: got=%s want=0xhead10", payload.ParentHash)
	}
	if payload.TxCount != 2 {
		t.Fatalf("unexpected payload tx count: got=%d want=2", payload.TxCount)
	}
	if payload.GasUsed != 42000 {
		t.Fatalf("unexpected payload gas used: got=%d want=42000", payload.GasUsed)
	}
}

func TestGetPayload_ReturnsErrorForUnknownPayloadID(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    1,
		Hash:      "0xhead1",
		StateRoot: "0xstate1",
		BaseFee:   1,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	svc, err := NewBuilderService(store, &stubAssembler{
		result: blockassembly.Result{},
	})
	if err != nil {
		t.Fatalf("new builder service: %v", err)
	}

	_, err = svc.GetPayload("missing-payload-id")
	if err == nil {
		t.Fatalf("expected unknown payload id error")
	}
}

func TestForkchoiceUpdatedWithAttributes_AllowsNilAttributesWithoutPayloadBuild(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    2,
		Hash:      "0xhead2",
		StateRoot: "0xstate2",
		BaseFee:   5,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	svc, err := NewBuilderService(store, &stubAssembler{
		result: blockassembly.Result{},
	})
	if err != nil {
		t.Fatalf("new builder service: %v", err)
	}

	result, err := svc.ForkchoiceUpdatedWithAttributes(ForkchoiceState{
		HeadBlockHash:      "0xhead2",
		SafeBlockHash:      "0xhead2",
		FinalizedBlockHash: "0xhead2",
	}, nil)
	if err != nil {
		t.Fatalf("forkchoice updated with nil attrs: %v", err)
	}

	if result.PayloadID != "" {
		t.Fatalf("expected empty payload id when attrs are nil")
	}
	if result.PayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected payload status: got=%s want=%s", result.PayloadStatus.Status, PayloadStatusValid)
	}
}
