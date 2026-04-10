package sdk

// CANONICAL OWNERSHIP: public wallet SDK and user-facing wallet path.

import (
	"crypto/ecdsa"
	"errors"

	chaincrypto "silachain/pkg/crypto"
)

var ErrInvalidPrivateKey = errors.New("sdk: invalid private key")

type Wallet struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func NewWallet() (Wallet, error) {
	priv, pub, err := chaincrypto.GenerateKeyPair()
	if err != nil {
		return Wallet{}, err
	}

	address := chaincrypto.PublicKeyToAddress(pub)

	privateKeyHex := chaincrypto.PrivateKeyToHex(priv)

	publicKeyHex := chaincrypto.PublicKeyToHex(pub)

	return Wallet{
		Address:    string(address),
		PrivateKey: privateKeyHex,
		PublicKey:  publicKeyHex,
	}, nil
}

func ImportWallet(privateKeyHex string) (Wallet, error) {
	if privateKeyHex == "" {
		return Wallet{}, ErrInvalidPrivateKey
	}

	priv, err := chaincrypto.HexToPrivateKey(privateKeyHex)
	if err != nil {
		return Wallet{}, err
	}

	pub, ok := priv.Public().(*ecdsa.PublicKey)
	if !ok || pub == nil {
		return Wallet{}, ErrInvalidPrivateKey
	}

	address := chaincrypto.PublicKeyToAddress(pub)

	normalizedPrivateKeyHex := chaincrypto.PrivateKeyToHex(priv)

	publicKeyHex := chaincrypto.PublicKeyToHex(pub)

	return Wallet{
		Address:    string(address),
		PrivateKey: normalizedPrivateKeyHex,
		PublicKey:  publicKeyHex,
	}, nil
}
