package validatorclient

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const (
	DefaultVotingPath     = "m/12381/3600/0/0/0"
	DefaultWithdrawalPath = "m/12381/3600/0/0"
)

type WithdrawalKeystoreResult struct {
	PublicKeyHex string
	Path         string
	KeystorePath string
	SecretPath   string
}

func randomPassphraseBytes() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	out := base64.RawStdEncoding.EncodeToString(b)
	return []byte(out), nil
}

func CreateWithdrawalKeystore(keystorePath string, secretPath string, derivationPath string, passphrase []byte) (*WithdrawalKeystoreResult, error) {
	if derivationPath == "" {
		derivationPath = DefaultWithdrawalPath
	}
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("withdrawal keystore passphrase is empty")
	}

	key, err := GenerateVotingKey(derivationPath)
	if err != nil {
		return nil, err
	}

	res, err := createKeystoreFromBundle(
		key,
		keystorePath,
		secretPath,
		passphrase,
		"Sila validator withdrawal key",
	)
	if err != nil {
		return nil, err
	}

	return &WithdrawalKeystoreResult{
		PublicKeyHex: res.PublicKeyHex,
		Path:         res.Path,
		KeystorePath: res.KeystorePath,
		SecretPath:   res.SecretPath,
	}, nil
}
