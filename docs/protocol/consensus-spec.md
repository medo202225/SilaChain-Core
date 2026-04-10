# Sila Consensus Specification v1.0

## Purpose
This document defines the high-level consensus model for Sila v1.0.

## Model
Sila v1.0 uses:
- Proof of Stake
- deterministic proposer selection
- validator-based block production
- finality logic defined by validator participation rules

## Validator Set
A validator is an authorized participant able to propose and attest to blocks.

Validator properties:
- address
- public_key
- stake
- status

## Epochs
Consensus operates in epochs.
At epoch boundaries, validator-set updates and protocol transitions may be applied.

## Proposer Selection
For each block height or slot:
- the proposer is selected deterministically from the active validator set
- selection rules must be identical on every node

## Block Proposal
The selected proposer:
1. collects valid transactions from mempool
2. builds a candidate block
3. executes transactions deterministically
4. computes resulting commitments
5. signs and broadcasts the proposed block

## Validation
Peers validate:
- proposer eligibility
- block structure
- transaction validity
- state transition correctness
- commitment correctness

## Finality
A block is considered finalized when the protocol's validator participation threshold is satisfied.
The exact threshold and attestation mechanism must be implemented consistently across all validators.

## Slashing Conditions
A validator may be penalized for:
- double proposal
- double vote
- invalid block signing
- safety rule violation
- persistent downtime depending on network policy

## Safety Goals
Consensus must prioritize:
- deterministic execution
- consistent proposer selection
- slashable misbehavior evidence
- replay-safe signatures
- stable recovery after restart

## v1 Scope
Sila v1.0 prioritizes:
- correctness
- determinism
- operational safety
- validator clarity

before advanced throughput optimizations.
