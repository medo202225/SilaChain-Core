# Sila Slashing Specification v1.0

## Purpose
This document defines slashable validator misbehavior in Sila.

## Goals
- protect consensus safety
- penalize provable misbehavior
- discourage equivocation and invalid signing
- maintain validator accountability

## Slashable Events
A validator may be slashed for:
- double proposal
- double vote
- signing conflicting history
- signing invalid consensus data
- violating explicit safety rules
- other protocol-defined evidence-backed faults

## Evidence Requirement
Slashing must only occur on verifiable evidence.
Evidence must be:
- machine-verifiable
- attributable to the validator
- valid under protocol rules

## Penalty Types
Penalties may include:
- stake reduction
- reward loss
- forced exit depending on severity
- slashing-associated ejection from active validator participation depending on protocol rules

## Safety Requirement
No validator may be slashed without valid protocol evidence.

## Operational Requirement
Nodes and validator services must preserve sufficient signing and protection data to avoid accidental slashable behavior.
