# XBIT Backend

Go backend scaffold for XBIT Web V2.

This is a standalone backend service directory. The frontend app remains in:

```text
/Users/zhangza/code/project/app-web-v2
```

The backend docs currently live with the frontend analysis docs:

- `../app-web-v2/BACKEND_DESIGN.md`
- `../app-web-v2/BACKEND_FRONTEND_CONTRACT.md`
- `../app-web-v2/BACKEND_GO_DEVELOPMENT_PLAN.md`

## What Is Implemented

- Phase 0 gateway skeleton.
- Phase 1 initial identity foundation.
- Current frontend GraphQL endpoint routing table.
- Health/readiness endpoints.
- Request ID, structured access logs, panic recovery and CORS.
- GraphQL-compatible error response helpers.
- JWT access token issuing/parsing and opaque refresh token hashing.
- Wallet login nonce challenge creation.
- Refresh token rotation and logout session revocation.
- Wallet service create/list/rename/order, whitelist and audit-event APIs.
- Trading MVP quote/order/status/network-fee APIs.
- Market-data MVP token list/detail/search/OHLC/transaction/pool APIs and indexer ingest checkpoints.
- Stream-bridge MVP event ingest, topic mapping, MQTT publishing and recent-event replay.
- Hypertrader MVP futures symbols/account/positions/smart-money/address-group APIs, order lifecycle, funding rates, audit events and GraphQL facades.
- Initial identity, wallet, trading, market-data and hypertrader migrations.
- Local Docker Compose for Postgres, Redis, Kafka and EMQX.

## Quick Start

```bash
cd xbit-backend
cp .env.example .env
set -a && source .env && set +a
make run-gateway
```

Gateway endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /api/meme/graphql`
- `POST /api/meme2/meme-gql`
- `POST /api/trading/trading-gql`
- `POST /api/user/user-gql`
- and the other frontend-compatible GraphQL paths in `internal/gateway/config.go`

Local infra:

```bash
make docker-up
make docker-down
```

Quality checks:

```bash
make fmt
make test
```

Apply migrations when `POSTGRES_DSN` points at a database:

```bash
make migrate-up SERVICE=identity
make migrate-up SERVICE=wallet
make migrate-up SERVICE=trading
make migrate-up SERVICE=market-data
make migrate-up SERVICE=hypertrader
make migrate-status SERVICE=identity
```

## Frontend Local Routing

To route the frontend through this gateway, set the frontend GraphQL env vars to `http://localhost:8080` paths, for example:

```env
VITE_GRAPHQL_HTTP_URL=http://localhost:8080/api/meme/graphql
VITE_GRAPHQL_MEME2_URL=http://localhost:8080/api/meme2/meme-gql
VITE_GRAPHQL_TRADING_HTTP_URL=http://localhost:8080/api/trading/trading-gql
VITE_GRAPHQL_USER_HTTP_URL=http://localhost:8080/api/user/user-gql
```

The gateway itself uses `XBIT_UPSTREAM_*` variables to decide where each request is proxied.
For local market-data testing, point `XBIT_UPSTREAM_MEME2_GRAPHQL_URL` and optionally `XBIT_UPSTREAM_CORE_GRAPHQL_URL` to `http://localhost:8084/graphql`.
For local futures/Hyperliquid testing, point `XBIT_UPSTREAM_SYMBOL_DEX_GRAPHQL_URL` and `XBIT_UPSTREAM_DEX_HYPERTRADER_GRAPHQL_URL` to `http://localhost:8086/graphql`. `XBIT_UPSTREAM_USER_GRAPHQL_URL` can point there only for isolated Hyperliquid signing smoke tests; normal user gql traffic should stay on the identity/user upstream.

## Identity Service

Run:

```bash
cd xbit-backend
SERVICE_ADDR=:8081 JWT_SIGNING_KEY=dev-secret go run ./cmd/identity
```

Implemented endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /v1/auth/nonce`
- `POST /v1/auth/dev-login`
- `POST /v1/auth/refresh`
- `POST /v1/auth/logout`
- `GET /v1/auth/me`

Example:

```bash
curl -sS -X POST http://127.0.0.1:8081/v1/auth/nonce \
  -H 'content-type: application/json' \
  -d '{"walletAddress":"0xabc","chainType":"EVM"}'

