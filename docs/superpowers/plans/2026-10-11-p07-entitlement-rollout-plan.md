# Implementation Plan — P07 Slice A: Entitlement & RBAC Rollout

| Field | Isi |
|---|---|
| **Fase** | P07 — Rollout RBAC, Realtime & Standar (**Slice A** dari 5) |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | P06 selesai (commit `c40baef`); migrasi 0001–0010 diterapkan |
| **Spec desain** | `docs/superpowers/specs/2026-10-11-p07-entitlement-rollout-design.md` |
| **Task terkait** | `TASK-P07-001`, `TASK-P07-002` (+ mirror frontend dari `frontend-entitlement-mirror.md`) |
| **Status** | Siap dieksekusi — belum ada kode ditulis |

> **Catatan path:** kode backend berada di `backend/internal/delivery/http/middleware/`
> (bukan `backend/internal/middleware/`). Router: `backend/internal/delivery/http/router.go`.
> Frontend: `frontend/hooks/`, `frontend/lib/`, `frontend/app/(dashboard)/user/settings/`.

> **Temuan yang mendasari plan ini** (lihat spec §1):
> 1. `NewEntitlementMiddleware` **tidak pernah dikonstruksi** — 0 dari 63 rute ter-gate.
> 2. Resolver mengembalikan **deny-all** untuk user tanpa baris `subscriptions`
>    (52 user vs 2 subscription di DB dev) → menyalakan enforcement akan mengunci ~96% user.
> 3. Matriks dokumentasi P07 menyebut endpoint yang tidak ada & memakai 429 (kode memakai 402).

---

## 1. Tujuan & Definition of Done

Menyalakan penegakan entitlement yang **sudah ada kodenya** ke rute user yang benar-benar
ada, tanpa mengunci user lama — dengan memperbaiki akar masalah fallback plan dan
menyediakan mirror frontend yang jujur.

**Definition of Done Slice A:**
- [ ] User tanpa baris `subscriptions` resolve ke plan `free` (bukan deny-all); fail-closed
      tetap berlaku bila plan `free` hilang.
- [ ] `EntitlementMiddleware` dikonstruksi di `main.go` dan disuntikkan ke `SetupRouter`.
- [ ] Rute ter-gate sesuai matriks §3.2 spec; rute read tetap terbuka.
- [ ] Kontrak blok: 403 `feature_not_in_plan`, 402 `limit_exceeded` / `limit_not_in_plan`,
      keduanya menyertakan `upgrade_url`.
- [ ] Usage resolver: `max_transactions` (non-soft-deleted), `ai_tokens_per_cycle`,
      `ocr_scans_per_cycle`; periode tidak ada → usage 0 **diizinkan**, error query → 500.
- [ ] Valve `ENTITLEMENT_ENFORCEMENT` (default on) terpasang & terdokumentasi.
- [ ] `useEntitlements` + `useRole` + halaman `/user/settings/subscription` (read-only).
- [ ] Test integrasi membuktikan user Free **diblokir** pada rute ter-gate (regression guard
      untuk cacat wiring).
- [ ] `go vet ./... && go test ./... && go build ./...` hijau (backend);
      `pnpm lint && pnpm build && pnpm test` hijau (frontend).
- [ ] Dokumentasi P07 dikoreksi (matriks nyata, 402 bukan 429, nama kunci nyata).
- [ ] Tanpa migrasi database.

---

## 2. Urutan Eksekusi (Wave)

| Wave | Langkah | Deskripsi | Prasyarat |
|---|---|---|---|
| **W1** | A-1 | Fallback plan `free` + sentinel `StatusNone` | — |
| **W1** | A-2 | Read-only `GetCurrentPeriod` (non-locking) | — |
| **W1** | A-3 | Usage resolver (`max_transactions`, tokens, ocr scans) | A-2 |
| **W2** | A-4 | `upgrade_url` + valve `ENTITLEMENT_ENFORCEMENT` pada middleware | — |
| **W2** | A-5 | Konstruksi middleware di `main.go` + `RouterConfig` | A-4 |
| **W2** | A-6 | Pemasangan gate per-rute di `router.go` | A-3, A-5 |
| **W3** | A-7 | Test unit backend (resolver, middleware, usage) | A-1…A-6 |
| **W3** | A-8 | Test integrasi router nyata (DB-backed) | A-7 |
| **W4** | A-9 | `useEntitlements` + `useRole` + query key | A-6 |
| **W4** | A-10 | Halaman `/user/settings/subscription` (read-only) | A-9 |
| **W4** | A-11 | Toast 403/402 + disable aksi pada mirror UI | A-9 |
| **W5** | A-12 | Test frontend (Vitest) + lint/build | A-9…A-11 |
| **W5** | A-13 | Koreksi dokumentasi P07 + tracking + log | A-8, A-12 |

