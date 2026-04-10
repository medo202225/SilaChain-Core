package blockbuilder

import (
	"errors"

	"silachain/internal/consensus/txpool"
)

var (
	ErrNilPool      = errors.New("blockbuilder: nil pool")
	ErrZeroGasLimit = errors.New("blockbuilder: zero gas limit")
)

type Builder struct {
	GasLimit uint64
}

type Result struct {
	Transactions []txpool.Tx
	GasUsed      uint64
	TotalTipFees uint64
}

func New(gasLimit uint64) (*Builder, error) {
	if gasLimit == 0 {
		return nil, ErrZeroGasLimit
	}
	return &Builder{GasLimit: gasLimit}, nil
}

func (b *Builder) Build(pool *txpool.Pool) (Result, error) {
	if pool == nil {
		return Result{}, ErrNilPool
	}
	if b == nil || b.GasLimit == 0 {
		return Result{}, ErrZeroGasLimit
	}

	ordered := pool.Ordered()
	out := Result{
		Transactions: make([]txpool.Tx, 0, len(ordered)),
	}

	baseFee := pool.BaseFee()

	for _, tx := range ordered {
		if tx.GasLimit == 0 {
			continue
		}
		if out.GasUsed+tx.GasLimit > b.GasLimit {
			continue
		}

		out.Transactions = append(out.Transactions, tx)
		out.GasUsed += tx.GasLimit

		effectiveFee := tx.EffectiveFee(baseFee)
		if effectiveFee > baseFee {
			out.TotalTipFees += (effectiveFee - baseFee) * tx.GasLimit
		}
	}

	return out, nil
}
