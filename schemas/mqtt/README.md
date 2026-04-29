# MQTT Schemas

MQTT payload schemas are versioned here.

Keep compatibility with the frontend topics documented in `../BACKEND_FRONTEND_CONTRACT.md`.

## Envelope

All new stream-bridge publishes use this envelope. Legacy consumers can keep reading `payload` fields while new consumers can use `seq` and `ts` for ordering.

```json
{
  "id": "event-id",
  "topic": "public/token_statistic/501/xbit-demo-token",
  "type": "market.token.statistic_updated",
  "source": "market-data",
  "aggregateId": "optional-source-id",
  "userId": "",
  "chainId": "501",
  "token": "xbit-demo-token",
  "ts": 1770000000,
  "seq": 1,
  "payload": {}
}
```

## Event To Topic Mapping

| Event type | Topic |
|---|---|
| `market.token.created` | `public/token/new`, plus `public/meme/new` when payload category is `meme` |
| `market.token.updated` | `public/meme/token_info/{chainId}/{token}` and `public/token_statistic/{chainId}/{token}` |
| `market.token.statistic_updated` | `public/token_statistic/{chainId}/{token}` |
| `market.transaction.created` | `public/transaction/new/{chainId}/{token}` |
| `market.ohlc.updated` | `public/kline/ohlc_{bucket}/{token}` |
| `network.fee.updated` | `public/network_fee_updated/{chain}` |
| `trading.order.updated` | `users/{userId}/order_updated` |
| `trading.order.submit_failed` | `users/{userId}/order_submit_failed` |
| `trading.order.confirmed` | `users/{userId}/order_confirmation` |
| `hypertrader.order.updated` | `users/{userId}/hypertrader/order_updated` |
| `hypertrader.fill.created` | `users/{userId}/hypertrader/fill_created` |
| `hypertrader.open_orders.snapshot` | `users/{userId}/hypertrader/open_orders` |
| `hypertrader.account.updated` | `users/{userId}/hypertrader/account_updated` |
| `hypertrader.position.updated` | `users/{userId}/hypertrader/position_updated` |
| `hypertrader.funding.updated` | `users/{userId}/hypertrader/funding_updated` |
| `hypertrader.ledger.updated` | `users/{userId}/hypertrader/ledger_updated` |
| `hypertrader.event` | `users/{userId}/hypertrader/event` |

Explicit `topic` on an event bypasses mapping and publishes to that topic.
