package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	coretypes "silachain/internal/core/types"
)

type TxBroadcaster struct {
	selfURL string
	seen    sync.Map
	client  *http.Client
}

func NewTxBroadcaster(selfURL string) *TxBroadcaster {
	return &TxBroadcaster{
		selfURL: NormalizePeer(selfURL),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (b *TxBroadcaster) BroadcastTransaction(peers []string, policy *PeerPolicy, tx *coretypes.Transaction) {
	if b == nil || tx == nil {
		return
	}

	if b.client == nil {
		b.client = &http.Client{Timeout: 10 * time.Second}
	}

	if _, loaded := b.seen.LoadOrStore(string(tx.Hash), struct{}{}); loaded {
		return
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return
	}

	for _, peer := range UniquePeers(peers, b.selfURL) {
		req, err := http.NewRequest(http.MethodPost, peer+"/tx/send", bytes.NewReader(raw))
		if err != nil {
			if policy != nil {
				policy.ReportFailure(peer, time.Now())
			}
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(BroadcastHeader, "1")

		resp, err := b.client.Do(req)
		if err != nil {
			if policy != nil {
				policy.ReportFailure(peer, time.Now())
			}
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if policy != nil {
				policy.ReportSuccess(peer)
			}
		} else {
			if policy != nil {
				policy.ReportFailure(peer, time.Now())
			}
		}
	}
}
