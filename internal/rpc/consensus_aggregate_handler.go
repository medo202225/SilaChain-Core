package rpc

import (
	"encoding/json"
	"net/http"

	consensus "silachain/internal/consensus"
)

type aggregateResponse struct {
	Slot            uint64   `json:"slot"`
	Epoch           uint64   `json:"epoch"`
	BlockHash       string   `json:"block_hash"`
	VoteCount       int      `json:"vote_count"`
	TotalValidators int      `json:"total_validators"`
	QuorumReached   bool     `json:"quorum_reached"`
	Validators      []string `json:"validators"`
}

func AttestationAggregatesHandler(state *consensus.ReadState, totalValidators int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == nil {
			http.Error(w, "consensus state is nil", http.StatusInternalServerError)
			return
		}

		aggregates := consensus.AggregateAttestations(state.AllAttestations())
		out := make([]aggregateResponse, 0, len(aggregates))

		for _, agg := range aggregates {
			q := consensus.CheckQuorum(agg.VoteCount, totalValidators)

			validators := make([]string, 0, len(agg.Validators))
			for _, v := range agg.Validators {
				validators = append(validators, string(v))
			}

			out = append(out, aggregateResponse{
				Slot:            agg.Slot,
				Epoch:           agg.Epoch,
				BlockHash:       agg.BlockHash,
				VoteCount:       agg.VoteCount,
				TotalValidators: q.TotalValidators,
				QuorumReached:   q.HasQuorum,
				Validators:      validators,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
