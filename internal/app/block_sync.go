package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"silachain/internal/chain"
	coretypes "silachain/internal/core/types"
	pkgtypes "silachain/pkg/types"
)

const BroadcastHeader = "X-Sila-Broadcasted"

var ErrBlockNotFound = errors.New("block not found")

type BlockSyncClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBlockSyncClient(baseURL string, timeout time.Duration) *BlockSyncClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &BlockSyncClient{
		baseURL: NormalizePeer(baseURL),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *BlockSyncClient) FetchBlockByHeight(ctx context.Context, height pkgtypes.Height) (*coretypes.Block, error) {
	if c == nil || c.httpClient == nil {
		return nil, errors.New("block sync client is nil")
	}

	url := c.baseURL + "/blocks/height?height=" + strconv.FormatUint(uint64(height), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(BroadcastHeader, "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status")
	}

	var block coretypes.Block
	if err := json.NewDecoder(resp.Body).Decode(&block); err != nil {
		return nil, err
	}

	return &block, nil
}

type BlockSyncService struct {
	blockchain *chain.Blockchain
	peers      []string
	selfURL    string
	interval   time.Duration
	logger     *log.Logger
	policy     *PeerPolicy
}

func NewBlockSyncService(blockchain *chain.Blockchain, peers []string, selfURL string, interval time.Duration) *BlockSyncService {
	if interval <= 0 {
		interval = time.Second
	}
	return &BlockSyncService{
		blockchain: blockchain,
		peers:      UniquePeers(peers, selfURL),
		selfURL:    NormalizePeer(selfURL),
		interval:   interval,
		logger:     log.Default(),
		policy:     NewPeerPolicy(),
	}
}

func (s *BlockSyncService) Start(ctx context.Context) {
	if s == nil {
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunSyncOnceForTest()
		}
	}
}

func (s *BlockSyncService) RunSyncOnceForTest() {
	if s == nil || s.blockchain == nil {
		return
	}

	for {
		height, err := s.blockchain.Height()
		if err != nil {
			return
		}
		nextHeight := height + 1

		synced := false
		activePeers := s.policy.ActivePeers(s.peers, time.Now(), s.selfURL)

		for _, peer := range activePeers {
			peer = NormalizePeer(peer)

			client := NewBlockSyncClient(peer, 10*time.Second)

			remoteHeight, err := client.FetchChainHeight(context.Background())
			if err != nil {
				s.policy.ReportFailure(peer, time.Now())
				_ = s.persistPeerPolicy()
				continue
			}

			if remoteHeight < nextHeight {
				continue
			}

			block, err := client.FetchBlockByHeight(context.Background(), nextHeight)
			if err != nil {
				s.policy.ReportFailure(peer, time.Now())
				_ = s.persistPeerPolicy()
				continue
			}

			if err := s.blockchain.AddBlock(block); err != nil {
				s.policy.ReportFailure(peer, time.Now())
				_ = s.persistPeerPolicy()
				continue
			}

			s.policy.ReportSuccess(peer)
			_ = s.persistPeerPolicy()
			synced = true
			break
		}

		if !synced {
			return
		}
	}
}

func (s *BlockSyncService) SetPeerBanPolicy(threshold int, duration time.Duration) {
	if s == nil || s.policy == nil {
		return
	}
	s.policy.SetBanPolicy(threshold, duration)
}

func (s *BlockSyncService) PeerFailureCount(peer string) int {
	if s == nil || s.policy == nil {
		return 0
	}
	return s.policy.FailureCount(peer)
}

func (s *BlockSyncService) PeerIsBanned(peer string, now time.Time) bool {
	if s == nil || s.policy == nil {
		return false
	}
	return s.policy.IsBanned(peer, now)
}

func (s *BlockSyncService) SetPeerPolicyPath(path string) error {
	if s == nil || s.policy == nil {
		return nil
	}
	s.policy.path = path

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var state struct {
		States map[string]struct {
			Failures    int       `json:"failures"`
			BannedUntil time.Time `json:"banned_until"`
		} `json:"states"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}

	s.policy.states = map[string]peerState{}
	for peer, st := range state.States {
		s.policy.states[peer] = peerState{
			failures:    st.Failures,
			bannedUntil: st.BannedUntil,
		}
	}

	return nil
}

func (s *BlockSyncService) persistPeerPolicy() error {
	if s == nil || s.policy == nil || s.policy.path == "" {
		return nil
	}

	state := struct {
		States map[string]struct {
			Failures    int       `json:"failures"`
			BannedUntil time.Time `json:"banned_until"`
		} `json:"states"`
	}{
		States: map[string]struct {
			Failures    int       `json:"failures"`
			BannedUntil time.Time `json:"banned_until"`
		}{},
	}

	for peer, st := range s.policy.states {
		state.States[peer] = struct {
			Failures    int       `json:"failures"`
			BannedUntil time.Time `json:"banned_until"`
		}{
			Failures:    st.failures,
			BannedUntil: st.bannedUntil,
		}
	}

	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.policy.path, raw, 0o600)
}

func (s *BlockSyncService) SetPeersPath(path string) error {
	peers, err := LoadPeersFile(path, s.selfURL)
	if err != nil {
		return err
	}
	s.peers = UniquePeers(append(s.peers, peers...), s.selfURL)
	return nil
}

func (s *BlockSyncService) Peers() []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s.peers))
	copy(out, s.peers)
	return out
}

func BroadcastRawTransaction(peersPath string, selfURL string, rawBody []byte) {
	peers, err := LoadPeersFile(peersPath, selfURL)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	for _, peer := range peers {
		req, err := http.NewRequest(http.MethodPost, strings.TrimRight(peer, "/")+"/tx/send", strings.NewReader(string(rawBody)))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(BroadcastHeader, "1")

		resp, err := client.Do(req)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
	}
}

type chainInfoResponse struct {
	Height uint64 `json:"height"`
}

func (c *BlockSyncClient) FetchChainHeight(ctx context.Context) (pkgtypes.Height, error) {
	if c == nil || c.httpClient == nil {
		return 0, errors.New("block sync client is nil")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/chain/info", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set(BroadcastHeader, "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.New("unexpected chain info status")
	}

	var out chainInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}

	return pkgtypes.Height(out.Height), nil
}
func (s *BlockSyncService) Policy() *PeerPolicy {
	if s == nil {
		return nil
	}
	return s.policy
}

func (s *BlockSyncService) SelfURL() string {
	if s == nil {
		return ""
	}
	return s.selfURL
}
