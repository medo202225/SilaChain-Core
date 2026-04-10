package chain

import (
	"os"
	"testing"

	"silachain/internal/core/state"
	"silachain/internal/storage"
	pkgtypes "silachain/pkg/types"
)

func TestNewBlockchain_TrieStateCommitment_ReloadPreservesStateRoot(t *testing.T) {
	dataDir := t.TempDir()

	bc1, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain first: %v", err)
	}

	addr := pkgtypes.Address("SILA_reload_test_account")
	if _, err := bc1.RegisterAccount(addr, "reload-pubkey"); err != nil {
		t.Fatalf("register account: %v", err)
	}
	if err := bc1.Faucet(addr, 12345); err != nil {
		t.Fatalf("faucet: %v", err)
	}

	root1, err := bc1.stateCommitment.ComputeStateRoot(bc1.accounts)
	if err != nil {
		t.Fatalf("compute state root first: %v", err)
	}
	if root1 == "" {
		t.Fatalf("expected non-empty first root")
	}

	db := storage.NewDB(dataDir)
	path := db.Path("state_trie_nodes.json")

	info1, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat trie node file after first boot: %v", err)
	}
	if info1.Size() == 0 {
		t.Fatalf("expected non-empty trie node file after first boot")
	}

	bc2, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain second: %v", err)
	}

	root2, err := bc2.stateCommitment.ComputeStateRoot(bc2.accounts)
	if err != nil {
		t.Fatalf("compute state root second: %v", err)
	}
	if root2 == "" {
		t.Fatalf("expected non-empty second root")
	}

	if root1 != root2 {
		t.Fatalf("expected equal state roots across reload: got=%s want=%s", root2, root1)
	}

	info2, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat trie node file after reload: %v", err)
	}
	if info2.Size() == 0 {
		t.Fatalf("expected non-empty trie node file after reload")
	}
}
