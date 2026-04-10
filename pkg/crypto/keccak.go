package crypto

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

func Keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(data)
	return h.Sum(nil)
}

func Keccak256Hex(data []byte) string {
	return hex.EncodeToString(Keccak256(data))
}
