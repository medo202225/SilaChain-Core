package state

import (
	"fmt"
	"sort"

	"silachain/internal/accounts"
	"silachain/pkg/crypto"
	"silachain/pkg/types"
)

type TrieNodeRecord struct {
	Hash  string `json:"hash"`
	Left  string `json:"left,omitempty"`
	Right string `json:"right,omitempty"`
	Leaf  string `json:"leaf,omitempty"`
}

type TrieNodeStore interface {
	SaveTrieNodes(nodes map[string]TrieNodeRecord) error
}

type TrieNodeLoader interface {
	LoadTrieNodes() (map[string]TrieNodeRecord, error)
}

type ContractTrieNodeStore interface {
	SaveContractTrieNodes(contract string, nodes map[string]TrieNodeRecord) error
	LoadContractTrieNodes(contract string) (map[string]TrieNodeRecord, error)
}

type TrieStateCommitment struct {
	nodeStore TrieNodeStore
}

type TrieContractStorageCommitment struct {
	nodeStore ContractTrieNodeStore
}

type trieLeaf struct {
	Key   string
	Value string
}

type trieNode struct {
	Left  string `json:"left,omitempty"`
	Right string `json:"right,omitempty"`
	Leaf  string `json:"leaf,omitempty"`
}

func NewTrieStateCommitment() *TrieStateCommitment {
	return &TrieStateCommitment{}
}

func NewTrieContractStorageCommitment() *TrieContractStorageCommitment {
	return &TrieContractStorageCommitment{}
}

func (c *TrieContractStorageCommitment) WithContractNodeStore(store ContractTrieNodeStore) *TrieContractStorageCommitment {
	if c == nil {
		return nil
	}
	c.nodeStore = store
	return c
}

func (c *TrieStateCommitment) WithNodeStore(store TrieNodeStore) *TrieStateCommitment {
	if c == nil {
		return nil
	}
	c.nodeStore = store
	return c
}
func (c *TrieStateCommitment) LoadPersistedStateRoot() (types.Hash, error) {
	if c == nil || c.nodeStore == nil {
		return "", nil
	}

	loader, ok := c.nodeStore.(TrieNodeLoader)
	if !ok || loader == nil {
		return "", nil
	}

	nodes, err := loader.LoadTrieNodes()
	if err != nil {
		return "", err
	}

	return inferPersistedTrieRoot(nodes)
}

func inferPersistedTrieRoot(nodes map[string]TrieNodeRecord) (types.Hash, error) {
	if len(nodes) == 0 {
		return "", nil
	}

	referenced := make(map[string]struct{})
	for _, node := range nodes {
		if node.Left != "" {
			referenced[node.Left] = struct{}{}
		}
		if node.Right != "" {
			referenced[node.Right] = struct{}{}
		}
	}

	roots := make([]string, 0)
	for hash := range nodes {
		if _, ok := referenced[hash]; !ok {
			roots = append(roots, hash)
		}
	}

	sort.Strings(roots)

	if len(roots) == 0 {
		return "", fmt.Errorf("persisted trie root not found")
	}
	if len(roots) > 1 {
		return "", fmt.Errorf("multiple persisted trie roots found: %d", len(roots))
	}

	return types.Hash(roots[0]), nil
}

func (c *TrieStateCommitment) ComputeStateRoot(manager *accounts.Manager) (types.Hash, error) {
	if manager == nil {
		if c != nil && c.nodeStore != nil {
			if err := c.nodeStore.SaveTrieNodes(map[string]TrieNodeRecord{}); err != nil {
				return "", err
			}
		}
		return "", nil
	}

	raw := manager.All()
	leaves := make([]trieLeaf, 0, len(raw))

	for _, acc := range raw {
		value, err := crypto.HashJSON(stateCommitmentAccount{
			Address:     acc.Address,
			PublicKey:   acc.PublicKey,
			Balance:     acc.Balance,
			Nonce:       acc.Nonce,
			CodeHash:    acc.CodeHash,
			StorageRoot: acc.StorageRoot,
		})
		if err != nil {
			return "", err
		}

		leaves = append(leaves, trieLeaf{
			Key:   string(acc.Address),
			Value: value,
		})
	}

	root, nodes, err := buildTrieLikeRoot(leaves)
	if err != nil {
		return "", err
	}

	if c != nil && c.nodeStore != nil {
		if err := c.nodeStore.SaveTrieNodes(nodes); err != nil {
			return "", err
		}
	}

	return root, nil
}

func (c *TrieContractStorageCommitment) ComputeStorageRoot(storage *ContractStorage, contract types.Address) (types.Hash, error) {
	if storage == nil {
		return "", nil
	}

	allStorage := storage.All()
	slots := allStorage[contract]
	leaves := make([]trieLeaf, 0, len(slots))

	for key, value := range slots {
		encoded, err := crypto.HashJSON(map[string]string{
			"key":   key,
			"value": value,
		})
		if err != nil {
			return "", err
		}

		leaves = append(leaves, trieLeaf{
			Key:   key,
			Value: encoded,
		})
	}

	root, nodes, err := buildTrieLikeRoot(leaves)
	if err != nil {
		return "", err
	}

	if c != nil && c.nodeStore != nil {
		if err := c.nodeStore.SaveContractTrieNodes(string(contract), nodes); err != nil {
			return "", err
		}
	}

	return root, nil
}

func buildTrieLikeRoot(leaves []trieLeaf) (types.Hash, map[string]TrieNodeRecord, error) {
	if len(leaves) == 0 {
		return "", map[string]TrieNodeRecord{}, nil
	}

	sort.Slice(leaves, func(i, j int) bool {
		return leaves[i].Key < leaves[j].Key
	})

	nodes := make(map[string]TrieNodeRecord)

	level := make([]string, 0, len(leaves))
	for _, leaf := range leaves {
		sum, err := crypto.HashJSON(trieNode{
			Leaf: leaf.Key + ":" + leaf.Value,
		})
		if err != nil {
			return "", nil, err
		}

		hash := string(types.Hash(sum))
		nodes[hash] = TrieNodeRecord{
			Hash: hash,
			Leaf: leaf.Key + ":" + leaf.Value,
		}
		level = append(level, hash)
	}

	for len(level) > 1 {
		next := make([]string, 0, (len(level)+1)/2)

		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := ""
			if i+1 < len(level) {
				right = level[i+1]
			}

			sum, err := crypto.HashJSON(trieNode{
				Left:  left,
				Right: right,
			})
			if err != nil {
				return "", nil, err
			}

			hash := string(types.Hash(sum))
			nodes[hash] = TrieNodeRecord{
				Hash:  hash,
				Left:  left,
				Right: right,
			}
			next = append(next, hash)
		}

		level = next
	}

	return types.Hash(level[0]), nodes, nil
}

func (c *TrieContractStorageCommitment) LoadPersistedStorageRoot(contract types.Address) (types.Hash, error) {
	if c == nil || c.nodeStore == nil || contract == "" {
		return "", nil
	}

	nodes, err := c.nodeStore.LoadContractTrieNodes(string(contract))
	if err != nil {
		return "", err
	}

	return inferPersistedTrieRoot(nodes)
}
