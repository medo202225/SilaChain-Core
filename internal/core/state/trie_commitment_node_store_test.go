package state

import (
	"testing"

	"silachain/internal/accounts"
	"silachain/pkg/types"
)

type recordingTrieNodeStore struct {
	called bool
	nodes  map[string]TrieNodeRecord
}

func (s *recordingTrieNodeStore) SaveTrieNodes(nodes map[string]TrieNodeRecord) error {
	s.called = true
	s.nodes = nodes
	return nil
}

func TestTrieStateCommitment_PersistsNodesToNodeStore(t *testing.T) {
	accountManager := accounts.NewManager()

	acc := &accounts.Account{
		Address:     types.Address("SILA_test_account"),
		PublicKey:   "pubkey",
		Balance:     100,
		Nonce:       1,
		CodeHash:    "",
		StorageRoot: "",
	}

	if err := accountManager.Set(acc); err != nil {
		t.Fatalf("set account: %v", err)
	}

	store := &recordingTrieNodeStore{}
	commitment := NewTrieStateCommitment().WithNodeStore(store)

	root, err := commitment.ComputeStateRoot(accountManager)
	if err != nil {
		t.Fatalf("compute state root: %v", err)
	}

	if root == "" {
		t.Fatalf("expected non-empty root")
	}

	if !store.called {
		t.Fatalf("expected node store to be called")
	}

	if len(store.nodes) == 0 {
		t.Fatalf("expected persisted trie nodes")
	}
}
