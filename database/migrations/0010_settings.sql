-- ============================================================
-- 0010_settings.sql
-- Super Admin Dashboard (P06): extend the global settings store and
-- introduce the `super_admin` role.
--
-- Additive & idempotent. Reuses the existing `settings` table created in
-- 0009 (key/value JSONB) rather than introducing a duplicate store.
-- ============================================================

-- ------------------------------------------------------------
-- Role: allow `super_admin` alongside `admin` and `user`.
-- ------------------------------------------------------------
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'user', 'super_admin'));

-- ------------------------------------------------------------
-- settings: human-readable description for the settings UI.
-- ------------------------------------------------------------
ALTER TABLE settings ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';

-- ------------------------------------------------------------
-- Seed well-known settings consumed by the Super Admin portal.
-- `updated_by` is left NULL (system default); idempotent.
-- ------------------------------------------------------------
INSERT INTO settings (key, value, description) VALUES
    ('payment_gateway_default', '"manual"'::jsonb, 'Gateway pembayaran default (midtrans|manual)'),
    ('grace_period_hours',      '72'::jsonb,        'Jam masa tenggang sebelum fitur dibekukan'),
    ('tax_rate_percent',        '0'::jsonb,         'Persentase PPN/PPN (integer, 0-100)'),
    ('invoice_due_hours',       '24'::jsonb,        'Jam jatuh tempo invoice sejak diterbitkan'),
    ('ai_kill_switch',          '{}'::jsonb,        'Status Emergency Kill Switch AI Gateway per modalitas'),
    ('feature_flags',           '{}'::jsonb,        'Toggle fitur eksperimental global')
ON CONFLICT (key) DO UPDATE
    SET description = EXCLUDED.description
    WHERE settings.description = '';

-- ------------------------------------------------------------
-- payments: admin review & refund columns (P06-012).
-- Additive; `status` remains the optimistic concurrency guard for review
-- (review only succeeds from 'pending_review'), so no version column needed.
-- ------------------------------------------------------------
ALTER TABLE payments ADD COLUMN IF NOT EXISTS reviewed_by     uuid REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS reviewed_at     timestamptz;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS review_reason   text;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS refund_amount_idr bigint NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS refund_reason   text;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS refund_ref      varchar(128);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS refunded_at     timestamptz;

-- ------------------------------------------------------------
-- payment_events — append-only history of admin actions on a payment.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payment_events (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id uuid NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    actor_id   uuid REFERENCES users(id) ON DELETE SET NULL,
    event      text NOT NULL,          -- approve | reject | refund | settle
    reason     text,
    amount_idr bigint NOT NULL DEFAULT 0,
    metadata   jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_events_payment ON payment_events (payment_id, created_at DESC);
