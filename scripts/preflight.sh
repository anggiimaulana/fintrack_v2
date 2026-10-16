#!/usr/bin/env bash
# ============================================================
# preflight.sh — check the local stack BEFORE starting it.
#
# FinTrack v2 can run two ways, and they fight over the same ports:
#
#   hybrid  Docker for db + redis only; backend & frontend on the host
#           (backend: `go run ./cmd/server`, frontend: `pnpm dev`).
#           This is README "Cara 2" and the fastest dev loop.
#
#   docker  Everything in containers (`docker compose up -d`).
#           This is README "Cara 1".
#
# The classic failure: you leave `pnpm dev` (host) running, then try
# `docker compose up -d`. Docker cannot bind :3001/:8081 and fails with
# "ports are not available ... Only one usage of each socket address".
# This script detects that up front and tells you which process to stop.
#
# Usage:
#   scripts/preflight.sh                 # auto-detect + report everything
#   scripts/preflight.sh --mode hybrid   # "can I start host dev now?"
#   scripts/preflight.sh --mode docker   # "can I run compose up now?"
#
# Exit codes: 0 = no blockers, 1 = at least one blocker for the given mode.
# ============================================================
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

MODE="auto"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) MODE="${2:-auto}"; shift 2 ;;
    --mode=*) MODE="${1#*=}"; shift ;;
    -h|--help) sed -n '2,25p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; exit 2 ;;
  esac
done

case "${MODE}" in
  auto|hybrid|docker) ;;
  *) echo "Invalid --mode '${MODE}' (expected auto|hybrid|docker)" >&2; exit 2 ;;
esac

# --- presentation -------------------------------------------------
if [[ -t 1 ]]; then
  C_RED=$'\033[31m'; C_YEL=$'\033[33m'; C_GRN=$'\033[32m'
  C_DIM=$'\033[2m'; C_BLD=$'\033[1m'; C_RST=$'\033[0m'
else
  C_RED=""; C_YEL=""; C_GRN=""; C_DIM=""; C_BLD=""; C_RST=""
fi

BLOCKERS=0
WARNINGS=0
pass() { printf '  %sPASS%s %s\n' "${C_GRN}" "${C_RST}" "$1"; }
warn() { printf '  %sWARN%s %s\n' "${C_YEL}" "${C_RST}" "$1"; WARNINGS=$((WARNINGS + 1)); }
fail() { printf '  %sFAIL%s %s\n' "${C_RED}" "${C_RST}" "$1"; BLOCKERS=$((BLOCKERS + 1)); }
info() { printf '       %s%s%s\n' "${C_DIM}" "$1" "${C_RST}"; }
head2() { printf '\n%s%s%s\n' "${C_BLD}" "$1" "${C_RST}"; }

# --- helpers ------------------------------------------------------

# Print the PIDs listening on a TCP port (empty when the port is free).
port_pids() {
  local port="$1"
  netstat -ano 2>/dev/null | tr -d '\r' \
    | awk -v p=":${port}\$" '$1=="TCP" && $4=="LISTENING" && $2 ~ p {print $5}' \
    | sort -u
}

