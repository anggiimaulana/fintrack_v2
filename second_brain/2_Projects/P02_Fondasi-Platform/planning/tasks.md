# Tasks Breakdown — P02 Fondasi Platform

Daftar rincian task teknis untuk pengerjaan Fase P02. Mengikuti format `TASK-P02-NNN`.

---

## 1. Infrastruktur & Shared Redis

### TASK-P02-001: Provisioning Redis Container & Healthcheck
- **Deskripsi**: Menambahkan service Redis 7-Alpine ke `docker-compose.yml` dengan konfigurasi persistent volume (`redis_data`), command binding, password via environment, memory limit, dan healthcheck `redis-cli ping`.
- **Target Files**: `docker-compose.yml`, `.env.example`.
- **Kriteria Penerimaan**:
  - Service `redis` start otomatis pada `docker compose up -d`.
  - Healthcheck mengembalikan status `healthy` dalam waktu 5 detik.

### TASK-P02-002: Client Pool & Lifecycle Manager Redis di Go
- **Deskripsi**: Membuat package `internal/infra/redis` dengan inisialisasi koneksi `go-redis/v9`, connection pool configuration, ping test on boot, dan graceful shutdown handler.
- **Target Files**: `internal/infra/redis/client.go`, `internal/infra/redis/config.go`.
- **Kriteria Penerimaan**:
  - Koneksi berhasil terhubung ke Redis saat boot backend.
  - Jika Redis offline saat boot, aplikasi memberikan pesan log yang jelas dan menerapkan graceful degradation.

### TASK-P02-003: Versioned Cache Middleware & Engine
- **Deskripsi**: Mengembangkan engine cache berversi di `internal/infra/redis/cache.go` dan HTTP middleware di `internal/middleware/cache.go`. Format key: `cache:u:{user_id}:v{ver}:{hash_request}`. Menyediakan helper `BumpUserVersion(ctx, userID)` untuk invalidasi instan.
- **Target Files**: `internal/infra/redis/cache.go`, `internal/middleware/cache.go`.
- **Kriteria Penerimaan**:
  - HTTP GET request yang dicache mengembalikan header `X-Cache: HIT`.
  - Memanggil `BumpUserVersion` menaikkan nomor versi di Redis; request berikutnya menghasilkan `X-Cache: MISS` dan memperbarui cache baru.

### TASK-P02-004: Redis Pub/Sub Broadcaster untuk Invalidasi Cache Global
- **Deskripsi**: Membuat subscriber background worker yang mendengarkan channel `cache:invalidate`. Node backend merespons pesan pub/sub untuk membersihkan in-memory lookup table lokal.
- **Target Files**: `internal/infra/redis/pubsub.go`.
- **Kriteria Penerimaan**:
  - Pesan invalidasi yang dipublish oleh satu node diterima oleh semua subscriber node dalam waktu < 50ms.

---

## 2. Audit Logging & Compliance (UU PDP)

### TASK-P02-005: Migrasi Database Tabel Append-Only audit_logs
- **Deskripsi**: Menulis skrip migrasi SQL `0005_create_audit_logs.sql` untuk membuat tabel `audit_logs` lengkap dengan index B-Tree pada `created_at`, `actor_id`, `entity_type`, dan trigger PostgreSQL untuk memblokir statement `UPDATE` dan `DELETE`.
- **Target Files**: `database/migrations/0005_create_audit_logs.sql`.
- **Kriteria Penerimaan**:
  - Migrasi sukses dieksekusi tanpa error.
  - Percobaan `UPDATE` atau `DELETE` pada tabel `audit_logs` melempar exception SQL: `table audit_logs is append-only`.

### TASK-P02-006: Modul internal/audit & Asynchronous Worker Queue
- **Deskripsi**: Membangun modul Go `internal/audit` yang menyediakan struktur `AuditEvent`, buffered channel worker pool, auto-masking field sensitif, dan batch insert ke PostgreSQL setiap 100 item atau flush tiap 500ms.
- **Target Files**: `internal/audit/audit.go`, `internal/audit/worker.go`, `internal/audit/masker.go`.
- **Kriteria Penerimaan**:
  - Pemanggilan `audit.Log(ctx, event)` return non-blocking dalam latensi < 1ms.
  - Data sensitif seperti `password` dan `token` otomatis diganti menjadi `[REDACTED]`.

---

## 3. Dynamic Feature Flags

### TASK-P02-007: Migrasi Database Tabel feature_flags
- **Deskripsi**: Menulis file migrasi `0006_create_feature_flags.sql` berisi tabel `feature_flags` (id, key, name, description, is_enabled, strategy, percentage, rules JSONB, created_at, updated_at).
- **Target Files**: `database/migrations/0006_create_feature_flags.sql`.
- **Kriteria Penerimaan**:
  - Tabel terbuat dengan unique constraint pada kolom `key`.

