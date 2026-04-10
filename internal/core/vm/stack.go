package vm

import "math/big"

type Stack struct {
	maxDepth uint16
	data     []*big.Int
}

func NewStack(maxDepth uint16) *Stack {
	return &Stack{
		maxDepth: maxDepth,
		data:     make([]*big.Int, 0, maxDepth),
	}
}

func (s *Stack) Len() int {
	return len(s.data)
}

func (s *Stack) Push(v *big.Int) error {
	if uint16(len(s.data)) >= s.maxDepth {
		return ErrStackOverflow
	}
	if v == nil {
		v = new(big.Int)
	}
	s.data = append(s.data, new(big.Int).Set(v))
	return nil
}

func (s *Stack) Pop() (*big.Int, error) {
	if len(s.data) == 0 {
		return nil, ErrStackUnderflow
	}
	last := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return new(big.Int).Set(last), nil
}

func (s *Stack) Peek() (*big.Int, error) {
	if len(s.data) == 0 {
		return nil, ErrStackUnderflow
	}
	return new(big.Int).Set(s.data[len(s.data)-1]), nil
}