# Map PIDs -> "pid|processname" lines, one PowerShell call for all of them.
pid_names() {
  local ids=("$@") joined
  [[ ${#ids[@]} -eq 0 ]] && return 0
  joined="$(printf '%s,' "${ids[@]}")"
  joined="${joined%,}"
  powershell -NoProfile -Command \
    "\$ids=@(${joined}); Get-Process -Id \$ids -ErrorAction SilentlyContinue | ForEach-Object { \"\$(\$_.Id)|\$(\$_.ProcessName)\" }" \
    2>/dev/null | tr -d '\r'
}

# Human-readable owner of a port: "docker" or the host process names.
port_owner() {
  local port="$1" pids
  pids="$(port_pids "${port}")"
  [[ -z "${pids}" ]] && { echo ""; return 0; }
  local names
  names="$(pid_names ${pids} | cut -d'|' -f2 | sort -u | paste -sd, -)"
  # Docker Desktop on Windows publishes ports via com.docker.backend.exe
  # (sometimes alongside wslrelay.exe for WSL2-backed containers).
  if [[ "${names}" == *"com.docker.backend"* ]]; then
    echo "docker"
  else
    echo "${names}"
  fi
}

# Read KEY from an env file (last occurrence wins), CR-safe.
env_val() {
  local file="$1" key="$2"
  [[ -f "${file}" ]] || { echo ""; return 0; }
  grep -E "^${key}=" "${file}" 2>/dev/null | tail -1 | cut -d= -f2- | tr -d '\r' | sed 's/[[:space:]]*$//'
}

# Epoch seconds from an ISO-8601 timestamp (GNU date, present in Git Bash).
iso_epoch() { date -d "$1" +%s 2>/dev/null || echo 0; }

# Container status word: running|exited|missing.
container_state() {
  local name="$1"
  docker inspect "${name}" --format '{{.State.Status}}' 2>/dev/null || echo "missing"
}

cd "${ROOT_DIR}" || exit 2

# --- environment sanity -------------------------------------------
# The port checks shell out to Windows tools (netstat, powershell). On this
# machine `bash` on PATH can resolve to WSL bash, which has neither — and
# would then report every port as "free", masking the exact conflict this
# script exists to catch. Refuse to run rather than lie.
missing_tools=""
for tool in netstat powershell; do
  command -v "${tool}" >/dev/null 2>&1 || missing_tools="${missing_tools} ${tool}"
done
if [[ -n "${missing_tools}" ]]; then
  printf '%sCannot run: missing Windows tool(s):%s%s\n' "${C_RED}" "${missing_tools}" "${C_RST}" >&2
  printf 'You are probably running this under WSL bash, which cannot see Windows\n' >&2
  printf 'processes or Docker Desktop. Use Git Bash instead:\n\n' >&2
  printf '  "C:\\Program Files\\Git\\bin\\bash.exe" scripts/preflight.sh\n\n' >&2
  printf 'or on Windows:  scripts\\preflight.cmd\n' >&2
  exit 2
fi

printf '%sFinTrack v2 — preflight%s  %s(mode: %s)%s\n' \
  "${C_BLD}" "${C_RST}" "${C_DIM}" "${MODE}" "${C_RST}"

# ============================================================
head2 "1. Port ownership"
# ============================================================
declare -A PORT_ROLE=(
  [3001]="frontend"
  [8081]="backend"
  [5435]="postgres"
  [6380]="redis"
)
declare -A PORT_OWNER=()
for port in 3001 8081 5435 6380; do
  owner="$(port_owner "${port}")"
  PORT_OWNER[$port]="${owner}"
  if [[ -z "${owner}" ]]; then
    printf '  %-5s %-9s free\n' "${port}" "${PORT_ROLE[$port]}"
  else
    printf '  %-5s %-9s %s\n' "${port}" "${PORT_ROLE[$port]}" "${owner}"
  fi
done

# ============================================================
head2 "2. Container status"
# ============================================================
CONTAINERS=(fintrack_v2_db fintrack_v2_redis fintrack_v2_backend fintrack_v2_frontend)
declare -A C_STATE=()
if ! command -v docker >/dev/null 2>&1; then
  fail "docker CLI not found on PATH"
else
  if ! docker info >/dev/null 2>&1; then
    fail "Docker daemon is not reachable (is Docker Desktop running?)"
  else
    for c in "${CONTAINERS[@]}"; do
      st="$(container_state "${c}")"
      C_STATE[$c]="${st}"
      case "${st}" in
        running) printf '  %sPASS%s %-22s running\n' "${C_GRN}" "${C_RST}" "${c}" ;;
        exited)  printf '  %s----%s %-22s exited\n' "${C_DIM}" "${C_RST}" "${c}" ;;
        missing) printf '  %s----%s %-22s not created\n' "${C_DIM}" "${C_RST}" "${c}" ;;
        *)       printf '  %s----%s %-22s %s\n' "${C_DIM}" "${C_RST}" "${c}" "${st}" ;;
      esac
    done
  fi
fi

# ============================================================
head2 "3. JWT_SECRET consistency"
# ============================================================
# The frontend verifies the ft_at cookie the backend signs, so both sides
# MUST use the same secret. Each run mode reads a different file:
#   docker  -> root .env          (compose `env_file`)
#   hybrid  -> backend/.env       (godotenv.Load) + frontend/.env.local (next dev)
SEC_ROOT="$(env_val ".env" "JWT_SECRET")"
SEC_BACKEND="$(env_val "backend/.env" "JWT_SECRET")"
SEC_FRONTEND="$(env_val "frontend/.env.local" "JWT_SECRET")"

short() { local v="$1"; [[ -z "${v}" ]] && { echo "(unset)"; return; }; printf '%s…(%d chars)' "${v:0:10}" "${#v}"; }

printf '  %-28s %s\n' "root .env        (docker)" "$(short "${SEC_ROOT}")"
printf '  %-28s %s\n' "backend/.env     (hybrid)" "$(short "${SEC_BACKEND}")"
printf '  %-28s %s\n' "frontend/.env.local (hybrid)" "$(short "${SEC_FRONTEND}")"

