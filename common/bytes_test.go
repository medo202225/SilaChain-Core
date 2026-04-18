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

package common

import (
	"bytes"
	"testing"

	"silachain/common/hexutil"
)

func TestFromHex(t *testing.T) {
	tests := []struct {
		input  string
		output []byte
	}{
		{"", []byte{}},
		{"0x", []byte{}},
		{"0X", []byte{}},
		{"0x0", []byte{0}},
		{"0x00", []byte{0}},
		{"0x01", []byte{1}},
		{"0x1", []byte{1}},
		{"0x10", []byte{16}},
		{"10", []byte{16}},
		{"0xff", []byte{255}},
		{"ff", []byte{255}},
		{"0xFF", []byte{255}},
		{"FF", []byte{255}},
		{"0xffff", []byte{255, 255}},
		{"ffff", []byte{255, 255}},
		{"0xffffffff", []byte{255, 255, 255, 255}},
		{"ffffffff", []byte{255, 255, 255, 255}},
	}

	for _, test := range tests {
		got := FromHex(test.input)
		if !bytes.Equal(got, test.output) {
			t.Errorf("FromHex(%q) = %x, want %x", test.input, got, test.output)
		}
	}
}

func TestCopyBytes(t *testing.T) {
	tests := []struct {
		input  []byte
		output []byte
	}{
		{nil, nil},
		{[]byte{}, []byte{}},
		{[]byte{0}, []byte{0}},
		{[]byte{1, 2, 3}, []byte{1, 2, 3}},
	}

	for _, test := range tests {
		got := CopyBytes(test.input)
		if !bytes.Equal(got, test.output) {
			t.Errorf("CopyBytes(%x) = %x, want %x", test.input, got, test.output)
		}
		if test.input != nil && &got[0] == &test.input[0] {
			t.Errorf("CopyBytes(%x): slice points to same array", test.input)
		}
	}
}

func TestHas0xPrefix(t *testing.T) {
	tests := []struct {
		input  string
		output bool
	}{
		{"", false},
		{"0", false},
		{"0x", true},
		{"0X", true},
		{"0x123", true},
		{"0X123", true},
		{"0xX", true},
		{"0xx", true},
		{"1x", false},
		{"x0", false},
	}

	for _, test := range tests {
		got := has0xPrefix(test.input)
		if got != test.output {
			t.Errorf("has0xPrefix(%q) = %v, want %v", test.input, got, test.output)
		}
	}
}

func TestIsHex(t *testing.T) {
	tests := []struct {
		input  string
		output bool
	}{
		{"", true},
		{"0", false},
		{"00", true},
		{"01", true},
		{"0g", false},
		{"0G", false},
		{"ff", true},
		{"FF", true},
		{"ffff", true},
		{"fffff", false},
		{"0x00", false},
	}

	for _, test := range tests {
		got := isHex(test.input)
		if got != test.output {
			t.Errorf("isHex(%q) = %v, want %v", test.input, got, test.output)
		}
	}
}

func TestBytes2Hex(t *testing.T) {
	tests := []struct {
		input  []byte
		output string
	}{
		{[]byte{}, ""},
		{[]byte{0}, "00"},
		{[]byte{1}, "01"},
		{[]byte{255}, "ff"},
		{[]byte{1, 2, 3}, "010203"},
	}

	for _, test := range tests {
		got := Bytes2Hex(test.input)
		if got != test.output {
			t.Errorf("Bytes2Hex(%x) = %q, want %q", test.input, got, test.output)
		}
	}
}

func TestHex2Bytes(t *testing.T) {
	tests := []struct {
		input  string
		output []byte
	}{
		{"", []byte{}},
		{"00", []byte{0}},
		{"01", []byte{1}},
		{"ff", []byte{255}},
		{"FF", []byte{255}},
		{"010203", []byte{1, 2, 3}},
	}

	for _, test := range tests {
		got := Hex2Bytes(test.input)
		if !bytes.Equal(got, test.output) {
			t.Errorf("Hex2Bytes(%q) = %x, want %x", test.input, got, test.output)
		}
	}
}

func TestHex2BytesFixed(t *testing.T) {
	tests := []struct {
		input  string
		flen   int
		output []byte
	}{
		{"", 4, []byte{0, 0, 0, 0}},
		{"00", 4, []byte{0, 0, 0, 0}},
		{"01020304", 4, []byte{1, 2, 3, 4}},
		{"0102030405", 4, []byte{2, 3, 4, 5}},
		{"010203", 4, []byte{0, 1, 2, 3}},
	}

	for _, test := range tests {
		got := Hex2BytesFixed(test.input, test.flen)
		if !bytes.Equal(got, test.output) {
			t.Errorf("Hex2BytesFixed(%q, %d) = %x, want %x", test.input, test.flen, got, test.output)
		}
	}
}

