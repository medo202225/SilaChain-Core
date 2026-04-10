package sdk

type DeployTxRequest struct {
	From         string `json:"from"`
	Value        uint64 `json:"value"`
	Fee          uint64 `json:"fee"`
	GasPrice     uint64 `json:"gas_price"`
	GasLimit     uint64 `json:"gas_limit"`
	Nonce        uint64 `json:"nonce"`
	ChainID      uint64 `json:"chain_id"`
	Timestamp    int64  `json:"timestamp"`
	VMVersion    uint16 `json:"vm_version"`
	ContractCode string `json:"contract_code"`
	PublicKey    string `json:"public_key"`
	Signature    string `json:"signature"`
	Hash         string `json:"hash"`
}

type CallTxRequest struct {
	From          string `json:"from"`
	Address       string `json:"address"`
	Value         uint64 `json:"value"`
	Fee           uint64 `json:"fee"`
	GasPrice      uint64 `json:"gas_price"`
	GasLimit      uint64 `json:"gas_limit"`
	Nonce         uint64 `json:"nonce"`
	ChainID       uint64 `json:"chain_id"`
	Timestamp     int64  `json:"timestamp"`
	VMVersion     uint16 `json:"vm_version"`
	ContractInput string `json:"contract_input"`
	PublicKey     string `json:"public_key"`
	Signature     string `json:"signature"`
	Hash          string `json:"hash"`
}

type ReadOnlyCallRequest struct {
	To        string `json:"to"`
	Input     string `json:"input"`
	VMVersion uint16 `json:"vm_version"`
	GasLimit  uint64 `json:"gas_limit"`
}

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}
