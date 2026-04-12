// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package fdlimit

import "testing"

func TestSilaFileDescriptorLimits(t *testing.T) {
	target := 4096

	hardLimit, err := Maximum()
	if err != nil {
		t.Fatal(err)
	}
	if hardLimit < target {
		t.Skipf("system limit is less than desired test target: %d < %d", hardLimit, target)
	}

	if limit, err := Current(); err != nil || limit <= 0 {
		t.Fatalf("failed to retrieve file descriptor limit (%d): %v", limit, err)
	}
	if _, err := Raise(uint64(target)); err != nil {
		t.Fatalf("failed to raise file allowance: %v", err)
	}
	if limit, err := Current(); err != nil || limit < target {
		t.Fatalf("failed to retrieve raised descriptor limit (have %v, want %v): %v", limit, target, err)
	}
}
