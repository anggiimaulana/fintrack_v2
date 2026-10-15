# 06 — Rencana Implementasi

[← Index](./README.md) · [← Model Biaya](./05-model-biaya-dan-margin.md)

---

## 1. Prinsip pelaksanaan

| # | Prinsip |
|---|---|
| P-1 | **Expand → Migrate → Contract** (aturan `database/migrations/README.md`) |
| P-2 | Setiap fase **berdiri sendiri & bisa di-deploy** (tanpa memutus fase lain) |
| P-3 | Rasio default `1` ⇒ fase 1–2 **tidak mengubah perilaku** apa pun |
| P-4 | Verifikasi di akhir **setiap** fase, bukan hanya di akhir |
| P-5 | Frontend tidak mendahului backend (UI hanya menampilkan yang API sediakan) |

---

## 2. Peta fase

```
F1 ─ Migrasi (kolom) + Domain ─┐
F2 ─ Ledger (inti)            ─┤ Backend, tidak mengubah perilaku (rasio=1)
F3 ─ Repository + Wiring      ─┘
                                ↓  ← titik ini: rasio masih 1, sistem identik
F4 ─ Katalog + Lapisan        ─┐
     Konfigurasi Dinamis        ┤ Semua kebijakan pindah ke DB
     (setting/metric/feature)   ┘
                                ↓  ← tidak ada lagi hardcode kebijakan
F5 ─ API Admin + Kalkulator   ─┐
F6 ─ UI Super Admin           ─┤ Operator mengatur SEMUA dari UI
F7 ─ UI User (label dinamis)  ─┘
                                ↓  ← operator mengaktifkan rasio > 1
F8 ─ Dokumentasi + Rollout
```

---

## 3. Fase 1 — Migrasi & Domain

### 3.1 Tugas

| # | Tugas | File |
|---|---|---|
| 1.1 | Tulis migrasi `0028_plan_token_quota_ratio.sql` | `database/migrations/` |
| 1.2 | Uji migrasi maju + ulang (idempoten) | `fintrack_test` |
| 1.3 | Tambah `Plan.TokenQuotaRatio` | `billing/domain.go` |
| 1.4 | Tambah `TokenPeriod.QuotaRatio` | `billing/domain.go` |
| 1.5 | Tambah `TokenLedgerEntry.RawTokens` + helper `RawTokensOrAmount()` | `billing/domain.go` |
| 1.6 | Ubah signature `QuotaResolver.ResolveQuota` (+`ratio`) | `billing/ledger.go` |
| 1.7 | Tambah `RoundUnits` (mode dinamis) | `billing/rounding.go` |

### 3.2 Kriteria selesai

- [ ] Migrasi jalan maju tanpa error
- [ ] Migrasi idempoten (dijalankan 2× sukses)
- [ ] `go build ./...` sukses (belum ada pemanggil baru yang gagal)
- [ ] `go vet ./...` bersih

### 3.3 Risiko

| Risiko | Mitigasi |
|---|---|
| Signature `QuotaResolver` berubah → pemanggil rusak | Ubah **semua** pemanggil dalam satu commit; `go build` menangkap |
| Migrasi gagal karena `token_ledger` besar | Gunakan `NOT VALID` + `VALIDATE` (lihat [02 §2.2](./02-skema-database-dan-migrasi.md)) |

---

## 4. Fase 2 — Ledger (inti, paling sensitif)

### 4.1 Tugas

| # | Tugas | File |
|---|---|---|
| 2.1 | `Reserve` menerima **RAW**, konversi ke unit internal | `billing/ledger.go` |
| 2.2 | `Commit` menerima **RAW**, konversi, tulis `raw_tokens`, `cost_idr` dari RAW | `billing/ledger.go` |
| 2.3 | `Release` tidak berubah (sudah unit) | `billing/ledger.go` |
| 2.4 | `Refund` menerima **RAW**, konversi | `billing/ledger.go` |
| 2.5 | `ensurePeriodLocked` teruskan `ratio` ke `GetOrCreatePeriod` | `billing/ledger.go` |
| 2.6 | Unit test invarian (I-1 … I-8) | `billing/ledger_ratio_test.go` |

### 4.2 Kontrak fungsi (sesudah)

```go
// Semua fungsi menerima RAW TOKENS dan mengonversi internal.
// Reservation.Amount SELALU dalam UNIT (dipakai Release).
Reserve(ctx, userID, estRawTokens int64, modality, idempotencyKey) (*Reservation, error)
Commit (ctx, userID, reservationID, rawTokens int64, idempotencyKey) error
Release(ctx, userID, reservationID, idempotencyKey) error
Refund (ctx, userID, rawTokens int64, modality, idempotencyKey) error
```

