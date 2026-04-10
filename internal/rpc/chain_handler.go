package rpc

import (
	"encoding/json"
	"net/http"

	"silachain/internal/chain"
	"silachain/internal/protocol"
)

func ChainInfoHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		height, err := blockchain.Height()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		latest, err := blockchain.LatestBlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		metrics := blockchain.MonetaryMetrics()

		_ = json.NewEncoder(w).Encode(map[string]any{
			"protocol_name":    protocol.ProtocolName,
			"protocol_version": protocol.ProtocolVersion,
			"network_name":     protocol.NetworkName,
			"chain_id":         protocol.DefaultMainnetParams().ChainID,
			"symbol":           protocol.NativeSymbol,
			"height":           height,
			"latest_hash":      latest.Header.Hash,
			"monetary_metrics": metrics,
			"monetary_policy": map[string]any{
				"burn_enabled":             false,
				"treasury_enabled":         false,
				"monetary_policy_frozen":   true,
				"block_reward":             10,
				"unbonding_delay":          3,
				"min_validator_stake":      1,
				"validator_commission_bps": 1000,
			},
		})
	}
}
