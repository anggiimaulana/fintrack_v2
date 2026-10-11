# Sprint Plan — P04: Subscription & Token Core

## Informasi Sprint
- **Nama Sprint:** P04 — Subscription & Token Core Engineering
- **Fokus Utama:** Implementasi modul `internal/billing/` & `internal/entitlement/`, skema migrasi database 0007, alur 2-phase token ledger, dynamic pricing engine, dan versioned entitlement cache middleware.
- **Durasi:** 2 Pekan (10 Hari Kerja)
- **Target Mulai:** 2026-10-19
- **Target Selesai:** 2026-10-30
- **Dependensi Hulu:** P02 (Fondasi Platform & Redis), P03 (AI Gateway Terpusat & Adapter Interface)
- **Keluaran Hilir:** P05 (Payments & Billing Webhook), P06 (Super Admin Dashboard & Plan Management)

---

## Sasaran Sprint (Sprint Goal)
Membangun pondasi monetisasi FinTrack v2 yang tangguh dan terisolasi secara transaksional dengan menjamin:
1. Skema database subscription, token ledger, dan entitlement terpasang tanpa downtime.
2. Alur mutasi token 2-fase (reserve-commit-release) dengan proteksi idempotensi dan konkurensi tinggi.
3. Mesin harga dinamis integer Rupiah untuk seluruh modalitas AI.
4. Middleware otorisasi entitlement dengan cache berversi berkecepatan < 2ms dan invalidasi seketika tanpa perlu relogin.
5. Kebijakan upgrade prorata instan dan downgrade soft-limit aman tanpa penghapusan data pengguna.

---

## Alokasi Kapasitas dan Milestones

| Hari | Target Kerja / Milestones | Fokus Area |
|---|---|---|
| **Hari 1–2** | **Desain Skema DB & Migrasi DDL Replayable**<br>- Pembuatan file migrasi `0008_subscription_token_core.sql`<br>- Skema tabel plans, entitlements, limits, subscriptions, token_ledger, pricing_rules<br>- Script rollback & seed data default plan (Free, Pro, Business) | Basis Data & ORM/Repository SQL |
| **Hari 3–4** | **Domain Model & Repository Layer (`internal/billing/`)**<br>- Domain entity, value objects, domain errors<br>- Postgres repository dengan `SELECT ... FOR UPDATE`<br>- Implementasi unit test transaksi repository dengan isolation test | Backend Core / Repository |
| **Hari 5–6** | **Token Ledger Engine & Pricing Engine**<br>- Implementasi 2-phase reserve-call-commit/release flow<br>- Dynamic pricing calculator (integer IDR, modality weights)<br>- Idempotency validator & auto-release background cleaner worker | Backend Service / Ledger |
| **Hari 7–8** | **Entitlement Resolver & Versioned Cache System**<br>- Implementasi L1 in-memory + L2 Redis cache manager<br>- Redis Pub/Sub invalidation worker & user version counter<br>- Gin HTTP middleware `RequireFeature` & `WithinLimit`<br>- Handler endpoint `/api/v1/me/entitlements` | Backend Middleware & API |
| **Hari 9** | **Subscription Lifecycle & Kebijakan Prorata/Downgrade**<br>- Transisi state machine: trialing, active, past_due, grace, canceled, expired<br>- Kalkulasi prorated upgrade credit & implementasi soft-limit downgrade<br>- Event auditor untuk pencatatan `subscription_events` | Backend Service / Lifecycle |
| **Hari 10** | **Integrasi P03/AI Gateway, Chaos Testing & Finalisasi**<br>- Pengujian integrasi ledger token dengan mock AI Gateway<br>- Concurrency load test (race condition check, double-spending prevention)<br>- Audit compliance UU PDP check & kelengkapan dokumentasi | QA, Integration & Hardening |

---

## Rincian Sprint Backlog

### Milestone 1: Database & Foundation (Hari 1–2)
- **TASK-P04-001:** DDL migrasi skema tabel subscription, plan, entitlement, limits, token, dan pricing rules.
- **TASK-P04-002:** Seeder default plans (Free, Pro, Business) dan konfigurasi pricing rules awal.
- **TASK-P04-003:** Domain model, enum status subscription, dan value objects di `internal/billing/domain/`.

