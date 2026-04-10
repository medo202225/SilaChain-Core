package staking

type RewardRegistry struct {
	items []Reward
}

func NewRewardRegistry() *RewardRegistry {
	return &RewardRegistry{
		items: []Reward{},
	}
}

func (r *RewardRegistry) Add(item Reward) {
	if r == nil {
		return
	}
	r.items = append(r.items, item)
}

func (r *RewardRegistry) Load(items []Reward) {
	if r == nil {
		return
	}
	r.items = make([]Reward, len(items))
	copy(r.items, items)
}

func (r *RewardRegistry) All() []Reward {
	if r == nil {
		return nil
	}
	out := make([]Reward, len(r.items))
	copy(out, r.items)
	return out
}
