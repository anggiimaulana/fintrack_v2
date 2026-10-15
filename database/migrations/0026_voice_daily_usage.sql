-- ============================================================
-- 0026_voice_daily_usage.sql
-- Append-only daily voice counter.
--
-- The daily limit (voice_requests_per_day) must NOT be resettable by deleting
-- history: otherwise a user could clear their history to regain quota the same
-- day (design risk R-14). This table records one row per (user, day) and is
-- incremented on every transcription — it is never deleted by the user, so the
-- counter is authoritative for the day it belongs to.
--
-- Pure EXPAND migration: new table, no legacy consumers. Idempotent.
-- ============================================================

CREATE TABLE IF NOT EXISTS voice_daily_usage (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    usage_day  date NOT NULL,
    used       integer NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, usage_day)
);

-- Retention/pruning of old counters is by day alone.
CREATE INDEX IF NOT EXISTS idx_voice_daily_usage_day
    ON voice_daily_usage (usage_day);

-- down: DROP TABLE IF EXISTS voice_daily_usage;
