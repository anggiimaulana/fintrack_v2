# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Gambaran Proyek

FinTrack v2 — monorepo manajemen keuangan (Go + PostgreSQL + Next.js) dengan AI Multi-Model Gateway, subscription/billing (Midtrans + transfer manual), entitlement/RBAC, dan realtime SSE. Dokumentasi produk lengkap ada di `README.md`; knowledge base operasional di `second_brain/`.

## Batas Zona (WAJIB)

- **Kode produksi:** hanya di `frontend/`, `backend/`, `database/`. Jangan mencemari dengan script test/dump/scratch.
- **Sandbox testing:** semua script debug/test AI → `tester/` (di-`.gitignore`, tidak pernah di-commit).
- **Knowledge & logging:** dokumentasi & knowledge → `second_brain/` (PARA, di-commit); catat aksi teknis signifikan di `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`, dan perbarui HUD progres di `second_brain/2_Projects/P01_FinTrack_v2/roadmap.md`.
- **Secret:** jangan pernah commit `.env`, secret, API key, atau dump berisi secret baru. Hanya `.env.example` yang di-commit.

## Perintah Kanonik

Backend (`backend/`):
```bash
go vet ./... && go test ./... && go build ./...     # gerbang lengkap
go test ./internal/usecase/ -run TestNamaTest -v    # satu test (pola -run)
go test ./internal/aigateway/... -bench . -benchmem # benchmark gateway
go run ./cmd/server                                 # jalankan server (butuh .env)
go run ./cmd/backfill --limit 500                   # backfill embedding RAG
```

Test integrasi/E2E (`backend/tests/` dan `repository/postgres`) butuh DB + Redis aktif; akan **skip** bila env tidak diset. Jalankan dengan:
```bash
REDIS_URL="redis://localhost:6380/0" \
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5435/fintrack_test?sslmode=disable" \
go test -count=1 ./tests/
```
`fintrack_test` dibuat terpisah tanpa seed — lihat `database/README.md`.

Frontend (`frontend/`, pnpm):
```bash
pnpm dev          # dev server di :3001
pnpm build        # production build (wajib hijau)
pnpm lint         # ESLint, 0 error
pnpm test         # Vitest (run sekali); pnpm test:watch untuk watch
pnpm test -- lib/format.test.ts                     # satu file
pnpm test -- -t "nama test"                          # satu test by name
```
Vitest punya dua project: `unit` (node, `lib/**` & `hooks/**`) dan `dom` (jsdom, `*.dom.test.tsx` / `*.a11y.test.tsx`).

Database:
```bash
docker compose up -d db redis
APP_DB_PASSWORD=<pw> database/scripts/migrate.sh --seed   # apply migrasi + seed demo
database/scripts/dump.sh                                   # regenerate dump final (commit)
database/scripts/restore.sh --db fintrack_restore_test     # restore dump
database/scripts/backup.sh                                 # backup bertimestamp (gitignored)
```

Full stack: `docker compose up --build -d`, lalu cek `curl -i http://localhost:8081/health/ready`.

Load test (k6, `scripts/loadtest/`): `gateway_sustained_benchmark.js`, `token_quota_concurrency.js`, `webhook_replay.js`.

## Arsitektur Backend (Go)

Clean Architecture berlapis di `backend/internal/`:

- `delivery/http/` — `router.go` (semua rute + wiring middleware) dan `handler/` (HTTP binding). Rute didaftarkan manual di `SetupRouter`, bukan auto-register.
- `usecase/` — logika bisnis. Orkestrasi lintas domain ada di sini (mis. `TransactionUsecase` memicu `SPKQueue` async; `AIUsecase` merutekan panggilan model lewat gateway terpusat).
- `domain/` — entitas & interface (repository contracts). Tidak boleh import lapisan lain.
- `repository/postgres/` — implementasi pgxpool. Semua query di sini.
- `infrastructure/` — `auth`, `crypto`, `db`, `embedding`, `metrics`, `redis` (adapter teknis).
- `admin/`, `billing/`, `payment/`, `entitlement/`, `realtime/`, `aigateway/`, `audit/`, `featureflag/`, `user/` — modul domain mandiri dengan handler/route/service sendiri (Super Admin memakai `admin.RegisterRoutes`).
- `pkg/` — util lintas-lapisan: `response` (sanitasi + envelope error sukses/gagal), `pagination`, `validator`, `contracts` (envelope event SSE), `logger` (Zap).

