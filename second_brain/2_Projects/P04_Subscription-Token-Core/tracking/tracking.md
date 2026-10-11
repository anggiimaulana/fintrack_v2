# Tracking Status — P04: Subscription & Token Core

Dokumen ini memantau progres eksekusi tugas (*task tracking*) untuk Fase P04 secara berkala menggunakan status simbolik standar:
- ⬜ : Belum dimulai (*To Do*)
- 🔄 : Sedang dikerjakan (*In Progress*)
- ✅ : Selesai dan terverifikasi (*Done* — kode ada, test hijau)

> **Status: BERJALAN.** W1 (skema, seeder, domain, repo) selesai. ✅ hanya setelah dibuktikan test nyata.
> Status diperbarui saat eksekusi berjalan.

---

## Ringkasan Progres

| Total Tugas | Belum Dimulai (⬜) | Sedang Dikerjakan (🔄) | Selesai (✅) | Persentase Selesai |
|---|---|---|---|---|
| **17** | **17** | **0** | **0** | **0%** |

---

## Tabel Status Tugas (TASK-P04-001 .. TASK-P04-017)

| Task ID | Komponen | Judul Pekerjaan | Assignee | Status | Catatan |
|---|---|---|---|:---:|---|
| **TASK-P04-001** | Database | Skema Migrasi DDL `0008_subscription_token_core.sql` | Backend Core | ✅ | 8 tabel diterapkan; trigger append-only `token_ledger` (UPDATE/DELETE ditolak); check constraint & idempotency unique terbukti; replay OK |
| **TASK-P04-002** | Database | Seeder Default Plans (Free/Pro/Business) & Pricing Rules | Backend Core | ✅ | `seeds/0002_subscription_token_seeds.sql` — 3 plan + 24 entitlement + 18 limit + pricing default; replay idempoten (tetap 3 plan) |
| **TASK-P04-003** | Billing | Domain Entities, Value Objects & Domain Errors | Backend Core | ✅ | `internal/billing/{domain.go,repository.go}` — entity, enum, `TokenPeriod.Available()`, error sentinel, interface repo; build OK |
| **TASK-P04-004** | Billing | PostgreSQL Repository Layer dengan Row Locking | Backend Core | ✅ | `repository/postgres/billing_repo.go` (plan/subscription/token/pricing, `SELECT ... FOR UPDATE`, idempotency 23505→ErrDuplicate); 2 test PASS vs DB live |
| **TASK-P04-005** | Billing | Token Ledger Service — Alur Reserve | Backend Core | ✅ | `billing/ledger.go` Reserve (row-lock FOR UPDATE, idempoten); test PASS |
| **TASK-P04-006** | Billing | Token Ledger Service — Commit & Release Flow | Backend Core | ✅ | `billing/ledger.go` Commit/Release (release sisa reserve, idempoten); test PASS |
| **TASK-P04-007** | Billing | Background Cleaner Worker (Auto-Release Timeout) | Backend Core | ✅ | `billing/cleaner.go` (sweep reservation basi); test PASS (auto-release) |
| **TASK-P04-008** | Billing | Dynamic Pricing Engine Calculator (Integer IDR) | Backend Core | ✅ | `billing/pricing.go` (base+markup+modality, floor); 6 test PASS |
| **TASK-P04-009** | Entitlement | Domain Model & Aggregation Repository Resolver | Backend Core | ✅ | `entitlement/domain.go` + `resolver.go`; `EntitlementSource` (join subscription+plan); 6 test PASS |
| **TASK-P04-010** | Entitlement | Versioned Cache Manager L1 (In-Memory) & L2 (Redis) | Backend Core | ✅ | `resolver.go` L1+L2 berversi (`entitlement:u:{uid}:v{ver}`); test invalidation & TTL PASS |
| **TASK-P04-011** | Entitlement | Redis Pub/Sub Subscriber Invalidation Lintas Node | Backend Core | ✅ | `entitlement/broadcaster.go` (channel `entitlement:invalidations`); wired di main |
| **TASK-P04-012** | Entitlement | Gin Middleware `RequireFeature` & `WithinLimit` | Backend Core | ✅ | `middleware/entitlement.go` (403/402, super_admin bypass, fail-closed); 6 test PASS |
| **TASK-P04-013** | Entitlement | REST API Endpoint `GET /api/v1/me/entitlements` | Backend Core | ✅ | `handler/entitlement_handler.go` + route; wired di main |
| **TASK-P04-014** | Billing | Subscription Lifecycle State Machine & Audit Events | Backend Core | ✅ | `billing/lifecycle.go` (matriks transisi + audit event + invalidate); test PASS |
| **TASK-P04-015** | Billing | Upgrade Prorata Instan & Downgrade Soft-Limit Logic | Backend Core | ✅ | `billing/subscription_management.go` (prorata floor integer, schedule downgrade soft-limit); 5 test PASS |
| **TASK-P04-016** | QA / Test | Concurrency Testing & Anti Double-Spending Check | QA Engineer | ⬜ | Belum dikerjakan |
| **TASK-P04-017** | Observability | Prometheus Metrics, Tracing & Handoff Kontrak P05/P06 | DevOps / Lead | ⬜ | Belum dikerjakan |

---

## Log Pembaruan Terkini

- **2026-10-11**: Perencanaan & spesifikasi teknis fase P04 disusun (17 task, analisis K-P04-01..K-P04-20).
  **Implementasi belum dimulai.**

---

## Checklist Definition of Done Umum (Fase P04)

- [ ] `go vet && go test && go build` hijau.
- [ ] Migrasi database `0008_subscription_token_core.sql` replayable maju dan mundur tanpa error.
- [ ] Fitur penegakan kuota terpasang di balik feature flag konfigurasi.
- [ ] Test konkurensi membuktikan zero double-spending pada token ledger (`go test -race`).
- [ ] Audit log mencatat mutasi status langganan dan koreksi ledger secara append-only.
- [ ] Tidak ada data finansial pengguna yang terhapus saat terjadi penurunan paket (soft-limit aktif).
- [ ] Latensi middleware hak akses berada di bawah 2ms untuk cache hit.
- [ ] Dokumen teknis pada direktori `teknis/` terisi lengkap dan akurat.
