# Sila Incident and Rollback Procedures v1

## Status
Approved

## Purpose
This document defines the minimum incident response and rollback procedures for Sila Chain production-like mainnet operations.

---

## 1) Incident Categories
Typical incident classes:
- node unavailable
- RPC unavailable
- chain height stuck
- data inconsistency after restart
- receipt/index inconsistency
- contract execution anomaly
- validator reward/slash accounting anomaly
- configuration mismatch
- unintended endpoint exposure

## 2) First Response Rules
When an incident is detected:
- do not panic-change configs blindly
- record current symptoms
- capture height, latest hash, and endpoint status
- identify whether incident is local, validator-specific, or network-wide
- preserve logs and evidence before destructive actions

## 3) Immediate Triage Checklist
Capture:
- current chain height
- latest block hash
- mempool count
- `/health` response
- `/chain/info` response
- `/explorer/network` response
- affected tx hash or contract address if relevant
- whether restart already occurred

## 4) Incident Severity Levels

### Severity 1
- explorer/read issue only
- no state corruption indicated
- node still advancing

### Severity 2
- RPC unavailable
- block production stalled locally
- restart needed
- no evidence of state corruption yet

### Severity 3
- receipt/index inconsistency
- contract state inconsistency
- repeated restart failure
- possible data corruption
- possible config mismatch affecting consensus/economics

## 5) Restart Response
If issue appears restart-recoverable:
- record pre-restart height/hash
- stop node cleanly
- restart process
- verify post-restart height/hash continuity
- verify tx lookup and receipt lookup
- verify explorer/network consistency

## 6) Rollback Preconditions
Do NOT perform rollback unless:
- backups exist
- incident evidence is preserved
- restart-only recovery failed
- operator understands the recovery target state
- release/version mismatch has been evaluated

## 7) Rollback Procedure
- stop node
- isolate current data directory
- preserve a forensic copy if possible
- restore the last known-good backup
- restart with the intended release/config set
- verify health, chain info, height continuity, receipts, and critical explorer/API views
- verify monetary policy values remain frozen and correct

## 8) Config Rollback Procedure
If issue is caused by bad config:
- restore previous known-good config
- verify chain id and monetary policy values
- verify validator and staking parameters
- restart node
- re-check health and explorer/API outputs

## 9) Upgrade Rollback Procedure
If issue follows upgrade:
- stop node
- restore previous binary/build
- restore previous config if needed
- restart node
- verify chain continuity and endpoint behavior
- document the failed upgrade state

## 10) Communication Rules
During incident handling:
- keep a timestamped incident log
- record every operator action
- communicate severity clearly
- do not claim full recovery until checks pass
- document whether rollback occurred

## 11) Recovery Validation Checklist
After incident or rollback:
- `/health` healthy
- `/chain/info` correct
- chain height readable
- latest hash readable
- tx lookup works
- receipt lookup works
- contract explorer works
- logs explorer works
- monetary metrics visible
- frozen monetary policy flags remain correct

## 12) Post-Incident Review
After recovery:
- write incident summary
- identify root cause
- record exact remediation steps
- record whether docs/checklists need updates
- record whether new automated tests are needed

## 13) Final Operator Sign-Off
Before closing incident:
- recovery validated
- rollback state documented if used
- operator review complete
- release/ops owner informed

