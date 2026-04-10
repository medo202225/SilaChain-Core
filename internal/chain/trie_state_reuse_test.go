package chain

import (
	"testing"

	"silachain/internal/core/state"
	pkgtypes "silachain/pkg/types"
)

func TestNewBlockchain_TrieStateCommitment_LoadsPersistedRootOnReload(t *testing.T) {
	dataDir := t.TempDir()

	bc1, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain first: %v", err)
	}

	addr := pkgtypes.Address("SILA_reuse_test_account")
	if _, err := bc1.RegisterAccount(addr, "reuse-pubkey"); err != nil {
		t.Fatalf("register account: %v", err)
	}
	if err := bc1.Faucet(addr, 777); err != nil {
		t.Fatalf("faucet: %v", err)
	}

	bc2, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain second: %v", err)
	}

	trieCommitment, ok := bc2.stateCommitment.(*state.TrieStateCommitment)
	if !ok || trieCommitment == nil {
		t.Fatalf("expected trie state commitment")
	}

	persistedRoot, err := trieCommitment.LoadPersistedStateRoot()
	if err != nil {
		t.Fatalf("load persisted root: %v", err)
	}
	if persistedRoot == "" {
		t.Fatalf("expected non-empty persisted root")
	}

	recomputedRoot, err := bc2.stateCommitment.ComputeStateRoot(bc2.accounts)
	if err != nil {
		t.Fatalf("compute recomputed root: %v", err)
	}
	if recomputedRoot == "" {
		t.Fatalf("expected non-empty recomputed root")
	}

	if persistedRoot != recomputedRoot {
		t.Fatalf("persisted root mismatch: got=%s want=%s", persistedRoot, recomputedRoot)
	}
}
