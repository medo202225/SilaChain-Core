// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"bytes"
	"testing"
)

func TestCloneBytes(t *testing.T) {
	input := []byte{1, 2, 3, 4}

	cloned := CloneBytes(input)
	if !bytes.Equal(cloned, []byte{1, 2, 3, 4}) {
		t.Fatal("cloned bytes do not match input")
	}

	cloned[0] = 99
	if bytes.Equal(cloned, input) {
		t.Fatal("clone shares backing data with input")
	}
}

func TestLeftPadZeroBytes(t *testing.T) {
	value := []byte{1, 2, 3, 4}
	expected := []byte{0, 0, 0, 0, 1, 2, 3, 4}

	if got := LeftPadZeroBytes(value, 8); !bytes.Equal(got, expected) {
		t.Fatalf("LeftPadZeroBytes(%v, 8) = %v", value, got)
	}

	if got := LeftPadZeroBytes(value, 2); !bytes.Equal(got, value) {
		t.Fatalf("LeftPadZeroBytes(%v, 2) = %v", value, got)
	}
}

func TestRightPadZeroBytes(t *testing.T) {
	value := []byte{1, 2, 3, 4}
	expected := []byte{1, 2, 3, 4, 0, 0, 0, 0}

	if got := RightPadZeroBytes(value, 8); !bytes.Equal(got, expected) {
		t.Fatalf("RightPadZeroBytes(%v, 8) = %v", value, got)
	}

	if got := RightPadZeroBytes(value, 2); !bytes.Equal(got, value) {
		t.Fatalf("RightPadZeroBytes(%v, 2) = %v", value, got)
	}
}

func TestDecodeHexBytes(t *testing.T) {
	input := "0x01"
	expected := []byte{1}

	got, err := DecodeHexBytes(input)
	if err != nil {
		t.Fatalf("DecodeHexBytes(%q) returned error: %v", input, err)
	}
	if !bytes.Equal(expected, got) {
		t.Errorf("expected %x got %x", expected, got)
	}
}

func TestIsEvenLengthHex(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
	}{
		{"", true},
		{"0", false},
		{"00", true},
		{"a9e67e", true},
		{"A9E67E", true},
		{"0xa9e67e", false},
		{"a9e67e001", false},
		{"0xHELLO_MY_NAME_IS_SILA_123", false},
	}

	for _, test := range tests {
		if ok := IsEvenLengthHex(test.input); ok != test.ok {
			t.Errorf("IsEvenLengthHex(%q) = %v, want %v", test.input, ok, test.ok)
		}
	}
}

func TestDecodeHexBytesOddLength(t *testing.T) {
	input := "0x1"
	expected := []byte{1}

	got, err := DecodeHexBytes(input)
	if err != nil {
		t.Fatalf("DecodeHexBytes(%q) returned error: %v", input, err)
	}
	if !bytes.Equal(expected, got) {
		t.Errorf("expected %x got %x", expected, got)
	}
}

func TestDecodeHexBytesNoPrefixOddLength(t *testing.T) {
	input := "1"
	expected := []byte{1}

	got, err := DecodeHexBytes(input)
	if err != nil {
		t.Fatalf("DecodeHexBytes(%q) returned error: %v", input, err)
	}
	if !bytes.Equal(expected, got) {
		t.Errorf("expected %x got %x", expected, got)
	}
}

func TestTrimTrailingZeroBytes(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{MustDecodeHexBytes("0x00ffff00ff0000"), MustDecodeHexBytes("0x00ffff00ff")},
		{MustDecodeHexBytes("0x00000000000000"), []byte{}},
		{MustDecodeHexBytes("0xff"), MustDecodeHexBytes("0xff")},
		{[]byte{}, []byte{}},
		{MustDecodeHexBytes("0x00ffffffffffff"), MustDecodeHexBytes("0x00ffffffffffff")},
	}

	for i, test := range tests {
		got := TrimTrailingZeroBytes(test.input)
		if !bytes.Equal(got, test.expected) {
			t.Errorf("test %d: got %x expected %x", i, got, test.expected)
		}
	}
}