### 4.3 Kriteria selesai

- [ ] `RoundUnits` benar di semua mode + lantai minimum (test)
- [ ] Rasio 1 ⇒ perilaku **identik** dengan sebelumnya (test regresi)
- [ ] `used_tokens` tidak pernah > `quota_tokens` (test)
- [ ] `cost_idr` **tidak berubah** saat rasio diubah (test I-6)
- [ ] `reserve.Amount` = unit; `commit` melepas **seluruhnya** (test I-4)
- [ ] Semua test lama tetap hijau

### 4.4 Risiko (TERTINGGI)

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Salah satuan (RAW vs unit) tercampur | Kuota salah hitung | Test I-4/I-6; komentar eksplisit di setiap parameter |
| `Release` melepas lebih dari yang ditahan | Kuota bisa negatif | Invarian I-4 + `chk_token_balance` |
| `Commit` memakai unit untuk biaya | Margin salah | Test I-6 (ubah rasio → biaya tetap) |
| Double-konversi (handler & ledger sama-sama konversi) | Kuota terlalu murah | **Hanya ledger** yang mengonversi; handler kirim RAW |

---

## 5. Fase 3 — Repository & Wiring

### 5.1 Tugas

| # | Tugas | File |
|---|---|---|
| 3.1 | `GetOrCreatePeriod` (+`ratio`) | `repository/postgres/billing_repo.go` |
| 3.2 | `getPeriodByStart`/`currentPeriod` baca `quota_ratio` | sama |
| 3.3 | `InsertLedger` tulis `raw_tokens` | sama |
| 3.4 | `GetLedgerByID`/`ByKey`/list baca `raw_tokens` | sama |
| 3.5 | `quotaResolver` kembalikan rasio | `cmd/server/main.go` |
| 3.6 | `GetPlanByID`/`ByCode` baca `token_quota_ratio` | `billing_repo.go` |
| 3.7 | Update semua pemanggil `GetOrCreatePeriod` | — |

### 5.2 Kriteria selesai

- [ ] `go build ./...` sukses
- [ ] Test integrasi repository hijau (rasio tersimpan & terbaca)
- [ ] `go test ./internal/repository/postgres/...` hijau

---

## 6. Fase 4 — Katalog + Lapisan Konfigurasi Dinamis

> **Fase baru** (permintaan user: "semuanya dinamis tanpa hardcode").
> Detail: [09-lapisan-konfigurasi-dinamis.md](./09-lapisan-konfigurasi-dinamis.md).

### 6.1 Tugas

| # | Tugas | File |
|---|---|---|
| 4.1 | Migrasi `0029_config_catalog.sql` (3 tabel) | `database/migrations/` |
| 4.2 | Seed katalog dari konstanta kode saat ini (**nilai sama**) | migrasi 0029 |
| 4.3 | `SettingsCatalog` baca `setting_definitions` (**cache ber-versi**, invalidasi event-driven) | `admin/settings_catalog.go` |
| 4.4 | `admin/settings.go`: `writableSettings` → katalog DB (fallback konstanta) | `admin/settings.go` |
| 4.4b | `ConfigCache` ber-versi + publish `config_changed` (SSE) saat simpan | `admin/config_cache.go` |
| 4.5 | `MetricsCatalog` + `FeatureCatalog` dari DB (+cache) | `admin/catalog.go` |
| 4.6 | `resolverRegistry` + `UsageFor` baca `metric_definitions` | `middleware/usage_resolvers.go` |
| 4.7 | `billing/rounding.go`: `RoundUnits` mode dari setting | `billing/rounding.go` |
| 4.8 | `EstimateStore`: overhead/chars dari setting | `billing/estimate.go` |
| 4.9 | Sentinel unlimited dari setting | `cmd/server/main.go` |
| 4.10 | Default kuota/rasio/label dari setting | `admin/plans.go` |
| 4.11 | Test: nilai dari DB == nilai lama | `*_test.go` |

### 6.2 Kriteria selesai

- [ ] Migrasi 0029 jalan + idempoten
- [ ] Seed menghasilkan katalog **identik** dengan konstanta lama
- [ ] `writableSettings` masih ada sebagai **fallback** (bila tabel kosong)
- [ ] Menambah baris di `setting_definitions` → setting baru bisa diubah **tanpa deploy**
- [ ] Menambah baris di `metric_definitions` → metrik baru muncul di dropdown **tanpa deploy**
- [ ] Mode pembulatan berubah mengikuti setting (test ketiga mode)
- [ ] Semua test lama hijau (nilai sama)

