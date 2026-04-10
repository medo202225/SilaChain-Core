package main

import (
	"fmt"
	"log"
	"os"

	"silachain/internal/config"
	validatorclient "silachain/internal/validatorclient"
)

func main() {
	configPath := "config/networks/mainnet/validator/validator-client.json"
	if len(os.Args) > 2 && os.Args[1] == "--config" {
		configPath = os.Args[2]
	}

	cfg, err := config.LoadValidatorClientConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	svc, err := validatorclient.NewValidatorService(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = svc.Close() }()

	log.Printf("validator startup listen=http://%s", cfg.ListenAddress)
	log.Printf("validator startup voting_public_key=%s", cfg.VotingPublicKey)
	log.Printf("validator startup slashing_db=%s", cfg.SlashingProtectionDBPath)

	if err := svc.Start(); err != nil {
		log.Fatal(fmt.Errorf("validator service stopped: %w", err))
	}
}
