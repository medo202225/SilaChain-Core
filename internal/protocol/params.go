package protocol

import "silachain/pkg/types"

type Params struct {
	ChainID         types.ChainID `json:"chain_id"`
	Decimals        uint8         `json:"decimals"`
	MinFee          types.Amount  `json:"min_fee"`
	BlockGasLimit   types.Gas     `json:"block_gas_limit"`
	MaxTxsPerBlock  uint64        `json:"max_txs_per_block"`
	BlockTimeSec    uint64        `json:"block_time_sec"`
	EpochLength     uint64        `json:"epoch_length"`
	UnbondingBlocks uint64        `json:"unbonding_blocks"`
}

func DefaultMainnetParams() Params {
	return Params{
		ChainID:         MainnetChainID,
		Decimals:        18,
		MinFee:          1,
		BlockGasLimit:   30000000,
		MaxTxsPerBlock:  5000,
		BlockTimeSec:    5,
		EpochLength:     10000,
		UnbondingBlocks: 120000,
	}
}

func DefaultTestnetParams() Params {
	return Params{
		ChainID:         TestnetChainID,
		Decimals:        18,
		MinFee:          1,
		BlockGasLimit:   30000000,
		MaxTxsPerBlock:  5000,
		BlockTimeSec:    5,
		EpochLength:     2000,
		UnbondingBlocks: 20000,
	}
}

func DefaultDevnetParams() Params {
	return Params{
		ChainID:         DevnetChainID,
		Decimals:        18,
		MinFee:          1,
		BlockGasLimit:   10000000,
		MaxTxsPerBlock:  1000,
		BlockTimeSec:    2,
		EpochLength:     100,
		UnbondingBlocks: 1000,
	}
}
