package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"silachain/internal/app"
	"silachain/internal/chain"
	"silachain/internal/config"
	"silachain/internal/rpc"
	validatorclient "silachain/internal/validatorclient"
)

const (
	mainnetRoot     = "config/networks/mainnet"
	defaultNodePath = "config/networks/mainnet/execution/node.json"
	validatorsPath  = "config/networks/mainnet/public/validators.json"
	peersPath       = "config/networks/mainnet/public/peers.json"
	bootnodesPath   = "config/networks/mainnet/public/bootnodes.json"
)

type BootnodesConfig struct {
	Bootnodes []string `json:"bootnodes"`
}

type Service struct {
	configPath          string
	listenAddress       string
	engineListenAddress string
	engineJWTSecretPath string
	dataDir             string
	validatorsCount     int
	peersCount          int
	bootnodesCount      int
	blockchain          *chain.Blockchain
	server              *rpc.Server
	engineServer        *EngineServer
	blockSync           *app.BlockSyncService
	blockSyncCancel     context.CancelFunc
}

func mustMainnetPath(p string) string {
	clean := filepath.Clean(p)

	mainnetRootClean := filepath.Clean(mainnetRoot)
	mainnetPrefix := mainnetRootClean + string(os.PathSeparator)

	runtimeRootClean := filepath.Clean("runtime")
	runtimePrefix := runtimeRootClean + string(os.PathSeparator)

	if clean == mainnetRootClean || strings.HasPrefix(clean, mainnetPrefix) {
		return clean
	}
	if clean == runtimeRootClean || strings.HasPrefix(clean, runtimePrefix) {
		return clean
	}

	log.Fatalf("mainnet/runtime-only path required: %s", p)
	return ""
}

func mustFile(p string) {
	info, err := os.Stat(p)
	if err != nil {
		log.Fatalf("required file missing: %s: %v", p, err)
	}
	if info.IsDir() {
		log.Fatalf("required file is a directory, not a file: %s", p)
	}
}

func loadBootnodes(path string) BootnodesConfig {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("load bootnodes failed: %v", err)
	}

	var cfg BootnodesConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("parse bootnodes failed: %v", err)
	}
	return cfg
}

func resolveConfigPath(args []string) string {
	configPath := defaultNodePath

	switch len(args) {
	case 0:
	case 2:
		if args[0] != "--config" {
			log.Fatalf("unsupported argument: %s", args[0])
		}
		configPath = args[1]
	default:
		log.Fatalf("usage: go run ./cmd/sila-execution --config %s", defaultNodePath)
	}

	return mustMainnetPath(configPath)
}

