package crypto

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func EncodeHex(data []byte) string {
	return hex.EncodeToString(data)
}

func DecodeHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidHex, err)
	}
	return b, nil
}
