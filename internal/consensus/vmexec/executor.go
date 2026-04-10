package vmexec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"silachain/internal/consensus/vmstate"
)

var (
	ErrCodeSizeExceeded      = errors.New("max code size exceeded")
	ErrNilState              = errors.New("vmexec: nil state")
	ErrEmptyCaller           = errors.New("vmexec: empty caller")
	ErrEmptyTo               = errors.New("vmexec: empty to")
	ErrExecutionRevert       = errors.New("vmexec: execution reverted")
	ErrStackUnderflow        = errors.New("vmexec: stack underflow")
	ErrInvalidOpcode         = errors.New("vmexec: invalid opcode")
	ErrTruncatedPush         = errors.New("vmexec: truncated push data")
	ErrInvalidJumpDest       = errors.New("vmexec: invalid jump destination")
	ErrReturnDataOutOfBounds = errors.New("vmexec: returndata out of bounds")
	ErrWriteProtection       = errors.New("vmexec: write protection")
	ErrOutOfGas              = errors.New("vmexec: out of gas")
)

const (
	SStoreClearRefundGas       = 4800
	WarmAccessCost             = 100
	ColdAccountAccessCost      = 2600
	WarmStorageReadCost        = 100
	ColdStorageReadCost        = 2100
	MaxRuntimeCodeSize         = 24576
	OpSTOP                byte = 0x00
	OpADD                 byte = 0x01
	OpLT                  byte = 0x10
	OpGT                  byte = 0x11
	OpEQ                  byte = 0x14
	OpISZERO              byte = 0x15
	OpADDRESS             byte = 0x30
	OpBALANCE             byte = 0x31
	OpORIGIN              byte = 0x32
	OpNUMBER              byte = 0x43
	OpTIMESTAMP           byte = 0x42
	OpBASEFEE             byte = 0x48
	OpCHAINID             byte = 0x46
	OpGASLIMIT            byte = 0x45
	OpSELFBALANCE         byte = 0x47
	OpCALLER              byte = 0x33
	OpCALLVALUE           byte = 0x34
	OpCALLDATALOAD        byte = 0x35
	OpCALLDATASIZE        byte = 0x36
	OpCALLDATACOPY        byte = 0x37
	OpCODESIZE            byte = 0x38
	OpCODECOPY            byte = 0x39
	OpEXTCODESIZE         byte = 0x3b
	OpEXTCODECOPY         byte = 0x3c
	OpEXTCODEHASH         byte = 0x3f
	OpRETURNDATASIZE      byte = 0x3d
	OpRETURNDATACOPY      byte = 0x3e
	OpMLOAD               byte = 0x51
	OpMSTORE              byte = 0x52
	OpMSTORE8             byte = 0x53
	OpSLOAD               byte = 0x54
	OpSSTORE              byte = 0x55
	OpJUMP                byte = 0x56
	OpJUMPI               byte = 0x57
	OpPC                  byte = 0x58
	OpMSIZE               byte = 0x59
	OpGAS                 byte = 0x5a
	OpJUMPDEST            byte = 0x5b
	OpPUSH1               byte = 0x60
	OpPUSH2               byte = 0x61
	OpLOG0                byte = 0xa0
	OpLOG1                byte = 0xa1
	OpCREATE              byte = 0xf0
	OpCREATE2             byte = 0xf5
	OpCALL                byte = 0xf1
	OpDELEGATECALL        byte = 0xf4
	OpSTATICCALL          byte = 0xfa
	OpRETURN              byte = 0xf3
	OpREVERT              byte = 0xfd
	OpSELFDESTRUCT        byte = 0xff
)

const MaxCallDepth = 1024

type Message struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    uint64 `json:"value"`
	GasLimit uint64 `json:"gasLimit"`
	Data     []byte `json:"data"`
}

type ExecutionContext struct {
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	Timestamp   uint64 `json:"timestamp"`
	GasLimit    uint64 `json:"gasLimit"`
	BaseFee     uint64 `json:"baseFee"`
	ChainID     uint64 `json:"chainId"`
}

type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    []byte   `json:"data"`
}

type Result struct {
	Success        bool   `json:"success"`
	Reverted       bool   `json:"reverted"`
	GasUsed        uint64 `json:"gasUsed"`
	ReturnData     []byte `json:"returnData"`
	Logs           []Log  `json:"logs"`
	CodeExecuted   bool   `json:"codeExecuted"`
	CodeSize       int    `json:"codeSize"`
	CreatedAccount bool   `json:"createdAccount"`
	Steps          int    `json:"steps"`
}

type Executor struct {
	state *vmstate.State
}

type word [32]byte

func zeroWord() word {
	return word{}
}

func wordFromUint64(v uint64) word {
	var w word
	binary.BigEndian.PutUint64(w[24:], v)
	return w
}

func (w word) Uint64() uint64 {
	return binary.BigEndian.Uint64(w[24:])
}

func (w word) IsZero() bool {
	return w == zeroWord()
}

func wordsEqual(a, b word) bool {
	return a == b
}

func addWords(a, b word) word {
	var out word
	carry := uint16(0)
	for i := 31; i >= 0; i-- {
		sum := uint16(a[i]) + uint16(b[i]) + carry
		out[i] = byte(sum & 0xff)
		carry = sum >> 8
	}
	return out
}

func wordLessThan(a, b word) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	return false
}

func wordGreaterThan(a, b word) bool {
	for i := 0; i < len(a); i++ {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
	}
	return false
}

type stack struct {
	items []word
}

func (s *stack) push(v word) {
	s.items = append(s.items, v)
}

func (s *stack) pushUint64(v uint64) {
	s.push(wordFromUint64(v))
}

func (s *stack) pop() (word, error) {
	if len(s.items) == 0 {
		return zeroWord(), ErrStackUnderflow
	}
	last := len(s.items) - 1
	v := s.items[last]
	s.items = s.items[:last]
	return v, nil
}

func (s *stack) popUint64() (uint64, error) {
	v, err := s.pop()
	if err != nil {
		return 0, err
	}
	return v.Uint64(), nil
}

type memory struct {
	buf []byte
}

func (m *memory) ensure(size int) {
	if size <= len(m.buf) {
		return
	}
	next := make([]byte, size)
	copy(next, m.buf)
	m.buf = next
}

func (m *memory) storeWord(offset uint64, value word) {
	start := int(offset)
	end := start + 32
	m.ensure(end)
	copy(m.buf[start:end], value[:])
}

func (m *memory) loadWord(offset uint64) word {
	start := int(offset)
	end := start + 32
	m.ensure(end)

	var out word
	copy(out[:], m.buf[start:end])
	return out
}

func (m *memory) storeBytes(offset uint64, value []byte) {
	start := int(offset)
	end := start + len(value)
	m.ensure(end)
	copy(m.buf[start:end], value)
}

func (m *memory) storeByte(offset uint64, value byte) {
	start := int(offset)
	end := start + 1
	m.ensure(end)
	m.buf[start] = value
}

func (m *memory) slice(offset uint64, size uint64) []byte {
	start := int(offset)
	end := start + int(size)
	m.ensure(end)
	out := make([]byte, int(size))
	copy(out, m.buf[start:end])
	return out
}

type callFrame struct {
	Contract      string
	StorageTarget string
	Caller        string
	Origin        string
	Value         uint64
	Calldata      []byte
	Depth         int
	Static        bool
	CreateCounter uint64
	GasLimit      uint64
	Refund        uint64
	WarmAccounts  map[string]struct{}
	WarmStorage   map[string]map[string]struct{}
}

func New(state *vmstate.State) (*Executor, error) {
	if state == nil {
		return nil, ErrNilState
	}
	return &Executor{state: state}, nil
}

