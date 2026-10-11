# Implementation Plan — P04 Subscription & Token Core

| Field | Isi |
|---|---|
| **Fase** | P04 — Subscription & Token Core |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | P01 (Monorepo Go + PostgreSQL + Next.js), P02 (Redis cache berversi + pub/sub + audit log), P03 (AI Gateway contracts) |
| **Master plan** | `../../IMPLEMENTATION-PLAN.md` |
| **Status** | Siap dieksekusi — belum ada kode ditulis |

> **Koreksi path & arsitektur:**
> - Frontend: `frontend/` (Next.js 16 App Router, bukan `frontend/src/`).
> - Backend: `backend/internal/billing/`, `backend/internal/entitlement/`.
> - Migrasi Database Kanonik: `0008_subscription_token_core.sql` (menggantikan penomoran lain).
> - Nilai Moneter: integer Rupiah bulat (`BIGINT` / `int64`), zero-floating-point.
> - Mutasi Ledger: 2-phase transaksional (reserve -> commit/release), append-only, idempoten.

---

## 1. Tujuan & Definisi Selesai

Menyediakan mesin **Subscription & Token Core** yang kuat, aman secara finansial, dan berkinerja tinggi untuk FinTrack v2:
1. **Langganan & Siklus:** Siklus harian berbasis kalender Jakarta (`Asia/Jakarta`), upgrade prorata instan integer IDR, dan downgrade soft-limit di akhir siklus.
2. **Token Ledger:** Model 2-fase reserve-commit/release berbasis row-level locking (`SELECT ... FOR UPDATE`), append-only, idempoten, anti double-spending.
3. **Dynamic Pricing Engine:** Perhitungan konsumsi token dalam integer Rupiah dengan pengali modalitas dan markup rules.
4. **Entitlement Engine:** Penegakan hak akses fitur boolean dan batas limit numerik real-time melalui L1 in-memory + L2 Redis versioned cache dan Gin HTTP middleware.

**Definition of Done Fase:**
- [ ] Seluruh 17 task (`TASK-P04-001` s.d. `TASK-P04-017`) terimplementasi tanpa placeholder/stub/TBD.
- [ ] Skrip migrasi kanonik `database/migrations/0008_subscription_token_core.sql` lolos uji up/down; dump schema `database/dump/` di-regenerate.
- [ ] Kompilasi backend bersih: `go vet ./...`, `go build ./...`, dan `go test ./...` lulus 100%.
- [ ] Concurrency testing anti double-spend (`go test -race`) lulus tanpa race condition maupun saldo token minus.
- [ ] Endpoint `/api/v1/me/entitlements` melayani respons dari cache dengan latensi p95 < 5ms.
- [ ] Cache invalidation teruji sinkron lintas instance via Redis Pub/Sub (`entitlement:invalidations`).
- [ ] Tidak ada regresi fungsional P01, P02, maupun P03.

---

## 2. Urutan Eksekusi (Wave)

| Wave | Task | Fokus Pekerjaan | Prasyarat |
|---|---|---|---|
| **W1** | TASK-P04-001, TASK-P04-002, TASK-P04-003, TASK-P04-004 | Skema DB 0008, Seeder Paket/Pricing, Domain Entity, Repository Postgres | P01, P02 |
| **W2** | TASK-P04-005, TASK-P04-006, TASK-P04-007 | Token Ledger 2-Phase: Reserve, Commit, Release, Background Cleaner | W1 |
| **W3** | TASK-P04-008, TASK-P04-009, TASK-P04-010, TASK-P04-011, TASK-P04-012, TASK-P04-013 | Dynamic Pricing IDR, Entitlement Domain/Repo, L1/L2 Versioned Cache, Pub/Sub, Middleware, Endpoint `/me/entitlements` | W1, W2 |
| **W4** | TASK-P04-014, TASK-P04-015 | Subscription Lifecycle State Machine, Upgrade Prorata Instan, Downgrade Soft-Limit | W2, W3 |
| **W5** | TASK-P04-016, TASK-P04-017 | Concurrency Testing Anti Double-Spend (`-race`), Prometheus Metrics, Kontrak Antarmuka Handoff P05/P06 | W1 s.d. W4 |

---

## 3. Detail Task

### Wave 1 — Database Foundation, Seeder & Core Repositories

