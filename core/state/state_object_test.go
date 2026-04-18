// Copyright 2026 The SILA Authors
// This file is part of the sila-library.
//
// The sila-library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila-library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila-library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"bytes"
	"testing"

	"silachain/common"
)

// BenchmarkCutOriginal - SILA benchmark for trimming zeroes using bytes.TrimLeft
func BenchmarkCutOriginal(b *testing.B) {
	value := common.HexToHash("0x01")
	for b.Loop() {
		bytes.TrimLeft(value[:], "\x00")
	}
}

// BenchmarkCutsetterFn - SILA benchmark using custom cutset function
func BenchmarkCutsetterFn(b *testing.B) {
	value := common.HexToHash("0x01")
	cutSetFn := func(r rune) bool { return r == 0 }
	for b.Loop() {
		bytes.TrimLeftFunc(value[:], cutSetFn)
	}
}

// BenchmarkCutCustomTrim - SILA benchmark for the custom TrimLeftZeroes function
func BenchmarkCutCustomTrim(b *testing.B) {
	value := common.HexToHash("0x01")
	for b.Loop() {
		common.TrimLeftZeroes(value[:])
	}
}

// BenchmarkSILATrimZeroesIdentity - SILA-specific benchmark for identity hash trimming
func BenchmarkSILATrimZeroesIdentity(b *testing.B) {
	value := append([]byte{0x53, 0x49, 0x4c, 0x41}, common.HexToHash("0x01").Bytes()...)
	for b.Loop() {
		common.TrimLeftZeroes(value[:])
	}
}

// TestSILATrimLeftZeroes - Tests SILA TrimLeftZeroes function with various inputs
func TestSILATrimLeftZeroes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "SILA single byte",
			input:    []byte{0x01},
			expected: []byte{0x01},
		},
		{
			name:     "SILA leading zeros",
			input:    []byte{0x00, 0x00, 0x01},
			expected: []byte{0x01},
		},
		{
			name:     "SILA all zeros",
			input:    []byte{0x00, 0x00, 0x00},
			expected: []byte{},
		},
		{
			name:     "SILA identity with signature",
			input:    []byte{0x53, 0x49, 0x4c, 0x41, 0x00, 0x00, 0x01},
			expected: []byte{0x53, 0x49, 0x4c, 0x41, 0x01},
		},
		{
			name:     "SILA empty slice",
			input:    []byte{},
			expected: []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.TrimLeftZeroes(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("SILA TrimLeftZeroes(%x) = %x, want %x", tt.input, result, tt.expected)
			}
		})
	}
}
