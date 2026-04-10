package rpc

import (
	"encoding/json"
	"net/http"

	consensus "silachain/internal/consensus"
)

func FinalizedVotesHandler(state *consensus.ReadState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == nil || state.FinalizationTracker == nil {
			http.Error(w, "consensus state is nil", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state.AllFinalized())
	}
}
