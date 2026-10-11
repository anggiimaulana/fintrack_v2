# Lembar Tracking Tugas & Status Deliverable: Fase P07

Status Legend:
- ⬜ : Belum Dimulai (Not Started)
- 🔄 : Sedang Dikerjakan (In Progress)
- ✅ : Selesai (Completed)

> **Status saat ini: SLICE A + B + C + D1 SELESAI (17/18).** Sisa 1 task: **TASK-P07-013 (BYOK) = D2**. (Tanda ✅ pada tabel
> "Status Deliverable" di bagian bawah hanya menandai bahwa **dokumen** sudah ditulis — bukan kode.)
>
> **Slice A — Entitlement & RBAC Rollout (2026-10-11)** mencakup TASK-P07-001 & TASK-P07-002
> plus mirror frontend. Lihat `docs/superpowers/specs/2026-10-11-p07-entitlement-rollout-design.md`
> dan `docs/superpowers/plans/2026-10-11-p07-entitlement-rollout-plan.md`.
>
> **Slice B — Realtime SSE & Cache Invalidation (2026-10-11)** mencakup TASK-P07-004 … TASK-P07-009
> plus perbaikan tiga cacat invalidasi entitlement. Lihat
> `docs/superpowers/specs/2026-10-11-p07-realtime-sse-design.md` dan
> `docs/superpowers/plans/2026-10-11-p07-realtime-sse-plan.md`.
>
> **Slice C — Standar UI, Aksesibilitas & Optimistic Update (2026-10-11)** mencakup
> TASK-P07-011, 012, 014, 015 (013 → slice D). Lihat
> `docs/superpowers/specs/2026-10-11-p07-ui-standards-a11y-design.md` dan
> `docs/superpowers/plans/2026-10-11-p07-ui-standards-a11y-plan.md`.
>
> **Slice D1 — Health Gates, Shutdown & Aturan Migrasi (2026-10-11)** mencakup TASK-P07-016, 017,
> 018 + verifikasi 003. Lihat `docs/superpowers/specs/2026-10-11-p07-health-shutdown-migrations-design.md`
> dan `docs/superpowers/plans/2026-10-11-p07-health-shutdown-migrations-plan.md`.
> **BYOK (TASK-P07-013) ditunda ke D2** — lihat catatan di bawah.

---

## 1. Status Task WBS (Work Breakdown Structure)

