#!/usr/bin/env bash
# ============================================================
# dump.sh — regenerate the committed dump.
#   database/dump/fintrack_v2.dump  (custom format, for restore)
#   database/dump/fintrack_v2.sql   (plain, for inspection)
#
# Usage: database/scripts/dump.sh [--db NAME]
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
OUT_DIR="${DB_ROOT}/dump"
mkdir -p "${OUT_DIR}"

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5435}"
PGDATABASE="${PGDATABASE:-fintrack}"
PGUSER="${PGUSER:-postgres}"
PGPASSWORD="${PGPASSWORD:-postgres}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --db) PGDATABASE="$2"; shift 2 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

export PGHOST PGPORT PGDATABASE PGUSER PGPASSWORD

echo "==> Dumping ${PGDATABASE} -> ${OUT_DIR}"
pg_dump -Fc -f "${OUT_DIR}/fintrack_v2.dump"
pg_dump    -f "${OUT_DIR}/fintrack_v2.sql"
echo "==> Wrote:"
ls -lh "${OUT_DIR}/fintrack_v2.dump" "${OUT_DIR}/fintrack_v2.sql"
