-- +goose Up
CREATE TABLE IF NOT EXISTS hyper_open_order_snapshot_meta (
  user_address TEXT PRIMARY KEY,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS hyper_open_order_snapshot_meta_user_idx
  ON hyper_open_order_snapshot_meta(lower(user_address));

CREATE TABLE IF NOT EXISTS hyper_open_order_snapshots (
  user_address TEXT NOT NULL,
  provider_order_id TEXT NOT NULL,
  order_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'open',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_address, provider_order_id)
);

CREATE INDEX IF NOT EXISTS hyper_open_order_snapshots_user_updated_idx
  ON hyper_open_order_snapshots(lower(user_address), updated_at DESC);

CREATE TABLE IF NOT EXISTS hyper_fills (
  id TEXT PRIMARY KEY,
  user_address TEXT NOT NULL,
  symbol TEXT NOT NULL,
  provider_order_id TEXT,
  fill_time BIGINT NOT NULL DEFAULT 0,
  fill_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS hyper_fills_user_time_idx
  ON hyper_fills(lower(user_address), fill_time DESC);

CREATE TABLE IF NOT EXISTS hyper_account_snapshots (
  user_address TEXT PRIMARY KEY,
  account_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS hyper_account_snapshots;
DROP TABLE IF EXISTS hyper_fills;
DROP TABLE IF EXISTS hyper_open_order_snapshots;
DROP TABLE IF EXISTS hyper_open_order_snapshot_meta;
