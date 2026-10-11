-- ============================================================
-- 0006_create_feature_flags.sql
-- Runtime feature flags with percentage rollout.
-- Evaluated per-request (Redis-cached) so flags can be toggled
-- without redeploy. Default: disabled (fail-closed).
--
-- Idempotent: safe to re-run.
-- ============================================================

CREATE TABLE IF NOT EXISTS feature_flags (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    key         text NOT NULL UNIQUE,
    name        text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    is_enabled  boolean NOT NULL DEFAULT false,
    -- Rollout percentage 0..100 (0 = nobody, 100 = everybody when enabled).
    rollout     integer NOT NULL DEFAULT 0 CHECK (rollout >= 0 AND rollout <= 100),
    -- Optional targeting rules (reserved for later phases).
    rules       jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_feature_flag_updated
    BEFORE UPDATE ON feature_flags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_feature_flags_enabled ON feature_flags (is_enabled);