Wiring raksasa ada di `cmd/server/main.go`: baca env → pool DB + Redis → konstruksi repository → usecase → service (gateway, billing, payment) → `SetupRouter`. **Semua dependency di-inject manual**; tidak ada DI container.

Konsep lintas-lapisan yang perlu dipahami sebelum mengubah kode:

- **Entitlement enforcement (P07)** — `entitlement.Resolver` me-resolve fitur + limit dari plan, di-cache berversi di Redis. Enforcement lewat middleware `RequireFeature`/`WithinLimit`, dipasang per-rute via helper `featureGate`/`limitGate` di `router.go`. Kalau `RouterConfig.Entitlement` nil (test lama), gating dimatikan total. Bypass operasional: `ENTITLEMENT_ENFORCEMENT=off`.
- **RBAC ada di Go, bukan RLS.** Role `fintrack_app` least-privilege (tanpa DDL). Tiga tingkat: `user` < `admin` < `super_admin`, dicek via middleware `RequireUser`/`RequireAdmin`/`RequireSuperAdmin`. Super admin di-resolve per-request agar demosi langsung berlaku.
- **Cache berversi + Pub/Sub.** `middleware.CacheMiddleware(30s)` membungkus rute GET; invalidasi lewat bump versi yang disiarkan via Redis Pub/Sub ke semua pod (tanpa restart). SSE stream **tidak boleh** masuk grup cached.
- **AI Gateway (P03)** — `aigateway/gateway` melakukan routing SWRR antar-combo, failover `priority ASC`, circuit breaker per-endpoint. Gerbang di-ON-kan hanya jika feature flag `feature_ai_gateway_centralized` ON **dan** ada route chat aktif **dan** ada endpoint sehat (`HasHealthyEndpoint`); kalau tidak, gateway tetap OFF dan tidak ada jalan AI (jangan ON-kan flag tanpa endpoint sehat). Kill switch `admin.KillSwitchStore` menghentikan use case AI via Pub/Sub dalam ~100ms.
- **Gateway adalah satu-satunya transport AI.** Jalur BYOK per-user sudah **dihapus total** (tabel `gemini_api_keys` di-drop oleh migrasi `0014`; `internal/domain/gemini_key.go`, `getAvailableKeys`, `findWorkingKey*`, `decryptAPIKey`, dan seluruh `callProvider`/`analyzeReceipt*`/`voiceProcess*` sudah dibuang). `AIUsecase` tidak lagi menyimpan `keyRepo`/`cryptoKey`/`httpClient`. Saat `gateway == nil`, chat/vision/voice mengembalikan pesan "layanan tidak tersedia" — bukan fallback.
- **Konfigurasi gateway dinamis (default env → DB menimpa).** `internal/aigateway/bootstrap` mengisi `ai_endpoints`/`ai_routes`/flag `feature_ai_gateway_centralized` dari env **hanya saat tabel masih kosong**; begitu ada data (dari Super Admin atau bootstrap sebelumnya), DB otoritatif dan env diabaikan untuk tabel itu. Env: `AI_GATEWAY_BASE_URL`, `AI_GATEWAY_API_KEY` (dienkripsi AES-256-GCM saat disimpan), `AI_GATEWAY_ENABLED`, `AI_GATEWAY_MODEL_{CHAT,REASONING,VISION,VOICE,EMBEDDING}`. Bootstrap berjalan sebelum health checker, yang lalu `RunOnce` sinkron agar keputusan ON/OFF tidak membaca health `unknown` yang basi. Setelah itu semuanya diubah lewat `/api/v1/admin/ai-gateway/*` tanpa redeploy.
- **Embedder mengikuti route, bukan hardcode.** `GatewayEmbedder` tidak lagi memaksa `text-embedding-004`; model embedding diambil dari route `embedding` di `ai_routes` (Super Admin). Gateway tetap mengirim `dimensions:768` dan mengunci `vector(768)`.
- **Realtime SSE (P07)** — `realtime.Hub` (lokal) + Redis Pub/Sub (lintas pod) + ring-buffer untuk resume `Last-Event-ID`. Event envelope ada di `pkg/contracts/events.go`. `REALTIME_ENABLED=off` mematikan endpoint (503) + publishing.
- **Sanitasi error universal (CWE-209).** Semua respons error melewati `pkg/response`; `sanitizeMessage`/`sanitizeErrorCode` menetralkan SQL/SQLSTATE/struct Go/stack trace sebelum keluar. Jangan kirim `err.Error()` mentah ke klien — pakai helper `response.*`. Panic ditangkap `gin.CustomRecovery` dan hanya di-log ke Zap.
- **Graceful shutdown** — `SHUTDOWN_TIMEOUT` (default 30s) harus ≥ `stop_grace_period` compose.

