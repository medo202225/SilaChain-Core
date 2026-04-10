package chain

import (
	"errors"
	block "silachain/internal/block"

	coretypes "silachain/internal/core/types"
)

var ErrCorruptChainData = errors.New("corrupt chain data")

func validateLoadedBlocks(blocks []*coretypes.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	for i := 0; i < len(blocks); i++ {
		if blocks[i] == nil {
			return ErrCorruptChainData
		}
		if err := block.Validate(blocks[i]); err != nil {
			return ErrCorruptChainData
		}

		if i == 0 {
			continue
		}

		prev := blocks[i-1]
		curr := blocks[i]

		if curr.Header.Height != prev.Header.Height+1 {
			return ErrCorruptChainData
		}
		if curr.Header.ParentHash != prev.Header.Hash {
			return ErrCorruptChainData
		}
	}

	return nil
}
