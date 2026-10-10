-- ============================================================
-- 0001_extensions.sql
-- Enable required PostgreSQL extensions for FinTrack v2.
-- - vector: pgvector for RAG embeddings (768-dim, HNSW)
--
-- NOTE: uuid-ossp is NOT needed (we use gen_random_uuid(), built-in
-- since PG13). pgcrypto is NOT used by the app (password hashing is
-- done in Go with bcrypt; demo hashes are written as literals in seeds).
-- ============================================================

CREATE EXTENSION IF NOT EXISTS vector;
