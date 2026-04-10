package state

import "silachain/internal/core/types"

type Executor struct {
	transition *Transition
}

func NewExecutor(transition *Transition) *Executor {
	return &Executor{
		transition: transition,
	}
}

func (e *Executor) Execute(transaction *types.Transaction) error {
	return e.transition.ApplyTransaction(transaction)
}

func (e *Executor) ExecuteWithResult(transaction *types.Transaction) (Result, error) {
	if transaction == nil {
		return FailedResult("", "", "", 0, 0, 0, 0, 0, nil, "", "", "", 0, types.ErrNilTransaction), types.ErrNilTransaction
	}

	return e.transition.ApplyTransactionWithResult(transaction)
}
