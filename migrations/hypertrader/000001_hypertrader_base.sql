-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS futures_symbols (
  symbol TEXT PRIMARY KEY,
  alias_name TEXT NOT NULL,
  max_leverage INTEGER NOT NULL DEFAULT 1,
  market_cap NUMERIC(78, 18) NOT NULL DEFAULT 0,
  volume NUMERIC(78, 18) NOT NULL DEFAULT 0,
  change_percent NUMERIC(32, 12) NOT NULL DEFAULT 0,
  open_interest NUMERIC(78, 18) NOT NULL DEFAULT 0,
  current_price NUMERIC(78, 18) NOT NULL DEFAULT 0,
  symbol_type TEXT NOT NULL DEFAULT 'PERP',
  quote_symbol TEXT NOT NULL DEFAULT 'USDC',
  category TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS futures_symbols_category_volume_idx ON futures_symbols(category, volume DESC);

INSERT INTO futures_symbols (symbol, alias_name, max_leverage, market_cap, volume, change_percent, open_interest, current_price, symbol_type, quote_symbol, category)
VALUES
  ('BTC', 'Bitcoin', 50, 1800000000000, 42000000000, 1.82, 12400000000, 95200, 'PERP', 'USDC', 'major'),
  ('ETH', 'Ethereum', 50, 420000000000, 18000000000, 0.95, 6800000000, 3200.5, 'PERP', 'USDC', 'major'),
  ('SOL', 'Solana', 20, 68000000000, 3200000000, 2.41, 960000000, 145.23, 'PERP', 'USDC', 'major'),
  ('HYPE', 'Hyperliquid', 10, 9000000000, 890000000, 4.8, 320000000, 27.4, 'PERP', 'USDC', 'defi')
ON CONFLICT (symbol) DO UPDATE SET
  alias_name = EXCLUDED.alias_name,
  max_leverage = EXCLUDED.max_leverage,
  market_cap = EXCLUDED.market_cap,
  volume = EXCLUDED.volume,
  change_percent = EXCLUDED.change_percent,
  open_interest = EXCLUDED.open_interest,
  current_price = EXCLUDED.current_price,
  symbol_type = EXCLUDED.symbol_type,
  quote_symbol = EXCLUDED.quote_symbol,
  category = EXCLUDED.category,
  updated_at = now();

CREATE TABLE IF NOT EXISTS symbol_preferences (
  user_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  is_favorite BOOLEAN NOT NULL DEFAULT false,
  leverage INTEGER NOT NULL DEFAULT 5,
  is_cross BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, symbol)
);

CREATE TABLE IF NOT EXISTS hyper_orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT,
  user_address TEXT,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  order_type TEXT NOT NULL,
  price NUMERIC(78, 18),
  size NUMERIC(78, 18) NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'pending',
  cloid TEXT,
  raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS hyper_orders_user_created_idx ON hyper_orders(COALESCE(user_id, user_address), created_at DESC);
CREATE INDEX IF NOT EXISTS hyper_orders_symbol_created_idx ON hyper_orders(symbol, created_at DESC);

