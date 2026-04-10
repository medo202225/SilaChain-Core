package validatorclient

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	blshd "github.com/protolambda/bls12-381-hd"
	blsu "github.com/protolambda/bls12-381-util"
)

type BLSKeyBundle struct {
	Path      string
	Secret    [32]byte
	Public    [48]byte
	SecretHex string
	PublicHex string
}

func randomSeed32() ([]byte, error) {
	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	if err != nil {
		return nil, err
	}
	return seed, nil
}

func GenerateVotingKey(path string) (*BLSKeyBundle, error) {
	if path == "" {
		path = "m/12381/3600/0/0/0"
	}

	seed, err := randomSeed32()
	if err != nil {
		return nil, fmt.Errorf("generate seed: %w", err)
	}

	skBytes, err := blshd.SecretKeyFromHD(seed, path)
	if err != nil {
		return nil, fmt.Errorf("derive BLS secret key: %w", err)
	}

	var sk blsu.SecretKey
	if err := sk.Deserialize(skBytes); err != nil {
		return nil, fmt.Errorf("deserialize BLS secret key: %w", err)
	}

	pk, err := blsu.SkToPk(&sk)
	if err != nil {
		return nil, fmt.Errorf("derive BLS public key: %w", err)
	}

	pubBytes := pk.Serialize()

	return &BLSKeyBundle{
		Path:      path,
		Secret:    *skBytes,
		Public:    pubBytes,
		SecretHex: hex.EncodeToString(skBytes[:]),
		PublicHex: hex.EncodeToString(pubBytes[:]),
	}, nil
}