func (e *Executor) Execute(ctx ExecutionContext, msg Message) (Result, error) {
	if e == nil || e.state == nil {
		return Result{}, ErrNilState
	}
	if msg.From == "" {
		return Result{}, ErrEmptyCaller
	}
	if msg.To == "" {
		return Result{}, ErrEmptyTo
	}

	if _, err := e.state.EnsureAccount(msg.From); err != nil {
		return Result{}, err
	}

	acct, ok := e.state.GetAccount(msg.To)
	created := false
	if !ok {
		var err error
		_, err = e.state.EnsureAccount(msg.To)
		if err != nil {
			return Result{}, err
		}
		acct, _ = e.state.GetAccount(msg.To)
		created = true
	}

	baseGas := uint64(21000)
	if len(acct.Code) == 0 {
		return Result{
			Success:        true,
			Reverted:       false,
			GasUsed:        baseGas,
			ReturnData:     []byte{},
			Logs:           []Log{},
			CodeExecuted:   false,
			CodeSize:       0,
			CreatedAccount: created,
			Steps:          0,
		}, nil
	}

	returnData, logs, reverted, steps, refund, err := executeBytecode(
		e.state,
		ctx,
		callFrame{
			Contract:      msg.To,
			StorageTarget: msg.To,
			Caller:        msg.From,
			Origin:        msg.From,
			Value:         msg.Value,
			Calldata:      msg.Data,
			Depth:         0,
			Static:        false,
			CreateCounter: 0,
			GasLimit:      msg.GasLimit,
		},
		acct.Code,
	)
	if err != nil {
		return Result{}, err
	}

	gasUsed := baseGas + uint64(len(acct.Code))
	maxRefund := gasUsed / 5
	if refund > maxRefund {
		refund = maxRefund
	}
	gasUsed -= refund

	return Result{
		Success:        !reverted,
		Reverted:       reverted,
		GasUsed:        gasUsed,
		ReturnData:     returnData,
		Logs:           logs,
		CodeExecuted:   true,
		CodeSize:       len(acct.Code),
		CreatedAccount: created,
		Steps:          steps,
	}, nil
}

