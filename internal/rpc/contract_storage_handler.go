package rpc

import (
	"net/http"

	"silachain/internal/chain"
	"silachain/pkg/types"
)

type CreateContractRequest struct {
	Address        string `json:"address"`
	CodeHash       string `json:"code_hash"`
	InitialBalance uint64 `json:"initial_balance"`
}

type ContractStorageWriteRequest struct {
	Address string `json:"address"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func CreateContractHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req CreateContractRequest
		if err := decodeStrictJSON(r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		acc, err := blockchain.CreateContractAccount(
			types.Address(req.Address),
			types.Hash(req.CodeHash),
			types.Amount(req.InitialBalance),
		)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"created": true,
			"account": acc,
		})
	}
}

func SetContractStorageHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req ContractStorageWriteRequest
		if err := decodeStrictJSON(r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := blockchain.SetContractStorage(types.Address(req.Address), req.Key, req.Value); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		root, err := blockchain.GetContractStorageRoot(types.Address(req.Address))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"updated":      true,
			"address":      req.Address,
			"key":          req.Key,
			"value":        req.Value,
			"storage_root": root,
		})
	}
}

func GetContractStorageHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		address := r.URL.Query().Get("address")
		key := r.URL.Query().Get("key")
		if address == "" || key == "" {
			writeJSONError(w, http.StatusBadRequest, "missing address or key")
			return
		}

		value, ok := blockchain.GetContractStorage(types.Address(address), key)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "storage value not found")
			return
		}

		root, err := blockchain.GetContractStorageRoot(types.Address(address))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"address":      address,
			"key":          key,
			"value":        value,
			"storage_root": root,
		})
	}
}
