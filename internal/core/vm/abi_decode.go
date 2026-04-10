package vm

import (
	"encoding/hex"
	"errors"
	"strings"
)

var ErrInvalidABIInput = errors.New("vm: invalid abi input")

func DecodeABIReturnUint256(data []byte) (string, error) {
	if len(data) < 32 {
		return "", ErrInvalidABIInput
	}
	word := NewWordFromBytes(data[:32])
	return word.String(), nil
}

func DecodeABIReturnBool(data []byte) (bool, error) {
	if len(data) < 32 {
		return false, ErrInvalidABIInput
	}
	word := NewWordFromBytes(data[:32])
	return word.Sign() != 0, nil
}

func DecodeABIReturnAddress(data []byte) (string, error) {
	if len(data) < 32 {
		return "", ErrInvalidABIInput
	}
	return hex.EncodeToString(data[12:32]), nil
}

func DecodeABIBytes32(data []byte) (string, error) {
	if len(data) < 32 {
		return "", ErrInvalidABIInput
	}
	return hex.EncodeToString(data[:32]), nil
}

func DecodeABIHexInput(input string) ([]byte, error) {
	s := strings.TrimSpace(input)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if s == "" {
		return nil, nil
	}
	if len(s)%2 != 0 {
		s = "0" + s
	}
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidABIInput
	}
	return decoded, nil
}
