package chain

import (
	"encoding/hex"

	"silachain/internal/core/vm"
	"silachain/pkg/types"
)

type ReadOnlyCallResult struct {
	Success    bool   `json:"success"`
	GasUsed    uint64 `json:"gas_used"`
	ReturnData string `json:"return_data,omitempty"`
	RevertData string `json:"revert_data,omitempty"`
	Error      string `json:"error,omitempty"`
}

func (bc *Blockchain) ReadOnlyCall(address types.Address, inputHex string, vmVersion uint16, gasLimit uint64) (ReadOnlyCallResult, error) {
	if bc == nil || bc.transition == nil {
		return ReadOnlyCallResult{}, ErrNilBlockchain
	}

	host := vm.NewRegistryBackedStateHost(
		bc.transition.CodeRegistry(),
		bc.transition.Storage(),
		bc.transition.Journal(),
	)
	interpreter := vm.NewInterpreterWithHost(vm.DefaultLimits(), host)

	code := host.GetCode(string(address))
	if len(code) == 0 {
		return ReadOnlyCallResult{
			Success: false,
			Error:   "contract code not found",
		}, nil
	}

	var input []byte
	if inputHex != "" {
		decoded, err := hex.DecodeString(inputHex)
		if err != nil {
			return ReadOnlyCallResult{}, err
		}
		input = decoded
	}

	if gasLimit == 0 {
		gasLimit = 100000
	}
	if vmVersion == 0 {
		vmVersion = 1
	}

	result := interpreter.Run(vm.ExecutionContext{
		VMVersion:    vmVersion,
		GasRemaining: gasLimit,
		ContractAddr: string(address),
		StorageAddr:  string(address),
		CodeAddr:     string(address),
		Caller:       "",
		Origin:       "",
		CallValue:    0,
		Input:        input,
		Static:       true,
	}, code)

	out := ReadOnlyCallResult{
		Success:    result.Succeeded(),
		GasUsed:    result.GasUsed,
		ReturnData: hex.EncodeToString(result.ReturnData),
		RevertData: hex.EncodeToString(result.RevertData),
	}
	if result.Err != nil {
		out.Error = result.Err.Error()
	}

	return out, nil
}
