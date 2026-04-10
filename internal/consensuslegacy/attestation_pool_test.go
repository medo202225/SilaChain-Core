package consensuslegacy

import (
	"testing"
	"time"
)

func TestAttestationPool_AddAndCount(t *testing.T) {
	key := mustLoadValidatorKeyForTest(t)

	pool := NewAttestationPool()

	a := NewAttestation(
		11,
		0,
		key.File.Address,
		key.File.PublicKey,
		"blockhash-1",
		0,
		0,
		time.Unix(1774443949, 0).UTC(),
	)

	if err := a.Sign(key); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if err := pool.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if pool.Count() != 1 {
		t.Fatalf("expected count 1, got %d", pool.Count())
	}
}

func TestAttestationPool_RejectsInvalidAttestation(t *testing.T) {
	key := mustLoadValidatorKeyForTest(t)

	pool := NewAttestationPool()

	a := NewAttestation(
		11,
		0,
		key.File.Address,
		key.File.PublicKey,
		"blockhash-1",
		0,
		0,
		time.Unix(1774443949, 0).UTC(),
	)

	if err := a.Sign(key); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	a.BlockHash = "tampered"

	if err := pool.Add(a); err == nil {
		t.Fatalf("expected invalid attestation to be rejected")
	}
}

func TestAttestationPool_Clear(t *testing.T) {
	key := mustLoadValidatorKeyForTest(t)

	pool := NewAttestationPool()

	a := NewAttestation(
		12,
		0,
		key.File.Address,
		key.File.PublicKey,
		"blockhash-2",
		0,
		0,
		time.Unix(1774443949, 0).UTC(),
	)

	if err := a.Sign(key); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if err := pool.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	pool.Clear()

	if pool.Count() != 0 {
		t.Fatalf("expected count 0 after clear, got %d", pool.Count())
	}
}
