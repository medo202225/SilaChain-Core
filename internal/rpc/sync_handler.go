package rpc

import (
	"encoding/json"
	"net/http"
	"time"

	"silachain/internal/app"
	"silachain/internal/chain"
	"silachain/internal/config"
)

func SyncStatusHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		height, _ := blockchain.Height()
		peersCfg, _ := config.LoadPeersConfig("config/networks/mainnet/public/peers.json")

		out := map[string]any{
			"local_height": height,
			"peer_count":   len(peersCfg.Peers),
			"peers":        peersCfg.Peers,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func SyncStatusWithPolicyHandler(blockchain *chain.Blockchain, peers []string, selfURL string, policy *app.PeerPolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		height, _ := blockchain.Height()
		now := time.Now()

		allPeers := app.UniquePeers(peers, selfURL)
		active := policy.ActivePeers(peers, now, selfURL)

		out := map[string]any{
			"local_height": height,
			"peer_count":   len(allPeers),
			"peers":        allPeers,
			"active_count": len(active),
			"active_peers": active,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func MempoolSyncHandler(blockchain *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		height, _ := blockchain.Height()
		peersCfg, _ := config.LoadPeersConfig("config/networks/mainnet/public/peers.json")

		out := map[string]any{
			"local_height":  height,
			"mempool_count": blockchain.Mempool().Count(),
			"peer_count":    len(peersCfg.Peers),
			"peers":         peersCfg.Peers,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func PeerPolicyStatusHandler(peers []string, selfURL string, policy *app.PeerPolicy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		snapshot := policy.Snapshot(now, peers, selfURL)
		active := policy.ActivePeers(peers, now, selfURL)

		out := map[string]any{
			"peer_count":   len(app.UniquePeers(peers, selfURL)),
			"active_count": len(active),
			"active_peers": active,
			"policy":       snapshot,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
