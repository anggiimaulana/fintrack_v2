# Lembar Tracking Tugas & Status Deliverable: Fase P07

Status Legend:
- ⬜ : Belum Dimulai (Not Started)
- 🔄 : Sedang Dikerjakan (In Progress)
- ✅ : Selesai (Completed)

> **Status saat ini: SLICE A SELESAI (2/18).** Sisa 16 task masih ⬜. (Tanda ✅ pada tabel
> "Status Deliverable" di bagian bawah hanya menandai bahwa **dokumen** sudah ditulis — bukan kode.)
>
> **Slice A — Entitlement & RBAC Rollout (2026-10-11)** mencakup TASK-P07-001 & TASK-P07-002
> plus mirror frontend. Lihat `docs/superpowers/specs/2026-10-11-p07-entitlement-rollout-design.md`
> dan `docs/superpowers/plans/2026-10-11-p07-entitlement-rollout-plan.md`.

---

## 1. Status Task WBS (Work Breakdown Structure)

| Kode Task | Nama Task & Deskripsi | PIC | Target Selesai | Status | Catatan / Bukti Verifikasi |
|---|---|---|---|:---:|---|
| **TASK-P07-001** | Middleware Entitlement & Limit di Backend Go | Backend Dev 1 | H-01 | ✅ | **Slice A.** Middleware sudah ada sejak P04 tetapi **tidak pernah dikonstruksi** → 0/63 rute ter-gate. Kini dikonstruksi di `cmd/server/main.go`, disuntikkan via `RouterConfig`, plus `upgrade_url` + valve `ENTITLEMENT_ENFORCEMENT` + 3 usage resolver. Bukti: `middleware/entitlement_test.go` (+6 test), `tests/entitlement_e2e_test.go`. |
| **TASK-P07-002** | Pasang Proteksi Entitlement ke 28 Endpoint Legacy | Backend Dev 1 & 2 | H-02 | ✅ | **Slice A.** 6 rute nyata ter-gate (bukan 28 — dokumen lama menyebut endpoint yang belum dibangun). Rute baca dibiarkan. 7 kunci seed belum dapat ditegakkan (tercatat di `teknis/rbac-rollout.md` §3.4). Uji mutasi membuktikan suite gagal bila gate dilepas. |
| **TASK-P07-003** | Migrasi globalCache In-Memory ke Redis Berversi | Backend Dev 2 | H-01 | ⬜ | Arsitektur namespace `ft:{env}:{module}:{entity}:{id}` |
| **TASK-P07-004** | Modul Hub SSE `internal/realtime/` | Backend Dev 3 | H-02 | ⬜ | Handler streaming HTTP/2 SSE |
| **TASK-P07-005** | Jembatan Redis Pub/Sub Distribusi SSE Multi-Pod | Backend Dev 3 | H-03 | ⬜ | Channel `realtime:events:{channel}` |
| **TASK-P07-006** | Resiliensi SSE: Cookie Auth, 15s Heartbeat, Resume | Backend Dev 3 | H-04 | ⬜ | Validasi ring-buffer redis & `Last-Event-ID` |
| **TASK-P07-007** | Standardisasi Skema JSON Event Domain SSE | Backend Lead | H-04 | ⬜ | Kontrak `contracts/events.go` |
| **TASK-P07-008** | Frontend Hook `useRealtimeStream` & Auto-Reconnect | Frontend Dev 3 | H-06 | ⬜ | Exponential backoff jitter & cookie auth |
| **TASK-P07-009** | Registri Query Key & Surgical Cache Invalidation | Frontend Lead | H-06 | ⬜ | Mengeliminasi invalidasi global tanpa filter |
| **TASK-P07-010** | Optimistic Updates Mutasi Transaksi & Anggaran | Frontend Dev 1 & 2 | H-07 | ⬜ | Rollback rapi saat terjadi error 409/429. **Sebagian oleh Slice A**: handler error 402/403 entitlement terpusat (`lib/entitlement-errors.ts`) + toast upgrade; optimistic update penuh belum. |
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

---

## 3. Log Pembaruan Status

- **2026-10-11 — Slice A (Entitlement & RBAC Rollout) SELESAI.** TASK-P07-001 & TASK-P07-002 ✅.
  - **Temuan utama:** middleware entitlement sudah ada dan bertest sejak P04, tetapi
    `NewEntitlementMiddleware` **tidak pernah dikonstruksi** — 0 dari 63 rute ter-gate. Semua
    endpoint legacy hanya dilindungi autentikasi (`RequireUser`), bukan otorisasi paket.
  - **Blocker yang diperbaiki:** resolver mengembalikan deny-all untuk user tanpa baris
    `subscriptions` (52 user vs 2 subscription di DB dev) → menyalakan enforcement akan mengunci
    ~96% user. Kini fallback ke plan `free` (`DEFAULT_PLAN_CODE`), tetap fail-closed bila plan
    `free` hilang.
  - **Dua bug tambahan yang ditemukan & diperbaiki:**
    1. `RequireUser` **menolak `super_admin`** → pemilik platform terkunci dari seluruh rute user
       (transaksi, dashboard, AI, laporan). Kini menerima `user`/`admin`/`super_admin`.
    2. `GetActivePlanForUser` membaca baris subscription **tanpa memfilter status** → langganan
       `expired`/`canceled` tetap mendapat fitur berbayar selamanya (eskalasi hak akses begitu
       enforcement menyala). Kini `canceled`/`expired` jatuh ke plan default.
  - **Ditegakkan (6 rute):** `POST /transactions` (`max_transactions`), `POST /ai/chat`
    (`ai_chat` + `ai_tokens_per_cycle`), `POST /ai/receipt` (`ocr_receipt` +
    `ocr_scans_per_cycle`), `POST /ai/receipt/transaction` (`ocr_receipt`), `POST /ai/voice`
    (`voice_input`), `GET /reports` (`reports_export`). Rute baca tidak digate.
  - **Kontrak blok:** 403 `feature_not_in_plan`; 402 `limit_exceeded` / `limit_not_in_plan`.
    Keduanya memuat `upgrade_url`. 402 dipertahankan (bukan 429) agar tidak ambigu dengan rate
    limiter; dokumen dikoreksi.
  - **Frontend mirror (advisory-only):** `useEntitlements`, `useRole`, halaman read-only
    `/user/settings/subscription`, helper `lib/entitlement-errors.ts`, tombol OCR/export
    dinonaktifkan sesuai paket.
  - **Valve:** `ENTITLEMENT_ENFORCEMENT` (default on) — bypass insiden saja.
  - **Bukti:** `go vet`/`go test`/`go build` hijau; `pnpm lint` 0 error, `pnpm build` hijau,
    `pnpm test` 66 lolos (dari 54). `tests/entitlement_e2e_test.go` 8 skenario lewat router nyata;
    uji mutasi (gate dilepas) → suite gagal seperti diharapkan.
  - **Belum ditegakkan:** `ai_requests_per_cycle`, `max_categories`, `max_wallets`, `rag_search`,
    `advanced_budgeting`, `api_access`, `bank_sync` — rincian alasan di `teknis/rbac-rollout.md` §3.4.
  - **Efek yang perlu diketahui:** user dengan transaksi sudah di atas batas Free (mis. user demo
    479 vs 100) akan ditolak pada `POST /transactions`; rute baca tetap terbuka. Ini enforcement
    yang benar, valve tersedia bila mengganggu.
