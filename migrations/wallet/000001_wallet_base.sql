-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS wallets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  chain_type TEXT NOT NULL,
  address TEXT NOT NULL,
  wallet_type TEXT NOT NULL,
  turnkey_org_id TEXT,
  turnkey_wallet_id TEXT,
  name TEXT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  exported_passphrase_at TIMESTAMPTZ,
  exported_private_key_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS wallets_user_id_idx ON wallets(user_id);
CREATE INDEX IF NOT EXISTS wallets_address_idx ON wallets(lower(address));
CREATE UNIQUE INDEX IF NOT EXISTS wallets_user_chain_address_unique
  ON wallets(user_id, chain_type, lower(address));

CREATE TABLE IF NOT EXISTS wallet_whitelist (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  chain_type TEXT NOT NULL,
  address TEXT NOT NULL,
  label TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS wallet_whitelist_user_chain_address_unique
  ON wallet_whitelist(user_id, chain_type, lower(address));

CREATE TABLE IF NOT EXISTS wallet_security_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  wallet_id UUID,
  action TEXT NOT NULL,
  risk_level TEXT NOT NULL DEFAULT 'low',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS wallet_security_events_user_id_idx ON wallet_security_events(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS turnkey_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL UNIQUE,
  organization_id TEXT NOT NULL UNIQUE,
  sub_organization_id TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS address_groups (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  name TEXT NOT NULL,
  color TEXT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS address_group_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  group_id UUID NOT NULL REFERENCES address_groups(id) ON DELETE CASCADE,
  address TEXT NOT NULL,
  chain_type TEXT NOT NULL,
  label TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS address_group_members_group_chain_address_unique
  ON address_group_members(group_id, chain_type, lower(address));

-- +goose Down
DROP TABLE IF EXISTS address_group_members;
DROP TABLE IF EXISTS address_groups;
DROP TABLE IF EXISTS turnkey_accounts;
DROP TABLE IF EXISTS wallet_security_events;
DROP TABLE IF EXISTS wallet_whitelist;
DROP TABLE IF EXISTS wallets;
