# Task Breakdown — P04: Subscription & Token Core

Dokumen ini mendefinisikan rincian pekerjaan teknis (*work breakdown structure*) untuk Fase P04 dalam format standar `TASK-P04-NNN`. Minimal 15 tugas terinci disediakan untuk menjamin eksekusi yang komprehensif.

---

### TASK-P04-001: Skema Migrasi Database 0008_subscription_token_core.sql
- **Komponen:** Database / Migrations
- **Tipe:** Engineering / DDL
- **Deskripsi:** Menulis skrip migrasi SQL PostgreSQL 17 yang mencakup pembuatan tabel `plans`, `plan_entitlements`, `plan_limits`, `subscriptions`, `subscription_events`, `token_periods`, `token_ledger`, dan `pricing_rules`. Seluruh kolom moneter dan kuota wajib menggunakan `BIGINT` atau `INTEGER`. Menyertakan skrip rollback (*down migration*) yang bersih.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Skrip SQL lolos eksekusi migrasi maju (*up*) dan mundur (*down*) tanpa galat sintaks.
  2. Tabel memiliki constraint unik, foreign key dengan klausa `ON DELETE RESTRICT`, dan indeks pada kolom yang sering dicari (`user_id`, `plan_id`, `idempotency_key`, `period_start/end`).
  3. Kolom uang bertipe `BIGINT` (Rupiah bulat) dan kuota token bertipe `BIGINT`.
- **Dependensi:** Fondasi Database P01/P02.

---

### TASK-P04-002: Seeder Data Default Plans dan Pricing Rules
- **Komponen:** Database / Seeding
- **Tipe:** Configuration / Data
- **Deskripsi:** Membuat skrip SQL seeder untuk mengisi paket bawaan: Free (Rp 0), Pro (Rp 99.000/bln, Rp 990.000/thn), dan Business (Rp 299.000/bln, Rp 2.990.000/thn), lengkap dengan entitlement boolean dan batasan limit numerik per paket. Menyediakan konfigurasi awal `pricing_rules` untuk modalitas chat, vision, voice, dan embedding.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Seeder tereksekusi secara idempoten (`ON CONFLICT DO NOTHING` atau `DO UPDATE`).
  2. Paket Free, Pro, dan Business terkonfigurasi dengan batasan kuota token per siklus yang tepat (Free: 10.000, Pro: 250.000, Business: 1.000.000).
  3. Entri `pricing_rules` memuat `base_per_1k_idr = 300` dengan multiplier modalitas default.
- **Dependensi:** TASK-P04-001.

---

### TASK-P04-003: Domain Entity dan Value Objects Modul Billing
- **Komponen:** Backend (`internal/billing/domain`)
- **Tipe:** Domain Modeling
- **Deskripsi:** Mendefinisikan struktur Go struct, type aliases, enum, dan domain errors untuk modul billing. Meliputi entitas `Plan`, `PlanEntitlement`, `PlanLimit`, `Subscription`, `SubscriptionStatus`, `SubscriptionEvent`, `TokenPeriod`, `TokenLedgerEntry`, `LedgerEntryType`, dan `PricingRule`.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Seluruh enum status langganan terdefinisi (`trialing`, `active`, `past_due`, `grace`, `canceled`, `expired`).
  2. Enum mutasi ledger terdefinisi (`reserve`, `commit`, `release`, `grant`, `adjust`).
  3. Struct moneter menggunakan tipe `int64` tanpa ada tipe `float32`/`float64` pada nominal uang.
  4. Domain error spesifik terdeklarasi (misal: `ErrInsufficientQuota`, `ErrSubscriptionExpired`, `ErrDuplicateIdempotencyKey`).
- **Dependensi:** TASK-P04-001.

---

