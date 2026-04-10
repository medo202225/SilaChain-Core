package state

// CANONICAL OWNERSHIP: state commitment interfaces and current hash-based implementations.
// Trie-backed and db-backed state commitments should be introduced behind these interfaces.

import (
	"sort"

	"silachain/internal/accounts"
	"silachain/pkg/crypto"
	"silachain/pkg/types"
)

type StateCommitment interface {
	ComputeStateRoot(manager *accounts.Manager) (types.Hash, error)
}

type ContractStorageCommitment interface {
	ComputeStorageRoot(storage *ContractStorage, contract types.Address) (types.Hash, error)
}

type HashStateCommitment struct{}

type HashContractStorageCommitment struct{}

func NewHashStateCommitment() *HashStateCommitment {
	return &HashStateCommitment{}
}

func NewHashContractStorageCommitment() *HashContractStorageCommitment {
	return &HashContractStorageCommitment{}
}

type stateCommitmentAccount struct {
	Address     types.Address `json:"address"`
	PublicKey   string        `json:"public_key"`
	Balance     types.Amount  `json:"balance"`
	Nonce       types.Nonce   `json:"nonce"`
	CodeHash    types.Hash    `json:"code_hash"`
	StorageRoot types.Hash    `json:"storage_root"`
}

func (c *HashStateCommitment) ComputeStateRoot(manager *accounts.Manager) (types.Hash, error) {
	if manager == nil {
		return "", nil
	}

	raw := manager.All()
	list := make([]stateCommitmentAccount, 0, len(raw))

	for _, acc := range raw {
		list = append(list, stateCommitmentAccount{
			Address:     acc.Address,
			PublicKey:   acc.PublicKey,
			Balance:     acc.Balance,
			Nonce:       acc.Nonce,
			CodeHash:    acc.CodeHash,
			StorageRoot: acc.StorageRoot,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Address < list[j].Address
	})

	sum, err := crypto.HashJSON(list)
	if err != nil {
		return "", err
	}

	return types.Hash(sum), nil
}

func (c *HashContractStorageCommitment) ComputeStorageRoot(storage *ContractStorage, contract types.Address) (types.Hash, error) {
	if storage == nil {
		return "", nil
	}

	return storage.ComputeRoot(contract)
}
