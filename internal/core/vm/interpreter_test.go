package vm

import "testing"

func TestInterpreterStop(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
	}

	result := vm.Run(ctx, []byte{OpStop})
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v", result.Reason.String())
	}
}

func TestInterpreterAdd(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
	}

	code := []byte{
		OpPush1, 0x02,
		OpPush1, 0x03,
		OpAdd,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v, err=%v", result.Reason.String(), result.Err)
	}
	if result.GasUsed == 0 {
		t.Fatalf("expected gas to be consumed")
	}
}

func TestInterpreterMStoreAndReturn(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpReturn,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v, err=%v", result.Reason.String(), result.Err)
	}

	if len(result.ReturnData) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(result.ReturnData))
	}

	if result.ReturnData[31] != 0x2a {
		t.Fatalf("expected last byte to be 0x2a, got 0x%x", result.ReturnData[31])
	}
}

func TestInterpreterRevert(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
	}

	code := []byte{
		OpPush1, 0x09,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpRevert,
	}

	result := vm.Run(ctx, code)
	if !result.Reverted() {
		t.Fatalf("expected revert, got %v", result.Reason.String())
	}

	if len(result.RevertData) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(result.RevertData))
	}

	if result.RevertData[31] != 0x09 {
		t.Fatalf("expected last byte to be 0x09, got 0x%x", result.RevertData[31])
	}
}

func TestInterpreterInvalidOpcode(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 100,
	}

	result := vm.Run(ctx, []byte{0xaa})
	if !result.Faulted() {
		t.Fatalf("expected fault")
	}
}

func TestInterpreterSStoreAndSLoad(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		StorageAddr:  "contract1",
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x01,
		OpSStore,
		OpPush1, 0x01,
		OpSLoad,
		OpPop,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v, err=%v", result.Reason.String(), result.Err)
	}

	stored := host.GetStorage("contract1", string(WordToBytes32(NewWordFromUint64(1))))
	if len(stored) != 32 {
		t.Fatalf("expected 32-byte stored value, got %d", len(stored))
	}
	if stored[31] != 0x2a {
		t.Fatalf("expected stored last byte to be 0x2a, got 0x%x", stored[31])
	}
}

func TestInterpreterSStoreFailsInStaticMode(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		StorageAddr:  "contract1",
		Static:       true,
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x01,
		OpSStore,
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

func TestInterpreterOutOfGas(t *testing.T) {
	vm := NewInterpreter(DefaultLimits())
	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 2,
	}

	code := []byte{
		OpPush1, 0x02,
		OpPush1, 0x03,
		OpAdd,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Faulted() {
		t.Fatalf("expected fault")
	}
	if result.Err != ErrOutOfGas {
		t.Fatalf("expected ErrOutOfGas, got %v", result.Err)
	}
}

func TestInterpreterRevertRollsBackStorage(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
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
		t.Fatalf("expected revert")
	}

	stored := host.GetStorage("contract1", string(WordToBytes32(NewWordFromUint64(1))))
	if len(stored) != 0 {
		t.Fatalf("expected storage rollback, got %d bytes", len(stored))
	}
}

func TestInterpreterLog0(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: "contract1",
		StorageAddr:  "contract1",
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpLog0,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v, err=%v", result.Reason.String(), result.Err)
	}

	if len(host.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(host.logs))
	}
	if len(host.logs[0].Data) != 32 {
		t.Fatalf("expected 32 bytes log data, got %d", len(host.logs[0].Data))
	}
	if host.logs[0].Data[31] != 0x2a {
		t.Fatalf("expected log last byte 0x2a, got 0x%x", host.logs[0].Data[31])
	}
}

func TestInterpreterLog1(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: "contract1",
		StorageAddr:  "contract1",
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x01,
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpLog1,
		OpStop,
	}

	result := vm.Run(ctx, code)
	if !result.Succeeded() {
		t.Fatalf("expected success, got %v, err=%v", result.Reason.String(), result.Err)
	}

	if len(host.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(host.logs))
	}
	if len(host.logs[0].Topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(host.logs[0].Topics))
	}
	topic := host.logs[0].Topics[0]
	if len(topic) != 32 {
		t.Fatalf("expected 32-byte topic, got %d", len(topic))
	}
	if topic[31] != 0x01 {
		t.Fatalf("expected topic last byte 0x01, got 0x%x", topic[31])
	}
}

func TestInterpreterRevertRollsBackLogs(t *testing.T) {
	host := newMockHost()
	vm := NewInterpreterWithHost(DefaultLimits(), host)

	ctx := ExecutionContext{
		VMVersion:    1,
		GasRemaining: 1000,
		ContractAddr: "contract1",
		StorageAddr:  "contract1",
	}

	code := []byte{
		OpPush1, 0x2a,
		OpPush1, 0x00,
		OpMStore,
		OpPush1, 0x20,
		OpPush1, 0x00,
		OpLog0,
		OpPush1, 0x00,
		OpPush1, 0x00,
		OpRevert,
	}

	result := vm.Run(ctx, code)
	if !result.Reverted() {
		t.Fatalf("expected revert")
	}

	if len(host.logs) != 0 {
		t.Fatalf("expected logs rollback, got %d logs", len(host.logs))
	}
}
