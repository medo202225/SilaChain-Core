package genesis

import (
	"silachain/internal/accounts"
	"silachain/pkg/types"
)

func ApplyAccounts(manager *accounts.Manager, cfg *Config) error {
	if cfg == nil {
		return nil
	}

	for _, a := range cfg.Accounts {
		acc := &accounts.Account{
			Address:   types.Address(a.Address),
			PublicKey: a.PublicKey,
			Balance:   types.Amount(a.Balance),
			Nonce:     types.Nonce(a.Nonce),
		}
		if err := manager.Set(acc); err != nil {
			return err
		}
	}

	return nil
}
