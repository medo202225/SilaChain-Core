package consensuslegacy

import (
	"testing"
	"time"
)

func TestAggregateAttestations_DeduplicatesSameValidator(t *testing.T) {
	key := mustLoadValidatorKeyForTest(t)

	a1 := NewAttestation(
		20,
		1,
		key.File.Address,
		key.File.PublicKey,
		"blockhash-x",
		1,
		1,
		time.Unix(1774443949, 0).UTC(),
	)
	if err := a1.Sign(key); err != nil {
		t.Fatalf("Sign a1 failed: %v", err)
	}

	a2 := NewAttestation(
		20,
		1,
		key.File.Address,
		key.File.PublicKey,
		"blockhash-x",
		1,
		1,
		time.Unix(1774443950, 0).UTC(),
	)
	if err := a2.Sign(key); err != nil {
		t.Fatalf("Sign a2 failed: %v", err)
	}

	aggregates := AggregateAttestations([]Attestation{a1, a2})
	if len(aggregates) != 1 {
		t.Fatalf("expected 1 aggregate, got %d", len(aggregates))
	}

	if aggregates[0].VoteCount != 1 {
		t.Fatalf("expected deduplicated vote count 1, got %d", aggregates[0].VoteCount)
	}
}