func executeBytecode(state *vmstate.State, ctx ExecutionContext, frame callFrame, code []byte) ([]byte, []Log, bool, int, uint64, error) {
	ensureWarmTracking(&frame)
	pc := 0
	st := &stack{}
	mem := &memory{}
	steps := 0
	logs := make([]Log, 0)
	lastReturnData := []byte{}
	gasRemaining := frame.GasLimit

	for pc < len(code) {
		op := code[pc]
		steps++

		switch op {
		case OpSTOP:
			return []byte{}, logs, false, steps, frame.Refund, nil

		case OpADD:
			a, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			b, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			st.push(addWords(a, b))

		case OpADDRESS:
			st.pushUint64(contractWordFromAddress(frame.Contract))

		case OpBALANCE:
			addressWord, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			target := contractAddressFromWord(addressWord)
			if err := chargeAccountAccessGas(&frame, &gasRemaining, target); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			acct, ok := state.GetAccount(target)
			if !ok {
				st.pushUint64(0)
			} else {
				st.pushUint64(acct.Balance)
			}

		case OpTIMESTAMP:
			st.pushUint64(ctx.Timestamp)

		case OpBASEFEE:
			st.pushUint64(ctx.BaseFee)

		case OpGASLIMIT:
			st.pushUint64(ctx.GasLimit)

		case OpCHAINID:
			st.pushUint64(ctx.ChainID)

		case OpNUMBER:
			st.pushUint64(ctx.BlockNumber)

		case OpSELFBALANCE:
			acct, ok := state.GetAccount(frame.Contract)
			if !ok {
				st.pushUint64(0)
			} else {
				st.pushUint64(acct.Balance)
			}

		case OpORIGIN:
			st.pushUint64(contractWordFromAddress(frame.Origin))

		case OpCALLER:
			st.pushUint64(contractWordFromAddress(frame.Caller))

		case OpCALLVALUE:
			st.pushUint64(frame.Value)

		case OpCALLDATALOAD:
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			st.push(calldataLoadWord(frame.Calldata, offset))

		case OpCALLDATASIZE:
			st.pushUint64(uint64(len(frame.Calldata)))

		case OpCALLDATACOPY:
			memOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			dataOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if err := consumeGas(&gasRemaining, copyGasCost(size)); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			copied := make([]byte, int(size))
			for i := uint64(0); i < size; i++ {
				idx := dataOffset + i
				if idx < uint64(len(frame.Calldata)) {
					copied[i] = frame.Calldata[idx]
				}
			}
			mem.storeBytes(memOffset, copied)

		case OpCODESIZE:
			st.pushUint64(uint64(len(code)))

		case OpCODECOPY:
			memOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			codeOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if size == 0 {
				break
			}
			out := make([]byte, int(size))
			for i := 0; i < int(size); i++ {
				src := int(codeOffset) + i
				if src >= 0 && src < len(code) {
					out[i] = code[src]
				}
			}
			mem.storeBytes(memOffset, out)

		case OpEXTCODESIZE:
			addressWord, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			target := contractAddressFromWord(addressWord)
			if err := chargeAccountAccessGas(&frame, &gasRemaining, target); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			acct, ok := state.GetAccount(target)
			if !ok {
				st.pushUint64(0)
			} else {
				st.pushUint64(uint64(len(acct.Code)))
			}

		case OpEXTCODECOPY:
			addressWord, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			destOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			codeOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}

			target := contractAddressFromWord(addressWord)
			if err := chargeAccountAccessGas(&frame, &gasRemaining, target); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			acct, ok := state.GetAccount(target)

			copied := make([]byte, int(size))
			if ok && len(acct.Code) > 0 && codeOffset < uint64(len(acct.Code)) {
				n := copy(copied, acct.Code[codeOffset:])
				for i := n; i < len(copied); i++ {
					copied[i] = 0
				}
			}
			mem.storeBytes(destOffset, copied)

		case OpEXTCODEHASH:
			addressWord, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			target := contractAddressFromWord(addressWord)
			if err := chargeAccountAccessGas(&frame, &gasRemaining, target); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			acct, ok := state.GetAccount(target)
			if !ok || len(acct.Code) == 0 {
				st.pushUint64(0)
			} else {
				st.pushUint64(uint64(len(acct.Code)))
			}

		case OpRETURNDATASIZE:
			st.pushUint64(uint64(len(lastReturnData)))

		case OpRETURNDATACOPY:
			memOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			dataOffset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if size == 0 {
				break
			}
			if err := consumeGas(&gasRemaining, copyGasCost(size)); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			start := int(dataOffset)
			end := start + int(size)
			if start < 0 || end < start || end > len(lastReturnData) {
				return nil, nil, false, steps, frame.Refund, ErrReturnDataOutOfBounds
			}
			mem.storeBytes(memOffset, lastReturnData[start:end])

		case OpLT:
			a, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			b, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if wordLessThan(b, a) {
				st.pushUint64(1)
			} else {
				st.pushUint64(0)
			}

		case OpGT:
			a, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			b, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if wordGreaterThan(b, a) {
				st.pushUint64(1)
			} else {
				st.pushUint64(0)
			}

		case OpEQ:
			a, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			b, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if b == a {
				st.pushUint64(1)
			} else {
				st.pushUint64(0)
			}

		case OpISZERO:
			v, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if v.IsZero() {
				st.pushUint64(1)
			} else {
				st.pushUint64(0)
			}

		case OpMLOAD:
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			st.push(mem.loadWord(offset))

		case OpMSTORE:
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			value, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			mem.storeWord(offset, value)

		case OpMSTORE8:
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			value, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			mem.storeByte(offset, value[31])

		case OpPC:
			st.pushUint64(uint64(pc))

		case OpMSIZE:
			st.pushUint64(uint64(len(mem.buf)))

		case OpSLOAD:
			key, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if err := chargeStorageAccessGas(&frame, &gasRemaining, frame.StorageTarget, storageKey(key)); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			val, ok := state.GetStorage(frame.StorageTarget, storageKey(key))
			if !ok {
				st.pushUint64(0)
			} else {
				st.pushUint64(parseStorageValue(val))
			}

		case OpSSTORE:
			if frame.Static {
				return nil, nil, false, steps, frame.Refund, ErrWriteProtection
			}
			key, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			value, err := st.pop()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			prevRaw, _ := state.GetStorage(frame.StorageTarget, storageKey(key))
			prevValue := parseStorageValue(prevRaw)
			newValue := value.Uint64()

			if err := state.SetStorage(frame.StorageTarget, storageKey(key), storageValue(newValue)); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if prevValue != 0 && newValue == 0 {
				frame.Refund += SStoreClearRefundGas
			}

		case OpJUMP:
			dest, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			next, err := validateJumpDest(code, dest)
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			pc = next
			continue

		case OpJUMPI:
			dest, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			cond, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if cond != 0 {
				next, err := validateJumpDest(code, dest)
				if err != nil {
					return nil, nil, false, steps, frame.Refund, err
				}
				pc = next
				continue
			}

		case OpGAS:
			st.pushUint64(gasRemaining)

		case OpJUMPDEST:

		case OpPUSH1:
			if pc+1 >= len(code) {
				return nil, nil, false, steps, frame.Refund, ErrInvalidOpcode
			}
			st.pushUint64(uint64(code[pc+1]))
			pc += 2
			continue

		case OpPUSH2:
			if pc+2 >= len(code) {
				return nil, nil, false, steps, frame.Refund, ErrInvalidOpcode
			}
			var w word
			w[30] = code[pc+1]
			w[31] = code[pc+2]
			st.push(w)
			pc += 3
			continue

		case OpLOG0:
			if frame.Static {
				return nil, nil, false, steps, frame.Refund, ErrWriteProtection
			}
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			logs = append(logs, Log{
				Address: frame.StorageTarget,
				Topics:  []string{},
				Data:    mem.slice(offset, size),
			})

		case OpLOG1:
			if frame.Static {
				return nil, nil, false, steps, frame.Refund, ErrWriteProtection
			}
			topic, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			logs = append(logs, Log{
				Address: frame.StorageTarget,
				Topics:  []string{storageKey(topic)},
				Data:    mem.slice(offset, size),
			})

		case OpCREATE:
			success, returnData, err := executeCreate(state, ctx, &frame, mem, st)
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			lastReturnData = returnData
			if success {
				st.pushUint64(returnDataToWord(returnData))
			} else {
				st.pushUint64(0)
			}

		case OpCALL:
			success, returnData, callLogs, err := executeCall(state, ctx, frame, mem, st)
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			lastReturnData = returnData
			logs = append(logs, callLogs...)
			if success {
				st.pushUint64(1)
			} else {
				st.pushUint64(0)
			}

		case OpDELEGATECALL:
			success, returnData, callLogs, err := executeDelegateCall(state, ctx, frame, mem, st)
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			lastReturnData = returnData
			logs = append(logs, callLogs...)
			if success {
				st.pushUint64(1)
			} else {
				st.pushUint64(0)
			}

		case OpCREATE2:
			success, returnData, err := executeCreate2(state, ctx, &frame, mem, st)
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			lastReturnData = returnData
			if success {
				st.pushUint64(returnDataToWord(returnData))
			} else {
				st.pushUint64(0)
			}

		case OpSTATICCALL:
			success, returnData, callLogs, err := executeStaticCall(state, ctx, frame, mem, st)
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			lastReturnData = returnData
			logs = append(logs, callLogs...)
			if success {
				st.pushUint64(1)
			} else {
				st.pushUint64(0)
			}

		case OpRETURN:
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			return mem.slice(offset, size), logs, false, steps, frame.Refund, nil

		case OpREVERT:
			size, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			offset, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			return mem.slice(offset, size), logs, true, steps, frame.Refund, nil

		case OpSELFDESTRUCT:
			if frame.Static {
				return nil, nil, false, steps, frame.Refund, ErrWriteProtection
			}
			beneficiaryWord, err := st.popUint64()
			if err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}

			beneficiary := contractAddressFromWord(beneficiaryWord)
			acct, ok := state.GetAccount(frame.StorageTarget)
			if ok && acct.Balance > 0 {
				if err := state.AddBalance(beneficiary, acct.Balance); err != nil {
					return nil, nil, false, steps, frame.Refund, err
				}
			}
			if err := state.SetBalance(frame.StorageTarget, 0); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if err := state.SetCode(frame.StorageTarget, nil); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			if err := state.ClearStorage(frame.StorageTarget); err != nil {
				return nil, nil, false, steps, frame.Refund, err
			}
			return []byte{}, logs, false, steps, frame.Refund, nil

		default:
			return nil, nil, false, steps, frame.Refund, ErrInvalidOpcode
		}

		pc++
	}

	return []byte{}, logs, false, steps, frame.Refund, nil
}