**Jalur kritis:** A-1 → A-5 → A-6 → A-8 → A-13 (A-1 adalah blocker: tanpa fallback,
A-6 mengunci user).

---

## 3. Detail Langkah

### W1 — Fondasi resolusi

#### A-1 — Fallback plan `free`
- **Target:** `backend/internal/repository/postgres/entitlement_source.go`,
  `backend/internal/billing/domain.go`
- **Langkah:**
  1. Tambahkan konstanta `StatusNone SubscriptionStatus = "none"` di `billing/domain.go`
     (terpisah dari seluruh status lifecycle: `trialing/active/past_due/grace/canceled/expired`).
  2. Di `GetActivePlanForUser`, bila `pgx.ErrNoRows`: baca `DEFAULT_PLAN_CODE`
     (default `"free"`) dan panggil `GetPlanByCode`; kembalikan plan tersebut dengan
     status `StatusNone`.
  3. Bila plan `free` tidak ada / gagal dibaca: kembalikan `(nil, "", nil)` seperti sekarang
     (deny-all) dan **log** kondisi tersebut.
  4. `Resolved.Status` belum punya konsumen, jadi penambahan ini aditif.
- **Verifikasi:** unit test — user tanpa subscription → features/limits Free (tidak kosong);
  plan `free` hilang → deny-all; user Pro aktif tidak berubah.

#### A-2 — Read-only `GetCurrentPeriod`
- **Target:** `backend/internal/billing/repository.go` (interface),
  `backend/internal/repository/postgres/billing_repo.go`, fake test
- **Langkah:**
  1. Tambahkan `GetCurrentPeriod(ctx, userID uuid.UUID, nowUnix int64) (*TokenPeriod, error)`
     ke `TokenRepository`.
  2. Implementasi **tanpa** `FOR UPDATE` (predikat sama dengan `GetPeriodForUpdate`:
     `period_start <= $2 AND period_end >= $2 ORDER BY period_start DESC LIMIT 1`).
  3. `pgx.ErrNoRows` → `(nil, nil)` (bukan error) — pemanggil memperlakukan sebagai usage 0.
  4. Perbarui fake/implementer yang ada (enumerasi lewat grep, bukan asumsi).
- **Verifikasi:** unit test periode ada/tidak ada; **tanpa** lock (bisa dipanggil di luar tx).

#### A-3 — Usage resolver
- **Target:** `backend/internal/delivery/http/middleware/entitlement.go` (atau file baru
  `usage_resolvers.go` di paket yang sama)
- **Langkah:**
  1. `max_transactions`: `COUNT(*)` transaksi user dengan `is_deleted = false`.
  2. `ai_tokens_per_cycle`: `used_tokens + reserved_tokens` periode saat ini; periode tidak
     ada → `0`.
  3. `ocr_scans_per_cycle`: `COUNT(*)` baris `token_ledger` `modality='vision'` pada periode
     saat ini; periode tidak ada → `0`.
  4. **Kontrak error:** error query → kembalikan error (middleware → 500). Data tidak ada →
     usage 0, **bukan** error. Jangan menyamakan keduanya.
- **Verifikasi:** unit test tiap resolver: ada data, tidak ada periode (→0), error DB (→error).

### W2 — Pemasangan enforcement

#### A-4 — `upgrade_url` + valve enforcement
- **Target:** `backend/internal/delivery/http/middleware/entitlement.go`
- **Langkah:**
  1. Tambahkan `"upgrade_url": "/user/settings/subscription"` pada payload 403
     `feature_not_in_plan` dan 402 `limit_exceeded` / `limit_not_in_plan`.
  2. Baca `ENTITLEMENT_ENFORCEMENT` (default `on`); bila `off`, middleware memanggil
     `c.Next()` dan menulis satu log peringatan bahwa otorisasi dinonaktifkan.
  3. Jangan mengubah kode status/kunci error yang sudah ada (402 dipertahankan).
- **Verifikasi:** unit test payload menyertakan `upgrade_url`; valve `off` → request lolos
  walau fitur tidak ada di paket.