#### TASK-P04-001 — Skema Migrasi Database 0008_subscription_token_core.sql
- **Target File:** `database/migrations/0008_subscription_token_core.sql`
- **Langkah:**
  1. Buat tabel `plans` (`id`, `code`, `name`, `price_monthly`, `price_annually`, `currency`, `daily_token_quota`, `is_active`, `created_at`, `updated_at`). Kolom harga dan token menggunakan `BIGINT`.
  2. Buat tabel `plan_entitlements` (`id`, `plan_id`, `feature_key`, `is_enabled`, `created_at`). Unique index pada `(plan_id, feature_key)`.
  3. Buat tabel `plan_limits` (`id`, `plan_id`, `limit_key`, `max_value`, `created_at`). Unique index pada `(plan_id, limit_key)`.
  4. Buat tabel `subscriptions` (`id`, `user_id`, `plan_id`, `status`, `billing_cycle`, `starts_at`, `current_period_start`, `current_period_end`, `cancel_at_period_end`, `canceled_at`, `created_at`, `updated_at`).
  5. Buat tabel `subscription_events` (`id`, `subscription_id`, `event_type`, `from_status`, `to_status`, `payload`, `created_at`).
  6. Buat tabel `token_periods` (`id`, `user_id`, `period_date`, `daily_quota`, `used_tokens`, `reserved_tokens`, `created_at`, `updated_at`). Unique index `(user_id, period_date)` dan check constraint `used_tokens >= 0 AND reserved_tokens >= 0`.
  7. Buat tabel `token_ledger` (`id`, `user_id`, `period_id`, `reservation_id`, `entry_type`, `amount`, `idempotency_key`, `model_name`, `metadata`, `created_at`). Unique index pada `idempotency_key`.
  8. Buat tabel `pricing_rules` (`id`, `model_id`, `base_per_1k_idr`, `markup_percent`, `modality_multipliers`, `is_active`, `created_at`, `updated_at`).
  9. Sertakan blok rollback SQL `-- +migrate Down` yang menghapus seluruh tabel di atas secara teratur dengan `DROP TABLE IF EXISTS ... CASCADE`.
- **Verifikasi (Bukti):** Eksekusi migrasi forward dan reverse via CLI migration runner; validasi `pg_dump` cocok dengan `database/dump/`.
- **Catatan Zero-Downtime & Keamanan:** Operasi DDL aditif. Tidak memodifikasi tabel transaksi P01 yang sedang beroperasi. Menggunakan `BIGINT` untuk mencegah integer overflow.

#### TASK-P04-002 — Seeder Data Default Plans dan Pricing Rules
- **Target File:** `database/seeds/0002_subscription_token_seeds.sql` (atau disisipkan idempoten dalam migrasi/seeder Go)
- **Langkah:**
  1. Insert data master paket:
     - Free: `code='free'`, `price_monthly=0`, `daily_token_quota=10000`.
     - Pro: `code='pro'`, `price_monthly=99000`, `price_annually=990000`, `daily_token_quota=250000`.
     - Business: `code='business'`, `price_monthly=299000`, `price_annually=2990000`, `daily_token_quota=1000000`.
  2. Insert entitlement boolean per plan (Free vs Pro vs Business) untuk fitur `ai_chat`, `smart_categorization`, `ocr_receipt`, `csv_export`, `advanced_analytics`, `multi_wallet`.
  3. Insert batasan limit numerik: `max_wallets` (Free: 2, Pro: 10, Business: -1), `max_budgets` (Free: 3, Pro: 20, Business: -1), `monthly_ocr_scans` (Free: 5, Pro: 100, Business: 500).
  4. Insert default `pricing_rules`: `base_per_1k_idr=300`, `markup_percent=10`, `modality_multipliers='{"chat": 1.0, "vision": 2.5, "voice": 3.0, "embedding": 0.5, "reasoning": 1.5}'`.
  5. Semua query menggunakan klausa `ON CONFLICT (code) DO NOTHING` atau `ON CONFLICT (model_id) DO NOTHING`.
- **Verifikasi (Bukti):** Data terisi di DB dan pemanggilan seeder 2x berturut-turut menghasilkan output idempoten tanpa duplikasi baris.

