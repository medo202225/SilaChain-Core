package runtime

import (
	"errors"

	"silachain/internal/consensus/engineapi"
)

var (
	ErrNilRuntime = errors.New("runtime: nil runtime")
)

func (r *Runtime) PruneCanonicalPayload(payloadID string) error {
	if r == nil {
		return ErrNilRuntime
	}
	if r.api == nil {
		return ErrNilHTTPServer
	}
	if r.pool == nil {
		return ErrNilRuntime
	}

	meta, err := r.api.GetPayloadMetadata(payloadID)
	if err != nil {
		return err
	}
	if !meta.Canonical {
		return nil
	}

	return r.pool.RemoveIncluded(engineapi.CanonicalTransactionsForPayload(meta))
}
