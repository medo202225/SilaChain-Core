package consensuslegacy

import (
	"path/filepath"
	"testing"
	"time"

	"silachain/internal/validator"
)

func mustLoadValidatorKeyForTest(t *testing.T) *validator.LoadedKey {
	t.Helper()

	keyPath := filepath.Join(t.TempDir(), "validator-test.key")

	if _, err := validator.CreateAndSaveKeyFile(keyPath); err != nil {
		t.Fatalf("CreateAndSaveKeyFile failed: %v", err)
	}

	key, err := validator.LoadKeyFile(keyPath)
	if err != nil {
		t.Fatalf("LoadKeyFile failed: %v", err)
	}

	return key
}

func TestAttestation_SignAndVerify(t *testing.T) {
	key := mustLoadValidatorKeyForTest(t)

	a := NewAttestation(
		10,
		0,
		key.File.Address,
		key.File.PublicKey,
		"abc123",
		0,
		0,
		time.Unix(1774443949, 0).UTC(),
	)

	if err := a.Sign(key); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if err := a.Verify(); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
}

func TestAttestation_VerifyFailsOnTamper(t *testing.T) {
	key := mustLoadValidatorKeyForTest(t)

	a := NewAttestation(
		10,
		0,
		key.File.Address,
		key.File.PublicKey,
		"abc123",
		0,
		0,
		time.Unix(1774443949, 0).UTC(),
	)

	if err := a.Sign(key); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	a.BlockHash = "tampered"

	if err := a.Verify(); err == nil {
		t.Fatalf("expected verify failure after tamper")
	}
}