curl -sS -X POST http://127.0.0.1:8081/v1/auth/dev-login \
  -H 'content-type: application/json' \
  -d '{"userId":"local-user","deviceId":"local-device"}'
```

`/v1/auth/dev-login` is controlled by `DEV_AUTH_ENABLED` and is only for local integration until the real wallet/OAuth flows are wired.

## Wallet Service

Run:

```bash
cd xbit-backend
SERVICE_ADDR=:8082 go run ./cmd/wallet
```

Implemented endpoints:

- `GET /healthz`
- `GET /readyz`
- `GET /v1/wallets?userId=...`
- `POST /v1/wallets`
- `PATCH /v1/wallets/{walletId}`
- `PATCH /v1/wallets/order`
- `GET /v1/wallet-whitelist?userId=...`
- `POST /v1/wallet-whitelist`
- `POST /v1/wallet-security-events`

Examples:

```bash
curl -sS -X POST http://127.0.0.1:8082/v1/wallets \
  -H 'content-type: application/json' \
  -d '{"userId":"local-user","chainType":"EVM","address":"0xabc","walletType":"embedded","name":"Main"}'

curl -sS 'http://127.0.0.1:8082/v1/wallets?userId=local-user'
```

Like identity, the service uses an in-memory store when `POSTGRES_DSN` is empty, and switches to Postgres when `POSTGRES_DSN` is set.

## Trading Service

Run:

```bash
cd xbit-backend
SERVICE_ADDR=:8083 go run ./cmd/trading
```

Implemented endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /graphql`
- `POST /trading-gql`
- `POST /api/trading/trading-gql`
- `GET /v1/trading/exchange-meta`
- `POST /v1/trading/quote`
- `GET /v1/trading/network-fee?chainType=...`
- `GET /v1/trading/orders?userId=...&status=...`
- `POST /v1/trading/orders`
- `GET /v1/trading/orders/{orderId}`
- `POST /v1/trading/orders/{orderId}/status`
- `POST /v1/trading/orders/{orderId}/cancel`

Examples:

```bash
curl -sS -X POST http://127.0.0.1:8083/v1/trading/quote \
  -H 'content-type: application/json' \
  -d '{"userId":"local-user","chainType":"EVM","inputToken":"ETH","outputToken":"USDC","inputAmount":"1","slippageBps":100}'

curl -sS -X POST http://127.0.0.1:8083/v1/trading/orders \
  -H 'content-type: application/json' \
  -d '{"userId":"local-user","chainType":"EVM","walletAddress":"0xabc","orderType":"market","side":"buy","inputToken":"ETH","outputToken":"USDC","inputAmount":"1","clientRequestId":"local-req-1"}'
```

Like the earlier services, trading uses an in-memory store when `POSTGRES_DSN` is empty, and switches to Postgres when `POSTGRES_DSN` is set.

The GraphQL facade currently covers the frontend's core trading operations: `getExchangeMeta`, `getExchangeMetaV2`, `getNetworkFee`, `createOrder`, `saveWeb3Order`, `cancelOrder`, `getPendingOrders`, `orders`, order history/transaction list reads and MVP swap route/tx operations.

## Market Data Service

Run:

```bash
cd xbit-backend
SERVICE_ADDR=:8084 go run ./cmd/market-data
```

