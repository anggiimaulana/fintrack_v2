# Tracking Status — P04: Subscription & Token Core

Dokumen ini memantau progres eksekusi tugas (*task tracking*) untuk Fase P04 secara berkala menggunakan status simbolik standar:
- ⬜ : Belum dimulai (*To Do*)
- 🔄 : Sedang dikerjakan (*In Progress*)
- ✅ : Selesai dan terverifikasi (*Done* — kode ada, test hijau)

> **Status saat ini: BELUM DIMULAI.** Seluruh task masih ⬜ — belum ada implementasi kode.
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
| **TASK-P04-001** | Database | Skema Migrasi DDL `0007_subscription_token_core.sql` | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-002** | Database | Seeder Default Plans (Free/Pro/Business) & Pricing Rules | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-003** | Billing | Domain Entities, Value Objects & Domain Errors | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-004** | Billing | PostgreSQL Repository Layer dengan Row Locking | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-005** | Billing | Token Ledger Service — Reserve Flow & Quota Check | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-006** | Billing | Token Ledger Service — Commit & Release Flow | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-007** | Billing | Background Cleaner Worker (Auto-Release Timeout) | Backend Worker | ⬜ | Belum dikerjakan |
| **TASK-P04-008** | Billing | Dynamic Pricing Engine Calculator (Integer IDR) | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-009** | Entitlement | Domain Model & Aggregation Repository Resolver | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-010** | Entitlement | Versioned Cache Manager L1 (In-Memory) & L2 (Redis) | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-011** | Entitlement | Redis Pub/Sub Subscriber Invalidation Lintas Node | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-012** | Entitlement | Gin Middleware `RequireFeature` & `WithinLimit` | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-013** | Entitlement | REST API Endpoint `GET /api/v1/me/entitlements` | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-014** | Billing | Subscription Lifecycle State Machine & Audit Events | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-015** | Billing | Upgrade Prorata Instan & Downgrade Soft-Limit Logic | Backend Core | ⬜ | Belum dikerjakan |
| **TASK-P04-016** | QA / Test | Concurrency Testing & Anti Double-Spending Check | QA Engineer | ⬜ | Belum dikerjakan |
| **TASK-P04-017** | Observability | Prometheus Metrics, Tracing & Handoff Kontrak P05/P06 | DevOps / Lead | ⬜ | Belum dikerjakan |

---

## Log Pembaruan Terkini

- **2026-10-11**: Perencanaan & spesifikasi teknis fase P04 disusun (17 task, analisis K-P04-01..K-P04-20).
  **Implementasi belum dimulai.**

---

## Checklist Definition of Done Umum (Fase P04)

- [ ] `go vet && go test && go build` hijau.
- [ ] Migrasi database `0007_subscription_token_core.sql` replayable maju dan mundur tanpa error.
- [ ] Fitur penegakan kuota terpasang di balik feature flag konfigurasi.
- [ ] Test konkurensi membuktikan zero double-spending pada token ledger (`go test -race`).
- [ ] Audit log mencatat mutasi status langganan dan koreksi ledger secara append-only.
- [ ] Tidak ada data finansial pengguna yang terhapus saat terjadi penurunan paket (soft-limit aktif).
- [ ] Latensi middleware hak akses berada di bawah 2ms untuk cache hit.
- [ ] Dokumen teknis pada direktori `teknis/` terisi lengkap dan akurat.
