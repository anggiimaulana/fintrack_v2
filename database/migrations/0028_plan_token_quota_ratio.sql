-- ============================================================
-- 0028_plan_token_quota_ratio.sql
--
-- Rasio konversi token provider → unit kuota plan.
--
-- Latar: kuota plan (plan_limits.ai_tokens_per_cycle) diukur dalam "unit",
-- sedangkan provider melaporkan token mentah. Satu percakapan nyata memakai
-- ~5.000–10.000 token, sehingga kuota Free 5.000 habis dalam satu chat.
--
-- Rasio memisahkan dua besaran:
--   * raw_tokens  → dasar BIAYA (Rupiah) & margin
--   * unit kuota  → raw_tokens / rasio, dibandingkan dengan kuota plan
--
-- Contoh (rasio Free = 5): chat 9.945 token menagih ceil(9945/5) = 1.989 unit,
-- sehingga kuota 5.000 masih menyisakan 3.011 — user bisa ~2–3 chat.
--
-- Default 1 = perilaku lama (1 token = 1 unit), jadi deploy ini TIDAK
-- mengubah akuntansi yang sedang berjalan.
--
-- Additive & idempotent (expand only; tidak ada DROP/RENAME/SET NOT NULL).
--
-- down:
--   ALTER TABLE token_ledger DROP CONSTRAINT IF EXISTS token_ledger_raw_tokens_check;
--   ALTER TABLE token_ledger DROP COLUMN IF EXISTS raw_tokens;
--   ALTER TABLE token_periods DROP CONSTRAINT IF EXISTS token_periods_quota_ratio_check;
--   ALTER TABLE token_periods DROP COLUMN IF EXISTS quota_ratio;
--   ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_token_quota_ratio_check;
--   ALTER TABLE plans DROP COLUMN IF EXISTS token_quota_ratio;
-- ============================================================

-- 1) Rasio per plan. Dipakai saat siklus kuota baru dibuat.
ALTER TABLE plans
    ADD COLUMN IF NOT EXISTS token_quota_ratio bigint NOT NULL DEFAULT 1;

-- Rasio harus >= 1 (1 = 1:1). Nilai < 1 berarti menagih LEBIH banyak unit
-- daripada token nyata, yang tidak masuk akal dan berisiko menyesatkan user.
ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_token_quota_ratio_check;
ALTER TABLE plans
    ADD CONSTRAINT plans_token_quota_ratio_check
    CHECK (token_quota_ratio >= 1);

-- 2) Snapshot rasio pada siklus kuota. Kolom ini IMMUTABLE: begitu siklus
--    berjalan, syaratnya (kuota + rasio) beku sampai siklus berakhir, sehingga
--    mengubah rasio plan tidak mengubah akuntansi siklus yang sedang berjalan.
ALTER TABLE token_periods
    ADD COLUMN IF NOT EXISTS quota_ratio bigint NOT NULL DEFAULT 1;

ALTER TABLE token_periods DROP CONSTRAINT IF EXISTS token_periods_quota_ratio_check;
ALTER TABLE token_periods
    ADD CONSTRAINT token_periods_quota_ratio_check
    CHECK (quota_ratio >= 1);

-- 3) Token mentah pada ledger. `amount` kini menyimpan UNIT kuota; `raw_tokens`
--    menyimpan token provider yang menjadi dasar biaya. Baris lama TIDAK
--    di-backfill (append-only) — dibaca via helper RawTokensOrAmount().
ALTER TABLE token_ledger
    ADD COLUMN IF NOT EXISTS raw_tokens bigint NOT NULL DEFAULT 0;

ALTER TABLE token_ledger DROP CONSTRAINT IF EXISTS token_ledger_raw_tokens_check;
ALTER TABLE token_ledger
    ADD CONSTRAINT token_ledger_raw_tokens_check
    CHECK (raw_tokens >= 0);