func TestParseHexOrString(t *testing.T) {
	tests := []struct {
		input   string
		output  []byte
		wantErr bool
	}{
		{"", []byte{}, false},
		{"0x", []byte{}, false},
		{"0x00", []byte{0}, false},
		{"0x01", []byte{1}, false},
		{"hello", []byte("hello"), false},
		{"0xZZ", nil, true},
	}

	for _, test := range tests {
		got, err := ParseHexOrString(test.input)
		if test.wantErr && err == nil {
			t.Errorf("ParseHexOrString(%q) expected error", test.input)
		}
		if !test.wantErr && err != nil {
			t.Errorf("ParseHexOrString(%q) unexpected error: %v", test.input, err)
		}
		if !bytes.Equal(got, test.output) {
			t.Errorf("ParseHexOrString(%q) = %x, want %x", test.input, got, test.output)
		}
	}
}

func TestRightPadBytes(t *testing.T) {
	tests := []struct {
		slice  []byte
		l      int
		output []byte
	}{
		{[]byte{1, 2, 3}, 5, []byte{1, 2, 3, 0, 0}},
		{[]byte{1, 2, 3}, 3, []byte{1, 2, 3}},
		{[]byte{1, 2, 3}, 2, []byte{1, 2, 3}},
		{[]byte{}, 2, []byte{0, 0}},
	}

	for _, test := range tests {
		got := RightPadBytes(test.slice, test.l)
		if !bytes.Equal(got, test.output) {
			t.Errorf("RightPadBytes(%x, %d) = %x, want %x", test.slice, test.l, got, test.output)
		}
	}
}

func TestLeftPadBytes(t *testing.T) {
	tests := []struct {
		slice  []byte
		l      int
		output []byte
	}{
		{[]byte{1, 2, 3}, 5, []byte{0, 0, 1, 2, 3}},
		{[]byte{1, 2, 3}, 3, []byte{1, 2, 3}},
		{[]byte{1, 2, 3}, 2, []byte{1, 2, 3}},
		{[]byte{}, 2, []byte{0, 0}},
	}

	for _, test := range tests {
		got := LeftPadBytes(test.slice, test.l)
		if !bytes.Equal(got, test.output) {
			t.Errorf("LeftPadBytes(%x, %d) = %x, want %x", test.slice, test.l, got, test.output)
		}
	}
}

func TestTrimLeftZeroes(t *testing.T) {
	tests := []struct {
		input  []byte
		output []byte
	}{
		{[]byte{}, []byte{}},
		{[]byte{0}, []byte{}},
		{[]byte{0, 0}, []byte{}},
		{[]byte{0, 1}, []byte{1}},
		{[]byte{1, 0}, []byte{1, 0}},
		{[]byte{1, 2, 3}, []byte{1, 2, 3}},
	}

	for _, test := range tests {
		got := TrimLeftZeroes(test.input)
		if !bytes.Equal(got, test.output) {
			t.Errorf("TrimLeftZeroes(%x) = %x, want %x", test.input, got, test.output)
		}
	}
}

func TestTrimRightZeroes(t *testing.T) {
	tests := []struct {
		input  []byte
		output []byte
	}{
		{[]byte{}, []byte{}},
		{[]byte{0}, []byte{}},
		{[]byte{0, 0}, []byte{}},
		{[]byte{1, 0}, []byte{1}},
		{[]byte{0, 1}, []byte{0, 1}},
		{[]byte{1, 2, 3}, []byte{1, 2, 3}},
	}

	for _, test := range tests {
		got := TrimRightZeroes(test.input)
		if !bytes.Equal(got, test.output) {
			t.Errorf("TrimRightZeroes(%x) = %x, want %x", test.input, got, test.output)
		}
	}
}

func BenchmarkFromHex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromHex("0x123456789abcdef")
	}
}

func BenchmarkBytes2Hex(b *testing.B) {
	data := Hex2Bytes("123456789abcdef")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Bytes2Hex(data)
	}
}

func BenchmarkHex2Bytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Hex2Bytes("123456789abcdef")
	}
}

func BenchmarkParseHexOrString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseHexOrString("0x123456789abcdef")
	}
}

func TestHexUtilCompatibility(t *testing.T) {
	// Test compatibility with hexutil package
	tests := []string{
		"0x",
		"0x0",
		"0x00",
		"0x01",
		"0xff",
		"0xffff",
	}

	for _, test := range tests {
		got := FromHex(test)
		want, _ := hexutil.Decode(test)
		if !bytes.Equal(got, want) {
			t.Errorf("FromHex(%q) = %x, hexutil.Decode = %x", test, got, want)
		}
	}
}
