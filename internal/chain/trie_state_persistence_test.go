package chain

import (
	"os"
	"testing"

	"silachain/internal/core/state"
	"silachain/internal/storage"
)

func TestNewBlockchain_TrieStateCommitment_PersistsTrieNodes(t *testing.T) {
	dataDir := t.TempDir()

	_, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}

	db := storage.NewDB(dataDir)
	path := db.Path("state_trie_nodes.json")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat trie node file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected non-empty trie node file")
	}
}