CREATE TABLE IF NOT EXISTS smart_money_traders (
  user_address TEXT PRIMARY KEY,
  roi NUMERIC(32, 12) NOT NULL DEFAULT 0,
  net_pnl NUMERIC(78, 18) NOT NULL DEFAULT 0,
  avg_win_rate NUMERIC(32, 12) NOT NULL DEFAULT 0,
  max_drawdown NUMERIC(32, 12) NOT NULL DEFAULT 0,
  period_days INTEGER NOT NULL DEFAULT 30,
  sharpe_ratio NUMERIC(32, 12) NOT NULL DEFAULT 0,
  profit_loss_ratio NUMERIC(32, 12) NOT NULL DEFAULT 0,
  profit_factor NUMERIC(32, 12) NOT NULL DEFAULT 0,
  total_volume NUMERIC(78, 18) NOT NULL DEFAULT 0,
  avg_daily_volume NUMERIC(78, 18) NOT NULL DEFAULT 0,
  trading_days INTEGER NOT NULL DEFAULT 0,
  total_trades INTEGER NOT NULL DEFAULT 0,
  unique_coins_count INTEGER NOT NULL DEFAULT 0,
  avg_trades_per_day NUMERIC(32, 12) NOT NULL DEFAULT 0,
  total_long_pnl NUMERIC(78, 18) NOT NULL DEFAULT 0,
  total_short_pnl NUMERIC(78, 18) NOT NULL DEFAULT 0,
  winning_pnl_total NUMERIC(78, 18) NOT NULL DEFAULT 0,
  losing_pnl_total NUMERIC(78, 18) NOT NULL DEFAULT 0,
  kol_labels TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  kol_labels_description TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  follower_count INTEGER NOT NULL DEFAULT 0,
  remark_name TEXT NOT NULL DEFAULT '',
  group_ids TEXT[] NOT NULL DEFAULT ARRAY['default']::TEXT[],
  portfolio_data JSONB NOT NULL DEFAULT '{}'::jsonb,
  last_operation JSONB NOT NULL DEFAULT '{}'::jsonb,
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO smart_money_traders (
  user_address, roi, net_pnl, avg_win_rate, max_drawdown, period_days, sharpe_ratio, profit_loss_ratio, profit_factor,
  total_volume, avg_daily_volume, trading_days, total_trades, unique_coins_count, avg_trades_per_day,
  total_long_pnl, total_short_pnl, winning_pnl_total, losing_pnl_total,
  kol_labels, kol_labels_description, follower_count, remark_name, group_ids, portfolio_data, last_operation, tags
)
VALUES
  ('0xsmart001', 82.4, 42100, 0.62, 0.08, 30, 2.1, 1.8, 2.4, 1250000, 41666, 24, 138, 12, 5.75, 42100, 1200, 42100, -4100, ARRAY['trend'], ARRAY['Trend follower'], 120, 'Smart 001', ARRAY['default'], '{"accountValue":"250000"}', '{"symbol":"BTC","time":0,"pnl":"830","pnlPercent":"0.024","dir":"Open Long","hash":"0xop","oid":1,"px":"95200","startPosition":"0","sz":"0.2","fee":"1.2","feeToken":"USDC","tid":10}', '[{"id":1,"category":"style","name":"trend","nameCn":"趋势","color":"#46C2A9","priority":1,"description":"Trend follower"}]'),
  ('0xsmart002', 63.1, 31880, 0.62, 0.08, 30, 2.1, 1.8, 2.4, 1250000, 41666, 24, 138, 12, 5.75, 31880, 1200, 31880, -4100, ARRAY['trend'], ARRAY['Trend follower'], 120, 'Smart 002', ARRAY['default'], '{"accountValue":"250000"}', '{"symbol":"BTC","time":0,"pnl":"830","pnlPercent":"0.024","dir":"Open Long","hash":"0xop","oid":1,"px":"95200","startPosition":"0","sz":"0.2","fee":"1.2","feeToken":"USDC","tid":10}', '[{"id":1,"category":"style","name":"trend","nameCn":"趋势","color":"#46C2A9","priority":1,"description":"Trend follower"}]'),
  ('0xsmart003', 41.9, 22040, 0.62, 0.08, 30, 2.1, 1.8, 2.4, 1250000, 41666, 24, 138, 12, 5.75, 22040, 1200, 22040, -4100, ARRAY['trend'], ARRAY['Trend follower'], 120, 'Smart 003', ARRAY['default'], '{"accountValue":"250000"}', '{"symbol":"BTC","time":0,"pnl":"830","pnlPercent":"0.024","dir":"Open Long","hash":"0xop","oid":1,"px":"95200","startPosition":"0","sz":"0.2","fee":"1.2","feeToken":"USDC","tid":10}', '[{"id":1,"category":"style","name":"trend","nameCn":"趋势","color":"#46C2A9","priority":1,"description":"Trend follower"}]')
ON CONFLICT (user_address) DO UPDATE SET
  roi = EXCLUDED.roi,
  net_pnl = EXCLUDED.net_pnl,
  updated_at = now();

CREATE TABLE IF NOT EXISTS address_groups (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
  name TEXT NOT NULL,
  user_id TEXT,
  is_default BOOLEAN NOT NULL DEFAULT false,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO address_groups (id, name, user_id, is_default, display_order)
VALUES ('default', 'Default', 'local-user', true, 0)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, is_default = EXCLUDED.is_default, updated_at = now();

CREATE TABLE IF NOT EXISTS followed_addresses (
  id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
  address TEXT NOT NULL,
  remark_name TEXT NOT NULL DEFAULT '',
  group_ids TEXT[] NOT NULL DEFAULT ARRAY['default']::TEXT[],
  owner_user_id TEXT,
  user_address TEXT NOT NULL,
  profit_1d NUMERIC(78, 18) NOT NULL DEFAULT 0,
  profit_7d NUMERIC(78, 18) NOT NULL DEFAULT 0,
  profit_30d NUMERIC(78, 18) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS followed_addresses_address_idx ON followed_addresses(lower(address));
CREATE INDEX IF NOT EXISTS followed_addresses_group_ids_idx ON followed_addresses USING GIN(group_ids);

INSERT INTO followed_addresses (address, remark_name, group_ids, owner_user_id, user_address, profit_1d, profit_7d, profit_30d)
VALUES
  ('0xsmart001', 'Trader 001', ARRAY['default'], 'local-user', '0xsmart001', 1200, 8600, 42100),
  ('0xsmart002', 'Trader 002', ARRAY['default'], 'local-user', '0xsmart002', 1200, 8600, 42100),
  ('0xsmart003', 'Trader 003', ARRAY['default'], 'local-user', '0xsmart003', 1200, 8600, 42100);

-- +goose Down
DROP TABLE IF EXISTS followed_addresses;
DROP TABLE IF EXISTS address_groups;
DROP TABLE IF EXISTS smart_money_traders;
DROP TABLE IF EXISTS hyper_orders;
DROP TABLE IF EXISTS symbol_preferences;
DROP TABLE IF EXISTS futures_symbols;
