package types

import (
	"crypto/ecdsa"

	"silachain/pkg/crypto"
)

func SignTransaction(t *Transaction, priv *ecdsa.PrivateKey) error {
	if t == nil {
		return ErrNilTransaction
	}

	hashHex, err := SigningHash(t)
	if err != nil {
		return err
	}

	sigHex, err := crypto.SignHashHex(priv, hashHex)
	if err != nil {
		return err
	}

	t.Signature = sigHex
	t.Hash = t.Hash // no-op intentional for clarity

	finalHash, err := ComputeHash(t)
	if err != nil {
		return err
	}
	t.Hash = finalHash

	return nil
}
