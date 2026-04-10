package validator

import (
	"path/filepath"
	"testing"
)

func TestGenerateValidatorKey(t *testing.T) {
	key, err := GenerateValidatorKey()
	if err != nil {
		t.Fatalf("GenerateValidatorKey failed: %v", err)
	}
	if key == nil {
		t.Fatalf("expected key, got nil")
	}
	if key.PrivateHex == "" || key.PublicHex == "" || key.Address == "" {
		t.Fatalf("expected full generated key data")
	}
}

func TestCreateAndLoadKeyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "validator_key.json")

	file, err := CreateAndSaveKeyFile(path)
	if err != nil {
		t.Fatalf("CreateAndSaveKeyFile failed: %v", err)
	}
	if file == nil {
		t.Fatalf("expected key file, got nil")
	}

	loaded, err := LoadKeyFile(path)
	if err != nil {
		t.Fatalf("LoadKeyFile failed: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected loaded key, got nil")
	}
	if loaded.File.Address != file.Address {
		t.Fatalf("expected address %s, got %s", file.Address, loaded.File.Address)
	}
}
