package p2p

import (
	"path/filepath"
	"testing"
)

func TestStartSilaDiscovery(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Enabled:            true,
		NetworkName:        "sila-mainnet",
		ListenIP:           "127.0.0.1",
		TCPPort:            9200,
		UDPPort:            9200,
		MaxPeers:           50,
		KeyFile:            filepath.Join(dir, "nodekey.json"),
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

	svc, err := StartSilaDiscovery(cfg, id, canonical)
	if err != nil {
		t.Fatalf("StartSilaDiscovery failed: %v", err)
	}
	defer svc.Close()

	if svc.SelfText() == "" {
		t.Fatal("self text is empty")
	}
	if svc.SelfRecord() == nil {
		t.Fatal("self record is nil")
	}
}
