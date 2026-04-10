package consensus

// CANONICAL OWNERSHIP: root consensus package is limited to beacon state, scheduling, transition coordination, and validator coordination.
// Engine, engine API, forkchoice, runtime, txpool, executionstate, and p2p ownership live in their dedicated subpackages.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	validatorclient "silachain/internal/validatorclient"
)

type ValidatorClient struct {
	endpoint        string
	httpClient      *http.Client
	votingPublicKey string
}

func NewValidatorClient(endpoint string) (*ValidatorClient, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("validator endpoint is empty")
	}
	return &ValidatorClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *ValidatorClient) Health(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/validator/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validator http status %d: %v", resp.StatusCode, out)
	}
	return out, nil
}

func (c *ValidatorClient) Status(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/validator/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validator http status %d: %v", resp.StatusCode, out)
	}

	if pk, ok := out["voting_public_key"].(string); ok {
		c.votingPublicKey = pk
	}

	return out, nil
}

func (c *ValidatorClient) VotingPublicKey() string {
	if c == nil {
		return ""
	}
	return c.votingPublicKey
}

func (c *ValidatorClient) SignProposal(ctx context.Context, duty validatorclient.ProposalDuty) (*validatorclient.SignatureResult, error) {
	body := map[string]any{
		"slot":         duty.Slot,
		"signing_root": fmt.Sprintf("%x", duty.SigningRoot),
	}
	var out struct {
		SignatureHex string `json:"signature_hex"`
	}
	if err := c.postJSON(ctx, "/validator/duties/sign/proposal", body, &out); err != nil {
		return nil, err
	}
	return &validatorclient.SignatureResult{SignatureHex: out.SignatureHex}, nil
}

func (c *ValidatorClient) SignAttestation(ctx context.Context, duty validatorclient.AttestationDuty) (*validatorclient.SignatureResult, error) {
	body := map[string]any{
		"source_epoch": duty.SourceEpoch,
		"target_epoch": duty.TargetEpoch,
		"signing_root": fmt.Sprintf("%x", duty.SigningRoot),
	}
	var out struct {
		SignatureHex string `json:"signature_hex"`
	}
	if err := c.postJSON(ctx, "/validator/duties/sign/attestation", body, &out); err != nil {
		return nil, err
	}
	return &validatorclient.SignatureResult{SignatureHex: out.SignatureHex}, nil
}

func (c *ValidatorClient) postJSON(ctx context.Context, path string, reqBody any, out any) error {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		msg := strings.TrimSpace(buf.String())
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("validator http status %d: %s", resp.StatusCode, msg)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
