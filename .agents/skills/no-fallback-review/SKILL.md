---
name: no-fallback-review
description: Review backend changes for unauthorized fallbacks, fake compatibility, silent defaults, and hidden best-effort behavior. Use for signer, order, wallet, gateway, provider, realtime, or API compatibility changes.
---

# No Fallback Review

Use this workflow before closing C2+ backend work that touches runtime behavior.

## Steps

1. Identify each fallback, default, compatibility route, retry, and best-effort branch.
2. For each one, name the failure mode and caller.
3. Verify it has a bound, test, and observable signal.
4. Reject any fallback in signer, nonce, order, wallet, account, leverage, transfer, or withdrawal paths unless explicitly approved.
5. Check `docs/harness/NO_FALLBACK_POLICY.md`.

## Output

Report:

- Allowed fallback:
- Rejected fallback:
- Compatibility caller:
- Verification evidence:
- Remaining risk:
