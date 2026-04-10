package rpc

import (
	"encoding/json"
	"net/http"

	"silachain/internal/chain"
	"silachain/pkg/types"
)

func RegisterAccountHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		acc, err := blockchain.RegisterAccount(types.Address(req.Address), req.PublicKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(acc)
	}
}

func AccountBalanceHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := r.URL.Query().Get("address")
		acc, err := blockchain.GetAccount(types.Address(address))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"address": acc.Address,
			"balance": acc.Balance,
			"nonce":   acc.Nonce,
		})
	}
}

func AccountsListHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockchain.AllAccounts())
	}
}

func FaucetHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req FaucetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := blockchain.Faucet(types.Address(req.Address), types.Amount(req.Amount)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		acc, err := blockchain.GetAccount(types.Address(req.Address))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(acc)
	}
}
