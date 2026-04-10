# Sila Transaction Specification v1.0

## Purpose
This document defines the canonical transaction model for Sila.

## Transaction Model
Sila uses an account-based transaction model.

## Transaction Type v1
Native transfer transaction fields:

- from
- to
- value
- fee
- nonce
- chain_id
- timestamp
- public_key
- signature

## Field Definitions
### from
Canonical sender address.

### to
Recipient address.

### value
Amount transferred in the smallest native unit.

### fee
Transaction fee paid by the sender.

### nonce
Sender account nonce. Must match the sender account state at validation time.

### chain_id
Network identifier preventing replay across environments.

### timestamp
Unix timestamp in seconds.

### public_key
Sender public key in canonical hex encoding.

### signature
Signature over the canonical signing payload.

## Canonical Signing Payload
The signing payload for v1 transfer transactions includes:

- from
- to
- value
- fee
- nonce
- chain_id
- timestamp

The signing payload MUST NOT include:
- signature
- derived hash
- execution result fields

## Canonical Validation Rules
A valid transaction MUST satisfy all of the following:
- valid `from` address
- valid `to` address
- `value` > 0
- `fee` > 0 or meets protocol minimum
- `nonce` matches sender account nonce
- `public_key` is valid secp256k1 public key
- derived address from `public_key` matches `from`
- signature is valid
- sender balance covers `value + fee`
- chain_id matches local network chain_id

## Transaction Hash
The transaction hash is derived from the canonical transaction hash payload.
The protocol implementation may define it as:
- the same as signing payload hash, or
- a canonical payload including signature

The implementation MUST choose one method and keep it fixed across the network.

## Rejection Conditions
A node MUST reject a transaction if:
- signature is invalid
- sender not found
- nonce mismatch
- balance insufficient
- malformed public key
- malformed address
- wrong chain_id
- duplicate transaction hash in mempool
