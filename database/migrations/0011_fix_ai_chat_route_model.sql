-- ============================================================
-- 0011_fix_ai_chat_route_model.sql
--
-- Point the `chat` route at a model the configured gateway actually serves.
--
-- Why: migration 0007 seeded the chat route with `gpt-4o-mini`. Measured against
-- the local 9router on 2026-10-11:
--
--   POST /v1/chat/completions {"model":"gpt-4o-mini"}    -> 503
--   POST /v1/chat/completions {"model":"deepseek-r1"}    -> 200
--
-- A 503 from the gateway means the upstream provider has no such model. Because
-- the centralized gateway path has NO fallback to BYOK, enabling the gateway with
-- a dead chat route would break AI for every user.
--
-- This migration only changes the model on the existing chat route; it does not
-- add or remove routes, and it is idempotent (re-running is a no-op once the model
-- already matches).
--
-- Rollback (documented, not automated):
--   UPDATE ai_routes SET model = 'gpt-4o-mini' WHERE use_case = 'chat';
--
-- NOTE: vision/voice/embedding still point at models that this gateway does not
-- serve (gpt-4o-mini / whisper-1 / text-embedding-004). Those paths are NOT yet
-- safe to route through the gateway; see the P07 D1 findings. Embedding failure
-- degrades gracefully (the transaction is still saved), but chat does not.
-- ============================================================

UPDATE ai_routes
SET model = 'deepseek-r1',
    updated_at = now()
WHERE use_case = 'chat'
  AND model <> 'deepseek-r1';
