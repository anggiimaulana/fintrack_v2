# P04 — Subscription & Token Core

## Ringkasan Proyek
Fase P04 adalah fondasi inti monetisasi dan pembatasan fitur FinTrack v2. Fase ini mengimplementasikan modul backend `internal/billing/` dan `internal/entitlement/`, skema basis data langganan dan token, alur transaksi ledger token (reserve-call-commit/release), mesin kalkulasi harga token dinamis berbasis integer rupiah, siklus kuota per langganan (Asia/Jakarta), siklus hidup subscription (active, past_due, grace, canceled, expired, trialing), serta middleware penegakan hak akses (*entitlement enforcement*) berbasis cache berversi dengan zero-downtime dan tanpa keharusan relogin.

## Ruang Lingkup (Scope)
1. **Model Data & Migrasi Basis Data:**
   - Tabel `plans`: Definisi paket (Free, Pro, Business), harga dasar per periode siklus (integer rupiah), status aktif, fallback gateway.
   - Tabel `plan_entitlements`: Hak akses biner (boolean flags) per plan (e.g. `export_csv`, `ai_chat`, `bank_sync`, `ocr_receipt`, `advanced_budgeting`).
   - Tabel `plan_limits`: Batasan numerik kuota per plan (e.g. `max_accounts`, `max_budgets`, `ai_token_quota`, `ocr_quota`).
   - Tabel `subscriptions`: Status langganan per-user (`trialing`, `active`, `past_due`, `grace`, `canceled`, `expired`), timestamp periode (`current_period_start`, `current_period_end`), gateway referensi, flag `cancel_at_period_end`.
   - Tabel `subscription_events`: Audit log append-only untuk transisi siklus hidup langganan (state before -> after, alasan, metadata payload).
   - Tabel `token_periods`: Pelacakan periode kuota token per user, kuota alokasi, terpakai (`used`), dan terkunci sementara (`reserved`).
   - Tabel `token_ledger`: Pencatatan mutasi token append-only dengan jenis mutasi `reserve`, `commit`, `release`, `grant`, `adjust`, dan idempotency key unik.
   - Tabel `pricing_rules`: Konfigurasi penetapan harga token dinamis (`base_per_1k_idr`, `markup_multiplier`, `markup_percent`, `modality_multipliers` JSONB).

2. **Mesin Billing & Token Ledger (`internal/billing/`):**
   - Transaksi 2-fase ledger token: `Reserve(user, tokens, idempotency_key)` -> `Call AI` -> `Commit(user, actual_tokens, reserve_id)` atau `Release(user, reserve_id)`.
   - Pencegahan *double-charge*, *race condition*, dan kuota negatif (*concurrency control* via `SELECT ... FOR UPDATE` dan constraint DB).
   - Engine harga dinamis: Konversi unit token AI (chat/vision/voice/embedding) ke estimasi rupiah dengan integer arithmetic.
   - Siklus kuota berbasis hari kalender Asia/Jakarta (reset otomatis awal siklus, no rollover).
   - Upgrade paket instan (prorated kredit) dan Downgrade paket di akhir siklus dengan kebijakan *soft-limit*.

3. **Mesin Entitlement & Middleware (`internal/entitlement/`):**
   - Entitlement resolver: Menggabungkan status subscription, plan entitlements, dan plan limits.
   - Cache berversi multi-level (Local In-Memory LRU + Redis Distributed Cache) berformat `user_entitlement:{user_id}:v{version}`.
   - Cache invalidation seketika via Redis Pub/Sub dan bumping user version counter saat terjadi mutasi plan/langganan.
   - Gin HTTP Middleware: `RequireFeature(featureKey)` dan `WithinLimit(limitKey, currentUsageResolver)`.
   - REST API Endpoint: `GET /api/v1/me/entitlements` untuk sinkronisasi antarmuka pengguna (Next.js TanStack Query).

## Deliverables
1. Skema SQL migrasi DDL replayable `database/migrations/0008_subscription_token_core.sql` beserta rollback script.
2. Modul backend Go:
   - `internal/billing/domain/`: Entities, value objects, domain errors.
   - `internal/billing/repository/`: Implementasi PostgreSQL repository dengan transaksi aman.
   - `internal/billing/service/`: Subscription lifecycle service, token ledger manager, pricing engine.
   - `internal/entitlement/domain/`: Fitur, batasan, evaluasi aturan.
   - `internal/entitlement/service/`: Entitlement resolver, versioned cache manager.
   - `internal/entitlement/middleware/`: Gin middleware `RequireFeature` & `WithinLimit`.
   - `internal/billing/handler/`: Endpoint `/api/v1/plans`, `/api/v1/subscriptions/me`, `/api/v1/me/entitlements`.
3. Unit tests dan integration test konkurensi (pembuktian zero double-spending token ledger & cache invalidation latency < 50ms).
4. Dokumentasi teknis komprehensif pada direktori `second_brain/2_Projects/P04_Subscription-Token-Core/`.

## Kriteria Sukses (Success Criteria)
1. [x] **Integritas Finansial & Token:** Zero double-spending / saldo negatif. *(diuji: 40 reserve paralel kuota 1000 → tepat 10 sukses; 10 commit ganda → 1 efek; reserved ≤ quota)*
2. [x] **Latensi Penegakan Hak Akses:** middleware entitlement cache-hit (L1) cepat. *(diuji L1 cache hit tanpa query DB)*
3. [x] **Propagasi Invalidasi Hak Akses:** perubahan plan efektif seketika tanpa logout. *(version bump + pub/sub `entitlement:invalidations`; diuji)*
4. [x] **Soft-Limit Downgrade:** tidak ada data historis user yang dihapus saat downgrade. *(downgrade = `cancel_at_period_end` + `scheduled_plan_code`; tanpa delete)*
5. [x] **Kepatuhan Tipe Data:** seluruh nilai rupiah integer (BIGINT/int64). *(pricing engine integer floor; diuji)*
6. [x] **Zero Downtime:** migrasi aditif & replayable. *(0008 idempoten; diuji)*
