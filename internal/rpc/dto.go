package rpc

type RegisterAccountRequest struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
}

type FaucetRequest struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
}

type SendTxRequest struct {
	Type       string `json:"type"`
	From       string `json:"from"`
	To         string `json:"to"`
	Value      uint64 `json:"value"`
	Fee        uint64 `json:"fee"`
	GasPrice   uint64 `json:"gas_price"`
	GasLimit   uint64 `json:"gas_limit"`
	Nonce      uint64 `json:"nonce"`
	ChainID    uint64 `json:"chain_id"`
	Timestamp  int64  `json:"timestamp"`
	CallMethod string `json:"call_method,omitempty"`
	CallKey    string `json:"call_key,omitempty"`
	CallValue  string `json:"call_value,omitempty"`
	PublicKey  string `json:"public_key"`
	Signature  string `json:"signature"`
	Hash       string `json:"hash"`
}

type MineRequest struct {
	Proposer string `json:"proposer"`
}

type DelegationRequest struct {
	Delegator string `json:"delegator"`
	Validator string `json:"validator"`
	Amount    uint64 `json:"amount"`
}
