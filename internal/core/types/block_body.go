package types

type BlockBody struct {
	Transactions []Transaction `json:"transactions"`
	Receipts     []Receipt     `json:"receipts,omitempty"`
}

func NewBlockBody() BlockBody {
	return BlockBody{
		Transactions: []Transaction{},
		Receipts:     []Receipt{},
	}
}
