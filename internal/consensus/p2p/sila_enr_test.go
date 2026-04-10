package p2p

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSilaENR(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Enabled:            true,
		NetworkName:        "sila-mainnet",
		ListenIP:           "127.0.0.1",
		TCPPort:            9000,
		UDPPort:            9000,
		MaxPeers:           50,
		KeyFile:            filepath.Join(dir, "nodekey.json"),
		ExecutionNetworkID: 919191,
		GenesisHash:        mustGenesis(),
	}

	id, err := LoadOrCreateIdentity(cfg.KeyFile)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity failed: %v", err)
	}

	record, err := BuildSilaENR(cfg, id)
	if err != nil {
		t.Fatalf("BuildSilaENR failed: %v", err)
	}

	if record.IP != "127.0.0.1" {
		t.Fatalf("unexpected IP: %s", record.IP)
	}
	if record.TCP != 9000 || record.UDP != 9000 {
		t.Fatalf("unexpected ports: tcp=%d udp=%d", record.TCP, record.UDP)
	}
	if record.PeerID == "" {
		t.Fatal("peer id is empty")
	}
	if record.PublicKey == "" {
		t.Fatal("public key is empty")
	}
	if record.Signature == "" {
		t.Fatal("signature is empty")
	}

	text, err := record.Text()
	if err != nil {
		t.Fatalf("Text failed: %v", err)
	}
	if !strings.HasPrefix(text, "enr:") {
		t.Fatalf("unexpected ENR text: %s", text)
	}
}
