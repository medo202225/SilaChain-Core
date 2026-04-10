package consensuslegacy

import (
	"testing"
	"time"
)

func TestSlotClock_CurrentSlot(t *testing.T) {
	clock := NewSlotClock(1000, 12*time.Second, 32)

	slot := clock.CurrentSlot(time.Unix(1036, 0))
	if slot != 3 {
		t.Fatalf("expected slot 3, got %d", slot)
	}
}

func TestSlotClock_EpochForSlot(t *testing.T) {
	clock := NewSlotClock(1000, 12*time.Second, 32)

	epoch := clock.EpochForSlot(64)
	if epoch != 2 {
		t.Fatalf("expected epoch 2, got %d", epoch)
	}
}
