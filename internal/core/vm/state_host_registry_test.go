package vm

import (
	"testing"

	"silachain/internal/core/state"
)

func TestRegistryBackedStateHostStorageCommit(t *testing.T) {
	codeRegistry := state.NewContractCodeRegistry()
	storage := state.NewContractStorage()
	journal := state.NewJournal()

	host := NewRegistryBackedStateHost(codeRegistry, storage, journal)
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 2000,
		ContractAddr: "contract1",
		StorageAddr:  "contract1",
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x01,
		OpSStore,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}

	value, ok := storage.Get("contract1", string(WordToBytes32(NewWordFromUint64(1))))
	if !ok {
		t.Fatalf("expected stored value")
	}
	if value == "" {
		t.Fatalf("expected non-empty stored value")
	}
}

func TestRegistryBackedStateHostStorageRevert(t *testing.T) {
	codeRegistry := state.NewContractCodeRegistry()
	storage := state.NewContractStorage()
	journal := state.NewJournal()

	host := NewRegistryBackedStateHost(codeRegistry, storage, journal)
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 2000,
		ContractAddr: "contract1",
		StorageAddr:  "contract1",
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x01,
		OpSStore,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpRevert,
	}

	result := vm.Run(ctx, code)
	if !result.Reverted() {
		t.Fatalf("expected revert, got %v err=%v", result.Reason.String(), result.Err)
	}

	_, ok := storage.Get("contract1", string(WordToBytes32(NewWordFromUint64(1))))
	if ok {
		t.Fatalf("expected storage rollback after revert")
	}
}

func TestRegistryBackedStateHostSetCode(t *testing.T) {
	codeRegistry := state.NewContractCodeRegistry()
	storage := state.NewContractStorage()
	journal := state.NewJournal()

	host := NewRegistryBackedStateHost(codeRegistry, storage, journal)

	code := []byte{OpStop}
	if err := host.SetCode("contract-code-1", code); err != nil {
		t.Fatalf("unexpected set code error: %v", err)
	}

	got := host.GetCode("contract-code-1")
	if len(got) != 1 {
		t.Fatalf("expected 1 byte code, got %d", len(got))
	}
	if got[0] != OpStop {
		t.Fatalf("expected STOP opcode, got 0x%x", got[0])
	}
}

func TestRegistryBackedStateHostCheckpointLifecycle(t *testing.T) {
	codeRegistry := state.NewContractCodeRegistry()
	storage := state.NewContractStorage()
	journal := state.NewJournal()

	host := NewRegistryBackedStateHost(codeRegistry, storage, journal)

	checkpointID := host.CreateCheckpoint()
	if checkpointID < 0 {
		t.Fatalf("expected valid checkpoint id")
	}

	if err := host.SetStorage("contract1", "key1", []byte{0xaa}); err != nil {
		t.Fatalf("unexpected set storage error: %v", err)
	}

	if err := host.RevertCheckpoint(checkpointID); err != nil {
		t.Fatalf("unexpected revert checkpoint error: %v", err)
	}

	value := host.GetStorage("contract1", "key1")
	if len(value) != 0 {
		t.Fatalf("expected reverted storage to be empty")
	}
}