func executeCreate(state *vmstate.State, ctx ExecutionContext, frame *callFrame, mem *memory, st *stack) (bool, []byte, error) {
	if frame.Static {
		return false, nil, ErrWriteProtection
	}

	value, err := st.pop()
	if err != nil {
		return false, nil, err
	}
	offset, err := st.popUint64()
	if err != nil {
		return false, nil, err
	}
	size, err := st.popUint64()
	if err != nil {
		return false, nil, err
	}

	frame.CreateCounter++
	newAddress := deriveCreateAddress(frame.StorageTarget, frame.CreateCounter)

	initCode := mem.slice(offset, size)
	if _, err := state.EnsureAccount(newAddress); err != nil {
		return false, nil, err
	}

	runtimeCode, _, reverted, _, _, err := executeBytecode(
		state,
		ctx,
		callFrame{
			Contract:      newAddress,
			StorageTarget: newAddress,
			Caller:        frame.StorageTarget,
			Value:         value.Uint64(),
			Calldata:      []byte{},
			Depth:         frame.Depth + 1,
			Static:        false,
			CreateCounter: 0,
			GasLimit:      forwardedGas(frame.GasLimit, frame.GasLimit),
		},
		initCode,
	)
	if err != nil {
		return false, nil, err
	}
	if reverted {
		return false, runtimeCode, nil
	}
	if len(runtimeCode) > MaxRuntimeCodeSize {
		return false, nil, ErrCodeSizeExceeded
	}
	if err := state.SetCode(newAddress, runtimeCode); err != nil {
		return false, nil, err
	}
	return true, addressToReturnData(newAddress), nil
}

