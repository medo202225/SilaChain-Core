# Sila Fork Choice Specification v1.0

## Purpose
This document defines how nodes choose between competing valid chain branches.

## Principle
Fork choice must be deterministic and identical across all honest nodes.

## Primary Rule
Nodes prefer the valid branch that satisfies the protocol's fork-choice rule under the active consensus model.

## Mandatory Conditions
A candidate branch is only eligible if:
- all blocks are structurally valid
- all transactions are valid
- all state transitions are valid
- proposer rules are satisfied
- finality rules are not violated

## Safety Override
A node MUST NOT choose any branch that conflicts with finalized history.

## Tie Handling
If two competing branches satisfy the same weight or score rules, the protocol implementation must define a deterministic tie-breaker.

## Restart Consistency
After restart, a node must recompute the same fork-choice result from persisted state and chain data.

## Goal
Fork choice must maximize consistency, safety, and deterministic convergence.
