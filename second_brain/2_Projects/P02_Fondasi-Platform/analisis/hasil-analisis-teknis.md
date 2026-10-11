# Hasil Analisis Teknis — P02 Fondasi Platform

Dokumen ini mendokumentasikan keputusan teknis arsitektur untuk Fase P02. Setiap keputusan mengikat dan diturunkan langsung dari Keputusan Terkunci Proyek (K-01 s/d K-30).

---

## Daftar Keputusan Teknis (K-P02-01 s/d K-P02-14)

### K-P02-01: Engine dan Driver Redis
- **Keputusan**: Menggunakan Redis 7-Alpine via official Go driver `github.com/redis/go-redis/v9`.
- **Dasar Pemikiran**: Mendukung Connection Pooling teruji, context-cancellation native, Redis Sentinel/Cluster ready, pipeline command, serta built-in Pub/Sub handler.
- **Rujukan**: Selaras K-21 (Infra Redis cache bersama + pub/sub).

### K-P02-02: Isolasi Cache Berversi Tanpa Scan Key
- **Keputusan**: Pola key Redis menggunakan format `cache:u:{user_id}:v{version}:{resource_scope}`. Versi user disimpan di Redis string `cache:u:{user_id}:ver`.
- **Mekanisme Invalidasi**: Invalidasi data user dilakukan dengan menjalankan atomic increment (`INCR cache:u:{user_id}:ver`).
- **Dasar Pemikiran**: Operasi `KEYS` atau `SCAN` dilarang keras di production karena memblokir event loop Redis. Dengan memajukan nomor versi (atomic integer), seluruh cache lama otomatis invalid (miss) dan akan dibersihkan oleh Redis TTL (default TTL 1 jam pada cache entry). Beban O(1).
- **Rujukan**: Selaras K-14 (Realtime RBAC lookup per-request + cache berversi).

### K-P02-03: Sinkronisasi Invalidasi Lintas Instance via Pub/Sub
- **Keputusan**: Redis Pub/Sub pada channel `cache:invalidate` digunakan untuk memberitahu seluruh node backend jika ada entitas global (misal: plan, route AI, feature flag) yang berubah.
- **Payload Format**: JSON string `{"scope": "global|user", "id": "<id>", "version": <int>, "ts": <epoch_ms>}`.
- **Dasar Pemikiran**: Node backend yang memegang in-memory L1 cache (jika ada) langsung menghapus entri lokal saat menerima pesan pub/sub tanpa menunggu polling.
- **Rujukan**: K-21, K-24 (Zero downtime & konsistensi multi-instance).

### K-P02-04: Tabel Audit Log Append-Only dengan Immutability Constraint
- **Keputusan**: Tabel `audit_logs` dibuat di PostgreSQL dengan trigger level database yang me-reject operasi `UPDATE` dan `DELETE`.
- **Struktur Kolom**: `id` (UUIDv7/BIGSERIAL), `actor_id` (UUID nullable untuk sistem), `actor_role` (VARCHAR), `action` (VARCHAR), `entity_type` (VARCHAR), `entity_id` (VARCHAR), `before_state` (JSONB nullable), `after_state` (JSONB nullable), `ip_address` (VARCHAR), `user_agent` (VARCHAR), `created_at` (TIMESTAMPTZ default NOW()).
- **Dasar Pemikiran**: Memenuhi standar regulasi UU PDP No. 27/2022 dan K-29 (Non-repudiation audit trail). Mencegah pengubahan riwayat transaksi bahkan oleh super admin langsung di level database.
- **Rujukan**: Selaras K-29 & K-30.

### K-P02-05: Eksekusi Audit Asinkron Non-Blocking via Buffer Channel Go
- **Keputusan**: Modul `internal/audit` menyediakan buffer Go channel (kapasitas 2.048 event) dengan pool worker background (default 3 worker) yang melakukan batch insert ke database.
- **Fallback Strategi**: Jika channel penuh, audit dijalankan sync atau di-spill ke emergency logger (stderr/file) tanpa memutus alur bisnis utama.
- **Dasar Pemikiran**: Write path audit tidak boleh memperlambat HTTP request user (overhead latensi API < 1ms).
- **Rujukan**: K-29.

### K-P02-06: Evaluasi Feature Flag Berbasis Database dan Cache Redis L2
- **Keputusan**: Master definisi feature flag disimpan di PostgreSQL tabel `feature_flags`. Evaluasi runtime dibaca dari Redis hash `ff:rules`.
- **Strategi Rollout**: Mendukung 4 mode: `BOOLEAN` (on/off global), `PERCENTAGE` (konsisten hashing berdasarkan `user_id % 100`), `USER_LIST` (whitelist user UUID), dan `ROLE` (hanya super_admin atau user).
- **Dasar Pemikiran**: Memungkinkan canary deployment, safe rollback, dan kontrol fitur bertahap tanpa restart backend.
- **Rujukan**: K-23 (Zero downtime via feature flag).

