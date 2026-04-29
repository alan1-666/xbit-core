# AGENTS.md

This file is the repo-level context router for `xbit-core`.

It only does three things:
1. classify the task
2. route to canonical owner docs
3. remind Codex which gates cannot be skipped

It is not a full spec, backlog, or historical archive.

## Startup

Before coding, state:

- role
- task
- success criteria
- intent
- domain
- complexity
- verification plan

For non-trivial work, also state:

- allowed change surface
- forbidden change surface
- deprecated code involved
- compatibility required: yes/no
- fallback allowed: yes/no
- Kill List
- Preserve List
- Unknown List

Use `docs/harness/CHANGE_INTENT_PROTOCOL.md` as the full task contract.

## Canonical Source Priority

1. `AGENTS.md`
2. `docs/harness/CONTROL_PLANE.md`
3. `docs/architecture/DOMAIN_BOUNDARIES.md`
4. harness discipline docs under `docs/harness/`
5. `docs/risk/RISK_INVARIANTS.md`
6. active service READMEs and schema files
7. generated state and migrations
8. historical or archived docs

Generated and historical docs explain state or history. They do not define current behavior.

## Domain Routing

- Identity/auth: `internal/identity`, `pkg/auth`, auth migrations.
- Wallet/custody/accounting: `internal/wallet`, wallet migrations.
- Spot/trading execution: `internal/trading`, trading migrations.
- Market data/indexing: `internal/marketdata`, market-data migrations.
- Futures/Hyperliquid/agent signer: `internal/hypertrader`, hypertrader migrations.
- Realtime event publishing: `internal/streambridge`, `schemas/mqtt`.
- Gateway compatibility routing: `internal/gateway`, `schemas/graphql`.
- Shared HTTP/errors/money: `internal/httpx`, `pkg/errors`, `pkg/money`, `pkg/requestid`.

For boundary decisions, read `docs/architecture/DOMAIN_BOUNDARIES.md` before changing code.

## Default Doctrine

- Compatibility is not default.
- Fallback is not default.
- Deprecated behavior must not be rescued.
- Unknown caller does not justify compatibility.
- Unknown risk effect requires fail-fast.
- Old behavior is suspect unless registered as active.
- Simple tasks must stay simple.
- Complex signing, risk, order, execution, custody, and realtime work must not be minimized into a fragile patch.

## Required Gates

- Run `python3 tools/discipline/verify.py` before closeout.
- Run `go test ./...` for code changes.
- Complete the review gate in `docs/harness/REVIEW_GATE.md` for C2+ changes.
- Check no-fallback and deprecation decisions for C2+ changes.
- Treat C3+ changes in signer, order, wallet, and risk paths as blocking until verified.

## Closeout

Before final answer or commit:

- run relevant tests/checks
- complete review gate
- check deprecated references
- check unauthorized fallback
- check boundary drift
- record remaining risk
