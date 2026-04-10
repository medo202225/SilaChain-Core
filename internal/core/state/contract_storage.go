package state

import (
	"sort"
	"sync"

	"silachain/pkg/crypto"
	"silachain/pkg/types"
)

type ContractStorage struct {
	mu   sync.RWMutex
	data map[types.Address]map[string]string
}

type storageEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func NewContractStorage() *ContractStorage {
	return &ContractStorage{
		data: make(map[types.Address]map[string]string),
	}
}

func (s *ContractStorage) Get(contract types.Address, key string) (string, bool) {
	if s == nil {
		return "", false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.data[contract]
	if !ok {
		return "", false
	}

	v, ok := m[key]
	return v, ok
}

func (s *ContractStorage) Set(contract types.Address, key string, value string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[contract]; !ok {
		s.data[contract] = make(map[string]string)
	}
	s.data[contract][key] = value
}

func (s *ContractStorage) Delete(contract types.Address, key string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.data[contract]
	if !ok {
		return
	}

	delete(m, key)
	if len(m) == 0 {
		delete(s.data, contract)
	}
}

func (s *ContractStorage) Entries(contract types.Address) []storageEntry {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.data[contract]
	if !ok {
		return nil
	}

	out := make([]storageEntry, 0, len(m))
	for k, v := range m {
		out = append(out, storageEntry{
			Key:   k,
			Value: v,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})

	return out
}

func (s *ContractStorage) ComputeRoot(contract types.Address) (types.Hash, error) {
	entries := s.Entries(contract)
	if len(entries) == 0 {
		return "", nil
	}

	sum, err := crypto.HashJSON(entries)
	if err != nil {
		return "", err
	}

	return types.Hash(sum), nil
}

func (s *ContractStorage) All() map[types.Address]map[string]string {
	if s == nil {
		return map[types.Address]map[string]string{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[types.Address]map[string]string, len(s.data))
	for contract, kv := range s.data {
		copyKV := make(map[string]string, len(kv))
		for k, v := range kv {
			copyKV[k] = v
		}
		out[contract] = copyKV
	}
	return out
}

func (s *ContractStorage) Load(data map[types.Address]map[string]string) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[types.Address]map[string]string, len(data))
	for contract, kv := range data {
		copyKV := make(map[string]string, len(kv))
		for k, v := range kv {
			copyKV[k] = v
		}
		s.data[contract] = copyKV
	}
}
