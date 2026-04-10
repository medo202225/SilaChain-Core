package vm

import "fmt"

type mockHost struct {
	balances       map[string]uint64
	codes          map[string][]byte
	storage        map[string]map[string][]byte
	logs           []LogEntry
	checkpoints    []mockCheckpoint
	nextContractID uint64
}

type mockCheckpoint struct {
	storage        map[string]map[string][]byte
	logsLen        int
	codes          map[string][]byte
	nextContractID uint64
}

func newMockHost() *mockHost {
	return &mockHost{
		balances:       make(map[string]uint64),
		codes:          make(map[string][]byte),
		storage:        make(map[string]map[string][]byte),
		logs:           make([]LogEntry, 0),
		nextContractID: 1,
	}
}

func (h *mockHost) AccountExists(address string) bool {
	_, okBalance := h.balances[address]
	_, okCode := h.codes[address]
	_, okStorage := h.storage[address]
	return okBalance || okCode || okStorage
}

func (h *mockHost) GetBalance(address string) uint64 {
	return h.balances[address]
}

func (h *mockHost) Transfer(from, to string, amount uint64) error {
	if h.balances[from] < amount {
		return ErrExecutionAborted
	}
	h.balances[from] -= amount
	h.balances[to] += amount
	return nil
}

func (h *mockHost) GetCode(address string) []byte {
	return cloneBytes(h.codes[address])
}

func (h *mockHost) SetCode(address string, code []byte) error {
	h.codes[address] = cloneBytes(code)
	return nil
}

func (h *mockHost) DeleteCode(address string) error {
	delete(h.codes, address)
	return nil
}

func (h *mockHost) GetStorage(address, key string) []byte {
	if _, ok := h.storage[address]; !ok {
		return nil
	}
	return cloneBytes(h.storage[address][key])
}

func (h *mockHost) SetStorage(address, key string, value []byte) error {
	if _, ok := h.storage[address]; !ok {
		h.storage[address] = make(map[string][]byte)
	}
	h.storage[address][key] = cloneBytes(value)
	return nil
}

func (h *mockHost) EmitLog(entry LogEntry) {
	h.logs = append(h.logs, entry)
}

func (h *mockHost) CreateCheckpoint() int {
	cp := mockCheckpoint{
		storage:        cloneStorageMap(h.storage),
		logsLen:        len(h.logs),
		codes:          cloneCodeMap(h.codes),
		nextContractID: h.nextContractID,
	}
	h.checkpoints = append(h.checkpoints, cp)
	return len(h.checkpoints) - 1
}

func (h *mockHost) CommitCheckpoint(id int) error {
	if id < 0 || id >= len(h.checkpoints) {
		return ErrExecutionAborted
	}
	return nil
}

func (h *mockHost) RevertCheckpoint(id int) error {
	if id < 0 || id >= len(h.checkpoints) {
		return ErrExecutionAborted
	}
	cp := h.checkpoints[id]
	h.storage = cloneStorageMap(cp.storage)
	h.logs = h.logs[:cp.logsLen]
	h.codes = cloneCodeMap(cp.codes)
	h.nextContractID = cp.nextContractID
	return nil
}

func (h *mockHost) CallContract(caller string, target string, input []byte, value uint64, gas uint64, static bool) CallResult {
	code := h.GetCode(target)
	if len(code) == 0 {
		return CallResult{
			Success:    true,
			ReturnData: nil,
			GasUsed:    0,
		}
	}

	vm := NewInterpreterWithHost(DefaultLimits(), h)
	ctx := ExecutionContext{
		VMVersion:    1,
		ContractAddr: target,
		CodeAddr:     target,
		StorageAddr:  target,
		Caller:       caller,
		Origin:       caller,
		CallValue:    value,
		Input:        cloneBytes(input),
		GasRemaining: gas,
		Static:       static,
	}

	result := vm.Run(ctx, code)
	return CallResult{
		Success:    result.Succeeded(),
		ReturnData: cloneBytes(result.ReturnData),
		RevertData: cloneBytes(result.RevertData),
		GasUsed:    result.GasUsed,
		Err:        result.Err,
	}
}

func (h *mockHost) CreateContractAddress(caller string) (string, error) {
	addr := fmt.Sprintf("contract-created-%d", h.nextContractID)
	h.nextContractID++
	if _, ok := h.storage[addr]; !ok {
		h.storage[addr] = make(map[string][]byte)
	}
	return addr, nil
}

func cloneStorageMap(in map[string]map[string][]byte) map[string]map[string][]byte {
	if in == nil {
		return make(map[string]map[string][]byte)
	}

	out := make(map[string]map[string][]byte, len(in))
	for addr, slots := range in {
		out[addr] = make(map[string][]byte, len(slots))
		for key, value := range slots {
			out[addr][key] = cloneBytes(value)
		}
	}
	return out
}

func cloneCodeMap(in map[string][]byte) map[string][]byte {
	if in == nil {
		return make(map[string][]byte)
	}

	out := make(map[string][]byte, len(in))
	for k, v := range in {
		out[k] = cloneBytes(v)
	}
	return out
}
