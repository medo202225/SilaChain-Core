# Sila Finality Specification v1.0

## Purpose
This document defines finality at the protocol level for Sila.

## Concept
A block is considered finalized when the network reaches the validator participation threshold required by the consensus protocol.

## Finality Goals
- prevent conflicting finalized histories
- provide a stable settlement point
- enable safe downstream integrations
- reduce uncertainty for users and applications

## Validation Requirements
To mark a block as finalized, nodes must verify:
- the block is valid
- the proposer was authorized
- required validator participation threshold was reached
- no conflicting finalized block exists for the same height

## Finality State
Each node tracks:
- latest finalized height
- latest finalized block hash

## Safety Rule
No node may accept two different finalized blocks at the same height.

## Operational Rule
Applications integrating with Sila should distinguish between:
- block observed
- block accepted
- block finalized

## v1 Scope
Sila v1.0 prioritizes clear and deterministic finality semantics before advanced optimizations.
