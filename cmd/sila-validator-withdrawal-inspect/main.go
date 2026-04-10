package main

import (
	"encoding/json"
	"fmt"
	"os"

	validatorclient "silachain/internal/validatorclient"
)

func main() {
	loaded, err := validatorclient.LoadVotingKeystore(
		"runtime/validators/node1/keystores/withdrawal-keystore.json",
		"runtime/validators/node1/secrets/withdrawal-keystore.pass",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	out := map[string]string{
		"path":       loaded.Path,
		"public_key": loaded.PublicHex,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
