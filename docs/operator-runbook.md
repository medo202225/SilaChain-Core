# Operator Runbook

## Before starting the node
- verify validator key file exists
- verify node config path
- verify peers file
- verify data directory permissions

## On startup
- check health endpoint
- confirm chain height
- confirm validator set loaded
- confirm no storage corruption errors

## On security failure
- stop the node
- preserve logs
- backup data directory
- verify validator key integrity
- inspect recent admin access attempts

## On fork detection
- inspect peer source
- inspect local latest block hash
- inspect peer latest block hash
- do not force manual chain replacement without review
