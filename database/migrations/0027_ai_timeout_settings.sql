-- ============================================================
-- 0027_ai_timeout_settings.sql
--
-- Seed the dynamic AI timeout budget into `settings` so a Super Admin can tune
-- it from the UI and have it applied to the running fleet without a restart.
--
-- Why a settings row and not just ai_routes.timeout_ms:
--
--   * `ai_routes.timeout_ms` is per route (endpoint+model) and only affects the
--     upstream call. The client budget and the HTTP write deadline live outside
--     the gateway, so they need their own source of truth.
--   * The `settings` table is the established dynamic-knob mechanism (breaker,
--     metering, voice): DB value overrides the env default, and a change is
--     fanned out to every instance over Redis Pub/Sub within milliseconds.
--
-- Precedence (handled in code, documented here for operators):
--   settings row  >  env var (AI_TIMEOUT_*_MS)  >  per-route timeout_ms  >  gateway default
--
-- The values below match the platform defaults, so a fresh install and an
-- upgraded install behave identically. Vision/voice get the largest budget
-- because OCR/transcription routinely run past 30s.
--
-- Idempotent: ON CONFLICT refreshes only the description of a row the operator
-- left blank; a value the operator has set is never overwritten.
--
-- Rollback (documented, not automated):
--   DELETE FROM settings WHERE key LIKE 'ai_timeout_%';
-- ============================================================

-- The `value` column is jsonb, so integers are stored as JSON numbers (not
-- quoted strings) — this is what GetInt reads back.
--
-- `ai_timeout_client_ms` is deliberately NOT seeded: when absent the backend
-- derives it automatically from the slowest use case plus a margin, so raising
-- (say) vision also raises the client budget. Seeding it would freeze that
-- relationship. An operator can still set it explicitly to override.
INSERT INTO settings (key, value, description) VALUES
    ('ai_timeout_chat_ms',      '60000'::jsonb,  'Batas waktu panggilan AI (ms) untuk Chat (streaming)'),
    ('ai_timeout_vision_ms',    '120000'::jsonb, 'Batas waktu panggilan AI (ms) untuk Vision/OCR (scan struk)'),
    ('ai_timeout_voice_ms',     '120000'::jsonb, 'Batas waktu panggilan AI (ms) untuk transkripsi suara'),
    ('ai_timeout_embedding_ms', '60000'::jsonb,  'Batas waktu panggilan AI (ms) untuk Embedding (RAG)'),
    ('ai_timeout_reasoning_ms', '120000'::jsonb, 'Batas waktu panggilan AI (ms) untuk mode Penalaran')
ON CONFLICT (key) DO UPDATE
    SET description = EXCLUDED.description
    WHERE settings.description = '';
