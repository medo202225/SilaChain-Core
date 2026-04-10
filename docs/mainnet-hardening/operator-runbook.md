# Sila Operator Runbook v1

## Status
Approved

## Purpose
This runbook defines the minimum operating procedures for running and maintaining a Sila Chain node in production-like mainnet conditions.

---

## 1) Node Role
A Sila node operator is responsible for:
- starting and monitoring the node
- protecting node keys and environment
- validating correct RPC exposure
- monitoring sync, mempool, and block production behavior
- handling restart and recovery safely
- participating in upgrade and incident procedures

## 2) Minimum Operational Requirements
- stable server or VM environment
- persistent disk storage
- secure access to node host
- controlled firewall rules
- regular backups of node data and keys
- time synchronization on the host

## 3) Startup Procedure
- verify config files exist under `config/networks/mainnet/public`
- verify data directory path is correct
- verify the RPC listen address is intentional
- verify local-only write endpoint policy and admin token policy if admin checks are enabled
- start the node using the production command used by the project
- confirm `/health` returns healthy status
- confirm `/chain/info` returns correct chain metadata
- confirm `/explorer/network` returns network metrics

## 4) Post-Startup Verification
After every startup:
- confirm chain height is readable
- confirm latest hash is readable
- confirm mempool endpoint responds
- confirm explorer summary responds
- confirm monetary policy fields are visible in API
- confirm explorer frontend is reachable if enabled

## 5) RPC Exposure Rules
Recommended exposure model:
- read-only explorer/API endpoints may be exposed intentionally
- write-sensitive endpoints must remain local-only in v1 mainnet operation
- admin token, if enabled, is an additional protection layer and does not replace local-only restrictions
- local-only protection must remain enabled for:
  - contract deploy
  - contract call if privileged
  - faucet
  - mine
  - contract storage mutation endpoints

## 6) Monitoring Checklist
Operators should monitor:
- chain height growth
- mempool count
- RPC response availability
- validator status
- jailed validator state if applicable
- staking and reward metrics
- repeated transaction rejection patterns
- restart/recovery anomalies

## 7) Backup Procedure
Back up regularly:
- node data directory
- validator/operator key material
- relevant config files
- deployment manifests/scripts
- release version reference

Backup rules:
- keep backups encrypted where appropriate
- keep more than one restore point
- test restore process periodically

## 8) Restart Procedure
Before restart:
- verify if restart is planned or emergency
- capture current height and latest hash
- capture current mempool status if relevant
- ensure no active maintenance conflict exists

Restart steps:
- stop the node cleanly
- restart the process
- confirm `/health`
- confirm `/chain/info`
- confirm height continuity
- confirm latest block continuity
- confirm receipts and tx lookup still work

## 9) Upgrade Procedure
Before upgrade:
- record current release version
- back up data and keys
- read upgrade notes
- confirm protocol/config compatibility

During upgrade:
- stop node cleanly
- deploy new binary/config
- restart node
- re-check health and chain info
- re-check explorer/network metrics

After upgrade:
- verify block progression
- verify validator status
- verify contract/read endpoints
- verify monetary policy fields remain correct

## 10) Unsafe Actions To Avoid
- do not expose sensitive write endpoints publicly under any circumstance in v1
- do not overwrite production data without backup
- do not rotate keys without a documented procedure
- do not modify frozen monetary parameters casually
- do not change chain identity values after mainnet freeze

## 11) Escalation Trigger Examples
Escalate when:
- node cannot recover height after restart
- receipts disappear after restart
- tx index inconsistency is observed
- unexpected chain id or monetary policy values appear
- validator rewards or slashes appear inconsistent
- explorer/API becomes inconsistent with chain state

## 12) Operator Sign-Off
Before production participation, operator confirms:
- configs reviewed
- backups configured
- restart procedure understood
- incident procedure understood
- upgrade procedure understood

