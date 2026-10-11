-- ============================================================
-- 0009_payments_invoices.sql
-- Payments & Billing: invoices, payments, and the global settings store.
--
-- Guarantees:
--   * Additive & idempotent (CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS).
--   * Money = BIGINT (integer Rupiah). No floats.
--   * Idempotent webhooks: payments.idempotency_key UNIQUE.
--   * PCI-DSS SAQ-A: no card data is ever stored.
--
-- Idempotent: safe to re-run.
-- ============================================================

-- ------------------------------------------------------------
-- settings — global key/value configuration (used by P05/P06)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS settings (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- invoices
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invoices (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number  varchar(64) NOT NULL UNIQUE,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id uuid REFERENCES subscriptions(id) ON DELETE SET NULL,
    subtotal_idr    bigint NOT NULL DEFAULT 0 CHECK (subtotal_idr >= 0),
    tax_idr         bigint NOT NULL DEFAULT 0 CHECK (tax_idr >= 0),
    total_idr       bigint NOT NULL DEFAULT 0 CHECK (total_idr >= 0),
    status          varchar(16) NOT NULL DEFAULT 'issued'
                    CHECK (status IN ('draft','issued','paid','past_due','voided')),
    notes           text,
    issued_at       timestamptz NOT NULL DEFAULT now(),
    due_at          timestamptz,
    paid_at         timestamptz,
    voided_at       timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_invoice_updated
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_invoices_user_id ON invoices (user_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status  ON invoices (status);
CREATE INDEX IF NOT EXISTS idx_invoices_due_at  ON invoices (due_at);

-- ------------------------------------------------------------
-- payments
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS payments (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id       uuid NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    gateway          varchar(32) NOT NULL, -- 'midtrans' | 'manual'
    gateway_ref      varchar(128),         -- order_id / external reference
    amount_idr       bigint NOT NULL CHECK (amount_idr > 0),
    status           varchar(20) NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','pending_review','settled','failed','expired','refunded')),
    payment_method   varchar(64),
    proof_url        text,
    idempotency_key  varchar(128) NOT NULL UNIQUE,
    payload_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_payment_updated
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_payments_invoice_id   ON payments (invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id      ON payments (user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status       ON payments (status);
CREATE INDEX IF NOT EXISTS idx_payments_gateway_ref  ON payments (gateway_ref);

-- ------------------------------------------------------------
-- invoice number sequence helper: INV/YYYYMMDD/NNNNN
-- Uses a per-day counter row locked atomically (safe across instances).
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS invoice_counters (
    day   date PRIMARY KEY,
    seq   bigint NOT NULL DEFAULT 0
);

-- ------------------------------------------------------------
-- Seed default settings (idempotent)
-- ------------------------------------------------------------
INSERT INTO settings (key, value) VALUES
    ('payment_gateway_default', '"manual"'::jsonb),
    ('tax_rate_percent',        '0'::jsonb),
    ('grace_period_hours',      '72'::jsonb),
    ('invoice_due_hours',       '24'::jsonb)
ON CONFLICT (key) DO NOTHING;
