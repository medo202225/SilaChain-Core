package main

import (
	"encoding/json"
	"fmt"
	"os"

	validatorclient "silachain/internal/validatorclient"
)

func main() {
	loaded, err := validatorclient.LoadVotingKeystore(
		"runtime/validators/node1/keystores/voting-keystore.json",
		"runtime/validators/node1/secrets/voting-keystore.pass",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	message := []byte("sila-validator-test-message")

	sig, err := validatorclient.SignMessage(loaded, message)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	ok, err := validatorclient.VerifySignatureHex(loaded.PublicHex, message, sig.SignatureHex)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	out := map[string]interface{}{
		"path":         loaded.Path,
		"public_key":   loaded.PublicHex,
		"signature":    sig.SignatureHex,
		"verified":     ok,
		"message_text": string(message),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
