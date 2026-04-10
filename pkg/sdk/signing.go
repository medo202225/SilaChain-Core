package sdk

import (
	"encoding/json"
	"errors"

	chaincrypto "silachain/pkg/crypto"
)

var ErrMissingPrivateKey = errors.New("sdk: missing private key")

type deploySigningPayload struct {
	Type         string `json:"type"`
	From         string `json:"from"`
	To           string `json:"to"`
	Value        uint64 `json:"value"`
	Fee          uint64 `json:"fee"`
	GasPrice     uint64 `json:"gas_price"`
	GasLimit     uint64 `json:"gas_limit"`
	Nonce        uint64 `json:"nonce"`
	ChainID      uint64 `json:"chain_id"`
	Timestamp    int64  `json:"timestamp"`
	VMVersion    uint16 `json:"vm_version"`
	ContractCode string `json:"contract_code,omitempty"`
}

type callSigningPayload struct {
	Type          string `json:"type"`
	From          string `json:"from"`
	To            string `json:"to"`
	Value         uint64 `json:"value"`
	Fee           uint64 `json:"fee"`
	GasPrice      uint64 `json:"gas_price"`
	GasLimit      uint64 `json:"gas_limit"`
	Nonce         uint64 `json:"nonce"`
	ChainID       uint64 `json:"chain_id"`
	Timestamp     int64  `json:"timestamp"`
	VMVersion     uint16 `json:"vm_version"`
	ContractInput string `json:"contract_input,omitempty"`
}

func SignDeployTx(req DeployTxRequest, privateKeyHex string) (DeployTxRequest, error) {
	if privateKeyHex == "" {
		return req, ErrMissingPrivateKey
	}

	payload := deploySigningPayload{
		Type:         "contract_deploy",
		From:         req.From,
		To:           "",
		Value:        req.Value,
		Fee:          req.Fee,
		GasPrice:     req.GasPrice,
		GasLimit:     req.GasLimit,
		Nonce:        req.Nonce,
		ChainID:      req.ChainID,
		Timestamp:    req.Timestamp,
		VMVersion:    req.VMVersion,
		ContractCode: req.ContractCode,
	}

	hashHex, sigHex, err := signPayload(payload, privateKeyHex)
	if err != nil {
		return req, err
	}

	req.Hash = hashHex
	req.Signature = sigHex
	return req, nil
}

func SignCallTx(req CallTxRequest, privateKeyHex string) (CallTxRequest, error) {
	if privateKeyHex == "" {
		return req, ErrMissingPrivateKey
	}

	payload := callSigningPayload{
		Type:          "contract_call",
		From:          req.From,
		To:            req.Address,
		Value:         req.Value,
		Fee:           req.Fee,
		GasPrice:      req.GasPrice,
		GasLimit:      req.GasLimit,
		Nonce:         req.Nonce,
		ChainID:       req.ChainID,
		Timestamp:     req.Timestamp,
		VMVersion:     req.VMVersion,
		ContractInput: req.ContractInput,
	}

	hashHex, sigHex, err := signPayload(payload, privateKeyHex)
	if err != nil {
		return req, err
	}

	req.Hash = hashHex
	req.Signature = sigHex
	return req, nil
}

func signPayload(payload any, privateKeyHex string) (string, string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	hash := chaincrypto.HashBytes(raw)
	priv, err := chaincrypto.HexToPrivateKey(privateKeyHex)
	if err != nil {
		return "", "", err
	}

	sig, err := chaincrypto.SignHashHex(priv, hash)
	if err != nil {
		return "", "", err
	}

	return hash, sig, nil
}
