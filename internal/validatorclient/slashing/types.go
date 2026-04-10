package slashing

import "time"

type SignedBlock struct {
	PubKey      []byte
	Slot        uint64
	SigningRoot []byte
	CreatedAt   time.Time
}

type SignedAttestation struct {
	PubKey      []byte
	SourceEpoch uint64
	TargetEpoch uint64
	SigningRoot []byte
	CreatedAt   time.Time
}