### 6.3 Risiko

| Risiko | Mitigasi |
|---|---|
| Tabel kosong → sistem rusak | **Fallback konstanta permanen** (P-12) |
| Query DB tiap request | Cache 30s |
| Seed menimpa penyesuaian operator | `ON CONFLICT ... WHERE label = ''` |

---

## 7. Fase 5 — API Admin + Kalkulator

### 7.1 Tugas

| # | Tugas | File |
|---|---|---|
| 5.1 | `planDTO`/`planRequest` +`token_quota_ratio` | `admin/plans.go` |
| 5.2 | Validasi rasio (1–1000) | `admin/plans.go` |
| 5.3 | `POST /admin/plans/estimate` | `admin/plans.go` (atau file baru `admin/quota_estimate.go`) |
| 5.4 | Katalog dari DB +`description`, `ratioAdjusted` | `admin/catalog.go` |
| 5.5 | `GET /me/entitlements` +`unit` per usage | `handler/usage_source.go` |
| 5.6 | Ledger admin +`raw_tokens` | `admin/tokens.go` |
| 5.7 | Route baru (katalog + estimate) | `admin/routes.go` |
| 5.8 | Test handler | `admin/*_test.go` |

### 7.2 Kriteria selesai

- [ ] `POST /admin/plans/estimate` mengembalikan kuota + biaya + margin + warnings
- [ ] Estimasi **read-only** (tidak mengubah data)
- [ ] Validasi rasio < 1 ditolak 400
- [ ] `go test ./internal/admin/...` hijau
- [ ] Contoh curl di [03 §9](./03-spesifikasi-api.md) berhasil

---

## 8. Fase 6 — UI Super Admin

### 8.1 Tugas

| # | Tugas | File |
|---|---|---|
| 5.1 | `LimitKeySelect` (dropdown katalog) | `components/super-admin/plans/` |
| 6.2 | `FeatureKeySelect` | sama |
| 6.3 | `QuotaRatioInput` | sama |
| 6.4 | `QuotaSimulationPanel` (inline) | sama |
| 6.5 | Integrasi ke `PlanFormDialog` | `PlanFormDialog.tsx` |
| 6.6 | Halaman `/super-admin/kuota` (kalkulator) | `app/(dashboard)/super-admin/kuota/page.tsx` |
| 6.7 | `useQuotaEstimate` hook | `hooks/` |
| 6.8 | `useSuperAdminPlans` +`token_quota_ratio` | `hooks/useSuperAdminPlans.ts` |
| 6.9 | `useEntitlementCatalog` + katalog dinamis | `hooks/useEntitlementCatalog.ts` |
| 6.10 | Halaman `/super-admin/settings`, `/metrics`, `/features` | `app/(dashboard)/super-admin/` |
| 6.11 | Menu sidebar Super Admin | `SuperAdminSidebar.tsx` |

### 8.2 Kriteria selesai

- [ ] Operator **tidak bisa** mengetik key sembarangan (dropdown)
- [ ] Setting/metrik/fitur dirender dari katalog DB
- [ ] Peringatan "belum ditegakkan" muncul untuk limit non-enforced
- [ ] Simulasi tampil otomatis saat rasio/kuota berubah
- [ ] Halaman kalkulator menampilkan kuota + biaya + margin + peringatan
- [ ] `pnpm lint` tanpa error baru
- [ ] Aksesibel (label, `aria-live`, kontras)

---

## 9. Fase 7 — UI User

### 9.1 Tugas

| # | Tugas | File |
|---|---|---|
| 7.1 | Label "Token AI" → "Kuota AI" | `QuotaUsageCard.tsx` |
| 7.2 | Catatan "kuota diukur dalam unit" | `QuotaUsageCard.tsx` |
| 7.3 | Badge chat "Sisa kuota" | `ChatWindow.tsx` |
| 7.4 | `LIMIT_MESSAGES` → "Kuota AI" | `lib/entitlement-errors.ts` |
| 7.5 | `UsageEntry.unit` | `hooks/use-entitlements.ts` |

### 9.2 Kriteria selesai

- [ ] Tidak ada lagi kata "token" di UI user untuk `ai_tokens_per_cycle`
- [ ] Angka kuota tampil dengan satuan yang benar
- [ ] Test a11y tetap hijau

---

## 10. Fase 8 — Dokumentasi & Rollout

### 10.1 Tugas

