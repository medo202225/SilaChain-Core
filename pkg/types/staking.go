package types

type Delegation struct {
	Delegator Address `json:"delegator"`
	Validator Address `json:"validator"`
	Amount    Amount  `json:"amount"`
}

type Unbonding struct {
	Delegator      Address   `json:"delegator"`
	Validator      Address   `json:"validator"`
	Amount         Amount    `json:"amount"`
	StartHeight    Height    `json:"start_height"`
	ReleaseHeight  Height    `json:"release_height"`
	StartTimestamp Timestamp `json:"start_timestamp"`
}
