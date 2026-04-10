package mempool

import "silachain/internal/core/types"

func ValidateAndAdd(p *Pool, t *types.Transaction, currentNonce types.Nonce) error {
	if err := types.Validate(t); err != nil {
		return err
	}
	if p == nil {
		return ErrNilPool
	}
	if t == nil {
		return ErrNilTx
	}

	if t.Nonce < currentNonce {
		return types.ErrInvalidNonce
	}

	if p.maxFutureNonceGap > 0 && t.Nonce > currentNonce+p.maxFutureNonceGap {
		return ErrNonceTooFarInFuture
	}

	return p.Add(t)
}
