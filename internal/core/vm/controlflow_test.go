package vm

import "testing"

func jumpDestIndex(t *testing.T, code []byte) byte {
	t.Helper()
	for i, op := range code {
		if op == OpJumpDest {
			return byte(i)
		}
	}
	t.Fatalf("missing JUMPDEST in test code")
	return 0
}

func TestInterpreterEQ(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{VMVersion: 1, GasRemaining: 100}

	code := []byte{
		OpPush1, 0x05,
		OpPush1, 0x05,
		OpEQ,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterLT(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{VMVersion: 1, GasRemaining: 100}

	code := []byte{
		OpPush1, 0x07,
		OpPush1, 0x05,
		OpLT,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterGT(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{VMVersion: 1, GasRemaining: 100}

	code := []byte{
		OpPush1, 0x05,
		OpPush1, 0x07,
		OpGT,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterIsZero(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{VMVersion: 1, GasRemaining: 100}

	code := []byte{
		OpPush1, 0x00,
		OpIsZero,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterJump(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{VMVersion: 1, GasRemaining: 100}

	code := []byte{
		OpPush1, 0x00, // placeholder, will be replaced with JUMPDEST index
		OpJump,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpJumpDest,
		OpStop,
	}
	code[1] = jumpDestIndex(t, code)

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterJumpI(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{VMVersion: 1, GasRemaining: 100}

	code := []byte{
		OpPush1, 0x01, // condition = true
		OpPush1, 0x00, // placeholder, will be replaced with JUMPDEST index
		OpJumpI,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpStop,
		OpJumpDest,
		OpStop,
	}
	code[3] = jumpDestIndex(t, code)

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterInvalidJump(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{VMVersion: 1, GasRemaining: 100}

	code := []byte{
		OpPush1, 0x03,
		OpJump,
		OpStop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Faulted() {
		t.Fatalf("expected fault")
	}
	if result.Err != ErrInvalidJump {
		t.Fatalf("expected ErrInvalidJump, got %v", result.Err)
	}
}
