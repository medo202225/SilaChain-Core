package chain

import (
	"errors"

	"silachain/internal/accounts"
	block "silachain/internal/block"
	"silachain/internal/core/state"
	coretypes "silachain/internal/core/types"
	"silachain/internal/mempool"
	"silachain/internal/staking"
	"silachain/internal/storage"
	pkgtypes "silachain/pkg/types"
)

var ErrInvalidDelegationAmount = errors.New("invalid delegation amount")
var ErrInvalidSlashAmount = errors.New("invalid slash amount")
var ErrNoPendingRewards = errors.New("no pending rewards")
var ErrInvalidUndelegationAmount = errors.New("invalid undelegation amount")
var ErrNoPendingUnbond = errors.New("no pending unbond")
var ErrKnownTransaction = errors.New("known transaction")
var ErrInvalidProposerTurn = errors.New("invalid proposer turn")

func (bc *Blockchain) AddBlock(b *coretypes.Block) error {
	if err := block.Validate(b); err != nil {
		return err
	}

	latest, err := bc.LatestBlock()
	if err != nil {
		return err
	}

	if b.Header.Height != latest.Header.Height+1 {
		return ErrInvalidHeight
	}

	if b.Header.ParentHash != latest.Header.Hash {
		return ErrInvalidParent
	}

	blockReceipts := make([]coretypes.Receipt, 0, len(b.Transactions))
	var cumulativeGasUsed pkgtypes.Gas

	for i := range b.Transactions {
		t := &b.Transactions[i]

		if err := coretypes.Validate(t); err != nil {
			return err
		}

		execResult, err := bc.executor.ExecuteWithResult(t)
		if err != nil {
			return err
		}

		cumulativeGasUsed += execResult.GasUsed

		gasLimit := t.GasLimit
		if gasLimit == 0 {
			gasLimit = state.TransferIntrinsicGas
		}

		chainReceipt, blockReceipt := buildReceiptsFromExecutionResult(
			execResult,
			b.Header.Hash,
			b.Header.Height,
			uint64(i),
			t.GasPrice,
			gasLimit,
			cumulativeGasUsed,
		)

		bc.receipts[string(t.Hash)] = chainReceipt

		bc.txIndex[string(t.Hash)] = storage.TxLocation{
			BlockHeight: uint64(b.Header.Height),
			TxIndex:     i,
		}

		blockReceipts = append(blockReceipts, blockReceipt)
	}

	b.Receipts = blockReceipts
	b.Header.GasUsed = cumulativeGasUsed

	if err := bc.finalizeBlockRoots(b); err != nil {
		return err
	}

	for i := range b.Transactions {
		t := &b.Transactions[i]

		receipt := bc.receipts[string(t.Hash)]
		receipt.BlockHash = b.Header.Hash
		receipt.BlockHeight = b.Header.Height
		receipt.TransactionIndex = uint64(i)
		bc.receipts[string(t.Hash)] = receipt

		b.Receipts[i].BlockHash = b.Header.Hash
		b.Receipts[i].BlockHeight = b.Header.Height
		b.Receipts[i].TransactionIndex = uint64(i)

		loc := bc.txIndex[string(t.Hash)]
		loc.BlockHeight = uint64(b.Header.Height)
		bc.txIndex[string(t.Hash)] = loc
	}
	if err := bc.finalizeAnchoredReceiptRoots(b); err != nil {
		return err
	}

	if bc.blockReward > 0 {
		commission := (bc.blockReward * bc.validatorCommissionBps) / 10000
		remainingReward := bc.blockReward - commission

		delegations := bc.delegationReg.ForValidator(b.Header.Proposer)
		if len(delegations) == 0 {
			commission = bc.blockReward
			remainingReward = 0
		}

		if st, ok := bc.stakingReg.Get(b.Header.Proposer); ok {
			bc.stakingReg.Set(b.Header.Proposer, st.Stake+commission)
		} else {
			bc.stakingReg.Set(b.Header.Proposer, commission)
		}

		bc.rewardReg.Add(staking.Reward{
			Validator:   b.Header.Proposer,
			BlockHeight: uint64(b.Header.Height),
			Amount:      commission,
			Reason:      "block_proposer_reward",
		})

		if len(delegations) > 0 && remainingReward > 0 {
			var totalDelegated uint64
			for _, d := range delegations {
				totalDelegated += d.Amount
			}

			if totalDelegated > 0 {
				var distributed uint64
				for i, d := range delegations {
					share := uint64(0)
					if i == len(delegations)-1 {
						share = remainingReward - distributed
					} else {
						share = (remainingReward * d.Amount) / totalDelegated
						distributed += share
					}

					bc.delegatorRewardReg.Add(staking.DelegatorReward{
						Delegator:   d.Delegator,
						Validator:   d.Validator,
						BlockHeight: uint64(b.Header.Height),
						Amount:      share,
						Reason:      "delegator_block_reward",
					})
				}
			}
		}
	}

	bc.blocks = append(bc.blocks, b)
	bc.rebuildActiveValidators()
	return bc.persist()
}

