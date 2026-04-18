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

package forkid

import (
	"bytes"
	"hash/crc32"
	"math"
	"math/big"
	"testing"

	"silachain/common"
	"silachain/core"
	"silachain/core/types"
	"silachain/params"
	"silachain/rlp"
)

// TestCreation tests that different genesis and fork rule combinations result in
// the correct fork ID.
func TestCreation(t *testing.T) {
	type testcase struct {
		head uint64
		time uint64
		want ID
	}
	tests := []struct {
		config  *params.ChainConfig
		genesis *types.Block
		cases   []testcase
	}{
		// Mainnet test cases
		{
			params.MainnetChainConfig,
			core.DefaultGenesisBlock().ToBlock(),
			[]testcase{
				{0, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{1149999, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},
				{1150000, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},
				{1919999, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},
				{1920000, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},
				{2462999, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},
				{2463000, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},
				{2674999, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},
				{2675000, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},
				{4369999, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},
				{4370000, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},
				{7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},
				{7280000, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},
				{9068999, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},
				{9069000, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},
				{9199999, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},
				{9200000, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},
				{12243999, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},
				{12244000, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}},
				{12964999, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}},
				{12965000, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}},
				{13772999, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}},
				{13773000, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}},
				{15049999, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}},
				{15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}},
				{20000000, 1681338454, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}},
				{20000000, 1681338455, ID{Hash: checksumToBytes(0xdce96c2d), Next: 1710338135}},
				{30000000, 1710338134, ID{Hash: checksumToBytes(0xdce96c2d), Next: 1710338135}},
				{30000000, 1710338135, ID{Hash: checksumToBytes(0x9f3d2254), Next: 1746612311}},
				{30000000, 1746022486, ID{Hash: checksumToBytes(0x9f3d2254), Next: 1746612311}},
				{30000000, 1746612311, ID{Hash: checksumToBytes(0xc376cf8b), Next: 1764798551}},
				{30000000, 1764798550, ID{Hash: checksumToBytes(0xc376cf8b), Next: 1764798551}},
				{30000000, 1764798551, ID{Hash: checksumToBytes(0x5167e2a6), Next: 1765290071}},
				{30000000, 1765290070, ID{Hash: checksumToBytes(0x5167e2a6), Next: 1765290071}},
				{30000000, 1765290071, ID{Hash: checksumToBytes(0xcba2a1c0), Next: 1767747671}},
				{30000000, 1767747670, ID{Hash: checksumToBytes(0xcba2a1c0), Next: 1767747671}},
				{30000000, 1767747671, ID{Hash: checksumToBytes(0x07c9462e), Next: 0}},
				{50000000, 2000000000, ID{Hash: checksumToBytes(0x07c9462e), Next: 0}},
			},
		},
	}
	for i, tt := range tests {
		for j, ttt := range tt.cases {
			if have := NewID(tt.config, tt.genesis, ttt.head, ttt.time); have != ttt.want {
				t.Errorf("test %d, case %d: fork ID mismatch: have %x, want %x", i, j, have, ttt.want)
			}
		}
	}
}

