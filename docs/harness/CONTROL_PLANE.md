# CONTROL_PLANE.md

This is the active governance source of truth for `xbit-core`.

## Doctrine

Codex must optimize for system integrity, not just passing tests.

Blocking failures:

- deprecated or archived behavior used in active runtime
- fallback added without named failure mode, caller, test, and observability
- fake compatibility for unknown callers
- domain boundary drift
- schema, contract, fixture, test mismatch
- complex signing, risk, custody, order, or execution change without plan and verification

## Gate Semantics

- P0: stop work or fail verification. Funds, signing, custody, auth, order execution, irreversible external writes.
- P1: stop work unless explicitly scoped and tested. Contract drift, migration drift, broken realtime delivery, unsafe compatibility.
- P2: report and fix when in touched surface. Naming, duplication, missing docs for active behavior.
- P3: report only unless current task promotes it.

## Complexity Levels

- C0: docs, comments, small naming fixes.
- C1: isolated helper, no runtime contract change.
- C2: service behavior or API shape with focused tests.
- C3: auth, wallet, signer, order lifecycle, realtime, migrations, or cross-service contracts.
- C4: production execution, custody, risk rules, nonce/signature policy, external exchange writes.

C3/C4 work requires the full task contract from `CHANGE_INTENT_PROTOCOL.md` and the review gate.

## Historical Content

Historical docs are archive-only unless migrated into canonical owners. If old docs conflict with canonical docs, canonical docs win.
