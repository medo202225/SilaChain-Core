package vm

import "testing"

func TestInterpreterCreate(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 4000,
		ContractAddr: "factory",
		StorageAddr:  "factory",
	}

	result := vm.Run(ctx, []byte{
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x01,
		OpPush1, 0x1f,
		OpPush1, 0x00,
		OpCreate,
		OpPop,
		OpStop,
	})
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v err=%v", result.Reason.String(), result.Err)
	}

	if len(host.codes) == 0 {
		t.Fatalf("expected created contract code")
	}
}

func TestInterpreterCreateInstallsReturnedRuntimeCode(t *testing.T) {
	host := newMockHost()

	addr, err := host.CreateContractAddress("factory")
	if err != nil {
		t.Fatalf("unexpected address creation error: %v", err)
	}
	if err := host.DeleteCode(addr); err != nil {
		t.Fatalf("unexpected delete code error: %v", err)
	}
	host.nextContractID--

	initCode := []byte{
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x01,
		OpPush1, 0x1f,
		OpReturn,
	}

	child := NewInterpreterWithHost(DefaultLimits(), host)
	childCtx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: addr,
		CodeAddr:     addr,
		StorageAddr:  addr,
		Caller:       "factory",
		Origin:       "factory",
	}

	createResult := child.Run(childCtx, initCode)
	if !createResult.Succeeded() {
		t.Fatalf("expected init success, got %v err=%v", createResult.Reason.String(), createResult.Err)
	}
	if len(createResult.ReturnData) != 1 {
		t.Fatalf("expected 1-byte runtime code, got %d", len(createResult.ReturnData))
	}
	if createResult.ReturnData[0] != OpStop {
		t.Fatalf("expected runtime byte STOP, got 0x%x", createResult.ReturnData[0])
	}
}

func TestInterpreterCreateInitRevertPreventsInstall(t *testing.T) {
	host := newMockHost()

	addr, err := host.CreateContractAddress("factory")
	if err != nil {
		t.Fatalf("unexpected address creation error: %v", err)
	}

	vm := NewInterpreterWithHost(DefaultLimits(), host)
	initCode := []byte{
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpRevert,
	}
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: addr,
		CodeAddr:     addr,
		StorageAddr:  addr,
		Caller:       "factory",
		Origin:       "factory",
	}

	result := vm.Run(ctx, initCode)
	if !result.Reverted() {
		t.Fatalf("expected revert, got %v err=%v", result.Reason.String(), result.Err)
	}

	if code := host.GetCode(addr); len(code) != 0 {
		t.Fatalf("expected no installed runtime code after revert, got %d bytes", len(code))
	}
}

func TestInterpreterCreateFailsInStaticMode(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 2000,
		ContractAddr: "factory",
		StorageAddr:  "factory",
		Static:       true,
	}

	code := []byte{
		OpPush1, 0x01,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpCreate,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Faulted() {
		t.Fatalf("expected fault")
	}
	if result.Err != ErrWriteProtection {
		t.Fatalf("expected ErrWriteProtection, got %v", result.Err)
	}
}