if [[ -z "${SEC_BACKEND}" || -z "${SEC_FRONTEND}" ]]; then
  warn "backend/.env or frontend/.env.local missing — hybrid mode auth will fail"
elif [[ "${SEC_BACKEND}" == "${SEC_FRONTEND}" ]]; then
  pass "hybrid pair matches (backend <-> frontend)"
else
  fail "hybrid pair MISMATCH — frontend cannot verify backend cookies"
  info "fix: make JWT_SECRET in backend/.env and frontend/.env.local identical"
fi

if [[ -n "${SEC_ROOT}" && "${SEC_ROOT}" != "${SEC_BACKEND}" ]]; then
  warn "docker secret differs from hybrid secret"
  info "fine if you never mix modes; switching modes invalidates existing cookies"
fi

# ============================================================
head2 "4. Image freshness"
# ============================================================
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  HEAD_EPOCH="$(iso_epoch "$(git log -1 --format=%cI 2>/dev/null)")"
  for svc in backend frontend; do
    img="fintrack_v2-${svc}:latest"
    created="$(docker inspect --format '{{.Created}}' "${img}" 2>/dev/null)"
    if [[ -z "${created}" ]]; then
      printf '  %s----%s %-22s image not built yet\n' "${C_DIM}" "${C_RST}" "${img}"
      continue
    fi
    img_epoch="$(iso_epoch "${created}")"
    behind="$(git rev-list --count --since="${created}" HEAD 2>/dev/null || echo 0)"
    if [[ "${behind}" -gt 0 ]]; then
      warn "${img} is ${behind} commit(s) behind HEAD"
      info "built ${created} — run 'docker compose build ${svc}' before 'up'"
    else
      pass "${img} up to date"
    fi
  done
fi

# ============================================================
head2 "5. Mode readiness"
# ============================================================
owner_is_docker() { [[ "$(port_owner "$1")" == "docker" ]]; }

case "${MODE}" in
  hybrid)
    for p in 5435 6380; do
      if owner_is_docker "${p}"; then
        pass "port ${p} served by Docker (${PORT_ROLE[$p]})"
      else
        fail "port ${p} is not served by Docker — run: docker compose up -d db redis"
      fi
    done
    for p in 3001 8081; do
      if owner_is_docker "${p}"; then
        fail "port ${p} held by a Docker container — 'docker compose stop ${PORT_ROLE[$p]}' first"
      elif [[ -n "${PORT_OWNER[$p]}" ]]; then
        pass "port ${p} already running on host (${PORT_OWNER[$p]})"
      else
        pass "port ${p} free for host dev"
      fi
    done
    ;;
  docker)
    for p in 3001 8081; do
      owner="$(port_owner "${p}")"
      if [[ -z "${owner}" ]]; then
        pass "port ${p} free"
      elif [[ "${owner}" == "docker" ]]; then
        pass "port ${p} already served by Docker"
      else
        fail "port ${p} held by host process (${owner}) — compose up WILL fail to bind"
        info "stop it, e.g. the 'pnpm dev' / 'go run' terminal for ${PORT_ROLE[$p]}"
      fi
    done
    for p in 5435 6380; do
      if [[ -z "${PORT_OWNER[$p]}" ]]; then
        warn "port ${p} free — compose will start ${PORT_ROLE[$p]} itself"
      fi
    done
    ;;
  auto)
    if owner_is_docker 3001 || owner_is_docker 8081; then
      info "detected: docker mode (app ports served by containers)"
    elif [[ -n "${PORT_OWNER[3001]}" || -n "${PORT_OWNER[8081]}" ]]; then
      info "detected: hybrid mode (app ports served by host processes)"
    else
      info "detected: nothing running yet"
    fi
    if [[ -n "${PORT_OWNER[3001]}" && "${PORT_OWNER[3001]}" != "docker" ]] \
       && [[ -n "${PORT_OWNER[8081]}" && "${PORT_OWNER[8081]}" != "docker" ]]; then
      warn "host dev holds :3001 & :8081 — 'docker compose up -d' would fail"
      info "use --mode hybrid to validate, or stop host dev before --mode docker"
    fi
    ;;
esac

# ============================================================
printf '\n%sSummary:%s %d blocker(s), %d warning(s)\n' \
  "${C_BLD}" "${C_RST}" "${BLOCKERS}" "${WARNINGS}"
if [[ "${BLOCKERS}" -gt 0 ]]; then
  printf '%sNot ready.%s Fix the FAIL lines above.\n' "${C_RED}" "${C_RST}"
  exit 1
fi
printf '%sReady.%s\n' "${C_GRN}" "${C_RST}"
exit 0