#### TASK-P04-003 — Domain Entity dan Value Objects Modul Billing
- **Target File:** `backend/internal/billing/domain/plan.go`, `subscription.go`, `token.go`, `pricing.go`, `errors.go`
- **Langkah:**
  1. Buat tipe data enum status langganan: `SubscriptionStatus` (`trialing`, `active`, `past_due`, `grace`, `canceled`, `expired`).
  2. Buat tipe data enum mutasi ledger: `LedgerEntryType` (`reserve`, `commit`, `release`, `grant`, `adjust`).
  3. Definisikan struct entitas `Plan`, `PlanEntitlement`, `PlanLimit`, `Subscription`, `SubscriptionEvent`, `TokenPeriod`, `TokenLedgerEntry`, dan `PricingRule`. Seluruh nominal uang dan token bertipe `int64`.
  4. Deklarasikan domain error terisolasi: `ErrInsufficientQuota`, `ErrSubscriptionExpired`, `ErrDuplicateIdempotencyKey`, `ErrInvalidStateTransition`, `ErrReservationNotFound`, `ErrReservationAlreadyResolved`.
- **Verifikasi (Bukti):** `go vet ./backend/internal/billing/domain/...` bersih; unit test memverifikasi kelengkapan validasi enum dan error matching.

#### TASK-P04-004 — PostgreSQL Repository Layer Modul Billing
- **Target File:** `backend/internal/billing/repository/postgres_plan.go`, `postgres_subscription.go`, `postgres_token.go`
- **Langkah:**
  1. Implementasikan `PlanRepository`: query `GetPlanByCode`, `GetPlanByID`, `ListActivePlans`.
  2. Implementasikan `SubscriptionRepository`: `GetActiveSubscriptionByUserID`, `CreateSubscription`, `UpdateSubscriptionStatus`, `RecordSubscriptionEvent`.
  3. Implementasikan `TokenRepository`:
     - `GetOrCreateCurrentPeriod(ctx, tx, userID, date, defaultQuota)`
     - `GetTokenPeriodForUpdate(ctx, tx, userID, date)` dengan klausa `SELECT ... FOR UPDATE` dalam transaksi SQL.
     - `UpdatePeriodTokens(ctx, tx, periodID, usedDelta, reservedDelta)`
     - `InsertLedgerEntry(ctx, tx, entry)` dengan penanganan benturan `idempotency_key`.
- **Verifikasi (Bukti):** Integration test DB mock/real: transaksi rollback saat error, isolasi row lock diverifikasi pada dua transaksi paralel.

---

### Wave 2 — Token Ledger 2-Phase Engine & Auto-Release Worker

#### TASK-P04-005 — Token Ledger Service — Alur Reserve
- **Target File:** `backend/internal/billing/service/token_ledger.go`
- **Langkah:**
  1. Implementasikan `ReserveToken(ctx, input ReserveTokenInput) (*ReservationResult, error)`.
  2. Tentukan tanggal aktif Asia/Jakarta (`time.Now().In(jakartaLocation).Format("2006-01-02")`).
  3. Buka transaksi database:
     - Dapatkan baris periode aktif dengan `SELECT ... FOR UPDATE`.
     - Cek idempotensi: bila `idempotency_key` sudah pernah dicatat dengan tipe `reserve`, kembalikan detail reservasi eksisting.
     - Hitung: jika `used_tokens + reserved_tokens + requested_tokens > daily_quota`, gagalkan transaksi dengan `ErrInsufficientQuota`.
     - Update: `reserved_tokens = reserved_tokens + requested_tokens`.
     - Insert entri mutasi ke `token_ledger` (`entry_type = 'reserve'`).
  4. Commit transaksi DB dan kembalikan `reservation_id` (UUIDv7/ULID/UUID) ke caller AI Gateway.
- **Verifikasi (Bukti):** Unit & integration test: kuota berkurang pada kolom `reserved_tokens`, idempotency key ganda mengembalikan respons yang sama tanpa duplikasi pemotongan.

#### TASK-P04-006 — Token Ledger Service — Alur Commit dan Release
- **Target File:** `backend/internal/billing/service/token_ledger.go`
- **Langkah:**
  1. Implementasikan `CommitToken(ctx, input CommitTokenInput) error`:
     - Buka transaksi DB, lock `token_periods` terkait.
     - Validasi reservasi aktif: periksa entri `reserve` awal.
     - Hitung delta: `reserved_tokens = reserved_tokens - reserved_amount`, `used_tokens = used_tokens + actual_amount`.
     - Pastikan `reserved_tokens >= 0` dan `used_tokens >= 0`.
     - Insert entri ke `token_ledger` (`entry_type = 'commit'`, catatan token riil, durasi eksekusi, metadata model).
  2. Implementasikan `ReleaseToken(ctx, input ReleaseTokenInput) error`:
     - Buka transaksi DB, lock baris periode.
     - Validasi reservasi belum pernah di-commit atau di-release.
     - Kurangi `reserved_tokens = reserved_tokens - reserved_amount` tanpa mengubah `used_tokens`.
     - Insert entri ke `token_ledger` (`entry_type = 'release'`, catatan alasan kegagalan/timeout).
