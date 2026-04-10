package wallet

import (
	coretypes "silachain/internal/core/types"
	"silachain/pkg/crypto"
)

func (w *Wallet) SignTransaction(t *coretypes.Transaction) error {
	priv, err := crypto.HexToPrivateKey(w.PrivateKeyHex)
	if err != nil {
		return err
	}

	t.PublicKey = w.PublicKeyHex
	return coretypes.SignTransaction(t, priv)
}
