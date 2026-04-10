package rpc

import (
	"encoding/json"
	"net/http"
	"strconv"

	"silachain/internal/chain"
)

func MineHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	_ = blockchain
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusGone, "direct mining endpoint is disabled; use the consensus/execution pipeline")
	}
}

func LatestBlockHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := blockchain.LatestBlock()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
	}
}

func BlockByHeightHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("height")
		if v == "" {
			writeJSONError(w, http.StatusBadRequest, "missing height value")
			return
		}

		height, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid height value")
			return
		}

		b, ok := blockchain.GetBlockByHeight(height)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "block not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
	}
}
