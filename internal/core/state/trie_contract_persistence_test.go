package state

import (
	"testing"

	"silachain/pkg/types"
)

type contractTrieNodeStoreStub struct {
	nodesByContract map[string]map[string]TrieNodeRecord
}

func (s *contractTrieNodeStoreStub) SaveContractTrieNodes(contract string, nodes map[string]TrieNodeRecord) error {
	if s.nodesByContract == nil {
		s.nodesByContract = make(map[string]map[string]TrieNodeRecord)
	}
	s.nodesByContract[contract] = nodes
	return nil
}

func (s *contractTrieNodeStoreStub) LoadContractTrieNodes(contract string) (map[string]TrieNodeRecord, error) {
	if s.nodesByContract == nil {
		return map[string]TrieNodeRecord{}, nil
	}
	return s.nodesByContract[contract], nil
}

func TestTrieContractStorageCommitment_PersistsAndLoadsStorageRoot(t *testing.T) {
	store := &contractTrieNodeStoreStub{}
	commitment := NewTrieContractStorageCommitment().WithContractNodeStore(store)

	storage := NewContractStorage()
	contract := types.Address("SILA_contract_persist_test")
	storage.Set(contract, "k1", "v1")
	storage.Set(contract, "k2", "v2")

	root1, err := commitment.ComputeStorageRoot(storage, contract)
	if err != nil {
		t.Fatalf("compute storage root: %v", err)
	}
	if root1 == "" {
		t.Fatalf("expected non-empty root1")
	}

	root2, err := commitment.LoadPersistedStorageRoot(contract)
	if err != nil {
		t.Fatalf("load persisted storage root: %v", err)
	}
	if root2 == "" {
		t.Fatalf("expected non-empty root2")
	}

	if root1 != root2 {
		t.Fatalf("expected equal roots: got=%s want=%s", root2, root1)
	}
}
