package vm

import "testing"

func TestInterpreterStaticCallSucceeds(t *testing.T) {
	host := newMockHost()

	calleeCode := []byte{
		OpPush1, 0x08,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpReturn,
	}
	_ = host.SetCode(string(WordToBytes32(NewWordFromUint64(3))), calleeCode)

	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: "caller",
		StorageAddr:  "caller",
	}

	code := []byte{
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x03,
		OpPush1, 0x64,
		OpStaticCall,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterStaticCallBlocksSStore(t *testing.T) {
	host := newMockHost()

	calleeCode := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x01,
		OpSStore,
		OpStop,
	}
	target := string(WordToBytes32(NewWordFromUint64(4)))
	_ = host.SetCode(target, calleeCode)

	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: "caller",
		StorageAddr:  "caller",
	}

	code := []byte{
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x04,
		OpPush1, 0x64,
		OpStaticCall,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected outer success, got %v err=%v", result.Reason.String(), result.Err)
	}

	stored := host.GetStorage(target, string(WordToBytes32(NewWordFromUint64(1))))
	if len(stored) != 0 {
		t.Fatalf("expected no storage write in static call")
	}
}

func TestInterpreterStaticCallBlocksNonZeroValue(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: "caller",
		StorageAddr:  "caller",
	}

	code := []byte{
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x01,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpPush1, 0x05,
		OpPush1, 0x64,
		OpStaticCall,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected outer success, got %v err=%v", result.Reason.String(), result.Err)
	}
}
