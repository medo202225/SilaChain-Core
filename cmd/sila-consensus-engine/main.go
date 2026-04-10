package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"silachain/internal/consensus/blockassembly"
	"silachain/internal/consensus/engineapi"
	consensusruntime "silachain/internal/consensus/runtime"
	"silachain/internal/execution"
)

func main() {
	execSvc, err := execution.NewService(os.Args[1:])
	if err != nil {
		log.Fatalf("create execution service: %v", err)
	}

	cfg := consensusruntime.Config{
		ListenAddress: "127.0.0.1:8552",
		GasLimit:      30000000,
		GenesisHead: blockassembly.Head{
			Number:    0,
			Hash:      "sila-consensus-genesis",
			StateRoot: "sila-consensus-state-0",
			BaseFee:   1,
		},
	}

	rt, err := consensusruntime.New(cfg)
	if err != nil {
		log.Fatalf("create consensus runtime: %v", err)
	}

	rt.SetReceiptReader(execSvc.Blockchain())

	metadataPath := filepath.Join("runtime", "consensus", "engine", "payload-metadata.json")

	if snapshot, err := engineapi.LoadPayloadMetadata(metadataPath); err == nil {
		rt.API().RestorePayloadMetadata(snapshot)
		log.Printf("sila consensus runtime payload metadata restored path=%s entries=%d", metadataPath, len(snapshot.Metadata))
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load payload metadata: %v", err)
	}

	errCh := make(chan error, 2)

	go func() {
		errCh <- execSvc.Start()
	}()

	go func() {
		errCh <- rt.Start()
	}()

	log.Printf("sila execution service composed into consensus engine runtime")
	log.Printf("sila consensus engine api listening on http://%s", cfg.ListenAddress)
	log.Printf("sila consensus runtime receipt reader wired from execution blockchain")
	log.Printf("sila consensus runtime payload metadata path=%s", metadataPath)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("shutdown signal received: %s", sig.String())
	case err := <-errCh:
		if err != nil {
			log.Fatalf("composed service stopped: %v", err)
		}
		return
	}

	if err := engineapi.SavePayloadMetadata(metadataPath, rt.API().SnapshotPayloadMetadata()); err != nil {
		log.Fatalf("save payload metadata: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rt.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown runtime: %v", err)
	}
}
