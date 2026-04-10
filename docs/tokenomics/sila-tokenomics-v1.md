# SILA Tokenomics v1

## Status
Approved

## Asset
- Name: Sila
- Symbol: SILA
- Network: Sila Chain
- Asset Type: Native Layer 1 coin

## Vision
SILA is the native digital asset of Sila Chain.
It is used for:
- transaction fees
- smart contract gas
- validator staking
- delegation security
- network-level economic alignment

## Core Utility

### 1) Gas and Fees
All on-chain execution consumes SILA-denominated fees.
This includes:
- transfers
- contract deployment
- contract execution
- read/write state transitions that enter block execution

### 2) Staking
Validators stake SILA to participate in consensus security.

### 3) Delegation
Delegators may delegate SILA to validators and share in rewards.

### 4) Security Bond
SILA acts as the economic security layer of the chain.
Misbehavior can lead to slashing.

## Initial Supply Policy

### Genesis Supply
- Initial genesis supply should be explicitly fixed in chain config
- Recommended current baseline: keep the existing configured genesis supply as the canonical v1 genesis number unless governance changes it later

### Circulating Supply
Circulating supply should be defined as:
- total minted supply
- minus locked treasury allocations
- minus non-circulating protocol reserves
- minus permanently burned supply

## Emission Policy v1

### Validator Rewards
Block rewards are paid in SILA to validators and delegators according to protocol reward distribution rules.

### Recommended Emission Direction
For v1 mainnet positioning, SILA should prefer:
- low predictable emission
- simple auditable reward schedule
- transparent reward accounting

### Suggested Policy
- fixed block reward at early stage OR
- epoch-based emission schedule with explicit governance updates

## Fee Policy

### Gas Pricing
Users pay:
- gas_price
- gas_limit

Effective fee is paid in SILA.

### Fee Distribution Recommendation
Recommended v1 split:
- majority to validators / network security
- optional future treasury fraction
- optional future burn fraction

## Burn Policy

### Recommended v1
Start with:
- no mandatory burn in earliest public phase
- keep protocol hooks ready for future fee burn activation

### Future Burn Option
A portion of transaction fees may be burned to:
- reduce net issuance
- align long-term scarcity
- improve monetary credibility

## Slashing Policy
SILA can be slashed for protocol-defined validator faults.
Recommended slashing classes:
- equivocation / double-sign
- prolonged liveness failure
- invalid protocol participation

## Treasury Policy
Recommended treasury principles:
- transparent
- auditable
- publicly documented
- governed by explicit rules, never implicit operator discretion

## Governance Direction
SILA monetary policy changes should only happen through:
- documented protocol upgrade
- explicit governance process
- versioned tokenomics revisions

## Transparency Metrics
Explorer and chain APIs should eventually expose:
- total supply
- circulating supply
- staked supply
- delegated supply
- reward emissions
- burned supply
- treasury balances

## v1 Recommended Public Positioning
SILA should be presented as:
- the native gas asset of Sila Chain
- the staking and delegation asset
- the economic security unit of the protocol
- the settlement unit of smart contract execution

## Immediate Next Steps
- lock the canonical genesis supply in public docs
- document current reward flow from live code behavior
- expose supply / staking / reward metrics in explorer APIs
- decide whether v1 launches with burn disabled or with partial burn

## Future Tokenomics Work
- finalized emission curve
- treasury rules
- burn activation logic
- on-chain governance hooks
- supply dashboards

