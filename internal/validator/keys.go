package validator

import (
	"crypto/ecdsa"

	chaincrypto "silachain/pkg/crypto"
	"silachain/pkg/types"
)

type GeneratedKey struct {
	PrivateKey *ecdsa.PrivateKey
	PrivateHex string
	PublicHex  string
	Address    types.Address
}

func GenerateValidatorKey() (*GeneratedKey, error) {
	priv, pub, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	privHex := chaincrypto.PrivateKeyToHex(priv)
	pubHex := chaincrypto.PublicKeyToHex(pub)
	addr := chaincrypto.PublicKeyToAddress(pub)

	return &GeneratedKey{
		PrivateKey: priv,
		PrivateHex: privHex,
		PublicHex:  pubHex,
		Address:    types.Address(addr),
	}, nil
}

func AddressMust(v string) types.Address {
	return types.Address(v)
}
