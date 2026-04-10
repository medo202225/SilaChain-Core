package block

// CANONICAL OWNERSHIP: block hashing and current block commitment helpers.
// Current tx and receipt roots are deterministic hash-based commitments and are not yet trie-backed.

import (
	coretypes "silachain/internal/core/types"
	"silachain/pkg/crypto"
	pkgtypes "silachain/pkg/types"
)

type headerHashPayload struct {
	Height      pkgtypes.Height    `json:"height"`
	ParentHash  pkgtypes.Hash      `json:"parent_hash"`
	StateRoot   pkgtypes.Hash      `json:"state_root"`
	TxRoot      pkgtypes.Hash      `json:"tx_root"`
	ReceiptRoot pkgtypes.Hash      `json:"receipt_root"`
	Timestamp   pkgtypes.Timestamp `json:"timestamp"`
	Proposer    pkgtypes.Address   `json:"proposer"`
	GasUsed     pkgtypes.Gas       `json:"gas_used"`
	GasLimit    pkgtypes.Gas       `json:"gas_limit"`
	TxCount     uint64             `json:"tx_count"`
}

type receiptLogPayload struct {
	Address string            `json:"address"`
	Topics  []string          `json:"topics"`
	Data    map[string]string `json:"data"`
}

type receiptRootPayload struct {
	TxHash            pkgtypes.Hash       `json:"tx_hash"`
	Success           bool                `json:"success"`
	GasUsed           pkgtypes.Gas        `json:"gas_used"`
	CumulativeGasUsed pkgtypes.Gas        `json:"cumulative_gas_used"`
	Error             string              `json:"error,omitempty"`
	ReturnData        string              `json:"return_data,omitempty"`
	RevertData        string              `json:"revert_data,omitempty"`
	CreatedAddress    pkgtypes.Address    `json:"created_address,omitempty"`
	Logs              []receiptLogPayload `json:"logs,omitempty"`
}

func HeaderHash(h coretypes.Header) (pkgtypes.Hash, error) {
	payload := headerHashPayload{
		Height:      h.Height,
		ParentHash:  h.ParentHash,
		StateRoot:   h.StateRoot,
		TxRoot:      h.TxRoot,
		ReceiptRoot: h.ReceiptRoot,
		Timestamp:   h.Timestamp,
		Proposer:    h.Proposer,
		GasUsed:     h.GasUsed,
		GasLimit:    h.GasLimit,
		TxCount:     h.TxCount,
	}

	sum, err := crypto.HashJSON(payload)
	if err != nil {
		return "", err
	}
	return pkgtypes.Hash(sum), nil
}

func TxRootHash(txs []coretypes.Transaction) (pkgtypes.Hash, error) {
	payload := make([]pkgtypes.Hash, 0, len(txs))
	for _, item := range txs {
		payload = append(payload, item.Hash)
	}

	sum, err := crypto.HashJSON(payload)
	if err != nil {
		return "", err
	}

	return pkgtypes.Hash(sum), nil
}

func ReceiptRootHash(receipts []coretypes.Receipt) (pkgtypes.Hash, error) {
	payload := make([]receiptRootPayload, 0, len(receipts))

	for _, item := range receipts {
		logs := make([]receiptLogPayload, 0, len(item.Logs))
		for _, lg := range item.Logs {
			logs = append(logs, receiptLogPayload{
				Address: lg.Address,
				Topics:  lg.Topics,
				Data:    lg.Data,
			})
		}

		payload = append(payload, receiptRootPayload{
			TxHash:            item.TxHash,
			Success:           item.Success,
			GasUsed:           item.GasUsed,
			CumulativeGasUsed: item.CumulativeGasUsed,
			Error:             item.Error,
			ReturnData:        item.ReturnData,
			RevertData:        item.RevertData,
			CreatedAddress:    item.CreatedAddress,
			Logs:              logs,
		})
	}

	sum, err := crypto.HashJSON(payload)
	if err != nil {
		return "", err
	}

	return pkgtypes.Hash(sum), nil
}
