package consensuslegacy

import "time"

type Config struct {
	GenesisTimeUnix     int64
	SlotDuration        time.Duration
	SlotsPerEpoch       uint64
	ProposerLookahead   uint64
	AttestationDeadline time.Duration
}

func DefaultConfig() Config {
	return Config{
		GenesisTimeUnix:     0,
		SlotDuration:        12 * time.Second,
		SlotsPerEpoch:       32,
		ProposerLookahead:   1,
		AttestationDeadline: 4 * time.Second,
	}
}
