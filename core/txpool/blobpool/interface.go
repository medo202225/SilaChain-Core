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

package blobpool

import (
	"silachain/common"
	"silachain/core/state"
	"silachain/core/types"
	"silachain/params"
)

// BlockChain defines the minimal set of methods needed to back a SILA blob pool with
// a chain. Exists to allow mocking the live chain out of tests.
type BlockChain interface {
	// Config retrieves the chain's fork configuration.
	Config() *params.ChainConfig

	// CurrentBlock returns the current head of the chain.
	CurrentBlock() *types.Header

	// CurrentFinalBlock returns the current block below which blobs should not
	// be maintained anymore for reorg purposes on SILA.
	CurrentFinalBlock() *types.Header

	// GetBlock retrieves a specific block, used during pool resets.
	GetBlock(hash common.Hash, number uint64) *types.Block

	// StateAt returns a state database for a given root hash (generally the head).
	StateAt(root common.Hash) (*state.StateDB, error)
}
