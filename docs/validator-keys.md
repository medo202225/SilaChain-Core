# Validator Keys

## Purpose
Each validator uses a dedicated private key to sign validator-related actions.

## Requirements
- private key must be stored outside source code
- file permissions should be restricted
- backup must be encrypted or stored offline
- public key and address must match the private key

## Recommended file location
config/validator/key.json

## File format
{
  "address": "SILA_...",
  "public_key": "...",
  "private_key": "..."
}

## Operational rules
- do not paste private keys into logs
- do not send key files over chat
- do not store validator keys in public repositories
