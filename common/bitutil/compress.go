// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package bitutil

import "errors"

var (
	// errSilaMissingData is returned when the compressed stream references bytes
	// beyond the available input.
	errSilaMissingData = errors.New("missing bytes on input")

	// errSilaUnreferencedData is returned when decompression leaves unused bytes behind.
	errSilaUnreferencedData = errors.New("extra bytes on input")

	// errSilaExceededTarget is returned when decompression would write beyond target size.
	errSilaExceededTarget = errors.New("target data size exceeded")

	// errSilaZeroContent is returned when a referenced non-zero byte is actually zero.
	errSilaZeroContent = errors.New("zero byte in input content")
)

// SilaCompressBytes compresses sparse byte slices using a recursive bitset scheme.
// If the compressed form is not smaller than the original, a copy of the original is returned.
func SilaCompressBytes(data []byte) []byte {
	if out := silaBitsetEncodeBytes(data); len(out) < len(data) {
		return out
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	return copied
}

func silaBitsetEncodeBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	if len(data) == 1 {
		if data[0] == 0 {
			return nil
		}
		return data
	}

	nonZeroBitset := make([]byte, (len(data)+7)/8)
	nonZeroBytes := make([]byte, 0, len(data))

	for i, b := range data {
		if b != 0 {
			nonZeroBytes = append(nonZeroBytes, b)
			nonZeroBitset[i/8] |= 1 << byte(7-i%8)
		}
	}
	if len(nonZeroBytes) == 0 {
		return nil
	}
	return append(silaBitsetEncodeBytes(nonZeroBitset), nonZeroBytes...)
}

// SilaDecompressBytes decompresses data into a buffer of known target size.
func SilaDecompressBytes(data []byte, target int) ([]byte, error) {
	if len(data) > target {
		return nil, errSilaExceededTarget
	}
	if len(data) == target {
		copied := make([]byte, len(data))
		copy(copied, data)
		return copied, nil
	}
	return silaBitsetDecodeBytes(data, target)
}

func silaBitsetDecodeBytes(data []byte, target int) ([]byte, error) {
	out, size, err := silaBitsetDecodePartialBytes(data, target)
	if err != nil {
		return nil, err
	}
	if size != len(data) {
		return nil, errSilaUnreferencedData
	}
	return out, nil
}

func silaBitsetDecodePartialBytes(data []byte, target int) ([]byte, int, error) {
	if target == 0 {
		return nil, 0, nil
	}

	decompressed := make([]byte, target)
	if len(data) == 0 {
		return decompressed, 0, nil
	}
	if target == 1 {
		decompressed[0] = data[0]
		if data[0] != 0 {
			return decompressed, 1, nil
		}
		return decompressed, 0, nil
	}

	nonZeroBitset, ptr, err := silaBitsetDecodePartialBytes(data, (target+7)/8)
	if err != nil {
		return nil, ptr, err
	}

	for i := 0; i < 8*len(nonZeroBitset); i++ {
		if nonZeroBitset[i/8]&(1<<byte(7-i%8)) != 0 {
			if ptr >= len(data) {
				return nil, 0, errSilaMissingData
			}
			if i >= len(decompressed) {
				return nil, 0, errSilaExceededTarget
			}
			if data[ptr] == 0 {
				return nil, 0, errSilaZeroContent
			}
			decompressed[i] = data[ptr]
			ptr++
		}
	}
	return decompressed, ptr, nil
}
