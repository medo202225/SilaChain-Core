// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package hexutil

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

var (
	ErrBlankInput        = errors.New("blank hex input")
	ErrPrefixRequired    = errors.New("0x prefix required")
	ErrOddHexLength      = errors.New("odd-length hex data")
	ErrInvalidHex        = errors.New("invalid hex data")
	ErrEmptyQuantity     = errors.New("empty hex quantity")
	ErrLeadingZeroDigits = errors.New("hex quantity has leading zero digits")
	ErrUint64Overflow    = errors.New("hex quantity exceeds uint64")
	ErrBig256Overflow    = errors.New("hex quantity exceeds 256 bits")
)

func Has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func NormalizePrefixedHex(input string) (string, error) {
	if input == "" {
		return "", ErrBlankInput
	}
	if !Has0xPrefix(input) {
		return "", ErrPrefixRequired
	}
	return input[2:], nil
}

func DecodeBytes(input string) ([]byte, error) {
	raw, err := NormalizePrefixedHex(input)
	if err != nil {
		return nil, err
	}
	if len(raw)%2 != 0 {
		return nil, ErrOddHexLength
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, ErrInvalidHex
	}
	return decoded, nil
}

func MustDecodeBytes(input string) []byte {
	decoded, err := DecodeBytes(input)
	if err != nil {
		panic(err)
	}
	return decoded
}

func EncodeBytes(data []byte) string {
	out := make([]byte, len(data)*2+2)
	copy(out, "0x")
	hex.Encode(out[2:], data)
	return string(out)
}

func DecodeQuantityUint64(input string) (uint64, error) {
	raw, err := normalizeQuantity(input)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(raw, 16, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, ErrUint64Overflow
		}
		return 0, ErrInvalidHex
	}
	return value, nil
}

func MustDecodeQuantityUint64(input string) uint64 {
	value, err := DecodeQuantityUint64(input)
	if err != nil {
		panic(err)
	}
	return value
}

func EncodeQuantityUint64(value uint64) string {
	buf := make([]byte, 2, 18)
	copy(buf, "0x")
	return string(strconv.AppendUint(buf, value, 16))
}

func DecodeQuantityBig(input string) (*big.Int, error) {
	raw, err := normalizeQuantity(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > 64 {
		return nil, ErrBig256Overflow
	}
	decoded := new(big.Int)
	if _, ok := decoded.SetString(raw, 16); !ok {
		return nil, ErrInvalidHex
	}
	return decoded, nil
}

func MustDecodeQuantityBig(input string) *big.Int {
	value, err := DecodeQuantityBig(input)
	if err != nil {
		panic(err)
	}
	return value
}

func EncodeQuantityBig(value *big.Int) string {
	if value == nil || value.Sign() == 0 {
		return "0x0"
	}
	if value.Sign() < 0 {
		text := value.Text(16)
		return "-0x" + text[1:]
	}
	return "0x" + value.Text(16)
}

func normalizeQuantity(input string) (string, error) {
	if input == "" {
		return "", ErrBlankInput
	}
	if !Has0xPrefix(input) {
		return "", ErrPrefixRequired
	}
	raw := input[2:]
	if raw == "" {
		return "", ErrEmptyQuantity
	}
	if len(raw) > 1 && raw[0] == '0' {
		return "", ErrLeadingZeroDigits
	}
	if !isHexString(raw) {
		return "", ErrInvalidHex
	}
	return raw, nil
}

func isHexString(input string) bool {
	for _, r := range input {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func DescribeQuantity(input string) (string, error) {
	value, err := DecodeQuantityBig(input)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("hex=%s dec=%s", strings.ToLower(input), value.String()), nil
}
