package validator

import (
	"encoding/json"
	"os"
	"strings"

	"silachain/pkg/types"
)

type validatorFileMember struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	Power     uint64 `json:"power"`
	Stake     uint64 `json:"stake"`
}

func LoadSet(path string) (*Set, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return NewSet(nil), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return NewSet(nil), nil
	}

	var raw []validatorFileMember
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	members := make([]Member, 0, len(raw))
	for _, item := range raw {
		members = append(members, Member{
			Address:   types.Address(item.Address),
			PublicKey: item.PublicKey,
			Power:     item.Power,
			Stake:     item.Stake,
		})
	}

	return NewSet(members), nil
}
