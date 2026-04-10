package forkchoice

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestUpdateSafety_TracksSafeAndFinalizedHeads(t *testing.T) {
	store, err := New(blockassembly.Head{
		Number:    0,
		Hash:      "0xgenesis",
		StateRoot: "0xstate0",
		BaseFee:   1,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	for _, block := range []BlockRef{
		{Number: 1, Hash: "0xblock1", ParentHash: "0xgenesis", StateRoot: "0xstate1"},
		{Number: 2, Hash: "0xblock2", ParentHash: "0xblock1", StateRoot: "0xstate2"},
		{Number: 3, Hash: "0xblock3", ParentHash: "0xblock2", StateRoot: "0xstate3"},
	} {
		if _, err := store.Apply(block); err != nil {
			t.Fatalf("apply %s: %v", block.Hash, err)
		}
	}

	result, err := store.UpdateSafety("0xblock2", "0xblock1")
	if err != nil {
		t.Fatalf("update safety: %v", err)
	}

	if result.SafeHead.Hash != "0xblock2" {
		t.Fatalf("unexpected safe head hash: got=%s want=0xblock2", result.SafeHead.Hash)
	}
	if result.FinalizedHead.Hash != "0xblock1" {
		t.Fatalf("unexpected finalized head hash: got=%s want=0xblock1", result.FinalizedHead.Hash)
	}

	safeHead, err := store.SafeHead()
	if err != nil {
		t.Fatalf("safe head: %v", err)
	}
	finalizedHead, err := store.FinalizedHead()
	if err != nil {
		t.Fatalf("finalized head: %v", err)
	}

	if safeHead.Hash != "0xblock2" {
		t.Fatalf("store safe head mismatch: got=%s want=0xblock2", safeHead.Hash)
	}
	if finalizedHead.Hash != "0xblock1" {
		t.Fatalf("store finalized head mismatch: got=%s want=0xblock1", finalizedHead.Hash)
	}
}
