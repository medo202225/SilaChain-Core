package storage

import (
	"os"
	"testing"
)

func TestStateStore_SaveLoadTrieNodes_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)
	store := NewStateStore(db)

	input := map[string]PersistedTrieNode{
		"hash-b": {
			Hash:  "hash-b",
			Left:  "left-b",
			Right: "right-b",
		},
		"hash-a": {
			Hash: "hash-a",
			Leaf: "leaf-a",
		},
	}

	if err := store.SaveTrieNodes(input); err != nil {
		t.Fatalf("save trie nodes: %v", err)
	}

	loaded, err := store.LoadTrieNodes()
	if err != nil {
		t.Fatalf("load trie nodes: %v", err)
	}

	if len(loaded) != len(input) {
		t.Fatalf("unexpected node count: got=%d want=%d", len(loaded), len(input))
	}

	for hash, want := range input {
		got, ok := loaded[hash]
		if !ok {
			t.Fatalf("missing node %s", hash)
		}
		if got.Hash != want.Hash || got.Left != want.Left || got.Right != want.Right || got.Leaf != want.Leaf {
			t.Fatalf("node mismatch for %s: got=%+v want=%+v", hash, got, want)
		}
	}
}

func TestStateStore_LoadTrieNodes_MissingFile(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)
	store := NewStateStore(db)

	loaded, err := store.LoadTrieNodes()
	if err != nil {
		t.Fatalf("load missing trie nodes: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected empty trie node set, got=%d", len(loaded))
	}
}

func TestStateStore_DeleteTrieNodes_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(dir)
	store := NewStateStore(db)

	input := map[string]PersistedTrieNode{
		"hash-a": {
			Hash: "hash-a",
			Leaf: "leaf-a",
		},
	}

	if err := store.SaveTrieNodes(input); err != nil {
		t.Fatalf("save trie nodes: %v", err)
	}

	if err := store.DeleteTrieNodes(); err != nil {
		t.Fatalf("delete trie nodes: %v", err)
	}

	if _, err := os.Stat(db.Path("state_trie_nodes.json")); !os.IsNotExist(err) {
		t.Fatalf("expected trie node file to be removed, got err=%v", err)
	}
}
