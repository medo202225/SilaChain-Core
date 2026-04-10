package state_test

import (
	"encoding/hex"
	"testing"

	"silachain/internal/accounts"
	"silachain/internal/core/state"
	coretypes "silachain/internal/core/types"
	"silachain/internal/core/vm"
	pkgtypes "silachain/pkg/types"
)

type testVMExecutor struct {
	interpreter *vm.Interpreter
	host        *vm.StateHost
}

func newTestVMExecutor(
	codeRegistry *state.ContractCodeRegistry,
	storage *state.ContractStorage,
	journal *state.Journal,
) *testVMExecutor {
	host := vm.NewRegistryBackedStateHost(codeRegistry, storage, journal)

	return &testVMExecutor{
		interpreter: vm.NewInterpreterWithHost(vm.DefaultLimits(), host),
		host:        host,
	}
}

func (e *testVMExecutor) ExecuteContract(input state.VMExecutionInput) state.VMExecutionOutput {
	if e == nil || e.interpreter == nil {
		return state.VMExecutionOutput{
			Success: false,
			Err:     vm.ErrExecutionAborted,
		}
	}

	ctx := vm.ExecutionContext{
		VMVersion:    input.VMVersion,
		GasRemaining: input.GasRemaining,
		ContractAddr: input.ContractAddr,
		StorageAddr:  input.StorageAddr,
		CodeAddr:     input.CodeAddr,
		Caller:       input.Caller,
		Origin:       input.Origin,
		CallValue:    input.CallValue,
		Input:        input.Input,
	}

	result := e.interpreter.Run(ctx, input.Code)
	return state.VMExecutionOutput{
		Success: result.Succeeded(),
		Err:     result.Err,
	}
}

func (e *testVMExecutor) GetContractCode(address string) []byte {
	if e == nil || e.host == nil {
		return nil
	}
	return e.host.GetCode(address)
}

func TestTransitionVMContractCallEndToEnd(t *testing.T) {
	accountManager := accounts.NewManager()

	from := pkgtypes.Address("alice")
	to := pkgtypes.Address("contract1")

	fromAcc, err := accountManager.RegisterAccount(from, "pub-alice")
	if err != nil {
		t.Fatalf("register from account: %v", err)
	}
	toAcc, err := accountManager.RegisterAccount(to, "pub-contract1")
	if err != nil {
		t.Fatalf("register to account: %v", err)
	}

	fromAcc.Credit(pkgtypes.Amount(1_000_000))
	toAcc.Credit(pkgtypes.Amount(0))

	manager := state.NewManager(accountManager)
	transition := state.NewTransition(manager, nil)

	vmExec := newTestVMExecutor(
		transition.CodeRegistry(),
		transition.Storage(),
		transition.Journal(),
	)
	transition.SetVMExecutor(vmExec)

	contractCode := []byte{
		vm.OpPush1, 0x2a,
		vm.OpPush1, 0x01,
		vm.OpSStore,
		vm.OpStop,
	}
	if err := vmExec.host.SetCode(string(to), contractCode); err != nil {
		t.Fatalf("set contract code: %v", err)
	}

	transaction := &coretypes.Transaction{
		Type:          coretypes.TypeContractCall,
		From:          from,
		To:            to,
		Value:         0,
		GasLimit:      pkgtypes.Gas(100000),
		GasPrice:      1,
		Nonce:         0,
		ChainID:       1,
		Timestamp:     1,
		VMVersion:     1,
		ContractInput: hex.EncodeToString([]byte{}),
	}

	result, err := transition.ApplyTransactionWithResult(transaction)
	if err != nil {
		t.Fatalf("apply transaction: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success result")
	}

	stored, ok := transition.Storage().Get(to, string(vm.WordToBytes32(vm.NewWordFromUint64(1))))
	if !ok {
		t.Fatalf("expected storage value")
	}
	if stored == "" {
		t.Fatalf("expected non-empty storage value")
	}
}

func TestTransitionVMContractDeployWithoutExecutorFails(t *testing.T) {
	accountManager := accounts.NewManager()

	from := pkgtypes.Address("alice")
	fromAcc, err := accountManager.RegisterAccount(from, "pub-alice")
	if err != nil {
		t.Fatalf("register from account: %v", err)
	}
	fromAcc.Credit(pkgtypes.Amount(1_000_000))

	manager := state.NewManager(accountManager)
	transition := state.NewTransition(manager, nil)

	transaction := &coretypes.Transaction{
		Type:         coretypes.TypeContractDeploy,
		From:         from,
		To:           "",
		Value:        0,
		GasLimit:     pkgtypes.Gas(100000),
		GasPrice:     1,
		Nonce:        0,
		ChainID:      1,
		Timestamp:    1,
		VMVersion:    1,
		ContractCode: hex.EncodeToString([]byte{byte(vm.OpStop)}),
	}

	_, err = transition.ApplyTransactionWithResult(transaction)
	if err == nil {
		t.Fatalf("expected error without vm executor")
	}
}
