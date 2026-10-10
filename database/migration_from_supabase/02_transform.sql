-- ============================================================
-- 02_transform.sql — load extracted CSVs into the NEW schema.
--
-- Run AFTER 01_extract.sh, against the NEW database (fintrack):
--   psql -h localhost -p 5435 -U postgres -d fintrack \
--        -v ON_ERROR_STOP=1 -f 02_transform.sql
--
-- The extract CSV path is inlined (psql cannot interpolate vars in \copy).
-- (default set below). It stages rows into schema `staging`, maps
-- auth.users + profiles -> users, then copies everything else.
-- The staging schema is dropped at the end.
-- ============================================================

\set ON_ERROR_STOP on

BEGIN;

DROP SCHEMA IF EXISTS staging CASCADE;
CREATE SCHEMA staging;

-- ---- Staging tables mirror the source shape -----------------
CREATE TABLE staging.auth_users (
    id uuid, email text, encrypted_password text, raw_user_meta_data jsonb, created_at timestamptz
);
CREATE TABLE staging.profiles (
    id uuid, full_name text, role text, avatar_url text, is_active boolean, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.income_categories (
    id uuid, name text, icon text, color text, description text, is_active boolean, sort_order int, created_by uuid, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.expense_categories (
    id uuid, name text, icon text, color text, description text, is_active boolean, sort_order int, created_by uuid, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.expense_subcategories (
    id uuid, category_id uuid, name text, icon text, color text, is_active boolean, sort_order int, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.savings_categories (
    id uuid, name text, icon text, color text, description text, is_active boolean, sort_order int, created_by uuid, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.transactions (
    id uuid, user_id uuid, type text, amount numeric(15,2), income_category_id uuid, expense_category_id uuid,
    savings_category_id uuid, subcategory_id uuid, description text, notes text, tags text[],
    transaction_date date, is_recurring boolean, recurring_interval text, source text,
    is_deleted boolean, deleted_at timestamptz, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.gemini_api_keys (
    id uuid, user_id uuid, alias text, encrypted_key text, key_preview text, priority int, status text,
    provider text, model text, supports_text boolean, supports_audio boolean, supports_vision boolean,
    last_used_at timestamptz, last_error text, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.spk_results (
    id uuid, user_id uuid, score numeric(5,2), status text, breakdown jsonb, period_start date, period_end date, calculated_at timestamptz
);
CREATE TABLE staging.chat_sessions (
    id uuid, user_id uuid, title text, source text, created_at timestamptz, updated_at timestamptz
);
CREATE TABLE staging.chat_messages (
    id uuid, session_id uuid, user_id uuid, role text, content text, model_used text, tokens_used int, created_at timestamptz
);
CREATE TABLE staging.ai_providers (
    id uuid, name text, display_name text, supports_fast boolean, supports_reasoning boolean, is_active boolean, created_at timestamptz
);

-- ---- Load CSVs (header true) --------------------------------
\copy staging.auth_users FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/auth_users.csv' WITH (FORMAT csv, HEADER true)
\copy staging.profiles FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/profiles.csv' WITH (FORMAT csv, HEADER true)
\copy staging.income_categories FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/income_categories.csv' WITH (FORMAT csv, HEADER true)
\copy staging.expense_categories FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/expense_categories.csv' WITH (FORMAT csv, HEADER true)
\copy staging.expense_subcategories FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/expense_subcategories.csv' WITH (FORMAT csv, HEADER true)
\copy staging.savings_categories FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/savings_categories.csv' WITH (FORMAT csv, HEADER true)
\copy staging.transactions FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/transactions.csv' WITH (FORMAT csv, HEADER true)
\copy staging.gemini_api_keys FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/gemini_api_keys.csv' WITH (FORMAT csv, HEADER true)
\copy staging.spk_results FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/spk_results.csv' WITH (FORMAT csv, HEADER true)
\copy staging.chat_sessions FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/chat_sessions.csv' WITH (FORMAT csv, HEADER true)
\copy staging.chat_messages FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/chat_messages.csv' WITH (FORMAT csv, HEADER true)
\copy staging.ai_providers FROM 'D:/project/finance/fintrack_v2/database/migration_from_supabase/extract/ai_providers.csv' WITH (FORMAT csv, HEADER true)

-- ---- users: auth.users + profiles merged ---------------------
INSERT INTO users (id, email, password_hash, full_name, role, avatar_url, is_active, created_at, updated_at)
SELECT
    u.id,
    lower(u.email),
    NULLIF(u.encrypted_password, ''),
    COALESCE(NULLIF(p.full_name, ''), split_part(u.email, '@', 1)),
    COALESCE(p.role, 'user'),
    p.avatar_url,
    COALESCE(p.is_active, true),
    u.created_at,
    now()
FROM staging.auth_users u
LEFT JOIN staging.profiles p ON p.id = u.id
ON CONFLICT (id) DO NOTHING;

-- ---- categories & subcategories -----------------------------
INSERT INTO income_categories (id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at)
SELECT id, name, icon, color, description, COALESCE(is_active,true), COALESCE(sort_order,0), created_by, created_at, updated_at
FROM staging.income_categories ON CONFLICT (id) DO NOTHING;

INSERT INTO expense_categories (id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at)
SELECT id, name, icon, color, description, COALESCE(is_active,true), COALESCE(sort_order,0), created_by, created_at, updated_at
FROM staging.expense_categories ON CONFLICT (id) DO NOTHING;

INSERT INTO savings_categories (id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at)
SELECT id, name, icon, color, description, COALESCE(is_active,true), COALESCE(sort_order,0), created_by, created_at, updated_at
FROM staging.savings_categories ON CONFLICT (id) DO NOTHING;

INSERT INTO expense_subcategories (id, category_id, name, icon, color, is_active, sort_order, created_at, updated_at)
SELECT id, category_id, name, icon, color, COALESCE(is_active,true), COALESCE(sort_order,0), created_at, updated_at
FROM staging.expense_subcategories ON CONFLICT (id) DO NOTHING;

-- ---- transactions -------------------------------------------
INSERT INTO transactions (
    id, user_id, type, amount, income_category_id, expense_category_id, savings_category_id,
    subcategory_id, description, notes, tags, transaction_date, is_recurring, recurring_interval,
    source, is_deleted, deleted_at, created_at, updated_at)
SELECT
    id, user_id, type, amount, income_category_id, expense_category_id, savings_category_id,
    subcategory_id, description, notes, COALESCE(tags, '{}'), transaction_date, COALESCE(is_recurring,false),
    recurring_interval, COALESCE(source,'manual'), COALESCE(is_deleted,false), deleted_at, created_at, updated_at
FROM staging.transactions ON CONFLICT (id) DO NOTHING;

-- ---- gemini keys --------------------------------------------
INSERT INTO gemini_api_keys (
    id, user_id, alias, encrypted_key, key_preview, priority, status, provider, model,
    supports_text, supports_audio, supports_vision, last_used_at, last_error, created_at, updated_at)
SELECT
    id, user_id, alias, encrypted_key, key_preview, COALESCE(priority,0), COALESCE(status,'active'),
    COALESCE(provider,'gemini'), COALESCE(model,''),
    COALESCE(supports_text,true), COALESCE(supports_audio,false), COALESCE(supports_vision,false),
    last_used_at, last_error, created_at, updated_at
FROM staging.gemini_api_keys ON CONFLICT (id) DO NOTHING;

-- ---- spk results --------------------------------------------
INSERT INTO spk_results (id, user_id, score, status, breakdown, period_start, period_end, calculated_at)
SELECT id, user_id, score, status, breakdown, period_start, period_end, COALESCE(calculated_at, now())
FROM staging.spk_results ON CONFLICT (id) DO NOTHING;

-- ---- chat ----------------------------------------------------
INSERT INTO chat_sessions (id, user_id, title, source, created_at, updated_at)
SELECT id, user_id, title, COALESCE(NULLIF(source,''),'chatbot'), created_at, updated_at
FROM staging.chat_sessions ON CONFLICT (id) DO NOTHING;

INSERT INTO chat_messages (id, session_id, user_id, role, content, model_used, tokens_used, created_at)
SELECT id, session_id, user_id, role, content, model_used, tokens_used, created_at
FROM staging.chat_messages ON CONFLICT (id) DO NOTHING;

-- ---- providers ----------------------------------------------
INSERT INTO ai_providers (id, name, display_name, supports_fast, supports_reasoning, is_active, created_at)
SELECT id, name, display_name, COALESCE(supports_fast,true), COALESCE(supports_reasoning,false), COALESCE(is_active,true), COALESCE(created_at, now())
FROM staging.ai_providers ON CONFLICT (name) DO NOTHING;

COMMIT;

-- Verification of the load.
\echo '== Loaded row counts =='
SELECT 'users' AS t, count(*) FROM users
UNION ALL SELECT 'transactions', count(*) FROM transactions
UNION ALL SELECT 'gemini_api_keys', count(*) FROM gemini_api_keys
UNION ALL SELECT 'spk_results', count(*) FROM spk_results
UNION ALL SELECT 'chat_sessions', count(*) FROM chat_sessions
UNION ALL SELECT 'chat_messages', count(*) FROM chat_messages
ORDER BY t;

-- Drop staging (comment out to inspect after a failed load).
DROP SCHEMA staging CASCADE;
