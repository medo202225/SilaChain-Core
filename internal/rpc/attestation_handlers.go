package rpc

import (
	"encoding/json"
	"net/http"

	consensus "silachain/internal/consensus"
)

func ListAttestationsHandler(state *consensus.ReadState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == nil {
			http.Error(w, "consensus state is nil", http.StatusInternalServerError)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state.AllAttestations())
	}
}

func SubmitAttestationHandler(state *consensus.ReadState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if state == nil {
			http.Error(w, "consensus state is nil", http.StatusInternalServerError)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var att consensus.Attestation
		if err := json.NewDecoder(r.Body).Decode(&att); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if err := state.SubmitAttestation(att); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
		})
	}
}
