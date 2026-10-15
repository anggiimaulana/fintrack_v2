-- ============================================================
-- 0024_ai_route_timeouts.sql
--
-- Give the slow AI modalities (vision/OCR, voice, and chat) enough time to
-- answer, so receipt scanning stops failing with a timeout.
--
-- Why (measured against the live 9router on 2026-10-15):
--
--   Vision (receipt OCR) and voice transcription regularly take longer than
--   30s end-to-end: the image/audio has to be uploaded, the model has to read
--   it, and the JSON answer has to come back. When the route's timeout_ms is
--   30000 the gateway aborts the upstream call mid-answer and the user sees
--   "Waktu analisis habis" even though the provider was healthy.
--
-- The chat routes keep a tighter budget (they stream token-by-token, so a slow
-- first token is the real risk), while vision/voice get a generous ceiling.
-- The values are intentionally identical to the platform defaults so a fresh
-- install and an upgraded install behave the same:
--
--   chat   : 60000  ms  (1 min)  — first token + streaming
--   vision : 120000 ms  (2 min)  — receipt OCR
--   voice  : 120000 ms  (2 min)  — audio transcription
--
-- Only raises timeouts (never lowers a route the operator tuned higher) and is
-- idempotent: re-running is a no-op once the values already match.
--
-- Rollback (documented, not automated):
--   UPDATE ai_routes SET timeout_ms = 30000
--   WHERE use_case IN ('chat','vision','voice');
-- ============================================================

UPDATE ai_routes
SET timeout_ms = 60000,
    updated_at = now()
WHERE use_case = 'chat'
  AND timeout_ms < 60000;

UPDATE ai_routes
SET timeout_ms = 120000,
    updated_at = now()
WHERE use_case IN ('vision', 'voice')
  AND timeout_ms < 120000;
