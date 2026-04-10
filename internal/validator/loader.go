package validator

// CANONICAL OWNERSHIP: validator domain package for member/set/registry and legacy-loaded signing key contracts used by chain, storage, and legacy consensus paths.

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"os"
	"strings"

	chaincrypto "silachain/pkg/crypto"
	"silachain/pkg/types"
)

type LoadedKey struct {
	File       KeyFile
	PrivateKey *ecdsa.PrivateKey
}

func LoadKeyFile(path string) (*LoadedKey, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrEmptyKeyPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyFileNotFound
		}
		return nil, err
	}

	var file KeyFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, ErrInvalidKeyFile
	}

	if err := file.Validate(); err != nil {
		return nil, err
	}

	priv, err := chaincrypto.HexToPrivateKey(file.PrivateKey)
	if err != nil {
		return nil, err
	}

	pubHex := chaincrypto.PublicKeyToHex(&priv.PublicKey)
	if pubHex != file.PublicKey {
		return nil, ErrInvalidKeyFile
	}

	addr := chaincrypto.PublicKeyToAddress(&priv.PublicKey)
	if types.Address(addr) != file.Address {
		return nil, ErrKeyAddressMismatch
	}

	return &LoadedKey{
		File:       file,
		PrivateKey: priv,
	}, nil
}

func CreateAndSaveKeyFile(path string) (*KeyFile, error) {
	gen, err := GenerateValidatorKey()
	if err != nil {
		return nil, err
	}

	file := &KeyFile{
		Address:    gen.Address,
		PublicKey:  gen.PublicHex,
		PrivateKey: gen.PrivateHex,
	}

	if err := SaveKeyFile(path, file); err != nil {
		return nil, err
	}

	return file, nil
}

func MustLoadAddress(path string) (types.Address, error) {
	loaded, err := LoadKeyFile(path)
	if err != nil {
		return "", err
	}
	if loaded == nil {
		return "", errors.New("loaded validator key is nil")
	}
	return loaded.File.Address, nil
}