- **Verifikasi (Bukti):** Test skenario parsial: reservasi 4.000 token, komit riil 1.250 token -> `reserved_tokens` kembali 0, `used_tokens` bertambah 1.250.

#### TASK-P04-007 — Background Cleaner Worker untuk Uncommitted Reservasi
- **Target File:** `backend/internal/billing/worker/ledger_cleaner.go`
- **Langkah:**
  1. Buat worker berkala (menjalankan `ticker` setiap 15 menit atau konfigurabel).
  2. Kueri reservasi menggantung: entri `token_ledger` bertipe `reserve` yang dibuat > 5 menit lalu tanpa pasangan `commit` atau `release`.
  3. Ambil batch maksimal 100 reservasi per siklus untuk mencegah spikes beban DB.
  4. Untuk setiap reservasi kadaluwarsa, panggil `ReleaseToken` dengan metadata `reason: "auto_release_timeout"`.
  5. Catat log struktural dan perbarui metrik pembersihan.
- **Verifikasi (Bukti):** Unit test dengan data reservasi buatan berumur 6 menit dieksekusi -> otomatis dilepas dan kuota user kembali normal.

---

### Wave 3 — Dynamic Pricing, Entitlement Engine & API Gateway Integration

#### TASK-P04-008 — Dynamic Pricing Engine Service
- **Target File:** `backend/internal/billing/service/pricing_engine.go`
- **Langkah:**
  1. Buat service `CalculateTokenCostIDR(ctx, modelID string, modality string, tokenCount int64) (int64, error)`.
  2. Ambil aturan harga dari database/cache berdasarkan `model_id`. Fallback ke default base rate jika model belum terdaftar.
  3. Rumus komputasi:
     $$\text{Cost IDR} = \lceil \frac{\text{tokenCount}}{1000} \times \text{base\_per\_1k\_idr} \times \text{modality\_multiplier} \times (1 + \frac{\text{markup\_percent}}{100}) \rceil$$
  4. Pastikan pembagian menggunakan integer ceiling rounding untuk menghindari rounding error yang merugikan platform:
     `ceil((tokens * base * mult * (100 + markup)) / (1000 * 100))`
  5. Seluruh kalkulasi menghasilkan nilai `int64` Rupiah bulat murni.
- **Verifikasi (Bukti):** Unit test table-driven menguji 20 kombinasi skenario modalitas (chat, vision, embedding, audio) dan volume token; tidak ada desimal mengambang.

#### TASK-P04-009 — Domain Model dan Repository Modul Entitlement
- **Target File:** `backend/internal/entitlement/domain/entitlement.go`, `backend/internal/entitlement/repository/postgres_entitlement.go`
- **Langkah:**
  1. Definisikan model `UserEntitlementContext`: `UserID`, `PlanCode`, `Features map[string]bool`, `Limits map[string]int64`, `Version int64`, `ExpiresAt time.Time`.
  2. Buat repository `GetPlanEntitlements(ctx, planID)` dan `GetPlanLimits(ctx, planID)`.
  3. Buat service resolver `ResolveUserEntitlements(ctx, userID string) (*UserEntitlementContext, error)`:
     - Cari langganan aktif pengguna. Jika tidak ada / expired, pasang plan default `free`.
     - Ambil kumpulan flag fitur boolean dan limit numerik terkait plan tersebut.
- **Verifikasi (Bukti):** Test memverifikasi user tanpa langganan memperoleh fallback paket `free` dengan limit sesuai seeder.

#### TASK-P04-010 — Versioned Entitlement Cache Manager (L1/L2)
- **Target File:** `backend/internal/entitlement/cache/versioned_cache.go`
- **Langkah:**
  1. Struktur L1: In-memory cache berbasis LRU atau synchronized map dengan item TTL 60 detik.
  2. Struktur L2: Redis hash/string berversi dengan key: `entitlement:u:{userID}:v:{ver}` (TTL 24 jam).
  3. Version counter disimpan pada key Redis `entitlement:ver:{userID}` (default 1 jika kosong).
  4. Alur pembacaan `Get(ctx, userID)`:
     - Baca L1 memory lokal. Jika hit, langsung kembalikan (< 0.5 ms).
     - Jika miss L1, baca counter versi dari Redis, lalu ambil key berversi L2 (< 2 ms).
     - Jika miss L2, panggil DB `ResolveUserEntitlements`, simpan ke L2 dan L1.
  5. Alur `InvalidateUser(ctx, userID)`:
     - Eksekusi `INCR` pada Redis `entitlement:ver:{userID}`.
     - Hapus entri user pada L1 lokal.
