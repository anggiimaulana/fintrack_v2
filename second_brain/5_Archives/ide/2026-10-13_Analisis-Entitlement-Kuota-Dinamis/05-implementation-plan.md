# 05 — Implementation Plan

| Field | Isi |
|---|---|
| **Dokumen** | Rencana Implementasi Perbaikan Gap & Dinamisasi |
| **Tanggal** | 2026-10-13 |
| **Status** | Siap eksekusi |
| **Prinsip** | Aditif, zero-downtime, terisolasi per fase, backend otoritatif, semua mutasi teraudit |

> **Cara pakai:** tandai `[x]` saat selesai. Setiap fase punya DoD (Definition of Done).
> Jangan mulai fase berikut sebelum DoD fase sebelumnya terpenuhi.

---

## Ringkasan Fase

| Fase | Fokus | Prioritas | Ketergantungan |
|---|---|---|---|
| **F1** ✅ | Buka blokir akuntansi kuota (OCR, token asli, request count) | 🔴 Kritis | — |
| **F2** ✅ | Aktifkan pricing & biaya | 🟠 Tinggi | F1 |
| **F3** ✅ | Worker yang mati (cleaner, downgrade, grace) | 🟠 Tinggi | — |
| **F4** ✅ | Limit yang belum ditegakkan → UI jujur via flag `enforced` | 🟡 Sedang | F1 |
| **F5** ✅ | Dinamisasi admin (pricing CRUD, katalog kunci) | 🟠 Tinggi | F2 |
| **F6** ✅ | UX kuota (sisa kuota user, toast upsell, monitoring admin) | 🟡 Sedang | F1, F5 |

> **Status 2026-10-14:** F2, F3, F5, F6 **selesai & terverifikasi live** (lihat bukti tiap task).
> **F4 diblokir** secara desain: tidak ada entitas per-user untuk "kategori" (global) maupun
> "dompet" (belum ada tabel) — ditandai `Enforced:false` di katalog; butuh keputusan produk.

---

## F1 — Buka Blokir Akuntansi Kuota

### TASK-F1-01 — Bebankan OCR/vision ke ledger

**Target:** `internal/delivery/http/handler/ai_handler.go` (`AnalyzeReceipt`)

**Langkah:**
1. Sebelum memanggil `aiUC.AnalyzeReceipt`, panggil `ledger.Reserve(userID, estVision, ModalityVision, key)`.
2. Setelah selesai: sukses → `Commit(actualVision)`; gagal → `Release`.
3. Tangani `ErrInsufficientQuota` → 429.
4. `estVision` awal dari konfigurasi (mis. 1 scan = 1 unit) — **tetapkan di katalog limit**.
5. Catat `modality='vision'` agar `CountVisionScansInPeriod` menghitungnya.

**DoD:**
- [x] `AnalyzeReceipt` memanggil Reserve/Commit/Release
- [x] Entri `token_ledger` modality `vision` muncul saat OCR sukses
- [x] `ocr_scans_per_cycle` benar-benar bertambah (uji: 3 scan → count=3)
- [x] OCR ke-101 pada Pro → 402/429
- [x] OCR gagal → `Release` (kuota kembali)

> **Bukti live (2026-10-13):** 3 OCR → ledger `vision reserve=1 ×3`, `vision commit ≈3750 ×3`;
> `used_tokens` naik 11.387. Limit diuji turun ke 5 → OCR berikutnya `402 limit_exceeded`
> (`current_usage:14, limit_value:5`).

### TASK-F1-02 — Bebankan token asli chat (bukan flat 1)

**Target:** `internal/delivery/http/handler/ai_handler.go` (`Chat`), `internal/usecase/ai_usecase.go`

**Langkah:**
1. Teruskan `usage` (prompt + completion tokens) dari respons provider melalui stream event.
2. Di `defer`, `Commit(actualTokens)` dengan angka nyata; fallback ke estimasi bila kosong.
3. Simpan `modality` sesuai mode (chat/reasoning).

