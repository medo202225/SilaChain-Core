package protocol

type FeatureFlags struct {
	StakingEnabled      bool `json:"staking_enabled"`
	DelegationEnabled   bool `json:"delegation_enabled"`
	SlashingEnabled     bool `json:"slashing_enabled"`
	WebSocketRPCEnabled bool `json:"websocket_rpc_enabled"`
}

func DefaultFeatureFlags() FeatureFlags {
	return FeatureFlags{
		StakingEnabled:      true,
		DelegationEnabled:   true,
		SlashingEnabled:     true,
		WebSocketRPCEnabled: true,
	}
}
