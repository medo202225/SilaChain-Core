package crypto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func HashBytes(data []byte) string {
	return hex.EncodeToString(Keccak256(data))
}

func HashString(s string) string {
	return HashBytes([]byte(s))
}

func HashJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal for hashing: %w", err)
	}
	return HashBytes(b), nil
}

func HashJSONBytes(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal for hashing: %w", err)
	}
	return Keccak256(b), nil
}
