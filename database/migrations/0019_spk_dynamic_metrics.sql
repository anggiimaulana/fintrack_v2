-- ============================================================
-- 0019_spk_dynamic_metrics.sql
-- SPK (Sistem Pendukung Keputusan) dynamic metrics.
--
-- Until now the SPK scoring model was hard-coded: five indicators with fixed
-- weights, a fixed "needs" category list, and fixed thresholds. This migration
-- makes it data-driven:
--
--   * `spk_metric_defaults` — the super-admin-owned default model (one active
--     row). Every user inherits it unless they opt into a personal override.
--   * `spk_metric_overrides` — a per-user model that shadows the default. When
--     a user customises their metrics, only their own rows change.
--
-- Resolution order at scoring time: user override → default → compiled fallback
-- (so the engine still works on a database that has not been seeded yet).
--
-- Additive & idempotent. JSONB stores the indicator list; weights are validated
-- in the application (they must sum to 1.0), because a CHECK across a JSONB
-- array would be opaque and hard to evolve.
-- ============================================================

-- ------------------------------------------------------------
-- spk_metric_defaults — the super-admin default SPK model.
-- A single active row (`is_active = true`) is authoritative; older rows are
-- kept for history/rollback, mirroring the pricing-rules pattern (F5).
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS spk_metric_defaults (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL DEFAULT 'Model SPK Default',
    description text NOT NULL DEFAULT '',
    -- indicators: [{key,label,weight,target,type,categories?,direction}]
    --   key        stable identifier used by the scoring engine
    --   label      human-readable name shown in the UI
    --   weight     0..1, all weights must sum to 1.0
    --   target     the "ideal" value that scores 100 (e.g. 0.20 saving ratio)
    --   type       how the raw value is derived (saving_ratio|needs_ratio|
    --              consistency|expense_trend|diversification)
    --   categories optional list (used by needs_ratio)
    --   direction  "higher_better" (default) | "lower_better"
    indicators  jsonb NOT NULL DEFAULT '[]'::jsonb,
    -- status_thresholds: [{min,label}] evaluated top-down, e.g.
    --   [{"min":80,"label":"Sangat Baik"}, ...]
    status_thresholds jsonb NOT NULL DEFAULT '[]'::jsonb,
    is_active   boolean NOT NULL DEFAULT true,
    updated_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- At most one active default. A partial unique index enforces it without
-- blocking the historical (inactive) rows.
CREATE UNIQUE INDEX IF NOT EXISTS spk_metric_defaults_one_active
    ON spk_metric_defaults (is_active)
    WHERE is_active;

-- ------------------------------------------------------------
-- spk_metric_overrides — a user's personal SPK model.
-- One row per user (upsert). Deleting the row reverts the user to the default.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS spk_metric_overrides (
    user_id     uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    indicators  jsonb NOT NULL DEFAULT '[]'::jsonb,
    status_thresholds jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- Seed the default model with the exact values the engine used to hard-code,
-- so behaviour is unchanged until an operator edits it. Idempotent: only
-- inserted when no active default exists.
-- ------------------------------------------------------------
INSERT INTO spk_metric_defaults (name, description, indicators, status_thresholds, is_active)
SELECT
    'Model SPK Default',
    'Model penilaian kesehatan finansial bawaan (metode 50/30/20, rasio tabungan, tren arus kas).',
    '[
      {"key":"saving_ratio","label":"Rasio Tabungan","weight":0.30,"target":0.20,"type":"saving_ratio","direction":"higher_better"},
      {"key":"needs_vs_wants","label":"Kebutuhan vs Keinginan","weight":0.25,"target":0.70,"type":"needs_ratio","direction":"higher_better","categories":["Makanan & Minuman","Kesehatan","Utilitas & Tagihan","Transportasi","Pendidikan","Tempat Tinggal"]},
      {"key":"consistency","label":"Konsistensi Catatan","weight":0.15,"target":1.0,"type":"consistency","direction":"higher_better"},
      {"key":"expense_trend","label":"Tren Pengeluaran","weight":0.20,"target":1.0,"type":"expense_trend","direction":"higher_better"},
      {"key":"diversification","label":"Diversifikasi","weight":0.10,"target":1.0,"type":"diversification","direction":"higher_better"}
    ]'::jsonb,
    '[
      {"min":80,"label":"Sangat Baik"},
      {"min":60,"label":"Baik"},
      {"min":40,"label":"Cukup"},
      {"min":20,"label":"Boros"},
      {"min":0,"label":"Kritis"}
    ]'::jsonb,
    true
WHERE NOT EXISTS (SELECT 1 FROM spk_metric_defaults WHERE is_active);
