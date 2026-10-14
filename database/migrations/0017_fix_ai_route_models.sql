-- ============================================================
-- 0017_fix_ai_route_models.sql
--
-- Point the chat/vision routes at models the configured gateway (9router)
-- actually serves, and give the reasoning route enough time to think.
--
-- Why (measured against the live 9router on 2026-10-14):
--
--   POST /v1/chat/completions {"model":"gemini-2.5-flash"}  -> 503 (0/5 runs)
--   POST /v1/chat/completions {"model":"gemini/gemini-3.5-flash-lite"} -> 200 (5/5, ~0.7-1.5s, tool-calls OK)
--   POST /v1/chat/completions {"model":"gemini/gemini-3.5-flash"}      -> 200 (reliable, tool-calls OK, vision OK)
--   POST /v1/chat/completions {"model":"deepseek-r1"}       -> 200 but 4s-103s (exceeds the 30s route timeout)
--
-- Consequences being fixed:
--   * chat CEPAT was routed to a dead model (gemini-2.5-flash) and its failover
--     was a slow reasoning model (deepseek-r1) — so "fast" was neither fast nor
--     reliable.
--   * vision (receipt scanning) was routed to the same dead model -> 503.
--   * reasoning (deepseek-r1) routinely exceeds timeout_ms=30000 while thinking,
--     so PENALARAN requests were being cut off.
--
-- This migration only rewrites the `model` column of the affected routes and
-- raises the reasoning route timeout. It does not add or delete routes, and it
-- is idempotent (re-running is a no-op once the models already match).
--
-- Rollback (documented, not automated):
--   UPDATE ai_routes SET model='gemini-2.5-flash' WHERE use_case='chat' AND model='gemini/gemini-3.5-flash-lite';
--   UPDATE ai_routes SET model='deepseek-r1'      WHERE use_case='chat' AND model='gemini/gemini-3.5-flash';
--   UPDATE ai_routes SET model='gemini-2.5-flash' WHERE use_case='vision';
--   UPDATE ai_routes SET timeout_ms=30000 WHERE use_case='reasoning';
-- ============================================================

-- Chat: the dead primary model -> the fast, reliable one.
UPDATE ai_routes
SET model = 'gemini/gemini-3.5-flash-lite',
    updated_at = now()
WHERE use_case = 'chat'
  AND model = 'gemini-2.5-flash';

-- Chat: the slow reasoning model used as failover -> a reliable general model.
UPDATE ai_routes
SET model = 'gemini/gemini-3.5-flash',
    updated_at = now()
WHERE use_case = 'chat'
  AND model = 'deepseek-r1';

-- Vision (receipt scanning): dead model -> a model that actually reads images.
UPDATE ai_routes
SET model = 'gemini/gemini-3.5-flash',
    updated_at = now()
WHERE use_case = 'vision'
  AND model = 'gemini-2.5-flash';

-- Reasoning: keep the dedicated reasoning model (deepseek-r1) but allow enough
-- time for its chain-of-thought, which routinely runs past the 30s default.
UPDATE ai_routes
SET timeout_ms = 120000,
    updated_at = now()
WHERE use_case = 'reasoning'
  AND timeout_ms < 120000;
