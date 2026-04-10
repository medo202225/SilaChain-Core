package miner

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"silachain/internal/consensus/blockassembly"
)

var (
	ErrEmptyParentHash    = errors.New("miner: empty parent hash")
	ErrEmptyFeeRecipient  = errors.New("miner: empty fee recipient")
	ErrEmptyRandom        = errors.New("miner: empty random")
	ErrEmptyStateRoot     = errors.New("miner: empty state root")
	ErrParentHashMismatch = errors.New("miner: parent hash mismatch")
	ErrInvalidBlockNumber = errors.New("miner: invalid block number")
)

type PayloadVersion byte

type PayloadID [8]byte

func (id PayloadID) String() string {
	return hex.EncodeToString(id[:])
}

type BuildPayloadArgs struct {
	ParentHash   string
	Timestamp    uint64
	FeeRecipient string
	Random       string
	GasLimit     uint64
	Version      PayloadVersion
}

func (args BuildPayloadArgs) Validate() error {
	if args.ParentHash == "" {
		return ErrEmptyParentHash
	}
	if args.FeeRecipient == "" {
		return ErrEmptyFeeRecipient
	}
	if args.Random == "" {
		return ErrEmptyRandom
	}
	return nil
}

func (args BuildPayloadArgs) ID() PayloadID {
	h := sha256.New()
	_, _ = h.Write([]byte(args.ParentHash))
	_ = binary.Write(h, binary.BigEndian, args.Timestamp)
	_, _ = h.Write([]byte(args.FeeRecipient))
	_, _ = h.Write([]byte(args.Random))
	_ = binary.Write(h, binary.BigEndian, args.GasLimit)

	var out PayloadID
	copy(out[:], h.Sum(nil)[:8])
	out[0] = byte(args.Version)
	return out
}

type ExecutableData struct {
	PayloadID       string
	BlockNumber     uint64
	BlockHash       string
	ParentHash      string
	ParentStateRoot string
	StateRoot       string
	BaseFee         uint64
	GasLimit        uint64
	GasUsed         uint64
	TxCount         int
}

type Payload struct {
	id       PayloadID
	args     BuildPayloadArgs
	envelope ExecutableData
	lock     sync.Mutex
}

func NewPayload(args BuildPayloadArgs, built blockassembly.Result, stateRoot string) (*Payload, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}
	if stateRoot == "" {
		return nil, ErrEmptyStateRoot
	}
	if built.ParentHash == "" {
		return nil, ErrEmptyParentHash
	}
	if built.ParentHash != args.ParentHash {
		return nil, ErrParentHashMismatch
	}
	if built.BlockNumber != built.ParentNumber+1 {
		return nil, fmt.Errorf("%w: parent=%d block=%d", ErrInvalidBlockNumber, built.ParentNumber, built.BlockNumber)
	}

	id := args.ID()
	blockHash := derivePayloadBlockHash(built)

	return &Payload{
		id:   id,
		args: args,
		envelope: ExecutableData{
			PayloadID:       id.String(),
			BlockNumber:     built.BlockNumber,
			BlockHash:       blockHash,
			ParentHash:      built.ParentHash,
			ParentStateRoot: built.ParentStateRoot,
			StateRoot:       stateRoot,
			BaseFee:         built.BaseFee,
			GasLimit:        built.GasLimit,
			GasUsed:         built.Selection.GasUsed,
			TxCount:         len(built.Selection.Transactions),
		},
	}, nil
}

func (p *Payload) ID() PayloadID {
	if p == nil {
		return PayloadID{}
	}
	return p.id
}

func (p *Payload) Resolve() ExecutableData {
	if p == nil {
		return ExecutableData{}
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.envelope
}

func derivePayloadBlockHash(built blockassembly.Result) string {
	return fmt.Sprintf(
		"sila-block-%d-%s-%d",
		built.BlockNumber,
		built.ParentHash,
		len(built.Selection.Transactions),
	)
}
