#!/usr/bin/env bash
# ============================================================
# backup.sh — timestamped dump into database/backups/ (gitignored).
#
# Usage: database/scripts/backup.sh
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

echo "==> Backing up ${PGDATABASE} -> ${OUT} (via ${DUMP_VIA})"
if [[ "${DUMP_VIA}" == "local" ]]; then
    pg_dump -Fc -f "${OUT}"
else
    DEST_FILE="${OUT}"
    if command -v wslpath >/dev/null 2>&1; then
        DEST_FILE="$(wslpath -w "${OUT}")"
    fi
    "${DUMP_VIA}" exec "${DB_CONTAINER}" pg_dump -U "${PGUSER}" -d "${PGDATABASE}" -Fc -f /tmp/backup.dump
    "${DUMP_VIA}" cp "${DB_CONTAINER}:/tmp/backup.dump" "${DEST_FILE}"
    "${DUMP_VIA}" exec "${DB_CONTAINER}" rm -f /tmp/backup.dump
fi
ls -lh "${OUT}"
