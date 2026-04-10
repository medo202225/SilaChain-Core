package protocol

import "silachain/pkg/types"

const (
	MainnetChainID types.ChainID = 1001
	TestnetChainID types.ChainID = 1002
	DevnetChainID  types.ChainID = 1003
)

func IsSupportedChainID(id types.ChainID) bool {
	switch id {
	case MainnetChainID, TestnetChainID, DevnetChainID:
		return true
	default:
		return false
	}
}
