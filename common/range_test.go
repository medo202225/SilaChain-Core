// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package common

import (
	"slices"
	"testing"
)

func TestSilaRangeIter(t *testing.T) {
	r := NewSilaRange[uint32](1, 7)
	values := slices.Collect(r.Iter())
	if !slices.Equal(values, []uint32{1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("wrong iter values: %v", values)
	}

	empty := NewSilaRange[uint32](1, 0)
	values = slices.Collect(empty.Iter())
	if !slices.Equal(values, []uint32{}) {
		t.Fatalf("wrong iter values: %v", values)
	}
}
