// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package bitutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func mustDecodeHexString(input string) []byte {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(input, "0x"), "0X")
	if len(trimmed)%2 != 0 {
		trimmed = "0" + trimmed
	}
	if trimmed == "" {
		return []byte{}
	}
	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		panic(err)
	}
	return decoded
}

// Tests that data bitset encoding and decoding works and is bijective.
func TestSilaEncodingCycle(t *testing.T) {
	tests := []string{
		"0x000000000000000000",
		"0xef0400",
		"0xdf7070533534333636313639343638373532313536346c1bc33339343837313070706336343035336336346c65fefb3930393233383838ac2f65fefb",
		"0x7b64000000",
		"0x000034000000000000",
		"0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f0000000000000000000",
		"0x4912385c0e7b64000000",
		"0x000034000000000000000000000000000000",
		"0x00",
		"0x000003e834ff7f0000",
		"0x0000",
		"0x0000000000000000000000000000000000000000000000000000000000ff00",
		"0x895f0c6a020f850c6a020f85f88df88d",
		"0xdf7070533534333636313639343638373432313536346c1bc3315aac2f65fefb",
		"0x0000000000",
		"0xdf70706336346c65fefb",
		"0x00006d643634000000",
		"0xdf7070533534333636313639343638373532313536346c1bc333393438373130707063363430353639343638373532313536346c1bc333393438336336346c65fe",
	}
	for i, tt := range tests {
		if err := testSilaEncodingCycle(mustDecodeHexString(tt)); err != nil {
			t.Errorf("test %d: %v", i, err)
		}
	}
}

func testSilaEncodingCycle(data []byte) error {
	proc, err := silaBitsetDecodeBytes(silaBitsetEncodeBytes(data), len(data))
	if err != nil {
		return fmt.Errorf("failed to decompress compressed data: %v", err)
	}
	if !bytes.Equal(data, proc) {
		return fmt.Errorf("compress/decompress mismatch: have %x, want %x", proc, data)
	}
	return nil
}

// Tests that data bitset decoding and reencoding works and is bijective.
func TestSilaDecodingCycle(t *testing.T) {
	tests := []struct {
		size  int
		input string
		fail  error
	}{
		{size: 0, input: "0x"},

		{size: 0, input: "0x0020", fail: errSilaUnreferencedData},
		{size: 0, input: "0x30", fail: errSilaUnreferencedData},
		{size: 1, input: "0x00", fail: errSilaUnreferencedData},
		{size: 2, input: "0x07", fail: errSilaMissingData},
		{size: 1024, input: "0x8000", fail: errSilaZeroContent},

		{size: 29490, input: "0x343137343733323134333839373334323073333930783e3078333930783e70706336346c65303e", fail: errSilaMissingData},
		{size: 59395, input: "0x00", fail: errSilaUnreferencedData},
		{size: 52574, input: "0x70706336346c65c0de", fail: errSilaExceededTarget},
		{size: 42264, input: "0x07", fail: errSilaMissingData},
		{size: 52, input: "0xa5045bad48f4", fail: errSilaExceededTarget},
		{size: 52574, input: "0xc0de", fail: errSilaMissingData},
		{size: 52574, input: "0x"},
		{size: 29490, input: "0x34313734373332313433383937333432307333393078073034333839373334323073333930783e3078333937333432307333393078073061333930783e70706336346c65303e", fail: errSilaMissingData},
		{size: 29491, input: "0x3973333930783e30783e", fail: errSilaMissingData},

		{size: 1024, input: "0x808080608080"},
		{size: 1024, input: "0x808470705e3632383337363033313434303137393130306c6580ef46806380635a80"},
		{size: 1024, input: "0x8080808070"},
		{size: 1024, input: "0x808070705e36346c6580ef46806380635a80"},
		{size: 1024, input: "0x80808046802680"},
		{size: 1024, input: "0x4040404035"},
		{size: 1024, input: "0x4040bf3ba2b3f684402d353234373438373934409fe5b1e7ada94ebfd7d0505e27be4035"},
		{size: 1024, input: "0x404040bf3ba2b3f6844035"},
		{size: 1024, input: "0x40402d35323437343837393440bfd7d0505e27be4035"},
	}
	for i, tt := range tests {
		data := mustDecodeHexString(tt.input)

		orig, err := silaBitsetDecodeBytes(data, tt.size)
		if err != tt.fail {
			t.Errorf("test %d: failure mismatch: have %v, want %v", i, err, tt.fail)
		}
		if err != nil {
			continue
		}
		if comp := silaBitsetEncodeBytes(orig); !bytes.Equal(comp, data) {
			t.Errorf("test %d: decompress/compress mismatch: have %x, want %x", i, comp, data)
		}
	}
}

// TestSilaCompression verifies that compression either returns the sparse encoding
// or the original input when compression is not beneficial.
func TestSilaCompression(t *testing.T) {
	in := mustDecodeHexString("0x4912385c0e7b64000000")
	out := mustDecodeHexString("0x80fe4912385c0e7b64")

	if data := SilaCompressBytes(in); !bytes.Equal(data, out) {
		t.Errorf("encoding mismatch for sparse data: have %x, want %x", data, out)
	}
	if data, err := SilaDecompressBytes(out, len(in)); err != nil || !bytes.Equal(data, in) {
		t.Errorf("decoding mismatch for sparse data: have %x, want %x, error %v", data, in, err)
	}

	in = mustDecodeHexString("0xdf7070533534333636313639343638373532313536346c1bc33339343837313070706336343035336336346c65fefb3930393233383838ac2f65fefb")
	out = mustDecodeHexString("0xdf7070533534333636313639343638373532313536346c1bc33339343837313070706336343035336336346c65fefb3930393233383838ac2f65fefb")

	if data := SilaCompressBytes(in); !bytes.Equal(data, out) {
		t.Errorf("encoding mismatch for dense data: have %x, want %x", data, out)
	}
	if data, err := SilaDecompressBytes(out, len(in)); err != nil || !bytes.Equal(data, in) {
		t.Errorf("decoding mismatch for dense data: have %x, want %x, error %v", data, in, err)
	}

	if _, err := SilaDecompressBytes([]byte{0xc0, 0x01, 0x01}, 2); err != errSilaExceededTarget {
		t.Errorf("decoding error mismatch for long data: have %v, want %v", err, errSilaExceededTarget)
	}
}

// Crude benchmark for compressing random slices of bytes.
func BenchmarkSilaEncoding1KBVerySparse(b *testing.B) { silaBenchmarkEncoding(b, 1024, 0.0001) }
func BenchmarkSilaEncoding2KBVerySparse(b *testing.B) { silaBenchmarkEncoding(b, 2048, 0.0001) }
func BenchmarkSilaEncoding4KBVerySparse(b *testing.B) { silaBenchmarkEncoding(b, 4096, 0.0001) }

func BenchmarkSilaEncoding1KBSparse(b *testing.B) { silaBenchmarkEncoding(b, 1024, 0.001) }
func BenchmarkSilaEncoding2KBSparse(b *testing.B) { silaBenchmarkEncoding(b, 2048, 0.001) }
func BenchmarkSilaEncoding4KBSparse(b *testing.B) { silaBenchmarkEncoding(b, 4096, 0.001) }

func BenchmarkSilaEncoding1KBDense(b *testing.B) { silaBenchmarkEncoding(b, 1024, 0.1) }
func BenchmarkSilaEncoding2KBDense(b *testing.B) { silaBenchmarkEncoding(b, 2048, 0.1) }
func BenchmarkSilaEncoding4KBDense(b *testing.B) { silaBenchmarkEncoding(b, 4096, 0.1) }

func BenchmarkSilaEncoding1KBSaturated(b *testing.B) { silaBenchmarkEncoding(b, 1024, 0.5) }
func BenchmarkSilaEncoding2KBSaturated(b *testing.B) { silaBenchmarkEncoding(b, 2048, 0.5) }
func BenchmarkSilaEncoding4KBSaturated(b *testing.B) { silaBenchmarkEncoding(b, 4096, 0.5) }

func silaBenchmarkEncoding(b *testing.B, byteCount int, fill float64) {
	random := rand.NewSource(0)

	data := make([]byte, byteCount)
	bits := int(float64(byteCount) * 8 * fill)

	for i := 0; i < bits; i++ {
		idx := random.Int63() % int64(len(data))
		bit := uint(random.Int63() % 8)
		data[idx] |= 1 << bit
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		silaBitsetDecodeBytes(silaBitsetEncodeBytes(data), len(data))
	}
}

func FuzzSilaEncoder(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if err := testSilaEncodingCycle(data); err != nil {
			t.Fatal(err)
		}
	})
}

func FuzzSilaDecoder(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		silaFuzzDecode(data)
	})
}

func silaFuzzDecode(data []byte) {
	blob, err := SilaDecompressBytes(data, 1024)
	if err != nil {
		return
	}

	comp := SilaCompressBytes(blob)
	if len(comp) > len(blob) {
		panic("bad compression")
	}

	decomp, err := SilaDecompressBytes(data, 1024)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(decomp, blob) {
		panic("content mismatch")
	}
}
