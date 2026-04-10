package rpc

import (
	"net/http"

	"silachain/internal/chain"
)

func MempoolStatusHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if blockchain == nil || blockchain.Mempool() == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "mempool is unavailable")
			return
		}

		mp := blockchain.Mempool()

		writeJSON(w, http.StatusOK, map[string]any{
			"count":                mp.Count(),
			"max_pool_size":        mp.MaxPoolSize(),
			"max_per_sender":       mp.MaxPerSender(),
			"max_future_nonce_gap": mp.MaxFutureNonceGap(),
			"replacement_bump_bps": mp.ReplacementBumpBps(),
		})
	}
}
