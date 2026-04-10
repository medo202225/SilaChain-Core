package chain

import (
	"testing"

	"silachain/internal/core/state"
	pkgtypes "silachain/pkg/types"
)

func TestBlockchain_BlockLifecycle_FinalizesCanonicalRoots(t *testing.T) {
	dataDir := t.TempDir()

	bc, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}

	addr := pkgtypes.Address("SILA_block_lifecycle_account")
	if _, err := bc.RegisterAccount(addr, "block-lifecycle-pubkey"); err != nil {
		t.Fatalf("register account: %v", err)
	}

	if err := bc.Faucet(addr, 1000); err != nil {
		t.Fatalf("faucet: %v", err)
	}

	latest, err := bc.LatestBlock()
	if err != nil {
		t.Fatalf("latest block before finalize: %v", err)
	}

	if err := bc.finalizeBlockRoots(latest); err != nil {
		t.Fatalf("finalize block roots: %v", err)
	}

	if latest.Header.StateRoot == "" {
		t.Fatalf("expected non-empty state root")
	}
	if latest.Header.TxRoot == "" && len(latest.Transactions) > 0 {
		t.Fatalf("expected non-empty tx root")
	}
	if latest.Header.ReceiptRoot == "" && len(latest.Receipts) > 0 {
		t.Fatalf("expected non-empty receipt root")
	}
	if latest.Header.Hash == "" {
		t.Fatalf("expected non-empty block hash")
	}
}
