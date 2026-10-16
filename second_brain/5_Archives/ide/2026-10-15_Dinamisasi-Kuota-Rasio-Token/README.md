# Dinamisasi Kuota, Matriks Fitur, & Rasio Token per Plan

| Field | Isi |
|---|---|
| **Dokumen** | Desain, Spesifikasi, & Rencana Implementasi+Testing |
| **Tanggal** | 2026-10-15 |
| **Status** | ✅ **Diimplementasikan** (2026-10-15) — migrasi 0028/0029, backend + frontend; lihat ADR-012 |
| **Pemilik** | Tim FinTrack v2 |
| **Konteks** | Lanjutan `../2026-10-15_AI-Chat-Kuota-Render-Sesi/00-audit-ai-chat.md` |
| **Keputusan terkait** | ADR-002 (ledger), ADR-003 (uang integer), ADR-011 (radius) |

---

## Masalah yang diselesaikan

1. **Kuota & fitur plan masih terasa hardcode.** Nilai sudah tersimpan di tabel
   (`plan_limits`, `plan_entitlements`) dan bisa diedit lewat Super Admin, tetapi
   editor masih memakai **input teks bebas** untuk key sehingga rawan typo
   (`ai_chatt` → fitur mati diam-diam), dan tidak ada tempat untuk melihat
   **dampak** sebuah konfigurasi.
2. **Kuota Free 5.000 token habis dalam satu chat.** Satu percakapan nyata memakai
   ~5.000–10.000 token (system prompt + tools + RAG + riwayat 20 pesan). Angka kuota
   yang terlihat "kecil" bertabrakan dengan biaya token nyata yang besar.
3. **Tidak ada pemisahan antara "kuota yang dilihat user" dan "biaya nyata yang
   kita bayar".** Keduanya kini sama-sama token mentah, sehingga menaikkan kuota
   agar user nyaman berarti menaikkan eksposur biaya tanpa kendali.
4. **Tidak ada perhitungan di sisi Super Admin** untuk menjawab: *"kalau kuota Free
   saya set 5.000, user bisa chat berapa kali, dan berapa biaya saya?"*

## Solusi yang diusulkan (ringkas)

Memperkenalkan **rasio token per plan** — pemisahan eksplisit antara dua besaran:

| Besaran | Arti | Dipakai untuk |
|---|---|---|
| **Token provider (raw)** | Yang dilaporkan model upstream | Menghitung **biaya Rupiah** & margin |
| **Unit kuota plan** | `raw / rasio` | Membandingkan dengan **kuota plan** yang dilihat user |

Contoh (rasio Free = 5):

```
Chat nyata          : 9.945 token provider
Rasio plan Free     : 5
Ditagih ke kuota    : ceil(9.945 / 5) = 1.989 unit
Kuota Free 5.000    : sisa 3.011 unit  → masih bisa ~2-3 chat ✅
Biaya Rupiah        : tetap dihitung dari 9.945 token nyata (margin akurat)
```

Kuota tetap tampil sebagai angka bulat yang ramah ("Sisa 3.011 dari 5.000"),
sementara bisnis tetap melihat biaya sebenarnya.

## Daftar dokumen

| # | Dokumen | Isi |
|---|---|---|
| [00](./00-ringkasan-eksekutif.md) | Ringkasan Eksekutif | Masalah, solusi, dampak, keputusan yang diminta |
| [01](./01-desain-teknis.md) | Desain Teknis | Alur metering baru, titik konversi, invarian, keputusan arsitektur |
| [02](./02-skema-database-dan-migrasi.md) | Skema & Migrasi | DDL lengkap, pola expand→migrate→contract, backfill, indeks |
| [03](./03-spesifikasi-api.md) | Spesifikasi API | Endpoint admin & user, request/response, validasi, error |
| [04](./04-desain-ui.md) | Desain UI | Editor plan (dropdown katalog), kalkulator kuota, tampilan user |
| [05](./05-model-biaya-dan-margin.md) | Model Biaya & Margin | Rumus, contoh perhitungan, simulasi skenario |
| [06](./06-rencana-implementasi.md) | Rencana Implementasi | Fase bertahap, checklist per fase, urutan deploy |
| [07](./07-rencana-testing.md) | Rencana Testing | Unit, integrasi, e2e, concurrency, regresi |
| [08](./08-risiko-dan-rollback.md) | Risiko & Rollback | Risiko, mitigasi, prosedur rollback, migrasi data |
| [09](./09-lapisan-konfigurasi-dinamis.md) | **Lapisan Konfigurasi Dinamis** | Katalog setting/metrik/fitur jadi data; pembulatan & default dinamis; seam yang diakui |

