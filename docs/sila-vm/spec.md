# Sila VM v1

## Status
Draft

## Scope
Sila VM v1 is the native smart contract execution engine for Sila Chain.

This version currently defines and/or implements:

- 256-bit word model
- stack-based execution
- transient memory
- persistent storage
- gas metering
- success / revert / fault outcomes
- checkpoint-backed rollback
- event logs
- topics-based logs
- static execution protection
- internal contract calls
- static internal calls
- contract creation
- comparison opcodes
- control flow opcodes

## Core Principles

- Deterministic execution only
- No filesystem access
- No network access
- No OS time access
- No non-deterministic behavior
- Gas-bounded execution
- Explicit commit / rollback semantics

## Current Opcode Groups

### Arithmetic
- ADD
- SUB

### Comparison
- LT
- GT
- EQ
- ISZERO

### Context
- ADDRESS
- CALLER
- CALLVALUE
- CALLDATALOAD
- CALLDATASIZE

### Stack
- POP
- PUSH1

### Memory
- MLOAD
- MSTORE

### Storage
- SLOAD
- SSTORE

### Control Flow
- JUMP
- JUMPI
- JUMPDEST

### Logs
- LOG0
- LOG1
- LOG2
- LOG3
- LOG4

### Calls / Creation
- CALL
- STATICCALL
- CREATE

### Control / Exit
- STOP
- RETURN
- REVERT

## Execution Outcomes

### Success
Execution completed normally and state changes may commit.

### Revert
Execution failed semantically and all state changes in scope must roll back.

### Fault
Execution failed due to invalid opcode, out-of-gas, invalid jump, invalid access, or forbidden operation.

## Current Safety Rules

- SSTORE is forbidden in static mode
- LOG opcodes are forbidden in static mode
- CREATE is forbidden in static mode
- STATICCALL cannot transfer non-zero value
- Out-of-gas causes fault
- Revert rolls back storage changes
- Revert rolls back logs
- Runtime code size is bounded by VM limits
- Jumps are only valid to JUMPDEST

## Current CREATE Model

Sila VM CREATE now follows an early init/runtime lifecycle:

1. CREATE reads init code bytes from memory
2. VM allocates a new contract address through the host
3. init code executes in a child VM context
4. init code RETURN data becomes installed runtime code
5. if init code fails or reverts, contract creation returns zero and deployment is rolled back

## Current Limitations of CREATE

The current CREATE implementation is still an early v1 version:

- gas forwarding for child create execution is still simplified
- create-specific revert payload propagation is still minimal
- advanced deployment validation is not yet implemented
- address derivation is still mock-host driven in tests
- code size validation for returned runtime code still needs tighter enforcement

## Next Planned CREATE Upgrades

- final contract address derivation rules
- stricter runtime code validation
- create gas forwarding rules
- rollback-hardening across state integrations
- protocol-level deployment receipts

## Next Planned Features

- ABI formalization
- bytecode object format
- deeper gas schedule
- VM integration with Sila state and execution layers
- DeployContractTx
- CallContractTx
