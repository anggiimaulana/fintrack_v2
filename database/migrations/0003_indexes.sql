-- ============================================================
-- 0003_indexes.sql
-- Valid indexes only (v1's 011_performance_indices.sql referenced
-- non-existent columns; rewritten clean here).
-- ============================================================

-- users ------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_users_role_active ON users (role, is_active);

-- refresh_tokens ---------------------------------------------
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id  ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires  ON refresh_tokens (expires_at);

-- transactions -----------------------------------------------
CREATE INDEX IF NOT EXISTS idx_transactions_user_date    ON transactions (user_id, transaction_date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_type_date ON transactions (user_id, type, transaction_date);
CREATE INDEX IF NOT EXISTS idx_transactions_user_deleted ON transactions (user_id, is_deleted);
CREATE INDEX IF NOT EXISTS idx_transactions_user_created ON transactions (user_id, created_at DESC);

-- transaction_embeddings -------------------------------------
CREATE INDEX IF NOT EXISTS idx_transaction_embeddings_hnsw
    ON transaction_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_transaction_embeddings_user_id
    ON transaction_embeddings (user_id);

-- chat_messages ----------------------------------------------
CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages (session_id, created_at);

-- chat_sessions ----------------------------------------------
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_source ON chat_sessions (user_id, source);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_updated_at  ON chat_sessions (updated_at DESC);

-- gemini_api_keys --------------------------------------------
CREATE INDEX IF NOT EXISTS idx_gemini_keys_user_status_priority
    ON gemini_api_keys (user_id, status, priority);
CREATE INDEX IF NOT EXISTS idx_gemini_keys_modality
    ON gemini_api_keys (user_id, status, supports_text, supports_audio, supports_vision);

-- spk_results ------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_spk_results_user_calculated
    ON spk_results (user_id, calculated_at DESC);

-- categories -------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_income_categories_active  ON income_categories  (is_active, sort_order);
CREATE INDEX IF NOT EXISTS idx_expense_categories_active ON expense_categories (is_active, sort_order);
CREATE INDEX IF NOT EXISTS idx_savings_categories_active ON savings_categories (is_active, sort_order);
