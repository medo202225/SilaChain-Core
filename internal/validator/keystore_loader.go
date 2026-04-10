package validator

import (
	chaincrypto "silachain/pkg/crypto"
)

func LoadEncryptedKeystoreWithSecret(keystorePath string, secretPath string) (*LoadedKey, error) {
	password, err := LoadSecretFile(secretPath)
	if err != nil {
		return nil, err
	}

	file, err := LoadEncryptedKeyFile(keystorePath, password)
	if err != nil {
		return nil, err
	}

	priv, err := chaincrypto.HexToPrivateKey(file.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &LoadedKey{
		File:       *file,
		PrivateKey: priv,
	}, nil
}
