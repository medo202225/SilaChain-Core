package types

type Transaction struct {
	From      Address   `json:"from"`
	To        Address   `json:"to"`
	Value     Amount    `json:"value"`
	Fee       Amount    `json:"fee"`
	GasPrice  Amount    `json:"gas_price"`
	GasLimit  Gas       `json:"gas_limit"`
	Nonce     Nonce     `json:"nonce"`
	ChainID   ChainID   `json:"chain_id"`
	Timestamp Timestamp `json:"timestamp"`
	PublicKey string    `json:"public_key"`
	Signature string    `json:"signature"`
	Hash      Hash      `json:"hash"`
}

type TransactionSigningPayload struct {
	From      Address   `json:"from"`
	To        Address   `json:"to"`
	Value     Amount    `json:"value"`
	Fee       Amount    `json:"fee"`
	GasPrice  Amount    `json:"gas_price"`
	GasLimit  Gas       `json:"gas_limit"`
	Nonce     Nonce     `json:"nonce"`
	ChainID   ChainID   `json:"chain_id"`
	Timestamp Timestamp `json:"timestamp"`
}

type TransactionReceipt struct {
	TxHash            Hash   `json:"tx_hash"`
	BlockHash         Hash   `json:"block_hash"`
	BlockHeight       Height `json:"block_height"`
	TransactionIndex  uint64 `json:"transaction_index"`
	Success           bool   `json:"success"`
	GasUsed           Gas    `json:"gas_used"`
	CumulativeGasUsed Gas    `json:"cumulative_gas_used"`
	Error             string `json:"error,omitempty"`
}
