-- ============================================================
-- 0002_tables.sql
-- Final FinTrack v2 schema. No RLS, no Supabase `auth.*` schema.
-- All FKs point to public.users. Authorization is enforced in Go;
-- the DB role fintrack_app is least-privilege (no DDL).
--
-- Tables (13):
--   users, refresh_tokens,
--   income_categories, expense_categories, savings_categories,
--   expense_subcategories, transactions, transaction_embeddings,
--   gemini_api_keys, spk_results,
--   chat_sessions, chat_messages, ai_providers
--
-- NOTE: `ai_conversations` is intentionally NOT created (dead code
-- in v1). `uuid-ossp`/`pgcrypto` are not used: we rely on
-- gen_random_uuid() and Go-side bcrypt.
-- ============================================================

-- ------------------------------------------------------------
-- Helper: auto-update updated_at
-- ------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

-- ------------------------------------------------------------
-- users
-- Replaces Supabase auth.users + public.profiles.
-- password_hash NULL => account must reset password (migrated
-- users without an encrypted_password).
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL UNIQUE,
    password_hash text,
    full_name     text NOT NULL DEFAULT '',
    role          text NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    avatar_url    text,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_user_updated
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- refresh_tokens
-- Rotation-based refresh tokens. Only sha256 hex of the raw token
-- is stored. `replaced_by` links a rotated token to its successor
-- for reuse detection.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   text NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    replaced_by  uuid REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    user_agent   text
);

-- ------------------------------------------------------------
-- income_categories
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS income_categories (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL UNIQUE,
    icon        text NOT NULL DEFAULT 'circle',
    color       text NOT NULL DEFAULT '#3b5bdb',
    description text,
    is_active   boolean NOT NULL DEFAULT true,
    sort_order  integer NOT NULL DEFAULT 0,
    created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_income_category_updated
    BEFORE UPDATE ON income_categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- expense_categories
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS expense_categories (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL UNIQUE,
    icon        text NOT NULL DEFAULT 'circle',
    color       text NOT NULL DEFAULT '#3b5bdb',
    description text,
    is_active   boolean NOT NULL DEFAULT true,
    sort_order  integer NOT NULL DEFAULT 0,
    created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_expense_category_updated
    BEFORE UPDATE ON expense_categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- savings_categories
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS savings_categories (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    icon        text NOT NULL DEFAULT 'piggy-bank',
    color       text NOT NULL DEFAULT '#f59e0b',
    description text,
    is_active   boolean NOT NULL DEFAULT true,
    sort_order  integer NOT NULL DEFAULT 0,
    created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_savings_category_updated
    BEFORE UPDATE ON savings_categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- expense_subcategories
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS expense_subcategories (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id uuid NOT NULL REFERENCES expense_categories(id) ON DELETE CASCADE,
    name        text NOT NULL,
    icon        text NOT NULL DEFAULT 'circle',
    color       text NOT NULL DEFAULT '#3b5bdb',
    is_active   boolean NOT NULL DEFAULT true,
    sort_order  integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (category_id, name)
);

CREATE OR REPLACE TRIGGER on_expense_subcategory_updated
    BEFORE UPDATE ON expense_subcategories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- transactions
-- Supports income / expense / savings with three conditional
-- category constraints, soft delete, and recurrence.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transactions (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type                text NOT NULL CHECK (type IN ('income', 'expense', 'savings')),
    amount              numeric(15,2) NOT NULL CHECK (amount > 0),
    income_category_id  uuid REFERENCES income_categories(id),
    expense_category_id uuid REFERENCES expense_categories(id),
    savings_category_id uuid REFERENCES savings_categories(id),
    subcategory_id      uuid REFERENCES expense_subcategories(id),
    description         text,
    notes               text,
    tags                text[] NOT NULL DEFAULT '{}',
    transaction_date    date NOT NULL DEFAULT CURRENT_DATE,
    is_recurring        boolean NOT NULL DEFAULT false,
    recurring_interval  text CHECK (recurring_interval IN ('daily', 'weekly', 'monthly')),
    source              text NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'ai', 'voice', 'receipt')),
    is_deleted          boolean NOT NULL DEFAULT false,
    deleted_at          timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT check_income_category
        CHECK ((type = 'income' AND income_category_id IS NOT NULL) OR type <> 'income'),
    CONSTRAINT check_expense_category
        CHECK ((type = 'expense' AND expense_category_id IS NOT NULL) OR type <> 'expense'),
    CONSTRAINT check_savings_category
        CHECK ((type = 'savings' AND savings_category_id IS NOT NULL) OR type <> 'savings')
);

CREATE OR REPLACE TRIGGER on_transaction_updated
    BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- transaction_embeddings
-- RAG store (768-dim). Differs from v1: transaction_id is
-- NOT NULL + UNIQUE so Upsert can use ON CONFLICT (transaction_id).
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transaction_embeddings (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transaction_id uuid NOT NULL UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
    content        text NOT NULL,
    embedding      vector(768) NOT NULL,
    metadata       jsonb NOT NULL DEFAULT '{}',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_transaction_embedding_updated
    BEFORE UPDATE ON transaction_embeddings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- gemini_api_keys
-- Per-user encrypted provider keys with priority fallback and
-- modality flags.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS gemini_api_keys (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alias           text NOT NULL,
    encrypted_key   text NOT NULL,
    key_preview     text,
    priority        integer NOT NULL DEFAULT 0,
    status          text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'quota_exceeded', 'error', 'inactive')),
    provider        text NOT NULL DEFAULT 'gemini',
    model           text NOT NULL DEFAULT '',
    supports_text   boolean NOT NULL DEFAULT true,
    supports_audio  boolean NOT NULL DEFAULT false,
    supports_vision boolean NOT NULL DEFAULT false,
    last_used_at    timestamptz,
    last_error      text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, alias)
);

CREATE OR REPLACE TRIGGER on_gemini_key_updated
    BEFORE UPDATE ON gemini_api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- spk_results
-- SPK (decision support) financial health scores.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS spk_results (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score         numeric(5,2) NOT NULL,
    status        text NOT NULL,
    breakdown     jsonb,
    period_start  date,
    period_end    date,
    calculated_at timestamptz NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- chat_sessions
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS chat_sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      text NOT NULL DEFAULT 'Percakapan baru',
    source     text NOT NULL DEFAULT 'chatbot' CHECK (source IN ('chatbot', 'transactions')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_chat_session_updated
    BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- chat_messages
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS chat_messages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  uuid NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        text NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content     text NOT NULL,
    model_used  text,
    tokens_used integer,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- ai_providers
-- Provider registry (admin-managed).
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ai_providers (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                text NOT NULL UNIQUE,
    display_name        text NOT NULL,
    supports_fast       boolean NOT NULL DEFAULT true,
    supports_reasoning  boolean NOT NULL DEFAULT false,
    is_active           boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now()
);
