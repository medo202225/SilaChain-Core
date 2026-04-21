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

package params

import (
"bytes"
"testing"
)

func TestChainConfig_LoadForks(t *testing.T) {
const config = `
GENESIS_FORK_VERSION: 0x00000000

ALTAIR_FORK_VERSION: 0x00000001
ALTAIR_FORK_EPOCH: 1

EIP7928_FORK_VERSION: 0xb0000038
EIP7928_FORK_EPOCH: 18446744073709551615

EIP7XXX_FORK_VERSION: 
EIP7XXX_FORK_EPOCH: 

BLOB_SCHEDULE: []
`
c := &ChainConfig{}
err := c.LoadForks([]byte(config))
if err != nil {
t.Fatal(err)
}

for _, fork := range c.Forks {
if fork.Name == "GENESIS" && (fork.Epoch != 0) {
t.Errorf("unexpected genesis fork epoch %d", fork.Epoch)
}
if fork.Name == "ALTAIR" && (fork.Epoch != 1 || !bytes.Equal(fork.Version, []byte{0, 0, 0, 1})) {
t.Errorf("unexpected altair fork epoch %d version %x", fork.Epoch, fork.Version)
}
}
}
