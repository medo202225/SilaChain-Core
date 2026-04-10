package txpoolapi

import (
	"errors"

	"silachain/internal/consensus/txpool"
)

var (
	ErrNilPool  = errors.New("txpoolapi: nil tx pool")
	ErrNilState = errors.New("txpoolapi: nil state")
)

type State interface {
	SenderNonce(sender string) uint64
}

type Service struct {
	pool  *txpool.Pool
	state State
}

type AddTxRequest struct {
	Hash                 string `json:"hash"`
	From                 string `json:"from"`
	Nonce                uint64 `json:"nonce"`
	GasLimit             uint64 `json:"gas_limit"`
	MaxFeePerGas         uint64 `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas uint64 `json:"max_priority_fee_per_gas"`
	Timestamp            int64  `json:"timestamp"`
}

type AddTxResult struct {
	Accepted     bool   `json:"accepted"`
	PendingCount int    `json:"pending_count"`
	Hash         string `json:"hash"`
}

type StatusResult struct {
	PendingCount int    `json:"pending_count"`
	BaseFee      uint64 `json:"base_fee"`
}

func New(pool *txpool.Pool, state State) (*Service, error) {
	if pool == nil {
		return nil, ErrNilPool
	}
	if state == nil {
		return nil, ErrNilState
	}

	return &Service{
		pool:  pool,
		state: state,
	}, nil
}

func (s *Service) Add(req AddTxRequest) (AddTxResult, error) {
	if s == nil || s.pool == nil {
		return AddTxResult{}, ErrNilPool
	}
	if s.state == nil {
		return AddTxResult{}, ErrNilState
	}

	senderNonce := s.state.SenderNonce(req.From)
	if err := s.pool.SetSenderStateNonce(req.From, senderNonce); err != nil {
		return AddTxResult{}, err
	}

	tx := txpool.Tx{
		Hash:                 req.Hash,
		From:                 req.From,
		Nonce:                req.Nonce,
		GasLimit:             req.GasLimit,
		MaxFeePerGas:         req.MaxFeePerGas,
		MaxPriorityFeePerGas: req.MaxPriorityFeePerGas,
		Timestamp:            req.Timestamp,
	}

	if err := s.pool.Add(tx); err != nil {
		return AddTxResult{}, err
	}

	return AddTxResult{
		Accepted:     true,
		PendingCount: s.pool.PendingCount(),
		Hash:         req.Hash,
	}, nil
}

func (s *Service) Status() (StatusResult, error) {
	if s == nil || s.pool == nil {
		return StatusResult{}, ErrNilPool
	}

	return StatusResult{
		PendingCount: s.pool.PendingCount(),
		BaseFee:      s.pool.BaseFee(),
	}, nil
}
