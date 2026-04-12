// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// DecodeHexBytes converts a hex string into bytes.
// It accepts values with or without a 0x prefix.
// Odd-length hex strings are left-padded with a zero nibble.
func DecodeHexBytes(input string) ([]byte, error) {
	normalized := strings.TrimSpace(input)
	if strings.HasPrefix(normalized, "0x") || strings.HasPrefix(normalized, "0X") {
		normalized = normalized[2:]
	}
	if normalized == "" {
		return []byte{}, nil
	}
	if len(normalized)%2 != 0 {
		normalized = "0" + normalized
	}
	decoded, err := hex.DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("decode hex bytes: %w", err)
	}
	return decoded, nil
}

// MustDecodeHexBytes converts a hex string into bytes and returns nil on failure.
func MustDecodeHexBytes(input string) []byte {
	decoded, err := DecodeHexBytes(input)
	if err != nil {
		return nil
	}
	return decoded
}

// CloneBytes returns a copy of the provided byte slice.
func CloneBytes(input []byte) []byte {
	if input == nil {
		return nil
	}
	cloned := make([]byte, len(input))
	copy(cloned, input)
	return cloned
}

// HasHexPrefix reports whether the string begins with 0x or 0X.
func HasHexPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

// IsHexDigit reports whether b is a valid hexadecimal digit.
func IsHexDigit(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'f') || ('A' <= b && b <= 'F')
}

// IsEvenLengthHex reports whether the full string is valid even-length hexadecimal.
func IsEvenLengthHex(input string) bool {
	if len(input)%2 != 0 {
		return false
	}
	for i := 0; i < len(input); i++ {
		if !IsHexDigit(input[i]) {
			return false
		}
	}
	return true
}

// EncodeHexBytes returns the lowercase hexadecimal form of the provided bytes.
func EncodeHexBytes(input []byte) string {
	return hex.EncodeToString(input)
}

// DecodeHexBytesFixed converts a hex string into a fixed-length byte slice.
// If decoded bytes are longer than size, the rightmost bytes are kept.
// If shorter, the result is left-padded with zeroes.
func DecodeHexBytesFixed(input string, size int) ([]byte, error) {
	if size < 0 {
		return nil, fmt.Errorf("fixed size must be non-negative")
	}
	decoded, err := DecodeHexBytes(input)
	if err != nil {
		return nil, err
	}
	if len(decoded) == size {
		return decoded, nil
	}
	if len(decoded) > size {
		return decoded[len(decoded)-size:], nil
	}
	out := make([]byte, size)
	copy(out[size-len(decoded):], decoded)
	return out, nil
}

// ParseHexOrRawBytes decodes a 0x-prefixed value as hex.
// Without the prefix, it returns the raw UTF-8 bytes of the input string.
func ParseHexOrRawBytes(input string) ([]byte, error) {
	trimmed := strings.TrimSpace(input)
	if HasHexPrefix(trimmed) {
		return DecodeHexBytes(trimmed)
	}
	return []byte(trimmed), nil
}

// RightPadZeroBytes pads the slice with zero bytes on the right up to size.
func RightPadZeroBytes(input []byte, size int) []byte {
	if size <= len(input) {
		return CloneBytes(input)
	}
	out := make([]byte, size)
	copy(out, input)
	return out
}

// LeftPadZeroBytes pads the slice with zero bytes on the left up to size.
func LeftPadZeroBytes(input []byte, size int) []byte {
	if size <= len(input) {
		return CloneBytes(input)
	}
	out := make([]byte, size)
	copy(out[size-len(input):], input)
	return out
}

// TrimLeadingZeroBytes removes leading zero bytes.
func TrimLeadingZeroBytes(input []byte) []byte {
	index := 0
	for index < len(input) && input[index] == 0 {
		index++
	}
	return input[index:]
}

// TrimTrailingZeroBytes removes trailing zero bytes.
func TrimTrailingZeroBytes(input []byte) []byte {
	index := len(input)
	for index > 0 && input[index-1] == 0 {
		index--
	}
	return input[:index]
}
