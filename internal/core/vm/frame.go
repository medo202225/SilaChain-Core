package vm

type CallFrame struct {
	Context            ExecutionContext
	Code               []byte
	PC                 uint64
	Stack              *Stack
	Memory             *Memory
	ReturnData         []byte
	RevertData         []byte
	LastCreatedAddress string
}

func NewCallFrame(ctx ExecutionContext, code []byte, limits Limits) *CallFrame {
	return &CallFrame{
		Context: ctx,
		Code:    cloneBytes(code),
		PC:      0,
		Stack:   NewStack(limits.MaxStackDepth),
		Memory:  NewMemory(),
	}
}

func (f *CallFrame) RemainingGas() uint64 {
	return f.Context.GasRemaining
}

func (f *CallFrame) ConsumeGas(amount uint64) error {
	if f.Context.GasRemaining < amount {
		f.Context.GasRemaining = 0
		return ErrOutOfGas
	}
	f.Context.GasRemaining -= amount
	return nil
}

func (f *CallFrame) SetReturnData(data []byte) {
	f.ReturnData = cloneBytes(data)
}

func (f *CallFrame) SetRevertData(data []byte) {
	f.RevertData = cloneBytes(data)
}

func (f *CallFrame) SetCreatedAddress(address string) {
	f.LastCreatedAddress = address
}
