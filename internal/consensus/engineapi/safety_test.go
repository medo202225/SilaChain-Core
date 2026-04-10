package engineapi

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/forkchoice"
)

func TestForkchoiceUpdated_TracksSafeAndFinalizedHeads(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    0,
		Hash:      "0xgenesis",
		StateRoot: "0xstate0",
		BaseFee:   1,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	api, err := New(store)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	for _, payload := range []PayloadEnvelope{
		{BlockNumber: 1, BlockHash: "0xblock1", ParentHash: "0xgenesis", StateRoot: "0xstate1"},
		{BlockNumber: 2, BlockHash: "0xblock2", ParentHash: "0xblock1", StateRoot: "0xstate2"},
		{BlockNumber: 3, BlockHash: "0xblock3", ParentHash: "0xblock2", StateRoot: "0xstate3"},
	} {
		status, err := api.NewPayload(payload)
		if err != nil {
			t.Fatalf("new payload %s: %v", payload.BlockHash, err)
		}
		if status.Status != PayloadStatusValid {
			t.Fatalf("unexpected payload status for %s: got=%s want=%s", payload.BlockHash, status.Status, PayloadStatusValid)
		}
	}

	result, err := api.ForkchoiceUpdated(ForkchoiceState{
		HeadBlockHash:      "0xblock3",
		SafeBlockHash:      "0xblock2",
		FinalizedBlockHash: "0xblock1",
	})
	if err != nil {
		t.Fatalf("forkchoice updated: %v", err)
	}

	if result.PayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected payload status: got=%s want=%s", result.PayloadStatus.Status, PayloadStatusValid)
	}
	if result.CanonicalHead.Hash != "0xblock3" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xblock3", result.CanonicalHead.Hash)
	}
	if result.SafeHead.Hash != "0xblock2" {
		t.Fatalf("unexpected safe head hash: got=%s want=0xblock2", result.SafeHead.Hash)
	}
	if result.FinalizedHead.Hash != "0xblock1" {
		t.Fatalf("unexpected finalized head hash: got=%s want=0xblock1", result.FinalizedHead.Hash)
	}
}
