package accounts

import "silachain/pkg/types"

type Account struct {
	Address     types.Address `json:"address"`
	PublicKey   string        `json:"public_key"`
	Balance     types.Amount  `json:"balance"`
	Nonce       types.Nonce   `json:"nonce"`
	CodeHash    types.Hash    `json:"code_hash"`
	StorageRoot types.Hash    `json:"storage_root"`
}

func NewAccount(address types.Address, publicKey string) *Account {
	return &Account{
		Address:     address,
		PublicKey:   publicKey,
		Balance:     0,
		Nonce:       0,
		CodeHash:    "",
		StorageRoot: "",
	}
}

func (a *Account) Credit(amount types.Amount) {
	a.Balance += amount
}

func (a *Account) CanDebit(amount types.Amount) bool {
	return a.Balance >= amount
}

func (a *Account) Debit(amount types.Amount) error {
	if !a.CanDebit(amount) {
		return ErrInsufficientBalance
	}
	a.Balance -= amount
	return nil
}

func (a *Account) IncrementNonce() {
	a.Nonce++
}

func (a *Account) SetCodeHash(codeHash types.Hash) {
	a.CodeHash = codeHash
}

func (a *Account) SetStorageRoot(storageRoot types.Hash) {
	a.StorageRoot = storageRoot
}

func (a *Account) HasCode() bool {
	return a != nil && a.CodeHash != ""
}

func (a *Account) IsContract() bool {
	return a != nil && a.HasCode()
}

func (a *Account) IsEOA() bool {
	return a != nil && !a.HasCode()
}
