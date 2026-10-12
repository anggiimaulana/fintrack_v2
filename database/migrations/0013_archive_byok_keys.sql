-- ============================================================
-- 0013_archive_byok_keys.sql
--
-- TASK-P07-013: Pembersihan Tuntas Menu & Kode Bring Your Own Key (BYOK)
-- Archive any remaining user-level Gemini API keys and set status to inactive.
-- All AI workloads are now routed through centralized AI Gateway (P03).
--
-- Additive & idempotent.
-- ============================================================

UPDATE gemini_api_keys
SET is_archived = true,
    archived_at = COALESCE(archived_at, now()),
    status = 'inactive',
    updated_at = now()
WHERE is_archived = false OR status = 'active';

CREATE INDEX IF NOT EXISTS idx_gemini_keys_archived ON gemini_api_keys (is_archived);