Implemented endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /graphql`
- `POST /meme-gql`
- `POST /api/meme2/meme-gql`
- `POST /api/meme/graphql`
- `GET /v1/market/tokens`
- `GET /v1/market/tokens/search?q=...`
- `POST /v1/market/tokens`
- `GET /v1/market/tokens/{chainId}/{address}`
- `GET /v1/market/tokens/{chainId}/{address}/ohlc`
- `GET /v1/market/tokens/{chainId}/{address}/transactions`
- `GET /v1/market/tokens/{chainId}/{address}/pools`
- `GET /v1/market/categories`
- `POST /v1/indexer/tokens`
- `POST /v1/indexer/transactions`
- `GET /v1/indexer/checkpoints/{source}`
- `PUT /v1/indexer/checkpoints/{source}`

Examples:

```bash
curl -sS 'http://127.0.0.1:8084/v1/market/tokens?limit=5'
curl -sS 'http://127.0.0.1:8084/v1/market/tokens/search?q=sol'
curl -sS 'http://127.0.0.1:8084/v1/market/tokens/501/So11111111111111111111111111111111111111112/ohlc?limit=20'
```

The GraphQL facade currently covers the frontend's high-use meme/token operations: trending/new/meme/favorite token lists, `tokens`, `getTokenDetail`, prices, metadata, search, OHLC, categories, token transactions, pool transactions, token pools and token symbols. It returns seeded read-model data in memory and switches to Postgres when `POSTGRES_DSN` is set.

## Hypertrader Service

Run:

```bash
cd xbit-backend
SERVICE_ADDR=:8086 go run ./cmd/hypertrader
```

Run with Hyperliquid HTTP reads enabled:

```bash
HYPERLIQUID_PROVIDER_MODE=http \
HYPERLIQUID_API_URL=https://api.hyperliquid.xyz \
SERVICE_ADDR=:8086 go run ./cmd/hypertrader
```

Implemented endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /graphql`
- `POST /api/graphql-dex`
- `POST /api/dex-hypertrader/graphql`
- `POST /api/user/user-gql`
- `GET /v1/futures/symbols`
- `GET /v1/futures/account?userAddress=...`
- `GET /v1/futures/trades`
- `GET /v1/futures/smart-money`
- `GET /v1/futures/funding-rates`
- `GET /v1/futures/orders?userId=...&status=...`
- `POST /v1/futures/orders`
- `POST /v1/futures/orders/{orderId}/cancel`
- `POST /v1/futures/orders/{orderId}/sync`
- `POST /v1/futures/leverage`
- `GET /v1/futures/audit-events?userId=...`

Examples:

```bash
curl -sS 'http://127.0.0.1:8086/v1/futures/symbols?limit=5'
curl -sS 'http://127.0.0.1:8086/v1/futures/account?userAddress=0xabc'
curl -sS -X POST http://127.0.0.1:8086/v1/futures/orders \
  -H 'content-type: application/json' \
  -d '{"userId":"local-user","userAddress":"0xabc","symbol":"BTC","side":"buy","orderType":"market","size":"0.1","clientRequestId":"local-futures-1"}'
curl -sS -X POST http://127.0.0.1:8086/v1/futures/orders/{orderId}/sync \
  -H 'content-type: application/json' \
  -d '{}'
```

The GraphQL facade currently covers the frontend's futures symbol, account, Hyperliquid signing, order lifecycle, order status sync and Smart Money address-management operations. It uses seeded in-memory data and the local Hyperliquid provider locally, then switches persistence to Postgres when `POSTGRES_DSN` is set.

Provider modes:

- `HYPERLIQUID_PROVIDER_MODE=local`: default for local development; deterministic signing, seeded account/funding data.
- `HYPERLIQUID_PROVIDER_MODE=http`: reads account state, funding history and order status from Hyperliquid `/info`; `/exchange` writes are forwarded only when the request contains an already-signed `exchangePayload` with `action`, `nonce` and `signature`.

## Stream Bridge Service

Run in dry-run/in-memory mode:

```bash
cd xbit-backend
SERVICE_ADDR=:8085 go run ./cmd/stream-bridge
```

Run with local EMQX:

```bash
MQTT_ENABLED=true MQTT_BROKER_URL=tcp://localhost:1883 SERVICE_ADDR=:8085 go run ./cmd/stream-bridge
```

Implemented endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /v1/stream/events`
- `POST /v1/stream/events/batch`
- `GET /v1/stream/topics`
- `GET /v1/stream/events?topic=...&limit=...`

Examples:

```bash
curl -sS -X POST http://127.0.0.1:8085/v1/stream/events \
  -H 'content-type: application/json' \
  -d '{"type":"market.ohlc.updated","chainId":"501","token":"xbit-demo-token","bucket":"1m","payload":{"close":"0.042"}}'
```

Current topic mapping covers market token creation/update/statistics, transaction stream, K-line updates, network fee updates and private trading order updates. Payloads are wrapped in the MQTT envelope documented in `schemas/mqtt/README.md`.

## Migrations

Initial migration sets:

- `migrations/identity/000001_identity_base.sql`
- `migrations/wallet/000001_wallet_base.sql`
- `migrations/trading/000001_trading_base.sql`
- `migrations/market-data/000001_market_data_base.sql`
- `migrations/hypertrader/000001_hypertrader_base.sql`
- `migrations/hypertrader/000002_hypertrader_orders_audit.sql`
