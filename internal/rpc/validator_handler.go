package rpc

import (
	"encoding/json"
	"net/http"

	"silachain/internal/chain"
)

func ValidatorsHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockchain.Validators())
	}
}

func ActiveValidatorsHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockchain.ActiveValidators())
	}
}

func WeightedValidatorsHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockchain.WeightedValidators())
	}
}
