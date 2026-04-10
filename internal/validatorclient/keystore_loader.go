package validatorclient

// CANONICAL OWNERSHIP: validator client package for keystores, duties, slashing protection, signing, and validator service runtime.
// Planned final architectural name is validatorclient after dependency cleanup.

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	blsu "github.com/protolambda/bls12-381-util"
	keystorev4 "github.com/protolambda/go-keystorev4"
)

type LoadedVotingKeystore struct {
	Keystore  *keystorev4.Keystore
	SecretKey *blsu.SecretKey
	PublicKey *blsu.Pubkey
	Path      string
	PublicHex string
}

func LoadVotingKeystore(keystorePath string, secretPath string) (*LoadedVotingKeystore, error) {
	passphrase, err := LoadSecretFile(secretPath)
	if err != nil {
		return nil, fmt.Errorf("load secret file: %w", err)
	}

	data, err := os.ReadFile(keystorePath)
	if err != nil {
		return nil, fmt.Errorf("read keystore: %w", err)
	}

	var ks keystorev4.Keystore
	if err := json.Unmarshal(data, &ks); err != nil {
		return nil, fmt.Errorf("decode keystore json: %w", err)
	}

	secretBytes, err := ks.Decrypt(passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypt keystore: %w", err)
	}

	if len(secretBytes) != 32 {
		return nil, fmt.Errorf("unexpected secret key length: %d", len(secretBytes))
	}

	var skBytes [32]byte
	copy(skBytes[:], secretBytes)

	var sk blsu.SecretKey
	if err := sk.Deserialize(&skBytes); err != nil {
		return nil, fmt.Errorf("deserialize secret key: %w", err)
	}

	pk, err := blsu.SkToPk(&sk)
	if err != nil {
		return nil, fmt.Errorf("derive public key: %w", err)
	}

	pubBytes := pk.Serialize()
	pubHex := hex.EncodeToString(pubBytes[:])

	return &LoadedVotingKeystore{
		Keystore:  &ks,
		SecretKey: &sk,
		PublicKey: pk,
		Path:      ks.Path,
		PublicHex: pubHex,
	}, nil
}
