package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"strings"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	priv, err := secp.GeneratePrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("generate secp256k1 private key: %w", err)
	}

	ecdsaPriv := priv.ToECDSA()
	return ecdsaPriv, &ecdsaPriv.PublicKey, nil
}

func PrivateKeyToHex(priv *ecdsa.PrivateKey) string {
	if priv == nil {
		return ""
	}

	b := priv.D.Bytes()
	if len(b) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(b):], b)
		b = padded
	}
	return hex.EncodeToString(b)
}

func PublicKeyToBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil {
		return nil
	}
	return elliptic.Marshal(secp.S256(), pub.X, pub.Y)
}

func PublicKeyToHex(pub *ecdsa.PublicKey) string {
	raw := PublicKeyToBytes(pub)
	if len(raw) == 0 {
		return ""
	}
	return hex.EncodeToString(raw)
}

func HexToPrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	s := normalizeHex(hexKey)
	if s == "" {
		return nil, fmt.Errorf("empty private key hex")
	}

	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode private key hex: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("invalid private key length: %d", len(raw))
	}

	priv := secp.PrivKeyFromBytes(raw)
	return priv.ToECDSA(), nil
}

func HexToPublicKey(hexKey string) (*ecdsa.PublicKey, error) {
	s := normalizeHex(hexKey)
	if s == "" {
		return nil, fmt.Errorf("empty public key hex")
	}

	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode public key hex: %w", err)
	}

	pub, err := secp.ParsePubKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	ecdsaPub := pub.ToECDSA()
	return ecdsaPub, nil
}

func EncodeHexString(b []byte) string {
	return hex.EncodeToString(b)
}

func DecodeHexString(s string) ([]byte, error) {
	return hex.DecodeString(normalizeHex(s))
}

func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return s
}
