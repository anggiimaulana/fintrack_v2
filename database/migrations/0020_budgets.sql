-- ============================================================
-- 0020_budgets.sql
-- Per-user monthly budgets, backing the `advanced_budgeting` plan feature.
--
-- A budget is a per-category monthly ceiling the user sets; the reports page
-- compares it against actual spend for the selected period. There is no
-- "overall" row: an overall budget is the sum of the category budgets, which
-- keeps one source of truth and avoids a second code path.
--
-- Additive & idempotent. Money is BIGINT (integer Rupiah), matching K-27.
-- ============================================================

CREATE TABLE IF NOT EXISTS budgets (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- The expense category this ceiling applies to. Restricted to expense
    -- categories: budgeting income or savings is not meaningful.
    category_id uuid NOT NULL REFERENCES expense_categories(id) ON DELETE CASCADE,
    amount_idr  bigint NOT NULL CHECK (amount_idr >= 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    -- One budget per category per user.
    UNIQUE (user_id, category_id)
);

-- Listing a user's budgets is the only hot read.
CREATE INDEX IF NOT EXISTS idx_budgets_user ON budgets (user_id);
