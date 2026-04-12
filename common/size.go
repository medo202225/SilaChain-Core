// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"fmt"
)

// SilaStorageSize is a wrapper around a float value that supports user friendly formatting.
type SilaStorageSize float64

// String returns a human-readable storage size with a space before the unit.
func (s SilaStorageSize) String() string {
	if s >= 1099511627776 {
		return fmt.Sprintf("%.2f TiB", s/1099511627776)
	} else if s >= 1073741824 {
		return fmt.Sprintf("%.2f GiB", s/1073741824)
	} else if s >= 1048576 {
		return fmt.Sprintf("%.2f MiB", s/1048576)
	} else if s >= 1024 {
		return fmt.Sprintf("%.2f KiB", s/1024)
	} else {
		return fmt.Sprintf("%.2f B", s)
	}
}

// TerminalString returns a compact storage size string without a space before the unit.
func (s SilaStorageSize) TerminalString() string {
	if s >= 1099511627776 {
		return fmt.Sprintf("%.2fTiB", s/1099511627776)
	} else if s >= 1073741824 {
		return fmt.Sprintf("%.2fGiB", s/1073741824)
	} else if s >= 1048576 {
		return fmt.Sprintf("%.2fMiB", s/1048576)
	} else if s >= 1024 {
		return fmt.Sprintf("%.2fKiB", s/1024)
	} else {
		return fmt.Sprintf("%.2fB", s)
	}
}
