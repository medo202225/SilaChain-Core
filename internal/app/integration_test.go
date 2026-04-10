package app_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"silachain/internal/app"
	"silachain/internal/chain"
	coretypes "silachain/internal/core/types"
	"silachain/internal/rpc"
	"silachain/internal/validator"
	chaincrypto "silachain/pkg/crypto"
	pkgtypes "silachain/pkg/types"
)

func mustAppKeyAndAddress(t *testing.T) (*ecdsa.PrivateKey, string, string) {
	t.Helper()

	priv, pub, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	pubHex := chaincrypto.PublicKeyToHex(pub)
	if err != nil {
		t.Fatalf("PublicKeyToHex failed: %v", err)
	}

	addr := chaincrypto.PublicKeyToAddress(pub)
	if err != nil {
		t.Fatalf("PublicKeyToAddress failed: %v", err)
	}

	return priv, pubHex, string(addr)
}

func mustAppBlockchainWithSharedState(
	t *testing.T,
	dataDir string,
	senderAddr pkgtypes.Address,
	senderPubHex string,
	receiverAddr pkgtypes.Address,
	receiverPubHex string,
) *chain.Blockchain {
	t.Helper()

	validatorSet := validator.NewSet([]validator.Member{
		{
			Address:   senderAddr,
			PublicKey: senderPubHex,
			Power:     100,
			Stake:     1000,
		},
	})

	bc, err := chain.NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain failed: %v", err)
	}

	if _, err := bc.RegisterAccount(senderAddr, senderPubHex); err != nil {
		t.Fatalf("RegisterAccount sender failed: %v", err)
	}
	if _, err := bc.RegisterAccount(receiverAddr, receiverPubHex); err != nil {
		t.Fatalf("RegisterAccount receiver failed: %v", err)
	}
	if err := bc.Faucet(senderAddr, 1000); err != nil {
		t.Fatalf("Faucet failed: %v", err)
	}

	return bc
}

func mustAppBlockchain(t *testing.T, dataDir string) (*chain.Blockchain, pkgtypes.Address, *ecdsa.PrivateKey, string, pkgtypes.Address, *ecdsa.PrivateKey, string, *validator.Set) {
	t.Helper()

	senderPriv, senderPubHex, senderAddrStr := mustAppKeyAndAddress(t)
	receiverPriv, receiverPubHex, receiverAddrStr := mustAppKeyAndAddress(t)

	validatorSet := validator.NewSet([]validator.Member{
		{
			Address:   pkgtypes.Address(senderAddrStr),
			PublicKey: senderPubHex,
			Power:     100,
			Stake:     1000,
		},
	})

	bc, err := chain.NewBlockchain(dataDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain failed: %v", err)
	}

	senderAddr := pkgtypes.Address(senderAddrStr)
	receiverAddr := pkgtypes.Address(receiverAddrStr)

	if _, err := bc.RegisterAccount(senderAddr, senderPubHex); err != nil {
		t.Fatalf("RegisterAccount sender failed: %v", err)
	}
	if _, err := bc.RegisterAccount(receiverAddr, receiverPubHex); err != nil {
		t.Fatalf("RegisterAccount receiver failed: %v", err)
	}
	if err := bc.Faucet(senderAddr, 1000); err != nil {
		t.Fatalf("Faucet failed: %v", err)
	}

	return bc, senderAddr, senderPriv, senderPubHex, receiverAddr, receiverPriv, receiverPubHex, validatorSet
}

func mustAppSignedTx(t *testing.T, from pkgtypes.Address, to pkgtypes.Address, nonce pkgtypes.Nonce, value pkgtypes.Amount, fee pkgtypes.Amount, chainID pkgtypes.ChainID, pubHex string, priv *ecdsa.PrivateKey) *coretypes.Transaction {
	t.Helper()

	transaction := &coretypes.Transaction{
		From:      from,
		To:        to,
		Value:     value,
		Fee:       fee,
		Nonce:     nonce,
		ChainID:   chainID,
		Timestamp: pkgtypes.Timestamp(time.Now().Unix()),
		PublicKey: pubHex,
	}

	hash, err := coretypes.ComputeHash(transaction)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}
	transaction.Hash = hash

	if err := coretypes.SignTransaction(transaction, priv); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	return transaction
}

