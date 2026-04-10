package p2p

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCanonicalENR(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "nodekey.json")

	cfg := &Config{
		Enabled:            true,
		NetworkName:        "sila-mainnet",
		ListenIP:           "127.0.0.1",
		TCPPort:            9000,
		UDPPort:            9000,
		MaxPeers:           50,
		KeyFile:            keyPath,
		Bootnodes:          []string{},
		ExecutionNetworkID: 919191,
		GenesisHash:        mustGenesis(),
	}

	id, err := LoadOrCreateIdentity(cfg.KeyFile)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity failed: %v", err)
	}

	canonical, err := BuildCanonicalENR(cfg, id)
	if err != nil {
		t.Fatalf("BuildCanonicalENR failed: %v", err)
	}
	defer canonical.DB.Close()

	if canonical.Sila == nil {
		t.Fatal("canonical sila enr is nil")
	}
	if canonical.Text == "" {
		t.Fatal("canonical enr text is empty")
	}
	if !strings.HasPrefix(canonical.Text, "enr:") {
		t.Fatalf("unexpected canonical text: %s", canonical.Text)
	}
	if canonical.DBPath == "" {
		t.Fatal("canonical enr db path is empty")
	}
}
