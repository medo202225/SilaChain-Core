package accounts

import "silachain/pkg/types"

func NextNonce(acc *Account) types.Nonce {
	if acc == nil {
		return 0
	}
	return acc.Nonce
}
