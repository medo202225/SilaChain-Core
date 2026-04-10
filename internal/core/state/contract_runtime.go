package state

import (
	"errors"
	"strconv"
	"strings"

	"silachain/internal/accounts"
	"silachain/pkg/types"
)

var ErrMissingDeployedCode = errors.New("missing deployed contract code")
var ErrInvalidNumericValue = errors.New("invalid numeric storage value")

type ContractRuntime struct {
	storage *ContractStorage
	code    *ContractCodeRegistry
}

func NewContractRuntime(storage *ContractStorage, code *ContractCodeRegistry) *ContractRuntime {
	if storage == nil {
		storage = NewContractStorage()
	}
	if code == nil {
		code = NewContractCodeRegistry()
	}

	return &ContractRuntime{
		storage: storage,
		code:    code,
	}
}

func (rt *ContractRuntime) ensureCode(address types.Address) (string, error) {
	if rt == nil || rt.code == nil {
		return "", ErrMissingDeployedCode
	}

	code, ok := rt.code.Get(address)
	if !ok || strings.TrimSpace(code) == "" {
		return "", ErrMissingDeployedCode
	}

	return code, nil
}

func runtimeEvent(address types.Address, name string, data map[string]string) types.Event {
	return types.Event{
		Address: string(address),
		Name:    name,
		Data:    data,
	}
}

func (rt *ContractRuntime) Execute(
	acc *accounts.Account,
	address types.Address,
	method string,
	key string,
	value string,
) (ContractCallResult, error) {
	result := ContractCallResult{
		Address: address,
		Method:  strings.ToLower(strings.TrimSpace(method)),
		Key:     key,
		Logs:    []types.Event{},
	}

	if acc == nil {
		return result, ErrContractAccountRequired
	}
	if strings.TrimSpace(key) == "" {
		return result, ErrMissingContractStorageKey
	}

	code, err := rt.ensureCode(address)
	if err != nil {
		return result, err
	}

	_ = code

	switch result.Method {
	case "get":
		v, ok := rt.storage.Get(address, key)
		result.Value = v
		result.Found = ok
		result.Mutated = false
		result.StorageRoot = acc.StorageRoot
		result.Logs = append(result.Logs, runtimeEvent(address, "RuntimeStorageRead", map[string]string{
			"key":   key,
			"value": v,
			"found": map[bool]string{true: "true", false: "false"}[ok],
		}))
		return result, nil

	case "set":
		rt.storage.Set(address, key, value)

		root, err := rt.storage.ComputeRoot(address)
		if err != nil {
			return result, err
		}
		acc.SetStorageRoot(root)

		result.Value = value
		result.Found = true
		result.Mutated = true
		result.StorageRoot = root
		result.Logs = append(result.Logs, runtimeEvent(address, "RuntimeStorageWrite", map[string]string{
			"key":          key,
			"value":        value,
			"storage_root": string(root),
		}))
		return result, nil

	case "delete":
		rt.storage.Delete(address, key)

		root, err := rt.storage.ComputeRoot(address)
		if err != nil {
			return result, err
		}
		acc.SetStorageRoot(root)

		result.Found = false
		result.Mutated = true
		result.StorageRoot = root
		result.Logs = append(result.Logs, runtimeEvent(address, "RuntimeStorageDelete", map[string]string{
			"key":          key,
			"storage_root": string(root),
		}))
		return result, nil

	case "inc", "dec":
		current, ok := rt.storage.Get(address, key)
		if !ok || strings.TrimSpace(current) == "" {
			current = "0"
		}

		n, err := strconv.ParseInt(current, 10, 64)
		if err != nil {
			return result, ErrInvalidNumericValue
		}

		if result.Method == "inc" {
			n++
		} else {
			n--
		}

		next := strconv.FormatInt(n, 10)
		rt.storage.Set(address, key, next)

		root, err := rt.storage.ComputeRoot(address)
		if err != nil {
			return result, err
		}
		acc.SetStorageRoot(root)

		result.Value = next
		result.Found = true
		result.Mutated = true
		result.StorageRoot = root
		result.Logs = append(result.Logs, runtimeEvent(address, "RuntimeCounterUpdate", map[string]string{
			"key":          key,
			"value":        next,
			"operation":    result.Method,
			"storage_root": string(root),
		}))
		return result, nil

	default:
		return result, ErrUnsupportedContractMethod
	}
}