## Keputusan yang sudah dikunci (dari user)

| # | Keputusan | Nilai |
|---|---|---|
| K-1 | Mekanisme | **Rasio token**, biaya tetap dari token nyata |
| K-2 | Cakupan rasio | **Per-plan**, dinamis (kolom di tabel `plans`) |
| K-3 | Tampilan user | Satuan **"kuota"** (dinamis via setting) |
| K-4 | Pengerjaan | Dokumen desain dulu (dokumen ini) |
| **K-5** | **Prinsip utama** | **SEMUA dinamis dari Super Admin — tanpa hardcode** |

### Jawaban keputusan P-1 … P-16 (sudah dikonfirmasi user)

| # | Jawaban |
|---|---|
| P-1 | Pembulatan = `ceil` → **setting dinamis** `quota_rounding_mode` |
| P-2 | Rasio = **dinamis per plan**, jangan hardcode |
| P-3 | Label satuan = **"kuota"** → setting `quota_unit_label` |
| P-4 | Plan unlimited = **tetap dibagi rasio** |
| P-5 | Migrasi plan lama = **default 1** |
| P-6 | Backfill = **Pilihan B** (tanpa UPDATE, jaga append-only) |
| P-7 | Constraint = **`NOT VALID` bila > 10 juta baris** |
| P-8 | Seed rasio = **serahkan ke operator** (tidak ada migrasi seed) |
| P-9 | Kuota default = **10.000**, dinamis untuk semua plan |
| P-10 | Harga Pro/Business = **evaluasi setelah biaya nyata**, dinamis |
| P-11 | Biaya provider = **WAJIB**, diisi **default dinamis** |
| P-12 | Konstanta lama = **simpan sebagai fallback permanen** |
| P-13 | Kesegaran konfigurasi = **invalidasi event-driven (SSE + Redis pub/sub)**; TTL 5 menit hanya jaring terakhir |
| P-14 | Resolver baru = boleh via kode (seam diakui) |
| P-15 | Lantai tagih minimum = **ada** (default 1, dinamis) |
| P-16 | Hapus definisi = **soft delete** |

## Yang sudah ada vs yang baru

| Komponen | Status | Catatan |
|---|---|---|
| Tabel `plans` | ✅ Ada | Perlu +1 kolom: `token_quota_ratio` |
| Tabel `plan_limits` | ✅ Ada | Nilai kuota per plan |
| Tabel `plan_entitlements` | ✅ Ada | Matriks fitur per plan |
| Tabel `pricing_rules` | ✅ Ada | Jual/beli per 1k/10k, markup, per-modalitas |
| `PricingEngine` | ✅ Ada | Hitung biaya & margin dari token |
| `token_ledger` | ✅ Ada | Perlu +1 kolom: `raw_tokens` |
| `EstimateStore.ForChat` | ✅ Baru (2026-10-15) | Reserve proporsional |
| Katalog entitlement | ⚠️ **Hardcode** | `entitlement/catalog.go` → pindah ke `metric_definitions`/`feature_definitions` |
| Allowlist setting | ⚠️ **Hardcode** | `admin/settings.go` `writableSettings` → pindah ke `setting_definitions` |
| Pemetaan usage resolver | ⚠️ **Hardcode** | `usage_resolvers.go` `UsageFor()` → pindah ke `metric_definitions.resolver_key` |
| **Rasio token** | ❌ **Baru** | Inti dokumen ini |
| **Editor limit berbasis dropdown** | ❌ **Baru** | Mengganti input teks bebas |
| **Kalkulator kuota di Super Admin** | ❌ **Baru** | Simulasi "berapa chat per kuota" |
| **Lapisan konfigurasi dinamis** | ❌ **Baru** | 3 tabel katalog ([09](./09-lapisan-konfigurasi-dinamis.md)) |

