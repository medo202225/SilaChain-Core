# Sila Block Specification v1.0

## Purpose
This document defines the canonical block structure for Sila.

## Block Structure
A block consists of:
- header
- transactions
- receipts (optional in transport depending on endpoint)

## Block Header Fields
- height
- parent_hash
- state_root
- tx_root
- receipt_root
- timestamp
- proposer
- gas_used
- gas_limit
- tx_count
- block_hash

## Field Definitions
### height
Monotonic block number starting from genesis height 0.

### parent_hash
Hash of the previous block header.

### state_root
State commitment after block execution.

### tx_root
Commitment over included transactions.

### receipt_root
Commitment over transaction receipts.

### timestamp
Block production timestamp in Unix seconds.

### proposer
Validator address that proposed the block.

### gas_used
Total gas consumed by included transactions.

### gas_limit
Maximum gas allowed for the block.

### tx_count
Number of transactions included in the block.

### block_hash
Canonical hash of the block header payload.

## Canonical Header Hash Payload
The block hash is computed over the canonical header payload excluding `block_hash` itself.

## Validation Rules
A block is valid only if:
- parent_hash matches local parent
- height is parent height + 1
- proposer is authorized for that height/slot
- block hash is correct
- tx_count matches included transactions
- tx_root matches included transactions
- execution yields the claimed state_root
- receipt_root matches execution receipts
- gas_used does not exceed gas_limit
- timestamp is within accepted protocol bounds

## Genesis Block
Genesis is height 0 and is derived from the canonical genesis configuration.
All nodes must produce the same genesis hash from the same genesis file.