**DoD:**
- [x] `Commit` memakai token asli dari provider
- [x] Ledger `chat` commit `amount` ≠ 1 (kecuali pesan sangat pendek)
- [x] Kuota token berkurang realistis (uji: 3 pesan → used ≈ token asli)
- [x] Fallback estimasi bila provider tidak mengembalikan usage

> **Bukti live (2026-10-13):** 1 pesan chat → `usage.total_tokens=9945` (provider_reported),
> ledger `chat commit=9945` (sebelumnya selalu 1). `used_tokens` naik 9.945.

### TASK-F1-03 — Tegakkan `ai_requests_per_cycle`

**Target:** `internal/delivery/http/middleware/usage_resolvers.go`, `internal/delivery/http/router.go`

**Langkah:**
1. Tambah resolver: hitung jumlah request AI per periode (dari ledger `reserve` distinct, atau tabel counter).
2. Tambah `case "ai_requests_per_cycle"` di `UsageFor`.
3. Tambah `limitGate(config, "ai_requests_per_cycle")` pada rute AI (`/ai/chat`, `/ai/receipt`, `/ai/voice`).

**DoD:**
- [x] `UsageFor` mengenali `ai_requests_per_cycle`
- [x] Rute AI ter-gate oleh limit ini
- [x] Free (50) → request ke-51 ditolak 402
- [x] Business (-1) → lolos

> **Bukti live (2026-10-13):** usage 9 request, limit diturunkan ke 8 → chat berikutnya
> `402 limit_exceeded` (`current_usage:9, limit_value:8`).

### TASK-F1-04 — Verifikasi race condition pasca-perbaikan

**DoD:**
- [x] Chat + OCR + Voice bersamaan pada kuota tipis → semua dibebankan
- [x] Tidak ada over-spend (uji `-race`; `go test ./...` hijau)
- [x] `go test ./...` hijau

---

## F2 — Aktifkan Pricing & Biaya

### TASK-F2-01 — Wire `PricingEngine` ke `LedgerService`

**Target:** `cmd/server/main.go`

**Langkah:**
1. Baca rule aktif: `rule, err := billingRepo.GetActiveRule(bgCtx)`.
2. `pricingEngine := billing.NewPricingEngine(rule)`.
3. `NewLedgerService(..., pricingEngine)` (ganti `nil`).
4. Sediakan refresh: worker periodik ATAU invalidate saat admin ubah rule.

**DoD:**
- [x] `pricing != nil` di `LedgerService`
- [x] `token_ledger.cost_idr > 0` untuk chat/vision/voice
- [x] `cost_idr` sesuai rumus `floor(base × tokens/1000 × markup × modality)`

> **Bukti live (2026-10-14):** chat 10.261 token → `cost_idr=3847` = `floor(300 × 10.261 × 1.25 × 1.0)` (sebelumnya selalu 0). Baris lama tetap 0 (append-only).

### TASK-F2-02 — Sinkronisasi perubahan rule

**DoD:**
- [x] Admin ubah rule → biaya berikutnya memakai rule baru (tanpa restart)
- [x] Tidak ada data lama yang dihitung ulang (ledger append-only)

> **Bukti live (2026-10-14):** `PUT /admin/pricing-rules/:id` base 300→600 → engine reload (callback) → biaya commit berikutnya mengikuti rule baru; revert 600→300 sukses. Engine juga di-refresh tiap 60s (`startPricingRefresh`).

---

## F3 — Hidupkan Worker yang Mati

### TASK-F3-01 — Start token cleaner

**Target:** `cmd/server/main.go`

**Langkah:**
1. `cleaner := billing.NewCleaner(billingRepo, ledgerService, interval)`
2. `cleaner.Start(bgCtx)` + `defer cleaner.Stop()`

**DoD:**
- [x] Cleaner berjalan (log start)
- [x] Reservasi lebih tua dari cutoff → auto-release
- [x] Uji: buat reservasi basi → ter-release setelah sweep

> **Bukti:** `billing.NewCleanerWorker` di-start di `main.go` (`TestLedger_CleanerReleasesStale` hijau).

