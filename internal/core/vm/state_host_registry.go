package vm

import (
	"encoding/hex"

	"silachain/internal/core/state"
	"silachain/pkg/types"
)

func NewRegistryBackedStateHost(
	codeRegistry *state.ContractCodeRegistry,
	storage *state.ContractStorage,
	journal *state.Journal,
) *StateHost {
	host := NewStateHost()

	host.WithGetCode(func(address string) []byte {
		if codeRegistry == nil {
			return nil
		}

		codeHex, ok := codeRegistry.Get(types.Address(address))
		if !ok || codeHex == "" {
			return nil
		}

		decoded, err := hex.DecodeString(codeHex)
		if err != nil {
			return nil
		}
		return decoded
	})

	host.WithSetCode(func(address string, code []byte) error {
		if codeRegistry == nil {
			return ErrExecutionAborted
		}

		codeRegistry.Set(types.Address(address), hex.EncodeToString(code))
		return nil
	})

	host.WithDeleteCode(func(address string) error {
		if codeRegistry == nil {
			return ErrExecutionAborted
		}

		codeRegistry.Delete(types.Address(address))
		return nil
	})

	host.WithGetStorage(func(address, key string) []byte {
		if storage == nil {
			return nil
		}

		valueHex, ok := storage.Get(types.Address(address), key)
		if !ok || valueHex == "" {
			return nil
		}

		decoded, err := hex.DecodeString(valueHex)
		if err != nil {
			return nil
		}
		return decoded
	})

	host.WithSetStorage(func(address, key string, value []byte) error {
		if storage == nil {
			return ErrExecutionAborted
		}

		storage.Set(types.Address(address), key, hex.EncodeToString(value))
		return nil
	})

	host.WithCreateCheckpoint(func() int {
		if journal == nil {
			return -1
		}
		return journal.CreateSnapshot(codeRegistry, storage)
	})

	host.WithCommitCheckpoint(func(id int) error {
		if journal == nil {
			return ErrExecutionAborted
		}
		journal.Commit(id)
		return nil
	})

	host.WithRevertCheckpoint(func(id int) error {
		if journal == nil {
			return ErrExecutionAborted
		}
		ok := journal.Revert(id, codeRegistry, storage)
		if !ok {
			return ErrExecutionAborted
		}
		return nil
	})

	return host
}
