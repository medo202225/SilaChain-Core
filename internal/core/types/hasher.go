package types

import (
	"silachain/pkg/crypto"
)

func SigningHash(t *Transaction) (string, error) {
	if t == nil {
		return "", ErrNilTransaction
	}
	return crypto.HashJSON(t.SigningPayload())
}

func ComputeHash(t *Transaction) (Hash, error) {
	h, err := SigningHash(t)
	if err != nil {
		return "", err
	}
	return Hash(h), nil
}
