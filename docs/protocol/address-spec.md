# Sila Address Specification v1.0

## Purpose
This document defines the canonical address format for the Sila blockchain.

## Cryptographic Basis
- Curve: secp256k1
- Public Key Format: uncompressed public key bytes
- Hash Function: Keccak-256

## Address Derivation
1. Generate a secp256k1 keypair.
2. Obtain the uncompressed public key bytes.
3. Remove the first prefix byte if public key serialization includes it.
4. Compute Keccak-256 of the remaining public key bytes.
5. Take the last 20 bytes of the hash result.
6. Encode the 20-byte address body as lowercase hexadecimal.
7. Prefix the final result with `SILA_`.

## Canonical Address Format
SILA_<40 lowercase hex chars>

Example:
SILA_0123456789abcdef0123456789abcdef01234567

## Rules
- Address hex MUST be lowercase.
- Address body MUST be exactly 20 bytes = 40 hex chars.
- Prefix MUST be exactly `SILA_`.
- Total printable length MUST be 45 characters.

## Validation Rules
A valid Sila address MUST:
- start with `SILA_`
- contain exactly 40 lowercase hexadecimal characters after the prefix
- contain no spaces
- contain no checksum capitalization scheme in v1.0

## Notes
- v1.0 does not use mixed-case checksum encoding.
- Future versions may introduce optional checksum representations, but canonical storage remains lowercase.

## Rationale
This format preserves a strong and familiar account-based model while clearly distinguishing Sila addresses from Ethereum-style addresses.
