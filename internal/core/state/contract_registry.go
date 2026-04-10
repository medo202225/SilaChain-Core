package state

// CANONICAL OWNERSHIP: state domain layer for state root and contract/account state transitions.

import (
	"errors"
	"strings"

	"silachain/internal/accounts"
	"silachain/pkg/types"
)

var ErrNilContractStateManager = errors.New("contract state manager is nil")
var ErrContractAccountRequired = errors.New("contract account required")
var ErrUnsupportedContractMethod = errors.New("unsupported contract method")
var ErrMissingContractStorageKey = errors.New("missing contract storage key")
var ErrMissingContractCode = errors.New("missing contract code")

type ContractCallResult struct {
	Address     types.Address `json:"address"`
	Method      string        `json:"method"`
	Key         string        `json:"key"`
	Value       string        `json:"value,omitempty"`
	Found       bool          `json:"found"`
	Mutated     bool          `json:"mutated"`
	StorageRoot types.Hash    `json:"storage_root"`
	Logs        []types.Event `json:"logs,omitempty"`
}

type ContractRegistry struct {
	state      *Manager
	storage    *ContractStorage
	code       *ContractCodeRegistry
	runtime    *ContractRuntime
	commitment ContractStorageCommitment
}

func NewContractRegistry(stateManager *Manager, storage *ContractStorage) *ContractRegistry {
	if storage == nil {
		storage = NewContractStorage()
	}

	codeRegistry := NewContractCodeRegistry()

	return &ContractRegistry{
		state:      stateManager,
		storage:    storage,
		code:       codeRegistry,
		runtime:    NewContractRuntime(storage, codeRegistry),
		commitment: NewHashContractStorageCommitment(),
	}
}

func (r *ContractRegistry) CreateContractAccount(contract types.Address, codeHash types.Hash, initialBalance types.Amount) (*accounts.Account, error) {
	if r == nil || r.state == nil {
		return nil, ErrNilContractStateManager
	}

	acc, err := r.state.GetAccount(contract)
	if err == nil && acc != nil {
		acc.SetCodeHash(codeHash)

		root, rootErr := r.commitment.ComputeStorageRoot(r.storage, contract)
		if rootErr != nil {
			return nil, rootErr
		}
		acc.SetStorageRoot(root)
		acc.Balance = initialBalance
		return acc, nil
	}

	acc = accounts.NewAccount(contract, "")
	acc.SetCodeHash(codeHash)
	acc.SetStorageRoot("")
	acc.Balance = initialBalance

	if err := r.state.accounts.Set(acc); err != nil {
		return nil, err
	}

	root, rootErr := r.commitment.ComputeStorageRoot(r.storage, contract)
	if rootErr != nil {
		return nil, rootErr
	}
	acc.SetStorageRoot(root)

	return acc, nil
}

func (r *ContractRegistry) DeployContract(contract types.Address, code string, initialBalance types.Amount) (*accounts.Account, types.Hash, error) {
	if r == nil || r.state == nil {
		return nil, "", ErrNilContractStateManager
	}
	if strings.TrimSpace(code) == "" {
		return nil, "", ErrMissingContractCode
	}

	codeHash, err := ComputeCodeHash(code)
	if err != nil {
		return nil, "", err
	}

	acc, err := r.CreateContractAccount(contract, codeHash, initialBalance)
	if err != nil {
		return nil, "", err
	}

	r.code.Set(contract, code)
	return acc, codeHash, nil
}

func (r *ContractRegistry) GetCode(contract types.Address) (string, bool) {
	if r == nil || r.code == nil {
		return "", false
	}
	return r.code.Get(contract)
}

func (r *ContractRegistry) AllCode() map[types.Address]string {
	if r == nil || r.code == nil {
		return map[types.Address]string{}
	}
	return r.code.All()
}

func (r *ContractRegistry) LoadCode(code map[types.Address]string) {
	if r == nil || r.code == nil {
		return
	}
	r.code.Load(code)
}

func (r *ContractRegistry) AllStorage() map[types.Address]map[string]string {
	if r == nil || r.storage == nil {
		return map[types.Address]map[string]string{}
	}
	return r.storage.All()
}

func (r *ContractRegistry) LoadStorage(data map[types.Address]map[string]string) {
	if r == nil || r.storage == nil {
		return
	}
	r.storage.Load(data)
}

func (r *ContractRegistry) ensureContractAccount(contract types.Address) (*accounts.Account, error) {
	if r == nil || r.state == nil {
		return nil, ErrNilContractStateManager
	}

	acc, err := r.state.GetAccount(contract)
	if err != nil {
		return nil, err
	}
	if !acc.IsContract() {
		return nil, ErrContractAccountRequired
	}

	return acc, nil
}

func (r *ContractRegistry) SetStorage(contract types.Address, key string, value string) error {
	if r == nil || r.state == nil || r.storage == nil {
		return nil
	}

	acc, err := r.ensureContractAccount(contract)
	if err != nil {
		return err
	}

	r.storage.Set(contract, key, value)

	root, err := r.commitment.ComputeStorageRoot(r.storage, contract)
	if err != nil {
		return err
	}

	acc.SetStorageRoot(root)
	return nil
}

func (r *ContractRegistry) DeleteStorage(contract types.Address, key string) error {
	if r == nil || r.state == nil || r.storage == nil {
		return nil
	}

	acc, err := r.ensureContractAccount(contract)
	if err != nil {
		return err
	}

	r.storage.Delete(contract, key)

	root, err := r.commitment.ComputeStorageRoot(r.storage, contract)
	if err != nil {
		return err
	}

	acc.SetStorageRoot(root)
	return nil
}

func (r *ContractRegistry) GetStorage(contract types.Address, key string) (string, bool) {
	if r == nil || r.storage == nil {
		return "", false
	}

	return r.storage.Get(contract, key)
}

func (r *ContractRegistry) StorageRoot(contract types.Address) (types.Hash, error) {
	if r == nil || r.storage == nil {
		return "", nil
	}

	return r.commitment.ComputeStorageRoot(r.storage, contract)
}

func (r *ContractRegistry) Call(contract types.Address, method string, key string, value string) (ContractCallResult, error) {
	if r == nil || r.state == nil || r.runtime == nil {
		return ContractCallResult{}, ErrNilContractStateManager
	}

	acc, err := r.ensureContractAccount(contract)
	if err != nil {
		return ContractCallResult{}, err
	}

	return r.runtime.Execute(acc, contract, method, key, value)
}

func (r *ContractRegistry) SetCommitment(commitment ContractStorageCommitment) {
	if r == nil || commitment == nil {
		return
	}
	r.commitment = commitment
}
