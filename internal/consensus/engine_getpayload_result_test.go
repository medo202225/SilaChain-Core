package consensus

import "testing"

func TestParseGetPayloadResult(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{
			"executionPayload": map[string]any{
				"blockHash":  "0xabc",
				"parentHash": "0xdef",
			},
		},
	}

	parsed, err := parseGetPayloadResult(resp)
	if err != nil {
		t.Fatalf("parse getPayload result failed: %v", err)
	}

	if parsed.ExecutionPayload.BlockHash != "0xabc" {
		t.Fatalf("expected blockHash 0xabc, got %s", parsed.ExecutionPayload.BlockHash)
	}
	if parsed.ExecutionPayload.ParentHash != "0xdef" {
		t.Fatalf("expected parentHash 0xdef, got %s", parsed.ExecutionPayload.ParentHash)
	}
	if parsed.ExecutionPayload.RawPayload["blockHash"] != "0xabc" {
		t.Fatalf("expected raw blockHash 0xabc, got %v", parsed.ExecutionPayload.RawPayload["blockHash"])
	}
}

func TestParseGetPayloadResultRequiresExecutionPayload(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{},
	}

	_, err := parseGetPayloadResult(resp)
	if err == nil {
		t.Fatalf("expected missing executionPayload to fail")
	}
}

func TestParseGetPayloadResultRequiresBlockHash(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{
			"executionPayload": map[string]any{
				"parentHash": "0xdef",
			},
		},
	}

	_, err := parseGetPayloadResult(resp)
	if err == nil {
		t.Fatalf("expected missing blockHash to fail")
	}
}

func TestParseNewPayloadResult(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{
			"status":          "VALID",
			"latestValidHash": "0xabc",
			"validationError": "",
		},
	}

	parsed, err := parseNewPayloadResult(resp)
	if err != nil {
		t.Fatalf("parse newPayload result failed: %v", err)
	}

	if parsed.Status != "VALID" {
		t.Fatalf("expected VALID, got %s", parsed.Status)
	}
	if !parsed.Accepted {
		t.Fatalf("expected accepted newPayload result")
	}
	if parsed.LatestValidHash != "0xabc" {
		t.Fatalf("expected latestValidHash 0xabc, got %s", parsed.LatestValidHash)
	}
}

func TestParseNewPayloadResultRejectsInvalid(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{
			"status": "INVALID",
		},
	}

	parsed, err := parseNewPayloadResult(resp)
	if err != nil {
		t.Fatalf("expected INVALID to parse without parser error, got %v", err)
	}
	if parsed.Accepted {
		t.Fatalf("expected invalid payload to be rejected")
	}
}
