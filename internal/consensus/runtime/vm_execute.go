package runtime

import (
	"errors"

	"silachain/internal/consensus/vmexec"
)

var (
	ErrNilRuntimeVMExecute = errors.New("runtime: nil runtime vm execute")
	ErrNilStateVMExecute   = errors.New("runtime: nil state vm execute")
)

type VMExecuteRequest struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    uint64 `json:"value"`
	GasLimit uint64 `json:"gasLimit"`
	Data     []byte `json:"data"`
}

type VMExecuteLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    []byte   `json:"data"`
}

type VMExecuteResult struct {
	Success        bool           `json:"success"`
	Reverted       bool           `json:"reverted"`
	GasUsed        uint64         `json:"gasUsed"`
	ReturnData     []byte         `json:"returnData"`
	Logs           []VMExecuteLog `json:"logs"`
	CodeExecuted   bool           `json:"codeExecuted"`
	CodeSize       int            `json:"codeSize"`
	CreatedAccount bool           `json:"createdAccount"`
	Steps          int            `json:"steps"`
}

func (r *Runtime) VMExecute(req VMExecuteRequest) (VMExecuteResult, error) {
	if r == nil {
		return VMExecuteResult{}, ErrNilRuntimeVMExecute
	}
	if r.state == nil || r.state.vm == nil {
		return VMExecuteResult{}, ErrNilStateVMExecute
	}

	exec, err := vmexec.New(r.state.vm)
	if err != nil {
		return VMExecuteResult{}, err
	}

	head := r.state.Head()

	result, err := exec.Execute(vmexec.ExecutionContext{
		BlockNumber: head.Number,
		BlockHash:   head.Hash,
		Timestamp:   0,
		GasLimit:    r.cfg.GasLimit,
		BaseFee:     head.BaseFee,
		ChainID:     1001,
	}, vmexec.Message{
		From:     req.From,
		To:       req.To,
		Value:    req.Value,
		GasLimit: req.GasLimit,
		Data:     req.Data,
	})
	if err != nil {
		return VMExecuteResult{}, err
	}

	logs := make([]VMExecuteLog, 0, len(result.Logs))
	for _, log := range result.Logs {
		topics := make([]string, len(log.Topics))
		copy(topics, log.Topics)

		data := make([]byte, len(log.Data))
		copy(data, log.Data)

		logs = append(logs, VMExecuteLog{
			Address: log.Address,
			Topics:  topics,
			Data:    data,
		})
	}

	return VMExecuteResult{
		Success:        result.Success,
		Reverted:       result.Reverted,
		GasUsed:        result.GasUsed,
		ReturnData:     result.ReturnData,
		Logs:           logs,
		CodeExecuted:   result.CodeExecuted,
		CodeSize:       result.CodeSize,
		CreatedAccount: result.CreatedAccount,
		Steps:          result.Steps,
	}, nil
}
