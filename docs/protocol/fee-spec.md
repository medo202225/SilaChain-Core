# Sila Fee Specification v1.0

## Purpose
This document defines the transaction fee model for Sila.

## Fee Model
Sila v1.0 uses a native fee model denominated in the smallest unit of the SILA token.

## Transaction Fee Fields
Each transaction includes:
- fee

## Validation Rules
A valid transaction fee MUST:
- be present
- be a non-negative integer
- satisfy the protocol minimum fee
- be fully covered by the sender balance together with transfer value

## Total Sender Cost
For a native transfer transaction:

total_cost = value + fee

The sender balance must be at least `total_cost`.

## Minimum Fee Policy
The protocol defines a minimum acceptable fee.
Transactions below the minimum fee MUST be rejected by nodes.

## Future Extensions
Future protocol versions may introduce:
- dynamic fee markets
- block congestion pricing
- base fee + priority fee
- gas metering across execution types

v1.0 keeps the fee model intentionally simple and deterministic.

## Fee Recipient
Transaction fees are credited according to the active consensus and validator reward policy.

## Determinism Rule
All nodes MUST compute the same total cost and apply identical fee validation rules.