### TASK-P04-004: PostgreSQL Repository Layer Modul Billing
- **Komponen:** Backend (`internal/billing/repository`)
- **Tipe:** Database Access
- **Deskripsi:** Mengimplementasikan interface repository database PostgreSQL untuk billing dan token ledger. Mengimplementasikan method: `GetActiveSubscriptionByUserID`, `GetTokenPeriodForUpdate`, `UpsertTokenPeriod`, `InsertLedgerEntry`, dan `RecordSubscriptionEvent`.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. `GetTokenPeriodForUpdate` menggunakan kueri SQL dengan klausa `SELECT ... FOR UPDATE` dalam transaksi untuk menjamin isolasi konkuren.
  2. Penanganan duplikasi `idempotency_key` pada `InsertLedgerEntry` mengembalikan error terkontrol atau entri yang sudah ada.
  3. Seluruh kueri menggunakan parameterized queries untuk mencegah SQL injection.
- **Dependensi:** TASK-P04-003.

---

### TASK-P04-005: Token Ledger Service — Alur Reserve
- **Komponen:** Backend (`internal/billing/service`)
- **Tipe:** Business Logic
- **Deskripsi:** Mengembangkan service `ReserveToken` yang bertugas memvalidasi kuota token pengguna untuk hari berjalan (TZ Asia/Jakarta), mengunci kuota yang diminta ke dalam kolom `reserved_tokens`, dan mencatat entri ledger bertipe `reserve` dengan `idempotency_key` unik.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Menolak permintaan dengan error `ErrInsufficientQuota` jika `used_tokens + reserved_tokens + requested > quota_tokens`.
  2. Mengembalikan `reservation_id` unik jika kuota mencukupi.
  3. Bersifat aman secara konkuren saat diuji dengan banyak permintaan simultan.
- **Dependensi:** TASK-P04-004.

---

### TASK-P04-006: Token Ledger Service — Alur Commit dan Release
- **Komponen:** Backend (`internal/billing/service`)
- **Tipe:** Business Logic
- **Deskripsi:** Mengembangkan service `CommitToken` dan `ReleaseToken`. `CommitToken` mencatat jumlah token aktual yang dipakai AI Gateway, mengurangi `reserved_tokens`, menambah `used_tokens`, dan mencatat entri ledger `commit`. `ReleaseToken` mengembalikan kuota yang terkunci ke status bebas saat pemanggilan AI gagal atau dibatalkan.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. `CommitToken` menghitung selisih token reservasi dan aktual dengan presisi tanpa membuat saldo minus.
  2. `ReleaseToken` mengembalikan seluruh token yang di-reserve tanpa menambah `used_tokens`.
  3. Masing-masing fungsi mencatat riwayat mutasi ke tabel `token_ledger` append-only.
- **Dependensi:** TASK-P04-005.

---

### TASK-P04-007: Background Cleaner Worker untuk Uncommitted Reservasi
- **Komponen:** Backend (`internal/billing/worker`)
- **Tipe:** Background Job
- **Deskripsi:** Mengembangkan background worker yang berjalan secara berkala (interval 15 menit) untuk memindai transaksi reservasi kuota yang menggantung (*uncommitted reservations*) dengan usia lebih dari 5 menit akibat crash jaringan atau restart instance, lalu mengeksekusi `ReleaseToken` otomatis.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Worker menemukan seluruh reservasi pending kedaluwarsa secara berkala.
  2. Melepaskan `reserved_tokens` dan mencatat entri ledger dengan catatan sistem `auto_release_timeout`.
  3. Beban query DB ringan dan menggunakan batching (maksimal 100 reservasi per putaran).
- **Dependensi:** TASK-P04-006.

---

### TASK-P04-008: Dynamic Pricing Engine Service
- **Komponen:** Backend (`internal/billing/service`)
- **Tipe:** Calculation Engine
- **Deskripsi:** Mengembangkan service kalkulasi harga token dinamis (`CalculateTokenCostIDR`). Menghitung biaya pemakaian token dalam Rupiah bulat berdasarkan model dan modalitas (chat, vision, voice, embedding, reasoning), harga dasar per 1.000 token, faktor pengali markup, persentase markup, dan bobot modalitas dari tabel `pricing_rules`.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Hasil perhitungan selalu berupa bilangan bulat (`int64`), dibulatkan ke atas (*ceil*) pada batas pecahan terkecil.
  2. Mendukung konfigurasi modalitas dinamis melalui JSONB `modality_multipliers`.
  3. Unit test memverifikasi presisi kalkulasi dengan 20 skenario kombinasi modalitas dan token count.
- **Dependensi:** TASK-P04-004.