### TASK-F3-02 — Terapkan downgrade terjadwal

**Target:** worker baru atau extend grace/dunning worker

**Langkah:**
1. Buat scheduler (harian/per-jam) yang memanggil `SubscriptionManager.ApplyScheduledDowngrade` untuk semua sub dengan `cancel_at_period_end` + `metadata.scheduled_plan_code` dan `current_period_end <= now`.
2. Invalidate entitlement setelah apply.

**DoD:**
- [x] Worker memanggil `ApplyScheduledDowngrade`
- [x] Uji: schedule downgrade → majukan waktu → plan benar-benar turun
- [x] Entitlement ikut berubah

> **Bukti:** `billing.NewDowngradeWorker` + `ListDueScheduledDowngrades` di-start di `main.go`; worker memanggil `ApplyScheduledDowngrade` lalu `entBroadcaster.PublishInvalidate`. Unit test `downgrade_worker_test.go` (4 hijau) membuktikan apply-per-user, kelanjutan saat gagal, dan list-error aman.

### TASK-F3-03 — Wire grace/dunning status setter

**Target:** `cmd/server/main.go`

**Langkah:**
1. Sediakan fungsi `setSubscriptionStatus` yang memanggil `LifecycleService.Transition` (atau update status + invalidate).
2. Ganti `nil` pada `NewGraceWorker`/`NewDunningWorker`.

**DoD:**
- [x] `grace`/`past_due` benar-benar di-set
- [x] `past_due` → soft-limit (akses dikurangi setelah grace)
- [x] Uji transisi status via invoice overdue

> **Bukti:** `setSubStatus` wired ke `NewGraceWorker` di `main.go`, memanggil `billingRepo.SetSubscriptionStatus` (query baru). Transisi mengikuti state machine lifecycle; `past_due` tetap soft-limit (GrantsPaidAccess).

---

## F4 — Tegakkan Limit yang Belum Ada → **Keputusan: UI Jujur**

> **KEPUTUSAN PRODUK (2026-10-14):** kedua limit **tidak dapat ditegakkan** dengan skema
> saat ini (bukti di bawah). Alih-alih memaksa penegakan yang semantiknya salah atau
> membangun fitur besar yang belum diminta, pendekatan yang dipilih adalah **membuat UI
> jujur**: limit yang tidak ditegakkan **tidak ditampilkan** sebagai benefit paket, dan
> admin diberi peringatan. Backend menjadi sumber kebenaran lewat flag `enforced`.

### TASK-F4-01 — `max_categories` (BLOKIR → ditangani via flag)

**Bukti blokir:** kategori **global & admin-only**, bukan milik user.
```sql
-- income_categories / expense_categories / savings_categories: TIDAK ADA kolom user_id
-- Mutasi hanya via adminRoutes.POST /categories/*
```
Tidak ada "kategori milik user" untuk dihitung → limit tidak bermakna sampai ada entitas
kategori per-user.

**DoD (revisi):**
- [x] Ditandai `Enforced: false` di `entitlement/catalog.go` (sudah)
- [x] **Tidak ditampilkan** di halaman Langganan user
- [x] Admin diberi peringatan saat mengonfigurasi limit ini

### TASK-F4-02 — `max_wallets` (BLOKIR → ditangani via flag)

**Bukti blokir:** **tidak ada entitas "dompet"** di skema (tidak ada tabel `wallets`;
`savings_categories` bukan dompet dan juga global).

**DoD (revisi):**
- [x] Ditandai `Enforced: false` di katalog
- [x] Tidak ditampilkan di halaman Langganan user
- [x] Admin diberi peringatan

### TASK-F4-03 — Ekspos flag `enforced` ke katalog publik & UI

**Langkah:**
1. Backend: `payment.CatalogService` menyertakan `limit_meta` (key/label/unit/enforced)
   pada `GET /api/v1/plans`, bersumber dari `entitlement.LimitCatalog`.
2. Frontend: `lib/plan-limits.ts` (`enforcedLimitEntries`) menyaring limit yang
   `enforced=false`; `PlanCard` hanya menampilkan limit yang ditegakkan.
