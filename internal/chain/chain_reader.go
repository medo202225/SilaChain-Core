package chain

import (
	"sort"

	"silachain/internal/accounts"
	coretypes "silachain/internal/core/types"
	"silachain/internal/staking"
	"silachain/internal/validator"
	pkgtypes "silachain/pkg/types"
)

func (bc *Blockchain) GetBlockByHeight(height uint64) (*coretypes.Block, bool) {
	if bc == nil {
		return nil, false
	}
	if int(height) >= len(bc.blocks) {
		return nil, false
	}
	return bc.blocks[height], true
}

func (bc *Blockchain) AllAccounts() map[string]*accounts.Account {
	raw := bc.accounts.All()
	out := make(map[string]*accounts.Account, len(raw))
	for addr, acc := range raw {
		out[string(addr)] = acc
	}
	return out
}

func (bc *Blockchain) GetTransactionByHash(hash string) (*coretypes.Transaction, pkgtypes.Height, bool) {
	if bc == nil {
		return nil, 0, false
	}

	loc, ok := bc.txIndex[hash]
	if !ok {
		return nil, 0, false
	}

	if int(loc.BlockHeight) >= len(bc.blocks) {
		return nil, 0, false
	}

	b := bc.blocks[loc.BlockHeight]
	if loc.TxIndex < 0 || loc.TxIndex >= len(b.Transactions) {
		return nil, 0, false
	}

	return &b.Transactions[loc.TxIndex], b.Header.Height, true
}

func (bc *Blockchain) GetReceiptByHash(hash string) (*coretypes.Receipt, bool) {
	if bc == nil {
		return nil, false
	}
	r, ok := bc.receipts[hash]
	if !ok {
		return nil, false
	}
	return &r, true
}

func (bc *Blockchain) CurrentProposer() (pkgtypes.Address, bool) {
	return "", false
}

func (bc *Blockchain) RotationState() (int, uint64) {
	return 0, 0
}

func (bc *Blockchain) Validators() []validator.Member {
	if bc == nil || bc.validatorReg == nil {
		return nil
	}

	members := bc.validatorReg.Members()
	for i := range members {
		selfStake := uint64(0)
		if bc.stakingReg != nil {
			if st, ok := bc.stakingReg.Get(members[i].Address); ok {
				selfStake = st.Stake
			}
		}
		delegatedStake := uint64(0)
		if bc.delegationReg != nil {
			delegatedStake = bc.delegationReg.TotalForValidator(members[i].Address)
		}
		members[i].Stake = selfStake + delegatedStake
	}
	return members
}

func (bc *Blockchain) ActiveValidators() []validator.Member {
	if bc == nil {
		return nil
	}
	out := make([]validator.Member, len(bc.activeValidators))
	copy(out, bc.activeValidators)
	return out
}

func (bc *Blockchain) WeightedValidators() []validator.Member {
	if bc == nil {
		return nil
	}
	out := make([]validator.Member, len(bc.weightedValidators))
	copy(out, bc.weightedValidators)
	return out
}

func (bc *Blockchain) Stakes() []staking.Entry {
	if bc == nil || bc.stakingReg == nil {
		return nil
	}
	return bc.stakingReg.All()
}

func (bc *Blockchain) Delegations() []staking.Delegation {
	if bc == nil || bc.delegationReg == nil {
		return nil
	}
	return bc.delegationReg.All()
}

func (bc *Blockchain) Undelegations() []staking.Undelegation {
	if bc == nil || bc.undelegationReg == nil {
		return nil
	}
	return bc.undelegationReg.All()
}

func (bc *Blockchain) Slashes() []staking.Slash {
	if bc == nil || bc.slashReg == nil {
		return nil
	}
	return bc.slashReg.All()
}

func (bc *Blockchain) Rewards() []staking.Reward {
	if bc == nil || bc.rewardReg == nil {
		return nil
	}
	return bc.rewardReg.All()
}