- **Verifikasi (Bukti):** Test benchmark & unit test: L1 latency < 0.5 ms, verifikasi cache miss otomatis terhidrasi dari DB saat versi dinaikkan.

#### TASK-P04-011 — Redis Pub/Sub Synchronization untuk Invalidasi Multi-Instance
- **Target File:** `backend/internal/entitlement/cache/pubsub_listener.go`
- **Langkah:**
  1. Buat publisher: Saat `InvalidateUser` dipanggil, publish payload JSON `{"user_id": "...", "timestamp": ...}` ke channel Redis `entitlement:invalidations`.
  2. Buat subscriber latar belakang (goroutine) yang mendengarkan channel tersebut pada setiap node instance backend.
  3. Saat menerima pesan pub/sub, bersihkan L1 memory cache pada node lokal tersebut.
  4. Lengkapi dengan auto-reconnect loop jika koneksi Redis drop/reboot.
- **Verifikasi (Bukti):** Simulasi integrasi: publisher pada node A memicu pembersihan L1 cache pada node B secara otomatis dalam < 50ms.

#### TASK-P04-012 — Gin HTTP Middleware Penegakan Hak Akses (RequireFeature & WithinLimit)
- **Target File:** `backend/internal/entitlement/middleware/require_feature.go`, `within_limit.go`
- **Langkah:**
  1. Implementasikan `RequireFeature(featureKey string) gin.HandlerFunc`:
     - Ekstrak user dari context JWT/session.
     - Jika user role `super_admin`, bypass pengecekan (`c.Next()`).
     - Ambil entitlement dari `VersionedCache`.
     - Jika flag `featureKey` bernilai false / tidak ditemukan, hentikan request dengan HTTP `403 Forbidden` JSON (`code: "FEATURE_LOCKED"`, `upgrade_url: "/pricing"`).
  2. Implementasikan `WithinLimit(limitKey string, countProvider func(c *gin.Context) (int64, error)) gin.HandlerFunc`:
     - Bypass jika role `super_admin`.
     - Ambil entitlement limit numerik. Jika limit bernilai `-1` (unlimited), izinkan request.
     - Hitung pemakaian terkini via `countProvider`. Jika `currentUsage >= maxAllowed`, hentikan request dengan HTTP `429 Too Many Requests` JSON (`code: "LIMIT_EXCEEDED"`, info batasan dan upgrade link).
- **Verifikasi (Bukti):** Unit test handler Gin dengan mock middleware: validasi status HTTP 403, 429, dan 200 untuk berbagai peran pengguna.

#### TASK-P04-013 — Handler REST API Endpoint /api/v1/me/entitlements
- **Target File:** `backend/internal/billing/handler/entitlement_handler.go`, registrasi rute di router API
- **Langkah:**
  1. Buat HTTP GET handler `/api/v1/me/entitlements` terlindungi JWT auth middleware.
  2. Ambil profil langganan dan entitlement terkini dari `VersionedCache`.
  3. Ambil sisa kuota token hari ini dari `token_periods` (`daily_quota`, `used_tokens`, `available_quota`).
  4. Susun JSON respons terstruktur:
     ```json
     {
       "status": "success",
       "data": {
         "plan": { "code": "pro", "name": "Pro Tier", "status": "active" },
         "features": { "ai_chat": true, "ocr_receipt": true },
         "limits": { "max_wallets": { "used": 3, "limit": 10 } },
         "tokens": { "daily_quota": 250000, "used": 15000, "available": 235000 },
         "version": 4
       }
     }
     ```
- **Verifikasi (Bukti):** HTTP integration test: endpoint mengembalikan status 200 OK dengan payload lengkap dan latensi respons < 5ms.

---

### Wave 4 — Subscription Lifecycle, Proration & Soft-Limit Policies

