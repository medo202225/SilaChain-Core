package rpc

import (
	"encoding/json"
	"net/http"

	"silachain/internal/config"
)

func ProtocolInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.LoadProtocolConfig("config/networks/mainnet/public/protocol.json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		out := map[string]any{
			"block_reward":             cfg.BlockReward,
			"unbonding_delay":          cfg.UnbondingDelay,
			"min_validator_stake":      cfg.MinValidatorStake,
			"validator_commission_bps": cfg.ValidatorCommissionBps,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
