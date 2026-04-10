# SilaChain Devnet Runbook

## Nodes

- node1 -> 127.0.0.1:8090
- node2 -> 127.0.0.1:8091
- node3 -> 127.0.0.1:8092

## Config files

- config/node1.json
- config/node2.json
- config/node3.json

## Devnet config bundle

- config/devnet/bootnodes.json
- config/devnet/validators.json
- config/devnet/node.yaml
- config/devnet/p2p.yaml
- config/devnet/rpc.yaml
- config/devnet/storage.yaml
- config/devnet/mempool.yaml
- config/devnet/metrics.yaml
- config/devnet/consensus.yaml
- config/devnet/genesis.json

## Start

powershell -ExecutionPolicy Bypass -File .\scripts\devnet-start.ps1

## Stop

powershell -ExecutionPolicy Bypass -File .\scripts\devnet-stop.ps1

## Reset

powershell -ExecutionPolicy Bypass -File .\scripts\devnet-reset.ps1

## Expected checks

Invoke-WebRequest http://127.0.0.1:8090/health
Invoke-WebRequest http://127.0.0.1:8091/health
Invoke-WebRequest http://127.0.0.1:8092/health

Invoke-WebRequest http://127.0.0.1:8090/chain/info
Invoke-WebRequest http://127.0.0.1:8091/chain/info
Invoke-WebRequest http://127.0.0.1:8092/chain/info

## Notes

This runbook is for local multi-node devnet only.
