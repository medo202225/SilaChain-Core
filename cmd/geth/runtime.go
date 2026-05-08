// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"fmt"

	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/log"
	"github.com/urfave/cli/v2"
)

// prepare manipulates memory cache allowance and setups metric system.
// This function should be called before launching devp2p stack.
func prepare(ctx *cli.Context) {
	// If we're running a known preset, log it for convenience.
	switch {
	case ctx.IsSet(utils.SepoliaFlag.Name):
		log.Info("Starting Sila on Sepolia testnet...")

	case ctx.IsSet(utils.HoleskyFlag.Name):
		log.Info("Starting Sila on Holesky testnet...")

	case ctx.IsSet(utils.HoodiFlag.Name):
		log.Info("Starting Sila on Hoodi testnet...")

	case !ctx.IsSet(utils.NetworkIdFlag.Name):
		log.Info("Starting Sila on Sila mainnet...")
	}
}

func runSilaNode(ctx *cli.Context, isConsole bool) error {
	return runSilaRuntime(ctx, isConsole)
}

func runSilaRuntime(ctx *cli.Context, isConsole bool) error {
	if args := ctx.Args().Slice(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	prepare(ctx)
	stack := makeFullNode(ctx)
	defer stack.Close()

	startNode(ctx, stack, isConsole)
	stack.Wait()
	return nil
}
