package consensus

// CANONICAL OWNERSHIP: root consensus package is limited to beacon state, scheduling, transition coordination, and validator coordination.
// Engine, engine API, forkchoice, runtime, txpool, executionstate, and p2p ownership live in their dedicated subpackages.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type EngineCapabilitiesResult struct {
	Capabilities []string       `json:"capabilities"`
	RawResponse  map[string]any `json:"raw_response"`
}

type EngineIdentityResult struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Chain       string         `json:"chain"`
	RawResponse map[string]any `json:"raw_response"`
}

type EngineClient struct {
	endpoint   string
	jwtSecret  string
	httpClient *http.Client
}

func NewEngineClient(endpoint string, jwtSecretPath string) (*EngineClient, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("engine endpoint is empty")
	}
	if strings.TrimSpace(jwtSecretPath) == "" {
		return nil, fmt.Errorf("engine jwt secret path is empty")
	}

	raw, err := os.ReadFile(jwtSecretPath)
	if err != nil {
		return nil, fmt.Errorf("read engine jwt secret failed: %w", err)
	}

	secret := strings.TrimSpace(string(raw))
	if len(secret) != 64 {
		return nil, fmt.Errorf("engine jwt secret must be 64 hex chars, got %d", len(secret))
	}
	if _, err := hex.DecodeString(secret); err != nil {
		return nil, fmt.Errorf("decode engine jwt secret failed: %w", err)
	}

	return &EngineClient{
		endpoint:  endpoint,
		jwtSecret: secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *EngineClient) makeJWT() (string, error) {
	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]any{
		"iat": time.Now().Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header failed: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal jwt payload failed: %w", err)
	}

	headerSeg := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadSeg := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerSeg + "." + payloadSeg

	secretBytes, err := hex.DecodeString(c.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("decode jwt secret failed: %w", err)
	}

	mac := hmac.New(sha256.New, secretBytes)
	_, _ = mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)
	sigSeg := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigSeg, nil
}

func (c *EngineClient) Call(method string, params any) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("engine client is nil")
	}

	token, err := c.makeJWT()
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal engine request failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create engine request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode engine response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("engine http status %d: %v", resp.StatusCode, out)
	}

	return out, nil
}

func parseExchangeCapabilitiesResult(resp map[string]any) (*EngineCapabilitiesResult, error) {
	result, ok := resp["result"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing exchangeCapabilities result")
	}

	caps := make([]string, 0, len(result))
	for _, item := range result {
		s, ok := item.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("invalid capability entry")
		}
		caps = append(caps, s)
	}

	return &EngineCapabilitiesResult{
		Capabilities: caps,
		RawResponse:  resp,
	}, nil
}

func parseIdentityResult(resp map[string]any) (*EngineIdentityResult, error) {
	result, ok := resp["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing identity result")
	}

	name, _ := result["name"].(string)
	version, _ := result["version"].(string)
	chain, _ := result["chain"].(string)

	return &EngineIdentityResult{
		Name:        name,
		Version:     version,
		Chain:       chain,
		RawResponse: resp,
	}, nil
}

func (c *EngineClient) ExchangeCapabilities() (map[string]any, error) {
	return c.Call("engine_exchangeCapabilities", []any{[]string{
		"engine_exchangeCapabilities",
		"engine_identity",
		"engine_forkchoiceUpdatedV1",
		"engine_getPayloadV1",
		"engine_newPayloadV1",
	}})
}

func (c *EngineClient) ExchangeCapabilitiesParsed() (*EngineCapabilitiesResult, error) {
	resp, err := c.ExchangeCapabilities()
	if err != nil {
		return nil, err
	}
	return parseExchangeCapabilitiesResult(resp)
}

func (c *EngineClient) Identity() (map[string]any, error) {
	return c.Call("engine_identity", []any{})
}

func (c *EngineClient) IdentityParsed() (*EngineIdentityResult, error) {
	resp, err := c.Identity()
	if err != nil {
		return nil, err
	}
	return parseIdentityResult(resp)
}