| # | Tugas |
|---|---|
| 8.1 | ADR baru: "Rasio token per plan + konfigurasi dinamis" di `DECISION_LOG.md` |
| 8.2 | Update `second_brain/0_Inbox/brief_awal.md` bila perlu |
| 8.3 | Regenerate `database/dump/fintrack_v2.sql` |
| 8.4 | Log aktivitas |
| 8.5 | Panduan operator: settings, metrik, rasio, biaya |
| 8.6 | **Perbaiki dump** yang belum memuat kolom migrasi 0018 (lihat catatan) |

### 10.2 Catatan penting: dump tertinggal

Ditemukan saat audit: `database/dump/fintrack_v2.sql:441` **belum** memuat kolom
`provider_cost_per_1k_idr` dan `price_unit` dari migrasi **0018**. Dump harus
di-regenerate agar konsisten. Ini tugas wajib di fase 7.

### 10.3 Urutan rollout yang disarankan

```
1. Deploy F1–F3 (backend, rasio=1)     → perilaku TIDAK berubah, aman
2. Verifikasi produksi 1–2 hari        → pastikan tidak ada regresi
3. Deploy F4 (katalog dinamis)         → nilai identik, kebijakan pindah ke DB
4. Deploy F5–F7 (API + UI)             → operator melihat semua editor
5. Operator isi provider_cost_per_1k   → margin terukur
6. Operator set rasio (Free 5, dst.)   → berlaku siklus BERIKUTNYA
7. Monitor /super-admin/quota-monitor  → pastikan pemakaian wajar
```

---

## 11. Checklist global

### 11.1 Backend

- [ ] `go vet ./...` bersih
- [ ] `go build ./...` bersih
- [ ] `go test ./...` hijau
- [ ] Test invarian rasio (I-1 … I-8) hijau
- [ ] Test regresi rasio=1 (perilaku identik) hijau
- [ ] Test concurrency kuota tetap hijau

### 11.2 Database

- [ ] Migrasi maju + ulang (idempoten)
- [ ] Backfill terverifikasi
- [ ] Dump di-regenerate
- [ ] Tidak ada secret di dump

### 11.3 Frontend

- [ ] `pnpm lint` (tanpa error baru)
- [ ] `pnpm test` hijau (termasuk test baru)
- [ ] `pnpm build` sukses
- [ ] Test a11y hijau

### 11.4 Dokumentasi

- [ ] ADR ditulis
- [ ] Log aktivitas diperbarui
- [ ] Panduan operator tersedia

---

## 12. Definisi Selesai (DoD)

Fitur dianggap selesai bila **semua** terpenuhi:

1. Operator dapat mengubah kuota, fitur, **dan rasio** per plan dari Super Admin
   **tanpa menyentuh kode atau SQL**.
2. **Seluruh kebijakan dinamis**: default kuota/rasio, mode pembulatan, label satuan,
   overhead estimasi, sentinel unlimited, biaya provider default — semuanya dari
   setting di DB, dapat diubah operator.
3. **Katalog dinamis**: setting/metrik/fitur dirender dari tabel katalog; menambah
   entri baru **tidak** butuh deploy (kecuali resolver yang benar-benar baru).
4. Kalkulator menampilkan kuota, biaya, margin, dan peringatan secara akurat.
5. User Free dapat melakukan **≥ 2 chat** per siklus (bukan 0).
6. Biaya Rupiah **tidak berubah** oleh rasio (terverifikasi test).
7. Rasio default 1 ⇒ sistem identik dengan sebelum perubahan (test regresi).
8. **Tabel katalog kosong tidak merusak sistem** (fallback konstanta permanen).
9. Semua gate (vet/build/test/lint) hijau.
10. Dokumentasi (ADR + panduan + log) lengkap.

---

## 13. Perkiraan waktu

| Fase | Perkiraan | Catatan |
|---|---|---|
| F1 Migrasi (kolom) & Domain | 1–2 hari | Termasuk uji idempoten |
| F2 Ledger | 1–2 hari | Paling sensitif; banyak test |
| F3 Repository & Wiring | 0,5–1 hari | Mekanis |
| **F4 Katalog + Lapisan Dinamis** | **1–2 hari** | 3 tabel + 3 resolver katalog + rounding |
| F5 API Admin | 1–2 hari | Termasuk kalkulator + katalog API |
| F6 UI Super Admin | 1,5–2,5 hari | Plan + settings + metrik + fitur |
| F7 UI User | 0,5–1 hari | Perubahan label |
| F8 Docs & Rollout | 1 hari | Termasuk regenerate dump |
| **Total** | **8–13 hari kerja** | |

---

**Navigasi:** [← Model Biaya](./05-model-biaya-dan-margin.md) · [Rencana Testing →](./07-rencana-testing.md)
