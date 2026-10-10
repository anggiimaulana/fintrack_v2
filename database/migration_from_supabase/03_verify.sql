-- ============================================================
-- 03_verify.sql — compare OLD (Supabase) vs NEW (fintrack).
--
-- Run once against each database and diff the output:
--   OLD: psql -h localhost -p 54322 -U postgres -d postgres \
--             -v users_table=auth.users -f 03_verify.sql
--   NEW: psql -h localhost -p 5435  -U postgres -d fintrack \
--             -v users_table=users -f 03_verify.sql
--
-- `:users_table` selects the authoritative user table (auth.users on the
-- old side, users on the new side).
-- ============================================================

\if :{?users_table}
\else
\set users_table 'users'
\endif

\echo '== Per-table row counts =='
SELECT 'users' AS table_name, count(*) AS rows FROM :users_table
UNION ALL SELECT 'income_categories', count(*) FROM income_categories
UNION ALL SELECT 'expense_categories', count(*) FROM expense_categories
UNION ALL SELECT 'expense_subcategories', count(*) FROM expense_subcategories
UNION ALL SELECT 'savings_categories', count(*) FROM savings_categories
UNION ALL SELECT 'transactions', count(*) FROM transactions
UNION ALL SELECT 'gemini_api_keys', count(*) FROM gemini_api_keys
UNION ALL SELECT 'spk_results', count(*) FROM spk_results
UNION ALL SELECT 'chat_sessions', count(*) FROM chat_sessions
UNION ALL SELECT 'chat_messages', count(*) FROM chat_messages
UNION ALL SELECT 'ai_providers', count(*) FROM ai_providers
ORDER BY table_name;

\echo ''
\echo '== Transactions: count + sum(amount) per user (non-deleted) =='
SELECT user_id, count(*) AS tx_count, sum(amount) AS total_amount
FROM transactions
WHERE is_deleted = false
GROUP BY user_id
ORDER BY user_id;

\echo ''
\echo '== gemini_api_keys by status =='
SELECT status, count(*) FROM gemini_api_keys GROUP BY status ORDER BY status;
