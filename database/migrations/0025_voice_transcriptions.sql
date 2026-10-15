-- ============================================================
-- 0025_voice_transcriptions.sql
-- Voice input history + daily-limit accounting.
--
-- Records every transcription (server via /ai/voice, browser via
-- POST /me/voice-history) so:
--   1. the user has a durable, cross-device history (Q-04),
--   2. the daily limit `voice_requests_per_day` can be counted per day (Q-03),
--   3. the retention worker can prune rows older than the configured window (N-01).
--
-- Pure EXPAND migration: a brand-new table with no legacy consumers, so it is
-- safe under a rolling deploy. Idempotent (IF NOT EXISTS).
-- ============================================================

CREATE TABLE IF NOT EXISTS voice_transcriptions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- The ORIGINAL transcript (before any user edit). Edits live in chat history.
    transcript  text NOT NULL,
    language    varchar(16) NOT NULL DEFAULT 'id',
    mode        varchar(16) NOT NULL CHECK (mode IN ('server', 'browser')),
    duration_ms integer NOT NULL DEFAULT 0,
    tokens_used bigint  NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- Serves both the paged history (newest first) and the daily count
-- (`WHERE user_id = $1 AND created_at >= <day start>`).
CREATE INDEX IF NOT EXISTS idx_voice_tx_user_created
    ON voice_transcriptions (user_id, created_at DESC);

-- Retention sweeps filter by age alone across all users; a bare created_at
-- index keeps the delete cheap.
CREATE INDEX IF NOT EXISTS idx_voice_tx_created
    ON voice_transcriptions (created_at);

-- down: DROP TABLE IF EXISTS voice_transcriptions;