#### A-5 — Konstruksi & injeksi middleware
- **Target:** `backend/cmd/server/main.go`, `backend/internal/delivery/http/router.go`
- **Langkah:**
  1. Di `main.go`, konstruksi `entMiddleware := middleware.NewEntitlementMiddleware(entResolver)`
     (sekarang belum pernah dibuat) beserta usage resolver dari A-3.
  2. Tambahkan field ke `RouterConfig` (mis. `Entitlement *middleware.EntitlementMiddleware`),
     atau parameter eksplisit — ikuti gaya parameter eksplisit yang sudah dipakai `SetupRouter`.
  3. Teruskan pada pemanggilan `SetupRouter` (`main.go:282`).
- **Verifikasi:** `go build ./...`; middleware non-nil pada runtime (diuji di A-8).

#### A-6 — Pemasangan gate per-rute
- **Target:** `backend/internal/delivery/http/router.go`
- **Langkah:** pasang **per-rute** dengan `.Use(...)`, bukan pada grup `userRoutes` bersama,
  agar cache 30s rute read tidak berubah.

  | Rute | Gate | Kunci |
  |---|---|---|
  | `POST /transactions` | WithinLimit | `max_transactions` |
  | `POST /ai/chat` | RequireFeature + WithinLimit | `ai_chat` / `ai_tokens_per_cycle` |
  | `POST /ai/receipt` | RequireFeature + WithinLimit | `ocr_receipt` / `ocr_scans_per_cycle` |
  | `POST /ai/receipt/transaction` | RequireFeature | `ocr_receipt` |
  | `POST /ai/voice` | RequireFeature | `voice_input` |
  | `GET /reports` | RequireFeature | `reports_export` |

  1. Rute read (`GET /transactions`, `/dashboard/*`, `/transactions/stats`, `/categories/*`,
     `/spk/*`, `/ai/sessions*`, `/providers`) **tidak** diberi gate.
  2. `POST /ai/sessions` + update/delete sesi **tidak** digate (bukan fitur termeter).
  3. `POST /spk/calculate` **tidak** digate (tidak ada kunci seed yang cocok; sudah
     rate-limited 30/menit).
  4. `POST /gemini-keys*` dibiarkan (penghapusan BYOK = slice D).
- **Verifikasi:** test integrasi A-8.

### W3 — Verifikasi backend

#### A-7 — Test unit
- **Target:** `backend/internal/entitlement/resolver_test.go`,
  `backend/internal/delivery/http/middleware/entitlement_test.go`, test usage resolver
- **Kasus:**
  - resolver: tanpa subscription → Free; Pro aktif → Pro; `free` hilang → deny-all.
  - middleware: 403 feature-denied; 402 limit-exceeded; 402 fail-closed kunci absen;
    super_admin bypass; error usage → **500 (bukan allow)**; **periode tidak ada → diizinkan**;
    valve `off` → lolos; payload memuat `upgrade_url`.
- **Verifikasi:** `go test ./internal/...` hijau.

#### A-8 — Test integrasi router nyata (regression guard utama)
- **Target:** `backend/tests/entitlement_e2e_test.go` (pola harness P06 yang sudah ada:
  HTTP asli + middleware produksi + PostgreSQL nyata, env-gated `TEST_DATABASE_URL`)
- **Kasus:**
  - Free user → `GET /reports` = 403 `feature_not_in_plan`
  - Pro user → `GET /reports` = 200
  - Free user pada 100 transaksi → `POST /transactions` = 402 `limit_exceeded`
  - Free user 99 transaksi + 1 soft-deleted → tetap diizinkan
  - Free user **tanpa token period** → `POST /ai/chat` diizinkan (usage 0, bukan 402)
  - Free user → `POST /ai/voice` = 403; Pro user → 200
  - super_admin → bypass semua gate
  - User **tanpa baris subscription** → diperlakukan sebagai Free (`ai_chat` diizinkan,
    `ocr_receipt` diblokir)
  - payload 403/402 memuat `upgrade_url`
- **Verifikasi:** `TEST_DATABASE_URL=... go test -run TestEntitlement ./tests/...` hijau.
  Test ini adalah penjaga yang akan menangkap middleware yang tidak ter-wiring kembali.

### W4 — Mirror frontend

#### A-9 — Hook `useEntitlements` + `useRole`
- **Target:** `frontend/hooks/use-entitlements.ts`, `frontend/hooks/use-role.ts`,
  `frontend/lib/cache-keys.ts`
- **Langkah:**
  1. `useEntitlements()` — satu TanStack Query ke `GET /me/entitlements` memakai axios `api`
     yang sudah ada (cookie httpOnly + single-flight refresh sudah ditangani). Ekspos
     `hasFeature(key)` dan `withinLimit(key, current?)`; fail-closed bila data null.
  2. Daftarkan `entitlements: { me: ["entitlements","me"] }` di `cache-keys.ts`.
  3. `useRole()` — wrapper bertipe atas `useSession()` yang sudah ada, mengembalikan
     `normalizeRole(user?.role)` dari `lib/roles.ts`. **Jangan** fetch `/me` lagi.
