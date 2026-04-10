package types

type BlockHeader struct {
	Height      Height    `json:"height"`
	ParentHash  Hash      `json:"parent_hash"`
	StateRoot   Hash      `json:"state_root"`
	TxRoot      Hash      `json:"tx_root"`
	ReceiptRoot Hash      `json:"receipt_root"`
	Timestamp   Timestamp `json:"timestamp"`
	Proposer    Address   `json:"proposer"`
	GasUsed     Gas       `json:"gas_used"`
	GasLimit    Gas       `json:"gas_limit"`
	TxCount     uint64    `json:"tx_count"`
	Hash        Hash      `json:"hash"`
}

type Block struct {
	Header       BlockHeader          `json:"header"`
	Transactions []Transaction        `json:"transactions"`
	Receipts     []TransactionReceipt `json:"receipts,omitempty"`
}
