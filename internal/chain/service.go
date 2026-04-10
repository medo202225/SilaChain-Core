package chain

type Service struct {
	blockchain *Blockchain
}

func NewService(blockchain *Blockchain) *Service {
	return &Service{
		blockchain: blockchain,
	}
}

func (s *Service) Blockchain() *Blockchain {
	return s.blockchain
}
