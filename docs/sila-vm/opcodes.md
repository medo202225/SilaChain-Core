# Sila VM v1 Opcodes

## Arithmetic
- `0x01` ADD
- `0x03` SUB

## Comparison
- `0x10` LT
- `0x11` GT
- `0x14` EQ
- `0x15` ISZERO

## Context
- `0x30` ADDRESS
- `0x33` CALLER
- `0x34` CALLVALUE
- `0x35` CALLDATALOAD
- `0x36` CALLDATASIZE

## Stack / Memory / Storage
- `0x50` POP
- `0x51` MLOAD
- `0x52` MSTORE
- `0x54` SLOAD
- `0x55` SSTORE

## Control Flow
- `0x56` JUMP
- `0x57` JUMPI
- `0x5b` JUMPDEST

## Push
- `0x60` PUSH1

## Logs
- `0xa0` LOG0
- `0xa1` LOG1
- `0xa2` LOG2
- `0xa3` LOG3
- `0xa4` LOG4

## Creation / Calls
- `0xf0` CREATE
- `0xf1` CALL
- `0xfa` STATICCALL

## Exit
- `0x00` STOP
- `0xf3` RETURN
- `0xfd` REVERT

## Notes

### Static Restrictions
The following operations are forbidden in static execution mode:
- SSTORE
- LOG0..LOG4
- CREATE

### Jump Rules
JUMP and JUMPI are only valid when the destination points exactly to a JUMPDEST opcode.

### Current CREATE Semantics
CREATE reads init code from memory, executes it in a child VM context, and installs the child RETURN data as runtime code.
If child init execution fails, CREATE pushes zero and deployment does not complete.
