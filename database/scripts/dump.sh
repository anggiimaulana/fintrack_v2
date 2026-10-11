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
    for d in "/c/Program Files/PostgreSQL"/*/bin "/mnt/c/Program Files/PostgreSQL"/*/bin "C:/Program Files/PostgreSQL"/*/bin /usr/lib/postgresql/*/bin /opt/homebrew/opt/postgresql*/bin; do
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

# --- fall back to the running db container when no local pg_dump exists ---
DB_CONTAINER="${DB_CONTAINER:-fintrack_v2_db}"
DUMP_VIA="local"
if ! command -v pg_dump >/dev/null 2>&1; then
    DUMP_VIA=""
    for candidate in docker docker.exe "/mnt/c/Program Files/Docker/Docker/resources/bin/docker.exe"; do
        if command -v "${candidate}" >/dev/null 2>&1 && "${candidate}" exec "${DB_CONTAINER}" pg_dump --version >/dev/null 2>&1; then
            DUMP_VIA="${candidate}"
            break
        fi
    done
    if [[ -z "${DUMP_VIA}" ]]; then
        echo "pg_dump not found locally and container '${DB_CONTAINER}' is not reachable" >&2
        exit 1
    fi
fi

echo "==> Dumping ${PGDATABASE} -> ${OUT_DIR} (via ${DUMP_VIA})"
if [[ "${DUMP_VIA}" == "local" ]]; then
    pg_dump -Fc -f "${OUT_DIR}/fintrack_v2.dump"
    pg_dump    -f "${OUT_DIR}/fintrack_v2.sql"
else
    # dump inside the container then copy, so the byte stream never crosses a
    # Windows console/pipe (keeps the custom-format archive intact).
    # A Windows docker.exe cannot resolve POSIX paths, so hand it the Windows form.
    DEST_DIR="${OUT_DIR}"
    if command -v wslpath >/dev/null 2>&1; then
        DEST_DIR="$(wslpath -w "${OUT_DIR}")"
    fi
    "${DUMP_VIA}" exec "${DB_CONTAINER}" pg_dump -U "${PGUSER}" -d "${PGDATABASE}" -Fc -f /tmp/fintrack_v2.dump
    "${DUMP_VIA}" exec "${DB_CONTAINER}" pg_dump -U "${PGUSER}" -d "${PGDATABASE}"    -f /tmp/fintrack_v2.sql
    "${DUMP_VIA}" cp "${DB_CONTAINER}:/tmp/fintrack_v2.dump" "${DEST_DIR}/fintrack_v2.dump"
    "${DUMP_VIA}" cp "${DB_CONTAINER}:/tmp/fintrack_v2.sql"  "${DEST_DIR}/fintrack_v2.sql"
    "${DUMP_VIA}" exec "${DB_CONTAINER}" rm -f /tmp/fintrack_v2.dump /tmp/fintrack_v2.sql
fi
echo "==> Wrote:"
ls -lh "${OUT_DIR}/fintrack_v2.dump" "${OUT_DIR}/fintrack_v2.sql"
