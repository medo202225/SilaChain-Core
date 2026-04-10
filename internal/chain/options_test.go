package chain

import (
	"testing"

	"silachain/internal/core/state"
)

func TestNewBlockchain_UsesInjectedTrieStateCommitment(t *testing.T) {
	dataDir := t.TempDir()

	bc, err := NewBlockchain(dataDir, nil, 0, WithStateCommitment(state.NewTrieStateCommitment()))
	if err != nil {
		t.Fatalf("new blockchain with trie commitment: %v", err)
	}

	if bc == nil {
		t.Fatalf("expected blockchain instance")
	}
	if bc.stateCommitment == nil {
		t.Fatalf("expected state commitment to be set")
	}

	if _, ok := bc.stateCommitment.(*state.TrieStateCommitment); !ok {
		t.Fatalf("expected trie state commitment, got %T", bc.stateCommitment)
	}
}

func TestNewBlockchain_DefaultsToTrieStateCommitment(t *testing.T) {
	dataDir := t.TempDir()

	bc, err := NewBlockchain(dataDir, nil, 0)
	if err != nil {
		t.Fatalf("new blockchain default: %v", err)
	}

	if bc == nil {
		t.Fatalf("expected blockchain instance")
	}
	if bc.stateCommitment == nil {
		t.Fatalf("expected default state commitment")
	}

	if _, ok := bc.stateCommitment.(*state.HashStateCommitment); !ok {
		t.Fatalf("expected hash state commitment by default, got %T", bc.stateCommitment)
	}
}
