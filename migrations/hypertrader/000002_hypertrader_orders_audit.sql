-- +goose Up
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'local-hyperliquid';
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS provider_order_id TEXT;
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS client_request_id TEXT;
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS reduce_only BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS time_in_force TEXT;
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS response_payload JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ;
ALTER TABLE hyper_orders ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS hyper_orders_user_client_request_unique
  ON hyper_orders(user_id, client_request_id)
  WHERE client_request_id IS NOT NULL AND client_request_id <> '' AND user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS hyper_orders_user_status_updated_idx
  ON hyper_orders(COALESCE(user_id, user_address), status, updated_at DESC);

CREATE TABLE IF NOT EXISTS hyper_audit_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT,
  user_address TEXT,
  action TEXT NOT NULL,
  risk_level TEXT NOT NULL DEFAULT 'low',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS hyper_audit_events_user_created_idx
  ON hyper_audit_events(COALESCE(user_id, user_address), created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS hyper_audit_events;
DROP INDEX IF EXISTS hyper_orders_user_status_updated_idx;
DROP INDEX IF EXISTS hyper_orders_user_client_request_unique;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS submitted_at;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS response_payload;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS time_in_force;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS reduce_only;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS client_request_id;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS provider_order_id;
ALTER TABLE hyper_orders DROP COLUMN IF EXISTS provider;
