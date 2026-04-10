package consensus

import (
	"context"
	"fmt"
	"log"
)

type ProposalTransitionResult struct {
	PayloadID        string                 `json:"payload_id"`
	PayloadResponse  map[string]any         `json:"payload_response"`
	NewPayloadResp   map[string]any         `json:"new_payload_response"`
	PayloadStatus    string                 `json:"payload_status"`
	PayloadAccepted  bool                   `json:"payload_accepted"`
	ExecutionPayload ExecutionPayloadResult `json:"execution_payload"`
	GetPayloadResult *GetPayloadResult      `json:"get_payload_result"`
	NewPayloadResult *NewPayloadResult      `json:"new_payload_result"`
}

type EngineProposalExecutor struct {
	client *EngineClient
	state  *BeaconStateV1
}

func NewEngineProposalExecutor(client *EngineClient, state *BeaconStateV1) (*EngineProposalExecutor, error) {
	if client == nil {
		return nil, fmt.Errorf("engine client is nil")
	}
	if state == nil {
		return nil, fmt.Errorf("beacon state is nil")
	}
	return &EngineProposalExecutor{
		client: client,
		state:  state,
	}, nil
}

func (e *EngineProposalExecutor) ProposeBlock(ctx context.Context) error {
	_, err := e.ProposeBlockWithResult(ctx)
	return err
}

func (e *EngineProposalExecutor) ProposeBlockWithResult(ctx context.Context) (*ProposalTransitionResult, error) {
	_ = ctx
	if e == nil || e.client == nil {
		return nil, fmt.Errorf("engine proposal executor is nil")
	}
	if e.state == nil {
		return nil, fmt.Errorf("engine proposal executor state is nil")
	}
	if e.state.LatestPayloadID == "" {
		return nil, fmt.Errorf("missing payload id")
	}

	payloadResp, err := e.client.GetPayloadV1(e.state.LatestPayloadID)
	if err != nil {
		return nil, err
	}
	log.Printf("consensus executor: engine_getPayloadV1 ok: slot=%d payload_id=%s response=%v", e.state.Slot, e.state.LatestPayloadID, payloadResp)

	getPayloadResult, err := parseGetPayloadResult(payloadResp)
	if err != nil {
		return nil, err
	}

	newPayloadResp, err := e.client.NewPayloadV1(getPayloadResult.ExecutionPayload.RawPayload)
	if err != nil {
		return nil, err
	}
	log.Printf("consensus executor: engine_newPayloadV1 ok: slot=%d payload_id=%s response=%v", e.state.Slot, e.state.LatestPayloadID, newPayloadResp)

	newPayloadResult, err := parseNewPayloadResult(newPayloadResp)
	if err != nil {
		return nil, err
	}
	if !newPayloadResult.Accepted {
		return nil, fmt.Errorf("engine newPayload returned non-accepted status %s", newPayloadResult.Status)
	}

	e.state.HeadBlockRoot = getPayloadResult.ExecutionPayload.BlockHash
	e.state.SafeBlockRoot = getPayloadResult.ExecutionPayload.ParentHash

	return &ProposalTransitionResult{
		PayloadID:        e.state.LatestPayloadID,
		PayloadResponse:  payloadResp,
		NewPayloadResp:   newPayloadResp,
		PayloadStatus:    newPayloadResult.Status,
		PayloadAccepted:  newPayloadResult.Accepted,
		ExecutionPayload: getPayloadResult.ExecutionPayload,
		GetPayloadResult: getPayloadResult,
		NewPayloadResult: newPayloadResult,
	}, nil
}
