package main

import (
	"fmt"
	"log"

	execstate "silachain/internal/consensus/executionstate"
	"silachain/internal/consensus/p2p"
)

func main() {
	cfg, err := p2p.LoadConfig("config/networks/mainnet/consensus/p2p.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if err := cfg.EnsurePaths(); err != nil {
		log.Fatal(err)
	}

	id, err := p2p.LoadOrCreateIdentity(cfg.KeyFile)
	if err != nil {
		log.Fatal(err)
	}

	canonical, err := p2p.BuildCanonicalENR(cfg, id)
	if err != nil {
		log.Fatal(err)
	}
	defer canonical.DB.Close()

	state := execstate.NewState(cfg.GenesisHash)

	silaDiscovery, err := p2p.StartSilaDiscovery(cfg, id, canonical)
	if err != nil {
		log.Fatal(err)
	}
	defer silaDiscovery.Close()

	execTransport, err := p2p.StartSilaExecutionTransport(cfg, id, canonical, state, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer execTransport.Stop()

	silaText := ""
	if canonical.Sila != nil {
		silaText, err = canonical.Sila.Text()
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("peer_id:", id.PeerID)
	fmt.Println("public_key_hex:", id.PublicKeyHex)
	fmt.Println("key_file:", cfg.KeyFile)
	fmt.Println("execution_network_id:", cfg.ExecutionNetworkID)
	fmt.Println("genesis_hash:", cfg.GenesisHash)
	fmt.Println("sila_enr_node:", silaText)
	fmt.Println("canonical_enr_node:", canonical.Text)
	fmt.Println("canonical_enr_db:", canonical.DBPath)
	fmt.Println("sila_discovery_self:", silaDiscovery.SelfText())
	fmt.Println("sila_execution_protocol_name:", execTransport.Name())
	fmt.Println("sila_execution_transport_addr:", execTransport.SelfAddr())
	fmt.Println("execution_state_head_number:", execTransport.StateHeadNumber())
	fmt.Println("execution_state_pending_count:", execTransport.StatePendingCount())
}
