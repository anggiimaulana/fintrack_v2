# Tracking Status — P04: Subscription & Token Core

Dokumen ini memantau progres eksekusi tugas (*task tracking*) untuk Fase P04 secara berkala menggunakan status simbolik standar:
- ⬜ : Belum dimulai (*To Do*)
- 🔄 : Sedang dikerjakan (*In Progress*)
- ✅ : Selesai dan terverifikasi (*Done*)

---

## Ringkasan Progres

| Total Tugas | Belum Dimulai (⬜) | Sedang Dikerjakan (🔄) | Selesai (✅) | Persentase Selesai |
|---|---|---|---|---|
| **17** | **14** | **1** | **2** | **11.7%** |

---

## Tabel Status Tugas (TASK-P04-001 .. TASK-P04-017)

| Task ID | Komponen | Judul Pekerjaan | Assignee | Status | Target Selesai | Catatan / Verifikasi |
|---|---|---|---|:---:|---|---|
| **TASK-P04-001** | Database | Skema Migrasi DDL `0007_subscription_token_core.sql` | Backend Core | ✅ | 2026-10-19 | Skema tabel lengkap, integer moneter & kuota, DDL replayable |
| **TASK-P04-002** | Database | Seeder Default Plans (Free/Pro/Business) & Pricing Rules | Backend Core | ✅ | 2026-10-20 | Seeder idempoten terverifikasi dengan batasan kuota default |
| **TASK-P04-003** | Billing | Domain Entities, Value Objects & Domain Errors | Backend Core | 🔄 | 2026-10-20 | Struct enum status langganan dan ledger types dalam penulisan |
| **TASK-P04-004** | Billing | PostgreSQL Repository Layer dengan Row Locking | Backend Core | ⬜ | 2026-10-21 | Implementasi query `SELECT ... FOR UPDATE` |
| **TASK-P04-005** | Billing | Token Ledger Service — Reserve Flow & Quota Check | Backend Core | ⬜ | 2026-10-22 | Alur validasi kuota siklus & isolasi reservasi |
| **TASK-P04-006** | Billing | Token Ledger Service — Commit & Release Flow | Backend Core | ⬜ | 2026-10-23 | Integrasi callback pasca-eksekusi AI Gateway |
| **TASK-P04-007** | Billing | Background Cleaner Worker (Auto-Release Timeout) | Backend Worker | ⬜ | 2026-10-23 | Pembersih reservasi kedaluwarsa interval 15 menit |
| **TASK-P04-008** | Billing | Dynamic Pricing Engine Calculator (Integer IDR) | Backend Core | ⬜ | 2026-10-26 | Formula harga dinamis IDR dengan bobot modalitas |
| **TASK-P04-009** | Entitlement | Domain Model & Aggregation Repository Resolver | Backend Core | ⬜ | 2026-10-26 | Resolver fitur boolean dan limit kuota numerik |
| **TASK-P04-010** | Entitlement | Versioned Cache Manager L1 (In-Memory) & L2 (Redis) | Backend Core | ⬜ | 2026-10-27 | Pola cache key `entitlement:u:{uid}:v:{ver}` |
| **TASK-P04-011** | Entitlement | Redis Pub/Sub Subscriber Invalidation Lintas Node | Backend Core | ⬜ | 2026-10-27 | Sinkronisasi pembersihan L1 lokal < 50ms |
| **TASK-P04-012** | Entitlement | Gin Middleware `RequireFeature` & `WithinLimit` | Backend Core | ⬜ | 2026-10-28 | HTTP guard dengan bypass super_admin |
| **TASK-P04-013** | Entitlement | REST API Endpoint `GET /api/v1/me/entitlements` | Backend Core | ⬜ | 2026-10-28 | Penyajian DTO untuk TanStack Query frontend |
| **TASK-P04-014** | Billing | Subscription Lifecycle State Machine & Audit Events | Backend Core | ⬜ | 2026-10-29 | Transisi status langganan dan log `subscription_events` |
| **TASK-P04-015** | Billing | Upgrade Prorata Instan & Downgrade Soft-Limit Logic | Backend Core | ⬜ | 2026-10-29 | Kalkulasi kredit sisa hari dan proteksi data user |
| **TASK-P04-016** | QA / Test | Concurrency Testing & Anti Double-Spending Check | QA Engineer | ⬜ | 2026-10-30 | Pengujian goroutines paralel anti-race condition |
| **TASK-P04-017** | Observability | Prometheus Metrics, Tracing & Handoff Kontrak P05/P06 | DevOps / Lead | ⬜ | 2026-10-30 | Metrik latensi cache, gauge langganan, interface handoff |

---

## Log Pembaruan Terkini

- **2026-10-19:**
  - Desain skema basis data dan penyusunan migrasi DDL selesai (`TASK-P04-001`). Seluruh tabel relasional, constraint, dan index terdefinisi.
  - Seeder bawaan untuk paket Free, Pro, Business dan konfigurasi pricing awal terpasang (`TASK-P04-002`).
- **2026-10-20:**
  - Pengerjaan domain entity dan value objects pada modul `internal/billing/domain/` dimulai (`TASK-P04-003`).

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
