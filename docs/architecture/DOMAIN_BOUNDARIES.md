# DOMAIN_BOUNDARIES.md

This file owns service boundaries for `xbit-core`.

## Active Domains

| Domain | Owns | Must Not Own |
| --- | --- | --- |
| `identity` | login, sessions, auth challenge, JWT refresh/revoke | wallet custody, trading decisions |
| `wallet` | wallet records, withdrawal allowlists, security events | market execution, Hyperliquid signing |
| `trading` | spot/route quotes, spot order lifecycle, transaction status | futures order semantics, custody |
| `marketdata` | token read models, OHLC, pools, transactions, checkpoints | user private state, signing |
| `hypertrader` | futures symbols, Hyperliquid account/order/fill/funding read model, agent signer MVP | generic wallet custody, identity sessions |
| `streambridge` | event envelopes and MQTT/in-memory publication | source-of-truth business decisions |
| `gateway` | compatibility routing to upstream or local services | business logic that belongs in a service |

## Boundary Rules

- Signing authority must be explicit and domain-owned.
- Frontend-compatible GraphQL facades may route traffic but must not become an unbounded compatibility layer.
- Cross-service data dependencies need a named contract or schema.
- A service may expose read-model fallbacks only when the failure mode is named, tested, and observable.
- Migrations define durable storage, not business policy.

## Hypertrader Boundary

`hypertrader` owns futures/contract flows:

- Hyperliquid read facade
- local provider and HTTP provider adapters
- order create/cancel/sync lifecycle
- private WS reconciliation snapshots
- agent wallet registration, activation, nonce tracking, and guarded signing

It must not own:

- user login lifecycle
- generic wallet custody
- arbitrary transfer/withdraw signing outside the registered agent signer policy

## Gateway Boundary

The gateway can preserve route shape for known frontend callers. It must not add fake compatibility for unknown callers.
