package chain

import (
	"testing"

	"silachain/internal/core/state"
	pkgtypes "silachain/pkg/types"
)

func TestNewBlockchain_TrieStateCommitment_UsesPersistedRootAsReloadSource(t *testing.T) {
	dataDir := t.TempDir()

	bc1, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain first: %v", err)
	}

	addr := pkgtypes.Address("SILA_canonical_reload_account")
	if _, err := bc1.RegisterAccount(addr, "canonical-pubkey"); err != nil {
		t.Fatalf("register account: %v", err)
	}
	if err := bc1.Faucet(addr, 999); err != nil {
		t.Fatalf("faucet: %v", err)
	}

	root1, err := bc1.stateCommitment.ComputeStateRoot(bc1.accounts)
	if err != nil {
		t.Fatalf("compute root first: %v", err)
	}
	if root1 == "" {
		t.Fatalf("expected non-empty root1")
	}

	bc2, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain second: %v", err)
	}

	latest, err := bc2.LatestBlock()
	if err != nil {
		t.Fatalf("latest block: %v", err)
	}

	if latest.Header.StateRoot == "" {
		t.Fatalf("expected non-empty state root on reload")
	}

	if latest.Header.StateRoot != root1 {
		t.Fatalf("expected persisted reload root to match prior root: got=%s want=%s", latest.Header.StateRoot, root1)
	}
}