---

### TASK-P04-009: Domain Model dan Repository Modul Entitlement
- **Komponen:** Backend (`internal/entitlement/domain` & `repository`)
- **Tipe:** Domain & Data Access
- **Deskripsi:** Membangun domain entities, contract interfaces, dan repository PostgreSQL untuk modul entitlement. Mengambil konfigurasi hak akses boolean (`plan_entitlements`) dan batasan limit numerik (`plan_limits`) berdasarkan paket langganan aktif pengguna.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Repository menyediakan fungsi `GetPlanEntitlementsByPlanID` dan `GetPlanLimitsByPlanID`.
  2. Menyediakan fungsi agregasi `ResolveUserEntitlements(userID)` yang menggabungkan status subscription dengan fitur dan kuota pengguna.
  3. Penanganan paket default `free` jika user belum memiliki data langganan khusus.
- **Dependensi:** TASK-P04-001, TASK-P04-003.

---

### TASK-P04-010: Versioned Entitlement Cache Manager (L1/L2)
- **Komponen:** Backend (`internal/entitlement/cache`)
- **Tipe:** Caching Engine
- **Deskripsi:** Mengembangkan sistem caching multi-level: L1 in-memory LRU cache (TTL 60 detik) dan L2 Redis (TTL 24 jam). Cache disimpan dengan pola key `entitlement:u:{userID}:v:{version}` di mana versi dibaca dari counter integer Redis `entitlement:ver:{userID}`.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Cache hit pada L1 merespons dalam < 0.5 ms; cache hit pada L2 (Redis) < 2 ms.
  2. Mengimplementasikan fungsi `InvalidateUser(userID)` yang mengeksekusi `INCR` pada version counter dan menghapus entri L1 lokal.
  3. Mendukung fallback pembacaan langsung ke database PostgreSQL jika Redis tidak tersedia.
- **Dependensi:** TASK-P04-009, Fondasi Redis P02.

---

### TASK-P04-011: Redis Pub/Sub Synchronization untuk Invalidasi Multi-Instance
- **Komponen:** Backend (`internal/entitlement/cache`)
- **Tipe:** Realtime Synchronization
- **Deskripsi:** Mengintegrasikan Redis Pub/Sub subscriber pada backend untuk mendengarkan channel `entitlement:invalidations`. Ketika ada instance yang memicu invalidasi hak akses user (akibat upgrade, downgrade, atau admin action), pesan disiarkan ke seluruh node backend untuk membersihkan L1 cache lokal masing-masing.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Pesan siaran invalidasi diterima oleh semua node backend dalam waktu < 50ms.
  2. L1 in-memory cache pada seluruh node bersih seketika setelah menerima pesan.
  3. Koneksi listener melakukan auto-reconnect jika koneksi Redis terputus.
- **Dependensi:** TASK-P04-010.

---

### TASK-P04-012: Gin HTTP Middleware Penegakan Hak Akses (RequireFeature & WithinLimit)
- **Komponen:** Backend (`internal/entitlement/middleware`)
- **Tipe:** HTTP Middleware
- **Deskripsi:** Membangun middleware Gin HTTP:
  - `RequireFeature(featureKey string)`: Memeriksa apakah fitur boolean diaktifkan untuk user. Mengembalikan status HTTP `403 Forbidden` jika dinonaktifkan.
  - `WithinLimit(limitKey string, usageProvider func(c *gin.Context) (int64, error))`: Memeriksa apakah penggunaan entitas masih di bawah limit kuota. Mengembalikan HTTP `429 Too Many Requests` jika limit terlampaui.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Menggunakan data entitlement dari versioned cache.
  2. Mengembalikan response JSON standar yang memuat detail alasan penolakan dan link upgrade.
  3. Melewatkan request (*bypass*) jika pengguna memiliki role `super_admin`.
- **Dependensi:** TASK-P04-010.

---

