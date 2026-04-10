package validatorclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	keystorev4 "github.com/protolambda/go-keystorev4"
)

type KeystoreCreateResult struct {
	PublicKeyHex string
	Path         string
	KeystorePath string
	SecretPath   string
}

func SavePasswordFile(path string, passphrase []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, passphrase, 0o600)
}

func createKeystoreFromBundle(bundle *BLSKeyBundle, keystorePath string, secretPath string, passphrase []byte, description string) (*KeystoreCreateResult, error) {
	if bundle == nil {
		return nil, fmt.Errorf("BLS key bundle is nil")
	}

	ks, err := keystorev4.EncryptToKeystore(bundle.Secret[:], passphrase)
	if err != nil {
		return nil, fmt.Errorf("encrypt to EIP-2335 keystore: %w", err)
	}

	ks.Path = bundle.Path
	ks.Pubkey = keystorev4.JsonBytes(bundle.Public[:])
	ks.Description = description

	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(keystorePath), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keystorePath, data, 0o600); err != nil {
		return nil, err
	}

	if err := SavePasswordFile(secretPath, passphrase); err != nil {
		return nil, err
	}

	return &KeystoreCreateResult{
		PublicKeyHex: bundle.PublicHex,
		Path:         bundle.Path,
		KeystorePath: keystorePath,
		SecretPath:   secretPath,
	}, nil
}

func CreateVotingKeystore(keystorePath string, secretPath string, derivationPath string, passphrase []byte) (*KeystoreCreateResult, error) {
	key, err := GenerateVotingKey(derivationPath)
	if err != nil {
		return nil, err
	}

	return createKeystoreFromBundle(
		key,
		keystorePath,
		secretPath,
		passphrase,
		"Sila validator voting key",
	)
}
