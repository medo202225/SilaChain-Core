package types

import (
	"strings"

	"silachain/pkg/crypto"
)

func VerifySignature(t *Transaction) error {
	if t == nil {
		return ErrNilTransaction
	}
	if strings.TrimSpace(t.PublicKey) == "" {
		return ErrMissingPublicKey
	}
	if strings.TrimSpace(t.Signature) == "" {
		return ErrMissingSignature
	}

	pub, err := crypto.HexToPublicKey(t.PublicKey)
	if err != nil {
		return ErrMalformedPublicKey
	}

	derived := crypto.PublicKeyToAddress(pub)
	if strings.TrimSpace(string(derived)) != strings.TrimSpace(string(t.From)) {
		return ErrPublicKeyMismatch
	}

	hashHex, err := SigningHash(t)
	if err != nil {
		return err
	}

	ok, err := crypto.VerifyHashHex(pub, hashHex, t.Signature)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSignatureVerification
	}

	return nil
}
