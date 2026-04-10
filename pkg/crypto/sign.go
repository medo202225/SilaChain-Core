package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

func Sign(hash []byte, priv *ecdsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, fmt.Errorf("nil private key")
	}
	if len(hash) != 32 {
		return nil, fmt.Errorf("invalid hash length: got %d want 32", len(hash))
	}

	raw := priv.D.Bytes()
	if len(raw) > 32 {
		return nil, fmt.Errorf("invalid private key length: %d", len(raw))
	}
	if len(raw) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(raw):], raw)
		raw = padded
	}

	secpPriv := secp.PrivKeyFromBytes(raw)

	compact := secpecdsa.SignCompact(secpPriv, hash, false)
	if len(compact) != 65 {
		return nil, fmt.Errorf("invalid compact signature length: %d", len(compact))
	}

	header := compact[0]
	if header < 27 {
		return nil, fmt.Errorf("invalid compact signature header: %d", header)
	}

	recID := header - 27
	if recID >= 4 {
		recID -= 4
	}
	if recID > 3 {
		return nil, fmt.Errorf("invalid recovery id: %d", recID)
	}

	sig := make([]byte, 65)
	copy(sig[:64], compact[1:])
	sig[64] = recID

	return sig, nil
}

func SignToHex(hash []byte, priv *ecdsa.PrivateKey) (string, error) {
	sig, err := Sign(hash, priv)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}

func SignHashHex(priv *ecdsa.PrivateKey, hash string) (string, error) {
	rawHash, err := DecodeHexString(hash)
	if err != nil {
		return "", fmt.Errorf("decode hash hex: %w", err)
	}
	return SignToHex(rawHash, priv)
}
