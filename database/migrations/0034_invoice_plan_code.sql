-- ============================================================
-- 0034_invoice_plan_code.sql
--
-- Give invoices a dedicated `plan_code` column instead of overloading `notes`.
--
-- Problem addressed:
--   The checkout flow stored the plan code in the invoice's free-text `notes`
--   column, and activation read it back from there (planCodeFromInvoice). That
--   made `notes` a load-bearing field: anything else that needed to record a
--   note (e.g. the renewal worker's per-cycle idempotency marker) would be
--   mistaken for a plan code, so a paid renewal invoice would try to activate a
--   plan named "renewal:<id>:<ts>" and fail — the user pays and gets nothing.
--
--   This adds `plan_code` as the single source of truth for "which plan does
--   this invoice buy", backfills it from the existing notes values that look
--   like a plan code, and frees `notes` to carry non-plan metadata.
--
-- Expand-only & idempotent: the column is nullable and additive, so an older
-- app version keeps working during rollout.
-- ============================================================

BEGIN;

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS plan_code varchar(64);

-- Backfill from notes, but only when the value actually names a plan. This
-- avoids copying an arbitrary note into the plan column.
UPDATE invoices i
SET plan_code = i.notes
FROM plans p
WHERE i.plan_code IS NULL
  AND i.notes IS NOT NULL
  AND i.notes = p.code;

CREATE INDEX IF NOT EXISTS idx_invoices_plan_code ON invoices (plan_code);

-- The renewal idempotency marker now lives in `notes` (prefixed `renewal:`),
-- which is safe because activation no longer reads plan_code from notes.
-- A partial index keeps the per-cycle dedupe lookup cheap.
CREATE INDEX IF NOT EXISTS idx_invoices_renewal_marker
    ON invoices (notes)
    WHERE notes LIKE 'renewal:%';

COMMIT;
