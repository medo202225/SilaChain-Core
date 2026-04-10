package consensus

import "testing"

func TestParseExchangeCapabilitiesResult(t *testing.T) {
	resp := map[string]any{
		"result": []any{
			"engine_exchangeCapabilities",
			"engine_identity",
			"engine_forkchoiceUpdatedV1",
		},
	}

	parsed, err := parseExchangeCapabilitiesResult(resp)
	if err != nil {
		t.Fatalf("parse exchangeCapabilities result failed: %v", err)
	}

	if len(parsed.Capabilities) != 3 {
		t.Fatalf("expected 3 capabilities, got %d", len(parsed.Capabilities))
	}
	if parsed.Capabilities[0] != "engine_exchangeCapabilities" {
		t.Fatalf("unexpected first capability %s", parsed.Capabilities[0])
	}
}

func TestParseIdentityResult(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{
			"name":    "sila-el",
			"version": "1.0.0",
			"chain":   "sila-mainnet",
		},
	}

	parsed, err := parseIdentityResult(resp)
	if err != nil {
		t.Fatalf("parse identity result failed: %v", err)
	}

	if parsed.Name != "sila-el" {
		t.Fatalf("expected name sila-el, got %s", parsed.Name)
	}
	if parsed.Version != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", parsed.Version)
	}
	if parsed.Chain != "sila-mainnet" {
		t.Fatalf("expected chain sila-mainnet, got %s", parsed.Chain)
	}
}