| Kode Task | Nama Task & Deskripsi | PIC | Target Selesai | Status | Catatan / Bukti Verifikasi |
|---|---|---|---|:---:|---|
| **TASK-P07-001** | Middleware Entitlement & Limit di Backend Go | Backend Dev 1 | H-01 | ✅ | **Slice A.** Middleware sudah ada sejak P04 tetapi **tidak pernah dikonstruksi** → 0/63 rute ter-gate. Kini dikonstruksi di `cmd/server/main.go`, disuntikkan via `RouterConfig`, plus `upgrade_url` + valve `ENTITLEMENT_ENFORCEMENT` + 3 usage resolver. Bukti: `middleware/entitlement_test.go` (+6 test), `tests/entitlement_e2e_test.go`. |
| **TASK-P07-002** | Pasang Proteksi Entitlement ke 28 Endpoint Legacy | Backend Dev 1 & 2 | H-02 | ✅ | **Slice A.** 6 rute nyata ter-gate (bukan 28 — dokumen lama menyebut endpoint yang belum dibangun). Rute baca dibiarkan. 7 kunci seed belum dapat ditegakkan (tercatat di `teknis/rbac-rollout.md` §3.4). Uji mutasi membuktikan suite gagal bila gate dilepas. |
| **TASK-P07-003** | Migrasi globalCache In-Memory ke Redis Berversi | Backend Dev 2 | H-01 | ✅ | **Slice D1 (terverifikasi).** AC terpenuhi **tanpa kode baru**: grep `sync.Map` & `globalCache` di seluruh `backend/` (non-test) = **nol hasil**; cache sudah Redis berversi sejak P04 (`CacheEngine.GetVersion`/`BumpVersion`/`BumpUserVersion`, dipakai resolver entitlement dengan scope `ent:{user}` dan `ent:plan:{id}`). Tidak ada pekerjaan tersisa. |
| **TASK-P07-004** | Modul Hub SSE `internal/realtime/` | Backend Dev 3 | H-02 | ✅ | **Slice B.** Modul `internal/realtime/` (hub, client, buffer, publisher, subscriber, handler). Endpoint `GET /api/v1/realtime/stream`, auth cookie `ft_at`, header SSE benar, batas 5 koneksi/user. |
| **TASK-P07-005** | Jembatan Redis Pub/Sub Distribusi SSE Multi-Pod | Backend Dev 3 | H-03 | ✅ | **Slice B.** Fan-out Redis: channel `realtime:u:{id}` + `realtime:admins`; `SubscribePattern` baru di pubsub; degradasi ke hub lokal bila Redis mati. |
| **TASK-P07-006** | Resiliensi SSE: Cookie Auth, 15s Heartbeat, Resume | Backend Dev 3 | H-04 | ✅ | **Slice B.** Heartbeat `: ping` 15s, resume `Last-Event-ID` (header atau query `last_event_id`), ring-buffer Redis `realtime:buf:{id}` TTL 10 menit. **Temuan:** header tidak di-flush saat connect → klien menunggu 15s; diperbaiki, waktu test integrasi turun 91.7s → 2.9s. |
| **TASK-P07-007** | Standardisasi Skema JSON Event Domain SSE | Backend Lead | H-04 | ✅ | **Slice B.** Kontrak di `pkg/contracts/events.go` (paket daun, tanpa impor internal) + 5 payload. Producer nyata: payment webhook/approve, upload bukti, perubahan langganan/plan, siklus token, breaker gateway. |
| **TASK-P07-008** | Frontend Hook `useRealtimeStream` & Auto-Reconnect | Frontend Dev 3 | H-06 | ✅ | **Slice B.** `hooks/use-realtime-stream.ts` + `providers/realtime-provider.tsx`. Backoff manual 1s→16s + jitter; `last_event_id` sebagai query param karena EventSource yang dibuat ulang tidak mengirim header otomatis. |
| **TASK-P07-009** | Registri Query Key & Surgical Cache Invalidation | Frontend Lead | H-06 | ✅ | **Slice B.** `lib/realtime-dispatcher.ts` memetakan event ke query key NYATA (dokumen memakai `billing.*` yang tidak ada). Test memastikan tidak ada `invalidateQueries` tanpa key. |
| **TASK-P07-010** | Optimistic Updates Mutasi Transaksi & Anggaran | Frontend Dev 1 & 2 | H-07 | ✅ | **Slice C (dengan catatan).** Optimistic update untuk **update & delete** transaksi (`onMutate` snapshot semua halaman list, patch/remove, `onError` rollback, `onSettled` invalidate); rollback menangani 409, 402/403 (via `notifyEntitlementError`), dan generik. **Create sengaja TIDAK optimistik**: list dipaginasi + di-sort server-side sehingga posisi baris baru tidak diketahui. **Halaman anggaran (`/budgets`) tidak ada** di codebase, jadi bagian "Anggaran" pada task ini tidak berlaku. Uji mutasi: `restoreLists` dilepas → 3 test rollback gagal → terbukti terjaga. |
| **TASK-P07-011** | Standardisasi Komponen `<DataTable>` TanStack v8 | Frontend Dev 1 | H-01 | ✅ | **Slice C.** `<DataTable>` dipakai di `/user/transactions` dengan state di URL (`useDataTableUrlState`): page/sort/order/q bookmarkable. Debounce 300ms dari `DataTableToolbar` (sebelumnya search terkirim per ketikan). |
| **TASK-P07-012** | Refactor Seluruh Halaman List ke `<DataTable>` | Frontend Dev 1 & 2 | H-04 | ✅ | **Slice C (sebagian).** Hanya `/user/transactions` yang dikerjakan: `/budgets` dan `/categories` (user) **tidak ada** di codebase — kategori dikelola admin. Refactor sekaligus menghapus komponen baris custom (kode mati). |
| **TASK-P07-013** | Pembersihan Tuntas Menu & Kode Legacy BYOK | Fullstack Dev | H-03 | ⬜ | Hapus form/halaman AI keys, arsip DB tabel |
| **TASK-P07-014** | Audit Universal Cursor Pointer & Styling Hover | Frontend Dev 3 | H-07 | ✅ | **Slice C.** Audit menyeluruh: akar masalah di komponen dasar (`Button` tanpa cursor sama sekali + `disabled:pointer-events-none` yang membuat `cursor-not-allowed` mustahil; `SelectTrigger` juga). 79 elemen native diperbaiki; verifikasi akhir 0 tersisa. Laporan di `teknis/cursor-pointer-audit.md`. |
| **TASK-P07-015** | Audit Aksesibilitas WCAG 2.1 AA & Keyboard Nav | Frontend Dev 3 & QA | H-08 | ✅ | **Slice C (dengan catatan).** `eslint-plugin-jsx-a11y` aktif (40 pelanggaran nyata diperbaiki, bukan aturan dimatikan) + axe-core via Vitest untuk komponen kunci (nol pelanggaran). Kontras dihitung manual dan 4 temuan diperbaiki dengan nilai terverifikasi. **AC "Lighthouse ≥ 95" TIDAK terpenuhi** — Lighthouse tidak ada di repo; ditandai gap, bukan diklaim. |
| **TASK-P07-016** | Dual Health Gate Endpoint (`/health/live` & `/ready`) | Backend Dev 1 | H-06 | ✅ | **Slice D1.** `/health/live` (tanpa I/O), `/health/ready` (PostgreSQL + pgvector + Redis dengan latensi per dependensi), `/health` alias live. DB mati → **503** (hard dependency); Redis mati → **200 degraded** (Redis opsional di codebase ini). Bukti: 5 test + verifikasi terhadap DB/Redis nyata (`postgres ok 2ms, redis ok 1ms`). |
| **TASK-P07-017** | Graceful Shutdown 30s & Zero-Downtime Rolling Test | DevOps Engineer | H-07 | ✅ | **Slice D1.** Drain `10s → 30s` via `SHUTDOWN_TIMEOUT` (default 30s, divalidasi test). Compose: healthcheck `backend` memakai `/health/ready` + `stop_grace_period: 30s`; perintah healthcheck diuji langsung di container (wget tersedia, endpoint 200). **Gap:** template k8s tidak dibuat (`deploy/` tidak ada di repo) dan uji rolling restart 200 RPS tanpa 5xx tidak dijalankan. |
| **TASK-P07-018** | Template & Prosedur Migrasi Skema Expand/Contract | Backend Lead & DBA | H-08 | ✅ | **Slice D1.** `database/migrations/README.md` baru: larangan `DROP COLUMN` langsung / `SET NOT NULL` tanpa backfill / `ADD COLUMN NOT NULL` tanpa default / `CREATE INDEX` blocking / mengedit migrasi lama; wajib aditif+idempoten, `CREATE INDEX CONCURRENTLY`, `NOT VALID` + `VALIDATE CONSTRAINT`, dan pola expand→migrate→contract 3 rilis untuk rename kolom. |

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

