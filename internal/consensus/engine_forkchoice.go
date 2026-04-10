package consensus

import "fmt"

type ForkchoicePayloadStatus struct {
	Status          string `json:"status"`
	LatestValidHash string `json:"latest_valid_hash"`
	ValidationError string `json:"validation_error"`
}

type ForkchoiceUpdatedResult struct {
	PayloadStatus ForkchoicePayloadStatus `json:"payload_status"`
	PayloadID     string                  `json:"payload_id"`
	RawResponse   map[string]any          `json:"raw_response"`
}

type ExecutionPayloadResult struct {
	BlockHash  string         `json:"block_hash"`
	ParentHash string         `json:"parent_hash"`
	RawPayload map[string]any `json:"raw_payload"`
}

type GetPayloadResult struct {
	ExecutionPayload ExecutionPayloadResult `json:"execution_payload"`
	RawResponse      map[string]any         `json:"raw_response"`
}

type NewPayloadResult struct {
	Status          string         `json:"status"`
	LatestValidHash string         `json:"latest_valid_hash"`
	ValidationError string         `json:"validation_error"`
	Accepted        bool           `json:"accepted"`
	RawResponse     map[string]any `json:"raw_response"`
}

func (c *EngineClient) ForkchoiceUpdatedV1(headBlockHash string, safeBlockHash string, finalizedBlockHash string, payloadAttributes map[string]any) (map[string]any, error) {
	params := []any{
		map[string]any{
			"headBlockHash":      headBlockHash,
			"safeBlockHash":      safeBlockHash,
			"finalizedBlockHash": finalizedBlockHash,
		},
		payloadAttributes,
	}
	return c.Call("engine_forkchoiceUpdatedV1", params)
}

func parseForkchoiceUpdatedResult(resp map[string]any) (*ForkchoiceUpdatedResult, error) {
	result, ok := resp["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing forkchoiceUpdated result")
	}

	payloadStatusMap, ok := result["payloadStatus"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing forkchoiceUpdated payloadStatus")
	}

	status, _ := payloadStatusMap["status"].(string)
	if status == "" {
		return nil, fmt.Errorf("missing forkchoiceUpdated payloadStatus.status")
	}

	latestValidHash, _ := payloadStatusMap["latestValidHash"].(string)
	validationError, _ := payloadStatusMap["validationError"].(string)
	payloadID, _ := result["payloadId"].(string)

	return &ForkchoiceUpdatedResult{
		PayloadStatus: ForkchoicePayloadStatus{
			Status:          status,
			LatestValidHash: latestValidHash,
			ValidationError: validationError,
		},
		PayloadID:   payloadID,
		RawResponse: resp,
	}, nil
}

func parseGetPayloadResult(resp map[string]any) (*GetPayloadResult, error) {
	result, ok := resp["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing getPayload result")
	}

	executionPayload, ok := result["executionPayload"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing executionPayload")
	}

	blockHash, _ := executionPayload["blockHash"].(string)
	if blockHash == "" {
		return nil, fmt.Errorf("missing executionPayload.blockHash")
	}

	parentHash, _ := executionPayload["parentHash"].(string)
	if parentHash == "" {
		return nil, fmt.Errorf("missing executionPayload.parentHash")
	}

	return &GetPayloadResult{
		ExecutionPayload: ExecutionPayloadResult{
			BlockHash:  blockHash,
			ParentHash: parentHash,
			RawPayload: executionPayload,
		},
		RawResponse: resp,
	}, nil
}

func (c *EngineClient) GetPayloadV1(payloadID string) (map[string]any, error) {
	return c.Call("engine_getPayloadV1", []any{payloadID})
}

func parseNewPayloadResult(resp map[string]any) (*NewPayloadResult, error) {
	result, ok := resp["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing newPayload result")
	}

	status, ok := result["status"].(string)
	if !ok || status == "" {
		return nil, fmt.Errorf("missing newPayload status")
	}

	latestValidHash, _ := result["latestValidHash"].(string)
	validationError, _ := result["validationError"].(string)

	out := &NewPayloadResult{
		Status:          status,
		LatestValidHash: latestValidHash,
		ValidationError: validationError,
		RawResponse:     resp,
	}

	switch status {
	case "VALID", "ACCEPTED", "SYNCING":
		out.Accepted = true
	case "INVALID", "INVALID_BLOCK_HASH":
		out.Accepted = false
	default:
		return nil, fmt.Errorf("unknown newPayload status %s", status)
	}

	return out, nil
}

func (c *EngineClient) NewPayloadV1(payload map[string]any) (map[string]any, error) {
	return c.Call("engine_newPayloadV1", []any{payload})
}
