package consensus

func LoadBeaconStateFromValidatorsFile(path string) (*BeaconStateV1, error) {
	return LoadBeaconStateFromDepositSource(path, 32)
}
