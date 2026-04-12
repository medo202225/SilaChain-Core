// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package fdlimit

import "fmt"

// silaHardLimit is the practical per-process file descriptor cap used on Windows.
const silaHardLimit = 16384

// Raise tries to raise the file descriptor allowance up to the requested maximum.
func Raise(max uint64) (uint64, error) {
	if max > silaHardLimit {
		return silaHardLimit, fmt.Errorf("file descriptor limit (%d) reached", silaHardLimit)
	}
	return max, nil
}

// Current returns the current file descriptor allowance for this process.
func Current() (int, error) {
	return silaHardLimit, nil
}

// Maximum returns the maximum file descriptor allowance this process can request.
func Maximum() (int, error) {
	return Current()
}