## Prasyarat baca

- `backend/internal/billing/ledger.go` — ledger 2-fase
- `backend/internal/billing/domain.go` — entitas & sentinel `-1 = unlimited`
- `backend/internal/billing/estimate.go` — estimasi reserve
- `backend/internal/admin/plans.go` — CRUD plan
- `database/migrations/README.md` — **aturan migrasi mengikat**
- `second_brain/3_Areas/dokumentasi/DECISION_LOG.md` — ADR-002/003/011

---

> **Catatan penting:** dokumen ini adalah *proposal*. Tidak ada kode atau skema yang
> diubah sampai fase implementasi disetujui.

---

## Status Implementasi (2026-10-15)

| Fase | Status | Bukti |
|---|---|---|
| F1 Migrasi + Domain | ✅ | Migrasi `0028`/`0029` dijalankan + idempoten diuji pada DB |
| F2 Ledger (konversi) | ✅ | 7 test invarian (`ledger_ratio_unit_test.go`) PASS tanpa DB |
| F3 Repository + Wiring | ✅ | `go build`/`go vet` bersih |
| F4 Katalog Dinamis | ✅ | `configcatalog` + `UsageFor` data-driven + event `config_changed` |
| F5 API Admin | ✅ | `/admin/plans/estimate`, katalog, `/admin/resolvers`, `/admin/config/reload` + 4 test |
| F6 UI Super Admin | ✅ | Halaman `/super-admin/kuota` & `/super-admin/metrics`, `QuotaSettingsTab` dinamis |
| F7 UI User | ✅ | Label "Kuota AI", catatan satuan, badge chat |
| F8 Verifikasi | ✅/⚠️ | `go vet`/`go build`/`go test ./internal/...` hijau; `pnpm test` 338 passed; `pnpm build` sukses; `pnpm lint` error **hanya** di file fitur voice yang belum di-commit (bukan pekerjaan ini) |

### Verifikasi lengkap (DB hidup, 2026-10-15)

| Perintah | Hasil |
|---|---|
| `go vet ./...` | ✅ bersih |
| `go build ./...` | ✅ bersih |
| `go test ./internal/...` | ✅ semua ok |
| `go test ./internal/repository/postgres/ -run TestLedger` | ✅ **11/11 PASS** (termasuk concurrency + regresi) |
| `go test ./tests/` (e2e) | ✅ **PASS** (setelah migrasi dijalankan di `fintrack_test`) |
| `pnpm test` | ✅ 338 passed (43 files) |
| `pnpm build` | ✅ sukses, 36/36 halaman (termasuk `/super-admin/kuota`, `/super-admin/metrics`) |
| `pg_dump` | ✅ dump di-regenerate (termasuk kolom migrasi `0018` yang tadinya hilang) |

### Bug nyata yang ditemukan test (dan diperbaiki)

1. **`settingInt` memperlakukan `0` sebagai "unset"** — overhead estimasi 0 (nilai sah)
   diabaikan, membuat reserve selalu 3800. Ditemukan oleh test e2e
   `TestP08_001_TokenQuotaConcurrency`. Diperbaiki: baca nilai mentah agar bisa
   membedakan "absen" dari "0".
2. **Migrasi belum dijalankan di `fintrack_test`** — test e2e memakai DB `fintrack_test`,
   bukan `fintrack`. Setelah migrasi `0028`/`0029` dijalankan di sana, semua e2e PASS.

### Tidak dapat diverifikasi (di luar pekerjaan ini)

- **Test voice** (`TestVoiceRepo_*`): tabel `voice_daily_usage` belum ada — bagian dari
  fitur voice yang belum selesai (uncommitted sebelum sesi ini), bukan regresi fitur ini.
- **`pnpm lint`**: 10 error, semuanya di file fitur voice yang belum di-commit.

### Catatan penting

- Migrasi awal bernomor `0025`/`0026` **bentrok** dengan fitur voice yang belum di-commit
  (sudah memakai `0025`). Migrasi fitur ini di-renumber ke **`0028`/`0029`**.
