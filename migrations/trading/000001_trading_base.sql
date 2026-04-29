-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  chain_type TEXT NOT NULL,
  wallet_address TEXT NOT NULL,
  order_type TEXT NOT NULL,
  side TEXT NOT NULL,
  input_token TEXT NOT NULL,
  output_token TEXT NOT NULL,
  input_amount NUMERIC(78, 18) NOT NULL,
  expected_output_amount NUMERIC(78, 18) NOT NULL,
  min_output_amount NUMERIC(78, 18) NOT NULL,
  slippage_bps INTEGER NOT NULL DEFAULT 100,
  route_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'pending',
  tx_hash TEXT,
  failure_code TEXT,
  client_request_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  filled_at TIMESTAMPTZ,
  expired_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS orders_user_status_created_idx ON orders(user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS orders_wallet_created_idx ON orders(lower(wallet_address), created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS orders_user_client_request_unique
  ON orders(user_id, client_request_id)
  WHERE client_request_id IS NOT NULL AND client_request_id <> '';

CREATE TABLE IF NOT EXISTS order_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_events_order_created_idx ON order_events(order_id, created_at ASC);

CREATE TABLE IF NOT EXISTS quote_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT,
  chain_type TEXT NOT NULL,
  input_token TEXT NOT NULL,
  output_token TEXT NOT NULL,
  input_amount NUMERIC(78, 18) NOT NULL,
  output_amount NUMERIC(78, 18) NOT NULL,
  min_output_amount NUMERIC(78, 18) NOT NULL,
  slippage_bps INTEGER NOT NULL,
  platform_fee_amount NUMERIC(78, 18) NOT NULL DEFAULT 0,
  route_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS quote_snapshots_user_created_idx ON quote_snapshots(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS network_fee_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  chain_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  source TEXT NOT NULL DEFAULT 'trading-svc',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS network_fee_snapshots_chain_created_idx ON network_fee_snapshots(chain_type, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS network_fee_snapshots;
DROP TABLE IF EXISTS quote_snapshots;
DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS orders;
