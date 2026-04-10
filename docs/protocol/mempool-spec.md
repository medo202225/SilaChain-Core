# Sila Mempool Specification v1.0

## Purpose
This document defines the mempool behavior for pending transactions in Sila.

## Mempool Role
The mempool stores valid, not-yet-included transactions awaiting block inclusion.

## Admission Rules
A node may admit a transaction into mempool only if:
- transaction structure is valid
- signature is valid
- sender exists or is otherwise protocol-acceptable
- nonce is valid for admission policy
- sender balance is sufficient
- fee satisfies minimum policy
- chain_id matches local chain
- transaction hash is not already present

## Duplicate Handling
Duplicate transaction hashes MUST NOT be stored more than once.

## Ordering
The mempool implementation may order transactions by:
- arrival time
- fee priority
- nonce constraints
- sender-local sequencing

The chosen ordering policy must preserve transaction validity.

## Removal Conditions
A transaction must be removed if:
- it is included in a block
- it becomes invalid after state changes
- it expires under local policy
- it is replaced under a valid replacement policy, if supported

## Restart Behavior
Nodes may persist and restore mempool contents, but restored transactions must be revalidated.

## Safety Rule
Mempool presence does not imply final acceptance into the chain.
