package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engine"
	"silachain/internal/consensus/engineapi"
	"silachain/internal/consensus/engineapiserver"
	"silachain/internal/consensus/executionstate"
	"silachain/internal/consensus/txpool"
	"silachain/internal/consensus/txpoolapi"
	"silachain/internal/consensus/vmstate"
	coretypes "silachain/internal/core/types"
)

var (
	ErrEmptyListenAddress = errors.New("runtime: empty listen address")
	ErrZeroGasLimit       = errors.New("runtime: zero gas limit")
	ErrNilHTTPServer      = errors.New("runtime: nil http server")
)

type Config struct {
	ListenAddress string
	GasLimit      uint64
	GenesisHead   blockassembly.Head
}

type State struct {
	head blockassembly.Head
	vm   *vmstate.State
	exec *executionstate.State
}

func NewState(head blockassembly.Head) *State {
	return &State{
		head: head,
		vm:   vmstate.New(),
		exec: executionstate.NewState(head.Hash),
	}
}

func (s *State) Head() blockassembly.Head {
	return s.head
}

func (s *State) SetHead(head blockassembly.Head) error {
	s.head = head
	return nil
}

func (s *State) SetSenderNonce(sender string, nonce uint64) error {
	return s.vm.SetNonce(sender, nonce)
}

func (s *State) SenderNonce(sender string) uint64 {
	acct, ok := s.vm.GetAccount(sender)
	if !ok {
		return 0
	}
	return acct.Nonce
}

func (s *State) SetBalance(address string, balance uint64) error {
	return s.vm.SetBalance(address, balance)
}

func (s *State) AddBalance(address string, amount uint64) error {
	return s.vm.AddBalance(address, amount)
}

func (s *State) SetCode(address string, code []byte) error {
	return s.vm.SetCode(address, code)
}

func (s *State) SetStorage(address, key, value string) error {
	return s.vm.SetStorage(address, key, value)
}

func (s *State) Account(address string) (vmstate.Account, bool) {
	return s.vm.GetAccount(address)
}

func (s *State) Code(address string) ([]byte, bool) {
	return s.vm.GetCode(address)
}

func (s *State) Storage(address, key string) (string, bool) {
	return s.vm.GetStorage(address, key)
}

// Runtime currently has no chain-backed persisted lookup reader.
// Restart-stable receipt/tx/log introspection will require a small reader seam
// so runtime can query persisted chain data without depending on chain.Blockchain directly.

type ReceiptReader interface {
	GetReceiptByHash(hash string) (*coretypes.Receipt, bool)
}

type Runtime struct {
	cfg                 Config
	state               *State
	pool                *txpool.Pool
	engine              *engine.Engine
	api                 *engineapi.BuilderService
	receiptReader       ReceiptReader
	apiService          *APIService
	server              *engineapiserver.Server
	produceBlockServer  *ProduceBlockServer
	introspectionServer *IntrospectionServer
	txpoolAPI           *txpoolapi.Service
	txpoolServer        *txpoolapi.Server
	httpServer          *http.Server
}

func New(cfg Config) (*Runtime, error) {
	if cfg.ListenAddress == "" {
		return nil, ErrEmptyListenAddress
	}
	if cfg.GasLimit == 0 {
		return nil, ErrZeroGasLimit
	}
	if cfg.GenesisHead.Hash == "" {
		cfg.GenesisHead = blockassembly.Head{
			Number:    0,
			Hash:      "sila-genesis-v2",
			StateRoot: "sila-state-0",
			BaseFee:   1,
		}
	}

	state := NewState(cfg.GenesisHead)
	pool := txpool.NewPool(cfg.GenesisHead.BaseFee)

	eng, err := engine.New(state, pool, cfg.GasLimit)
	if err != nil {
		return nil, err
	}

	api, err := engineapi.NewBuilderServiceFromEngine(eng)
	if err != nil {
		return nil, err
	}

	rt := &Runtime{
		cfg:    cfg,
		state:  state,
		pool:   pool,
		engine: eng,
		api:    api,
	}

	apiService := NewAPIService(rt)
	server, err := engineapiserver.New(apiService)
	if err != nil {
		return nil, err
	}

	produceBlockServer, err := NewProduceBlockServer(rt)
	if err != nil {
		return nil, err
	}

	introspectionServer, err := NewIntrospectionServer(rt)
	if err != nil {
		return nil, err
	}

	txAPI, err := txpoolapi.New(pool, state)
	if err != nil {
		return nil, err
	}

	txServer, err := txpoolapi.NewServer(txAPI)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/engine/", http.StripPrefix("", server.Handler()))
	mux.Handle("/txpool/", http.StripPrefix("", txServer.Handler()))
	mux.Handle("/engine/produceBlock", http.StripPrefix("", produceBlockServer.Handler()))
	mux.Handle("/chain/", http.StripPrefix("", introspectionServer.Handler()))
	mux.Handle("/state/", http.StripPrefix("", introspectionServer.Handler()))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"sila-consensus-engine"}`))
	})

	httpServer := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	rt.apiService = apiService
	rt.server = server
	rt.produceBlockServer = produceBlockServer
	rt.introspectionServer = introspectionServer
	rt.txpoolAPI = txAPI
	rt.txpoolServer = txServer
	rt.httpServer = httpServer

	return rt, nil
}

func (r *Runtime) Config() Config {
	return r.cfg
}

func (r *Runtime) State() *State {
	return r.state
}

func (r *Runtime) Pool() *txpool.Pool {
	return r.pool
}

func (r *Runtime) Engine() *engine.Engine {
	return r.engine
}

func (r *Runtime) API() *engineapi.BuilderService {
	return r.api
}

func (r *Runtime) SetReceiptReader(reader ReceiptReader) {
	if r == nil {
		return
	}
	r.receiptReader = reader
}

func (r *Runtime) HTTPServer() *http.Server {
	return r.httpServer
}

func (r *Runtime) Start() error {
	if r == nil || r.httpServer == nil {
		return ErrNilHTTPServer
	}
	return r.httpServer.ListenAndServe()
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil || r.httpServer == nil {
		return ErrNilHTTPServer
	}
	return r.httpServer.Shutdown(ctx)
}

func (s *State) ExecuteBlock(req executionstate.BlockExecutionRequest) (executionstate.BlockExecutionResult, error) {
	if s == nil {
		return executionstate.BlockExecutionResult{}, errors.New("runtime: nil state")
	}
	if s.exec == nil {
		s.exec = executionstate.NewState(s.head.Hash)
	}

	for i := uint64(s.exec.HeadNumber() + 1); i <= req.Block.Number-1; i++ {
		hash := "0xseed-runtime-block"
		if i == req.Block.Number-1 {
			hash = req.Block.ParentHash
		} else {
			hash = fmt.Sprintf("0xseed-runtime-block-%d", i)
		}

		parentHash := s.exec.HeadHash()
		if err := s.exec.ImportBlock(executionstate.ImportedBlock{
			Number:     i,
			Hash:       hash,
			ParentHash: parentHash,
			Timestamp:  i,
			TxHashes:   nil,
		}); err != nil {
			return executionstate.BlockExecutionResult{}, err
		}
	}

	for _, tx := range req.Txs {
		s.exec.SetBalance(tx.From, 1000000000)
	}

	result, err := s.exec.ExecuteBlock(req)
	if err != nil {
		return executionstate.BlockExecutionResult{}, err
	}

	s.head = blockassembly.Head{
		Number:    result.BlockNumber,
		Hash:      result.BlockHash,
		StateRoot: result.StateRoot,
		BaseFee:   s.head.BaseFee,
	}
	return result, nil
}
