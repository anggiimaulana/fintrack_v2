# Multi-Currency Rate Provider & Auto-Fetch Sync Design

**Date:** 2026-10-20  
**Status:** In Review  
**Target Subsystem:** `backend/internal/currency`, `backend/internal/delivery/http/handler`, `frontend/app/(dashboard)/super-admin/currency`  
**Related Specs:** `second_brain/0_Inbox/ide/multi-currency-spec/` (Decision D5: provider auto-fetch)

---

## 1. Problem & Context

Saat ini FinTrack v2 mendukung multi-currency (IDR sebagai base currency, JPY sebagai target rilis pertama). Namun:
1. Penambahan mata uang baru mengharuskan Super Admin memasukkan kurs secara manual sebelum mata uang dapat diaktifkan (`ErrCurrencyActivation` / `422 currency_no_rate`).
2. Belum ada mekanisme pembaruan kurs otomatis berkala dari provider pasar.
3. Skema database sudah menyiapkan kolom `currencies.rate_source` (`manual` | `provider`) dan `currency_rates.source` (`manual` | `provider`), tetapi implementasi adapter provider dan scheduler sinkronisasi belum tersedia.

Tujuan: Mengotomatisasi penarikan kurs saat mata uang dibuat dan menyediakan sinkronisasi berkala di latar belakang (background sync) tanpa membebani frontend dengan polling.

---

## 2. Goals & Non-Goals

### Goals
1. **On-Demand Auto-Fetch saat Create**: Ketika Super Admin menambahkan mata uang dengan `rate_source: "provider"`, sistem secara otomatis mengambil kurs pasar terkini dari provider eksternal, menyimpan baris kurs awal ke `currency_rates`, dan langsung mengaktifkan mata uang tanpa input manual.
2. **Scheduled Sync Worker**: Background worker di Go backend yang berjalan secara periodik (setiap 6 jam) untuk mengambil kurs terbaru bagi semua mata uang aktif ber-`rate_source = 'provider'`.
3. **Append-Only History Preservation**: Setiap penarikan kurs baru disimpan sebagai baris baru di `currency_rates` untuk menjaga keutuhan riwayat transaksi masa lalu (as-of audit trail).
4. **Resilience & Fallback**: Jika provider eksternal down atau timeout saat worker berjalan, sistem mencatat log peringatan dan transaksi pengguna tetap berjalan aman menggunakan kurs terakhir yang valid di database (`RateAsOf`).
5. **Deduplication**: Jika nilai kurs yang diambil dari provider bernilai sama persis dengan kurs terakhir pada hari yang sama, worker melewati insert baru untuk mencegah bloat database.
6. **Pluggable Adapter**: Menghadirkan interface `RateProvider` yang bersih di Go sehingga provider dapat diganti atau ditambahkan (mis. Open Exchange Rates, FloatRates, Frankfurter, Bank Indonesia) tanpa mengubah logika usecase.

### Non-Goals
1. Tidak ada client-side polling dari frontend.
2. Tidak mengubah base accounting currency (tetap `IDR`).
3. Tidak mendukung override kurs manual per transaksi oleh pengguna reguler.
4. Tidak membuat UI admin baru yang rumit; cukup menggunakan kontrol yang ada di halaman Super Admin Currency.

---

## 3. Architecture & Data Flow

```
[Super Admin UI]
       │
       │ POST /admin/currencies (code="JPY", rate_source="provider", is_active=true)
       ▼
[CurrencyHandler / CurrencyUsecase]
       │
       ├─► [RateProvider (FloatRates/OpenER)] ──(HTTP GET)──► External Exchange API
       │        │
       │        ▼ (Rate: 105.42)
       ├─► [CurrencyRepo.InsertRate] ──────► PostgreSQL (currency_rates: source='provider')
       │
       └─► [CurrencyRepo.UpsertCurrency] ──► PostgreSQL (currencies: is_active=true)

[Background RateSyncWorker] (Every 6h via time.NewTicker)
       │
       ├─► [CurrencyRepo.List(active=true)] (Filter: rate_source='provider' AND !is_base)
       │
       ├─► For each currency:
       │        ├─► [RateProvider.FetchRate]
       │        └─► Compare with newest rate in DB (dedup)
       │             └─► [CurrencyRepo.InsertRate] (if changed / new day)
       ▼
[Log & Metrics]
```

---

## 4. Detailed Component Design

### 4.1 Interface `RateProvider` (`backend/internal/currency/provider.go`)

