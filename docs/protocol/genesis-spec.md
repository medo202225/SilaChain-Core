# Sila Genesis Specification v1.0

## Purpose
This document defines the canonical genesis file structure and rules.

## Genesis Responsibilities
The genesis file defines:
- chain identity
- initial accounts
- initial balances
- initial validator set
- protocol parameters active at launch
- initial timestamp and network configuration

## Required Genesis Fields
- chain_name
- chain_id
- genesis_time
- native_symbol
- decimals
- total_supply
- initial_accounts
- initial_validators
- protocol_params

## Initial Accounts
Each account entry includes:
- address
- balance
- optional metadata fields allowed only if explicitly specified by the protocol

## Initial Validators
Each validator entry includes:
- address
- stake
- public_key
- status

## Protocol Params
Examples:
- min_fee
- block_time_seconds
- max_txs_per_block
- max_block_gas
- epoch_length
- unbonding_period

## Genesis Determinism
The same genesis file must produce:
- the same genesis state
- the same genesis block
- the same genesis hash

across all honest nodes.

## Rules
- No duplicate addresses in initial accounts
- No duplicate validator addresses
- Total balances and allocations must be internally consistent
- Validator addresses must be valid Sila addresses
- Validator public keys must be valid for the chosen curve
- chain_id must uniquely identify the environment

## Environments
Distinct genesis files must exist for:
- devnet
- testnet
- mainnet

A node must refuse to join peers on a mismatched chain_id.
