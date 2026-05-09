// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import "github.com/sila-org/sila/cmd/silacli"

// NodeLifecycle represents a node lifecycle boundary.
type NodeLifecycle = silacli.NodeLifecycle

// RuntimeHooks represents execution runtime hooks.
type RuntimeHooks = silacli.RuntimeHooks

// RunRuntime runs the shared execution runtime.
var RunRuntime = silacli.RunRuntime

// Prepare prepares the shared execution runtime context.
var Prepare = silacli.Prepare
