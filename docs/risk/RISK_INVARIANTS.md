# RISK_INVARIANTS.md

These invariants protect signing, custody, and execution.

## Signer

- Dev signing must never be used as production signing.
- Nonce generation must be monotonic per managed agent.
- Agent signer may only sign allowed actions: `order`, `cancel`, `updateLeverage`.
- Withdrawals and transfers are excluded from managed agent signer MVP.
- API responses must not expose stored key material or `keyRef`.

## Orders

- Order side, symbol, size, reduce-only flag, time-in-force, leverage, and user address must not be silently rewritten.
- Provider order IDs and client request IDs must remain traceable.
- External exchange writes require signed payloads or an explicitly enabled signer path.

## Wallet And Funds

- Wallet custody changes must be explicit and audited.
- No service may infer a user's wallet or chain when the request is ambiguous.
- Withdrawal/transfer flows require stronger review than read-model changes.

## Realtime

- Reconnect reconciliation may repair read models, but it must not invent exchange execution.
- Private topics must remain user-scoped.