func TestBlockSync_ImportsMissingBlocks(t *testing.T) {
	leaderDir := t.TempDir()
	followerDir := t.TempDir()

	leaderPriv, leaderPubHex, leaderAddrStr := mustAppKeyAndAddress(t)
	_, receiverPubHex, receiverAddrStr := mustAppKeyAndAddress(t)

	leaderAddr := pkgtypes.Address(leaderAddrStr)
	receiverAddr := pkgtypes.Address(receiverAddrStr)

	leader := mustAppBlockchainWithSharedState(t, leaderDir, leaderAddr, leaderPubHex, receiverAddr, receiverPubHex)
	follower := mustAppBlockchainWithSharedState(t, followerDir, leaderAddr, leaderPubHex, receiverAddr, receiverPubHex)

	tx1 := mustAppSignedTx(t, leaderAddr, receiverAddr, 0, 10, 1, 1001, leaderPubHex, leaderPriv)
	if err := leader.SubmitTransaction(tx1); err != nil {
		t.Fatalf("SubmitTransaction tx1 failed: %v", err)
	}
	if _, err := leader.MinePending(leaderAddr); err != nil {
		t.Fatalf("MinePending tx1 failed: %v", err)
	}

	tx2 := mustAppSignedTx(t, leaderAddr, receiverAddr, 1, 10, 1, 1001, leaderPubHex, leaderPriv)
	if err := leader.SubmitTransaction(tx2); err != nil {
		t.Fatalf("SubmitTransaction tx2 failed: %v", err)
	}
	if _, err := leader.MinePending(leaderAddr); err != nil {
		t.Fatalf("MinePending tx2 failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux := http.NewServeMux()
		mux.HandleFunc("/chain/info", rpc.ChainInfoHandler(leader))
		mux.HandleFunc("/blocks/height", rpc.BlockByHeightHandler(leader))
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	syncer := app.NewBlockSyncService(follower, []string{server.URL}, "", time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		syncer.Start(ctx)
	}()

	time.Sleep(1200 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("block sync goroutine did not stop in time")
	}

	height, err := follower.Height()
	if err != nil {
		t.Fatalf("follower Height failed: %v", err)
	}
	if height != 2 {
		t.Fatalf("expected follower height 2, got %d", height)
	}
}

func TestBlockSync_ReorgGuardBlocksDivergentPeer(t *testing.T) {
	localDir := t.TempDir()
	peerDir := t.TempDir()

	sharedPriv, sharedPubHex, sharedAddrStr := mustAppKeyAndAddress(t)
	_, receiverPubHex, receiverAddrStr := mustAppKeyAndAddress(t)

	sharedAddr := pkgtypes.Address(sharedAddrStr)
	receiverAddr := pkgtypes.Address(receiverAddrStr)

	local := mustAppBlockchainWithSharedState(t, localDir, sharedAddr, sharedPubHex, receiverAddr, receiverPubHex)
	peer := mustAppBlockchainWithSharedState(t, peerDir, sharedAddr, sharedPubHex, receiverAddr, receiverPubHex)

	localAddr := sharedAddr
	peerAddr := sharedAddr
	localPriv := sharedPriv
	peerPriv := sharedPriv
	localPubHex := sharedPubHex
	peerPubHex := sharedPubHex
	localReceiver := receiverAddr
	peerReceiver := receiverAddr

	localTx := mustAppSignedTx(t, localAddr, localReceiver, 0, 10, 1, 1001, localPubHex, localPriv)
	if err := local.SubmitTransaction(localTx); err != nil {
		t.Fatalf("local SubmitTransaction failed: %v", err)
	}
	if _, err := local.MinePending(localAddr); err != nil {
		t.Fatalf("local MinePending failed: %v", err)
	}

	peerTx := mustAppSignedTx(t, peerAddr, peerReceiver, 0, 20, 1, 1001, peerPubHex, peerPriv)
	if err := peer.SubmitTransaction(peerTx); err != nil {
		t.Fatalf("peer SubmitTransaction failed: %v", err)
	}
	if _, err := peer.MinePending(peerAddr); err != nil {
		t.Fatalf("peer MinePending failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux := http.NewServeMux()
		mux.HandleFunc("/chain/info", rpc.ChainInfoHandler(peer))
		mux.HandleFunc("/blocks/height", rpc.BlockByHeightHandler(peer))
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	syncer := app.NewBlockSyncService(local, []string{server.URL}, "", time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		syncer.Start(ctx)
	}()

	time.Sleep(1200 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("block sync goroutine did not stop in time")
	}

	height, err := local.Height()
	if err != nil {
		t.Fatalf("local Height failed: %v", err)
	}
	if height != 1 {
		t.Fatalf("expected local height to remain 1, got %d", height)
	}
}

func TestBroadcastRawTransaction_SkipsSelfPeer(t *testing.T) {
	dir := t.TempDir()
	_, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, _ := mustAppBlockchain(t, dir)

	transaction := mustAppSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)
	rawBody, err := json.Marshal(transaction)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	peersPath := dir + "\\peers.json"
	peersJSON := `{"peers":["http://127.0.0.1:9999"]}`
	if err := os.WriteFile(peersPath, []byte(peersJSON), 0o600); err != nil {
		t.Fatalf("WriteFile peers failed: %v", err)
	}

	app.BroadcastRawTransaction(peersPath, "http://127.0.0.1:9999", rawBody)
}

func TestBroadcastHeader_PreventsRebroadcastLoopSignal(t *testing.T) {
	if app.BroadcastHeader != "X-Sila-Broadcasted" {
		t.Fatalf("unexpected broadcast header: %s", app.BroadcastHeader)
	}
}

func TestSyncStatusHandlers_ReturnData(t *testing.T) {
	dir := t.TempDir()
	bc, _, _, _, _, _, _, _ := mustAppBlockchain(t, dir)

	req := httptest.NewRequest(http.MethodGet, "/sync/status", nil)
	rr := httptest.NewRecorder()

	handler := rpc.SyncStatusHandler(bc)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestBlockSync_RestartFollowerFromPersistedState(t *testing.T) {
	leaderDir := t.TempDir()
	followerDir := t.TempDir()

	leader, leaderAddr, leaderPriv, leaderPubHex, receiverAddr, _, receiverPubHex, validatorSet := mustAppBlockchain(t, leaderDir)

	follower, err := chain.NewBlockchain(followerDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain follower failed: %v", err)
	}
	if _, err := follower.RegisterAccount(leaderAddr, leaderPubHex); err != nil {
		t.Fatalf("RegisterAccount follower sender failed: %v", err)
	}
	if _, err := follower.RegisterAccount(receiverAddr, receiverPubHex); err != nil {
		t.Fatalf("RegisterAccount follower receiver failed: %v", err)
	}
	if err := follower.Faucet(leaderAddr, 1000); err != nil {
		t.Fatalf("Faucet follower sender failed: %v", err)
	}

	tx1 := mustAppSignedTx(t, leaderAddr, receiverAddr, 0, 10, 1, 1001, leaderPubHex, leaderPriv)
	if err := leader.SubmitTransaction(tx1); err != nil {
		t.Fatalf("leader SubmitTransaction failed: %v", err)
	}
	if _, err := leader.MinePending(leaderAddr); err != nil {
		t.Fatalf("leader MinePending failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux := http.NewServeMux()
		mux.HandleFunc("/chain/info", rpc.ChainInfoHandler(leader))
		mux.HandleFunc("/blocks/height", rpc.BlockByHeightHandler(leader))
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	syncer := app.NewBlockSyncService(follower, []string{server.URL}, "", time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		syncer.Start(ctx)
	}()

	time.Sleep(1200 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("block sync goroutine did not stop in time")
	}

	reloadedFollower, err := chain.NewBlockchain(followerDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("reloaded follower failed: %v", err)
	}

	height, err := reloadedFollower.Height()
	if err != nil {
		t.Fatalf("reloaded follower Height failed: %v", err)
	}
	if height != 1 {
		t.Fatalf("expected reloaded follower height 1, got %d", height)
	}
}

func TestBlockSync_PeerPolicyBansFailingPeer(t *testing.T) {
	dir := t.TempDir()
	bc, _, _, _, _, _, _, _ := mustAppBlockchain(t, dir)

	syncer := app.NewBlockSyncService(bc, []string{
		"http://127.0.0.1:65530",
		"http://127.0.0.1:65530/",
	}, "", time.Second)

	syncer.SetPeerBanPolicy(2, time.Minute)

	syncer.RunSyncOnceForTest()
	if got := syncer.PeerFailureCount("http://127.0.0.1:65530"); got != 1 {
		t.Fatalf("expected failure count 1, got %d", got)
	}

	syncer.RunSyncOnceForTest()
	if !syncer.PeerIsBanned("http://127.0.0.1:65530", time.Now()) {
		t.Fatalf("expected peer to be banned after repeated failures")
	}
}

func TestBlockSync_PeerPolicyResetsAfterSuccess(t *testing.T) {
	leaderDir := t.TempDir()
	followerDir := t.TempDir()

	leader, leaderAddr, leaderPriv, leaderPubHex, receiverAddr, _, receiverPubHex, validatorSet := mustAppBlockchain(t, leaderDir)
	follower, err := chain.NewBlockchain(followerDir, validatorSet, 1)
	if err != nil {
		t.Fatalf("NewBlockchain follower failed: %v", err)
	}
	if _, err := follower.RegisterAccount(leaderAddr, leaderPubHex); err != nil {
		t.Fatalf("RegisterAccount follower sender failed: %v", err)
	}
	if _, err := follower.RegisterAccount(receiverAddr, receiverPubHex); err != nil {
		t.Fatalf("RegisterAccount follower receiver failed: %v", err)
	}
	if err := follower.Faucet(leaderAddr, 1000); err != nil {
		t.Fatalf("Faucet follower sender failed: %v", err)
	}

	tx1 := mustAppSignedTx(t, leaderAddr, receiverAddr, 0, 10, 1, 1001, leaderPubHex, leaderPriv)
	if err := leader.SubmitTransaction(tx1); err != nil {
		t.Fatalf("leader SubmitTransaction failed: %v", err)
	}
	if _, err := leader.MinePending(leaderAddr); err != nil {
		t.Fatalf("leader MinePending failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux := http.NewServeMux()
		mux.HandleFunc("/chain/info", rpc.ChainInfoHandler(leader))
		mux.HandleFunc("/blocks/height", rpc.BlockByHeightHandler(leader))
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	syncer := app.NewBlockSyncService(follower, []string{
		"http://127.0.0.1:65530",
		server.URL,
		server.URL + "/",
	}, "", time.Second)

	syncer.SetPeerBanPolicy(1, time.Minute)

	syncer.RunSyncOnceForTest()

	if got := syncer.PeerFailureCount(server.URL); got != 0 {
		t.Fatalf("expected successful peer failure count reset to 0, got %d", got)
	}

	height, err := follower.Height()
	if err != nil {
		t.Fatalf("follower Height failed: %v", err)
	}
	if height != 1 {
		t.Fatalf("expected follower height 1 after successful sync, got %d", height)
	}
}

func TestBlockSync_PeerPolicyPersistence_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	bc, _, _, _, _, _, _, _ := mustAppBlockchain(t, dir)

	statePath := dir + "\\peer_policy.json"

	syncer := app.NewBlockSyncService(bc, []string{
		"http://127.0.0.1:65530",
	}, "", time.Second)

	if err := syncer.SetPeerPolicyPath(statePath); err != nil {
		t.Fatalf("SetPeerPolicyPath failed: %v", err)
	}

	syncer.SetPeerBanPolicy(2, time.Minute)

	syncer.RunSyncOnceForTest()
	syncer.RunSyncOnceForTest()

	if !syncer.PeerIsBanned("http://127.0.0.1:65530", time.Now()) {
		t.Fatalf("expected peer banned before reload")
	}

	reloaded := app.NewBlockSyncService(bc, []string{
		"http://127.0.0.1:65530",
	}, "", time.Second)

	if err := reloaded.SetPeerPolicyPath(statePath); err != nil {
		t.Fatalf("reloaded SetPeerPolicyPath failed: %v", err)
	}

	if got := reloaded.PeerFailureCount("http://127.0.0.1:65530"); got < 2 {
		t.Fatalf("expected persisted failures >= 2, got %d", got)
	}

	if !reloaded.PeerIsBanned("http://127.0.0.1:65530", time.Now()) {
		t.Fatalf("expected persisted banned peer after reload")
	}
}

func TestBlockSync_PeerPolicyPersistence_IgnoresMissingFile(t *testing.T) {
	dir := t.TempDir()
	bc, _, _, _, _, _, _, _ := mustAppBlockchain(t, dir)

	syncer := app.NewBlockSyncService(bc, nil, "", time.Second)
	err := syncer.SetPeerPolicyPath(dir + "\\missing_peer_policy.json")
	if err != nil {
		t.Fatalf("expected nil for missing peer policy file, got %v", err)
	}
}

func TestLoadPeersFile_NormalizesAndFiltersSelf(t *testing.T) {
	dir := t.TempDir()
	path := dir + "\\peers.json"

	raw := []byte(`{
  "peers": [
    "127.0.0.1:8081",
    "http://127.0.0.1:8081/",
    "http://127.0.0.1:8082",
    "http://127.0.0.1:8080"
  ]
}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	peers, err := app.LoadPeersFile(path, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("LoadPeersFile failed: %v", err)
	}

	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d (%v)", len(peers), peers)
	}
}

func TestSavePeersFile_WritesNormalizedUniquePeers(t *testing.T) {
	dir := t.TempDir()
	path := dir + "\\peers.json"

	err := app.SavePeersFile(path, []string{
		"127.0.0.1:8081",
		"http://127.0.0.1:8081/",
		"http://127.0.0.1:8082",
		"http://127.0.0.1:8080",
	}, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("SavePeersFile failed: %v", err)
	}

	peers, err := app.LoadPeersFile(path, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("LoadPeersFile failed: %v", err)
	}

	if len(peers) != 2 {
		t.Fatalf("expected 2 peers after save/load, got %d (%v)", len(peers), peers)
	}
}

func TestBlockSync_SetPeersPath_LoadsAndMergesPeers(t *testing.T) {
	dir := t.TempDir()
	peersPath := dir + "\\peers.json"

	err := app.SavePeersFile(peersPath, []string{
		"http://127.0.0.1:9001",
		"http://127.0.0.1:9002",
	}, "")
	if err != nil {
		t.Fatalf("SavePeersFile failed: %v", err)
	}

	bc, _, _, _, _, _, _, _ := mustAppBlockchain(t, dir)
	syncer := app.NewBlockSyncService(bc, []string{
		"http://127.0.0.1:9002",
		"http://127.0.0.1:9003",
	}, "http://127.0.0.1:9000", time.Second)

	if err := syncer.SetPeersPath(peersPath); err != nil {
		t.Fatalf("SetPeersPath failed: %v", err)
	}

	peers := syncer.Peers()
	if len(peers) != 3 {
		t.Fatalf("expected 3 merged peers, got %d (%v)", len(peers), peers)
	}
}

func TestTxBroadcaster_BroadcastsToActivePeers(t *testing.T) {
	dir := t.TempDir()
	_, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, _ := mustAppBlockchain(t, dir)

	var hits1 int
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/tx/send" {
			t.Fatalf("expected /tx/send, got %s", r.URL.Path)
		}
		hits1++
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	var hits2 int
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits2++
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	transaction := mustAppSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	b := app.NewTxBroadcaster("")
	policy := app.NewPeerPolicy()

	b.BroadcastTransaction([]string{server1.URL, server2.URL}, policy, transaction)

	if hits1 != 1 || hits2 != 1 {
		t.Fatalf("expected both peers hit once, got hits1=%d hits2=%d", hits1, hits2)
	}
}

func TestTxBroadcaster_DoesNotRebroadcastSeenTransaction(t *testing.T) {
	dir := t.TempDir()
	_, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, _ := mustAppBlockchain(t, dir)

	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transaction := mustAppSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	b := app.NewTxBroadcaster("")
	policy := app.NewPeerPolicy()

	b.BroadcastTransaction([]string{server.URL}, policy, transaction)
	b.BroadcastTransaction([]string{server.URL}, policy, transaction)

	if hits != 1 {
		t.Fatalf("expected one broadcast only, got %d", hits)
	}
}

func TestTxBroadcaster_SkipsSelfPeer(t *testing.T) {
	dir := t.TempDir()
	_, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, _ := mustAppBlockchain(t, dir)

	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transaction := mustAppSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	b := app.NewTxBroadcaster(server.URL)
	policy := app.NewPeerPolicy()

	b.BroadcastTransaction([]string{server.URL}, policy, transaction)

	if hits != 0 {
		t.Fatalf("expected self peer skipped, got %d hits", hits)
	}
}

func TestTxBroadcaster_ReportsFailureToPeerPolicy(t *testing.T) {
	dir := t.TempDir()
	_, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, _ := mustAppBlockchain(t, dir)

	transaction := mustAppSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	b := app.NewTxBroadcaster("")
	policy := app.NewPeerPolicy()

	peer := "http://127.0.0.1:65530"
	b.BroadcastTransaction([]string{peer}, policy, transaction)

	if policy.FailureCount(peer) == 0 {
		t.Fatalf("expected failure count > 0")
	}
}

func TestTxBroadcaster_ReportsSuccessToPeerPolicy(t *testing.T) {
	dir := t.TempDir()
	_, senderAddr, senderPriv, senderPubHex, receiverAddr, _, _, _ := mustAppBlockchain(t, dir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transaction := mustAppSignedTx(t, senderAddr, receiverAddr, 0, 10, 1, 1001, senderPubHex, senderPriv)

	b := app.NewTxBroadcaster("")
	policy := app.NewPeerPolicy()

	peer := server.URL
	policy.ReportFailure(peer, time.Now())

	b.BroadcastTransaction([]string{peer}, policy, transaction)

	if policy.FailureCount(peer) != 0 {
		t.Fatalf("expected success to reset failure count, got %d", policy.FailureCount(peer))
	}
}
