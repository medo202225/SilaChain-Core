// Copyright 2026 The SILA Authors
// This file is part of the SILA library.
//
// The SILA library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The SILA library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the SILA library. If not, see <http://www.gnu.org/licenses/>.

package accounts

import (
"bytes"
"testing"

"github.com/sila-blockchain/sila-library/common/hexutil"
)

func TestTextHash(t *testing.T) {
t.Parallel()
hash := TextHash([]byte("Hello Joe"))
// Expected hash for SILA prefix: "\x19SILA Signed Message:\n9Hello Joe"
// This is the Keccak256 hash of the prefixed message
want := hexutil.MustDecode("0x16e1e7b424b0f39edfcfcfcffa5eefecdb079b24736d7d9d2dee687f7a8b72b9")
if !bytes.Equal(hash, want) {
t.Fatalf("wrong hash: %x", hash)
}
}