#### TASK-P04-014 — Subscription Lifecycle Service & State Transitions
- **Target File:** `backend/internal/billing/service/subscription_lifecycle.go`
- **Langkah:**
  1. Bangun state machine transisi status langganan:
     - `trialing` -> `active` / `expired`
     - `active` -> `past_due` / `canceled`
     - `past_due` -> `active` / `grace` / `expired`
     - `grace` -> `active` / `expired`
     - `canceled` -> `expired`
  2. Tolak transisi ilegal (misal: `expired` -> `past_due`) dengan `ErrInvalidStateTransition`.
  3. Setiap transisi sukses wajib:
     - Mengubah kolom `status` di tabel `subscriptions`.
     - Mencatat audit event pada `subscription_events` (`event_type`, payload `before` dan `after`).
     - Memanggil `InvalidateUser` untuk memaksa reload hak akses di seluruh node.
- **Verifikasi (Bukti):** Unit test transisi state: memvalidasi semua kombinasi matriks transisi yang sah dan menolak status tak sah.

#### TASK-P04-015 — Logika Upgrade Prorata Instan dan Downgrade Soft-Limit
- **Target File:** `backend/internal/billing/service/subscription_management.go`
- **Langkah:**
  1. Implementasikan `CalculateUpgradeProration(currentSub, newPlan) (int64, error)`:
     - Hitung total hari dalam siklus langganan aktif saat ini ($D_{total}$).
     - Hitung sisa hari dari waktu eksekusi sampai `current_period_end` ($D_{remain}$).
     - Hitung sisa kredit paket lama: $\lfloor \text{price}_{old} \times \frac{D_{remain}}{D_{total}} \rfloor$.
     - Hitung biaya paket baru untuk sisa periode: $\lfloor \text{price}_{new} \times \frac{D_{remain}}{D_{total}} \rfloor$.
     - Biaya upgrade instan = $\max(0, \text{Biaya Baru} - \text{Sisa Kredit Lama})$.
     - Seluruh perhitungan dibulatkan ke bawah (*floor*) dalam integer IDR.
  2. Implementasikan `ScheduleDowngrade(userID, newPlanID) error`:
     - Terapkan kebijakan **soft-limit**: Jangan langsung cabut hak akses saat user memilih downgrade.
     - Set kolom `cancel_at_period_end = true` dan catat rencana paket baru pada metadata `subscriptions`.
     - Paket lama tetap aktif sampai `current_period_end`.
     - Validasi prinsip read-only pada entitas berlebih: user yang turun ke paket Free tidak kehilangan dompet historisnya, namun terblokir menambah dompet baru oleh middleware `WithinLimit`.
- **Verifikasi (Bukti):** Unit test komputasi prorata menghasilkan nilai bilangan bulat matematis akurat; pengujian skenario downgrade tidak menghapus data historis pengguna.

---

### Wave 5 — Concurrency Testing, Telemetry & Handoff Interfaces

#### TASK-P04-016 — Concurrency Testing dan Verifikasi Anti Double-Spending
- **Target File:** `backend/internal/billing/service/token_ledger_test.go`
- **Langkah:**
  1. Buat integration test otomatis berbasis goroutine dengan runtime flag `-race`.
  2. Skenario uji:
     - Siapkan pengguna dengan kuota harian 10.000 token.
     - Jalankan 50 request konkuren secara simultan yang masing-masing meminta reservasi 500 token (total permintaan 25.000 token > kuota 10.000).
  3. Verifikasi:
     - Tepat 20 request berhasil di-reserve (20 x 500 = 10.000 token), dan 30 request lainnya ditolak dengan `ErrInsufficientQuota`.
     - Tidak ada kondisi balapan (*zero data race*).
     - Saldo sisa kuota pada `token_periods` bernilai tepat 0, tidak pernah tembus angka negatif.
     - Lakukan kombinasi 50 reservasi acak diikuti commit dan release; verifikasi total mutasi ledger append-only seimbang sempurna.
- **Verifikasi (Bukti):** `go test -v -race ./backend/internal/billing/service -run TestTokenLedger_Concurrency` menghasilkan exit code 0 tanpa race failure.

#### TASK-P04-017 — Metrik Prometheus, Observabilitas & Kontrak Antarmuka P05/P06
- **Target File:** `backend/internal/billing/telemetry/metrics.go`, `backend/internal/billing/contract.go`
- **Langkah:**
  1. Registrasikan counter dan histogram Prometheus:
     - `billing_token_reservations_total{status="success|failed|insufficient"}`
     - `billing_token_commits_total`
     - `billing_token_releases_total{reason="caller|timeout|error"}`
     - `entitlement_cache_requests_total{level="l1|l2|db", result="hit|miss"}`
     - `entitlement_middleware_duration_seconds`
  2. Definisikan interface publik yang bersih (*dependency injection*) untuk konsumsi modul berikutnya:
     - `BillingService`: aktivasi langganan dari webhook Midtrans P05, pembuatan invoice, dan upgrade plan.
     - `EntitlementService`: verifikasi fitur dan kuota user.
     - `TokenLedgerManager`: reserve, commit, release untuk orkestrasi AI Gateway P03.
     - Rute admin billing untuk dashboard Super Admin P06.
