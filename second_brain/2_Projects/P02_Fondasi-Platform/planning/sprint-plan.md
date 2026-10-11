# Sprint Plan — P02 Fondasi Platform

## 1. Tujuan Sprint (Sprint Goal)
Mendirikan fondasi infrastruktur shared Redis, sistem audit log append-only yang patuh UU PDP No. 27/2022, dynamic feature flags, generic DataTable frontend dengan integrasi URL state, dan baseline observability untuk mendukung modul-modul lanjutan (AI Gateway, Subscription, Payment, Super Admin) tanpa downtime.

---

## 2. Alokasi Kapasitas & Durasi
- **Durasi Sprint**: 2 Minggu (10 Hari Kerja Efektif)
- **Fokus Arsitektur**:
  - Minggu 1: Infra Redis, Cache Berversi, Audit Logging DB & Worker, Feature Flag Engine.
  - Minggu 2: Frontend DataTable Generic, Hook URL State, Kontrak API Pagination, Observability Metrics, E2E Verification.

---

## 3. Rencana Eksekusi Harian (Sprint Backlog)

### Minggu 1: Backend Foundation, Caching & Compliance

#### Hari 1: Shared Infrastructure Redis
- [ ] TASK-P02-001: Provisioning Redis Container & Healthcheck di `docker-compose.yml`.
- [ ] TASK-P02-002: Client Pool & Lifecycle Manager Redis di Go (`internal/infra/redis`).

#### Hari 2: Versioned Caching Engine
- [ ] TASK-P02-003: Versioned Cache Middleware & Engine (format key `cache:u:{user_id}:v{ver}:...`).
- Implementasi unit test untuk memvalidasi kenaikan versi user membersihkan cache lama secara logis.

#### Hari 3: Multi-Instance Pub/Sub Invalidation
- [ ] TASK-P02-004: Redis Pub/Sub Broadcaster untuk channel `cache:invalidate`.
- Pengujian simulasi 2 instance worker yang menerima signal invalidasi global.

#### Hari 4: Audit Logging Storage & Constraints
- [ ] TASK-P02-005: Migrasi Database Tabel Append-Only `audit_logs` (`0005_create_audit_logs.sql`).
- Uji trigger PostgreSQL untuk memastikan statement UPDATE/DELETE ditolak secara mutlak.

#### Hari 5: Asynchronous Audit Worker & Sanitasi Data
- [ ] TASK-P02-006: Modul `internal/audit` & Asynchronous Worker Queue dengan masking data sensitif (UU PDP).
- Benchmark throughput worker channel Go untuk memastikan latensi API tetap < 1ms.

---

## Minggu 2: Feature Flags, UI DataTable, & Observability

#### Hari 6: Dynamic Feature Flag Engine
- [ ] TASK-P02-007: Migrasi Database Tabel `feature_flags` (`0006_create_feature_flags.sql`).
- [ ] TASK-P02-008: Modul `internal/featureflag` & Evaluator Engine dengan cache Redis L2 dan fallback fail-closed.

#### Hari 7: Backend Pagination Parser & URL State Hook
- [ ] TASK-P02-009: Kontrak API Handler Paginated Data Table di Go (`pkg/pagination`).
- [ ] TASK-P02-010: Custom Hook `useDataTableUrlState` di Next.js (`frontend/src/hooks`).

#### Hari 8: Generic UI DataTable TanStack Table v8
- [ ] TASK-P02-011: Komponen Generic `<DataTable>` TanStack Table v8 dengan skeleton loader dan header sorting.
- Pengujian rendering tabel dengan mockup 100 baris data keuangan.

#### Hari 9: Toolbar, Facet Filtering, & Metrics Instrumentasi
- [ ] TASK-P02-012: Komponen Toolbar (search debounce 300ms) & Pagination selector (10/20/50/100 baris).
- [ ] TASK-P02-013: Instrumentasi Prometheus Metrics di Backend Go (`internal/middleware/metrics.go`).

#### Hari 10: Pengujian Integrasi End-to-End & Sprint Review
- [ ] TASK-P02-014: Integrasi Verifikasi End-to-End Fondasi Platform.
- Audit checklist zero downtime, verifikasi memory leak Redis, dan handover siap untuk Fase P03 (AI Gateway).

---

## 4. Manajemen Risiko Sprint

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Redis instance crash / memory leak | Cache down, potensi sistem lambat | Konfigurasi `maxmemory 256mb` dan `maxmemory-policy allkeys-lru` di redis.conf. Backend memiliki fallback bypass cache. |
| Audit channel overflow saat lonjakan trafik | Kehilangan data audit atau memory spike | Sediakan buffered channel 2.048 event, batch flush interval 500ms, dan failover write darurat ke log file lokal jika PostgreSQL lambat. |
| Perubahan state URL memicu re-render berlebihan | UI lag pada browser klien | Gunakan debounce 300ms pada search input, shallow routing Next.js (`router.replace` dengan opsi `{ scroll: false }`). |
