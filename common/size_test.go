// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import "testing"

func TestSilaStorageSizeString(t *testing.T) {
	tests := []struct {
		size SilaStorageSize
		want string
	}{
		{0, "0.00 B"},
		{1023, "1023.00 B"},
		{1024, "1.00 KiB"},
		{1048576, "1.00 MiB"},
		{1073741824, "1.00 GiB"},
		{1099511627776, "1.00 TiB"},
	}

	for _, tt := range tests {
		got := tt.size.String()
		if got != tt.want {
			t.Errorf("String() = %s, want %s", got, tt.want)
		}
	}
}

func TestSilaStorageSizeTerminalString(t *testing.T) {
	tests := []struct {
		size SilaStorageSize
		want string
	}{
		{0, "0.00B"},
		{1023, "1023.00B"},
		{1024, "1.00KiB"},
		{1048576, "1.00MiB"},
		{1073741824, "1.00GiB"},
		{1099511627776, "1.00TiB"},
	}

	for _, tt := range tests {
		got := tt.size.TerminalString()
		if got != tt.want {
			t.Errorf("TerminalString() = %s, want %s", got, tt.want)
		}
	}
}
