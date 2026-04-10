package main

import (
	"log"
	"os"

	"silachain/internal/execution"
)

func main() {
	// Execution currently starts as a standalone canonical service.
	// Cross-layer restart-stable wiring into consensus runtime is not composed here yet.
	// A higher canonical composition root is still required for execution+consensus reader injection.

	if err := execution.RunExecution(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
