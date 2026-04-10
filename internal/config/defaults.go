package config

func DefaultExecutionNodeConfig() *ExecutionNodeConfig {
	return &ExecutionNodeConfig{
		ListenAddress:       "0.0.0.0:8090",
		DataDir:             "runtime/execution/node1",
		MinValidatorStake:   1,
		PeersPath:           "config/networks/mainnet/public/peers.json",
		EngineListenAddress: "127.0.0.1:8551",
		EngineJWTSecretPath: "runtime/execution/node1/engine.jwt",
	}
}

func DefaultValidatorClientConfig() *ValidatorClientConfig {
	return &ValidatorClientConfig{
		ListenAddress:            "127.0.0.1:5062",
		VotingPublicKey:          "",
		VotingKeystorePath:       "runtime/validators/node1/keystores/voting-keystore.json",
		VotingSecretPath:         "runtime/validators/node1/secrets/voting-keystore.pass",
		WithdrawalPublicKey:      "",
		WithdrawalKeystorePath:   "runtime/validators/node1/keystores/withdrawal-keystore.json",
		WithdrawalSecretPath:     "runtime/validators/node1/secrets/withdrawal-keystore.pass",
		SlashingProtectionDBPath: "runtime/validators/node1/slashing-protection/slashing.sqlite",
	}
}

func DefaultConsensusClientConfig() *ConsensusClientConfig {
	return &ConsensusClientConfig{
		ListenAddress:       "127.0.0.1:5052",
		EngineEndpoint:      "http://127.0.0.1:8551",
		EngineJWTSecretPath: "runtime/execution/node1/engine.jwt",
		ValidatorEndpoint:   "http://127.0.0.1:5062",
		SlotsPerEpoch:       32,
		SlotDurationSeconds: 12,
	}
}
