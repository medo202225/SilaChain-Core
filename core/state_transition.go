package core

import (
	"fmt"

	statecore "silachain/core/state"
)

const TxGas uint64 = 21000

var (
	ErrNilStateTransition = fmt.Errorf("core: nil state transition")
	ErrNilStateDB         = fmt.Errorf("core: nil state db")
	ErrNilGasPool         = fmt.Errorf("core: nil gas pool")
	ErrEmptySender        = fmt.Errorf("core: empty sender")
	ErrNonceTooLow        = fmt.Errorf("core: nonce mismatch")
	ErrInsufficientFunds  = fmt.Errorf("core: insufficient funds")
	ErrIntrinsicGas       = fmt.Errorf("core: intrinsic gas too high")
)

type ExecutionResult struct {
	UsedGas      uint64
	MaxUsedGas   uint64
	RefundedGas  uint64
	RemainingGas uint64
	PurchaseGas  uint64
	PurchaseCost uint64
	Err          error
	ReturnData   []byte
}

func (result *ExecutionResult) Failed() bool {
	if result == nil {
		return true
	}
	return result.Err != nil
}

func (result *ExecutionResult) Return() []byte {
	if result == nil || result.Err != nil {
		return nil
	}
	out := make([]byte, len(result.ReturnData))
	copy(out, result.ReturnData)
	return out
}

func (result *ExecutionResult) Revert() []byte {
	if result == nil || result.Err == nil {
		return nil
	}
	out := make([]byte, len(result.ReturnData))
	copy(out, result.ReturnData)
	return out
}

type stateTransition struct {
	gp                *GasPool
	msg               Message
	state             *statecore.StateDB
	initialGas        uint64
	gasRemaining      uint64
	effectiveGasPrice uint64
}

func ApplyMessage(state *statecore.StateDB, gp *GasPool, msg Message) (*ExecutionResult, error) {
	if gp == nil {
		gp = NewGasPool(msg.GasLimit)
	}
	st := &stateTransition{
		gp:    gp,
		msg:   msg,
		state: state,
	}
	return st.TransitionDb()
}

func ApplyTransaction(ctx BlockContext, state *statecore.StateDB, gp *GasPool, txHash string, msg Message) (Receipt, error) {
	result, err := ApplyMessage(state, gp, msg)
	receipt := Receipt{
		TxHash:       txHash,
		From:         msg.From,
		To:           msg.ToAddress(),
		Nonce:        msg.Nonce,
		GasUsed:      0,
		RefundedGas:  0,
		RemainingGas: 0,
		Success:      false,
		ErrorText:    "",
	}
	if result != nil {
		receipt.GasUsed = result.UsedGas
		receipt.RefundedGas = result.RefundedGas
		receipt.RemainingGas = result.RemainingGas
		receipt.Success = !result.Failed()
	}
	if err != nil {
		receipt.Success = false
		receipt.ErrorText = err.Error()
		return receipt, err
	}
	_ = ctx
	return receipt, nil
}

func (st *stateTransition) preCheck() error {
	if st == nil {
		return ErrNilStateTransition
	}
	if st.state == nil {
		return ErrNilStateDB
	}
	if st.gp == nil {
		return ErrNilGasPool
	}
	if st.msg.From == "" {
		return ErrEmptySender
	}

	st.initialGas = st.msg.GasLimit
	st.gasRemaining = st.msg.GasLimit
	st.effectiveGasPrice = st.msg.EffectiveGasPrice()

	if !st.msg.SkipAccountLoad {
		st.state.EnsureAccount(st.msg.From)
	}

	if !st.msg.SkipNonceChecks {
		currentNonce := st.state.GetNonce(st.msg.From)
		if currentNonce != st.msg.Nonce {
			return fmt.Errorf("%w: got=%d want=%d", ErrNonceTooLow, st.msg.Nonce, currentNonce)
		}
	}

	intrinsicGas := IntrinsicGas(st.msg)
	if st.msg.GasLimit < intrinsicGas {
		return fmt.Errorf("%w: gaslimit=%d intrinsic=%d", ErrIntrinsicGas, st.msg.GasLimit, intrinsicGas)
	}

	required := st.msg.Value + st.purchaseCost()
	if st.state.GetBalance(st.msg.From) < required {
		return fmt.Errorf("%w: balance=%d required=%d", ErrInsufficientFunds, st.state.GetBalance(st.msg.From), required)
	}
	return nil
}

