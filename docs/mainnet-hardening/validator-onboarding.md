# Sila Validator Onboarding v1

## Status
Approved

## Purpose
This guide defines the minimum onboarding process for a validator joining Sila Chain mainnet operations.

---

## 1) Validator Responsibilities
A validator is responsible for:
- running a stable Sila node
- staking the required SILA amount
- protecting validator-related keys
- maintaining uptime and correct configuration
- following upgrade and incident procedures
- understanding slash and jail consequences

## 2) Minimum Entry Conditions
Before onboarding:
- validator has a secure host/server
- validator has reviewed the operator runbook
- validator has reviewed the monetary policy
- validator understands commission settings
- validator understands slashing and jail implications

## 3) Required Network Parameters
Validator must verify:
- chain id is the frozen public mainnet value
- min validator stake is the frozen public value
- validator commission policy matches mainnet rules
- burn and treasury policy visibility matches public API

## 4) Key Handling Requirements
- generate validator/operator wallet securely
- never expose private keys in public channels
- store backups securely
- restrict host access
- rotate keys only through documented procedure if ever needed

## 5) Funding and Stake Preparation
Validator should:
- obtain sufficient SILA
- fund operational wallet
- confirm staking amount is above minimum validator stake
- verify expected commission settings

## 6) Node Preparation
- deploy node environment
- verify config/networks/mainnet/public files
- verify data directory
- verify RPC exposure policy
- verify health endpoint after startup
- verify chain info endpoint after startup

## 7) Validator Activation Checklist
- node starts successfully
- chain info responds
- explorer network responds
- validator appears in validator-related endpoints if applicable
- stake is visible
- active validator status is confirmed when conditions are met

## 8) Ongoing Duties
Validator must:
- monitor uptime
- monitor chain progress
- monitor reward accumulation
- monitor jail/slash status
- monitor upgrade announcements
- maintain backups and recovery readiness

## 9) Risk Awareness
Validator acknowledges:
- misbehavior may cause slashing
- severe faults may cause jailing
- downtime may reduce participation quality
- protocol upgrades may require operator action
- monetary policy changes require explicit protocol process

## 10) Suspension / Exit Awareness
Validator should understand:
- undelegation and unbonding rules
- pending reward settlement flow
- unbonding delay impact
- restart/recovery procedures before attempting operational changes

## 11) Onboarding Sign-Off
Before declaring validator ready:
- operator runbook reviewed
- wallet/key handling completed
- stake funded
- node started successfully
- monitoring prepared
- incident procedure reviewed

