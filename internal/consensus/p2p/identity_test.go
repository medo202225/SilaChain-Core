package p2p

import (
	"path/filepath"
	"testing"

	chaincrypto "silachain/pkg/crypto"
)

func TestLoadOrCreateIdentityStableAcrossReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodekey.json")

	first, err := LoadOrCreateIdentity(path)
	if err != nil {
		t.Fatalf("first LoadOrCreateIdentity failed: %v", err)
	}

	second, err := LoadOrCreateIdentity(path)
	if err != nil {
		t.Fatalf("second LoadOrCreateIdentity failed: %v", err)
	}

	if first.PrivateKeyHex != second.PrivateKeyHex {
		t.Fatalf("private key changed across reload")
	}

	if first.PeerID != second.PeerID {
		t.Fatalf("peer id changed across reload: %s != %s", first.PeerID, second.PeerID)
	}
}

func TestDerivePeerIDFromPublicKeyKnownVector(t *testing.T) {
	priv, err := chaincrypto.HexToPrivateKey("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("HexToPrivateKey failed: %v", err)
	}

	got, err := DerivePeerIDFromPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("DerivePeerIDFromPublicKey failed: %v", err)
	}

	want := "16Uiu2HAm3cuhhRL2msUuLF62KRSfneFDx94RsuouyW25Ho42cFMq"
	if got != want {
		t.Fatalf("unexpected peer id: got %s want %s", got, want)
	}
}