- **Verifikasi (Bukti):** Metrik muncul pada endpoint `/metrics`; interface terkompilasi bersih tanpa cyclic dependency.

---

## 4. Matriks Bukti

| Task ID | Nama Task | Artefak Target | Bentuk Bukti Verifikasi |
|---|---|---|---|
| **TASK-P04-001** | Skema Migrasi DB 0008 | `database/migrations/0008_subscription_token_core.sql` | Output migration runner up/down sukses, schema dump terbarui |
| **TASK-P04-002** | Seeder Paket & Pricing | `database/seeds/0002_subscription_token_seeds.sql` | Kueri DB menghasilkan paket free/pro/business dan rules default |
| **TASK-P04-003** | Domain Entity Billing | `backend/internal/billing/domain/*.go` | `go vet` lulus, unit test domain errors & enums hijau |
| **TASK-P04-004** | Repository PostgreSQL Billing | `backend/internal/billing/repository/*.go` | Test kueri database mock/transaksional lulus row-lock test |
| **TASK-P04-005** | Token Reserve Flow | `backend/internal/billing/service/token_ledger.go` | Test isolasi transaksi `SELECT ... FOR UPDATE` dan idempotency |
| **TASK-P04-006** | Token Commit & Release | `backend/internal/billing/service/token_ledger.go` | Test mutasi ledger append-only dan rekonsiliasi kuota token |
| **TASK-P04-007** | Background Cleaner Worker | `backend/internal/billing/worker/ledger_cleaner.go` | Test worker auto-release reservasi kadaluwarsa > 5 menit |
| **TASK-P04-008** | Dynamic Pricing IDR | `backend/internal/billing/service/pricing_engine.go` | Table-driven test 20 kasus pembulatan Rupiah bulat (`int64`) |
| **TASK-P04-009** | Entitlement Domain & Repo | `backend/internal/entitlement/domain/`, `repository/` | Test resolusi entitlement fallback paket default `free` |
| **TASK-P04-010** | Versioned Cache L1/L2 | `backend/internal/entitlement/cache/versioned_cache.go` | Benchmark latensi L1 < 0.5ms, L2 < 2ms, invalidasi counter |
| **TASK-P04-011** | Redis Pub/Sub Synchronization | `backend/internal/entitlement/cache/pubsub_listener.go` | Multi-instance simulation test invalidasi memori lokal < 50ms |
| **TASK-P04-012** | Entitlement Middleware Gin | `backend/internal/entitlement/middleware/*.go` | HTTP test: 403 Forbidden, 429 Too Many Requests, admin bypass |
| **TASK-P04-013** | Endpoint /me/entitlements | `backend/internal/billing/handler/entitlement_handler.go` | HTTP integration test status 200 OK dengan payload lengkap |
| **TASK-P04-014** | Lifecycle State Transitions | `backend/internal/billing/service/subscription_lifecycle.go` | Unit test state machine & verifikasi rekaman audit event |
| **TASK-P04-015** | Upgrade Prorata & Downgrade | `backend/internal/billing/service/subscription_management.go` | Unit test kalkulasi prorata floor IDR & pengujian soft-limit |
| **TASK-P04-016** | Concurrency Testing Anti Double-Spend | `backend/internal/billing/service/token_ledger_test.go` | Eksekusi `go test -race` 50 goroutines lulus tanpa saldo minus |
| **TASK-P04-017** | Prometheus Metrics & Handoff | `backend/internal/billing/telemetry/`, `contract.go` | Scraping endpoint `/metrics` valid, kontrak interface Go siap |

---

## 5. Perintah Verifikasi

Perintah pengujian standar yang wajib dijalankan untuk membuktikan integritas kode:

