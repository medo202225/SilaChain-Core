package genesis

type Account struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	Balance   uint64 `json:"balance"`
	Nonce     uint64 `json:"nonce"`
}

type Config struct {
	ChainID     uint64    `json:"chain_id"`
	NetworkName string    `json:"network_name"`
	Symbol      string    `json:"symbol"`
	GenesisTime int64     `json:"genesis_time"`
	Accounts    []Account `json:"accounts"`
}
