package types

type SigningPayload struct {
	Type          string    `json:"type"`
	From          Address   `json:"from"`
	To            Address   `json:"to"`
	Value         Amount    `json:"value"`
	Fee           Amount    `json:"fee"`
	GasPrice      Amount    `json:"gas_price"`
	GasLimit      Gas       `json:"gas_limit"`
	Nonce         Nonce     `json:"nonce"`
	ChainID       ChainID   `json:"chain_id"`
	Timestamp     Timestamp `json:"timestamp"`
	CallMethod    string    `json:"call_method,omitempty"`
	CallKey       string    `json:"call_key,omitempty"`
	CallValue     string    `json:"call_value,omitempty"`
	VMVersion     uint16    `json:"vm_version,omitempty"`
	ContractCode  string    `json:"contract_code,omitempty"`
	ContractInput string    `json:"contract_input,omitempty"`
}