```bash
# 1. Verifikasi kode Go (Static Analysis, Build, Unit Tests)
cd /d/project/finance/fintrack_v2/backend
go vet ./internal/billing/... ./internal/entitlement/...
go build ./internal/billing/... ./internal/entitlement/...
go test -v ./internal/billing/... ./internal/entitlement/...

# 2. Pengujian Konkurensi & Anti Double-Spending (WAJIB dengan flag -race)
go test -v -race -run TestTokenLedger_Concurrency ./internal/billing/service/...

# 3. Pengujian Benchmarking Cache L1/L2
go test -bench=BenchmarkEntitlementCache -benchmem ./internal/entitlement/cache/...

# 4. Verifikasi Frontend Lint & Typecheck (konsistensi antarmuka)
cd /d/project/finance/fintrack_v2/frontend
pnpm lint
pnpm typecheck

# 5. Verifikasi Migrasi Database
# Pastikan migrasi up & down berjalan mulus pada database pengujian
migrate -path database/migrations -database "$DATABASE_URL" up
migrate -path database/migrations -database "$DATABASE_URL" down 1
migrate -path database/migrations -database "$DATABASE_URL" up
```

---

## 6. Prosedur Rollback

Bila terjadi kegagalan fatal pada tahap rilis staging maupun produksi:
1. **Rollback Migrasi Skema Basis Data:**
   - Jalankan migrasi mundur untuk file `0008`:
     `migrate -path database/migrations -database "$DATABASE_URL" down 1`
   - Skrip drop table otomatis membersihkan tabel `pricing_rules`, `token_ledger`, `token_periods`, `subscription_events`, `subscriptions`, `plan_limits`, `plan_entitlements`, dan `plans`.
2. **Rollback Kode Backend:**
   - Kembalikan service binary ke versi sebelum rilis P04 via Docker deployment rolling rollback.
   - Karena rute dan middleware baru berada di balik modul modular, tidak ada dependensi yang merusak rute transaksi core P01.
3. **Pembersihan Cache Redis:**
   - Jalankan pembersihan namespace key cache berversi melalui redis-cli:
     `redis-cli --scan --pattern "entitlement:*" | xargs -r redis-cli del`

---

## 7. Handoff ke Fase Berikutnya

Setelah Fase P04 tuntas dan seluruh kriteria Definition of Done tercapai, fondasi ini diserahterimakan ke:
1. **Fase P05 (Payments & Invoices):**
   - Modul `internal/payment` akan mengonsumsi interface `BillingService.ActivateSubscription` untuk mengaktifkan paket secara otomatis saat webhook pembayaran Midtrans Snap berstatus settlement.
   - Invoice engine P05 memanfaatkan riwayat event pada `subscription_events` dan data `plans`.
2. **Fase P06 (Settings & Admin Dashboard):**
   - Dashboard Super Admin mengonsumsi rute dan service `PlanRepository`, `PricingRuleRepository`, serta fungsi `AdjustTokenBalance` untuk mengelola paket dan aturan markup token secara dinamis.
3. **Fase P03 (AI Gateway Integration):**
   - Proxy handler AI mengonsumsi `TokenLedgerManager` untuk membungkus setiap pemanggilan model LLM ke dalam pola 2-fase: `ReserveToken` sebelum stream dan `CommitToken` / `ReleaseToken` saat stream berakhir.

---

## 8. Referensi

- **README Master:** `second_brain/2_Projects/P04_Subscription-Token-Core/README.md`
- **Work Breakdown Structure:** `second_brain/2_Projects/P04_Subscription-Token-Core/planning/tasks.md`
- **Timeline & Milestone:** `second_brain/2_Projects/P04_Subscription-Token-Core/planning/timeline.md`
- **Spesifikasi Arsitektur Billing:** `second_brain/2_Projects/P04_Subscription-Token-Core/teknis/billing-architecture.md`
- **Spesifikasi Token Ledger Engine:** `second_brain/2_Projects/P04_Subscription-Token-Core/teknis/token-ledger.md`
- **Spesifikasi Matriks Entitlement:** `second_brain/2_Projects/P04_Subscription-Token-Core/teknis/entitlement-matrix.md`
- **Spesifikasi Versioned Cache:** `second_brain/2_Projects/P04_Subscription-Token-Core/teknis/entitlement-cache.md`
- **Spesifikasi Middleware Penegakan:** `second_brain/2_Projects/P04_Subscription-Token-Core/teknis/enforcement-middleware.md`
- **Spesifikasi Pricing Engine:** `second_brain/2_Projects/P04_Subscription-Token-Core/teknis/pricing-engine.md`
- **Spesifikasi Siklus Langganan:** `second_brain/2_Projects/P04_Subscription-Token-Core/teknis/subscription-lifecycle.md`
- **Master Plan Proyek:** `second_brain/2_Projects/IMPLEMENTATION-PLAN.md`
