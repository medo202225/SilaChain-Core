package rpc

import (
	"net/http"

	"silachain/internal/chain"
	coretypes "silachain/internal/core/types"
	pkgtypes "silachain/pkg/types"
)

type ContractCallRequest struct {
	From          string `json:"from"`
	Address       string `json:"address"`
	Value         uint64 `json:"value"`
	Fee           uint64 `json:"fee"`
	GasPrice      uint64 `json:"gas_price"`
	GasLimit      uint64 `json:"gas_limit"`
	Nonce         uint64 `json:"nonce"`
	ChainID       uint64 `json:"chain_id"`
	Timestamp     int64  `json:"timestamp"`
	VMVersion     uint16 `json:"vm_version"`
	ContractInput string `json:"contract_input"`
	PublicKey     string `json:"public_key"`
	Signature     string `json:"signature"`
	Hash          string `json:"hash"`
}

func ContractCallHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req ContractCallRequest
		if err := decodeStrictJSON(r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		t := coretypes.Transaction{
			Type:          coretypes.TypeContractCall,
			From:          pkgtypes.Address(req.From),
			To:            pkgtypes.Address(req.Address),
			Value:         pkgtypes.Amount(req.Value),
			Fee:           pkgtypes.Amount(req.Fee),
			GasPrice:      pkgtypes.Amount(req.GasPrice),
			GasLimit:      pkgtypes.Gas(req.GasLimit),
			Nonce:         pkgtypes.Nonce(req.Nonce),
			ChainID:       pkgtypes.ChainID(req.ChainID),
			Timestamp:     pkgtypes.Timestamp(req.Timestamp),
			VMVersion:     req.VMVersion,
			ContractInput: req.ContractInput,
			PublicKey:     req.PublicKey,
			Signature:     req.Signature,
			Hash:          pkgtypes.Hash(req.Hash),
		}

		if err := blockchain.SubmitTransaction(&t); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"accepted":      true,
			"pending":       true,
			"tx_hash":       t.Hash,
			"type":          t.Type,
			"vm_version":    t.VMVersion,
			"effective_fee": t.EffectiveFee(),
			"gas_price":     t.GasPrice,
			"gas_limit":     t.GasLimit,
			"mempool_count": blockchain.Mempool().Count(),
			"note":          "use /tx/receipt after mining to get return_data, revert_data, success",
		})
	}
}
