package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Enabled            bool     `json:"enabled"`
	NetworkName        string   `json:"network_name"`
	ListenIP           string   `json:"listen_ip"`
	TCPPort            int      `json:"tcp_port"`
	UDPPort            int      `json:"udp_port"`
	MaxPeers           int      `json:"max_peers"`
	KeyFile            string   `json:"key_file"`
	Bootnodes          []string `json:"bootnodes"`
	ExecutionNetworkID uint64   `json:"execution_network_id"`
	GenesisHash        string   `json:"genesis_hash"`
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read p2p config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("decode p2p config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.NetworkName == "" {
		return errors.New("p2p config: network_name is required")
	}
	if c.ListenIP == "" {
		return errors.New("p2p config: listen_ip is required")
	}
	if c.TCPPort <= 0 || c.TCPPort > 65535 {
		return errors.New("p2p config: tcp_port must be between 1 and 65535")
	}
	if c.UDPPort <= 0 || c.UDPPort > 65535 {
		return errors.New("p2p config: udp_port must be between 1 and 65535")
	}
	if c.MaxPeers <= 0 {
		return errors.New("p2p config: max_peers must be > 0")
	}
	if c.KeyFile == "" {
		return errors.New("p2p config: key_file is required")
	}
	if c.ExecutionNetworkID == 0 {
		return errors.New("p2p config: execution_network_id must be > 0")
	}
	if strings.TrimSpace(c.GenesisHash) == "" {
		return errors.New("p2p config: genesis_hash is required")
	}
	return nil
}

func (c *Config) EnsurePaths() error {
	dir := filepath.Dir(c.KeyFile)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create p2p key directory: %w", err)
	}
	return nil
}
