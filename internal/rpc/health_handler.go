package rpc

import (
	"net/http"

	"silachain/internal/chain"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func NodeHealthHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var height uint64
		if blockchain != nil {
			if h, err := blockchain.Height(); err == nil {
				height = uint64(h)
			}
		}

		mempoolCount := 0
		proposerAvailable := false

		if blockchain != nil && blockchain.Mempool() != nil {
			mempoolCount = blockchain.Mempool().Count()
		}
		if blockchain != nil {
			_, proposerAvailable = blockchain.CurrentProposer()
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":                     "ok",
			"block_height":               height,
			"mempool_count":              mempoolCount,
			"current_proposer_available": proposerAvailable,
		})
	}
}