func (bc *Blockchain) DelegatorRewards() []staking.DelegatorReward {
	if bc == nil || bc.delegatorRewardReg == nil {
		return nil
	}
	return bc.delegatorRewardReg.All()
}

func (bc *Blockchain) Withdrawals() []staking.Withdrawal {
	if bc == nil || bc.withdrawalReg == nil {
		return nil
	}
	return bc.withdrawalReg.All()
}

func (bc *Blockchain) Jails() []staking.Jail {
	if bc == nil || bc.jailReg == nil {
		return nil
	}
	return bc.jailReg.All()
}

func (bc *Blockchain) UnbondClaims() []staking.UnbondClaim {
	if bc == nil || bc.unbondClaimReg == nil {
		return nil
	}
	return bc.unbondClaimReg.All()
}

func (bc *Blockchain) PendingUnbond(address pkgtypes.Address) uint64 {
	if bc == nil {
		return 0
	}

	height, err := bc.Height()
	if err != nil {
		return 0
	}

	var accrued uint64
	if bc.undelegationReg != nil {
		for _, u := range bc.undelegationReg.All() {
			if u.Delegator == address && uint64(height) >= u.UnlockHeight {
				accrued += u.Amount
			}
		}
	}

	var claimed uint64
	if bc.unbondClaimReg != nil {
		claimed = bc.unbondClaimReg.TotalForAddress(address)
	}

	if claimed >= accrued {
		return 0
	}
	return accrued - claimed
}

func (bc *Blockchain) PendingRewards(address pkgtypes.Address) uint64 {
	if bc == nil {
		return 0
	}

	var accrued uint64
	if bc.rewardReg != nil {
		for _, r := range bc.rewardReg.All() {
			if r.Validator == address {
				accrued += r.Amount
			}
		}
	}
	if bc.delegatorRewardReg != nil {
		for _, r := range bc.delegatorRewardReg.All() {
			if r.Delegator == address {
				accrued += r.Amount
			}
		}
	}

	var withdrawn uint64
	if bc.withdrawalReg != nil {
		withdrawn = bc.withdrawalReg.TotalForAddress(address)
	}

	if withdrawn >= accrued {
		return 0
	}
	return accrued - withdrawn
}

func (bc *Blockchain) DefaultProposer() (pkgtypes.Address, bool) {
	if bc == nil {
		return "", false
	}

	all := bc.accounts.All()
	if len(all) == 0 {
		return "", false
	}

	keys := make([]string, 0, len(all))
	for addr := range all {
		keys = append(keys, string(addr))
	}
	sort.Strings(keys)

	return pkgtypes.Address(keys[0]), true
}

type MonetaryMetrics struct {
	TotalSupply     uint64 `json:"total_supply"`
	StakedSupply    uint64 `json:"staked_supply"`
	DelegatedSupply uint64 `json:"delegated_supply"`
	PendingRewards  uint64 `json:"pending_rewards"`
	SlashedAmount   uint64 `json:"slashed_amount"`
}

func (bc *Blockchain) MonetaryMetrics() MonetaryMetrics {
	if bc == nil {
		return MonetaryMetrics{}
	}

	var totalSupply uint64
	for _, acc := range bc.accounts.All() {
		totalSupply += uint64(acc.Balance)
	}

	var stakedSupply uint64
	for _, s := range bc.Stakes() {
		stakedSupply += s.Stake
	}

	var delegatedSupply uint64
	for _, d := range bc.Delegations() {
		delegatedSupply += d.Amount
	}

	var slashedAmount uint64
	for _, s := range bc.Slashes() {
		slashedAmount += s.Amount
	}

	var pendingRewards uint64
	for _, acc := range bc.accounts.All() {
		pendingRewards += bc.PendingRewards(acc.Address)
	}

	return MonetaryMetrics{
		TotalSupply:     totalSupply,
		StakedSupply:    stakedSupply,
		DelegatedSupply: delegatedSupply,
		PendingRewards:  pendingRewards,
		SlashedAmount:   slashedAmount,
	}
}
