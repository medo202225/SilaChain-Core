package config

func (c *ExecutionNodeConfig) Normalize() {
	if c.ListenAddress == "" {
		c.ListenAddress = "0.0.0.0:8090"
	}
	if c.DataDir == "" {
		c.DataDir = "runtime/execution/node1"
	}
	if c.MinValidatorStake == 0 {
		c.MinValidatorStake = 1
	}
	if c.PeersPath == "" {
		c.PeersPath = "config/networks/mainnet/public/peers.json"
	}
	if c.EngineListenAddress == "" {
		c.EngineListenAddress = "127.0.0.1:8551"
	}
	if c.EngineJWTSecretPath == "" {
		c.EngineJWTSecretPath = "runtime/execution/node1/engine.jwt"
	}
}

func (c *ValidatorClientConfig) Normalize() {
	if c.ListenAddress == "" {
		c.ListenAddress = "127.0.0.1:5062"
	}
	if c.VotingKeystorePath == "" {
		c.VotingKeystorePath = "runtime/validators/node1/keystores/voting-keystore.json"
	}
	if c.VotingSecretPath == "" {
		c.VotingSecretPath = "runtime/validators/node1/secrets/voting-keystore.pass"
	}
	if c.WithdrawalKeystorePath == "" {
		c.WithdrawalKeystorePath = "runtime/validators/node1/keystores/withdrawal-keystore.json"
	}
	if c.WithdrawalSecretPath == "" {
		c.WithdrawalSecretPath = "runtime/validators/node1/secrets/withdrawal-keystore.pass"
	}
	if c.SlashingProtectionDBPath == "" {
		c.SlashingProtectionDBPath = "runtime/validators/node1/slashing-protection/slashing.sqlite"
	}
}

func (c *ConsensusClientConfig) Normalize() {
	if c.ListenAddress == "" {
		c.ListenAddress = "127.0.0.1:5052"
	}
	if c.EngineEndpoint == "" {
		c.EngineEndpoint = "http://127.0.0.1:8551"
	}
	if c.EngineJWTSecretPath == "" {
		c.EngineJWTSecretPath = "runtime/execution/node1/engine.jwt"
	}
	if c.ValidatorEndpoint == "" {
		c.ValidatorEndpoint = "http://127.0.0.1:5062"
	}
	if c.SlotsPerEpoch == 0 {
		c.SlotsPerEpoch = 32
	}
	if c.SlotDurationSeconds == 0 {
		c.SlotDurationSeconds = 12
	}
}
