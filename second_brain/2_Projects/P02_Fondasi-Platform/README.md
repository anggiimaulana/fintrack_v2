# P02 — Fondasi Platform

## 1. Konteks
FinTrack v2 bertransisi dari monolit sederhana (P01) menuju arsitektur modular multi-instance siap skala produksi. Fase P02 meletakkan fondasi shared infrastructure, logging audit tersertifikasi hukum, dynamic runtime configuration (feature flags), standardisasi presentasi data (DataTable & URL state), serta baseline observability.

Fase ini menjadi pondasi mutlak bagi fase P03 (AI Gateway), P04 (Subscription Engine), P05 (Payment Integration), P06 (Super Admin), P07 (Audit & Compliance), dan P08 (Hardening).

## 2. Ruang Lingkup (Scope)
- **Shared Infrastructure**: Tambah Redis 7 Alpine ke docker-compose (cache bersama + pub/sub invalidasi).
- **Versioning Cache**: Middleware cache Redis berversi (`user:{id}:v{version}:{resource}`) menggantikan cache in-memory lokal, isolasi per-user tanpa kebocoran data.
- **Audit Logging**: Modul Go `internal/audit/` dan tabel append-only `audit_logs` (aktor, aksi, target, diff JSONB, IP, User-Agent, signature tamper-evident) selaras UU PDP No. 27/2022.
- **Feature Flags**: Modul Go `internal/featureflag/` dan tabel `feature_flags` dengan evaluasi runtime per-user, rollout percentage, dan caching Redis.
- **Generic DataTable**: Komponen Next.js `<DataTable>` terstandarisasi TanStack Table v8, server-side pagination default 20, sorting, filtering, dan pencarian.
- **URL State Convention**: Custom hook Next.js untuk sinkronisasi dua arah state tabel dengan search params URL (`?page=&per_page=&sort=&order=&q=&filter_*`).
- **Observability Baseline**: Middleware instrumentasi metrik Four Golden Signals (Latency, Traffic, Errors, Saturation) dan counter/histogram bisnis untuk token, failover, serta payment.

## 3. Deliverables
1. File konfigurasi `docker-compose.yml` dengan service Redis persistent volume.
2. Migrasi database PostgreSQL:
   - `0005_create_audit_logs.sql`
   - `0006_create_feature_flags.sql`
3. Backend Go packages:
   - `internal/infra/redis`: Koneksi pool, pub/sub broadcaster, version manager.
   - `internal/audit`: Service append-only, asynchronous batch worker, hash chain helper.
   - `internal/featureflag`: Evaluator, fallback memory, cache warmer.
   - `internal/middleware/cache`: Cache middleware berversi per-user.
   - `internal/middleware/metrics`: Metrics middleware Prometheus/Expvar.
4. Frontend Next.js modules:
   - `components/data-table/data-table.tsx`: Komponen tabel generik TanStack Table v8.
   - `components/data-table/data-table-pagination.tsx`: Kontrol pagination dan limit 20.
   - `components/data-table/data-table-toolbar.tsx`: Global search dan dynamic filter chips.
   - `hooks/use-data-table-url-state.ts`: Hook sinkronisasi URL search params.
5. Dokumentasi teknis lengkap di folder `second_brain/2_Projects/P02_Fondasi-Platform/teknis/`.

## 4. Kriteria Sukses (Success Criteria)
- [x] Redis beroperasi normal di container Docker, terintegrasi healthcheck dengan backend Go. *(TASK-P02-001)*
- [x] Invalidation cache berbasis versi bekerja instan lintas instance tanpa flushall Redis. *(TASK-P02-003, 004)*
- [x] Mutasi administratif dan billing mencatat audit log append-only dengan latensi per-request backend < 5ms (asynchronous worker). *(TASK-P02-006 — test non-blocking < 10ms)*
- [x] Evaluasi feature flag per-user memiliki fallback aman (default-off) jika Redis offline. *(TASK-P02-008 — fail-closed)*
- [x] Komponen DataTable menyajikan data dari server dengan URL state lengkap (refresh halaman mempertahankan halaman, sorting, filter). *(TASK-P02-009..012)*
- [x] Seluruh migrasi database berprinsip expand/contract (zero-downtime compatible). *(TASK-P02-005, 007 — aditif, replayable)*
- [x] Tidak ada regresi pada fungsionalitas P01 eksisting. *(go test ./... + smoke test live: login 200, `X-Cache: HIT`)*
