# NO_FALLBACK_POLICY.md

Fallback is not default in `xbit-core`.

## Allowed Fallback Requirements

A fallback is allowed only when all are true:

- named failure mode
- named caller or user experience
- bounded scope
- test coverage
- observable signal, log, metric, audit event, or trace
- explicit owner in code or docs

Use `governance:allow-fallback` near the implementation when a fallback is intentionally approved.

## Hard Blocks

- No production fallback from real signing to dev signing.
- No production fallback from user-approved action to server-invented action.
- No fallback that silently changes order side, size, symbol, leverage, nonce, account, wallet, or chain.
- No fallback from private user data to public or seeded data in authenticated flows.
- No compatibility for unknown callers.

## Acceptable Non-Runtime Defaults

Parameter names like `fallback` in pure parsing helpers are allowed when they do not hide runtime failure.
