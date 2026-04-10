package wallet

// CANONICAL OWNERSHIP: internal wallet helper only.
// Public and user-facing wallet ownership lives in pkg/sdk.

import "silachain/pkg/crypto"

func NewRandomWallet() (*Wallet, error) {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	privHex := crypto.PrivateKeyToHex(priv)
	pubHex := crypto.PublicKeyToHex(pub)
	address := crypto.PublicKeyToAddress(pub)

	return &Wallet{
		Address:       string(address),
		PublicKeyHex:  pubHex,
		PrivateKeyHex: privHex,
	}, nil
}
