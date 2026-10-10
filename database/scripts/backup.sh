#!/usr/bin/env bash
# ============================================================
# backup.sh — timestamped dump into database/backups/ (gitignored).
#
# Usage: database/scripts/backup.sh
# ============================================================
set -euo pipefail

# --- locate pg_dump if not on PATH (Windows-friendly) ---
if ! command -v pg_dump >/dev/null 2>&1; then
    for d in "/c/Program Files/PostgreSQL"/*/bin "/mnt/c/Program Files/PostgreSQL"/*/bin /usr/lib/postgresql/*/bin /opt/homebrew/opt/postgresql*/bin; do
        [[ -x "$d/pg_dump" || -x "$d/pg_dump.exe" ]] && export PATH="$d:$PATH" && break
    done
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUT_DIR="${DB_ROOT}/backups"
mkdir -p "${OUT_DIR}"

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5435}"
PGDATABASE="${PGDATABASE:-fintrack}"
PGUSER="${PGUSER:-postgres}"
PGPASSWORD="${PGPASSWORD:-postgres}"
export PGHOST PGPORT PGDATABASE PGUSER PGPASSWORD

STAMP="$(date +%Y%m%d_%H%M%S)"
OUT="${OUT_DIR}/${PGDATABASE}_${STAMP}.dump"

echo "==> Backing up ${PGDATABASE} -> ${OUT}"
pg_dump -Fc -f "${OUT}"
ls -lh "${OUT}"
