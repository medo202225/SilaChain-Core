package vm

import (
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/sha3"
)

const FunctionSelectorSize = 4

func NormalizeFunctionSignature(signature string) string {
	return strings.TrimSpace(signature)
}

func FunctionSelector(signature string) []byte {
	sig := NormalizeFunctionSignature(signature)
	if sig == "" {
		return nil
	}

	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(sig))
	sum := h.Sum(nil)

	out := make([]byte, FunctionSelectorSize)
	copy(out, sum[:FunctionSelectorSize])
	return out
}

func FunctionSelectorHex(signature string) string {
	selector := FunctionSelector(signature)
	if len(selector) == 0 {
		return ""
	}
	return hex.EncodeToString(selector)
}

func MatchFunctionSelector(input []byte, signature string) bool {
	if len(input) < FunctionSelectorSize {
		return false
	}

	expected := FunctionSelector(signature)
	if len(expected) != FunctionSelectorSize {
		return false
	}

	for i := 0; i < FunctionSelectorSize; i++ {
		if input[i] != expected[i] {
			return false
		}
	}
	return true
}
