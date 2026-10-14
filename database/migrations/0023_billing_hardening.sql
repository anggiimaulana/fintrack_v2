-- ============================================================
-- 0023_billing_hardening.sql
-- Billing lifecycle hardening: cancellable invoices, auto-expiry, dynamic tax,
-- and a guarantee that a user has at most one open (unpaid) invoice.
--
-- Problems addressed:
--   1. A user could create unlimited unpaid invoices by clicking "Upgrade"
--      repeatedly — each click issued a new invoice.
--   2. Nothing ever expired an unpaid invoice, so they accumulated forever.
--   3. A user could not cancel an invoice they no longer wanted to pay.
--   4. PPN was effectively always-on at a rate that could not be toggled.
--
-- Additive & idempotent.
-- ============================================================

-- ------------------------------------------------------------
-- 1. New invoice status: 'expired'.
--    'voided' already exists and doubles as the user-cancelled state.
-- ------------------------------------------------------------
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices
    ADD CONSTRAINT invoices_status_check
    CHECK (status IN ('draft', 'issued', 'paid', 'past_due', 'voided', 'expired'));

-- ------------------------------------------------------------
-- 2. Track WHY an invoice closed, and when it expired, so the UI can explain
--    ("dibatalkan oleh Anda" vs "kedaluwarsa otomatis") and ops can audit it.
-- ------------------------------------------------------------
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS closed_reason text;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS expired_at timestamptz;

-- ------------------------------------------------------------
-- 3. Guarantee at most ONE open invoice per user.
--
--    An "open" invoice is issued or past_due (i.e. unpaid and not yet closed).
--    A partial unique index makes the database — not application code — the
--    final authority, so two concurrent checkouts can never both succeed.
--
--    First, close any duplicates that already exist (keep the newest), so the
--    index can be created on a dirty database.
-- ------------------------------------------------------------
WITH ranked AS (
    SELECT id,
           row_number() OVER (PARTITION BY user_id ORDER BY created_at DESC, id DESC) AS rn
    FROM invoices
    WHERE status IN ('issued', 'past_due')
)
UPDATE invoices i
SET status = 'voided',
    closed_reason = 'superseded_by_unique_open_invoice_constraint',
    voided_at = now(),
    updated_at = now()
FROM ranked r
WHERE i.id = r.id AND r.rn > 1;

-- Also close any open invoices already past their due date, so the index and
-- the new expiry semantics agree from the start.
UPDATE invoices
SET status = 'expired',
    closed_reason = 'expired_backfill',
    expired_at = now(),
    updated_at = now()
WHERE status IN ('issued', 'past_due')
  AND due_at IS NOT NULL
  AND due_at < now();

CREATE UNIQUE INDEX IF NOT EXISTS invoices_one_open_per_user
    ON invoices (user_id)
    WHERE status IN ('issued', 'past_due');

-- ------------------------------------------------------------
-- 4. Dynamic tax settings.
--    `tax_enabled` turns PPN on/off; `tax_rate_percent` (already present) sets
--    the rate. Both are read at invoice time, so a change applies to the next
--    invoice without a deploy.
-- ------------------------------------------------------------
INSERT INTO settings (key, value, description) VALUES
    ('tax_enabled',      'false'::jsonb, 'Aktifkan PPN pada faktur (true/false)'),
    ('tax_label',        '"PPN"'::jsonb, 'Label pajak yang ditampilkan pada faktur'),
    -- Invoice expiry, expressed in hours + minutes so the operator can set e.g.
    -- 1 hour 30 minutes. Resolved to minutes at read time.
    ('invoice_expiry_hours',   '24'::jsonb, 'Batas waktu pembayaran faktur (jam)'),
    ('invoice_expiry_minutes', '0'::jsonb,  'Batas waktu pembayaran faktur (menit tambahan)')
ON CONFLICT (key) DO UPDATE
    SET description = EXCLUDED.description
    WHERE settings.description = '';

-- ------------------------------------------------------------
-- 5. Sweep index: the expiry worker scans open invoices by due date.
-- ------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_invoices_open_due
    ON invoices (due_at)
    WHERE status IN ('issued', 'past_due');
