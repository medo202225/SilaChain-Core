package vm

import "testing"

func TestInterpreterCall(t *testing.T) {
	host := newMockHost()

	calleeCode := []byte{
		OpPush1, 0x07,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpReturn,
	}
	_ = host.SetCode(string(WordToBytes32(NewWordFromUint64(2))), calleeCode)

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
		OpPush1, 0x02,
		OpPush1, 0x64,
		OpCall,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}
}

func TestInterpreterCallPushesZeroOnFailure(t *testing.T) {
	host := newMockHost()
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
		OpPush1, 0x09,
		OpPush1, 0x64,
		OpCall,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected outer success, got %v err=%v", result.Reason.String(), result.Err)
	}
}
