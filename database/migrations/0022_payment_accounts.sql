-- ============================================================
-- 0022_payment_accounts.sql
-- Multiple manual payment destinations (bank accounts + e-wallets).
--
-- Before this, the manual gateway exposed exactly one bank account, configured
-- as three loose settings (`manual_bank_name` / `manual_account_number` /
-- `manual_account_holder`). Operators need several destinations (e.g. BCA,
-- DANA, SeaBank, GoPay, QRIS), so they live in a real table with ordering and
-- an active flag instead of an ever-growing set of settings keys.
--
-- The old settings remain as a fallback: a deployment with no rows still shows
-- the single configured account, so nothing breaks on upgrade.
--
-- Additive & idempotent.
-- ============================================================

CREATE TABLE IF NOT EXISTS payment_accounts (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- kind drives the icon/label in the UI: bank | ewallet | qris
    kind           varchar(16) NOT NULL DEFAULT 'bank'
        CHECK (kind IN ('bank', 'ewallet', 'qris')),
    -- Display name, e.g. "BCA", "DANA", "SeaBank", "GoPay", "QRIS".
    name           varchar(64) NOT NULL,
    account_number varchar(64) NOT NULL DEFAULT '',
    account_holder varchar(128) NOT NULL DEFAULT '',
    -- Optional QRIS image URL (only meaningful for kind = 'qris').
    qris_url       text NOT NULL DEFAULT '',
    -- Lower sorts first; the checkout modal lists them in this order.
    sort_order     integer NOT NULL DEFAULT 0,
    is_active      boolean NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- The only hot read is "active accounts, in display order".
CREATE INDEX IF NOT EXISTS idx_payment_accounts_active
    ON payment_accounts (is_active, sort_order);

-- ------------------------------------------------------------
-- Backfill: if a single manual account was already configured via settings,
-- promote it to a row so the operator sees it in the new UI. Idempotent: only
-- runs when the table is empty and the settings hold a real value.
-- ------------------------------------------------------------
INSERT INTO payment_accounts (kind, name, account_number, account_holder, sort_order, is_active)
SELECT
    'bank',
    COALESCE(NULLIF((SELECT value #>> '{}' FROM settings WHERE key = 'manual_bank_name'), ''), 'Transfer Bank'),
    COALESCE(NULLIF((SELECT value #>> '{}' FROM settings WHERE key = 'manual_account_number'), ''), ''),
    COALESCE(NULLIF((SELECT value #>> '{}' FROM settings WHERE key = 'manual_account_holder'), ''), ''),
    0,
    true
WHERE NOT EXISTS (SELECT 1 FROM payment_accounts)
  AND COALESCE((SELECT value #>> '{}' FROM settings WHERE key = 'manual_account_number'), '') <> '';
