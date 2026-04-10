package state

import (
	"sync"

	"silachain/pkg/crypto"
	"silachain/pkg/types"
)

type ContractCodeRegistry struct {
	mu   sync.RWMutex
	code map[types.Address]string
}

func NewContractCodeRegistry() *ContractCodeRegistry {
	return &ContractCodeRegistry{
		code: make(map[types.Address]string),
	}
}

func (r *ContractCodeRegistry) Set(address types.Address, code string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.code[address] = code
}

func (r *ContractCodeRegistry) Get(address types.Address) (string, bool) {
	if r == nil {
		return "", false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	code, ok := r.code[address]
	return code, ok
}

func (r *ContractCodeRegistry) Delete(address types.Address) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.code, address)
}

func (r *ContractCodeRegistry) All() map[types.Address]string {
	if r == nil {
		return map[types.Address]string{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[types.Address]string, len(r.code))
	for k, v := range r.code {
		out[k] = v
	}
	return out
}

func (r *ContractCodeRegistry) Load(code map[types.Address]string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.code = make(map[types.Address]string, len(code))
	for k, v := range code {
		r.code[k] = v
	}
}

func ComputeCodeHash(code string) (types.Hash, error) {
	if code == "" {
		return "", nil
	}

	sum, err := crypto.HashJSON(map[string]string{
		"code": code,
	})
	if err != nil {
		return "", err
	}

	return types.Hash(sum), nil
}