func executeCreate2(state *vmstate.State, ctx ExecutionContext, frame *callFrame, mem *memory, st *stack) (bool, []byte, error) {
	if frame.Static {
		return false, nil, ErrWriteProtection
	}

	value, err := st.pop()
	if err != nil {
		return false, nil, err
	}
	offset, err := st.popUint64()
	if err != nil {
		return false, nil, err
	}
	size, err := st.popUint64()
	if err != nil {
		return false, nil, err
	}
	salt, err := st.popUint64()
	if err != nil {
		return false, nil, err
	}

	initCode := mem.slice(offset, size)
	newAddress := deriveCreate2Address(frame.StorageTarget, salt, initCode)

	if _, err := state.EnsureAccount(newAddress); err != nil {
		return false, nil, err
	}

	runtimeCode, _, reverted, _, _, err := executeBytecode(
		state,
		ctx,
		callFrame{
			Contract:      newAddress,
			StorageTarget: newAddress,
			Caller:        frame.StorageTarget,
			Value:         value.Uint64(),
			Calldata:      []byte{},
			Depth:         frame.Depth + 1,
			Static:        false,
			CreateCounter: 0,
			GasLimit:      forwardedGas(frame.GasLimit, frame.GasLimit),
		},
		initCode,
	)
	if err != nil {
		return false, nil, err
	}
	if reverted {
		return false, runtimeCode, nil
	}
	if len(runtimeCode) > MaxRuntimeCodeSize {
		return false, nil, ErrCodeSizeExceeded
	}
	if err := state.SetCode(newAddress, runtimeCode); err != nil {
		return false, nil, err
	}
	return true, addressToReturnData(newAddress), nil
}

func executeCall(state *vmstate.State, ctx ExecutionContext, callerFrame callFrame, mem *memory, st *stack) (bool, []byte, []Log, error) {
	gas, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	to, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	value, err := st.pop()
	if err != nil {
		return false, nil, nil, err
	}
	argsOffset, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	argsSize, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	retOffset, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	retSize, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}

	gas = forwardedCallGas(gas, callerFrame.GasLimit, value.Uint64())
	if err := consumeGas(&callerFrame.GasLimit, gas); err != nil {
		return false, nil, nil, err
	}

	if callerFrame.Depth+1 > MaxCallDepth {
		mem.storeBytes(retOffset, make([]byte, int(retSize)))
		return false, []byte{}, []Log{}, nil
	}

	target := contractAddressFromWord(to)
	inputData := mem.slice(argsOffset, argsSize)

	acct, ok := state.GetAccount(target)
	if !ok || len(acct.Code) == 0 {
		mem.storeBytes(retOffset, make([]byte, int(retSize)))
		return true, []byte{}, []Log{}, nil
	}

	returnData, callLogs, reverted, _, _, err := executeBytecode(
		state,
		ctx,
		callFrame{
			Contract:      target,
			StorageTarget: target,
			Caller:        callerFrame.Contract,
			Value:         value.Uint64(),
			Calldata:      inputData,
			Depth:         callerFrame.Depth + 1,
			Static:        false,
			CreateCounter: 0,
			GasLimit:      gas,
		},
		acct.Code,
	)
	if err != nil {
		return false, nil, nil, err
	}

	toWrite := returnData
	if uint64(len(toWrite)) > retSize {
		toWrite = toWrite[:retSize]
	}
	mem.storeBytes(retOffset, toWrite)

	if uint64(len(toWrite)) < retSize {
		padding := make([]byte, int(retSize)-len(toWrite))
		mem.storeBytes(retOffset+uint64(len(toWrite)), padding)
	}

	if reverted {
		return false, returnData, []Log{}, nil
	}

	return true, returnData, callLogs, nil
}

