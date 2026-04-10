# Sila Mainnet Hardening v1

## Status
Approved

## Purpose
This document defines the hardening requirements before declaring Sila Chain production-grade mainnet infrastructure.

## Hardening Goals

- deterministic execution under sustained load
- stable node restart and recovery behavior
- safe transaction validation under adversarial input
- safe contract execution under gas and state stress
- predictable RPC behavior under high request volume
- clear operational controls for validator and node operators

## Hardening Tracks

### 1) Consensus and Chain Safety
- verify block validation under malformed block scenarios
- verify duplicate transaction handling
- verify invalid parent / invalid height rejection
- verify state root / receipt root integrity across long runs
- verify proposer rotation and validator set stability

### 2) Execution and VM Safety
- stress test CREATE / CALL / STATICCALL paths
- verify revert isolation and rollback integrity
- verify gas exhaustion behavior
- verify code size limits
- verify log topic indexing under heavy execution

### 3) RPC Hardening
- enforce request body limits
- confirm rate limiter coverage for write-heavy endpoints
- verify admin protection on sensitive endpoints
- add explorer-safe read endpoints only
- verify malformed JSON and unknown field rejection

### 4) Persistence and Recovery
- verify node restart from persisted state
- verify mempool recovery policy
- verify receipt availability after restart
- verify chain index consistency after restart
- verify behavior after partial disk corruption scenarios where possible

### 5) Validator / Staking Operations
- verify staking state consistency
- verify delegation and undelegation accounting
- verify reward accounting and withdrawal correctness
- verify jail / unjail / slash paths
- verify pending unbond claim logic

### 6) Operational Mainnet Readiness
- document validator minimum hardware
- document backup procedures
- document key handling and rotation procedures
- document upgrade rollout procedure
- document incident response procedure

## Test Targets Before Public Mainnet
- long-running chain soak test
- high-volume tx submission test
- repeated contract deploy/call test
- restart-and-recover test
- validator reward accounting test
- explorer/API read consistency test

## Required Outputs
- test report
- known limitations list
- operator runbook
- validator checklist
- release checklist

## Immediate Work Items
- add mainnet hardening checklist to docs
- define minimum production validator baseline
- document security assumptions
- add restart/recovery test plan
- add load test plan for tx, contracts, and logs

## Exit Criteria
Sila mainnet should only be declared production-ready when:
- core packages pass automated tests consistently
- recovery tests succeed
- load tests complete without state divergence
- economic policy is frozen and published
- validator and operator procedures are documented

