package storage

import (
	"encoding/json"
	"os"
	"sort"

	"silachain/internal/core/state"
)

type PersistedTrieNode = state.TrieNodeRecord

type StateStore struct {
	db *DB
}

func NewStateStore(db *DB) *StateStore {
	return &StateStore{db: db}
}

func (s *StateStore) path() string {
	return s.db.Path("state_trie_nodes.json")
}

func (s *StateStore) SaveTrieNodes(nodes map[string]PersistedTrieNode) error {
	if s == nil || s.db == nil {
		return nil
	}

	ordered := make([]PersistedTrieNode, 0, len(nodes))
	keys := make([]string, 0, len(nodes))
	for key := range nodes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		node := nodes[key]
		if node.Hash == "" {
			node.Hash = key
		}
		ordered = append(ordered, node)
	}

	data, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path(), data, 0o644)
}

func (s *StateStore) LoadTrieNodes() (map[string]PersistedTrieNode, error) {
	if s == nil || s.db == nil {
		return map[string]PersistedTrieNode{}, nil
	}

	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]PersistedTrieNode{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return map[string]PersistedTrieNode{}, nil
	}

	var ordered []PersistedTrieNode
	if err := json.Unmarshal(data, &ordered); err != nil {
		return nil, err
	}

	out := make(map[string]PersistedTrieNode, len(ordered))
	for _, node := range ordered {
		if node.Hash == "" {
			continue
		}
		out[node.Hash] = node
	}

	return out, nil
}

func (s *StateStore) DeleteTrieNodes() error {
	if s == nil || s.db == nil {
		return nil
	}

	err := os.Remove(s.path())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *StateStore) contractTriePath(contract string) string {
	return s.db.Path("contract_trie_nodes_" + contract + ".json")
}

func (s *StateStore) SaveContractTrieNodes(contract string, nodes map[string]PersistedTrieNode) error {
	if s == nil || s.db == nil || contract == "" {
		return nil
	}

	ordered := make([]PersistedTrieNode, 0, len(nodes))
	keys := make([]string, 0, len(nodes))
	for key := range nodes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		node := nodes[key]
		if node.Hash == "" {
			node.Hash = key
		}
		ordered = append(ordered, node)
	}

	data, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.contractTriePath(contract), data, 0o644)
}

func (s *StateStore) LoadContractTrieNodes(contract string) (map[string]PersistedTrieNode, error) {
	if s == nil || s.db == nil || contract == "" {
		return map[string]PersistedTrieNode{}, nil
	}

	data, err := os.ReadFile(s.contractTriePath(contract))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]PersistedTrieNode{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return map[string]PersistedTrieNode{}, nil
	}

	var ordered []PersistedTrieNode
	if err := json.Unmarshal(data, &ordered); err != nil {
		return nil, err
	}

	out := make(map[string]PersistedTrieNode, len(ordered))
	for _, node := range ordered {
		if node.Hash == "" {
			continue
		}
		out[node.Hash] = node
	}

	return out, nil
}
