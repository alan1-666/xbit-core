---
name: deprecation-enforcer
description: Enforce deprecation discipline in xbit-core. Use when changing GraphQL facades, gateway routes, generated schema callers, old endpoints, or migration paths.
---

# Deprecation Enforcer

## Steps

1. Identify whether the touched behavior is active, migrating, deprecated, or archived.
2. Confirm the active owner doc or service README.
3. Reject new active callers of deprecated behavior.
4. Allow compatibility only for known callers with a removal condition.
5. Run `python3 tools/discipline/check_deprecated_refs.py`.

## Output

- Deprecated code involved:
- Caller known:
- Replacement:
- Removal condition:
- Verification:
