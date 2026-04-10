package execution

import (
	"context"
	"time"

	"silachain/internal/app"
	"silachain/internal/chain"
)

func consensusBlockSync(blockchain *chain.Blockchain, peers []string, selfURL string) *app.BlockSyncService {
	return app.NewBlockSyncService(
		blockchain,
		peers,
		selfURL,
		3*time.Second,
	)
}

func startConsensusBlockSync(ctx context.Context, syncer *app.BlockSyncService) {
	if syncer == nil {
		return
	}
	go syncer.Start(ctx)
}
