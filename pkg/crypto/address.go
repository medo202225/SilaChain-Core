package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"

	"silachain/pkg/types"
)

func PublicKeyToAddress(pub *ecdsa.PublicKey) types.Address {
	pubBytes := PublicKeyToBytes(pub)
	if len(pubBytes) == 0 {
		return types.Address("")
	}

	// Ethereum-style address derivation from uncompressed public key, with Sila identity prefix.
	hasher := sha3.NewLegacyKeccak256()
	_, _ = hasher.Write(pubBytes[1:])
	sum := hasher.Sum(nil)

	return types.Address("SILA_" + hex.EncodeToString(sum[12:]))
}

func MustPublicKeyToAddress(pub *ecdsa.PublicKey) types.Address {
	addr := PublicKeyToAddress(pub)
	if addr == "" {
		panic(fmt.Errorf("failed to derive address from public key"))
	}
	return addr
}
