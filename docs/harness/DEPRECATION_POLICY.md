# DEPRECATION_POLICY.md

Deprecated behavior must be migrated, deleted, or registered.

## Status Values

- active: current source of truth
- migrating: replacement exists, callers are being moved
- deprecated: should not gain new callers
- archived: history only

## Rules

- Deprecated behavior must not be rescued by compatibility unless the caller is known.
- Compatibility routers may exist only while the migration owner and removal condition are clear.
- Generated schemas may contain deprecated fields, but active code should not add new dependencies on them.
- A deprecated endpoint kept for frontend compatibility must be listed in a service README or contract doc.

## Current Known Compatibility Areas

- Frontend-compatible GraphQL paths in `internal/gateway/config.go`.
- Hypertrader GraphQL facade paths documented in `README.md`.
- Generated schema comments under `schemas/` are descriptive, not active policy.
