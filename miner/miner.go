package miner

import (
	"context"
	"sync"
	"time"

	"silachain/internal/consensus/blockassembly"
)

type Backend interface{}

type Config struct {
	PendingFeeRecipient string
	ExtraData           []byte
	GasCeil             uint64
	Recommit            time.Duration
}

var DefaultConfig = Config{
	GasCeil:  60_000_000,
	Recommit: 2 * time.Second,
}

type Miner struct {
	confMu  sync.RWMutex
	config  *Config
	backend Backend
}

func New(backend Backend, config Config) *Miner {
	if config.GasCeil == 0 {
		config.GasCeil = DefaultConfig.GasCeil
	}
	if config.Recommit == 0 {
		config.Recommit = DefaultConfig.Recommit
	}
	return &Miner{
		config:  &config,
		backend: backend,
	}
}

func (m *Miner) Config() Config {
	if m == nil || m.config == nil {
		return Config{}
	}
	m.confMu.RLock()
	defer m.confMu.RUnlock()
	return *m.config
}

func (m *Miner) SetPendingFeeRecipient(recipient string) {
	if m == nil || m.config == nil {
		return
	}
	m.confMu.Lock()
	m.config.PendingFeeRecipient = recipient
	m.confMu.Unlock()
}

func (m *Miner) SetExtra(extra []byte) {
	if m == nil || m.config == nil {
		return
	}
	m.confMu.Lock()
	m.config.ExtraData = append([]byte(nil), extra...)
	m.confMu.Unlock()
}

func (m *Miner) SetGasCeil(ceil uint64) {
	if m == nil || m.config == nil {
		return
	}
	m.confMu.Lock()
	m.config.GasCeil = ceil
	m.confMu.Unlock()
}

func (m *Miner) BuildPayload(ctx context.Context, args BuildPayloadArgs, built blockassembly.Result, stateRoot string) (*Payload, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	_ = ctx
	return NewPayload(args, built, stateRoot)
}
