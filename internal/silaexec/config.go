// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import "github.com/sila-org/sila/cmd/silacli"

// ConfigTOMLSettings exposes the shared TOML config settings boundary.
var ConfigTOMLSettings = silacli.ConfigTOMLSettings

// ExecutionConfig represents the shared execution runtime configuration
// boundary used by Sila execution clients.
type ExecutionConfig = silacli.ExecutionConfig
