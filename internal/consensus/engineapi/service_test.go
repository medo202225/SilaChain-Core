package engineapi

import (
	"testing"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/forkchoice"
)

func TestNewPayload_ReturnsSyncingWhenParentUnknown(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    5,
		Hash:      "0xgenesis-head",
		StateRoot: "0xstate5",
		BaseFee:   10,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	api, err := New(store)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	status, err := api.NewPayload(PayloadEnvelope{
		BlockNumber: 6,
		BlockHash:   "0xblock6",
		ParentHash:  "0xmissing",
		StateRoot:   "0xstate6",
	})
	if err != nil {
		t.Fatalf("new payload: %v", err)
	}

	if status.Status != PayloadStatusSyncing {
		t.Fatalf("unexpected status: got=%s want=%s", status.Status, PayloadStatusSyncing)
	}
}

func TestNewPayload_ThenForkchoiceUpdated_CommitsBufferedChainAndAdvancesHead(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    9,
		Hash:      "0xhead9",
		StateRoot: "0xstate9",
		BaseFee:   10,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	api, err := New(store)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	status10, err := api.NewPayload(PayloadEnvelope{
		BlockNumber: 10,
		BlockHash:   "0xblock10",
		ParentHash:  "0xhead9",
		StateRoot:   "0xstate10",
	})
	if err != nil {
		t.Fatalf("new payload 10: %v", err)
	}
	if status10.Status != PayloadStatusValid {
		t.Fatalf("unexpected payload 10 status: got=%s want=%s", status10.Status, PayloadStatusValid)
	}

	status11, err := api.NewPayload(PayloadEnvelope{
		BlockNumber: 11,
		BlockHash:   "0xblock11",
		ParentHash:  "0xblock10",
		StateRoot:   "0xstate11",
	})
	if err != nil {
		t.Fatalf("new payload 11: %v", err)
	}
	if status11.Status != PayloadStatusValid {
		t.Fatalf("unexpected payload 11 status: got=%s want=%s", status11.Status, PayloadStatusValid)
	}

	result, err := api.ForkchoiceUpdated(ForkchoiceState{
		HeadBlockHash:      "0xblock11",
		SafeBlockHash:      "0xblock10",
		FinalizedBlockHash: "0xhead9",
	})
	if err != nil {
		t.Fatalf("forkchoice updated: %v", err)
	}

	if result.PayloadStatus.Status != PayloadStatusValid {
		t.Fatalf("unexpected forkchoice status: got=%s want=%s", result.PayloadStatus.Status, PayloadStatusValid)
	}
	if result.CanonicalHead.Hash != "0xblock11" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xblock11", result.CanonicalHead.Hash)
	}
	if result.CanonicalHead.Number != 11 {
		t.Fatalf("unexpected canonical head number: got=%d want=11", result.CanonicalHead.Number)
	}

	head, err := store.CanonicalHead()
	if err != nil {
		t.Fatalf("canonical head: %v", err)
	}
	if head.Hash != "0xblock11" {
		t.Fatalf("store canonical head hash mismatch: got=%s want=0xblock11", head.Hash)
	}
}

func TestForkchoiceUpdated_ReturnsSyncingForUnknownHead(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    2,
		Hash:      "0xhead2",
		StateRoot: "0xstate2",
		BaseFee:   5,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	api, err := New(store)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := api.ForkchoiceUpdated(ForkchoiceState{
		HeadBlockHash:      "0xunknown",
		SafeBlockHash:      "0xunknown",
		FinalizedBlockHash: "0xhead2",
	})
	if err != nil {
		t.Fatalf("forkchoice updated: %v", err)
	}

	if result.PayloadStatus.Status != PayloadStatusSyncing {
		t.Fatalf("unexpected status: got=%s want=%s", result.PayloadStatus.Status, PayloadStatusSyncing)
	}
	if result.CanonicalHead.Hash != "0xhead2" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xhead2", result.CanonicalHead.Hash)
	}
}

func TestUpdateCanonicalHead_AllowsKnownBranchSelection(t *testing.T) {
	store, err := forkchoice.New(blockassembly.Head{
		Number:    0,
		Hash:      "0xgenesis",
		StateRoot: "0xstate0",
		BaseFee:   1,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	if _, err := store.Apply(forkchoice.BlockRef{
		Number:     1,
		Hash:       "0xa1",
		ParentHash: "0xgenesis",
		StateRoot:  "0xstate-a1",
	}); err != nil {
		t.Fatalf("apply a1: %v", err)
	}

	if _, err := store.Apply(forkchoice.BlockRef{
		Number:     1,
		Hash:       "0xb1",
		ParentHash: "0xgenesis",
		StateRoot:  "0xstate-b1",
	}); err != nil {
		t.Fatalf("apply b1: %v", err)
	}

	result, err := store.UpdateCanonicalHead("0xb1")
	if err != nil {
		t.Fatalf("update canonical head: %v", err)
	}

	if !result.CanonicalChanged {
		t.Fatalf("expected canonical changed=true")
	}
	if result.CanonicalHead.Hash != "0xb1" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xb1", result.CanonicalHead.Hash)
	}
}
