package chain

import (
	"testing"

	"silachain/internal/core/state"
	pkgtypes "silachain/pkg/types"
)

func TestNewBlockchain_TrieContractStorageCommitment_UsesPersistedRootAsReloadSource(t *testing.T) {
	dataDir := t.TempDir()

	bc1, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain first: %v", err)
	}

	contract := pkgtypes.Address("SILA_contract_canonical_reload")
	_, _, err = bc1.DeployContract(contract, "contract-code", 0)
	if err != nil {
		t.Fatalf("deploy contract: %v", err)
	}

	if err := bc1.SetContractStorage(contract, "k", "v"); err != nil {
		t.Fatalf("set contract storage: %v", err)
	}

	root1, err := bc1.GetContractStorageRoot(contract)
	if err != nil {
		t.Fatalf("get root first: %v", err)
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
		t.Fatalf("get root second: %v", err)
	}
	if root2 == "" {
		t.Fatalf("expected non-empty root2")
	}

	if root2 != root1 {
		t.Fatalf("expected persisted contract storage root to match prior root: got=%s want=%s", root2, root1)
	}
}
