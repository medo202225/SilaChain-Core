package execution

import (
	"testing"

	"silachain/internal/core/state"
	"silachain/internal/core/vm"
)

func TestVMBridgeExecute(t *testing.T) {
	codeRegistry := state.NewContractCodeRegistry()
	storage := state.NewContractStorage()
	journal := state.NewJournal()

	bridge := NewVMBridge(codeRegistry, storage, journal)

	ctx := vm.ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: "contract1",
		StorageAddr:  "contract1",
	}

	code := []byte{
		vm.OpPush1, 0x2a,
		vm.OpPush1, 0x01,
		vm.OpSStore,
		vm.OpStop,
	}

	result := bridge.Execute(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}

	value, ok := storage.Get("contract1", string(vm.WordToBytes32(vm.NewWordFromUint64(1))))
	if !ok {
		t.Fatalf("expected storage write")
	}
	if value == "" {
		t.Fatalf("expected non-empty storage value")
	}
}

func TestVMBridgeHost(t *testing.T) {
	codeRegistry := state.NewContractCodeRegistry()
	storage := state.NewContractStorage()
	journal := state.NewJournal()

	bridge := NewVMBridge(codeRegistry, storage, journal)
	if bridge.Host() == nil {
		t.Fatalf("expected host")
	}
}
