-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS market_tokens (
  id UUID NOT NULL DEFAULT gen_random_uuid(),
  chain_id INTEGER NOT NULL,
  address TEXT NOT NULL,
  symbol TEXT NOT NULL,
  name TEXT NOT NULL,
  decimals INTEGER NOT NULL DEFAULT 18,
  logo_url TEXT,
  price NUMERIC(78, 18) NOT NULL DEFAULT 0,
  price_24h_change NUMERIC(78, 18) NOT NULL DEFAULT 0,
  market_cap NUMERIC(78, 18) NOT NULL DEFAULT 0,
  liquidity NUMERIC(78, 18) NOT NULL DEFAULT 0,
  volume_24h NUMERIC(78, 18) NOT NULL DEFAULT 0,
  holders BIGINT NOT NULL DEFAULT 0,
  dexes TEXT[] NOT NULL DEFAULT '{}',
  category TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, address)
);

CREATE UNIQUE INDEX IF NOT EXISTS market_tokens_chain_lower_address_unique ON market_tokens(chain_id, lower(address));
CREATE INDEX IF NOT EXISTS market_tokens_category_volume_idx ON market_tokens(category, volume_24h DESC);
CREATE INDEX IF NOT EXISTS market_tokens_symbol_idx ON market_tokens(lower(symbol));
CREATE INDEX IF NOT EXISTS market_tokens_updated_idx ON market_tokens(updated_at DESC);

CREATE TABLE IF NOT EXISTS token_ohlc (
  chain_id INTEGER NOT NULL,
  token TEXT NOT NULL,
  bucket TEXT NOT NULL,
  ts BIGINT NOT NULL,
  open NUMERIC(78, 18) NOT NULL,
  high NUMERIC(78, 18) NOT NULL,
  low NUMERIC(78, 18) NOT NULL,
  close NUMERIC(78, 18) NOT NULL,
  token_volume NUMERIC(78, 18) NOT NULL DEFAULT 0,
  usd_volume NUMERIC(78, 18) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, token, bucket, ts)
);

CREATE INDEX IF NOT EXISTS token_ohlc_lookup_idx ON token_ohlc(chain_id, lower(token), bucket, ts DESC);

CREATE TABLE IF NOT EXISTS token_transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  chain_id INTEGER NOT NULL,
  token TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  log_index INTEGER NOT NULL DEFAULT 0,
  event_index INTEGER NOT NULL DEFAULT 0,
  base_token TEXT NOT NULL,
  quote_token TEXT NOT NULL,
  pair TEXT NOT NULL,
  side TEXT NOT NULL,
  maker TEXT NOT NULL,
  base_amount NUMERIC(78, 18) NOT NULL DEFAULT 0,
  quote_amount NUMERIC(78, 18) NOT NULL DEFAULT 0,
  price NUMERIC(78, 18) NOT NULL DEFAULT 0,
  usd_amount NUMERIC(78, 18) NOT NULL DEFAULT 0,
  usd_price NUMERIC(78, 18) NOT NULL DEFAULT 0,
  liquidity NUMERIC(78, 18) NOT NULL DEFAULT 0,
  dex TEXT NOT NULL DEFAULT '',
  timestamp BIGINT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS token_transactions_tx_log_unique ON token_transactions(chain_id, tx_hash, log_index);
CREATE INDEX IF NOT EXISTS token_transactions_token_ts_idx ON token_transactions(chain_id, lower(base_token), timestamp DESC);

CREATE TABLE IF NOT EXISTS token_pools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  chain_id INTEGER NOT NULL,
  address TEXT NOT NULL,
  base_token TEXT NOT NULL,
  quote_token TEXT NOT NULL,
  quote_token_price NUMERIC(78, 18) NOT NULL DEFAULT 0,
  quote_symbol TEXT NOT NULL DEFAULT '',
  base_symbol TEXT NOT NULL DEFAULT '',
  base_token_liquidity NUMERIC(78, 18) NOT NULL DEFAULT 0,
  quote_liquidity NUMERIC(78, 18) NOT NULL DEFAULT 0,
  usd_liquidity NUMERIC(78, 18) NOT NULL DEFAULT 0,
  dex TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS token_pools_chain_address_unique ON token_pools(chain_id, lower(address));
CREATE INDEX IF NOT EXISTS token_pools_base_token_idx ON token_pools(chain_id, lower(base_token), usd_liquidity DESC);

CREATE TABLE IF NOT EXISTS indexer_checkpoints (
  source TEXT PRIMARY KEY,
  cursor TEXT NOT NULL DEFAULT '',
  block_number BIGINT NOT NULL DEFAULT 0,
  event_ts BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS indexer_checkpoints;
DROP TABLE IF EXISTS token_pools;
DROP TABLE IF EXISTS token_transactions;
DROP TABLE IF EXISTS token_ohlc;
DROP INDEX IF EXISTS market_tokens_updated_idx;
DROP INDEX IF EXISTS market_tokens_symbol_idx;
DROP INDEX IF EXISTS market_tokens_category_volume_idx;
DROP INDEX IF EXISTS market_tokens_chain_lower_address_unique;
DROP TABLE IF EXISTS market_tokens;
