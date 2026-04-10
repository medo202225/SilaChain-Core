package p2p

import (
	"path/filepath"
	"testing"
	"time"

	execstate "silachain/internal/execution/executionstate"
)

func TestSilaExecutionTransportStaticPeerConnection(t *testing.T) {
	dir := t.TempDir()

	cfg1 := &Config{
		Enabled:            true,
		NetworkName:        "sila-mainnet",
		ListenIP:           "127.0.0.1",
		TCPPort:            9400,
		UDPPort:            9400,
		MaxPeers:           10,
		KeyFile:            filepath.Join(dir, "node1", "nodekey.json"),
		ExecutionNetworkID: 919191,
		GenesisHash:        mustGenesis(),
	}
	cfg2 := &Config{
		Enabled:            true,
		NetworkName:        "sila-mainnet",
		ListenIP:           "127.0.0.1",
		TCPPort:            9401,
		UDPPort:            9401,
		MaxPeers:           10,
		KeyFile:            filepath.Join(dir, "node2", "nodekey.json"),
		ExecutionNetworkID: 919191,
		GenesisHash:        mustGenesis(),
	}

	id1, err := LoadOrCreateIdentity(cfg1.KeyFile)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity node1 failed: %v", err)
	}
	id2, err := LoadOrCreateIdentity(cfg2.KeyFile)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity node2 failed: %v", err)
	}

	canonical1, err := BuildCanonicalENR(cfg1, id1)
	if err != nil {
		t.Fatalf("BuildCanonicalENR node1 failed: %v", err)
	}
	defer canonical1.DB.Close()

	canonical2, err := BuildCanonicalENR(cfg2, id2)
	if err != nil {
		t.Fatalf("BuildCanonicalENR node2 failed: %v", err)
	}
	defer canonical2.DB.Close()

	state1 := execstate.NewState(cfg1.GenesisHash)
	state2 := execstate.NewState(cfg2.GenesisHash)

	s1, err := StartSilaExecutionTransport(cfg1, id1, canonical1, state1, nil)
	if err != nil {
		t.Fatalf("StartSilaExecutionTransport node1 failed: %v", err)
	}
	defer s1.Stop()

	s2, err := StartSilaExecutionTransport(cfg2, id2, canonical2, state2, []string{s1.SelfAddr()})
	if err != nil {
		t.Fatalf("StartSilaExecutionTransport node2 failed: %v", err)
	}
	defer s2.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s1.PeerCount() >= 1 &&
			s2.PeerCount() >= 1 &&
			s1.RunCount() >= 1 &&
			s2.RunCount() >= 1 &&
			s1.StatusCount() >= 1 &&
			s2.StatusCount() >= 1 &&
			s1.HeaderRequestCount() >= 1 &&
			s1.HeaderResponseCount() >= 1 &&
			s2.ReceivedHeaderCount() >= 1 &&
			s1.BodyRequestCount() >= 1 &&
			s1.BodyResponseCount() >= 1 &&
			s2.ReceivedBodyCount() >= 1 &&
			s1.NewBlockHashesSentCount() >= 1 &&
			s2.NewBlockHashesRecvCount() >= 1 &&
			s1.NewBlockSentCount() >= 1 &&
			s2.NewBlockRecvCount() >= 1 &&
			s1.MempoolAnnounceSentCount() >= 1 &&
			s2.MempoolAnnounceRecvCount() >= 1 &&
			s2.MempoolMissingReqCount() >= 1 &&
			s1.PooledRespCount() >= 1 &&
			s2.MempoolInsertedCount() >= 1 &&
			s2.SyncRemoteHeadCount() >= 1 &&
			s2.SyncRequestCount() >= 2 &&
			s2.SyncImportCount() >= 1 &&
			s1.StateHeadNumber() >= 1 &&
			s2.StateHeadNumber() >= 1 &&
			s1.StatePendingCount() >= 1 &&
			s2.StatePendingCount() >= 1 &&
			s1.ImportAcceptCount() >= 1 &&
			s2.ImportAcceptCount() >= 1 &&
			s1.ImportRejectCount() == 0 &&
			s2.ImportRejectCount() == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf(
		"expected native execution protocol with execution validation/import lifecycle, got s1 peers=%d runs=%d state_head=%d state_pending=%d import_accept=%d import_reject=%d, s2 peers=%d runs=%d state_head=%d state_pending=%d sync_import=%d import_accept=%d import_reject=%d",
		s1.PeerCount(), s1.RunCount(), s1.StateHeadNumber(), s1.StatePendingCount(), s1.ImportAcceptCount(), s1.ImportRejectCount(),
		s2.PeerCount(), s2.RunCount(), s2.StateHeadNumber(), s2.StatePendingCount(), s2.SyncImportCount(), s2.ImportAcceptCount(), s2.ImportRejectCount(),
	)
}