## Arsitektur Frontend (Next.js App Router)

- `app/` — route groups: `(auth)`, `(dashboard)/user`, `(dashboard)/admin`, `(dashboard)/super-admin`. Server component sebagai default.
- `lib/api.ts` — Axios client ke same-origin `/api/v1` (di-rewrite ke backend oleh `next.config.ts`), `withCredentials` sehingga cookie httpOnly terkirim. Ada single-flight refresh pada 401.
- `proxy.ts` — edge auth guard (bukan `middleware.ts`); memverifikasi cookie `ft_at` dengan `jose` lalu memanggil `lib/roles.ts:decideRoute` (satu sumber logika routing, dipakai juga oleh test).
- `hooks/` — TanStack Query hooks per domain (mis. `useTransactions`, `useSuperAdmin*`); `use-realtime-stream.ts` berlangganan SSE + dispatch event ke cache Query.
- `lib/cache-keys.ts` / `realtime-dispatcher.ts` — konvensi query key & pemetaan event SSE → invalidasi cache.
- `components/ui/` = shadcn; `components/data-table/` = tabel generik (TanStack Table); aksesibilitas diuji dengan axe-core (`*.a11y.test.tsx`).
- **Profil pengguna** (`app/(dashboard)/user/profile/page.tsx`) — nama, telepon, dan avatar. Avatar diunggah via `POST /api/v1/me/avatar` (multipart, maks 2 MB, magic-byte JPG/PNG/WEBP) dan disajikan dari endpoint **publik** `GET /api/v1/avatars/:file` (nama berkas memuat UUID user, jadi `<img>` bisa memuat tanpa cookie). `PUT /api/v1/me` bersifat **full update**: `phone: ""` mengosongkan (NULL), bukan mengabaikan. Telepon & avatar termasuk PII — ikut dihapus saat anonimisasi UU PDP.
- **Sidebar collapsible** (`components/layout/Sidebar.tsx` + `hooks/use-sidebar-collapsed.ts`) — tombol tutup/buka, animasi lebar 300ms, tooltip saat ciut, state disimpan di `localStorage` via `useSyncExternalStore` (aman SSR, tersinkron antar-tab).

Cookie auth: `ft_at` (access) & `ft_rt` (refresh), httpOnly, diset backend. `JWT_SECRET` frontend harus sama dengan backend.

## Database

PostgreSQL 17 + `pgvector`. Tanpa RLS, tanpa schema Supabase. Perubahan schema → **migrasi bernomor baru** (`database/migrations/000N_*.sql`), jangan edit migrasi lama yang sudah di-commit, lalu `dump.sh` untuk regenerate dump. `seeds/` bukan migrasi (data demo, idempoten). DDL dijalankan sebagai `postgres` (owner); `fintrack_app` tanpa DDL. Detail: `database/README.md`.

## Archival Rule

Never move projects to `5_Archives/` without explicit user confirmation.
