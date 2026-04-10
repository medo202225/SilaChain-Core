package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	coretypes "silachain/internal/core/types"
)

type balanceResponse struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

func GetAccountNonce(nodeURL string, address string) (uint64, error) {
	resp, err := http.Get(nodeURL + "/accounts/balance?address=" + address)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("%s", string(respBody))
	}

	var out balanceResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return 0, err
	}

	return out.Nonce, nil
}

func SendTransaction(nodeURL string, t *coretypes.Transaction) ([]byte, error) {
	body, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(nodeURL+"/tx/send", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, errors.New(string(respBody))
	}

	return respBody, nil
}