```go
package currency

import (
	"context"
	"errors"
)

var (
	ErrProviderUnavailable = errors.New("rate provider unavailable")
	ErrCurrencyUnsupported = errors.New("currency not supported by provider")
)

type RateProvider interface {
	// FetchRate returns the amount of base currency (IDR) for 1 unit of fromCode.
	// Example: FetchRate(ctx, "JPY", "IDR") returns ~105.5
	FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error)
}
```

### 4.2 Default HTTP Provider Adapter (`backend/internal/currency/open_provider.go`)

Menggunakan open exchange rate endpoint tanpa API key yang stabil untuk target fiat (IDR base).
- Pilihan endpoint: `https://open.er-api.com/v6/latest/IDR` (atau FloatRates fallback `https://www.floatrates.com/daily/idr.json`).
- Respon diparsing ke struktur struct Go dengan HTTP Client yang memiliki timeout maksimal 8 detik.
- Menghitung kurs direct/inverse: bila data kurs berbasis target, konversi secara presisi ke `1 fromCode = X IDR`.

### 4.3 Integrasi ke `CurrencyUsecase` (`backend/internal/usecase/currency_usecase.go`)

Modifikasi pada `UpsertCurrency`:
1. Jika `m.IsActive` bernilai `true` dan `m.RateSource == "provider"`:
2. Cek apakah sudah ada kurs aktif via `u.svc.RateAsOf(ctx, code, time.Now())`.
3. Jika belum ada:
   - Panggil `u.provider.FetchRate(ctx, code, currency.BaseCode)`.
   - Jika gagal, return error terbungkus `ErrCurrencyInvalid` dengan pesan deskriptif.
   - Jika sukses, panggil `u.repo.InsertRate(ctx, currency.Rate{...})` dengan `source: "provider"`, `note: "auto-fetch initial"`.
4. Lolos validasi kurs, dilanjutkan dengan `u.repo.UpsertCurrency(ctx, m)`.

### 4.4 Background Worker (`backend/internal/currency/rate_sync_worker.go`)

- Mengikuti standar worker background FinTrack (menggunakan interface lister/service minimal dan `time.NewTicker`).
- Siklus:
  1. `time.NewTicker(interval)` (default 6 jam, di-inject dari config/main.go).
  2. Ambil semua mata uang aktif dari repository.
  3. Filter: `c.IsActive && c.RateSource == "provider" && !c.IsBase`.
  4. Untuk tiap mata uang:
     - Ambil kurs terbaru di DB via `RateAsOf`.
     - Fetch kurs terbaru dari provider.
     - Jika kurs sama persis dengan kurs hari ini (toleransi selisih 0.0001 dan tanggal efektif hari ini), skip insert.
     - Jika berbeda, insert baris kurs baru:
       `RateToBase: fetchedRate`, `EffectiveAt: time.Now()`, `Source: "provider"`, `Note: "scheduled-sync"`.
  5. Jika ada error jaringan/provider pada salah satu mata uang, catat dengan `logger.Warn` dan lanjutkan ke mata uang berikutnya. Jangan panik atau menghentikan server.

---

## 5. Error Handling & Edge Cases

| Skenario | Penanganan |
|---|---|
| Admin input kode salah/tidak terdaftar (mis. `XYZ`) | Provider return 404 / unsupported -> Usecase tolak dengan 422: "Kode mata uang tidak didukung provider kurs". |
| Provider eksternal down saat admin simpan mata uang | Usecase tolak dengan 422: "Gagal mengambil kurs otomatis dari provider. Periksa koneksi atau masukkan kurs manual." |
| Provider down saat worker sinkronisasi berkala | Worker log `WARN`, lewati siklus. Transaksi pengguna tetap menggunakan kurs terakhir (`RateAsOf`). |
| Perbedaan time zone / jam trading bursa valas | Kurs menggunakan timestamp UTC standar pada kolom `effective_at`. |
| Nilai kurs tidak berubah | Dilewati (deduplikasi) agar tidak membebani kapasitas tabel `currency_rates`. |

---

## 6. Testing Plan

1. **Unit Test Provider (`open_provider_test.go`)**:
   - `httptest.Server` menyimulasikan respon 200 JSON valid.
   - `httptest.Server` menyimulasikan respon 404 mata uang tidak ditemukan.
   - Pengujian timeout context.
2. **Unit Test Worker (`rate_sync_worker_test.go`)**:
   - Mock repository & mock provider.
   - Uji pemanggilan `FetchRate` untuk mata uang `provider`.
   - Pastikan mata uang `manual` diabaikan oleh worker.
   - Pastikan deduplikasi bekerja jika rate identik.
3. **Integration Test Usecase (`currency_usecase_test.go`)**:
   - Uji create mata uang baru dengan `rate_source = "provider"`, pastikan rate langsung tercipta di database dan mata uang langsung berstatus aktif.