- **2026-10-11 — Slice B (Realtime SSE & Cache Invalidation) SELESAI.** TASK-P07-004 … TASK-P07-009 ✅.
  - **Transport:** modul `internal/realtime/` (hub, client, buffer, publisher, subscriber, handler).
    Endpoint `GET /api/v1/realtime/stream` di luar grup `cached`; auth memakai `AuthMiddleware`
    yang sudah ada (cookie `ft_at`). Heartbeat 15s; resume `Last-Event-ID` (header atau query).
  - **Fan-out:** channel `realtime:u:{userID}` untuk event user dan `realtime:admins` untuk event
    admin. Tanpa Redis, publish jatuh ke hub lokal (single-instance tetap jalan).
  - **Lima producer nyata:** `payment_received` (webhook settled + approve manual),
    `manual_approval_needed` (upload bukti), `plan_updated` (perubahan langganan/plan),
    `quota_reset` (periode token baru dibuat), `ai_endpoint_down` (circuit breaker terbuka).
  - **Frontend:** dispatcher murni + hook `EventSource` dengan backoff manual 1s→16s + jitter +
    provider yang memasang stream hanya saat terautentikasi.
  - **Bug yang ditemukan & diperbaiki:**
    1. **Header SSE tidak di-flush saat connect** → klien menunggu sampai heartbeat pertama
       (15s) sebelum melihat stream terbuka; `onopen` browser juga tidak menyala. Terdeteksi dari
       durasi test integrasi yang kelipatan tepat 15s (91.7s → 2.9s setelah perbaikan).
    2. **Tiga cacat invalidasi entitlement** (4a/4b/4c) — lihat `teknis/cache-invalidation.md` §7.2.
       Yang paling berdampak: mengubah fitur sebuah plan **tidak menginvalidasi apa pun**, jadi
       semua subscriber memakai entitlement basi sampai 24 jam.
  - **Bukti:** `go vet`/`go test`/`go build` hijau; `tests/realtime_e2e_test.go` 6 skenario lewat
    router + Redis nyata (auth 401, disabled 503, delivery, isolasi antar user, event admin tidak
    bocor ke user biasa, resume backlog); uji mutasi membuktikan test invalidasi plan menangkap
    regresi. Frontend: `pnpm lint` 0 error, `pnpm build` hijau, `pnpm test` 81 lolos (dari 66).
  - **Gap yang dicatat (bukan dikerjakan di sini):** reset kuota tidak terjadwal (lazy saat
    pemakaian AI berikutnya — tidak ada worker rotasi siklus); tidak ada inbox notifikasi durabel.
  - **Valve:** `REALTIME_ENABLED` (default on). Batas 5 koneksi/user.

