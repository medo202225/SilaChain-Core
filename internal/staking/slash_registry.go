package staking

type SlashRegistry struct {
	items []Slash
}

func NewSlashRegistry() *SlashRegistry {
	return &SlashRegistry{
		items: []Slash{},
	}
}

func (r *SlashRegistry) Add(s Slash) {
	if r == nil {
		return
	}
	r.items = append(r.items, s)
}

func (r *SlashRegistry) Load(items []Slash) {
	if r == nil {
		return
	}
	r.items = make([]Slash, len(items))
	copy(r.items, items)
}

func (r *SlashRegistry) All() []Slash {
	if r == nil {
		return nil
	}
	out := make([]Slash, len(r.items))
	copy(out, r.items)
	return out
}
