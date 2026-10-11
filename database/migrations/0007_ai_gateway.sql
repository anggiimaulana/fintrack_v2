-- ============================================================
-- 0007_ai_gateway.sql
-- Centralized AI gateway: multi-endpoint pool + per-use-case routing.
-- Replaces the per-user BYOK model (gemini_api_keys) with a
-- super-admin-managed pool (9router primary).
--
-- Guarantees:
--   * Additive & idempotent (CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS).
--   * gemini_api_keys is NOT dropped: legacy keys are archived for audit.
--
-- Idempotent: safe to re-run.
-- ============================================================

-- ------------------------------------------------------------
-- ai_endpoints — pooled upstream endpoints (9router, direct providers)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ai_endpoints (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name              text NOT NULL UNIQUE,
    base_url          text NOT NULL,
    -- Encrypted API key (hex string produced by internal/infrastructure/crypto).
    -- NULL for keyless endpoints (e.g. a local 9router proxy).
    api_key_encrypted text,
    provider          text NOT NULL DEFAULT '9router',
    priority          integer NOT NULL DEFAULT 1,
    -- status is the admin kill switch: 'inactive' removes the endpoint from the pool.
    status            text NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active', 'degraded', 'inactive', 'suspended')),
    health            text NOT NULL DEFAULT 'unknown'
                      CHECK (health IN ('healthy', 'unhealthy', 'unknown')),
    last_check_at     timestamptz,
    last_error        text,
    modalities        text[] NOT NULL DEFAULT '{chat,vision,voice,embedding,reasoning}',
    weight            integer NOT NULL DEFAULT 100,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_ai_endpoint_updated
    BEFORE UPDATE ON ai_endpoints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_ai_endpoints_status_priority
    ON ai_endpoints (status, priority);

-- ------------------------------------------------------------
-- ai_routes — per-use-case mapping to an endpoint + model
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ai_routes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    use_case    text NOT NULL
                CHECK (use_case IN ('chat', 'vision', 'voice', 'embedding', 'reasoning')),
    endpoint_id uuid NOT NULL REFERENCES ai_endpoints(id) ON DELETE CASCADE,
    model       text NOT NULL DEFAULT '',
    combo       text NOT NULL DEFAULT 'default',
    priority    integer NOT NULL DEFAULT 1,
    is_active   boolean NOT NULL DEFAULT true,
    timeout_ms  integer NOT NULL DEFAULT 30000,
    max_retries integer NOT NULL DEFAULT 2,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_ai_route_updated
    BEFORE UPDATE ON ai_routes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_ai_routes_lookup
    ON ai_routes (use_case, is_active, priority);

-- ------------------------------------------------------------
-- Archive legacy BYOK keys (kept for audit; never hard-deleted).
-- ------------------------------------------------------------
ALTER TABLE gemini_api_keys ADD COLUMN IF NOT EXISTS is_archived boolean NOT NULL DEFAULT true;
ALTER TABLE gemini_api_keys ADD COLUMN IF NOT EXISTS archived_at timestamptz DEFAULT now();

-- ------------------------------------------------------------
-- Seed: default 9router endpoint + default routes per use case.
-- Keyless endpoint (local proxy). Idempotent via ON CONFLICT (name).
-- ------------------------------------------------------------
INSERT INTO ai_endpoints (name, base_url, provider, priority, status, health, modalities)
VALUES ('9router-primary', 'http://localhost:20128/v1', '9router', 1, 'active', 'unknown',
        '{chat,vision,voice,embedding,reasoning}')
ON CONFLICT (name) DO NOTHING;

-- Default routes (idempotent: only insert when the use_case has no route yet).
INSERT INTO ai_routes (use_case, endpoint_id, model, combo, priority)
SELECT v.use_case, e.id, v.model, v.combo, 1
FROM (VALUES
    ('chat',      'gpt-4o-mini',        'default'),
    ('vision',    'gpt-4o-mini',        'default'),
    ('voice',     'whisper-1',          'default'),
    ('embedding', 'text-embedding-004', 'default'),
    ('reasoning', 'deepseek-r1',        'default')
) AS v(use_case, model, combo)
CROSS JOIN ai_endpoints e
WHERE e.name = '9router-primary'
  AND NOT EXISTS (
      SELECT 1 FROM ai_routes r WHERE r.use_case = v.use_case
  );
