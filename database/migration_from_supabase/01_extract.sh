#!/usr/bin/env bash
# ============================================================
# 01_extract.sh — extract data from the OLD Supabase Postgres.
#
# Produces CSV files (one per table) in this directory's `extract/` folder,
# suitable for loading into the new schema's `staging` schema.
#
# The old DB is the local Supabase stack (default port 54322).
#   Usage: database/migration_from_supabase/01_extract.sh
#
# Env (defaults): OLD_PGHOST=localhost OLD_PGPORT=54322 OLD_PGUSER=postgres
#                 OLD_PGPASSWORD=postgres OLD_PGDATABASE=postgres
#
# Exports: auth.users (subset), profiles, categories, subcategories,
#          savings_categories, transactions, gemini_api_keys, spk_results,
#          chat_sessions, chat_messages, ai_providers.
# NOT exported: ai_conversations (dead code), transaction_embeddings (regenerated).
# ============================================================
set -euo pipefail

if ! command -v psql >/dev/null 2>&1; then
    for d in "/c/Program Files/PostgreSQL"/*/bin "/mnt/c/Program Files/PostgreSQL"/*/bin /usr/lib/postgresql/*/bin; do
        [[ -x "$d/psql" || -x "$d/psql.exe" ]] && export PATH="$d:$PATH" && break
    done
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${SCRIPT_DIR}/extract"
mkdir -p "${OUT_DIR}"
# psql \copy needs a forward-slash path on Windows.
OUT_DIR="$(echo "${OUT_DIR}" | sed 's|^/d/|D:/|')"

OLD_PGHOST="${OLD_PGHOST:-localhost}"
OLD_PGPORT="${OLD_PGPORT:-54322}"
OLD_PGUSER="${OLD_PGUSER:-postgres}"
OLD_PGPASSWORD="${OLD_PGPASSWORD:-postgres}"
OLD_PGDATABASE="${OLD_PGDATABASE:-postgres}"
export PGHOST="$OLD_PGHOST" PGPORT="$OLD_PGPORT" PGUSER="$OLD_PGUSER" PGPASSWORD="$OLD_PGPASSWORD" PGDATABASE="$OLD_PGDATABASE"

echo "==> Extracting from ${OLD_PGUSER}@${OLD_PGHOST}:${OLD_PGPORT}/${OLD_PGDATABASE}"

copy() {
    local query="$1" file="$2"
    echo "    -> ${file}"
    psql -v ON_ERROR_STOP=1 -q -c "\copy (${query}) TO '${OUT_DIR}/${file}' WITH (FORMAT csv, HEADER true)"
}

copy "SELECT id, email, encrypted_password, raw_user_meta_data, created_at FROM auth.users" auth_users.csv
copy "SELECT id, full_name, role, avatar_url, is_active, created_at, updated_at FROM profiles" profiles.csv
copy "SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at FROM income_categories" income_categories.csv
copy "SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at FROM expense_categories" expense_categories.csv
copy "SELECT id, category_id, name, icon, color, is_active, sort_order, created_at, updated_at FROM expense_subcategories" expense_subcategories.csv
copy "SELECT id, name, icon, color, description, is_active, sort_order, created_by, created_at, updated_at FROM savings_categories" savings_categories.csv
copy "SELECT id, user_id, type, amount, income_category_id, expense_category_id, savings_category_id, subcategory_id, description, notes, tags, transaction_date, is_recurring, recurring_interval, source, is_deleted, deleted_at, created_at, updated_at FROM transactions" transactions.csv
copy "SELECT id, user_id, alias, encrypted_key, key_preview, priority, status, provider, model, supports_text, supports_audio, supports_vision, last_used_at, last_error, created_at, updated_at FROM gemini_api_keys" gemini_api_keys.csv
copy "SELECT id, user_id, score, status, breakdown, period_start, period_end, calculated_at FROM spk_results" spk_results.csv
copy "SELECT id, user_id, title, COALESCE(source,'chatbot') AS source, created_at, updated_at FROM chat_sessions" chat_sessions.csv
copy "SELECT id, session_id, user_id, role, content, model_used, tokens_used, created_at FROM chat_messages" chat_messages.csv
copy "SELECT id, name, display_name, supports_fast, supports_reasoning, is_active, created_at FROM ai_providers" ai_providers.csv

echo "==> Extraction complete -> ${OUT_DIR}"
ls -la "${OUT_DIR}"
