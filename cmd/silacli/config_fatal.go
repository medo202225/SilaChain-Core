// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

import "github.com/sila-org/sila/cmd/utils"

func LoadConfigOrFatal(file string, cfg any) {
	if file == "" {
		return
	}
	if err := LoadConfig(file, cfg); err != nil {
		utils.Fatalf("%v", err)
	}
}
