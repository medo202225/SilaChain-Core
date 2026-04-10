# SILA Monetary Policy v1

## Status
Approved

## Asset
- Name: Sila
- Symbol: SILA
- Type: Native Layer 1 asset of Sila Chain

## Monetary Role
SILA is the native economic unit of Sila Chain and is used for:
- transaction fees
- smart contract gas
- validator staking
- delegation-based security
- reward settlement

## Policy Direction
SILA monetary policy should prioritize:
- simplicity
- predictability
- auditability
- low surprise for validators, users, and developers

## Supply Model

### Genesis Supply
The genesis supply must be frozen and published as the canonical mainnet genesis amount.

### Recommended v1 Rule
- keep a fixed genesis supply at launch
- avoid arbitrary operator minting
- any future issuance changes must be versioned and documented

## Emission Model

### Validator Rewards
Block rewards are paid in SILA to validators, with delegator sharing according to protocol reward distribution.

### Final v1 Recommendation
- low, explicit, predictable emission
- fixed reward schedule for the first public mainnet phase
- no hidden or discretionary inflation

### Governance Rule
Any change to reward emission must require:
- protocol upgrade
- public documentation update
- release version note

## Fee Policy

### Current Economic Role
Users pay SILA-denominated execution fees through gas pricing.

### Final v1 Recommendation
Fee flow should be documented as:
- validator compensation component
- optional treasury component
- optional burn component

## Burn Policy

### Final v1 Recommendation
For earliest stable public mainnet:
- burn disabled by default OR
- explicitly small fixed fee-burn fraction

### Preferred Conservative Start
- launch with burn disabled
- enable future burn only after usage metrics are understood

## Treasury Policy

### Recommendation
If treasury exists, it must be:
- transparent
- explicitly bounded
- publicly documented
- visible in explorer and accounting outputs

## Slashing Policy
SILA may be slashed for:
- double-sign / equivocation
- severe validator faults
- protocol-defined security violations

Slash events must be:
- explicit
- queryable
- auditable

## Mainnet Monetary Freeze Conditions
Before declaring monetary policy final for public mainnet:
- genesis supply must be frozen
- reward schedule must be frozen
- fee handling must be frozen
- treasury rule must be frozen
- burn setting must be frozen

## Public Metrics To Expose
Explorer / APIs should expose:
- total supply
- staked supply
- delegated supply
- pending rewards
- slashed amount
- burned amount
- treasury balances if enabled

## Final v1 Positioning
SILA should be publicly positioned as:
- the gas asset of Sila Chain
- the staking and delegation asset
- the settlement asset of smart contract execution
- the economic security unit of the protocol

## Immediate Next Steps
- freeze canonical genesis supply
- freeze initial block reward rule
- decide burn setting for v1
- publish validator reward split
- expose monetary metrics through explorer and APIs