func (bc *Blockchain) SubmitTransaction(t *coretypes.Transaction) error {
	if err := coretypes.Validate(t); err != nil {
		return err
	}

	if _, exists := bc.txIndex[string(t.Hash)]; exists {
		return ErrKnownTransaction
	}

	if bc.mempool.HasHash(t.Hash) {
		return mempool.ErrDuplicateTx
	}

	if bc.mempool.HasSenderNonce(t.From, t.Nonce) {
		return mempool.ErrDuplicateSenderNonce
	}

	fromAcc, ok := bc.accounts.Get(t.From)
	if !ok {
		return accounts.ErrAccountNotFound
	}

	expectedNonce := bc.mempool.NextNonceForSender(t.From, fromAcc.Nonce)
	if t.Nonce < expectedNonce {
		return coretypes.ErrInvalidNonce
	}

	if fromAcc.Balance < t.TotalCost() {
		return accounts.ErrInsufficientBalance
	}

	return mempool.ValidateAndAdd(bc.mempool, t, expectedNonce)
}

// MinePending is a local/manual mining helper for chain-owned tests and legacy callers.
// It is not the canonical consensus block-construction path.
// The canonical gas-limit-aware consensus construction path is runtime -> engine -> blockassembly -> blockbuilder.

func (bc *Blockchain) MinePending(proposer pkgtypes.Address) (*coretypes.Block, error) {
	pending := mempool.OrderedPending(bc.mempool)
	if len(pending) == 0 {
		return nil, ErrEmptyMempool
	}

	latest, err := bc.LatestBlock()
	if err != nil {
		return nil, err
	}

	newBlock, err := block.NewBlock(
		latest.Header.Height+1,
		latest.Header.Hash,
		latest.Header.StateRoot,
		"",
		"",
		proposer,
		0,
		0,
		pending,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if err := bc.AddBlock(newBlock); err != nil {
		return nil, err
	}

	// Local/legacy mining path uses full mempool clear after successful block application.
	// Canonical consensus cleanup/pruning is selective and lives in consensus runtime -> txpool.RemoveIncluded(...).
	bc.mempool.Clear()
	return newBlock, nil
}

func (bc *Blockchain) AddDelegation(delegator pkgtypes.Address, validatorAddr pkgtypes.Address, amount uint64) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	if amount == 0 {
		return ErrInvalidDelegationAmount
	}

	if _, err := bc.GetAccount(delegator); err != nil {
		return err
	}

	found := false
	for _, v := range bc.validatorReg.Members() {
		if v.Address == validatorAddr {
			found = true
			break
		}
	}
	if !found {
		return accounts.ErrAccountNotFound
	}

	bc.delegationReg.Set(delegator, validatorAddr, amount)
	bc.rebuildActiveValidators()
	return bc.persist()
}

func (bc *Blockchain) Undelegate(delegator pkgtypes.Address, validatorAddr pkgtypes.Address, amount uint64, reason string) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	if amount == 0 {
		return ErrInvalidUndelegationAmount
	}

	d, ok := bc.delegationReg.Get(delegator, validatorAddr)
	if !ok {
		return accounts.ErrAccountNotFound
	}
	if amount > d.Amount {
		return ErrInvalidUndelegationAmount
	}

	remaining := d.Amount - amount
	if remaining == 0 {
		bc.delegationReg.Delete(delegator, validatorAddr)
	} else {
		bc.delegationReg.Set(delegator, validatorAddr, remaining)
	}

	height, err := bc.Height()
	if err != nil {
		return err
	}

	bc.undelegationReg.Add(staking.Undelegation{
		Delegator:    delegator,
		Validator:    validatorAddr,
		Amount:       amount,
		Reason:       reason,
		UnlockHeight: uint64(height) + bc.unbondingDelay,
	})

	bc.rebuildActiveValidators()
	return bc.persist()
}

func (bc *Blockchain) AddSlash(validatorAddr pkgtypes.Address, amount uint64, reason string) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	if amount == 0 {
		return ErrInvalidSlashAmount
	}

	st, ok := bc.stakingReg.Get(validatorAddr)
	if !ok {
		return accounts.ErrAccountNotFound
	}

	originalStake := st.Stake
	if amount >= st.Stake {
		st.Stake = 0
	} else {
		st.Stake -= amount
	}
	bc.stakingReg.Set(validatorAddr, st.Stake)

	if bc.delegationReg != nil && originalStake > 0 {
		delegations := bc.delegationReg.ForValidator(validatorAddr)
		for _, d := range delegations {
			var newAmount uint64
			if amount >= originalStake {
				newAmount = 0
			} else {
				newAmount = (d.Amount * st.Stake) / originalStake
			}

			if newAmount == 0 {
				bc.delegationReg.Delete(d.Delegator, d.Validator)
			} else {
				bc.delegationReg.Set(d.Delegator, d.Validator, newAmount)
			}
		}
	}

	bc.slashReg.Add(staking.Slash{
		Validator: validatorAddr,
		Amount:    amount,
		Reason:    reason,
	})

	bc.rebuildActiveValidators()
	return bc.persist()
}

