package rpc

import (
	"encoding/json"
	"net/http"

	"silachain/internal/chain"
)

func ConsensusProposerHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proposer, ok := blockchain.CurrentProposer()

		out := map[string]any{
			"has_proposer": ok,
			"proposer":     proposer,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func ConsensusRotationHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nextIndex, epoch := blockchain.RotationState()
		weighted := blockchain.WeightedValidators()

		out := map[string]any{
			"next_validator_index": nextIndex,
			"epoch":                epoch,
			"weighted_count":       len(weighted),
			"weighted_validators":  weighted,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
