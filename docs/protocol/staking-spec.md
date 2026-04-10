# Sila Staking Specification v1.0

## Purpose
This document defines staking at a protocol level for Sila.

## Model
Sila uses a Proof of Stake model.
Validators participate by staking the native token according to protocol rules.

## Validator Staking
A validator must have:
- a valid address
- a valid public key
- an active status
- sufficient stake at or above protocol minimum

## Staking Roles
The protocol may support:
- self-stake
- delegation
- reward distribution
- unbonding

## Core Rules
- stake must be tracked deterministically
- validator eligibility depends on active status and stake rules
- staking transitions must be reflected in protocol state
- validator-set updates must follow epoch and consensus rules

## Rewards
Active validators may receive rewards according to:
- block production
- participation
- protocol reward schedule

## Unbonding
If unbonding is supported, stake removal is delayed by the protocol-defined unbonding period.

## Safety Requirement
Slashed stake, locked stake, and active stake must be accounted for consistently across all nodes.
