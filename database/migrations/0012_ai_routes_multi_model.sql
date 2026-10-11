-- ============================================================
-- 0012_ai_routes_multi_model.sql
--
-- Multi-model dynamic AI routing with combos and failover.
--
-- 1. Updates vision and embedding routes to models proven to respond (200 OK):
--    - vision: gpt-4o-mini (503) -> gemini-2.5-flash (200)
--    - embedding: text-embedding-004 (404) -> gemini-embedding-001 (200 with dimensions=768)
--
-- 2. Inserts secondary failover routes for chat:
--    - (chat, default, gemini-2.5-flash, priority 2): in-combo priority failover if deepseek-r1 fails
--    - (chat, backup, gemini-2.5-flash, priority 1): separate combo for weighted round-robin load distribution
--
-- Additive & idempotent: only updates if model differs; only inserts if route does not already exist.
-- Does NOT delete existing routes, preserving any custom admin routes.
-- ============================================================

-- Fix vision route model to gemini-2.5-flash (proven 200 OK)
UPDATE ai_routes
SET model = 'gemini-2.5-flash',
    combo = COALESCE(NULLIF(combo, ''), 'default'),
    updated_at = now()
WHERE use_case = 'vision'
  AND model <> 'gemini-2.5-flash';

-- Fix embedding route model to gemini-embedding-001 (proven 200 OK)
UPDATE ai_routes
SET model = 'gemini-embedding-001',
    combo = COALESCE(NULLIF(combo, ''), 'default'),
    updated_at = now()
WHERE use_case = 'embedding'
  AND model <> 'gemini-embedding-001';

-- Ensure existing chat routes have a normalized combo
UPDATE ai_routes
SET combo = 'default',
    updated_at = now()
WHERE use_case = 'chat'
  AND (combo IS NULL OR combo = '');

-- Add secondary in-combo failover route for chat: (chat, default, gemini-2.5-flash, priority 2)
INSERT INTO ai_routes (use_case, endpoint_id, model, combo, priority, is_active, timeout_ms, max_retries)
SELECT 'chat', e.id, 'gemini-2.5-flash', 'default', 2, true, 30000, 2
FROM ai_endpoints e
WHERE e.name = '9router-primary'
  AND NOT EXISTS (
      SELECT 1 FROM ai_routes r
      WHERE r.use_case = 'chat'
        AND r.combo = 'default'
        AND r.model = 'gemini-2.5-flash'
  );

-- Add secondary combo route for chat WRR: (chat, backup, gemini-2.5-flash, priority 1)
INSERT INTO ai_routes (use_case, endpoint_id, model, combo, priority, is_active, timeout_ms, max_retries)
SELECT 'chat', e.id, 'gemini-2.5-flash', 'backup', 1, true, 30000, 2
FROM ai_endpoints e
WHERE e.name = '9router-primary'
  AND NOT EXISTS (
      SELECT 1 FROM ai_routes r
      WHERE r.use_case = 'chat'
        AND r.combo = 'backup'
        AND r.model = 'gemini-2.5-flash'
  );
