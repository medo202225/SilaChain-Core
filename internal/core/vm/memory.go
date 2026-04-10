package vm

type Memory struct {
	data    []byte
	maxSize uint64
}

func NewMemory() *Memory {
	return &Memory{
		data:    make([]byte, 0),
		maxSize: DefaultMaxMemorySize,
	}
}

func NewMemoryWithLimit(maxSize uint64) *Memory {
	if maxSize == 0 {
		maxSize = DefaultMaxMemorySize
	}

	return &Memory{
		data:    make([]byte, 0),
		maxSize: maxSize,
	}
}

func (m *Memory) Len() uint64 {
	return uint64(len(m.data))
}

func (m *Memory) Bytes() []byte {
	return cloneBytes(m.data)
}

func (m *Memory) EnsureSize(size uint64) error {
	if size <= uint64(len(m.data)) {
		return nil
	}

	if size > m.maxSize {
		return ErrMemoryLimitExceeded
	}

	if size > uint64(^uint(0)>>1) {
		return ErrMemoryLimitExceeded
	}

	next := make([]byte, size)
	copy(next, m.data)
	m.data = next
	return nil
}

func (m *Memory) Store(offset uint64, value []byte) error {
	end := offset + uint64(len(value))
	if end < offset {
		return ErrMemoryLimitExceeded
	}
	if err := m.EnsureSize(end); err != nil {
		return err
	}
	copy(m.data[offset:end], value)
	return nil
}

func (m *Memory) Load(offset, size uint64) ([]byte, error) {
	end := offset + size
	if end < offset {
		return nil, ErrMemoryLimitExceeded
	}
	if err := m.EnsureSize(end); err != nil {
		return nil, err
	}
	out := make([]byte, size)
	copy(out, m.data[offset:end])
	return out, nil
}
