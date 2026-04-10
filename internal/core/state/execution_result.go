package state

import (
	"silachain/pkg/types"
)

const TransferIntrinsicGas types.Gas = 21000

type Result struct {
	TxHash         types.Hash      `json:"tx_hash"`
	From           types.Address   `json:"from"`
	To             types.Address   `json:"to"`
	Value          types.Amount    `json:"value"`
	Fee            types.Amount    `json:"fee"`
	TotalCost      types.Amount    `json:"total_cost"`
	GasUsed        types.Gas       `json:"gas_used"`
	AppliedNonce   types.Nonce     `json:"applied_nonce"`
	Success        bool            `json:"success"`
	Error          string          `json:"error,omitempty"`
	Logs           []types.Event   `json:"logs,omitempty"`
	ReturnData     string          `json:"return_data,omitempty"`
	RevertData     string          `json:"revert_data,omitempty"`
	CreatedAddress types.Address   `json:"created_address,omitempty"`
	Timestamp      types.Timestamp `json:"timestamp"`
}

func SuccessResult(
	txHash types.Hash,
	from types.Address,
	to types.Address,
	value types.Amount,
	fee types.Amount,
	totalCost types.Amount,
	gasUsed types.Gas,
	appliedNonce types.Nonce,
	logs []types.Event,
	returnData string,
	revertData string,
	createdAddress types.Address,
	ts types.Timestamp,
) Result {
	return Result{
		TxHash:         txHash,
		From:           from,
		To:             to,
		Value:          value,
		Fee:            fee,
		TotalCost:      totalCost,
		GasUsed:        gasUsed,
		AppliedNonce:   appliedNonce,
		Success:        true,
		Logs:           logs,
		ReturnData:     returnData,
		RevertData:     revertData,
		CreatedAddress: createdAddress,
		Timestamp:      ts,
	}
}

func FailedResult(
	txHash types.Hash,
	from types.Address,
	to types.Address,
	value types.Amount,
	fee types.Amount,
	totalCost types.Amount,
	gasUsed types.Gas,
	appliedNonce types.Nonce,
	logs []types.Event,
	returnData string,
	revertData string,
	createdAddress types.Address,
	ts types.Timestamp,
	err error,
) Result {
	msg := ""
	if err != nil {
		msg = err.Error()
	}

	return Result{
		TxHash:         txHash,
		From:           from,
		To:             to,
		Value:          value,
		Fee:            fee,
		TotalCost:      totalCost,
		GasUsed:        gasUsed,
		AppliedNonce:   appliedNonce,
		Success:        false,
		Error:          msg,
		Logs:           logs,
		ReturnData:     returnData,
		RevertData:     revertData,
		CreatedAddress: createdAddress,
		Timestamp:      ts,
	}
}