// TestValidation tests that a local peer correctly validates and accepts a remote
// fork ID.
func TestValidation(t *testing.T) {
	legacyConfig := *params.MainnetChainConfig
	legacyConfig.ShanghaiTime = nil
	legacyConfig.CancunTime = nil
	legacyConfig.PragueTime = nil
	legacyConfig.OsakaTime = nil
	legacyConfig.BPO1Time = nil
	legacyConfig.BPO2Time = nil

	tests := []struct {
		config *params.ChainConfig
		head   uint64
		time   uint64
		id     ID
		err    error
	}{
		// Block based tests
		{&legacyConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},
		{&legacyConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: math.MaxUint64}, nil},
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: math.MaxUint64}, nil},
		{&legacyConfig, 7280000, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}, nil},
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}, nil},
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 0}, nil},
		{&legacyConfig, 4369999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, nil},
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 0}, ErrRemoteStale},
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0x5cddc0e1), Next: 0}, ErrLocalIncompatibleOrStale},
		{&legacyConfig, 7987396, 0, ID{Hash: checksumToBytes(0xafec6b27), Next: 0}, ErrLocalIncompatibleOrStale},
		{&legacyConfig, 88888888, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 88888888}, ErrLocalIncompatibleOrStale},
		{&legacyConfig, 7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7279999}, ErrLocalIncompatibleOrStale},

		// Block to timestamp transition tests
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}, nil},
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: math.MaxUint64}, nil},
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}, nil},
		{params.MainnetChainConfig, 20123456, 1681338456, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}, nil},
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}, nil},
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}, nil},
		{params.MainnetChainConfig, 13773000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, nil},
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 0}, ErrRemoteStale},
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(checksumUpdate(0xf0afd0e3, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},
		{params.MainnetChainConfig, 15050000, 0, ID{Hash: checksumToBytes(checksumUpdate(0xdce96c2d, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},
		{params.MainnetChainConfig, 888888888, 1660000000, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1660000000}, ErrLocalIncompatibleOrStale},
		{params.MainnetChainConfig, 19999999, 1667999999, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1667999999}, ErrLocalIncompatibleOrStale},

		// Timestamp based tests
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}, nil},
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0xdce96c2d), Next: math.MaxUint64}, nil},
		{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}, nil},
		{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0xdce96c2d), Next: 1710338135}, nil},
		{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(0xdce96c2d), Next: math.MaxUint64}, nil},
		{params.MainnetChainConfig, 21000000, 1710338135, ID{Hash: checksumToBytes(0xdce96c2d), Next: 1710338135}, nil},
		{params.MainnetChainConfig, 21123456, 1710338136, ID{Hash: checksumToBytes(0xdce96c2d), Next: 1710338135}, nil},
		{params.MainnetChainConfig, 0, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 1710338135}, nil},
		{params.MainnetChainConfig, 21000000, 1700000000, ID{Hash: checksumToBytes(0x9f3d2254), Next: 0}, nil},
		{params.MainnetChainConfig, 21000000, 1678000000, ID{Hash: checksumToBytes(0xc376cf8b), Next: 0}, nil},
		{params.MainnetChainConfig, 21000000, 1710338135, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}, ErrRemoteStale},
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(checksumUpdate(0xdce96c2d, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},
		{params.MainnetChainConfig, 20000000, 1668000000, ID{Hash: checksumToBytes(checksumUpdate(0x9f3d2254, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0x12345678), Next: 0}, ErrLocalIncompatibleOrStale},
		{params.MainnetChainConfig, 88888888, 8888888888, ID{Hash: checksumToBytes(0x07c9462e), Next: 8888888888}, ErrLocalIncompatibleOrStale},
		{params.MainnetChainConfig, 20999999, 1699999999, ID{Hash: checksumToBytes(0x71147644), Next: 1700000000}, ErrLocalIncompatibleOrStale},
	}
	genesis := core.DefaultGenesisBlock().ToBlock()
	for i, tt := range tests {
		filter := newFilter(tt.config, genesis, func() (uint64, uint64) { return tt.head, tt.time })
		if err := filter(tt.id); err != tt.err {
			t.Errorf("test %d: validation error mismatch: have %v, want %v", i, err, tt.err)
		}
	}
}

// Tests that IDs are properly RLP encoded.
func TestEncoding(t *testing.T) {
	tests := []struct {
		id   ID
		want []byte
	}{
		{ID{Hash: checksumToBytes(0), Next: 0}, common.Hex2Bytes("c6840000000080")},
		{ID{Hash: checksumToBytes(0xdeadbeef), Next: 0xBADDCAFE}, common.Hex2Bytes("ca84deadbeef84baddcafe")},
		{ID{Hash: checksumToBytes(math.MaxUint32), Next: math.MaxUint64}, common.Hex2Bytes("ce84ffffffff88ffffffffffffffff")},
	}
	for i, tt := range tests {
		have, err := rlp.EncodeToBytes(tt.id)
		if err != nil {
			t.Errorf("test %d: failed to encode forkid: %v", i, err)
			continue
		}
		if !bytes.Equal(have, tt.want) {
			t.Errorf("test %d: RLP mismatch: have %x, want %x", i, have, tt.want)
		}
	}
}

// Tests that time-based forks which are active at genesis are not included in
// forkid hash.
func TestTimeBasedForkInGenesis(t *testing.T) {
	var (
		time       = uint64(1690475657)
		genesis    = types.NewBlockWithHeader(&types.Header{Time: time})
		forkidHash = checksumToBytes(crc32.ChecksumIEEE(genesis.Hash().Bytes()))
		config     = func(shanghai, cancun uint64) *params.ChainConfig {
			return &params.ChainConfig{
				ChainID:                 big.NewInt(1337),
				HomesteadBlock:          big.NewInt(0),
				DAOForkBlock:            nil,
				DAOForkSupport:          true,
				EIP150Block:             big.NewInt(0),
				EIP155Block:             big.NewInt(0),
				EIP158Block:             big.NewInt(0),
				ByzantiumBlock:          big.NewInt(0),
				ConstantinopleBlock:     big.NewInt(0),
				PetersburgBlock:         big.NewInt(0),
				IstanbulBlock:           big.NewInt(0),
				MuirGlacierBlock:        big.NewInt(0),
				BerlinBlock:             big.NewInt(0),
				LondonBlock:             big.NewInt(0),
				TerminalTotalDifficulty: big.NewInt(0),
				MergeNetsplitBlock:      big.NewInt(0),
				ShanghaiTime:            &shanghai,
				CancunTime:              &cancun,
				Ethash:                  new(params.EthashConfig),
			}
		}
	)
	tests := []struct {
		config *params.ChainConfig
		want   ID
	}{
		{config(time-1, time+1), ID{Hash: forkidHash, Next: time + 1}},
		{config(time, time+1), ID{Hash: forkidHash, Next: time + 1}},
		{config(time+1, time+2), ID{Hash: forkidHash, Next: time + 1}},
	}
	for _, tt := range tests {
		if have := NewID(tt.config, genesis, 0, time); have != tt.want {
			t.Fatalf("incorrect forkid hash: have %x, want %x", have, tt.want)
		}
	}
}
