-- +goose Up
CREATE TABLE IF NOT EXISTS hyper_agent_wallets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT,
  user_address TEXT NOT NULL,
  agent_address TEXT NOT NULL,
  agent_name TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending_approval',
  key_ref TEXT,
  policy JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  approved_at TIMESTAMPTZ,
  UNIQUE (user_address, agent_address)
);

CREATE INDEX IF NOT EXISTS hyper_agent_wallets_user_status_idx
  ON hyper_agent_wallets(lower(user_address), status, updated_at DESC);

CREATE TABLE IF NOT EXISTS hyper_agent_nonces (
  agent_address TEXT PRIMARY KEY,
  last_nonce BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS hyper_agent_nonces;
DROP TABLE IF EXISTS hyper_agent_wallets;
