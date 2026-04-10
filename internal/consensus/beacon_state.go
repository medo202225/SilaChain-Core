package consensus

// CANONICAL OWNERSHIP: root consensus package is limited to beacon state, scheduling, transition coordination, and validator coordination.
// Engine, engine API, forkchoice, runtime, txpool, executionstate, and p2p ownership live in their dedicated subpackages.

const farFutureEpoch uint64 = ^uint64(0)

type Checkpoint struct {
	Epoch uint64 `json:"epoch"`
	Root  string `json:"root"`
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root"`
	DepositCount uint64 `json:"deposit_count"`
	BlockHash    string `json:"block_hash"`
}

type ValidatorRecord struct {
	PublicKey                  string `json:"public_key"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           uint64 `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch uint64 `json:"activation_eligibility_epoch"`
	ActivationEpoch            uint64 `json:"activation_epoch"`
	ExitEpoch                  uint64 `json:"exit_epoch"`
	WithdrawableEpoch          uint64 `json:"withdrawable_epoch"`
}

type DepositData struct {
	PublicKey             string `json:"public_key"`
	WithdrawalCredentials string `json:"withdrawal_credentials"`
	Amount                uint64 `json:"amount"`
	Signature             string `json:"signature"`
}

type Deposit struct {
	Index uint64      `json:"index"`
	Root  string      `json:"root"`
	Proof [][]string  `json:"proof"`
	Data  DepositData `json:"data"`
}

type BeaconStateV1 struct {
	Slot                       uint64            `json:"slot"`
	Epoch                      uint64            `json:"epoch"`
	Validators                 []ValidatorRecord `json:"validators"`
	Balances                   []uint64          `json:"balances"`
	Eth1DepositIndex           uint64            `json:"eth1_deposit_index"`
	DepositRoot                string            `json:"deposit_root"`
	Eth1Data                   Eth1Data          `json:"eth1_data"`
	CurrentJustifiedCheckpoint Checkpoint        `json:"current_justified_checkpoint"`
	FinalizedCheckpoint        Checkpoint        `json:"finalized_checkpoint"`
	HeadBlockRoot              string            `json:"head_block_root"`
	SafeBlockRoot              string            `json:"safe_block_root"`
	LatestPayloadID            string            `json:"latest_payload_id"`
}

func NewBeaconStateV1(validators []ValidatorRecord) *BeaconStateV1 {
	balances := make([]uint64, 0, len(validators))
	normalized := make([]ValidatorRecord, 0, len(validators))

	for _, v := range validators {
		if v.ExitEpoch == 0 {
			v.ExitEpoch = farFutureEpoch
		}
		if v.WithdrawableEpoch == 0 {
			v.WithdrawableEpoch = farFutureEpoch
		}
		normalized = append(normalized, v)
		balances = append(balances, v.EffectiveBalance)
	}

	return &BeaconStateV1{
		Validators:       normalized,
		Balances:         balances,
		Eth1DepositIndex: 0,
		DepositRoot:      "",
		Eth1Data: Eth1Data{
			DepositRoot:  "",
			DepositCount: 0,
			BlockHash:    "",
		},
		CurrentJustifiedCheckpoint: Checkpoint{
			Epoch: 0,
			Root:  "",
		},
		FinalizedCheckpoint: Checkpoint{
			Epoch: 0,
			Root:  "",
		},
	}
}
