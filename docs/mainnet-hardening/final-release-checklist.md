# Sila Final Release Checklist v1

## Status
Approved

## Objective
This checklist defines the final release gate before declaring Sila Chain ready for public mainnet operation.

---

## A) Code / Test Gate
- [x] `go test ./internal/chain` passes
- [x] `go test ./internal/rpc` passes
- [x] `go test ./internal/state` passes
- [x] `go test ./internal/vm` passes
- [x] `go test ./pkg/sdk` passes
- [x] soak tests pass
- [x] restart/recovery tests pass
- [x] concurrent RPC stress tests pass

## B) Configuration Freeze
- [x] `chain_id = 1001`
- [x] `block_reward = 10`
- [x] `unbonding_delay = 3`
- [x] `min_validator_stake = 1`
- [x] `validator_commission_bps = 1000`
- [x] burn disabled in v1
- [x] treasury disabled in v1
- [x] monetary policy marked frozen in API

## C) Documentation Gate
- [x] tokenomics document published
- [x] monetary policy document published
- [x] mainnet hardening document published
- [x] security review checklist completed
- [x] operator runbook documented
- [x] validator onboarding guide documented

## D) Explorer / API Gate
- [x] `chain/info` exposes monetary metrics
- [x] explorer summary exposes monetary metrics
- [x] explorer network exposes monetary metrics
- [x] explorer contract endpoint works
- [x] explorer tx-vm endpoint works
- [x] explorer logs endpoint works
- [x] explorer frontend page works

## E) Wallet / SDK Gate
- [x] wallet CLI creates wallet
- [x] wallet CLI imports wallet
- [x] SDK calldata helper works
- [x] SDK tx builder works
- [x] SDK signing works
- [x] SDK receipt polling works

## F) Validator / Staking Gate
- [x] validator set loads correctly
- [x] active validator set rebuilds correctly
- [x] rewards accounting verified
- [x] slash accounting verified
- [x] pending unbond logic verified
- [x] jailed validator handling verified

## G) Operational Gate
- [x] node startup validated
- [x] restart and recovery validated
- [x] backup procedure documented
- [x] deployment rollout plan documented
- [x] rollback plan documented
- [x] incident response contact/process documented

## H) Final Sign-Off
- [x] engineering sign-off
- [x] economics sign-off
- [x] validator operations sign-off
- [x] release sign-off
- [x] public release approved
