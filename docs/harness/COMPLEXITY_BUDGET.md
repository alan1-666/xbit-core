# COMPLEXITY_BUDGET.md

Keep simple work simple and treat risky work as risky.

## C0/C1

- No new framework.
- No new abstraction unless it removes real duplication.
- Verification may be a focused command or doc check.

## C2

- Touch the smallest service boundary that owns the behavior.
- Add or update focused tests.
- Keep API compatibility explicit.

## C3/C4

- Require task contract, boundary decision, risk invariant check, and review gate.
- Prefer fail-fast over silent repair.
- Do not mix refactor and behavior changes unless unavoidable.
