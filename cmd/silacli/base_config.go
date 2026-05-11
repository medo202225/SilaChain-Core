package silacli

import (
	"github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/metrics"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

// DefaultExecutionConfig returns the shared execution defaults.
func DefaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		Eth:     ethconfig.Defaults,
		Node:    DefaultNodeConfig(),
		Metrics: metrics.DefaultConfig,
	}
}

// LoadBaseConfig loads the shared execution configuration
// from defaults, TOML config and CLI flags.
func LoadBaseConfig(
	ctx *cli.Context,
	configFile string,
	applyNode func(*cli.Context, *node.Config),
) ExecutionConfig {
	cfg := DefaultExecutionConfig()

	LoadConfigOrFatal(configFile, &cfg)

	applyNode(ctx, &cfg.Node)

	return cfg
}
