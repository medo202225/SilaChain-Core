package p2p

import (
	"path/filepath"
	"testing"
	"time"
)

func mustGenesis() string {
	return "0x1111111111111111111111111111111111111111111111111111111111111111"
}

func TestSilaDiscoveryBootnodeBetweenTwoNodes(t *testing.T) {
	dir := t.TempDir()

	cfg1 := &Config{
		Enabled:            true,
		NetworkName:        "sila-mainnet",
		ListenIP:           "127.0.0.1",
		TCPPort:            9300,
		UDPPort:            9300,
		MaxPeers:           50,
		KeyFile:            filepath.Join(dir, "node1", "nodekey.json"),
		ExecutionNetworkID: 919191,
		GenesisHash:        mustGenesis(),
		Bootnodes:          []string{},
	}

	id1, err := LoadOrCreateIdentity(cfg1.KeyFile)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity node1 failed: %v", err)
	}

	canonical1, err := BuildCanonicalENR(cfg1, id1)
	if err != nil {
		t.Fatalf("BuildCanonicalENR node1 failed: %v", err)
	}
	defer canonical1.DB.Close()

	svc1, err := StartSilaDiscovery(cfg1, id1, canonical1)
	if err != nil {
		t.Fatalf("StartSilaDiscovery node1 failed: %v", err)
	}
	defer svc1.Close()

	bootnodeText := svc1.SelfText()
	if bootnodeText == "" {
		t.Fatal("bootnode text is empty")
	}

	cfg2 := &Config{
		Enabled:            true,
		NetworkName:        "sila-mainnet",
		ListenIP:           "127.0.0.1",
		TCPPort:            9301,
		UDPPort:            9301,
		MaxPeers:           50,
		KeyFile:            filepath.Join(dir, "node2", "nodekey.json"),
		ExecutionNetworkID: 919191,
		GenesisHash:        mustGenesis(),
		Bootnodes:          []string{bootnodeText},
	}

	id2, err := LoadOrCreateIdentity(cfg2.KeyFile)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity node2 failed: %v", err)
	}

	canonical2, err := BuildCanonicalENR(cfg2, id2)
	if err != nil {
		t.Fatalf("BuildCanonicalENR node2 failed: %v", err)
	}
	defer canonical2.DB.Close()

	svc2, err := StartSilaDiscovery(cfg2, id2, canonical2)
	if err != nil {
		t.Fatalf("StartSilaDiscovery node2 failed: %v", err)
	}
	defer svc2.Close()

	if err := svc2.PingENRText(bootnodeText); err != nil {
		t.Fatalf("svc2 ping bootnode failed: %v", err)
	}
	if err := svc1.PingENRText(svc2.SelfText()); err != nil {
		t.Fatalf("svc1 ping node2 failed: %v", err)
	}

	node1PeerID := canonical1.Sila.PeerID
	node2PeerID := canonical2.Sila.PeerID

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if svc1.TableNodeCount() >= 1 && svc2.TableNodeCount() >= 1 {
			p1 := svc1.ResolvePeer(node2PeerID)
			p2 := svc2.ResolvePeer(node1PeerID)
			if p1 != nil && p2 != nil {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("expected both nodes to discover each other via native Sila bootnode, got node1=%d node2=%d", svc1.TableNodeCount(), svc2.TableNodeCount())
}
