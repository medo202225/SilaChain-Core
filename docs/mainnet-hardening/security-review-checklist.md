# Sila Security Review Checklist v1

## Status
Approved

## Objective
This checklist defines the minimum security review items before declaring Sila Chain ready for public mainnet release.

---

## 1) Consensus / Chain Safety
- [x] verify invalid parent rejection
- [x] verify invalid height rejection
- [x] verify duplicate transaction rejection
- [x] verify invalid block replay rejection
- [x] verify proposer rotation does not corrupt validator scheduling
- [x] verify chain height remains monotonic under restart

## 2) Transaction Validation
- [x] verify invalid signature rejection
- [x] verify invalid public key rejection
- [x] verify invalid sender address rejection
- [x] verify invalid nonce rejection
- [x] verify invalid chain id rejection
- [x] verify invalid gas limit rejection
- [x] verify malformed transaction payload rejection

## 3) VM / Smart Contract Safety
- [x] verify invalid opcode fault behavior
- [x] verify out-of-gas fault behavior
- [x] verify revert rollback behavior
- [x] verify storage rollback after revert
- [x] verify CREATE failure cleanup
- [x] verify static execution write protection
- [x] verify invalid jump rejection
- [x] verify code size limit enforcement
- [x] verify logs topics remain deterministic

## 4) State / Persistence Safety
- [x] verify receipts persist after restart
- [x] verify tx index persists after restart
- [x] verify contract code persists after restart
- [x] verify contract storage persists after restart
- [x] verify account balances persist after restart
- [x] verify validator and staking records persist after restart

## 5) RPC / API Safety
- [x] verify request body limits
- [x] verify unknown JSON field rejection
- [x] verify malformed JSON rejection
- [x] verify admin protection on sensitive endpoints
- [x] verify explorer endpoints are read-only
- [x] verify contract write endpoints are not exposed unintentionally
- [x] verify concurrent read stress remains stable

## 6) Economic / Monetary Safety
- [x] verify chain id matches frozen public value
- [x] verify block reward matches frozen public value
- [x] verify min validator stake matches frozen public value
- [x] verify validator commission matches frozen public value
- [x] verify burn_enabled is false in v1 public API
- [x] verify treasury_enabled is false in v1 public API
- [x] verify monetary_policy_frozen is true in public API

## 7) Staking / Validator Safety
- [x] verify staking accounting correctness
- [x] verify delegation accounting correctness
- [x] verify undelegation accounting correctness
- [x] verify pending rewards accounting correctness
- [x] verify slash accounting correctness
- [x] verify jailed validator exclusion correctness

## 8) Wallet / SDK Safety
- [x] verify wallet generation works correctly
- [x] verify wallet import works correctly
- [x] verify tx builder outputs valid payloads
- [x] verify signing helpers produce valid signatures
- [x] verify receipt polling handles unavailable receipts safely

## 9) Explorer / Developer UX Safety
- [x] verify explorer contract endpoint handles missing contract safely
- [x] verify explorer tx-vm endpoint handles missing receipt safely
- [x] verify explorer logs endpoint handles empty result safely
- [x] verify explorer frontend pages do not expose secret material

## 10) Review Sign-Off
- [x] core chain review complete
- [x] VM review complete
- [x] RPC review complete
- [x] staking/economics review complete
- [x] SDK/wallet review complete
- [x] release owner sign-off complete