func (st *stateTransition) purchaseCost() uint64 {
	return st.msg.GasLimit * st.effectiveGasPrice
}

func (st *stateTransition) buyGas() error {
	if err := st.gp.SubGas(st.msg.GasLimit); err != nil {
		return err
	}
	st.state.SetBalance(st.msg.From, st.state.GetBalance(st.msg.From)-st.purchaseCost())
	return nil
}

func (st *stateTransition) useGas(amount uint64) error {
	if st.gasRemaining < amount {
		return fmt.Errorf("%w: gaslimit=%d intrinsic=%d", ErrIntrinsicGas, st.initialGas, amount)
	}
	st.gasRemaining -= amount
	return nil
}

func (st *stateTransition) refundGas() uint64 {
	if st.gasRemaining == 0 {
		return 0
	}
	refund := st.gasRemaining * st.effectiveGasPrice
	st.state.SetBalance(st.msg.From, st.state.GetBalance(st.msg.From)+refund)
	return st.gasRemaining
}

func (st *stateTransition) returnGas() error {
	return st.gp.ReturnGas(st.gasRemaining)
}

func (st *stateTransition) gasUsed() uint64 {
	return st.initialGas - st.gasRemaining
}

func (st *stateTransition) canTransferValue() bool {
	if st.msg.Value == 0 {
		return true
	}
	return st.state.GetBalance(st.msg.From) >= st.msg.Value
}

func (st *stateTransition) transferValue() error {
	if st.msg.Value == 0 || st.msg.To == nil {
		return nil
	}
	if !st.canTransferValue() {
		return fmt.Errorf("%w: value=%d", ErrInsufficientFunds, st.msg.Value)
	}
	to := st.msg.ToAddress()
	st.state.EnsureAccount(to)
	st.state.SetBalance(st.msg.From, st.state.GetBalance(st.msg.From)-st.msg.Value)
	st.state.SetBalance(to, st.state.GetBalance(to)+st.msg.Value)
	return nil
}

func (st *stateTransition) writeData() []byte {
	if st.msg.To == nil || len(st.msg.Data) == 0 {
		return nil
	}
	st.state.SetState(st.msg.ToAddress(), "calldata", string(st.msg.Data))
	out := make([]byte, len(st.msg.Data))
	copy(out, st.msg.Data)
	return out
}

func (st *stateTransition) TransitionDb() (*ExecutionResult, error) {
	if err := st.preCheck(); err != nil {
		return &ExecutionResult{Err: err}, err
	}
	if err := st.buyGas(); err != nil {
		return &ExecutionResult{Err: err}, err
	}

	intrinsicGas := IntrinsicGas(st.msg)
	if err := st.useGas(intrinsicGas); err != nil {
		return &ExecutionResult{Err: err}, err
	}

	if err := st.transferValue(); err != nil {
		return &ExecutionResult{Err: err}, err
	}
	returnData := st.writeData()
	if !st.msg.SkipNonceChecks {
		st.state.SetNonce(st.msg.From, st.msg.Nonce+1)
	}

	refundedGas := st.refundGas()
	if err := st.returnGas(); err != nil {
		return &ExecutionResult{Err: err}, err
	}

	result := &ExecutionResult{
		UsedGas:      st.gasUsed(),
		MaxUsedGas:   st.gasUsed(),
		RefundedGas:  refundedGas,
		RemainingGas: st.gasRemaining,
		PurchaseGas:  st.initialGas,
		PurchaseCost: st.purchaseCost(),
		Err:          nil,
		ReturnData:   returnData,
	}
	return result, nil
}

func IntrinsicGas(msg Message) uint64 {
	if len(msg.Data) == 0 {
		return TxGas
	}
	return TxGas + uint64(len(msg.Data))*16
}

func PoolTxToMessage(txHash string, from string, to string, nonce, value, gasLimit, gasPrice uint64, data []byte) Message {
	var toPtr *string
	if to != "" {
		toPtr = &to
	}
	return Message{
		From:                  from,
		To:                    toPtr,
		Nonce:                 nonce,
		Value:                 value,
		GasLimit:              gasLimit,
		GasPrice:              gasPrice,
		GasFeeCap:             gasPrice,
		GasTipCap:             0,
		Data:                  data,
		AccessList:            nil,
		BlobGasFeeCap:         0,
		BlobHashes:            nil,
		SkipNonceChecks:       false,
		SkipTransactionChecks: false,
		SkipAccountLoad:       false,
	}
}
