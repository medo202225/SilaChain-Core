package execution

import (
	"silachain/internal/core/state"
	"silachain/internal/core/vm"
	"silachain/pkg/types"
)

type VMBridge struct {
	interpreter *vm.Interpreter
	host        *vm.StateHost
}

func NewVMBridge(
	codeRegistry *state.ContractCodeRegistry,
	storage *state.ContractStorage,
	journal *state.Journal,
) *VMBridge {
	host := vm.NewRegistryBackedStateHost(codeRegistry, storage, journal)

	return &VMBridge{
		interpreter: vm.NewInterpreterWithHost(vm.DefaultLimits(), host),
		host:        host,
	}
}

func (b *VMBridge) Execute(
	ctx vm.ExecutionContext,
	code []byte,
) vm.ExecutionResult {
	if b == nil || b.interpreter == nil {
		return vm.FaultResult(vm.ErrExecutionAborted, 0, ctx.GasRemaining, "")
	}

	return b.interpreter.Run(ctx, code)
}

func (b *VMBridge) ExecuteContract(input state.VMExecutionInput) state.VMExecutionOutput {
	if b == nil || b.interpreter == nil {
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

	result := b.interpreter.Run(ctx, input.Code)

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

func (b *VMBridge) GetContractCode(address string) []byte {
	if b == nil || b.host == nil {
		return nil
	}
	return b.host.GetCode(address)
}

func (b *VMBridge) Host() *vm.StateHost {
	if b == nil {
		return nil
	}
	return b.host
}
