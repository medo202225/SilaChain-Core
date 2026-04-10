package config

type ExecutionNodeConfig struct {
	ListenAddress       string `json:"listen_address"`
	DataDir             string `json:"data_dir"`
	MinValidatorStake   uint64 `json:"min_validator_stake"`
	PeersPath           string `json:"peers_path"`
	EngineListenAddress string `json:"engine_listen_address"`
	EngineJWTSecretPath string `json:"engine_jwt_secret_path"`
}

type ValidatorClientConfig struct {
	ListenAddress            string `json:"listen_address"`
	VotingPublicKey          string `json:"voting_public_key"`
	VotingKeystorePath       string `json:"voting_keystore_path"`
	VotingSecretPath         string `json:"voting_secret_path"`
	WithdrawalPublicKey      string `json:"withdrawal_public_key"`
	WithdrawalKeystorePath   string `json:"withdrawal_keystore_path"`
	WithdrawalSecretPath     string `json:"withdrawal_secret_path"`
	SlashingProtectionDBPath string `json:"slashing_protection_db_path"`
}

type ConsensusClientConfig struct {
	ListenAddress       string `json:"listen_address"`
	EngineEndpoint      string `json:"engine_endpoint"`
	EngineJWTSecretPath string `json:"engine_jwt_secret_path"`
	ValidatorEndpoint   string `json:"validator_endpoint"`
	SlotsPerEpoch       uint64 `json:"slots_per_epoch"`
	SlotDurationSeconds uint64 `json:"slot_duration_seconds"`
}