### TASK-P04-013: Handler REST API Endpoint /api/v1/me/entitlements
- **Komponen:** Backend (`internal/billing/handler`)
- **Tipe:** API Endpoint
- **Deskripsi:** Membangun controller dan rute HTTP `GET /api/v1/me/entitlements` yang mengembalikan rangkuman lengkap status paket, daftar fitur boolean yang aktif, dan kuota limit beserta jumlah penggunaan berjalan untuk konsumsi TanStack Query pada frontend.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Format JSON respons selaras dengan spesifikasi teknis (mencakup info plan, features map, limits map, dan version).
  2. Respons dilayani dengan latensi rata-rata < 5ms dari cache.
  3. Endpoint terlindungi oleh middleware autentikasi pengguna.
- **Dependensi:** TASK-P04-009, TASK-P04-010.

---

### TASK-P04-014: Subscription Lifecycle Service & State Transitions
- **Komponen:** Backend (`internal/billing/service`)
- **Tipe:** Business Logic
- **Deskripsi:** Mengembangkan service manajemen siklus hidup langganan yang menangani transisi state (`trialing`, `active`, `past_due`, `grace`, `canceled`, `expired`). Setiap pergantian state secara otomatis mencatat riwayat ke tabel `subscription_events` lengkap dengan payload `before` dan `after`.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Mencegah transisi state yang tidak valid (misal: `expired` langsung ke `past_due`).
  2. Setiap mutasi status memicu pencatatan audit log `subscription_events`.
  3. Setiap mutasi status memicu invalidasi cache entitlement user secara otomatis.
- **Dependensi:** TASK-P04-004, TASK-P04-010.

---

### TASK-P04-015: Logika Upgrade Prorata Instan dan Downgrade Soft-Limit
- **Komponen:** Backend (`internal/billing/service`)
- **Tipe:** Business Logic
- **Deskripsi:** Mengimplementasikan fungsi bisnis `CalculateUpgradeProration` dan `ScheduleDowngrade`. Menghitung kredit prorata berbasis hari tersisa (dibulatkan ke bawah ke integer IDR). Menjadwalkan downgrade di akhir masa siklus langganan aktif tanpa menghapus data entitas historis user (*soft-limit policy*).
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Perhitungan prorata akurat terhadap jumlah hari berjalan dalam siklus langganan.
  2. Downgrade menandai kolom `cancel_at_period_end = true` dan menyimpan rencana paket baru tanpa langsung mencabut hak akses saat ini.
  3. Penegakan soft-limit diuji: pengguna paket Free dengan 5 dompet lama tetap bisa melihat 5 dompet, namun gagal membuat dompet ke-6.
- **Dependensi:** TASK-P04-014.

---

### TASK-P04-016: Concurrency Testing dan Verifikasi Anti Double-Spending
- **Komponen:** Testing / QA
- **Tipe:** Test Engineering
- **Deskripsi:** Menulis unit dan integration test otomatis konkurensi menggunakan goroutines (`go test -race`). Mensimulasikan 50–100 request paralel yang mencoba melakukan reservasi token pada sisa kuota yang terbatas, serta memverifikasi tidak adanya sisa saldo negatif atau double-spending token.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Uji konkurensi berhasil dieksekusi tanpa deteksi data race (`go test -race`).
  2. Total token yang di-commit dan di-release persis seimbang dengan mutasi ledger.
  3. Saldo sisa kuota tidak pernah menembus angka negatif.
- **Dependensi:** TASK-P04-005, TASK-P04-006.

---

### TASK-P04-017: Metrik Prometheus, Observabilitas & Kontrak Antarmuka P05/P06
- **Komponen:** Observability / Integration
- **Tipe:** Telemetry & Handoff
- **Deskripsi:** Menambahkan metrik Prometheus untuk pelacakan reservasi token, commit, hit/miss rate cache entitlement, dan durasi evaluasi middleware. Menyusun dokumentasi kontrak antarmuka service Go untuk dikonsumsi oleh modul pembayaran P05 dan Super Admin dashboard P06.
- **Kriteria Penerimaan (Acceptance Criteria):**
  1. Metrik teregistrasi pada Prometheus registry aplikasi.
  2. Kontrak interface Go (`BillingService`, `EntitlementService`, `TokenLedgerManager`) terdokumentasi rapi dan siap diinjeksi (*dependency injection*).
  3. Seluruh checklist Definition of Done terpenuhi.
- **Dependensi:** TASK-P04-008, TASK-P04-012, TASK-P04-015.
