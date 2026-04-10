package main

// CANONICAL OWNERSHIP: legacy non-canonical node entrypoint.
// Canonical network binaries are cmd/sila-execution, cmd/sila-consensus, and cmd/sila-validator.

import (
	"log"
)

func main() {
	log.Fatal("cmd/sila-node is legacy and not a canonical mainnet path; use cmd/sila-execution")
}
