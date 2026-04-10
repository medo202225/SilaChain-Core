package consensus

import "testing"

func TestForkchoiceStoreUpdatesFromBeaconState(t *testing.T) {
	state := NewBeaconStateV1(nil)
	state.AdvanceSlot(64, 32)

	store := NewForkchoiceStore()
	if err := store.UpdateFromBeaconState(state); err != nil {
		t.Fatalf("update from beacon state failed: %v", err)
	}

	fc := store.State()
	if fc.HeadRoot != state.HeadBlockRoot {
		t.Fatalf("expected head root %s, got %s", state.HeadBlockRoot, fc.HeadRoot)
	}
	if fc.SafeRoot != state.SafeBlockRoot {
		t.Fatalf("expected safe root %s, got %s", state.SafeBlockRoot, fc.SafeRoot)
	}
	if fc.FinalizedRoot != state.FinalizedCheckpoint.Root {
		t.Fatalf("expected finalized root %s, got %s", state.FinalizedCheckpoint.Root, fc.FinalizedRoot)
	}
}