func (bc *Blockchain) SetStake(validatorAddr pkgtypes.Address, amount uint64) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	found := false
	for _, v := range bc.validatorReg.Members() {
		if v.Address == validatorAddr {
			found = true
			break
		}
	}
	if !found {
		return accounts.ErrAccountNotFound
	}

	bc.stakingReg.Set(validatorAddr, amount)
	bc.rebuildActiveValidators()
	return bc.persist()
}

func (bc *Blockchain) WithdrawRewards(address pkgtypes.Address) (uint64, error) {
	if bc == nil {
		return 0, ErrNilBlockchain
	}

	acc, err := bc.GetAccount(address)
	if err != nil {
		return 0, err
	}

	pending := bc.PendingRewards(address)
	if pending == 0 {
		return 0, ErrNoPendingRewards
	}

	acc.Credit(pkgtypes.Amount(pending))
	bc.withdrawalReg.Add(staking.Withdrawal{
		Address: address,
		Amount:  pending,
		Reason:  "reward_withdrawal",
	})

	if err := bc.persist(); err != nil {
		return 0, err
	}
	return pending, nil
}

func (bc *Blockchain) ClaimUnbond(address pkgtypes.Address) (uint64, error) {
	if bc == nil {
		return 0, ErrNilBlockchain
	}

	acc, err := bc.GetAccount(address)
	if err != nil {
		return 0, err
	}

	pending := bc.PendingUnbond(address)
	if pending == 0 {
		return 0, ErrNoPendingUnbond
	}

	acc.Credit(pkgtypes.Amount(pending))
	bc.unbondClaimReg.Add(staking.UnbondClaim{
		Address: address,
		Amount:  pending,
		Reason:  "unbond_claim",
	})

	if err := bc.persist(); err != nil {
		return 0, err
	}
	return pending, nil
}

func (bc *Blockchain) JailValidator(validatorAddr pkgtypes.Address, reason string) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	found := false
	for _, v := range bc.validatorReg.Members() {
		if v.Address == validatorAddr {
			found = true
			break
		}
	}
	if !found {
		return accounts.ErrAccountNotFound
	}

	bc.jailReg.Set(validatorAddr, true, reason)
	bc.rebuildActiveValidators()
	return bc.persist()
}

func (bc *Blockchain) UnjailValidator(validatorAddr pkgtypes.Address, reason string) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	found := false
	for _, v := range bc.validatorReg.Members() {
		if v.Address == validatorAddr {
			found = true
			break
		}
	}
	if !found {
		return accounts.ErrAccountNotFound
	}

	bc.jailReg.Set(validatorAddr, false, reason)
	bc.rebuildActiveValidators()
	return bc.persist()
}

func buildReceiptsFromExecutionResult(
	execResult state.Result,
	blockHash pkgtypes.Hash,
	blockHeight pkgtypes.Height,
	txIndex uint64,
	gasPrice pkgtypes.Amount,
	gasLimit pkgtypes.Gas,
	cumulativeGasUsed pkgtypes.Gas,
) (coretypes.Receipt, coretypes.Receipt) {
	chainReceipt := coretypes.Receipt{
		TxHash:            execResult.TxHash,
		BlockHash:         blockHash,
		BlockHeight:       blockHeight,
		TransactionIndex:  txIndex,
		From:              execResult.From,
		To:                execResult.To,
		Value:             execResult.Value,
		Fee:               execResult.Fee,
		GasPrice:          gasPrice,
		GasLimit:          gasLimit,
		GasUsed:           execResult.GasUsed,
		CumulativeGasUsed: cumulativeGasUsed,
		EffectiveFee:      execResult.Fee,
		Success:           execResult.Success,
		Error:             execResult.Error,
		Logs:              execResult.Logs,
		ReturnData:        execResult.ReturnData,
		RevertData:        execResult.RevertData,
		CreatedAddress:    execResult.CreatedAddress,
		Timestamp:         execResult.Timestamp,
	}

	blockReceipt := coretypes.Receipt{
		TxHash:            execResult.TxHash,
		BlockHash:         blockHash,
		BlockHeight:       blockHeight,
		TransactionIndex:  txIndex,
		Success:           execResult.Success,
		GasUsed:           execResult.GasUsed,
		CumulativeGasUsed: cumulativeGasUsed,
		Error:             execResult.Error,
		ReturnData:        execResult.ReturnData,
		RevertData:        execResult.RevertData,
		CreatedAddress:    execResult.CreatedAddress,
	}

	return chainReceipt, blockReceipt
}
