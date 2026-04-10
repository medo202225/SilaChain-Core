package runtime

import (
	"errors"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engineapi"
)

var (
	ErrNilAPIService = errors.New("runtime: nil api service")
)

type ProduceBlockRequest struct {
	Timestamp         uint64 `json:"timestamp"`
	FeeRecipient      string `json:"feeRecipient"`
	Random            string `json:"random"`
	SuggestedGasLimit uint64 `json:"suggestedGasLimit"`
}

type ProduceBlockResult struct {
	PayloadID     string                  `json:"payloadID"`
	PayloadStatus engineapi.PayloadStatus `json:"payloadStatus"`
	CanonicalHead any                     `json:"canonicalHead"`
	SafeHead      any                     `json:"safeHead"`
	FinalizedHead any                     `json:"finalizedHead"`
	TxPoolPending int                     `json:"txPoolPending"`
}

func (r *Runtime) ProduceBlock(req ProduceBlockRequest) (ProduceBlockResult, error) {
	if r == nil || r.apiService == nil {
		return ProduceBlockResult{}, ErrNilAPIService
	}

	attrs := blockassembly.PayloadAttributes{
		Timestamp:         req.Timestamp,
		FeeRecipient:      req.FeeRecipient,
		Random:            req.Random,
		SuggestedGasLimit: req.SuggestedGasLimit,
	}

	head := r.state.Head()

	buildResult, err := r.api.ForkchoiceUpdatedWithAttributes(
		engineapi.ForkchoiceState{
			HeadBlockHash:      head.Hash,
			SafeBlockHash:      head.Hash,
			FinalizedBlockHash: head.Hash,
		},
		&attrs,
	)
	if err != nil {
		return ProduceBlockResult{}, err
	}

	payload, err := r.api.GetPayload(buildResult.PayloadID)
	if err != nil {
		return ProduceBlockResult{}, err
	}

	_, err = r.engine.ProduceBlock(attrs)
	if err != nil {
		return ProduceBlockResult{}, err
	}

	payloadStatus, err := r.apiService.NewPayload(engineapi.PayloadEnvelope{
		BlockNumber: payload.BlockNumber,
		BlockHash:   payload.BlockHash,
		ParentHash:  payload.ParentHash,
		StateRoot:   payload.StateRoot,
	})
	if err != nil {
		return ProduceBlockResult{}, err
	}

	finalResult, err := r.apiService.ForkchoiceUpdated(engineapi.ForkchoiceState{
		HeadBlockHash:      payload.BlockHash,
		SafeBlockHash:      payload.BlockHash,
		FinalizedBlockHash: head.Hash,
	})
	if err != nil {
		return ProduceBlockResult{}, err
	}

	return ProduceBlockResult{
		PayloadID:     buildResult.PayloadID,
		PayloadStatus: payloadStatus,
		CanonicalHead: finalResult.CanonicalHead,
		SafeHead:      finalResult.SafeHead,
		FinalizedHead: finalResult.FinalizedHead,
		TxPoolPending: r.pool.PendingCount(),
	}, nil
}
