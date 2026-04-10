package rpc

import (
	"encoding/json"
	"errors"
	"net/http"

	"silachain/internal/chain"
	coretypes "silachain/internal/core/types"
)

const txBodyLimitBytes = int64(64 * 1024)

func decodeStrictJSON(r *http.Request, dst any) error {
	if r == nil || r.Body == nil {
		return errors.New("empty request body")
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	if dec.More() {
		return errors.New("unexpected extra json content")
	}

	return nil
}

func SendTxHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var t coretypes.Transaction
		if err := decodeStrictJSON(r, &t); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := blockchain.SubmitTransaction(&t); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"accepted":      true,
			"pending":       true,
			"tx_hash":       t.Hash,
			"effective_fee": t.EffectiveFee(),
			"gas_price":     t.GasPrice,
			"gas_limit":     t.GasLimit,
			"mempool_count": blockchain.Mempool().Count(),
		})
	}
}

func TxByHashHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		if hash == "" {
			writeJSONError(w, http.StatusBadRequest, "missing hash value")
			return
		}

		t, height, ok := blockchain.GetTransactionByHash(hash)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "transaction not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"height":        height,
			"transaction":   t,
			"effective_fee": t.EffectiveFee(),
			"gas_price":     t.GasPrice,
			"gas_limit":     t.GasLimit,
			"fee_mode": map[string]any{
				"uses_legacy_fee":  t.Fee > 0,
				"uses_gas_pricing": t.Fee == 0 && t.GasPrice > 0,
			},
		})
	}
}

func TxReceiptHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		if hash == "" {
			writeJSONError(w, http.StatusBadRequest, "missing hash value")
			return
		}

		receipt, ok := blockchain.GetReceiptByHash(hash)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "receipt not found")
			return
		}

		writeJSON(w, http.StatusOK, receipt)
	}
}
