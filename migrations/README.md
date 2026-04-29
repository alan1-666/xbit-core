# Database Migrations

Each service owns its migrations under a subdirectory:

```text
migrations/
├── identity/
├── wallet/
├── trading/
├── market-data/
└── hypertrader/
```

Use goose or atlas once the first service schemas are implemented.

Current initial migration sets:

- `identity/000001_identity_base.sql`
- `wallet/000001_wallet_base.sql`
- `trading/000001_trading_base.sql`
- `market-data/000001_market_data_base.sql`
- `hypertrader/000001_hypertrader_base.sql`
- `hypertrader/000002_hypertrader_orders_audit.sql`
- `hypertrader/000003_hypertrader_live_state.sql`
