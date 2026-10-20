# Multi-Currency Rate Provider & Auto-Fetch Sync Design (Fully Dynamic & Admin-Configured)

**Date:** 2026-10-20  
**Status:** In Review  
**Target Subsystem:** `backend/internal/currency`, `backend/internal/admin`, `backend/internal/runtimeconfig`, `backend/internal/delivery/http/handler`, `frontend/app/(dashboard)/super-admin`  
**Related Specs:** `second_brain/0_Inbox/ide/multi-currency-spec/` (Decision D5: provider auto-fetch)

---

## 1. Problem & Context

Saat ini FinTrack v2 mendukung multi-currency (IDR sebagai base currency, JPY sebagai target rilis pertama). Namun:
1. Penambahan mata uang baru mengharuskan Super Admin memasukkan kurs secara manual sebelum mata uang dapat diaktifkan (`ErrCurrencyActivation` / `422 currency_no_rate`).
2. Belum ada mekanisme pembaruan kurs otomatis berkala dari provider pasar.
3. **Prinsip Utama FinTrack**: Tidak boleh ada nilai operasional yang di-hardcode (URL provider, interval sync, timeout, status aktif/nonaktif worker, maupun API key). Seluruh parameter harus dinamis dan dapat diatur langsung oleh Super Admin melalui antarmuka admin dan tabel `settings`.

Tujuan: Mengotomatisasi penarikan kurs saat mata uang dibuat dan menyediakan background sync periodik dengan arsitektur 100% dinamis berbasis runtime configuration (DB-over-env).

---

## 2. Goals & Non-Goals

### Goals
1. **Zero Hardcoded Parameters**:
   - Provider URL, provider aktif, interval sinkronisasi, timeout HTTP, toleransi deduplikasi, dan sakelar worker dapat diubah langsung oleh Super Admin tanpa deploy ulang.
2. **On-Demand Auto-Fetch saat Create**:
   - Ketika Super Admin menambahkan mata uang dengan `rate_source: "provider"`, sistem secara otomatis mengambil kurs pasar terkini dari provider yang sedang aktif, menyimpan baris kurs awal ke `currency_rates`, dan langsung mengaktifkan mata uang tanpa input manual.
3. **Dynamic Scheduled Sync Worker**:
   - Background worker membaca `currency_auto_sync_enabled` dan `currency_sync_interval_hours` secara dinamis melalui `runtimeconfig.Resolver`. Jika dinonaktifkan oleh Super Admin, worker berhenti melakukan fetch. Jika interval diubah, ritme eksekusi menyesuaikan tanpa restart server.
4. **Append-Only History Preservation**:
   - Setiap kurs baru yang ditarik dicatat sebagai baris baru di tabel `currency_rates` untuk menjaga audit trail historis.
5. **Multi-Provider Adapter (Pluggable)**:
   - Mendukung pemilihan provider dinamis (`open_er_api`, `floatrates`, `frankfurter`, atau `custom` via base URL & API key).
6. **Graceful Fallback & Deduplication**:
   - Jika provider eksternal offline saat worker berjalan, sistem mencatat log warning dan transaksi pengguna tetap aman menggunakan kurs terakhir yang valid di database (`RateAsOf`).
   - Melewati insert jika nilai kurs belum berubah signifikan pada hari yang sama.

### Non-Goals
1. Tidak ada client-side polling dari frontend.
2. Tidak mengubah base accounting currency (tetap `IDR`).
3. Tidak mendukung override kurs manual per transaksi oleh pengguna reguler.

---

## 3. Dynamic Configuration Registry (Super Admin Settings)

Parameter baru didaftarkan ke tabel `settings` dan `backend/internal/admin/settings.go` (`writableSettings`):

