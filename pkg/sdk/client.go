package sdk

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

func (c *Client) PostJSON(path string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return c.HTTPClient.Post(c.BaseURL+path, "application/json", bytes.NewReader(body))
}
