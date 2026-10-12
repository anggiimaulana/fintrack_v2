-- ============================================================
-- 0014_drop_gemini_api_keys.sql
--
-- Contract Phase: Pembersihan permanen tabel gemini_api_keys
-- Seluruh trafik AI telah beralih 100% ke AI Gateway (9router) via ai_endpoints & ai_routes.
-- Model BYOK (Bring Your Own Key) telah resmi dihentikan sepenuhnya.
--
-- Additive & idempotent.
-- ============================================================

DROP TABLE IF EXISTS gemini_api_keys CASCADE;
