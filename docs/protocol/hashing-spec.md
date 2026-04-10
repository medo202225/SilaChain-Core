# Sila Hashing Specification v1.0

## Purpose
This document defines the canonical hashing rules used across Sila.

## Primary Hash Function
Sila uses:
- Keccak-256

## Hash Usage
Keccak-256 is used for:
- address derivation
- transaction signing digest
- transaction hash
- block hash
- receipt hash where applicable

## Canonical Input Requirement
Hashing must only be performed on canonical deterministic byte sequences.

This means:
- same logical object
- same field order
- same serialization
- same bytes
- same hash on every node

## Transaction Hashing
The transaction hash is computed from the canonical transaction hash payload.

## Transaction Signing Hash
The transaction signing hash is computed from the canonical signing payload.
This payload may differ from the final transport object.

## Block Hashing
The block hash is computed from the canonical block header payload.

## Encoding Rules
- Internal hash values are raw 32-byte values.
- External JSON transport uses lowercase hex unless otherwise specified.
- Display values should remain lowercase hex.

## Forbidden Practices
The following are forbidden:
- hashing non-deterministic maps
- hashing objects with unstable field ordering
- hashing pretty-printed JSON
- hashing values with locale-dependent formatting
- hashing timestamps in inconsistent units

## Determinism Rule
If two honest nodes process the same protocol object, they MUST derive the exact same hash bytes.