| Key Setting | Tipe | Default | Batasan / Enum | Deskripsi |
|---|---|---|---|---|
| `currency_auto_sync_enabled` | `bool` | `true` | boolean | Sakelar global untuk mengaktifkan/menonaktifkan background sync worker |
| `currency_sync_interval_hours` | `int` | `6` | min: 1, max: 168 (1 minggu) | Frekuensi sinkronisasi kurs otomatis (jam) |
| `currency_rate_provider` | `enum` | `"open_er_api"` | `open_er_api`, `floatrates`, `frankfurter`, `custom` | Driver provider kurs pasar aktif |
| `currency_provider_base_url` | `string` | `""` | maxLen: 256 | URL kustom jika menggunakan proxy/provider mandiri (override default) |
| `currency_provider_api_key` | `secret` | `""` | maxLen: 256 | Kunci API jika provider yang dipilih membutuhkan autentikasi |
| `currency_sync_timeout_seconds` | `int` | `10` | min: 2, max: 60 | Batas waktu timeout HTTP request saat menghubungi provider |
| `currency_dedup_hours` | `int` | `24` | min: 1, max: 72 | Jendela waktu pengecekan deduplikasi kurs identik |

Pengaturan ini otomatis muncul di panel Super Admin Settings (di bawah tab Keuangan/Mata Uang) dan didukung oleh endpoint `GET /admin/settings` serta `PUT /admin/settings`.

---

## 4. Architecture & Data Flow

```
[Super Admin Panel / Settings UI]
       │
       │ Update setting (e.g. interval=4h, provider="floatrates")
       ▼
[Database: settings table]
       │
       ▼ (Cached via runtimeconfig.Resolver TTL 5s)
[Runtime Config Resolver]
       │
       ├────────────────────────────────────────┬──────────────────────────────────────┐
       ▼                                        ▼                                      ▼
[RateProvider Factory]              [RateSyncWorker]                     [CurrencyUsecase]
  - resolves provider driver          - reads enabled flag                 - on-demand fetch on create
  - resolves base URL & secret        - respects dynamic interval          - uses active provider
  - configures dynamic timeout        - skips if disabled
```

---

## 5. Detailed Component Design

### 5.1 Dynamic Provider Interface & Factory (`backend/internal/currency/`)

```go
// RateProvider abstracts external rate lookups.
type RateProvider interface {
    Name() string
    FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error)
}

// ProviderConfig represents dynamic knobs resolved from runtimeconfig.
type ProviderConfig struct {
    Driver         string        // "open_er_api" | "floatrates" | "frankfurter" | "custom"
    BaseURL        string        // Optional custom URL
    APIKey         string        // Optional secret
    Timeout        time.Duration // Dynamic timeout
}

// ProviderFactory builds or resolves the active RateProvider based on dynamic settings.
type ProviderFactory interface {
    ResolveProvider(ctx context.Context) (RateProvider, error)
}
```

Implementasi driver:
1. `OpenERProvider`: `https://open.er-api.com/v6/latest/IDR` (default open fiat).
2. `FloatRatesProvider`: `https://www.floatrates.com/daily/idr.json`.
3. `FrankfurterProvider`: `https://api.frankfurter.app/latest?from=...&to=IDR`.
4. `CustomProvider`: Menggunakan URL dan API Key yang diinputkan Super Admin pada setting.

### 5.2 Dynamic Background Worker (`backend/internal/currency/rate_sync_worker.go`)

- Worker tidak memakai konstanta kaku. Pada setiap iterasi loop (atau saat bangun dari sleep):
  1. Cek `runtimeconfig`: apakah `currency_auto_sync_enabled == true`? Jika `false`, log debug dan tunggu siklus berikutnya.
  2. Baca `currency_sync_interval_hours` dinamis. Jika admin mengubah interval dari 6 jam menjadi 2 jam, worker langsung memperbarui interval sleep berikutnya.
  3. Baca `currency_sync_timeout_seconds` untuk membuat `context.WithTimeout`.
  4. Ambil provider aktif dari `ProviderFactory`.
  5. Jalankan sinkronisasi untuk semua mata uang aktif (`rate_source = 'provider' AND is_base = false`).
  6. Deduplikasi: Cek apakah kurs terbaru di DB dalam `currency_dedup_hours` memiliki nilai yang sama. Jika identik, lewati insert untuk mencegah penumpukan baris.
  7. Jika nilai berbeda: Insert baris baru ke `currency_rates` (`source: "provider"`, `note: "scheduled-sync"`).