- **2026-10-11 — Slice C (Standar UI, Aksesibilitas & Optimistic Update) SELESAI.** TASK-P07-011/012/014/015 ✅.
  - **a11y (C-1):** `eslint-plugin-jsx-a11y` aktif di `pnpm lint`. **40 pelanggaran nyata
    diperbaiki, bukan aturan dimatikan:** 23 label tanpa asosiasi kontrol, 6 klik pada elemen
    statis, 6 handler tanpa keyboard listener, 4 `autoFocus`, 1 `aria-role` invalid. Perbaikan
    substantif: label dihubungkan `id`/`htmlFor`; dropzone struk & tombol mic dijadikan `<button>`
    asli (bisa diaktifkan keyboard); backdrop dialog dijadikan elemen saudara sehingga
    `stopPropagation` tidak diperlukan; prop `role` pada `ChatMessage` diganti `author` (agar tidak
    dibaca sebagai atribut ARIA); `autoFocus` diganti fokus programatik via ref.
  - **Kontras (C-2):** dihitung manual dari token (axe tidak menguji kontras). **4 temuan nyata**
    diperbaiki dengan nilai terverifikasi: `--muted-foreground` light 0.55→0.53 (4.19→4.56);
    token baru `--primary-strong` untuk teks di atas `bg-primary/10` (4.27→5.22) dipakai di 5
    tempat; `red-600`→`red-700` di `red-100` (3.95→5.30); `green-600` di `green-100` (**3.00**)
    → `green-700` (4.57); `red-600` di `red-600/15` (3.82→5.01).
    **Koreksi jujur:** klaim awal bahwa badge Scan gagal (3.07) **SALAH** — kode aslinya
    `amber-700` di `amber-100` = 4.51 (lolos); perubahan yang tidak perlu dibatalkan.
  - **Cursor (C-4):** akarnya komponen dasar — `Button` tidak punya `cursor-pointer` sama sekali
    dan memakai `disabled:pointer-events-none` (membuat `cursor-not-allowed` mustahil);
    `SelectTrigger` juga. Diperbaiki di komponen dasar + 79 elemen native. Verifikasi akhir:
    **0 tersisa**.
  - **Tabel (C-7/C-8):** `/user/transactions` memakai `<DataTable>` dengan state di URL
    (`page`/`sort`/`order`/`q` bookmarkable) dan debounce 300ms (sebelumnya search terkirim tiap
    ketikan). Komponen baris custom dihapus (kode mati) — warning lint turun 67→54, di bawah
    baseline 62.
  - **Optimistic update (C-9/C-10):** update & delete optimistik dengan rollback; **create sengaja
    tidak optimistik** (list dipaginasi + di-sort server-side → posisi baris baru tidak diketahui).
    Rollback menangani 409, 402/403 (via `notifyEntitlementError`), dan error generik.
  - **Bukti:** `pnpm lint` 0 error, `pnpm build` hijau, `pnpm test` **102 lolos** (dari 66);
    nol pelanggaran axe untuk ConfirmDialog, TransactionRow (semua sumber), DataTable,
    DataTableToolbar. **Uji mutasi** pada rollback: `restoreLists` dilepas → 3 test gagal
    ("rolls back to the previous snapshot on a 409 conflict", dst) → terbukti menangkap regresi.
  - **GAP yang dicatat, tidak diklaim:**
    1. **AC-015 "Lighthouse Accessibility ≥ 95" TIDAK terpenuhi** — Lighthouse tidak ada di repo
       (butuh server + browser). Digantikan: jsx-a11y + axe + perhitungan kontras manual.
    2. **AC-010 "0ms saat menambah" tidak sepenuhnya terpenuhi** — create sengaja non-optimistik.
    3. **TASK-P07-012 hanya sebagian** — `/budgets` & `/categories` (user) tidak ada.