### Milestone 2: Token Ledger & Concurrency (Hari 3–5)
- **TASK-P04-004:** Postgres repository untuk `token_periods` dan `token_ledger` dengan locking row `FOR UPDATE`.
- **TASK-P04-005:** Service reserve token dengan validasi sisa kuota siklus berjalan dan penandaan `idempotency_key`.
- **TASK-P04-006:** Service commit dan release token dengan kalkulasi token aktual pasca-eksekusi AI Gateway.
- **TASK-P04-007:** Background worker pembersih reservasi kuota expired (auto-release timeout).
- **TASK-P04-008:** Dynamic pricing engine calculator dengan integer IDR dan modality multiplier.

### Milestone 3: Entitlement Engine & Middleware (Hari 6–8)
- **TASK-P04-009:** Domain model dan repository resolver untuk fitur boolean dan batas limit numerik.
- **TASK-P04-010:** Versioned cache manager (L1 In-Memory + L2 Redis) berformat key `entitlement:u:{uid}:v:{ver}`.
- **TASK-P04-011:** Redis Pub/Sub subscriber untuk invalidasi instan lintas node backend.
- **TASK-P04-012:** Gin Middleware `RequireFeature` dan `WithinLimit`.
- **TASK-P04-013:** Controller dan endpoint HTTP `GET /api/v1/me/entitlements`.

### Milestone 4: Subscription Lifecycle & Polish (Hari 9–10)
- **TASK-P04-014:** State machine subscription lifecycle dan pencatatan audit log `subscription_events`.
- **TASK-P04-015:** Logika upgrade instan prorata dan downgrade soft-limit akhir siklus.
- **TASK-P04-016:** Integration testing konkurensi (mencegah double-spending token & race conditions).
- **TASK-P04-017:** Observabilitas, metrik Prometheus, dan penyelarasan kontrak API dengan P05 dan P06.

---

## Manajemen Risiko & Rencana Mitigasi

| Risiko | Tingkat Dampak | Probabilitas | Rencana Mitigasi |
|---|---|---|---|
| **Race condition saat konkurensi kuota tinggi** | Tinggi | Sedang | Gunakan transaksi PostgreSQL dengan isolasi `READ COMMITTED` + `SELECT ... FOR UPDATE` pada baris `token_periods`. Uji dengan 100 concurrent worker di unit test. |
| **Lock contention memperlambat latency streaming AI** | Tinggi | Sedang | Pisahkan fase `Reserve` (lock singkat), eksekusi AI (di luar DB transaction), dan `Commit` (lock singkat terpisah). Jangan pernah menahan lock DB selama proses streaming AI. |
| **Redis downtime mematikan otorisasi endpoint** | Tinggi | Rendah | Buat fallback otomatis ke pembacaan database PostgreSQL langsung jika Redis tidak merespons dalam 50ms. |
| **Inkonsistensi pembulatan mata uang rupiah** | Sedang | Rendah | Kunci semua field moneter dan token ke tipe data integer bulat (`int64`). Dilarang keras menggunakan tipe `float64` untuk representasi mata uang IDR. |
| **Ketergantungan terhadap antarmuka P03 (AI Gateway)** | Sedang | Sedang | Sediakan mock interface `AIGatewayMeteringClient` pada layer service sehingga unit test billing tidak bergantung pada ketersediaan endpoint AI Gateway aktual. |

---

## Kriteria Selesai Sprint (Sprint Definition of Done)
1. Seluruh 17 tasks (`TASK-P04-001` hingga `TASK-P04-017`) terselesaikan dengan status selesai (Done).
2. Kode terisolasi dalam modul `internal/billing/` dan `internal/entitlement/` dengan clean architecture ber-interface.
3. Test suite unit test dan integrasi lulus 100% tanpa race condition terdeteksi (`go test -race ./...`).
4. Benchmark menunjukkan latensi middleware entitlement < 2ms pada kondisi cache hit.
5. Dokumentasi teknis lengkap dan selaras dengan implementasi kode tanpa penggunaan placeholder/TODO/TBD.
