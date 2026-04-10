package vm

type StateHost struct {
	accountExistsFn         func(address string) bool
	getBalanceFn            func(address string) uint64
	transferFn              func(from, to string, amount uint64) error
	getCodeFn               func(address string) []byte
	setCodeFn               func(address string, code []byte) error
	deleteCodeFn            func(address string) error
	getStorageFn            func(address, key string) []byte
	setStorageFn            func(address, key string, value []byte) error
	emitLogFn               func(entry LogEntry)
	createCheckpointFn      func() int
	commitCheckpointFn      func(id int) error
	revertCheckpointFn      func(id int) error
	callContractFn          func(caller, target string, input []byte, value uint64, gas uint64, static bool) CallResult
	createContractAddressFn func(caller string) (string, error)
}

func NewStateHost() *StateHost {
	return &StateHost{}
}

func (h *StateHost) WithAccountExists(fn func(address string) bool) *StateHost {
	h.accountExistsFn = fn
	return h
}

func (h *StateHost) WithGetBalance(fn func(address string) uint64) *StateHost {
	h.getBalanceFn = fn
	return h
}

func (h *StateHost) WithTransfer(fn func(from, to string, amount uint64) error) *StateHost {
	h.transferFn = fn
	return h
}

func (h *StateHost) WithGetCode(fn func(address string) []byte) *StateHost {
	h.getCodeFn = fn
	return h
}

func (h *StateHost) WithSetCode(fn func(address string, code []byte) error) *StateHost {
	h.setCodeFn = fn
	return h
}

func (h *StateHost) WithDeleteCode(fn func(address string) error) *StateHost {
	h.deleteCodeFn = fn
	return h
}

func (h *StateHost) WithGetStorage(fn func(address, key string) []byte) *StateHost {
	h.getStorageFn = fn
	return h
}

func (h *StateHost) WithSetStorage(fn func(address, key string, value []byte) error) *StateHost {
	h.setStorageFn = fn
	return h
}

func (h *StateHost) WithEmitLog(fn func(entry LogEntry)) *StateHost {
	h.emitLogFn = fn
	return h
}

func (h *StateHost) WithCreateCheckpoint(fn func() int) *StateHost {
	h.createCheckpointFn = fn
	return h
}

func (h *StateHost) WithCommitCheckpoint(fn func(id int) error) *StateHost {
	h.commitCheckpointFn = fn
	return h
}

func (h *StateHost) WithRevertCheckpoint(fn func(id int) error) *StateHost {
	h.revertCheckpointFn = fn
	return h
}

func (h *StateHost) WithCallContract(fn func(caller, target string, input []byte, value uint64, gas uint64, static bool) CallResult) *StateHost {
	h.callContractFn = fn
	return h
}

func (h *StateHost) WithCreateContractAddress(fn func(caller string) (string, error)) *StateHost {
	h.createContractAddressFn = fn
	return h
}

func (h *StateHost) AccountExists(address string) bool {
	if h.accountExistsFn == nil {
		return false
	}
	return h.accountExistsFn(address)
}

func (h *StateHost) GetBalance(address string) uint64 {
	if h.getBalanceFn == nil {
		return 0
	}
	return h.getBalanceFn(address)
}

func (h *StateHost) Transfer(from, to string, amount uint64) error {
	if h.transferFn == nil {
		return ErrExecutionAborted
	}
	return h.transferFn(from, to, amount)
}

func (h *StateHost) GetCode(address string) []byte {
	if h.getCodeFn == nil {
		return nil
	}
	return cloneBytes(h.getCodeFn(address))
}

func (h *StateHost) SetCode(address string, code []byte) error {
	if h.setCodeFn == nil {
		return ErrExecutionAborted
	}
	return h.setCodeFn(address, cloneBytes(code))
}

func (h *StateHost) DeleteCode(address string) error {
	if h.deleteCodeFn == nil {
		return ErrExecutionAborted
	}
	return h.deleteCodeFn(address)
}

func (h *StateHost) GetStorage(address, key string) []byte {
	if h.getStorageFn == nil {
		return nil
	}
	return cloneBytes(h.getStorageFn(address, key))
}

func (h *StateHost) SetStorage(address, key string, value []byte) error {
	if h.setStorageFn == nil {
		return ErrExecutionAborted
	}
	return h.setStorageFn(address, key, cloneBytes(value))
}

func (h *StateHost) EmitLog(entry LogEntry) {
	if h.emitLogFn != nil {
		h.emitLogFn(entry)
	}
}

func (h *StateHost) CreateCheckpoint() int {
	if h.createCheckpointFn == nil {
		return -1
	}
	return h.createCheckpointFn()
}

func (h *StateHost) CommitCheckpoint(id int) error {
	if h.commitCheckpointFn == nil {
		return ErrExecutionAborted
	}
	return h.commitCheckpointFn(id)
}

func (h *StateHost) RevertCheckpoint(id int) error {
	if h.revertCheckpointFn == nil {
		return ErrExecutionAborted
	}
	return h.revertCheckpointFn(id)
}

func (h *StateHost) CallContract(caller, target string, input []byte, value uint64, gas uint64, static bool) CallResult {
	if h.callContractFn == nil {
		return CallResult{
			Success: false,
			Err:     ErrExecutionAborted,
		}
	}
	return h.callContractFn(caller, target, cloneBytes(input), value, gas, static)
}

func (h *StateHost) CreateContractAddress(caller string) (string, error) {
	if h.createContractAddressFn == nil {
		return "", ErrExecutionAborted
	}
	return h.createContractAddressFn(caller)
}