- **2026-10-11 — Slice D1 (Health Gates, Shutdown & Aturan Migrasi) SELESAI.** TASK-P07-016/017/018 ✅,
  TASK-P07-003 ✅ (terverifikasi).
  - **Temuan besar: gateway AI terpusat TIDAK PERNAH aktif.** `IsEnabled(ctx, key, nil)`
    mengembalikan false untuk rollout parsial, dan flag bernilai `enabled=true rollout=25` →
    selalu false (log runtime: `"AI gateway centralized: disabled"`). Akibatnya **seluruh P03
    tidak berjalan** — 100% traffic AI lewat BYOK per-user. Flag itu sendiri **artefak uji**
    (di-toggle via `curl` saat pengujian P06).
  - **Koreksi klaim saya sendiri:** saya sempat melaporkan `ai_routes` kosong. **SALAH** — berisi
    **5 route aktif** (chat, vision, voice, embedding, reasoning) menunjuk `9router-primary`.
    Output sebelumnya terpotong dan saya salah baca. Dikoreksi setelah memverifikasi ulang.
  - **Konsekuensi & pengaman:** karena route chat ADA, pengaman berbasis-route saja akan
    **membuka** gateway; dan `chatViaGateway` **tidak punya fallback ke BYOK**. Jadi pengaman
    diperkuat: gateway hanya aktif bila flag ON **DAN** ada route chat **DAN** endpoint sehat
    (`health == "healthy"`; `unknown` tidak dihitung sehat).
    **Bukti runtime:** `WARN AI gateway: flag is ON and a chat route exists, but no endpoint is
    healthy; gateway stays disabled (BYOK fallback in use)` → gateway mati, AI tetap jalan.
    Gateway akan terbuka sendiri begitu health checker melaporkan endpoint sehat.
  - **Perbaikan semantik flag:** `IsEnabledForPlatform` (kill switch platform, tanpa bucketing
    user) — flag `enabled=true rollout=25` kini terbaca ON.
  - **Health gates:** `/health/live` (tanpa I/O), `/health/ready` (PostgreSQL + pgvector + Redis
    dengan latensi), `/health` alias. DB mati → 503; Redis mati → 200 `degraded`. Diverifikasi
    terhadap DB/Redis nyata: `postgres ok 2ms, redis ok 1ms`.
  - **Shutdown:** drain 30s via `SHUTDOWN_TIMEOUT`; compose healthcheck + `stop_grace_period: 30s`
    (perintah healthcheck diuji langsung di container).
  - **Migrasi:** `database/migrations/README.md` memuat aturan expand/contract.
  - **TASK-P07-003 terverifikasi terpenuhi tanpa kode baru:** grep `sync.Map`/`globalCache` di
    `backend/` (non-test) nol hasil.
  - **Bukti:** `go vet`/`go build`/`go test ./...` hijau; suite integrasi (P06 + P07 A/B) hijau;
    uji mutasi pengaman membuktikan test menangkap regresi (tanpa cek kesehatan, `unhealthy` dan
    `unknown` lolos).
  - **GAP yang dicatat, tidak diklaim:**
    1. **Gateway masih mati** setelah D1 (endpoint `unhealthy`) → "P03 tidak berjalan" **belum
       terselesaikan**; butuh endpoint sehat. Ini temuan **prioritas tinggi**.
    2. Template k8s tidak dibuat (`deploy/` tidak ada di repo).
    3. Uji rolling restart 200 RPS tanpa 5xx tidak dijalankan (uji beban terkoordinasi).
    4. **BYOK belum dihapus** → slice D2, dengan syarat gateway benar-benar melayani semua user.
