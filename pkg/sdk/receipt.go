package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func (c *Client) GetReceipt(txHash string) (*http.Response, error) {
	return c.HTTPClient.Get(c.BaseURL + "/tx/receipt?hash=" + url.QueryEscape(txHash))
}

func (c *Client) WaitForReceipt(txHash string, attempts int, delay time.Duration) (map[string]any, error) {
	if attempts <= 0 {
		attempts = 10
	}
	if delay <= 0 {
		delay = 2 * time.Second
	}

	var lastErr error

	for i := 0; i < attempts; i++ {
		resp, err := c.GetReceipt(txHash)
		if err != nil {
			lastErr = err
			time.Sleep(delay)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var out map[string]any
			if err := json.Unmarshal(body, &out); err != nil {
				return nil, err
			}
			return out, nil
		}

		lastErr = fmt.Errorf("receipt not ready: status=%d body=%s", resp.StatusCode, string(body))
		time.Sleep(delay)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("receipt not available")
	}
	return nil, lastErr
}