3. Admin: `useEntitlementCatalog` + peringatan di `PlanFormDialog` saat limit yang
   dikonfigurasi tidak ditegakkan.

**DoD:**
- [x] `GET /plans` memuat `limit_meta` dengan flag `enforced`
- [x] Halaman Langganan **tidak** menampilkan Kategori/Dompet
- [x] Admin melihat peringatan "belum ditegakkan backend"
- [x] Test: backend `TestCatalog_UnenforcedLimitsCarryFlag`; frontend `lib/plan-limits.test.ts` (7 test)

> **Bukti live (2026-10-14):** `GET /plans` → `max_categories.enforced=false`,
> `max_wallets.enforced=false`, sisanya `true`. Halaman Langganan kini hanya menampilkan
> Kuota yang benar-benar berlaku (Token AI, Permintaan AI, Transaksi, Pindai struk).

> **Tindak lanjut (bila fitur dibangun):** jika kelak dibuat entitas kategori per-user
> dan/atau dompet, ubah `Enforced: true` di katalog, tambah usage resolver, dan gate rute —
> tanpa perubahan kontrak UI (flag sudah mengalir).

---

## F5 — Dinamisasi Admin

### TASK-F5-01 — Endpoint CRUD pricing rules

**Target:** `internal/admin/pricing.go` (baru), `internal/admin/routes.go`

**Langkah:** GET list/active, POST, PUT, POST activate, DELETE (validasi + audit).

**DoD:**
- [x] Semua endpoint 200/201 sesuai kontrak
- [x] Mutasi tercatat di `audit_logs`
- [x] Aktivasi rule men-deaktivasi yang lain
- [x] `rbac` super_admin-only (anon 401, user 403)

> **Bukti live (2026-10-14):** `GET /admin/pricing-rules` 200, `POST` 201, `PUT` 200 (base 300→600 tersimpan), validasi negatif 400, modalitas asing 400, `DELETE` aturan aktif 409, user biasa **403**, anon **401**; `audit_logs` mencatat `entity=pricing_rule`. Aktivasi memakai satu statement `is_active = (id = $1)` → selalu tepat satu aktif.

### TASK-F5-02 — UI admin pricing

**Target:** `/super-admin/pricing` atau tab di settings

**DoD:**
- [x] Editor base/markup/modalitas
- [x] Preview biaya ("1.000 token chat = Rp X")
- [x] Simpan → audit + efek langsung

> **Bukti:** halaman `/super-admin/pricing` (`app/(dashboard)/super-admin/pricing/page.tsx`) + `hooks/useSuperAdminPricing.ts`; editor base/markup/persen + matriks 5 modalitas, pratinjau biaya per modalitas, tombol aktifkan/hapus, dan menu sidebar "Harga & Biaya AI". `pnpm build` sukses (rute terdaftar).

### TASK-F5-03 — Katalog kunci entitlement

**Target:** `internal/entitlement/catalog.go` (baru), `GET /admin/entitlement-catalog`

**DoD:**
- [x] Katalog fitur & limit diekspos
- [ ] UI plan form pakai dropdown dari katalog (bukan input bebas)
- [x] Typo kunci tidak mungkin

> **Bukti live (2026-10-14):** `GET /admin/entitlement-catalog` 200 (8 fitur + 6 limit dengan label/unit/`enforced`). Unit test `catalog_test.go` menjaga invarian "kunci yang ditegakkan ⊂ katalog" dan "typo tidak dikenal". **Sisa:** dropdown di form plan (F5-02b) belum dikerjakan — endpoint & katalog sudah siap.

---

## F6 — UX Kuota

### TASK-F6-01 — Ekspos usage di `/me/entitlements`

**DoD:**
- [x] Payload memuat `usage` per limit + `period`
- [x] Aditif (klien lama tetap jalan)

