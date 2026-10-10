-- ============================================================
-- verify.sql — row counts + money sums per table.
-- Usage: psql -f database/scripts/verify.sql
-- ============================================================
\echo '== Row counts =='
SELECT 'users' AS table_name, count(*) FROM users
UNION ALL SELECT 'refresh_tokens', count(*) FROM refresh_tokens
UNION ALL SELECT 'income_categories', count(*) FROM income_categories
UNION ALL SELECT 'expense_categories', count(*) FROM expense_categories
UNION ALL SELECT 'savings_categories', count(*) FROM savings_categories
UNION ALL SELECT 'expense_subcategories', count(*) FROM expense_subcategories
UNION ALL SELECT 'transactions', count(*) FROM transactions
UNION ALL SELECT 'transaction_embeddings', count(*) FROM transaction_embeddings
UNION ALL SELECT 'gemini_api_keys', count(*) FROM gemini_api_keys
UNION ALL SELECT 'spk_results', count(*) FROM spk_results
UNION ALL SELECT 'chat_sessions', count(*) FROM chat_sessions
UNION ALL SELECT 'chat_messages', count(*) FROM chat_messages
UNION ALL SELECT 'ai_providers', count(*) FROM ai_providers
ORDER BY table_name;

\echo ''
\echo '== Transactions: count + sum(amount) per user (non-deleted) =='
SELECT user_id,
       count(*)          AS tx_count,
       sum(amount)       AS total_amount
FROM transactions
WHERE is_deleted = false
GROUP BY user_id
ORDER BY user_id;

\echo ''
\echo '== Transactions by type =='
SELECT type, count(*) AS cnt, sum(amount) AS total
FROM transactions
WHERE is_deleted = false
GROUP BY type
ORDER BY type;
