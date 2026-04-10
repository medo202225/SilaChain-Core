package state

import "testing"

type loadableTrieNodeStore struct {
	nodes map[string]TrieNodeRecord
}

func (s *loadableTrieNodeStore) SaveTrieNodes(nodes map[string]TrieNodeRecord) error {
	s.nodes = nodes
	return nil
}

func (s *loadableTrieNodeStore) LoadTrieNodes() (map[string]TrieNodeRecord, error) {
	return s.nodes, nil
}

func TestTrieStateCommitment_LoadPersistedStateRoot(t *testing.T) {
	store := &loadableTrieNodeStore{
		nodes: map[string]TrieNodeRecord{
			"root-hash": {
				Hash:  "root-hash",
				Left:  "left-hash",
				Right: "right-hash",
			},
			"left-hash": {
				Hash: "left-hash",
				Leaf: "left",
			},
			"right-hash": {
				Hash: "right-hash",
				Leaf: "right",
			},
		},
	}

	root, err := NewTrieStateCommitment().WithNodeStore(store).LoadPersistedStateRoot()
	if err != nil {
		t.Fatalf("load persisted state root: %v", err)
	}

	if root != "root-hash" {
		t.Fatalf("unexpected persisted root: got=%s want=%s", root, "root-hash")
	}
}
