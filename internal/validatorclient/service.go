package validatorclient

// CANONICAL OWNERSHIP: validator client package for keystores, duties, slashing protection, signing, and validator service runtime.
// Planned final architectural name is validatorclient after dependency cleanup.

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"silachain/internal/config"
	"silachain/internal/validatorclient/slashing"
)

type ValidatorService struct {
	cfg    *config.ValidatorClientConfig
	loaded *LoadedVotingKeystore
	store  slashing.Store
	server *http.Server
}

type StatusResponse struct {
	Service                  string `json:"service"`
	ListenAddress            string `json:"listen_address"`
	VotingPublicKey          string `json:"voting_public_key"`
	SlashingProtectionDBPath string `json:"slashing_protection_db_path"`
}

type HealthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

type SignProposalRequest struct {
	Slot        uint64 `json:"slot"`
	SigningRoot string `json:"signing_root"`
}

type SignAttestationRequest struct {
	SourceEpoch uint64 `json:"source_epoch"`
	TargetEpoch uint64 `json:"target_epoch"`
	SigningRoot string `json:"signing_root"`
}

type SignResponse struct {
	PubKey       string `json:"pubkey"`
	SignatureHex string `json:"signature_hex"`
}

func NewValidatorService(cfg *config.ValidatorClientConfig) (*ValidatorService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("validator config is nil")
	}

	loaded, err := LoadVotingKeystore(cfg.VotingKeystorePath, cfg.VotingSecretPath)
	if err != nil {
		return nil, err
	}
	if cfg.VotingPublicKey != "" && loaded.PublicHex != cfg.VotingPublicKey {
		return nil, fmt.Errorf("voting public key mismatch between config and keystore")
	}

	store := slashing.NewSQLiteStore(cfg.SlashingProtectionDBPath)
	if err := store.Init(context.Background()); err != nil {
		return nil, err
	}

	svc := &ValidatorService{
		cfg:    cfg,
		loaded: loaded,
		store:  store,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/validator/health", svc.handleHealth)
	mux.HandleFunc("/validator/status", svc.handleStatus)
	mux.HandleFunc("/validator/duties/sign/proposal", svc.handleSignProposal)
	mux.HandleFunc("/validator/duties/sign/attestation", svc.handleSignAttestation)

	svc.server = &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: mux,
	}

	return svc, nil
}

func (s *ValidatorService) Close() error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *ValidatorService) Start() error {
	if s == nil || s.server == nil {
		return fmt.Errorf("validator service is nil")
	}
	return s.server.ListenAndServe()
}

func (s *ValidatorService) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	out := HealthResponse{
		Service: "sila-validator",
		Status:  "ok",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *ValidatorService) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	out := StatusResponse{
		Service:                  "sila-validator",
		ListenAddress:            s.cfg.ListenAddress,
		VotingPublicKey:          s.loaded.PublicHex,
		SlashingProtectionDBPath: s.cfg.SlashingProtectionDBPath,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *ValidatorService) handleSignProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SignProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	signingRoot, err := hex.DecodeString(req.SigningRoot)
	if err != nil || len(signingRoot) == 0 {
		http.Error(w, "invalid signing_root", http.StatusBadRequest)
		return
	}

	sig, err := ProtectedSignBlock(context.Background(), s.loaded, s.store, req.Slot, signingRoot)
	if err != nil {
		http.Error(w, "proposal signing failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	out := SignResponse{
		PubKey:       s.loaded.PublicHex,
		SignatureHex: sig.SignatureHex,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *ValidatorService) handleSignAttestation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SignAttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	signingRoot, err := hex.DecodeString(req.SigningRoot)
	if err != nil || len(signingRoot) == 0 {
		http.Error(w, "invalid signing_root", http.StatusBadRequest)
		return
	}

	sig, err := ProtectedSignAttestation(context.Background(), s.loaded, s.store, req.SourceEpoch, req.TargetEpoch, signingRoot)
	if err != nil {
		http.Error(w, "attestation signing failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	out := SignResponse{
		PubKey:       s.loaded.PublicHex,
		SignatureHex: sig.SignatureHex,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
