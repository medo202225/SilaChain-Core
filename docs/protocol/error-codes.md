# Sila Error Codes Specification v1.0

## Purpose
This document defines protocol-facing error categories for consistent node and client behavior.

## Principles
- errors should be deterministic where possible
- error categories should be stable
- transport-specific wording may vary, but semantic meaning should remain consistent

## Core Categories
### TX_INVALID_FORMAT
Malformed transaction structure.

### TX_INVALID_SIGNATURE
Transaction signature failed verification.

### TX_INVALID_PUBLIC_KEY
Public key is malformed or incompatible with protocol rules.

### TX_ADDRESS_MISMATCH
Derived address from public key does not match transaction sender.

### TX_INVALID_NONCE
Transaction nonce does not match expected sender nonce.

### TX_INSUFFICIENT_BALANCE
Sender balance cannot cover transaction total cost.

### TX_INVALID_CHAIN_ID
Transaction chain_id does not match local network.

### TX_DUPLICATE
Transaction already exists in mempool or chain context.

### BLOCK_INVALID_HASH
Block hash is incorrect.

### BLOCK_INVALID_PARENT
Parent hash or parent linkage is invalid.

### BLOCK_INVALID_PROPOSER
Block proposer is not authorized for the given height/slot.

### STATE_TRANSITION_FAILED
Block or transaction execution did not produce a valid state transition.

### CONSENSUS_RULE_VIOLATION
Consensus-specific validation rule was violated.

### GENESIS_MISMATCH
Local genesis configuration does not match network genesis.

### NETWORK_VERSION_MISMATCH
Peer or data version is incompatible with local protocol version.

## Requirement
Implementations should map internal errors into stable external categories where practical.
