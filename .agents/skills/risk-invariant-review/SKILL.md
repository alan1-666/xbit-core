---
name: risk-invariant-review
description: Review signer, wallet, order, leverage, exchange, and realtime changes against xbit-core risk invariants. Use for C3/C4 backend work.
---

# Risk Invariant Review

Read `docs/risk/RISK_INVARIANTS.md` before reviewing.

## Steps

1. List touched risk invariants.
2. Confirm no invariant is weakened.
3. Confirm tests cover the highest-risk path.
4. Confirm logs/audit events exist for high-risk operations.
5. Record remaining risk.
