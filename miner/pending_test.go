package miner

import (
	"testing"
	"time"
)

func TestPendingResolveNilWhenEmpty(t *testing.T) {
	var p pending
	if p.resolve("0xparent1") != nil {
		t.Fatalf("expected nil result for empty pending cache")
	}
}

func TestPendingResolveNilOnParentMismatch(t *testing.T) {
	p := pending{}
	result := &ExecutableData{BlockHash: "0xblock1"}
	p.update("0xparent1", result)

	if p.resolve("0xother") != nil {
		t.Fatalf("expected nil result on parent mismatch")
	}
}

func TestPendingResolveNilWhenExpired(t *testing.T) {
	p := pending{
		created:    time.Now().Add(-(pendingTTL + time.Millisecond)),
		parentHash: "0xparent1",
		result:     &ExecutableData{BlockHash: "0xblock1"},
	}

	if p.resolve("0xparent1") != nil {
		t.Fatalf("expected nil result after ttl expiry")
	}
}

func TestPendingResolveReturnsCachedResult(t *testing.T) {
	p := pending{}
	result := &ExecutableData{
		BlockHash:   "0xblock1",
		BlockNumber: 1,
		StateRoot:   "0xstate1",
	}

	p.update("0xparent1", result)

	got := p.resolve("0xparent1")
	if got == nil {
		t.Fatalf("expected cached result")
	}
	if got.BlockHash != "0xblock1" {
		t.Fatalf("unexpected block hash: got=%s want=0xblock1", got.BlockHash)
	}
	if got.StateRoot != "0xstate1" {
		t.Fatalf("unexpected state root: got=%s want=0xstate1", got.StateRoot)
	}
}
