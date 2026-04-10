package staking

type DelegatorRewardRegistry struct {
	items []DelegatorReward
}

func NewDelegatorRewardRegistry() *DelegatorRewardRegistry {
	return &DelegatorRewardRegistry{
		items: []DelegatorReward{},
	}
}

func (r *DelegatorRewardRegistry) Add(item DelegatorReward) {
	if r == nil {
		return
	}
	r.items = append(r.items, item)
}

func (r *DelegatorRewardRegistry) Load(items []DelegatorReward) {
	if r == nil {
		return
	}
	r.items = make([]DelegatorReward, len(items))
	copy(r.items, items)
}

func (r *DelegatorRewardRegistry) All() []DelegatorReward {
	if r == nil {
		return nil
	}
	out := make([]DelegatorReward, len(r.items))
	copy(out, r.items)
	return out
}
