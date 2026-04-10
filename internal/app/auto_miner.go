package app

type AutoMiner struct{}

func NewAutoMiner(_ any, _ any) *AutoMiner {
	return &AutoMiner{}
}

func (m *AutoMiner) Start() {}
