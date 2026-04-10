package types

type Receipt struct {
	TxHash            Hash      `json:"tx_hash"`
	BlockHash         Hash      `json:"block_hash"`
	BlockHeight       Height    `json:"block_height"`
	TransactionIndex  uint64    `json:"transaction_index"`
	From              Address   `json:"from"`
	To                Address   `json:"to"`
	Value             Amount    `json:"value"`
	Fee               Amount    `json:"fee"`
	GasPrice          Amount    `json:"gas_price"`
	GasLimit          Gas       `json:"gas_limit"`
	GasUsed           Gas       `json:"gas_used"`
	CumulativeGasUsed Gas       `json:"cumulative_gas_used"`
	EffectiveFee      Amount    `json:"effective_fee"`
	Success           bool      `json:"success"`
	Error             string    `json:"error,omitempty"`
	Logs              []Event   `json:"logs,omitempty"`
	ReturnData        string    `json:"return_data,omitempty"`
	RevertData        string    `json:"revert_data,omitempty"`
	CreatedAddress    Address   `json:"created_address,omitempty"`
	Timestamp         Timestamp `json:"timestamp"`
}
