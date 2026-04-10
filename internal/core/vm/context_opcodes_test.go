package vm

import "testing"

func TestInterpreterAddress(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
		ContractAddr: "contract1",
	}

	code := []byte{
		OpAddress,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterCaller(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
		Caller:       "caller1",
	}

	code := []byte{
		OpCaller,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterCallValue(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
		CallValue:    42,
	}

	code := []byte{
		OpCallValue,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterCallDataSize(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
		Input:        []byte{0x11, 0x22, 0x33},
	}

	code := []byte{
		OpCallDataSize,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterCallDataLoad(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
		Input:        []byte{0xaa, 0xbb, 0xcc},
	}

	code := []byte{
		OpPush1, 0x00,
		OpCallDataLoad,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}
