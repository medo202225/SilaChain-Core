# SilaChain Security Baseline

## Current controls
- RPC method enforcement
- JSON content-type enforcement
- request body size limits
- per-endpoint rate limiting
- local-only protection for sensitive endpoints
- admin token middleware support
- mempool anti-spam limits
- stronger block and transaction validation
- atomic storage writes
- temp file cleanup on startup
- corrupt JSON detection in storage

## Sensitive endpoints
Sensitive endpoints must not be exposed publicly:
- /mine
- /faucet
- any future admin endpoint

## Validator key policy
Validator keys must:
- be stored in a dedicated key file
- never be printed in logs
- never be committed to git
- be backed up offline
- use restricted file permissions

## Startup policy
Node startup should fail closed when:
- validator key file is invalid
- validator address does not match configured validator identity
- critical chain data is corrupt

## Next hardening targets
- restart/recovery integration tests
- replay testing across nodes
- peer reputation and abuse handling
- admin token rotation policy
