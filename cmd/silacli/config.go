// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

type AppConfig struct {
	Usage            string
	EnvPrefix        string
	ClientIdentifier string
}

var GethAppConfig = AppConfig{
	Usage:            "the SilaChain command line interface",
	EnvPrefix:        "GETH",
	ClientIdentifier: "sila",
}

var SilaAppConfig = AppConfig{
	Usage:            "the SilaChain command line interface",
	EnvPrefix:        "SILA",
	ClientIdentifier: "sila",
}