### 5.3 On-Demand Auto-Fetch saat Create Mata Uang (`backend/internal/usecase/currency_usecase.go`)

Modifikasi pada `UpsertCurrency`:
1. Jika mata uang baru dibuat atau diubah dengan `m.RateSource == "provider"`:
2. Cek apakah mata uang sudah memiliki rate aktif untuk waktu sekarang (`RateAsOf`).
3. Jika belum ada rate:
   - Ambil provider aktif melalui `ProviderFactory`.
   - Lakukan `FetchRate(ctx, code, "IDR")`.
   - Jika gagal, tolak dengan error ramah: *"Gagal mengambil kurs otomatis dari provider [%s]: %v. Periksa konfigurasi provider di Pengaturan Admin atau gunakan kurs manual."*
   - Jika berhasil, simpan kurs awal ke tabel `currency_rates` (`source: "provider"`, `note: "auto-fetch initial"`).
4. Setelah kurs tersimpan, validasi aktivasi mata uang lolos dan mata uang langsung berstatus **Aktif** (`is_active = true`).

---

## 6. Frontend Super Admin Enhancements

1. **Halaman Super Admin Mata Uang (`frontend/app/(dashboard)/super-admin/currency/page.tsx`)**:
   - Di modal "Mata Uang Baru" / "Ubah Mata Uang", tambahkan opsi pilihan sumber kurs:
     - `Sumber Kurs`: Dropdown / Radio (`Otomatis dari Provider` vs `Manual`).
     - Jika memilih `Otomatis dari Provider`, form tidak mewajibkan admin mengisi kurs awal secara manual.
   - Tambahkan tombol pintasan **"Sinkronkan Semua Kurs Sekarang"** (opsional trigger manual untuk Super Admin).
2. **Halaman Super Admin Pengaturan (`frontend/components/super-admin/settings/`)**:
   - Tambahkan grup pengaturan mata uang untuk mengedit:
     - Toggle Sinkronisasi Otomatis.
     - Interval Sinkronisasi (jam).
     - Pilihan Driver Provider Kurs.
     - Kunci API / Custom Endpoint URL.
     - Batas Waktu Request (detik).

---

## 7. Error Handling & Resilience Matrix

| Skenario | Mekanisme Dinamis |
|---|---|
| Provider eksternal sedang gangguan (HTTP 5xx / timeout) | Worker log `WARN`, lewati siklus. Transaksi pengguna tetap menggunakan kurs terakhir (`RateAsOf`). Admin dapat mengganti provider ke driver cadangan dari dashboard secara instan tanpa deploy. |
| Kuota API provider habis | Super Admin dapat memasukkan API Key baru atau mengganti provider langsung di panel Settings tanpa restart backend. |
| Admin ingin menghentikan sinkronisasi sementara | Cukup matikan toggle `currency_auto_sync_enabled` di admin settings; worker langsung berhenti fetch pada siklus berikutnya. |
| Kode mata uang tidak dikenal provider | Usecase tolak saat `UpsertCurrency` dengan pesan `422`: *"Mata uang tidak didukung oleh provider aktif"*. |

---

## 8. Verification & Testing

1. **Unit Test Dynamic Resolver**:
   - Memastikan perubahan setting di database langsung tercermin pada provider yang di-resolve tanpa restart proses.
2. **Unit Test Pluggable Provider Adapter**:
   - `httptest.Server` lokal untuk masing-masing driver (`open_er_api`, `floatrates`, `frankfurter`, `custom`).
3. **Unit Test Worker**:
   - Uji worker saat enabled vs disabled.
   - Uji penyesuaian interval dinamis.
   - Uji penanganan error jaringan tanpa panic.
   - Uji deduplikasi kurs identik.
4. **Integration Test End-to-End**:
   - `POST /admin/currencies` dengan `rate_source = 'provider'` langsung meng-insert rate dan mengaktifkan mata uang.
   - `PUT /admin/settings` untuk mengubah interval dan provider berhasil diterapkan.
