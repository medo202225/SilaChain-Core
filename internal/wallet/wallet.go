package wallet

// CANONICAL OWNERSHIP: internal wallet helper only.
// Public and user-facing wallet ownership lives in pkg/sdk.

type Wallet struct {
	Address       string `json:"address"`
	PublicKeyHex  string `json:"public_key"`
	PrivateKeyHex string `json:"private_key"`
}
