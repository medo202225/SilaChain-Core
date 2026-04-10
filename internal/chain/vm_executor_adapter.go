package chain

import (
	"silachain/internal/core/state"
	"silachain/internal/core/vm"
	"silachain/pkg/types"
)

type transitionVMExecutor struct {
	interpreter *vm.Interpreter
	host        *vm.StateHost
}

func newTransitionVMExecutor(
	codeRegistry *state.ContractCodeRegistry,
	storage *state.ContractStorage,
	journal *state.Journal,
) *transitionVMExecutor {
	host := vm.NewRegistryBackedStateHost(codeRegistry, storage, journal)

	return &transitionVMExecutor{
		interpreter: vm.NewInterpreterWithHost(vm.DefaultLimits(), host),
		host:        host,
	}
}

func (e *transitionVMExecutor) ExecuteContract(input state.VMExecutionInput) state.VMExecutionOutput {
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

	logs := make([]types.Event, 0, len(result.Logs))
	for _, lg := range result.Logs {
		logs = append(logs, types.Event{
			Address: lg.Address,
			Name:    "VMLog",
			Topics:  append([]string(nil), lg.Topics...),
			Data: map[string]string{
				"data": string(lg.Data),
			},
		})
	}

	return state.VMExecutionOutput{
		Success:        result.Succeeded(),
		GasUsed:        result.GasUsed,
		ReturnData:     append([]byte(nil), result.ReturnData...),
		RevertData:     append([]byte(nil), result.RevertData...),
		CreatedAddress: result.CreatedAddress,
		Logs:           logs,
		Err:            result.Err,
	}
}

func (e *transitionVMExecutor) GetContractCode(address string) []byte {
	if e == nil || e.host == nil {
		return nil
	}
	return e.host.GetCode(address)
}
