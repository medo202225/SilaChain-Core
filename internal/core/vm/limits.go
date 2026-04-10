package vm

const (
	DefaultMaxStackDepth   uint16 = 1024
	DefaultMaxCallDepth    uint16 = 1024
	DefaultMaxCodeSize     uint64 = 24 * 1024
	DefaultMaxInitCodeSize uint64 = 48 * 1024
	DefaultMaxMemorySize   uint64 = 16 * 1024 * 1024
)

type Limits struct {
	MaxStackDepth   uint16
	MaxCallDepth    uint16
	MaxCodeSize     uint64
	MaxInitCodeSize uint64
	MaxMemorySize   uint64
}

func DefaultLimits() Limits {
	return Limits{
		MaxStackDepth:   DefaultMaxStackDepth,
		MaxCallDepth:    DefaultMaxCallDepth,
		MaxCodeSize:     DefaultMaxCodeSize,
		MaxInitCodeSize: DefaultMaxInitCodeSize,
		MaxMemorySize:   DefaultMaxMemorySize,
	}
}
