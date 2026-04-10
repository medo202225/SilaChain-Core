package types

type Block struct {
	Header       Header        `json:"header"`
	Transactions []Transaction `json:"transactions"`
	Receipts     []Receipt     `json:"receipts,omitempty"`
}