- **Verifikasi:** test Vitest (A-12).

#### A-10 — Halaman `/user/settings/subscription`
- **Target:** `frontend/app/(dashboard)/user/settings/subscription/page.tsx`
- **Langkah:** halaman **read-only** dari `useEntitlements()`: nama plan + status, daftar
  fitur (aktif/nonaktif), tabel limit (`-1` → "Unlimited"). **Tanpa** alur pembelian.
- **Verifikasi:** render test dari respons termock.

#### A-11 — Mirror UX (advisory-only)
- **Langkah:** item menu/aksi untuk fitur ter-gate dirender disabled + hint upgrade saat
  `hasFeature()` false; respons 403/402 memunculkan toast menuju halaman subscription.
  Backend tetap satu-satunya otoritas; `proxy.ts` tetap menjaga berbasis role.
- **Verifikasi:** manual + test hook.

### W5 — Verifikasi menyeluruh & dokumentasi

#### A-12 — Test frontend
- **Kasus:** `use-entitlements` (fitur/limit, `-1` = unlimited, fail-closed null),
  `use-role`, render halaman subscription.
- **Verifikasi:** `pnpm lint && pnpm build && pnpm test` hijau.

#### A-13 — Koreksi dokumentasi
- **Target:** `P07/teknis/rbac-rollout.md` (matriks rute nyata; 429 → 402; nama kunci nyata
  `max_transactions`; `upgrade_url`; daftar kunci yang belum ditegakkan),
  `P07/planning/tasks.md` (AC TASK-P07-001: 429 → 402),
  `P07/tracking/tracking.md`, `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.
- **Wajib dicatat:** kunci seed yang **belum** ditegakkan (`ai_requests_per_cycle`,
  `max_categories`, `max_wallets`, `rag_search`, `advanced_budgeting`, `api_access`,
  `bank_sync`) beserta alasannya, agar tidak dikira selesai.

---

## 4. Matriks Bukti

| Langkah | Artefak Kode | Bukti |
|---|---|---|
| A-1 | `postgres/entitlement_source.go`, `billing/domain.go` | unit test fallback Free + fail-closed |
| A-2 | `billing/repository.go`, `postgres/billing_repo.go` | unit test periode ada/tidak ada |
| A-3 | `middleware/usage_resolvers.go` | unit test 3 resolver |
| A-4 | `middleware/entitlement.go` | unit test `upgrade_url` + valve |
| A-5/A-6 | `cmd/server/main.go`, `delivery/http/router.go` | test integrasi A-8 |
| A-7/A-8 | `tests/entitlement_e2e_test.go` | `go test -run TestEntitlement ./tests/...` |
| A-9…A-11 | `hooks/use-entitlements.ts`, `hooks/use-role.ts`, halaman subscription | `pnpm test` |
| A-13 | docs P07 + log | diff dokumentasi |

---

## 5. Perintah Verifikasi

```bash
# Backend
cd backend && go vet ./... && go test ./... && go build ./...

# Integrasi (butuh DB; fintrack_test sudah dimigrasi 0001–0010)
cd backend && TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5435/fintrack_test?sslmode=disable" \
  go test -run TestEntitlement -v ./tests/...

# Frontend
cd frontend && pnpm lint && pnpm build && pnpm test
```

---

## 6. Rollback

1. **Valve cepat:** set `ENTITLEMENT_ENFORCEMENT=off` lalu restart backend — middleware
   menjadi no-op tanpa mengubah kode (khusus insiden).
2. **Rollback kode:** revert commit slice A; karena tanpa migrasi, tidak ada perubahan skema
   yang perlu dibalik.
3. **Efek samping yang perlu diketahui:** user yang jumlah transaksinya sudah di atas batas
   Free (mis. user demo 479 vs 100) akan ditolak pada `POST /transactions` — rute read tetap
   terbuka. Ini enforcement yang benar, bukan bug; valve adalah jalan keluar bila mengganggu.

---

## 7. Handoff

Slice ini menutup `TASK-P07-001` & `TASK-P07-002`. Slice P07 berikutnya (belum dimulai):
**B** Realtime SSE, **C** Standar UI/DataTable/a11y, **D** BYOK + ops (health gates,
graceful shutdown 30s, expand/contract), dan sisanya sesuai `P07/planning/tasks.md`.