### K-P02-07: Fallback Fail-Closed Feature Flags
- **Keputusan**: Jika koneksi Redis down dan PostgreSQL timeout, feature flag evaluator harus default ke `false` (fail-closed/disabled) untuk fitur baru atau berisiko, dan hardcoded fallback untuk core services.
- **Dasar Pemikiran**: Mengurangi blast radius kegagalan infrastruktur cache.
- **Rujukan**: K-24.

### K-P02-08: Kontrak Standar Response API untuk Paginated Data Table
- **Keputusan**: Seluruh endpoint backend yang menyuplai data table wajib mengembalikan struktur envelope:
  ```json
  {
    "success": true,
    "data": [...],
    "meta": {
      "page": 1,
      "per_page": 20,
      "total_items": 142,
      "total_pages": 8,
      "sort_by": "created_at",
      "sort_order": "desc"
    }
  }
  ```
- **Dasar Pemikiran**: Standarisasi kontrak frontend-backend memudahkan satu komponen `<DataTable>` menangani semua resource (users, logs, transactions, subscriptions, flags).
- **Rujukan**: Selaras K-22.

### K-P02-09: Pagination Server-Side Standar 20 Baris
- **Keputusan**: Default pagination adalah 20 baris per halaman. Opsi yang diizinkan untuk klien: 10, 20, 50, 100. Nilai di luar opsi ini dipaksa (clamped) ke 20 oleh backend.
- **Dasar Pemikiran**: Mengoptimalkan ukuran payload JSON, performa render browser TanStack Table, dan query index PostgreSQL.
- **Rujukan**: K-22.

### K-P02-10: Sinkronisasi Status URL 2-Arah (URL-First State)
- **Keputusan**: State tabel (page, per_page, sort, order, q, filter) disimpan langsung di URL query string Next.js. Komponen UI membaca URL sebagai Single Source of Truth.
- **Naming Convention**:
  - `page`: integer (1-indexed)
  - `per_page`: integer (10|20|50|100)
  - `sort`: nama kolom field (snake_case)
  - `order`: `asc` | `desc`
  - `q`: string pencarian bebas
  - `filter_<col>`: nilai filter per kolom
- **Dasar Pemikiran**: User dapat melakukan bookmark link, copy-paste URL ke tab lain atau tim lain, dan tombol back/forward browser bekerja normal tanpa kehilangan filter.
- **Rujukan**: K-22.

### K-P02-11: Komponen Generic `<DataTable>` Berbasis TanStack Table v8
- **Keputusan**: Komponen dibuat headless menggunakan TanStack Table v8 dengan wrapper komponen UI Tailwind CSS / Shadcn UI.
- **Komposisi Komponen**:
  1. `<DataTableToolbar>`: Input search ber-debounce 300ms + Dropdown facet filter + Column view toggle.
  2. `<DataTableContent>`: Skeleton loader native saat fetching, sortable header, sticky action column.
  3. `<DataTablePagination>`: Info total item, dropdown ukuran halaman, tombol navigasi halaman.
- **Dasar Pemikiran**: Mengeliminasi duplikasi kode tabel di dashboard user dan super-admin.
- **Rujukan**: K-22.

### K-P02-12: Zero Downtime Database Migrations (Expand/Contract)
- **Keputusan**: Semua migrasi schema di fase ini dan ke depan menerapkan pola Expand/Contract.
  - Tahap Expand: Tambah tabel/kolom nullable atau dengan default tanpa lock eksklusif panjang.
  - Tahap Contract: Penghapusan atau pengetatan constraint hanya dilakukan setelah seluruh node aplikasi baru live.
- **Aturan DDL**: Dilarang menggunakan `ALTER TABLE ... ADD COLUMN NOT NULL` tanpa default, dilarang rename kolom aktif.
- **Rujukan**: K-23.

### K-P02-13: Observability Baseline (Four Golden Signals & Prometheus Metrics)
- **Keputusan**: Tambah middleware Prometheus di Go (`github.com/prometheus/client_golang`) yang mengekspos endpoint internal `/metrics` (hanya dapat diakses jaringan private / internal host).
- **Metrik Dasar**:
  - `http_requests_total{method, path, status}`
  - `http_request_duration_seconds{method, path}` (Histogram)
  - `redis_operations_total{command, status}`
  - `audit_queue_depth` (Gauge)
  - `feature_flag_evaluations_total{flag, result}`
- **Dasar Pemikiran**: Menjadi dasar pemantauan operasional sebelum fase token (P03/P04) dan payment (P05) berjalan.
- **Rujukan**: K-24 & Best Practice SRE.

### K-P02-14: Penanganan Data Sensitif dan UU PDP pada Payload Audit
- **Keputusan**: JSON serializer modul audit wajib menjalankan data masking (sanitasi) pada kolom sensitif sebelum disimpan ke database: password, salt, bearer token, credit card number, dan NIK diganti dengan `[REDACTED]`.
- **Dasar Pemikiran**: Memenuhi kewajiban pasal perlindungan kerahasiaan data pribadi UU PDP No. 27/2022.
- **Rujukan**: K-30.
