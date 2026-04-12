package state

type AccessEvents struct {
	accounts map[string]struct{}
	slots    map[string]map[string]struct{}
}

func NewAccessEvents() *AccessEvents {
	return &AccessEvents{
		accounts: make(map[string]struct{}),
		slots:    make(map[string]map[string]struct{}),
	}
}

func (ae *AccessEvents) AddAccount(address string) {
	if ae == nil || address == "" {
		return
	}
	ae.accounts[address] = struct{}{}
}

func (ae *AccessEvents) AddSlot(address, slot string) {
	if ae == nil || address == "" || slot == "" {
		return
	}
	ae.AddAccount(address)
	if ae.slots[address] == nil {
		ae.slots[address] = make(map[string]struct{})
	}
	ae.slots[address][slot] = struct{}{}
}

func (ae *AccessEvents) Reset() {
	if ae == nil {
		return
	}
	ae.accounts = make(map[string]struct{})
	ae.slots = make(map[string]map[string]struct{})
}
