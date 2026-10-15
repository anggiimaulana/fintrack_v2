-- ============================================================
-- 0030_ai_routes_normalize.sql
--
-- Repair and PIN the AI routing table so a full `migrate.sh` replay is
-- idempotent — on a FRESH install and on an existing one alike.
--
-- Root cause (a pre-existing bug in the committed data-migrations 0011/0012/0017,
-- which migrate.sh re-applies on EVERY run because the whole chain replays):
--
--   0011  UPDATE ... SET model='deepseek-r1'
--           WHERE use_case='chat' AND model <> 'deepseek-r1'
--         -> rewrites ALL chat routes to deepseek-r1.
--   0012  UPDATE ... SET model='gemini-embedding-001' WHERE use_case='embedding'
--         -> drops the 'gemini/' prefix the gateway needs.
--   0012  INSERT 2 chat failover routes guarded by
--           NOT EXISTS (... model='gemini-2.5-flash')
--         -> the guard can never match (0017 already renamed those models), so
--            two duplicate chat routes are appended on EVERY run.
--   0017  UPDATE ... SET model='gemini/gemini-3.5-flash-lite'
--           WHERE model='gemini-2.5-flash'
--         UPDATE ... SET model='gemini/gemini-3.5-flash' WHERE model='deepseek-r1'
--         -> renames the freshly-inserted rows and the originals, so the fast
--            'flash-lite' chat route is lost and the table drifts.
--
-- This migration is deliberately ID-AGNOSTIC: it never references a route id or
-- an endpoint id, because both are generated (gen_random_uuid) and therefore
-- differ between a fresh install and an existing one. Earlier revisions of this
-- file hardcoded those ids and failed on a fresh database with a foreign-key
-- violation; this version resolves the endpoint by NAME (as 0007/0012 do) and
-- matches routes by their logical identity.
--
-- It converges BOTH starting states to the same canonical 8-route topology:
--
--   1. Normalize the single-route modalities (model mangled by the replay).
--   2. Normalize the unambiguous chat slots (default/1 = fast, default/2 =
--      failover). Only gateway-managed models are touched, so an operator's
--      custom model is never overwritten.
--   3. Ensure the four canonical chat routes exist (a fresh install is missing
--      the second backup route).
--   4. Delete exact duplicates — the rows 0012 appends every replay.
--
-- Idempotent: re-running changes nothing once the table is canonical.
--
-- Rollback: not applicable (this only restores the documented intended state).
-- ============================================================

BEGIN;

-- The gateway-managed model names, used to avoid clobbering a custom route.
-- (Kept inline rather than in a temp table so the statement is self-contained.)

-- ── 1. Single-route modalities: restore the intended model. ──────────────────
UPDATE ai_routes SET model = 'gemini/gemini-3.5-flash',     updated_at = now() WHERE use_case = 'vision';
UPDATE ai_routes SET model = 'gemini/gemini-embedding-001', updated_at = now() WHERE use_case = 'embedding';
UPDATE ai_routes SET model = 'whisper-1',                   updated_at = now() WHERE use_case = 'voice';
UPDATE ai_routes SET model = 'deepseek-r1',                 updated_at = now() WHERE use_case = 'reasoning';

-- ── 2. Chat slots that are unambiguous by (combo, priority). ─────────────────
-- default/1 is the fast model, default/2 the failover. The guard on the managed
-- model set keeps an operator-authored custom route intact.
UPDATE ai_routes
   SET model = 'gemini/gemini-3.5-flash-lite', updated_at = now()
 WHERE use_case = 'chat' AND combo = 'default' AND priority = 1
   AND model IN ('gemini/gemini-3.5-flash-lite', 'gemini/gemini-3.5-flash', 'gemini-2.5-flash', 'deepseek-r1', 'gpt-4o-mini');

UPDATE ai_routes
   SET model = 'gemini/gemini-3.5-flash', updated_at = now()
 WHERE use_case = 'chat' AND combo = 'default' AND priority = 2
   AND model IN ('gemini/gemini-3.5-flash-lite', 'gemini/gemini-3.5-flash', 'gemini-2.5-flash', 'deepseek-r1', 'gpt-4o-mini');

