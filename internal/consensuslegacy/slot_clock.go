package consensuslegacy

import "time"

type Slot uint64
type Epoch uint64

type SlotClock struct {
	genesisTime   time.Time
	slotDuration  time.Duration
	slotsPerEpoch uint64
}

func NewSlotClock(genesisUnix int64, slotDuration time.Duration, slotsPerEpoch uint64) *SlotClock {
	if slotDuration <= 0 {
		slotDuration = 12 * time.Second
	}
	if slotsPerEpoch == 0 {
		slotsPerEpoch = 32
	}

	return &SlotClock{
		genesisTime:   time.Unix(genesisUnix, 0).UTC(),
		slotDuration:  slotDuration,
		slotsPerEpoch: slotsPerEpoch,
	}
}

func (c *SlotClock) CurrentSlot(now time.Time) Slot {
	if c == nil {
		return 0
	}

	now = now.UTC()
	if now.Before(c.genesisTime) {
		return 0
	}

	elapsed := now.Sub(c.genesisTime)
	return Slot(uint64(elapsed / c.slotDuration))
}

func (c *SlotClock) SlotStart(slot Slot) time.Time {
	if c == nil {
		return time.Unix(0, 0).UTC()
	}
	return c.genesisTime.Add(time.Duration(slot) * c.slotDuration).UTC()
}

func (c *SlotClock) SlotEnd(slot Slot) time.Time {
	if c == nil {
		return time.Unix(0, 0).UTC()
	}
	return c.SlotStart(slot).Add(c.slotDuration).UTC()
}

func (c *SlotClock) EpochForSlot(slot Slot) Epoch {
	if c == nil || c.slotsPerEpoch == 0 {
		return 0
	}
	return Epoch(uint64(slot) / c.slotsPerEpoch)
}

func (c *SlotClock) IsGenesisStarted(now time.Time) bool {
	if c == nil {
		return false
	}
	return !now.UTC().Before(c.genesisTime)
}
