package consensuslegacy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func BroadcastAttestation(peers []string, selfURL string, att Attestation) {
	body, err := json.Marshal(att)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 4 * time.Second}

	for _, peer := range peers {
		peer = strings.TrimSpace(peer)
		if peer == "" {
			continue
		}
		if strings.EqualFold(strings.TrimRight(peer, "/"), strings.TrimRight(selfURL, "/")) {
			continue
		}

		url := strings.TrimRight(peer, "/") + "/consensus/attestations/submit"

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}
}