-- ── 3. Ensure the four canonical chat routes exist. ─────────────────────────
-- A fresh install replays 0007+0011+0012+0017 and ends up missing the second
-- backup route, so the insert is required for the two starting states to agree.
INSERT INTO ai_routes (use_case, endpoint_id, model, combo, priority, is_active, timeout_ms, max_retries)
SELECT v.use_case, e.id, v.model, v.combo, v.priority, true, v.timeout_ms, 2
FROM (VALUES
    ('chat', 'gemini/gemini-3.5-flash-lite', 'default', 1, 60000),
    ('chat', 'gemini/gemini-3.5-flash',      'default', 2, 60000),
    ('chat', 'gemini/gemini-3.5-flash-lite', 'backup',  1, 60000),
    ('chat', 'gemini/gemini-3.5-flash',      'backup',  1, 60000)
) AS v(use_case, model, combo, priority, timeout_ms)
CROSS JOIN ai_endpoints e
WHERE e.name = '9router-primary'
  AND NOT EXISTS (
      SELECT 1 FROM ai_routes r
      WHERE r.use_case = v.use_case AND r.combo = v.combo AND r.model = v.model
  );

-- ── 4. Delete exact duplicates (keep the oldest row per identity). ──────────
-- This is the set migration 0012 appends on every replay. Matching on the full
-- identity (incl. model) means a genuinely distinct route is never removed.
DELETE FROM ai_routes a
USING ai_routes b
WHERE a.id <> b.id
  AND a.use_case    = b.use_case
  AND a.endpoint_id = b.endpoint_id
  AND a.model       = b.model
  AND a.combo       = b.combo
  AND a.priority    = b.priority
  AND (a.created_at, a.id) > (b.created_at, b.id);

-- ── 5. Register the AI timeout keys in the dynamic config catalog (0029) ────
-- so they appear in the Super Admin UI and are validated from DATA. Additive:
-- the Go allowlist still accepts them. Guarded so it is a no-op when the catalog
-- table is absent (a database restored from an older dump).
DO $$
BEGIN
  IF to_regclass('public.setting_definitions') IS NOT NULL THEN
    INSERT INTO setting_definitions
      (key, kind, min_value, max_value, default_value, group_key, label, description, sort_order) VALUES
      ('ai_timeout_chat_ms',      'int', 0, 1800000, '60000'::jsonb,  'ai_timeout', 'Batas waktu: Chat (ms)',        'Batas waktu panggilan AI untuk chat streaming. 0 = pakai nilai rute/default gateway.', 10),
      ('ai_timeout_vision_ms',    'int', 0, 1800000, '120000'::jsonb, 'ai_timeout', 'Batas waktu: Vision/OCR (ms)',  'Batas waktu panggilan AI untuk scan struk (OCR). 0 = pakai nilai rute/default gateway.', 20),
      ('ai_timeout_voice_ms',     'int', 0, 1800000, '120000'::jsonb, 'ai_timeout', 'Batas waktu: Suara (ms)',       'Batas waktu panggilan AI untuk transkripsi suara. 0 = pakai nilai rute/default gateway.', 30),
      ('ai_timeout_embedding_ms', 'int', 0, 1800000, '60000'::jsonb,  'ai_timeout', 'Batas waktu: Embedding (ms)',   'Batas waktu panggilan AI untuk embedding (RAG). 0 = pakai nilai rute/default gateway.', 40),
      ('ai_timeout_reasoning_ms', 'int', 0, 1800000, '120000'::jsonb, 'ai_timeout', 'Batas waktu: Penalaran (ms)',   'Batas waktu panggilan AI untuk mode penalaran. 0 = pakai nilai rute/default gateway.', 50),
      ('ai_timeout_client_ms',    'int', 0, 1800000, '0'::jsonb,      'ai_timeout', 'Batas waktu klien/browser (ms)','Batas waktu yang diberikan ke browser. 0 = otomatis (use case terlama + margin).', 60)
    ON CONFLICT (key) DO NOTHING;
  END IF;
END $$;

COMMIT;
