package consensus

import "testing"

func TestParseForkchoiceUpdatedResult(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{
			"payloadStatus": map[string]any{
				"status":          "VALID",
				"latestValidHash": "0xabc",
				"validationError": "",
			},
			"payloadId": "payload-123",
		},
	}

	parsed, err := parseForkchoiceUpdatedResult(resp)
	if err != nil {
		t.Fatalf("parse forkchoiceUpdated result failed: %v", err)
	}

	if parsed.PayloadStatus.Status != "VALID" {
		t.Fatalf("expected VALID, got %s", parsed.PayloadStatus.Status)
	}
	if parsed.PayloadStatus.LatestValidHash != "0xabc" {
		t.Fatalf("expected latest valid hash 0xabc, got %s", parsed.PayloadStatus.LatestValidHash)
	}
	if parsed.PayloadID != "payload-123" {
		t.Fatalf("expected payload-123, got %s", parsed.PayloadID)
	}
}

func TestParseForkchoiceUpdatedResultRequiresPayloadStatus(t *testing.T) {
	resp := map[string]any{
		"result": map[string]any{},
	}

	_, err := parseForkchoiceUpdatedResult(resp)
	if err == nil {
		t.Fatalf("expected missing payloadStatus to fail")
	}
}
