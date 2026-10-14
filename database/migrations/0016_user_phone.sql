-- ============================================================
-- 0016_user_phone.sql
--
-- EXPAND: add a phone number to the user profile.
--
-- The profile UI has always exposed a "Nomor Telepon" field, but there was no
-- column to store it, so the value was silently dropped and never persisted.
--
-- Nullable on purpose: existing rows have no phone, and the previous
-- application version does not write this column.
--
-- Additive & idempotent.
-- ============================================================

ALTER TABLE users ADD COLUMN IF NOT EXISTS phone text;

-- down: ALTER TABLE users DROP COLUMN IF EXISTS phone;