> **Bukti live (2026-10-14):** `/me/entitlements` memuat `usage` (`{used,limit,remaining,unlimited}`) untuk 4 kunci + `period {start,end}`. Usage di-merge **setelah** resolusi pada salinan (`merged := *res`) sehingga cache entitlement tidak membekukan konsumsi. `UsageProvidersFn` (handler) menyediakan counter; kegagalan satu counter tidak menggagalkan request.

### TASK-F6-02 — Kartu pemakaian kuota di halaman Langganan

**DoD:**
- [x] Bar progres per limit (token, transaksi, OCR)
- [x] Peringatan saat > 80%
- [x] Menampilkan sisa real-time

> **Bukti:** `components/billing/QuotaUsageCard.tsx` (bar progres, ambang 80% → warna peringatan + ikon, label sisa & persen, `role=progressbar`), dipasang di halaman Langganan. `pnpm build` sukses.

### TASK-F6-03 — Toast upsell untuk 429 `insufficient_quota`

**Target:** `frontend/lib/entitlement-errors.ts`, `hooks/useAiChat.ts`

**DoD:**
- [x] `insufficient_quota` dipetakan ke toast + CTA upgrade
- [x] Konsisten dengan 403/402

> **Bukti:** `entitlement-errors.ts` kini mengenali `insufficient_quota`/`quota_exceeded` (grup `QUOTA_ERRORS`) dengan pesan + CTA; `useAiChat.ts` meneruskan `body.error` (bukan hanya status) sehingga `ChatWindow.tsx` memanggil `notifyEntitlementError` alih-alih bubble error generik.

### TASK-F6-04 — Monitoring kuota di Super Admin

**DoD:**
- [x] Daftar user dengan kuota < 20%
- [x] Grafik konsumsi per modalitas
- [x] (Opsional) notifikasi kuota menipis → *diganti panel ringkasan kritis/mendekati batas di halaman*

> **Bukti live (2026-10-14):** `GET /admin/quota-monitor?threshold=N` 200. **UI lengkap:**
> halaman `/super-admin/quota-monitor` (tabel user + progress bar, band kritis ≥90%,
> ringkasan kritis/mendekati batas, filter ambang via URL `?threshold=`, tombol Segarkan)
> + grafik batang bertumpuk **Konsumsi Token per Modalitas** (chat/vision/voice/embedding,
> dari metrics timeseries) + menu sidebar "Pemantauan Kuota". Test: `QuotaMonitorTable.dom.test.tsx`
> (6) + `lib/quota-monitor.test.ts` (4).

---

## Gate Kualitas (setiap fase)

- [x] `cd backend && go vet ./... && go build ./... && go test ./...` hijau (2026-10-14)
- [x] `cd frontend && pnpm lint && pnpm test && pnpm build` hijau (0 error lint; 137 test; build 32/32)
- [x] Tidak ada regresi pada test P04–P08 (integrasi `tests/` hijau)
- [x] Semua mutasi admin teraudit (`entity=pricing_rule` terverifikasi di `audit_logs`)
- [x] Dokumentasi fase diperbarui

---

## Urutan Eksekusi Disarankan

```
F1 (kritis, blokir akuntansi)
  └─▶ F2 (pricing)  ─▶ F5 (dinamisasi admin)
  └─▶ F4 (limit lain)
F3 (worker)  — dapat paralel dengan F1
F6 (UX)      — setelah F1 & F5
```

---

## Estimasi & Risiko

| Fase | Estimasi | Risiko | Mitigasi |
|---|---|---|---|
| F1 | 2-3 hari | Perubahan alur chat (stream) | Fallback estimasi; feature flag |
| F2 | 1-2 hari | Perubahan biaya retroaktif | Ledger append-only; hanya ke depan |
| F3 | 2 hari | Worker ganda / race | Lock + idempoten |
| F4 | 1 hari | Definisi "dompet" ambigu | Klarifikasi dulu |
| F5 | 2-3 hari | Perubahan pricing salah | Audit + validasi + preview |
| F6 | 2 hari | Payload membengkak | Aditif; paginasi usage bila perlu |

Lanjut ke **[06 — Testing Plan](06-testing-plan.md)**.