func NewService(args []string) (*Service, error) {
	configPath := resolveConfigPath(args)
	mustFile(configPath)

	mustFile(mustMainnetPath(validatorsPath))
	mustFile(mustMainnetPath(peersPath))
	mustFile(mustMainnetPath(bootnodesPath))

	cfg, err := config.LoadExecutionNodeConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load execution node config failed: %w", err)
	}

	if strings.TrimSpace(cfg.ListenAddress) == "" {
		return nil, fmt.Errorf("execution config invalid: listen_address is required")
	}
	if strings.HasPrefix(cfg.ListenAddress, "127.0.0.1:") || strings.HasPrefix(strings.ToLower(cfg.ListenAddress), "localhost:") {
		return nil, fmt.Errorf("execution config invalid: mainnet listen_address must not be loopback: %s", cfg.ListenAddress)
	}
	if strings.TrimSpace(cfg.DataDir) == "" {
		return nil, fmt.Errorf("execution config invalid: data_dir is required")
	}
	if strings.TrimSpace(cfg.PeersPath) == "" {
		return nil, fmt.Errorf("execution config invalid: peers_path is required")
	}
	if strings.TrimSpace(cfg.EngineListenAddress) == "" {
		return nil, fmt.Errorf("execution config invalid: engine_listen_address is required")
	}
	if strings.TrimSpace(cfg.EngineJWTSecretPath) == "" {
		return nil, fmt.Errorf("execution config invalid: engine_jwt_secret_path is required")
	}

	cfg.PeersPath = mustMainnetPath(cfg.PeersPath)
	cfg.EngineJWTSecretPath = mustMainnetPath(cfg.EngineJWTSecretPath)

	if filepath.Clean(cfg.PeersPath) != filepath.Clean(peersPath) {
		return nil, fmt.Errorf("execution config invalid: peers_path must be %s, got %s", peersPath, cfg.PeersPath)
	}
	mustFile(cfg.PeersPath)

	validatorSet, err := validatorclient.LoadSet(validatorsPath)
	if err != nil {
		return nil, fmt.Errorf("load validators failed: %w", err)
	}
	if validatorSet == nil || len(validatorSet.All()) == 0 {
		return nil, fmt.Errorf("validator set is empty")
	}

	peersCfg, err := config.LoadPeersConfig(cfg.PeersPath)
	if err != nil {
		return nil, fmt.Errorf("load peers failed: %w", err)
	}

	bootnodesCfg := loadBootnodes(bootnodesPath)

	selfURL := "http://" + cfg.ListenAddress

	for _, peer := range peersCfg.Peers {
		if strings.TrimSpace(peer) == "" {
			return nil, fmt.Errorf("peers config invalid: empty peer entry")
		}
		if strings.EqualFold(strings.TrimSpace(peer), selfURL) {
			return nil, fmt.Errorf("peers config invalid: self peer is not allowed on mainnet: %s", peer)
		}
		if strings.Contains(peer, "127.0.0.1") || strings.Contains(strings.ToLower(peer), "localhost") {
			return nil, fmt.Errorf("peers config invalid: loopback peer is not allowed on mainnet: %s", peer)
		}
	}

	for _, bootnode := range bootnodesCfg.Bootnodes {
		if strings.TrimSpace(bootnode) == "" {
			return nil, fmt.Errorf("bootnodes config invalid: empty bootnode entry")
		}
		if strings.Contains(bootnode, "127.0.0.1") || strings.Contains(strings.ToLower(bootnode), "localhost") {
			return nil, fmt.Errorf("bootnodes config invalid: loopback bootnode is not allowed on mainnet: %s", bootnode)
		}
	}

	if len(peersCfg.Peers) == 0 && len(bootnodesCfg.Bootnodes) == 0 {
		return nil, fmt.Errorf("mainnet startup rejected: at least one remote peer or bootnode is required")
	}

	jwtSecret, err := EnsureJWTSecretFile(cfg.EngineJWTSecretPath)
	if err != nil {
		return nil, fmt.Errorf("ensure engine jwt secret failed: %w", err)
	}

	blockchain, err := chain.NewBlockchain(cfg.DataDir, nil, cfg.MinValidatorStake)
	if err != nil {
		return nil, fmt.Errorf("create blockchain failed: %w", err)
	}

	allPeers := append([]string{}, peersCfg.Peers...)
	allPeers = append(allPeers, bootnodesCfg.Bootnodes...)
	blockSync := app.NewBlockSyncService(blockchain, allPeers, selfURL, 15*time.Second)

	policyPath := filepath.Join(cfg.DataDir, "peer-policy.json")
	if err := blockSync.SetPeerPolicyPath(policyPath); err != nil {
		return nil, fmt.Errorf("load block sync peer policy failed: %w", err)
	}

	server := rpc.NewServer(cfg.ListenAddress)
	rpc.RegisterRoutes(server, blockchain, nil)
	server.Router().HandleFunc("/sync/peers", rpc.PeerPolicyStatusHandler(
		blockSync.Peers(),
		blockSync.SelfURL(),
		blockSync.Policy(),
	))

	engineServer := NewEngineServer(cfg.EngineListenAddress, jwtSecret, selfURL)

	return &Service{
		configPath:          configPath,
		listenAddress:       cfg.ListenAddress,
		engineListenAddress: cfg.EngineListenAddress,
		engineJWTSecretPath: cfg.EngineJWTSecretPath,
		dataDir:             cfg.DataDir,
		validatorsCount:     len(validatorSet.All()),
		peersCount:          len(peersCfg.Peers),
		bootnodesCount:      len(bootnodesCfg.Bootnodes),
		blockchain:          blockchain,
		server:              server,
		engineServer:        engineServer,
		blockSync:           blockSync,
	}, nil
}

func (s *Service) ConfigPath() string {
	if s == nil {
		return ""
	}
	return s.configPath
}

func (s *Service) Blockchain() *chain.Blockchain {
	if s == nil {
		return nil
	}
	return s.blockchain
}

func (s *Service) RPCServer() *rpc.Server {
	if s == nil {
		return nil
	}
	return s.server
}

func (s *Service) EngineServer() *EngineServer {
	if s == nil {
		return nil
	}
	return s.engineServer
}

func (s *Service) BlockSync() *app.BlockSyncService {
	if s == nil {
		return nil
	}
	return s.blockSync
}

func (s *Service) Start() error {
	if s == nil {
		return fmt.Errorf("execution service is nil")
	}
	if s.server == nil {
		return fmt.Errorf("execution rpc server is nil")
	}
	if s.engineServer == nil {
		return fmt.Errorf("execution engine server is nil")
	}

	go func() {
		log.Printf("execution engine startup listen=http://%s/engine", s.engineListenAddress)
		if err := s.engineServer.Start(); err != nil {
			log.Printf("execution engine server stopped: %v", err)
		}
	}()

	if s.blockSync != nil {
		syncCtx, cancel := context.WithCancel(context.Background())
		s.blockSyncCancel = cancel

		go s.blockSync.Start(syncCtx)
		log.Printf("execution block sync startup peers=%d bootnodes=%d", s.peersCount, s.bootnodesCount)
	}

	log.Printf("execution startup network=mainnet")
	log.Printf("execution startup node_config=%s", s.configPath)
	log.Printf("execution startup validators=%s count=%d", validatorsPath, s.validatorsCount)
	log.Printf("execution startup peers=%s count=%d", peersPath, s.peersCount)
	log.Printf("execution startup bootnodes=%s count=%d", bootnodesPath, s.bootnodesCount)
	log.Printf("execution startup listen=http://%s", s.listenAddress)
	log.Printf("execution startup engine_listen=http://%s/engine", s.engineListenAddress)
	log.Printf("execution startup engine_jwt_secret_path=%s", s.engineJWTSecretPath)
	log.Printf("execution startup data_dir=%s", s.dataDir)

	err := s.server.Start()
	if s.blockSyncCancel != nil {
		s.blockSyncCancel()
	}
	return err
}

func RunExecution(args []string) error {
	service, err := NewService(args)
	if err != nil {
		return err
	}
	return service.Start()
}
