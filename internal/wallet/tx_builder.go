package wallet

import (
	"strings"

	coretypes "silachain/internal/core/types"
	pkgtypes "silachain/pkg/types"
)

const (
	WalletTransferGas        uint64 = 21000
	WalletContractBaseGas    uint64 = 30000
	WalletContractReadGas    uint64 = 5000
	WalletContractWriteGas   uint64 = 15000
	WalletContractDeleteGas  uint64 = 12000
	WalletContractCounterGas uint64 = 10000
)

func contractCallGasLimit(method string) uint64 {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "get":
		return WalletContractBaseGas + WalletContractReadGas
	case "set":
		return WalletContractBaseGas + WalletContractWriteGas
	case "delete":
		return WalletContractBaseGas + WalletContractDeleteGas
	case "inc", "dec":
		return WalletContractBaseGas + WalletContractCounterGas
	default:
		return WalletContractBaseGas + WalletContractWriteGas
	}
}

func (w *Wallet) BuildSignedTransferTx(
	to string,
	value uint64,
	fee uint64,
	nonce uint64,
	chainID uint64,
	timestamp int64,
) (*coretypes.Transaction, error) {
	t := &coretypes.Transaction{
		Type:      coretypes.TypeTransfer,
		From:      pkgtypes.Address(w.Address),
		To:        pkgtypes.Address(to),
		Value:     pkgtypes.Amount(value),
		Fee:       pkgtypes.Amount(fee),
		GasPrice:  0,
		GasLimit:  pkgtypes.Gas(WalletTransferGas),
		Nonce:     pkgtypes.Nonce(nonce),
		ChainID:   pkgtypes.ChainID(chainID),
		Timestamp: pkgtypes.Timestamp(timestamp),
		PublicKey: w.PublicKeyHex,
	}

	if err := w.SignTransaction(t); err != nil {
		return nil, err
	}

	return t, nil
}

func (w *Wallet) BuildSignedTransferTxWithGasPrice(
	to string,
	value uint64,
	gasPrice uint64,
	gasLimit uint64,
	nonce uint64,
	chainID uint64,
	timestamp int64,
) (*coretypes.Transaction, error) {
	t := &coretypes.Transaction{
		Type:      coretypes.TypeTransfer,
		From:      pkgtypes.Address(w.Address),
		To:        pkgtypes.Address(to),
		Value:     pkgtypes.Amount(value),
		Fee:       0,
		GasPrice:  pkgtypes.Amount(gasPrice),
		GasLimit:  pkgtypes.Gas(gasLimit),
		Nonce:     pkgtypes.Nonce(nonce),
		ChainID:   pkgtypes.ChainID(chainID),
		Timestamp: pkgtypes.Timestamp(timestamp),
		PublicKey: w.PublicKeyHex,
	}

	if err := w.SignTransaction(t); err != nil {
		return nil, err
	}

	return t, nil
}

func (w *Wallet) BuildSignedContractCallTx(
	to string,
	fee uint64,
	nonce uint64,
	chainID uint64,
	timestamp int64,
	method string,
	key string,
	value string,
) (*coretypes.Transaction, error) {
	t := &coretypes.Transaction{
		Type:       coretypes.TypeContractCall,
		From:       pkgtypes.Address(w.Address),
		To:         pkgtypes.Address(to),
		Value:      0,
		Fee:        pkgtypes.Amount(fee),
		GasPrice:   0,
		GasLimit:   pkgtypes.Gas(contractCallGasLimit(method)),
		Nonce:      pkgtypes.Nonce(nonce),
		ChainID:    pkgtypes.ChainID(chainID),
		Timestamp:  pkgtypes.Timestamp(timestamp),
		CallMethod: method,
		CallKey:    key,
		CallValue:  value,
		PublicKey:  w.PublicKeyHex,
	}

	if err := w.SignTransaction(t); err != nil {
		return nil, err
	}

	return t, nil
}
