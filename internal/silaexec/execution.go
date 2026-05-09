// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/eth"
	ethconfig "github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/node"
)

// RegisterExecutionService registers the Sila execution service.
func RegisterExecutionService(stack *node.Node, cfg *ethconfig.Config) (*eth.EthAPIBackend, *eth.Ethereum) {
	return utils.RegisterEthService(stack, cfg)
}