func executeDelegateCall(state *vmstate.State, ctx ExecutionContext, callerFrame callFrame, mem *memory, st *stack) (bool, []byte, []Log, error) {
	gas, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	to, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	argsOffset, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	argsSize, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	retOffset, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	retSize, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}

	gas = forwardedGas(gas, callerFrame.GasLimit)
	if err := consumeGas(&callerFrame.GasLimit, gas); err != nil {
		return false, nil, nil, err
	}

	if callerFrame.Depth+1 > MaxCallDepth {
		mem.storeBytes(retOffset, make([]byte, int(retSize)))
		return false, []byte{}, []Log{}, nil
	}

	target := contractAddressFromWord(to)
	inputData := mem.slice(argsOffset, argsSize)

	acct, ok := state.GetAccount(target)
	if !ok || len(acct.Code) == 0 {
		mem.storeBytes(retOffset, make([]byte, int(retSize)))
		return true, []byte{}, []Log{}, nil
	}

	returnData, callLogs, reverted, _, _, err := executeBytecode(
		state,
		ctx,
		callFrame{
			Contract:      target,
			StorageTarget: callerFrame.StorageTarget,
			Caller:        callerFrame.Caller,
			Value:         callerFrame.Value,
			Calldata:      inputData,
			Depth:         callerFrame.Depth + 1,
			Static:        callerFrame.Static,
			CreateCounter: 0,
			GasLimit:      gas,
		},
		acct.Code,
	)
	if err != nil {
		return false, nil, nil, err
	}

	toWrite := returnData
	if uint64(len(toWrite)) > retSize {
		toWrite = toWrite[:retSize]
	}
	mem.storeBytes(retOffset, toWrite)

	if uint64(len(toWrite)) < retSize {
		padding := make([]byte, int(retSize)-len(toWrite))
		mem.storeBytes(retOffset+uint64(len(toWrite)), padding)
	}

	if reverted {
		return false, returnData, []Log{}, nil
	}

	return true, returnData, callLogs, nil
}

func executeStaticCall(state *vmstate.State, ctx ExecutionContext, callerFrame callFrame, mem *memory, st *stack) (bool, []byte, []Log, error) {
	gas, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	to, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	argsOffset, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	argsSize, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	retOffset, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}
	retSize, err := st.popUint64()
	if err != nil {
		return false, nil, nil, err
	}

	gas = forwardedGas(gas, callerFrame.GasLimit)
	if err := consumeGas(&callerFrame.GasLimit, gas); err != nil {
		return false, nil, nil, err
	}

	if callerFrame.Depth+1 > MaxCallDepth {
		mem.storeBytes(retOffset, make([]byte, int(retSize)))
		return false, []byte{}, []Log{}, nil
	}

	target := contractAddressFromWord(to)
	inputData := mem.slice(argsOffset, argsSize)

	acct, ok := state.GetAccount(target)
	if !ok || len(acct.Code) == 0 {
		mem.storeBytes(retOffset, make([]byte, int(retSize)))
		return true, []byte{}, []Log{}, nil
	}

	returnData, callLogs, reverted, _, _, err := executeBytecode(
		state,
		ctx,
		callFrame{
			Contract:      target,
			StorageTarget: target,
			Caller:        callerFrame.Contract,
			Value:         0,
			Calldata:      inputData,
			Depth:         callerFrame.Depth + 1,
			Static:        true,
			CreateCounter: 0,
			GasLimit:      gas,
		},
		acct.Code,
	)
	if err != nil {
		return false, nil, nil, err
	}

	toWrite := returnData
	if uint64(len(toWrite)) > retSize {
		toWrite = toWrite[:retSize]
	}
	mem.storeBytes(retOffset, toWrite)

	if uint64(len(toWrite)) < retSize {
		padding := make([]byte, int(retSize)-len(toWrite))
		mem.storeBytes(retOffset+uint64(len(toWrite)), padding)
	}

	if reverted {
		return false, returnData, []Log{}, nil
	}

	return true, returnData, callLogs, nil
}

func deriveCreateAddress(creator string, counter uint64) string {
	creatorWord := creatorAddressWord(creator)
	word := ((creatorWord + counter - 1) % 255) + 1
	return fmt.Sprintf("contract%d", word)
}

func deriveCreate2Address(creator string, salt uint64, initCode []byte) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(creator))

	var saltBuf [8]byte
	binary.BigEndian.PutUint64(saltBuf[:], salt)
	_, _ = h.Write(saltBuf[:])

	_, _ = h.Write(initCode)

	word := h.Sum64() % 255
	if word == 0 {
		word = 1
	}
	return fmt.Sprintf("contract%d", word)
}

