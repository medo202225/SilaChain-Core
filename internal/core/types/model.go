package types

const (
	TypeTransfer       = "transfer"
	TypeContractCall   = "contract_call"
	TypeContractDeploy = "contract_deploy"
)

type Transaction struct {
	Type      string    `json:"type"`
	From      Address   `json:"from"`
	To        Address   `json:"to"`
	Value     Amount    `json:"value"`
	Fee       Amount    `json:"fee"`
	GasPrice  Amount    `json:"gas_price"`
	GasLimit  Gas       `json:"gas_limit"`
	Nonce     Nonce     `json:"nonce"`
	ChainID   ChainID   `json:"chain_id"`
	Timestamp Timestamp `json:"timestamp"`

	// Legacy contract call fields (kept for backward compatibility)
	CallMethod string `json:"call_method,omitempty"`
	CallKey    string `json:"call_key,omitempty"`
	CallValue  string `json:"call_value,omitempty"`

	// Sila VM fields
	VMVersion     uint16 `json:"vm_version,omitempty"`
	ContractCode  string `json:"contract_code,omitempty"`
	ContractInput string `json:"contract_input,omitempty"`

	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
	Hash      Hash   `json:"hash"`
}

func (t *Transaction) NormalizedType() string {
	if t == nil {
		return ""
	}
	if t.Type != "" {
		return t.Type
	}
	if t.ContractCode != "" {
		return TypeContractDeploy
	}
	if t.CallMethod != "" || t.ContractInput != "" || t.VMVersion > 0 {
		return TypeContractCall
	}
	return TypeTransfer
}

func (t *Transaction) SigningPayload() SigningPayload {
	return SigningPayload{
		Type:          t.NormalizedType(),
		From:          t.From,
		To:            t.To,
		Value:         t.Value,
		Fee:           t.Fee,
		GasPrice:      t.GasPrice,
		GasLimit:      t.GasLimit,
		Nonce:         t.Nonce,
		ChainID:       t.ChainID,
		Timestamp:     t.Timestamp,
		CallMethod:    t.CallMethod,
		CallKey:       t.CallKey,
		CallValue:     t.CallValue,
		VMVersion:     t.VMVersion,
		ContractCode:  t.ContractCode,
		ContractInput: t.ContractInput,
	}
}

func (t *Transaction) EffectiveFee() Amount {
	if t.Fee > 0 {
		return t.Fee
	}

	if t.GasPrice > 0 && t.GasLimit > 0 {
		return Amount(uint64(t.GasPrice) * uint64(t.GasLimit))
	}

	return 0
}

func (t *Transaction) TotalCost() Amount {
	return t.Value + t.EffectiveFee()
}

func (t *Transaction) IsContractCall() bool {
	return t != nil && t.NormalizedType() == TypeContractCall
}

func (t *Transaction) IsContractDeploy() bool {
	return t != nil && t.NormalizedType() == TypeContractDeploy
}

func (t *Transaction) UsesVM() bool {
	if t == nil {
		return false
	}
	return t.VMVersion > 0 || t.ContractCode != "" || t.ContractInput != ""
}
