// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/cmd/silacli"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/node"
)

// ExecutionConfig represents the shared execution runtime configuration.
type ExecutionConfig = silacli.ExecutionConfig

// LoadBaseConfig loads the shared execution configuration.
var LoadBaseConfig = silacli.LoadBaseConfig

// ApplyNodeConfig applies node configuration defaults.
var ApplyNodeConfig = silacli.ApplyNodeConfig

// NewNodeOrFatal creates a node or exits on failure.
func NewNodeOrFatal(cfg *node.Config) *node.Node {
	stack, err := node.New(cfg)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	return stack
}

// Prepare prepares the shared runtime context.
var Prepare = silacli.Prepare
