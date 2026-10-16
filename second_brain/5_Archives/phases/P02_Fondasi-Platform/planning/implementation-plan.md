# Implementation Plan — P02 Fondasi Platform

| Field | Isi |
|---|---|
| **Fase** | P02 — Fondasi Platform |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | P01 selesai (monorepo Go + PostgreSQL + Next.js jalan) |
| **Master plan** | `../../IMPLEMENTATION-PLAN.md` |
| **Status** | Siap dieksekusi — belum ada kode ditulis |

> **Koreksi path:** kode frontend berada di `frontend/` (bukan `frontend/src/`).
> Path nyata: `frontend/app/`, `frontend/components/`, `frontend/hooks/`, `frontend/lib/`.
> Semua task di bawah memakai path ini.

---

## 1. Tujuan & Definisi Selesai

Menyediakan **fondasi bersama** yang dibutuhkan semua fase berikutnya:
Redis (cache berversi + pub/sub), audit log append-only, feature flag, DataTable generik + URL state,
dan observability dasar — **tanpa regresi** pada P01 dan **zero downtime**.

**Definition of Done fase:**
- [ ] Semua 14 task ✅ dengan bukti (test/artefak).
- [ ] `go vet && go test ./... && go build ./...` hijau.
- [ ] `pnpm build && pnpm lint && pnpm test` hijau.
- [ ] Migrasi `0005`, `0006` replayable; dump di-regenerate.
- [ ] Tidak ada regresi fungsional P01 (login, transaksi, dashboard, chat).
- [ ] Fitur baru di balik feature flag `ff_datatable` (default OFF).

---

## 2. Urutan Eksekusi (Wave)

| Wave | Task | Prasyarat |
|---|---|---|
| **W1** | 001, 002 | — |
| **W2** | 003, 004, 013 | W1 |
| **W3** | 005, 006, 007, 008 | W2 (Redis untuk 008) |
| **W4** | 009, 010, 011, 012 | W2 (009 tidak butuh Redis) |
| **W5** | 014 | semua |

---

## 3. Detail Task

### W1 — Infrastruktur Redis

#### TASK-P02-001 — Provisioning Redis Container & Healthcheck
- **Target:** `docker-compose.yml`, `.env.example`
- **Langkah:**
  1. Tambah service `redis` (image `redis:7-alpine`), port `6379`, volume `fintrack_v2_redis`.
  2. Konfigurasi `maxmemory 256mb`, `maxmemory-policy allkeys-lru`.
  3. Healthcheck: `redis-cli ping` → `PONG`.
  4. Tambah `REDIS_URL`/`REDIS_PASSWORD` di `.env.example`.
- **Verifikasi (bukti):** `docker compose up -d redis` → `docker compose ps` menunjukkan `healthy`; `docker compose exec redis redis-cli ping` → `PONG`.
- **Catatan zero-downtime:** penambahan service tidak mengganggu layanan berjalan.

#### TASK-P02-002 — Client Pool & Lifecycle Manager Redis (Go)
- **Target:** `backend/internal/infra/redis/client.go`, `config.go`
- **Langkah:**
  1. `go get github.com/redis/go-redis/v9`.
  2. Pool config dari env; `Ping` saat boot; **graceful degradation** bila Redis offline (log warning, lanjut tanpa cache).
  3. Graceful shutdown (tutup pool saat server stop).
- **Verifikasi:** test unit koneksi (skip bila `REDIS_URL` kosong, env-gated); log boot jelas.

### W2 — Cache Berversi, Pub/Sub, Observability

#### TASK-P02-003 — Versioned Cache Middleware & Engine
- **Target:** `backend/internal/infra/redis/cache.go`, `backend/internal/delivery/http/middleware/cache.go`
- **Desain:** key = `cache:u:{user_id}:v{ver}:{sha256(method:path:query)}`.
  - `GetVersion(ctx, userID)`, `BumpUserVersion(ctx, userID)` (O(1), `INCR`).
  - Migrasi dari `globalCache` in-memory → Redis, **mempertahankan** isolasi per-user (tidak bocor).
  - Bila Redis offline → fallback bypass (tanpa cache), bukan error.
- **Verifikasi:** GET pertama → `X-Cache: MISS`; GET kedua → `HIT`; `BumpUserVersion` → GET berikutnya `MISS` (cache baru). Test: `go test ./internal/... -run Cache`.

#### TASK-P02-004 — Redis Pub/Sub Broadcaster (Invalidasi Lintas Instance)
- **Target:** `backend/internal/infra/redis/pubsub.go`
- **Desain:** channel `cache:invalidate`; subscriber background membersihkan lookup lokal; publish saat mutasi (`InvalidateUserCache`, `InvalidateAllCache`).
- **Verifikasi:** simulasi 2 instance → publish di satu node, node lain menerima < 50ms. Test env-gated.

#### TASK-P02-013 — Observability Baseline (Metrik)
- **Target:** `backend/internal/delivery/http/middleware/metrics.go`, `backend/cmd/server/main.go`
- **Desain:** four golden signals (latency histogram, traffic counter, error counter, saturation gauge) + counter bisnis placeholder (token, failover, payment) untuk diisi fase lanjut. Endpoint `/metrics`.
- **Verifikasi:** `curl localhost:8080/metrics` → format eksposisi Prometheus.

### W3 — Audit & Feature Flag

#### TASK-P02-005 — Migrasi `audit_logs` (append-only)
- **Target:** `database/migrations/0005_create_audit_logs.sql`
- **Desain:** tabel `audit_logs(id, actor_id, action, entity, entity_id, before jsonb, after jsonb, ip, user_agent, created_at)`; index pada `created_at`, `actor_id`, `entity`; **trigger** menolak `UPDATE`/`DELETE`.
- **Verifikasi:** migrate sukses; `UPDATE audit_logs ...` → exception `table audit_logs is append-only`.

