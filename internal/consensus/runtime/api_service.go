package runtime

import (
	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engineapi"
)

type APIService struct {
	inner   *engineapi.BuilderService
	runtime *Runtime
}

func NewAPIService(rt *Runtime) *APIService {
	if rt == nil {
		return nil
	}
	return &APIService{
		inner:   rt.api,
		runtime: rt,
	}
}

func (s *APIService) NewPayload(payload engineapi.PayloadEnvelope) (engineapi.PayloadStatus, error) {
	return s.inner.NewPayload(payload)
}

func (s *APIService) ForkchoiceUpdated(state engineapi.ForkchoiceState) (engineapi.ForkchoiceUpdatedResult, error) {
	result, err := s.inner.ForkchoiceUpdated(state)
	if err != nil {
		return result, err
	}

	s.autoPruneCanonicalHead(result.CanonicalHead.Hash)
	return result, nil
}

func (s *APIService) ForkchoiceUpdatedWithAttributes(
	state engineapi.ForkchoiceState,
	attrs *blockassembly.PayloadAttributes,
) (engineapi.ForkchoiceUpdatedWithAttributesResult, error) {
	result, err := s.inner.ForkchoiceUpdatedWithAttributes(state, attrs)
	if err != nil {
		return result, err
	}

	s.autoPruneCanonicalHead(result.CanonicalHead.Hash)
	return result, nil
}

func (s *APIService) GetPayload(payloadID string) (engineapi.GetPayloadResult, error) {
	return s.inner.GetPayload(payloadID)
}

func (s *APIService) GetPayloadMetadata(payloadID string) (engineapi.PayloadMetadata, error) {
	return s.inner.GetPayloadMetadata(payloadID)
}

func (s *APIService) autoPruneCanonicalHead(blockHash string) {
	if s == nil || s.runtime == nil || s.runtime.pool == nil || s.inner == nil {
		return
	}
	if blockHash == "" {
		return
	}

	meta, ok := s.inner.GetPayloadMetadataByBlockHash(blockHash)
	if !ok {
		return
	}
	if !meta.Canonical {
		return
	}

	_ = s.runtime.pool.RemoveIncluded(meta.Transactions)
}