func creatorAddressWord(creator string) uint64 {
	if word := contractWordFromAddress(creator); word != 0 {
		return word
	}

	var hash uint64 = 2166136261
	for i := 0; i < len(creator); i++ {
		hash ^= uint64(creator[i])
		hash *= 16777619
	}

	word := hash % 255
	if word == 0 {
		word = 1
	}
	return word
}

func addressToReturnData(address string) []byte {
	return []byte{byte(contractWordFromAddress(address))}
}

func returnDataToWord(data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}
	return uint64(data[len(data)-1])
}

func contractAddressFromWord(v uint64) string {
	return fmt.Sprintf("contract%d", v)
}

func contractWordFromAddress(address string) uint64 {
	var index uint64
	n, _ := fmt.Sscanf(address, "contract%d", &index)
	if n == 1 {
		return index
	}
	return 0
}

func wordsForBytes(size uint64) uint64 {
	if size == 0 {
		return 0
	}
	return (size + 31) / 32
}

func copyGasCost(size uint64) uint64 {
	return wordsForBytes(size) * 3
}

func ensureWarmTracking(frame *callFrame) {
	if frame.WarmAccounts == nil {
		frame.WarmAccounts = make(map[string]struct{})
	}
	if frame.WarmStorage == nil {
		frame.WarmStorage = make(map[string]map[string]struct{})
	}
	if frame.Contract != "" {
		frame.WarmAccounts[frame.Contract] = struct{}{}
	}
	if frame.StorageTarget != "" {
		frame.WarmAccounts[frame.StorageTarget] = struct{}{}
	}
	if frame.Caller != "" {
		frame.WarmAccounts[frame.Caller] = struct{}{}
	}
}

func chargeAccountAccessGas(frame *callFrame, gasRemaining *uint64, address string) error {
	if frame.WarmAccounts == nil {
		frame.WarmAccounts = make(map[string]struct{})
	}
	if _, ok := frame.WarmAccounts[address]; ok {
		return consumeGas(gasRemaining, WarmAccessCost)
	}
	frame.WarmAccounts[address] = struct{}{}
	return consumeGas(gasRemaining, ColdAccountAccessCost)
}

func chargeStorageAccessGas(frame *callFrame, gasRemaining *uint64, address, key string) error {
	if frame.WarmStorage == nil {
		frame.WarmStorage = make(map[string]map[string]struct{})
	}
	slots, ok := frame.WarmStorage[address]
	if !ok {
		slots = make(map[string]struct{})
		frame.WarmStorage[address] = slots
	}
	if _, ok := slots[key]; ok {
		return consumeGas(gasRemaining, WarmStorageReadCost)
	}
	slots[key] = struct{}{}
	return consumeGas(gasRemaining, ColdStorageReadCost)
}

func consumeGas(gasRemaining *uint64, cost uint64) error {
	if gasRemaining == nil {
		return ErrOutOfGas
	}
	if *gasRemaining < cost {
		return ErrOutOfGas
	}
	*gasRemaining -= cost
	return nil
}

func calldataLoadWord(calldata []byte, offset uint64) word {
	var out word
	start := int(offset)
	if start >= len(calldata) {
		return zeroWord()
	}
	end := start + 32
	if end > len(calldata) {
		end = len(calldata)
	}
	copy(out[:], calldata[start:end])
	return out
}

func validateJumpDest(code []byte, dest uint64) (int, error) {
	i := int(dest)
	if i < 0 || i >= len(code) {
		return 0, ErrInvalidJumpDest
	}
	if code[i] != OpJUMPDEST {
		return 0, ErrInvalidJumpDest
	}
	return i, nil
}

func storageKey(v uint64) string {
	return fmt.Sprintf("0x%x", v)
}

func storageValue(v uint64) string {
	return fmt.Sprintf("0x%x", v)
}

func parseStorageValue(v string) uint64 {
	var out uint64
	_, _ = fmt.Sscanf(v, "0x%x", &out)
	return out
}

func forwardedGas(requested uint64, available uint64) uint64 {
	cap := available - (available / 64)
	if requested > cap {
		return cap
	}
	return requested
}

func forwardedCallGas(requested uint64, available uint64, value uint64) uint64 {
	gas := forwardedGas(requested, available)
	if value != 0 {
		gas += 2300
	}
	return gas
}
