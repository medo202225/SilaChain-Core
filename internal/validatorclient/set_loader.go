package validatorclient

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadSet(path string) (*ValidatorSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var vals []ValidatorRecord
	if err := json.Unmarshal(data, &vals); err != nil {
		return nil, fmt.Errorf("decode validator set: %w", err)
	}

	return &ValidatorSet{
		Validators: vals,
	}, nil
}
