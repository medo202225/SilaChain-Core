package state

type AccessTuple struct {
	Address     string
	StorageKeys []string
}

type AccessList struct {
	addresses map[string]struct{}
	slots     map[string]map[string]struct{}
}

func NewAccessList() *AccessList {
	return &AccessList{
		addresses: make(map[string]struct{}),
		slots:     make(map[string]map[string]struct{}),
	}
}

func (al *AccessList) AddAddress(address string) {
	if al == nil || address == "" {
		return
	}
	al.addresses[address] = struct{}{}
}

func (al *AccessList) AddSlot(address, slot string) {
	if al == nil || address == "" || slot == "" {
		return
	}
	al.AddAddress(address)
	if al.slots[address] == nil {
		al.slots[address] = make(map[string]struct{})
	}
	al.slots[address][slot] = struct{}{}
}

func (al *AccessList) ContainsAddress(address string) bool {
	if al == nil {
		return false
	}
	_, ok := al.addresses[address]
	return ok
}

func (al *AccessList) Contains(address, slot string) (bool, bool) {
	if al == nil {
		return false, false
	}
	_, addressOK := al.addresses[address]
	if !addressOK {
		return false, false
	}
	if slot == "" {
		return true, false
	}
	slots := al.slots[address]
	if slots == nil {
		return true, false
	}
	_, slotOK := slots[slot]
	return true, slotOK
}

func (al *AccessList) DeleteAddress(address string) {
	if al == nil {
		return
	}
	delete(al.addresses, address)
	delete(al.slots, address)
}

func (al *AccessList) DeleteSlot(address, slot string) {
	if al == nil {
		return
	}
	slots := al.slots[address]
	if slots == nil {
		return
	}
	delete(slots, slot)
	if len(slots) == 0 {
		delete(al.slots, address)
	}
}

func (al *AccessList) Copy() *AccessList {
	if al == nil {
		return NewAccessList()
	}
	out := NewAccessList()
	for address := range al.addresses {
		out.addresses[address] = struct{}{}
	}
	for address, slots := range al.slots {
		dst := make(map[string]struct{}, len(slots))
		for slot := range slots {
			dst[slot] = struct{}{}
		}
		out.slots[address] = dst
	}
	return out
}
