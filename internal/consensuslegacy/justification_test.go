package consensuslegacy

import "testing"

func TestJustificationTracker_AddIfQuorum(t *testing.T) {
	tracker := NewJustificationTracker()

	ok := tracker.AddIfQuorum(100, 3, "block-a", 2, 2)
	if !ok {
		t.Fatalf("expected quorum to justify vote")
	}

	if tracker.Count() != 1 {
		t.Fatalf("expected count 1, got %d", tracker.Count())
	}
}

func TestJustificationTracker_DoesNotAddWithoutQuorum(t *testing.T) {
	tracker := NewJustificationTracker()

	ok := tracker.AddIfQuorum(100, 3, "block-a", 1, 2)
	if ok {
		t.Fatalf("expected quorum check to fail")
	}

	if tracker.Count() != 0 {
		t.Fatalf("expected count 0, got %d", tracker.Count())
	}
}
