#!/usr/bin/env bash
# ============================================================
# restore.sh — restore a dump into a target database.
#   database/scripts/restore.sh [--db TARGET] [--file PATH]
#
# Defaults: --db fintrack, --file database/dump/fintrack_v2.dump
# The target database must already exist.
# ============================================================
set -euo pipefail

# --- locate pg_restore if not on PATH (Windows-friendly) ---
if ! command -v pg_restore >/dev/null 2>&1; then
    for d in "/c/Program Files/PostgreSQL"/*/bin "/mnt/c/Program Files/PostgreSQL"/*/bin /usr/lib/postgresql/*/bin /opt/homebrew/opt/postgresql*/bin; do
        [[ -x "$d/pg_restore" || -x "$d/pg_restore.exe" ]] && export PATH="$d:$PATH" && break
    done
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5435}"
PGDATABASE="${PGDATABASE:-fintrack}"
PGUSER="${PGUSER:-postgres}"
PGPASSWORD="${PGPASSWORD:-postgres}"
DUMP_FILE="${DB_ROOT}/dump/fintrack_v2.dump"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --db)   PGDATABASE="$2"; shift 2 ;;
        --file) DUMP_FILE="$2"; shift 2 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

export PGHOST PGPORT PGDATABASE PGUSER PGPASSWORD

if [[ ! -f "${DUMP_FILE}" ]]; then
    echo "Dump not found: ${DUMP_FILE}" >&2
    exit 1
fi

echo "==> Restoring ${DUMP_FILE} -> ${PGDATABASE}"
pg_restore --clean --if-exists --no-owner --no-privileges -d "${PGDATABASE}" "${DUMP_FILE}"
echo "==> Restore complete."
