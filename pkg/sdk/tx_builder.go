package sdk

import (
	"time"
)

func BuildDeployTx(from string, contractCode string, nonce uint64, chainID uint64) DeployTxRequest {
	return DeployTxRequest{
		From:         from,
		Value:        0,
		Fee:          0,
		GasPrice:     1,
		GasLimit:     300000,
		Nonce:        nonce,
		ChainID:      chainID,
		Timestamp:    time.Now().Unix(),
		VMVersion:    1,
		ContractCode: contractCode,
	}
}

func BuildCallTx(from string, address string, contractInput string, nonce uint64, chainID uint64) CallTxRequest {
	return CallTxRequest{
		From:          from,
		Address:       address,
		Value:         0,
		Fee:           0,
		GasPrice:      1,
		GasLimit:      200000,
		Nonce:         nonce,
		ChainID:       chainID,
		Timestamp:     time.Now().Unix(),
		VMVersion:     1,
		ContractInput: contractInput,
	}
}
