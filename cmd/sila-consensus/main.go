package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"silachain/internal/config"
	consensus "silachain/internal/consensus"
	validatorclient "silachain/internal/validatorclient"
)

func main() {
	consensusCfg, err := config.LoadConsensusClientConfig("config/networks/mainnet/consensus/consensus-client.json")
	if err != nil {
		log.Fatal(err)
	}

	validatorCfg, err := config.LoadValidatorClientConfig("config/networks/mainnet/validator/validator-client.json")
	if err != nil {
		log.Fatal(err)
	}

	validatorSvc, err := validatorclient.NewValidatorService(validatorCfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = validatorSvc.Close() }()

	validatorErrCh := make(chan error, 1)
	go func() {
		log.Printf("consensus launcher validator startup listen=http://%s", validatorCfg.ListenAddress)
		validatorErrCh <- validatorSvc.Start()
	}()

	validatorClient, err := consensus.NewValidatorClient("http://" + validatorCfg.ListenAddress)
	if err != nil {
		log.Fatal(err)
	}

	var health map[string]any
	for attempt := 0; attempt < 10; attempt++ {
		health, err = validatorClient.Health(context.Background())
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("consensus validator health ok: %v", health)

	status, err := validatorClient.Status(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("consensus validator status ok: %v", status)

	engineClient, err := consensus.NewEngineClient(
		consensusCfg.EngineEndpoint+"/engine",
		consensusCfg.EngineJWTSecretPath,
	)
	if err != nil {
		log.Fatal(err)
	}

	caps, err := engineClient.ExchangeCapabilitiesParsed()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("consensus engine exchangeCapabilities ok: capabilities=%v", caps.Capabilities)

	identity, err := engineClient.IdentityParsed()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("consensus engine identity ok: name=%s version=%s chain=%s", identity.Name, identity.Version, identity.Chain)

	engineNotifier, err := consensus.NewEngineForkchoiceNotifier(engineClient)
	if err != nil {
		log.Fatal(err)
	}

	state, err := consensus.LoadBeaconStateFromValidatorsFile("config/networks/mainnet/public/validators.json")
	if err != nil {
		log.Fatal(err)
	}

	dutyProvider := consensus.NewStaticDutyProvider(state, consensusCfg.SlotsPerEpoch)

	proposalExecutor, err := consensus.NewEngineProposalExecutor(engineClient, state)
	if err != nil {
		log.Fatal(err)
	}

	s := consensus.NewScheduler(
		validatorClient.VotingPublicKey(),
		validatorClient,
		dutyProvider,
		proposalExecutor,
		engineNotifier,
		time.Duration(consensusCfg.SlotDurationSeconds)*time.Second,
		consensusCfg.SlotsPerEpoch,
		state,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Start(ctx)

	log.Printf("consensus scheduler startup validator_endpoint=http://%s engine_endpoint=%s", validatorCfg.ListenAddress, consensusCfg.EngineEndpoint)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("consensus launcher shutdown signal received: %s", sig.String())
		cancel()
	case err := <-validatorErrCh:
		if err != nil {
			log.Fatalf("validator service stopped: %v", err)
		}
	}
}
