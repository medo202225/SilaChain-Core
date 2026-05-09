// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.
package main

import (
	"github.com/sila-org/sila/internal/silaexec"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

func prepare(ctx *cli.Context) {
	silaexec.Prepare(ctx)
}

func runSilaNode(ctx *cli.Context, isConsole bool) error {
	return silaexec.RunRuntime(ctx, silaRuntimeHooks(), isConsole)
}

func silaRuntimeHooks() silaexec.RuntimeHooks {
	return silaexec.RuntimeHooks{
		Prepare:      prepare,
		MakeFullNode: func(ctx *cli.Context) silaexec.NodeLifecycle { return makeFullNode(ctx) },
		StartNode: func(ctx *cli.Context, stack silaexec.NodeLifecycle, isConsole bool) {
			startNode(ctx, stack.(*node.Node), isConsole)
		},
	}
}
