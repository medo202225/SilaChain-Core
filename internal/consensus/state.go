package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

type State struct {
	CurrentSlot        uint64
	CurrentEpoch       uint64
	HeadBlockHash      string
	SafeBlockHash      string
	FinalizedBlockHash string
	LastPayloadID      string
}

func NewState() *State {
	s := &State{}
	s.Advance(0, 32)
	return s
}

func hashLabelAndUint64(label string, v uint64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	sum := sha256.Sum256(append([]byte(label+":"), buf...))
	return "0x" + hex.EncodeToString(sum[:])
}

func (s *State) Advance(slot uint64, slotsPerEpoch uint64) {
	if s == nil {
		return
	}
	if slotsPerEpoch == 0 {
		slotsPerEpoch = 32
	}

	s.CurrentSlot = slot
	s.CurrentEpoch = slot / slotsPerEpoch
	s.HeadBlockHash = hashLabelAndUint64("head", slot)

	if slot == 0 {
		s.SafeBlockHash = s.HeadBlockHash
	} else {
		s.SafeBlockHash = hashLabelAndUint64("safe", slot-1)
	}

	if s.CurrentEpoch == 0 {
		s.FinalizedBlockHash = s.HeadBlockHash
	} else {
		s.FinalizedBlockHash = hashLabelAndUint64("finalized-epoch", s.CurrentEpoch-1)
	}
}
