#!/usr/bin/env bash
# ============================================================
# migrate.sh — apply database/migrations/*.sql in order.
#
# Usage:
#   database/scripts/migrate.sh [--seed] [--db NAME] [--host H] [--port P]
#
# Env (with defaults):
#   PGHOST=localhost PGPORT=5435 PGDATABASE=fintrack
#   PGUSER=postgres PGPASSWORD=postgres
#   APP_DB_PASSWORD=<password for the fintrack_app role>
#
# Idempotent: migrations use IF NOT EXISTS / OR REPLACE / DO-blocks.
# ============================================================
set -euo pipefail

# --- locate psql/pg_dump if not on PATH (Windows-friendly) ---
if ! command -v psql >/dev/null 2>&1; then
    for d in "/c/Program Files/PostgreSQL"/*/bin "/mnt/c/Program Files/PostgreSQL"/*/bin /usr/lib/postgresql/*/bin /opt/homebrew/opt/postgresql*/bin; do
        [[ -x "$d/psql" || -x "$d/psql.exe" ]] && export PATH="$d:$PATH" && break
    done
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
MIG_DIR="${DB_ROOT}/migrations"
SEED_DIR="${DB_ROOT}/seeds"

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5435}"
PGDATABASE="${PGDATABASE:-fintrack}"
PGUSER="${PGUSER:-postgres}"
PGPASSWORD="${PGPASSWORD:-postgres}"
APP_DB_PASSWORD="${APP_DB_PASSWORD:-postgres}"
RUN_SEED=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --seed) RUN_SEED=1; shift ;;
        --db)   PGDATABASE="$2"; shift 2 ;;
        --host) PGHOST="$2"; shift 2 ;;
        --port) PGPORT="$2"; shift 2 ;;
        -h|--help) sed -n '2,16p' "${BASH_SOURCE[0]}"; exit 0 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

export PGHOST PGPORT PGDATABASE PGUSER PGPASSWORD

echo "==> Migrating ${PGUSER}@${PGHOST}:${PGPORT}/${PGDATABASE}"

for f in $(ls "${MIG_DIR}"/*.sql | sort); do
    name="$(basename "$f")"
    echo "    -> ${name}"
    psql -v ON_ERROR_STOP=1 \
         -v app_password="${APP_DB_PASSWORD}" \
         -q -f "$f"
done

echo "==> Migrations applied."

if [[ "$RUN_SEED" == "1" ]]; then
    echo "==> Seeding..."
    for f in $(ls "${SEED_DIR}"/*.sql | sort); do
        name="$(basename "$f")"
        echo "    -> ${name}"
        psql -v ON_ERROR_STOP=1 -q -f "$f"
    done
    echo "==> Seed applied."
fi

echo "==> Done."
