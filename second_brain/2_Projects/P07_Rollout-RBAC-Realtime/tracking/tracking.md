# Lembar Tracking Tugas & Status Deliverable: Fase P07

Status Legend:
- ⬜ : Belum Dimulai (Not Started)
- 🔄 : Sedang Dikerjakan (In Progress)
- ✅ : Selesai (Completed)

---

## 1. Status Task WBS (Work Breakdown Structure)

| Kode Task | Nama Task & Deskripsi | PIC | Target Selesai | Status | Catatan / Bukti Verifikasi |
|---|---|---|---|:---:|---|
| **TASK-P07-001** | Middleware Entitlement & Limit di Backend Go | Backend Dev 1 | H-01 | ⬜ | Siap implementasi di `internal/middleware/entitlement.go` |
| **TASK-P07-002** | Pasang Proteksi Entitlement ke 28 Endpoint Legacy | Backend Dev 1 & 2 | H-02 | ⬜ | Membutuhkan penyelesaian TASK-P07-001 |
| **TASK-P07-003** | Migrasi globalCache In-Memory ke Redis Berversi | Backend Dev 2 | H-01 | ⬜ | Arsitektur namespace `ft:{env}:{module}:{entity}:{id}` |
| **TASK-P07-004** | Modul Hub SSE `internal/realtime/` | Backend Dev 3 | H-02 | ⬜ | Handler streaming HTTP/2 SSE |
| **TASK-P07-005** | Jembatan Redis Pub/Sub Distribusi SSE Multi-Pod | Backend Dev 3 | H-03 | ⬜ | Channel `realtime:events:{channel}` |
| **TASK-P07-006** | Resiliensi SSE: Cookie Auth, 15s Heartbeat, Resume | Backend Dev 3 | H-04 | ⬜ | Validasi ring-buffer redis & `Last-Event-ID` |
| **TASK-P07-007** | Standardisasi Skema JSON Event Domain SSE | Backend Lead | H-04 | ⬜ | Kontrak `contracts/events.go` |
| **TASK-P07-008** | Frontend Hook `useRealtimeStream` & Auto-Reconnect | Frontend Dev 3 | H-06 | ⬜ | Exponential backoff jitter & cookie auth |
| **TASK-P07-009** | Registri Query Key & Surgical Cache Invalidation | Frontend Lead | H-06 | ⬜ | Mengeliminasi invalidasi global tanpa filter |
| **TASK-P07-010** | Optimistic Updates Mutasi Transaksi & Anggaran | Frontend Dev 1 & 2 | H-07 | ⬜ | Rollback rapi saat terjadi error 409/429 |
| **TASK-P07-011** | Standardisasi Komponen `<DataTable>` TanStack v8 | Frontend Dev 1 | H-01 | ⬜ | Sinkronisasi dua arah ke URL search params |
| **TASK-P07-012** | Refactor Seluruh Halaman List ke `<DataTable>` | Frontend Dev 1 & 2 | H-04 | ⬜ | Halaman transaksi, anggaran, kategori |
| **TASK-P07-013** | Pembersihan Tuntas Menu & Kode Legacy BYOK | Fullstack Dev | H-03 | ⬜ | Hapus form/halaman AI keys, arsip DB tabel |
| **TASK-P07-014** | Audit Universal Cursor Pointer & Styling Hover | Frontend Dev 3 | H-07 | ⬜ | Penerapan kelas `cursor-pointer` di seluruh elemen |
| **TASK-P07-015** | Audit Aksesibilitas WCAG 2.1 AA & Keyboard Nav | Frontend Dev 3 & QA | H-08 | ⬜ | Target skor Lighthouse Accessibility >= 95 |
| **TASK-P07-016** | Dual Health Gate Endpoint (`/health/live` & `/ready`) | Backend Dev 1 | H-06 | ⬜ | Cek database postgres + redis connectivity |
| **TASK-P07-017** | Graceful Shutdown 30s & Zero-Downtime Rolling Test | DevOps Engineer | H-07 | ⬜ | Pengujian rolling deploy tanpa 5xx pada 200 RPS |
| **TASK-P07-018** | Template & Prosedur Migrasi Skema Expand/Contract | Backend Lead & DBA | H-08 | ⬜ | Standardisasi non-blocking migrations |

---

## 2. Status Kelengkapan Dokumen Desain & Teknis P07

| Berkas Dokumen | Deskripsi Isi Dokumen | Status | Lokasi Penyimpanan |
|---|---|:---:|---|
| **README.md** | Context, Ruang Lingkup, Deliverables, Kriteria Sukses | ✅ | `P07_Rollout-RBAC-Realtime/README.md` |
| **analisis/hasil-analisis-teknis.md** | Keputusan teknis arsitektur P07 (K-P07-01 s/d K-P07-16) | ✅ | `P07_Rollout-RBAC-Realtime/analisis/hasil-analisis-teknis.md` |
| **planning/sprint-plan.md** | Rencana sprint 2 minggu, milestone, DOD | ✅ | `P07_Rollout-RBAC-Realtime/planning/sprint-plan.md` |
| **planning/tasks.md** | Detail breakdown 18 task backlog WBS lengkap | ✅ | `P07_Rollout-RBAC-Realtime/planning/tasks.md` |
| **planning/timeline.md** | Jadwal eksekusi 10 hari & analisis critical path | ✅ | `P07_Rollout-RBAC-Realtime/planning/timeline.md` |
| **rules/implementation-rules.md** | 18 aturan teknis mengikat tim engineering | ✅ | `P07_Rollout-RBAC-Realtime/rules/implementation-rules.md` |
| **tracking/tracking.md** | Lembar pelacakan status realtime & deliverable | ✅ | `P07_Rollout-RBAC-Realtime/tracking/tracking.md` |
| **teknis/rbac-rollout.md** | Detail matriks otorisasi & audit 28 endpoint | ✅ | `P07_Rollout-RBAC-Realtime/teknis/rbac-rollout.md` |
| **teknis/sse-strategy.md** | Arsitektur SSE, reconnect, heartbeat & domain events | ✅ | `P07_Rollout-RBAC-Realtime/teknis/sse-strategy.md` |
| **teknis/cache-invalidation.md** | Redis pub/sub & surgical TanStack Query keys | ✅ | `P07_Rollout-RBAC-Realtime/teknis/cache-invalidation.md` |
| **teknis/cursor-pointer-audit.md** | Laporan audit interaktivitas UI & cursor pointer | ✅ | `P07_Rollout-RBAC-Realtime/teknis/cursor-pointer-audit.md` |
| **teknis/zero-downtime-deploy.md** | Prosedur expand/contract, rolling deploy, health gate | ✅ | `P07_Rollout-RBAC-Realtime/teknis/zero-downtime-deploy.md` |
| **teknis/frontend-entitlement-mirror.md** | Cermin UX entitlement klien, guard hooks, UX fallback | ✅ | `P07_Rollout-RBAC-Realtime/teknis/frontend-entitlement-mirror.md` |
