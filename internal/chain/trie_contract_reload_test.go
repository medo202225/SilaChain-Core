package chain

import (
	"testing"

	"silachain/internal/core/state"
	pkgtypes "silachain/pkg/types"
)

func TestNewBlockchain_TrieContractStorageCommitment_ReloadPreservesStorageRoot(t *testing.T) {
	dataDir := t.TempDir()

	bc1, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain first: %v", err)
	}

	contract := pkgtypes.Address("SILA_contract_reload_test")
	_, _, err = bc1.DeployContract(contract, "contract-code", 0)
	if err != nil {
		t.Fatalf("deploy contract: %v", err)
	}

	if err := bc1.SetContractStorage(contract, "hello", "world"); err != nil {
		t.Fatalf("set contract storage: %v", err)
	}

	root1, err := bc1.GetContractStorageRoot(contract)
	if err != nil {
		t.Fatalf("get storage root first: %v", err)
	}
	if root1 == "" {
		t.Fatalf("expected non-empty root1")
	}

	bc2, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain second: %v", err)
	}

	root2, err := bc2.GetContractStorageRoot(contract)
	if err != nil {
		t.Fatalf("get storage root second: %v", err)
	}
	if root2 == "" {
		t.Fatalf("expected non-empty root2")
	}

	if root1 != root2 {
		t.Fatalf("expected equal storage roots across reload: got=%s want=%s", root2, root1)
	}
}
