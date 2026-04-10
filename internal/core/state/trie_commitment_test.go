package state

import (
	"testing"

	"silachain/internal/accounts"
	"silachain/pkg/types"
)

func TestTrieStateCommitment_EmptyManager(t *testing.T) {
	commitment := NewTrieStateCommitment()

	root, err := commitment.ComputeStateRoot(nil)
	if err != nil {
		t.Fatalf("compute state root: %v", err)
	}
	if root != "" {
		t.Fatalf("expected empty root, got %s", root)
	}
}

func TestTrieStateCommitment_Deterministic(t *testing.T) {
	accountManagerA := accounts.NewManager()
	accountManagerB := accounts.NewManager()

	acc1a := accounts.NewAccount(types.Address("SILA_b"), "pub-b")
	acc1a.Balance = 20
	acc1a.Nonce = 2
	acc1a.SetCodeHash("code-b")
	acc1a.SetStorageRoot("storage-b")

	acc2a := accounts.NewAccount(types.Address("SILA_a"), "pub-a")
	acc2a.Balance = 10
	acc2a.Nonce = 1
	acc2a.SetCodeHash("code-a")
	acc2a.SetStorageRoot("storage-a")

	acc1b := accounts.NewAccount(types.Address("SILA_a"), "pub-a")
	acc1b.Balance = 10
	acc1b.Nonce = 1
	acc1b.SetCodeHash("code-a")
	acc1b.SetStorageRoot("storage-a")

	acc2b := accounts.NewAccount(types.Address("SILA_b"), "pub-b")
	acc2b.Balance = 20
	acc2b.Nonce = 2
	acc2b.SetCodeHash("code-b")
	acc2b.SetStorageRoot("storage-b")

	if err := accountManagerA.Set(acc1a); err != nil {
		t.Fatalf("set acc1a: %v", err)
	}
	if err := accountManagerA.Set(acc2a); err != nil {
		t.Fatalf("set acc2a: %v", err)
	}
	if err := accountManagerB.Set(acc1b); err != nil {
		t.Fatalf("set acc1b: %v", err)
	}
	if err := accountManagerB.Set(acc2b); err != nil {
		t.Fatalf("set acc2b: %v", err)
	}

	rootA, err := NewTrieStateCommitment().ComputeStateRoot(accountManagerA)
	if err != nil {
		t.Fatalf("rootA: %v", err)
	}
	rootB, err := NewTrieStateCommitment().ComputeStateRoot(accountManagerB)
	if err != nil {
		t.Fatalf("rootB: %v", err)
	}

	if rootA == "" || rootB == "" {
		t.Fatalf("expected non-empty trie roots")
	}
	if rootA != rootB {
		t.Fatalf("expected deterministic equal roots: got %s want %s", rootA, rootB)
	}
}

func TestTrieStateCommitment_DiffersWhenStateChanges(t *testing.T) {
	accountManager := accounts.NewManager()

	acc := accounts.NewAccount(types.Address("SILA_test"), "pub")
	acc.Balance = 10
	acc.Nonce = 1

	if err := accountManager.Set(acc); err != nil {
		t.Fatalf("set account: %v", err)
	}

	commitment := NewTrieStateCommitment()

	root1, err := commitment.ComputeStateRoot(accountManager)
	if err != nil {
		t.Fatalf("root1: %v", err)
	}

	acc.Balance = 11

	root2, err := commitment.ComputeStateRoot(accountManager)
	if err != nil {
		t.Fatalf("root2: %v", err)
	}

	if root1 == root2 {
		t.Fatalf("expected root to change when account state changes")
	}
}

func TestTrieContractStorageCommitment_Deterministic(t *testing.T) {
	storageA := NewContractStorage()
	storageB := NewContractStorage()
	contract := types.Address("SILA_contract")

	storageA.Set(contract, "b", "2")
	storageA.Set(contract, "a", "1")

	storageB.Set(contract, "a", "1")
	storageB.Set(contract, "b", "2")

	commitment := NewTrieContractStorageCommitment()

	rootA, err := commitment.ComputeStorageRoot(storageA, contract)
	if err != nil {
		t.Fatalf("rootA: %v", err)
	}
	rootB, err := commitment.ComputeStorageRoot(storageB, contract)
	if err != nil {
		t.Fatalf("rootB: %v", err)
	}

	if rootA == "" || rootB == "" {
		t.Fatalf("expected non-empty storage roots")
	}
	if rootA != rootB {
		t.Fatalf("expected deterministic equal storage roots: got %s want %s", rootA, rootB)
	}
}

func TestTrieContractStorageCommitment_DiffersWhenStorageChanges(t *testing.T) {
	storage := NewContractStorage()
	contract := types.Address("SILA_contract")

	storage.Set(contract, "a", "1")

	commitment := NewTrieContractStorageCommitment()

	root1, err := commitment.ComputeStorageRoot(storage, contract)
	if err != nil {
		t.Fatalf("root1: %v", err)
	}

	storage.Set(contract, "a", "2")

	root2, err := commitment.ComputeStorageRoot(storage, contract)
	if err != nil {
		t.Fatalf("root2: %v", err)
	}

	if root1 == root2 {
		t.Fatalf("expected storage root to change when storage changes")
	}
}
