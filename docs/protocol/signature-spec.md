# Sila Signature Specification v1.0

## Purpose
This document defines how messages and transactions are signed and verified in Sila.

## Cryptographic Basis
- Curve: secp256k1
- Signature Algorithm: ECDSA over secp256k1
- Hash Function Before Signing: Keccak-256

## Signing Principle
Private keys are controlled by the wallet.
Nodes MUST NOT sign user transactions.
Nodes only verify signatures.

## Signed Object
Transactions are signed over the canonical transaction signing payload.
The signature MUST NOT include fields that are produced after signing.

## Signing Flow
1. Build the canonical transaction signing payload.
2. Serialize it deterministically.
3. Compute Keccak-256 of the serialized payload.
4. Sign the resulting 32-byte digest using secp256k1.
5. Encode the signature as lowercase hexadecimal.

## Signature Encoding
- Canonical transport encoding: lowercase hex
- Signature body: raw signature bytes as produced by the selected signing library
- Recovery byte policy:
  - v1.0 may store recovery id separately or append it at the end
  - implementation must choose one canonical form and apply it consistently

## Verification Rules
To verify a transaction signature:
1. Rebuild the exact canonical signing payload.
2. Serialize deterministically.
3. Compute Keccak-256.
4. Verify the signature against the sender public key.
5. Derive the address from the public key.
6. Ensure the derived address matches the transaction sender address.

## Security Rules
- The private key MUST never leave the wallet boundary unencrypted.
- The node MUST reject malformed signatures.
- The node MUST reject signatures whose public key does not match the sender address.
- The node MUST reject signatures over non-canonical payload encodings.

## Canonical Requirement
All implementations must use the same:
- payload fields
- field order
- serialization format
- hash function
- signature encoding

Any deviation causes invalid signatures across nodes.
