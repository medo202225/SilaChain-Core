package types

type Validator struct {
	Address   Address `json:"address"`
	PublicKey string  `json:"public_key"`
	Stake     Amount  `json:"stake"`
	Active    bool    `json:"active"`
}