### TASK-P02-008: Modul internal/featureflag & Evaluator Engine
- **Deskripsi**: Mengembangkan modul evaluator runtime feature flag di Go dengan layer pembacaan Redis cache dan fallback PostgreSQL. Menyediakan fungsi `IsEnabled(ctx, flagKey, evalContext) bool`.
- **Target Files**: `internal/featureflag/evaluator.go`, `internal/featureflag/service.go`.
- **Kriteria Penerimaan**:
  - Evaluasi boolean flag berjalan dalam < 1ms dari cache Redis.
  - Evaluasi rollout percentage konsisten untuk user ID yang sama.

---

## 4. Frontend Generic DataTable & URL State

### TASK-P02-009: Kontrak API Handler Paginated Data Table di Go
- **Deskripsi**: Membuat helper parser pagination, sorting, dan filtering di Go (`pkg/pagination/pagination.go`) yang menerjemahkan parameter query string ke query PostgreSQL yang aman (mencegah SQL injection).
- **Target Files**: `pkg/pagination/pagination.go`, `pkg/pagination/envelope.go`.
- **Kriteria Penerimaan**:
  - Query otomatis membatasi `per_page` maksimal 100 dan default 20.
  - Kolom sort divalidasi terhadap whitelist kolom yang diizinkan.

### TASK-P02-010: Custom Hook useDataTableUrlState di Next.js
- **Deskripsi**: Membuat hook Next.js `use-data-table-url-state.ts` yang membaca `useSearchParams` dan `useRouter` untuk mengelola state tabel (page, per_page, sort, order, search, filters) secara reaktif.
- **Target Files**: `frontend/src/hooks/use-data-table-url-state.ts`.
- **Kriteria Penerimaan**:
  - Mengubah halaman atau sortir memperbarui URL query string browser via `router.replace` tanpa refresh penuh.
  - Reload browser menjaga state tabel persis sama dengan parameter URL.

### TASK-P02-011: Komponen Generic DataTable TanStack Table v8
- **Deskripsi**: Membangun komponen UI `<DataTable>` di frontend berbasis TanStack Table v8 yang mencakup skeleton loader, sortable header icons, dynamic columns, dan error boundary.
- **Target Files**: `frontend/src/components/data-table/data-table.tsx`, `frontend/src/components/data-table/data-table-column-header.tsx`.
- **Kriteria Penerimaan**:
  - Komponen menerima props data, columns, pagination meta, dan loading state.
  - Klik header kolom memicu pengurutan (asc -> desc -> default).

### TASK-P02-012: Komponen Toolbar & Pagination DataTable
- **Deskripsi**: Membangun komponen pendukung `<DataTableToolbar>` (dengan input pencarian ter-debounce 300ms) dan `<DataTablePagination>` (dengan selector 10/20/50/100 baris dan info total halaman).
- **Target Files**: `frontend/src/components/data-table/data-table-toolbar.tsx`, `frontend/src/components/data-table/data-table-pagination.tsx`.
- **Kriteria Penerimaan**:
  - Pengetikan pada search bar mengupdate query param `q` setelah jeda 300ms.
  - Pilihan per-page mengubah limit baris dan me-reset posisi page ke 1.

---

## 5. Observability Baseline

### TASK-P02-013: Instrumentasi Prometheus Metrics di Backend Go
- **Deskripsi**: Menambahkan middleware HTTP instrumentasi Prometheus di backend Go (`internal/middleware/metrics.go`) untuk merekam Four Golden Signals (Latency, Traffic, Errors, Saturation) serta endpoint internal `/metrics`.
- **Target Files**: `internal/middleware/metrics.go`, `cmd/server/main.go`.
- **Kriteria Penerimaan**:
  - Endpoint `/metrics` mengekspos metrik standar Prometheus dalam format teks eksposisi.
  - Latensi request terekam dalam histogram bucket terstandar.

### TASK-P02-014: Integrasi Verifikasi End-to-End Fondasi Platform
- **Deskripsi**: Menulis skenario test integrasi yang memverifikasi alur: (1) Redis cache hit/miss/bump version, (2) Audit log tersimpan asinkron dengan masking, (3) Evaluasi feature flag, dan (4) Ekstraksi pagination backend.
- **Target Files**: `tests/integration/platform_foundation_test.go`.
- **Kriteria Penerimaan**:
  - Seluruh test integrasi berjalan sukses di test runner Go.
