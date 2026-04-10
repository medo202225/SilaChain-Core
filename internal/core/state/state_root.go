package state

// CANONICAL OWNERSHIP: state domain layer for state root and contract/account state transitions.
// Current state root is deterministic hash-based and not yet trie-backed.

import (
	"silachain/internal/accounts"
	"silachain/pkg/types"
)

func ComputeStateRoot(manager *accounts.Manager) (types.Hash, error) {
	return NewHashStateCommitment().ComputeStateRoot(manager)
}
