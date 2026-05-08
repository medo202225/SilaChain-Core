// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"fmt"
	"os"

	"github.com/sila-org/sila/cmd/silacli"
	"github.com/urfave/cli/v2"
)

func main() {
	app := silacli.NewApp(silacli.SilaAppConfig)
	app.Action = func(ctx *cli.Context) error {
		fmt.Fprintf(os.Stdout, "%s [%s]\n", silacli.SilaAppConfig.Usage, silacli.SilaAppConfig.EnvPrefix)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
