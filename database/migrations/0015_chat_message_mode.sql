-- ============================================================
-- 0015_chat_message_mode.sql
--
-- EXPAND: record which assistant mode ("fast" | "reasoning") produced each
-- assistant message, so the UI can label replies ([Cepat] / [Penalaran]) and
-- usage can be audited per mode.
--
-- Nullable on purpose: pre-existing rows have no recorded mode, and the old
-- application version does not write this column. No NOT NULL, no default that
-- would force a rewrite.
--
-- Additive & idempotent.
-- ============================================================

ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS mode text;

-- down: ALTER TABLE chat_messages DROP COLUMN IF EXISTS mode;
