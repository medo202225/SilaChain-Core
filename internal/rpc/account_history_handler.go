package rpc

import (
	"encoding/json"
	"net/http"

	"silachain/internal/chain"
	"silachain/pkg/types"
)

func AccountHistoryHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := types.Address(r.URL.Query().Get("address"))
		if address == "" {
			http.Error(w, "missing address", http.StatusBadRequest)
			return
		}

		rewards := make([]any, 0)
		for _, item := range blockchain.Rewards() {
			if item.Validator == address {
				rewards = append(rewards, item)
			}
		}

		delegatorRewards := make([]any, 0)
		for _, item := range blockchain.DelegatorRewards() {
			if item.Delegator == address {
				delegatorRewards = append(delegatorRewards, item)
			}
		}

		withdrawals := make([]any, 0)
		for _, item := range blockchain.Withdrawals() {
			if item.Address == address {
				withdrawals = append(withdrawals, item)
			}
		}

		unbondClaims := make([]any, 0)
		for _, item := range blockchain.UnbondClaims() {
			if item.Address == address {
				unbondClaims = append(unbondClaims, item)
			}
		}

		undelegations := make([]any, 0)
		for _, item := range blockchain.Undelegations() {
			if item.Delegator == address {
				undelegations = append(undelegations, item)
			}
		}

		out := map[string]any{
			"address":           address,
			"pending_rewards":   blockchain.PendingRewards(address),
			"pending_unbond":    blockchain.PendingUnbond(address),
			"validator_rewards": rewards,
			"delegator_rewards": delegatorRewards,
			"withdrawals":       withdrawals,
			"unbond_claims":     unbondClaims,
			"undelegations":     undelegations,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
