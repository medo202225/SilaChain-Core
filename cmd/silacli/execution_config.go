// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

import (
	"github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/metrics"
	"github.com/sila-org/sila/node"
)

type EthstatsConfig struct {
	URL string `toml:",omitempty"`
}

type ExecutionConfig struct {
	Eth      ethconfig.Config
	Node     node.Config
	Ethstats EthstatsConfig
	Metrics  metrics.Config
}
