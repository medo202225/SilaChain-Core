package consensuslegacy

import "testing"

func TestFinalizationTracker_FinalizesOnLaterJustifiedCheckpoint(t *testing.T) {
	tracker := NewFinalizationTracker()

	if tracker.AddJustified(10, "block-a") {
		t.Fatalf("first justified should not finalize")
	}

	tracker.AddJustified(11, "block-a")

	if tracker.Count() != 1 {
		t.Fatalf("expected 1 finalized vote, got %d", tracker.Count())
	}
}

func TestFinalizationTracker_DifferentBlockHashDoesNotFinalize(t *testing.T) {
	tracker := NewFinalizationTracker()

	tracker.AddJustified(10, "block-a")
	tracker.AddJustified(11, "block-b")

	if tracker.Count() != 0 {
		t.Fatalf("expected 0 finalized votes, got %d", tracker.Count())
	}
}