#### TASK-P02-006 — Modul `internal/audit` & Worker Asinkron
- **Target:** `backend/internal/audit/{audit.go,worker.go,masker.go}`
- **Desain:** buffered channel (2048), batch insert (100 event / 500ms), **masking** field sensitif (`password`, `token`, `api_key` → `[REDACTED]`), failover tulis ke log file bila DB lambat.
- **Verifikasi:** `audit.Log(ctx, e)` non-blocking < 1ms; test masking.

#### TASK-P02-007 — Migrasi `feature_flags`
- **Target:** `database/migrations/0006_create_feature_flags.sql`
- **Desain:** `feature_flags(id, key UNIQUE, name, description, is_enabled, rollout int, updated_by, created_at, updated_at)`.
- **Verifikasi:** migrate sukses; unique constraint pada `key`.

#### TASK-P02-008 — Modul `internal/featureflag` & Evaluator
- **Target:** `backend/internal/featureflag/{evaluator.go,service.go}`
- **Desain:** `IsEnabled(ctx, key, evalCtx) bool`; cache Redis + fallback DB; **fail-closed** (default OFF bila error); rollout percentage deterministik per user ID (hash).
- **Verifikasi:** evaluasi < 1ms (cache); user ID sama → hasil konsisten. Seed `ff_*` semua OFF (lihat doc 07).

### W4 — DataTable & URL State

#### TASK-P02-009 — Parser Pagination/Sort/Filter (Go)
- **Target:** `backend/pkg/pagination/{pagination.go,envelope.go}`
- **Desain:** parse `?page=&per_page=&sort=&order=&q=&filter_*`; clamp `per_page` 10–100 (default 20); **whitelist kolom sort** (anti SQL injection); pakai envelope `meta{total,page,per_page}` yang sudah ada.
- **Verifikasi:** test parser (clamp, whitelist, default).

#### TASK-P02-010 — Hook `useDataTableUrlState` (Next.js)
- **Target:** `frontend/hooks/use-data-table-url-state.ts`
- **Desain:** dua arah dengan `useSearchParams` + `router.replace({scroll:false})`; mengelola `page, per_page, sort, order, q, filter_*`.
- **Verifikasi:** Vitest — mengubah halaman/sort memperbarui URL; reload mempertahankan state.

#### TASK-P02-011 — Komponen `<DataTable>` (TanStack Table v8)
- **Target:** `frontend/components/data-table/{data-table.tsx,data-table-column-header.tsx}`
- **Desain:** headless TanStack Table v8; props `data, columns, meta, isLoading`; skeleton loader; sortable header (asc→desc→default); server-side (tidak sorting di klien); dibalik flag `ff_datatable`.
- **Verifikasi:** render 100 baris mock; klik header mengubah param `sort`/`order`.

#### TASK-P02-012 — Toolbar & Pagination
- **Target:** `frontend/components/data-table/{data-table-toolbar.tsx,data-table-pagination.tsx}`
- **Desain:** search debounce 300ms → param `q`; selector 10/20/50/100 → `per_page` (reset page=1); info total; semua kontrol `cursor-pointer`.
- **Verifikasi:** Vitest — debounce & reset page.

### W5 — Verifikasi

#### TASK-P02-014 — Integrasi E2E Fondasi
- **Target:** `backend/internal/repository/postgres/integration_test.go` (atau `platform_foundation_test.go`, env-gated)
- **Cakupan:** (1) cache HIT/MISS/bump version; (2) audit tersimpan asinkron + masking; (3) evaluasi feature flag; (4) parser pagination.
- **Verifikasi:** test hijau; checklist zero downtime; tidak ada regresi P01.

---

## 4. Matriks Bukti (Evidence)

| Task | Artefak bukti |
|---|---|
| 001 | output `docker compose ps` (healthy) |
| 002 | log boot + test koneksi |
| 003 | test cache + header `X-Cache` |
| 004 | test pub/sub 2 node |
| 005 | output migrasi + error trigger |
| 006 | test masking + benchmark < 1ms |
| 007 | output migrasi |
| 008 | test evaluator + seed flags |
| 009 | test parser |
| 010 | Vitest URL state |
| 011 | Vitest/screenshot DataTable |
| 012 | Vitest toolbar/pagination |
| 013 | `curl /metrics` |
| 014 | log test E2E hijau |

---

## 5. Perintah Verifikasi

```bash
# Backend
cd backend && go vet ./... && go test ./... && go build ./...

# Frontend
cd frontend && pnpm build && pnpm lint && pnpm test

# Migrasi (idempoten)
APP_DB_PASSWORD=<pw> database/scripts/migrate.sh
database/scripts/dump.sh
```

---

## 6. Rollback

- Semua fitur baru di balik feature flag → **flag OFF** = rollback instan tanpa deploy.
- Migrasi **aditif** → tidak perlu mundur; biarkan (aman untuk kode lama).
- Bila Redis bermasalah → backend fallback bypass (layanan tetap jalan).

---

## 7. Handoff ke P03

Setelah P02 selesai, tersedia untuk P03: Redis pool, cache berversi, audit log, feature flag, parser
pagination, komponen DataTable, dan `/metrics`. P03 (AI Gateway) dapat langsung memakai
`featureflag` (kill switch/canary), `audit` (log akses kunci), dan Redis (health/route cache).

## Referensi

- `README.md`, `planning/tasks.md`, `planning/timeline.md`, `teknis/*.md` (fase ini)
- `../../IMPLEMENTATION-PLAN.md`
- `../../../3_Areas/operasional/RUNBOOK_ZERO_DOWNTIME.md`
