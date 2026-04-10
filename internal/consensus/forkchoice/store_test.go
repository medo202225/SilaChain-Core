package forkchoice

import (
	"errors"
	"testing"

	"silachain/internal/consensus/blockassembly"
)

func TestApply_AcceptsDirectCanonicalProgression(t *testing.T) {
	store, err := New(blockassembly.Head{
		Number:    5,
		Hash:      "0xgenesis-head",
		StateRoot: "0xstate5",
		BaseFee:   10,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	result, err := store.Apply(BlockRef{
		Number:     6,
		Hash:       "0xblock6",
		ParentHash: "0xgenesis-head",
		StateRoot:  "0xstate6",
	})
	if err != nil {
		t.Fatalf("apply block: %v", err)
	}

	if !result.Accepted {
		t.Fatalf("expected accepted=true")
	}
	if !result.CanonicalChanged {
		t.Fatalf("expected canonical head change")
	}
	if result.CanonicalHead.Number != 6 {
		t.Fatalf("unexpected canonical head number: got=%d want=6", result.CanonicalHead.Number)
	}
	if result.CanonicalHead.Hash != "0xblock6" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xblock6", result.CanonicalHead.Hash)
	}
}

func TestApply_RejectsUnknownParent(t *testing.T) {
	store, err := New(blockassembly.Head{
		Number: 0,
		Hash:   "0xgenesis",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Apply(BlockRef{
		Number:     1,
		Hash:       "0xblock1",
		ParentHash: "0xmissing",
	})
	if err == nil {
		t.Fatalf("expected unknown parent error")
	}
	if !errors.Is(err, ErrUnknownParent) {
		t.Fatalf("expected ErrUnknownParent, got=%v", err)
	}
}

func TestApply_KeepsCanonicalHeadWhenShorterBranchArrives(t *testing.T) {
	store, err := New(blockassembly.Head{
		Number: 0,
		Hash:   "0xgenesis",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Apply(BlockRef{
		Number:     1,
		Hash:       "0xa1",
		ParentHash: "0xgenesis",
	})
	if err != nil {
		t.Fatalf("apply a1: %v", err)
	}

	_, err = store.Apply(BlockRef{
		Number:     2,
		Hash:       "0xa2",
		ParentHash: "0xa1",
	})
	if err != nil {
		t.Fatalf("apply a2: %v", err)
	}

	result, err := store.Apply(BlockRef{
		Number:     1,
		Hash:       "0xb1",
		ParentHash: "0xgenesis",
	})
	if err != nil {
		t.Fatalf("apply b1: %v", err)
	}

	if result.CanonicalChanged {
		t.Fatalf("did not expect canonical head to change")
	}

	head, err := store.CanonicalHead()
	if err != nil {
		t.Fatalf("canonical head: %v", err)
	}
	if head.Hash != "0xa2" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xa2", head.Hash)
	}
	if head.Number != 2 {
		t.Fatalf("unexpected canonical head number: got=%d want=2", head.Number)
	}
}

func TestApply_PromotesLongerBranchToCanonicalHead(t *testing.T) {
	store, err := New(blockassembly.Head{
		Number: 0,
		Hash:   "0xgenesis",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Apply(BlockRef{
		Number:     1,
		Hash:       "0xa1",
		ParentHash: "0xgenesis",
	})
	if err != nil {
		t.Fatalf("apply a1: %v", err)
	}

	_, err = store.Apply(BlockRef{
		Number:     1,
		Hash:       "0xb1",
		ParentHash: "0xgenesis",
	})
	if err != nil {
		t.Fatalf("apply b1: %v", err)
	}

	result, err := store.Apply(BlockRef{
		Number:     2,
		Hash:       "0xb2",
		ParentHash: "0xb1",
	})
	if err != nil {
		t.Fatalf("apply b2: %v", err)
	}

	if !result.CanonicalChanged {
		t.Fatalf("expected canonical head to change")
	}
	if result.CanonicalHead.Hash != "0xb2" {
		t.Fatalf("unexpected canonical head hash: got=%s want=0xb2", result.CanonicalHead.Hash)
	}
	if result.CanonicalHead.Number != 2 {
		t.Fatalf("unexpected canonical head number: got=%d want=2", result.CanonicalHead.Number)
	}
}

func TestApply_ReacceptsSameBlockHashConsistently(t *testing.T) {
	store, err := New(blockassembly.Head{
		Number: 0,
		Hash:   "0xgenesis",
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	block := BlockRef{
		Number:     1,
		Hash:       "0xblock1",
		ParentHash: "0xgenesis",
		StateRoot:  "0xstate1",
	}

	_, err = store.Apply(block)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}

	result, err := store.Apply(block)
	if err != nil {
		t.Fatalf("second apply should be accepted: %v", err)
	}

	if !result.Accepted {
		t.Fatalf("expected accepted=true")
	}
	if result.CanonicalChanged {
		t.Fatalf("did not expect canonical change on duplicate apply")
	}
}
