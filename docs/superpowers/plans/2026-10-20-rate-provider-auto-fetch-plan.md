# Implementation Plan — Multi-Currency Rate Provider & Auto-Fetch Sync (100% Dinamis)

| Field | Isi |
|---|---|
| **Fitur** | Auto-Fetch Kurs & Scheduled Sync Worker Dinamis |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-20 |
| **Prasyarat** | Multi-currency core (migrasi 0041/0042) aktif; tabel `currencies` & `currency_rates` siap |
| **Spec Desain** | `docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md` |
| **Status** | Siap Dieksekusi |

---

## 1. Tujuan & Definition of Done

Menghilangkan input manual kurs untuk mata uang baru dan menyediakan sinkronisasi otomatis berkala dari provider pasar, di mana **seluruh parameter 100% dinamis dan dikendalikan oleh Super Admin**.

**Definition of Done:**
- [ ] Migrasi `0043_currency_provider_settings.sql` menambahkan setting kontrol ke tabel `settings`.
- [ ] Pengaturan terdaftar di `backend/internal/admin/settings.go` (`writableSettings`) dan dapat diubah via `PUT /admin/settings`.
- [ ] Dump database (`database/dump/fintrack_v2.*`) di-regenerate via `database/dump/dump.sh`.
- [ ] Interface `RateProvider` dan `ProviderFactory` di `backend/internal/currency/` mendukung resolusi driver dinamis (`open_er_api`, `floatrates`, `frankfurter`, `custom`).
- [ ] `currency_usecase.UpsertCurrency` secara otomatis mengambil kurs pasar dari provider aktif jika `rate_source = 'provider'` dan belum ada kurs, mengizinkan mata uang langsung aktif tanpa langkah manual.
- [ ] `RateSyncWorker` berjalan di latar belakang server Go, membaca interval dan sakelar enabled secara dinamis dari `runtimeconfig.Resolver`, dengan deduplikasi kurs identik dan penanganan error graceful.
- [ ] Frontend modal `frontend/app/(dashboard)/super-admin/currency/page.tsx` menyediakan pilihan sumber kurs (Manual vs Otomatis dari Provider).
- [ ] Semua pengujian lolos: `cd backend && go test ./...` dan `cd frontend && pnpm test && pnpm build`.
- [ ] Dokumentasi & log aktivitas diperbarui di `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.

---

## 2. Urutan Eksekusi (Waves)

| Wave | ID | Langkah | Prasyarat |
|---|---|---|---|
| **W1** | S-01 | Migrasi `0043_currency_provider_settings.sql` (seed tabel `settings`) | — |
| **W1** | S-02 | Daftarkan key setting di `backend/internal/admin/settings.go` | S-01 |
| **W1** | S-03 | Regenerate dump database via `database/dump/dump.sh` | S-01 |
| **W2** | P-01 | Interface `RateProvider` & `ProviderFactory` (`backend/internal/currency/provider.go`) | — |
| **W2** | P-02 | Implementasi driver provider (`open_er_api.go`, `floatrates.go`, `frankfurter.go`) | P-01 |
| **W2** | P-03 | Unit test provider adapter dengan `httptest.Server` (`provider_test.go`) | P-02 |
| **W3** | U-01 | Integrasikan provider factory ke `CurrencyUsecase` (`currency_usecase.go`) | P-02 |
| **W3** | U-02 | Logika on-demand auto-fetch pada `UpsertCurrency` (`rate_source = 'provider'`) | U-01 |
| **W3** | U-03 | Unit test usecase auto-fetch (`currency_usecase_test.go`) | U-02 |
| **W4** | W-01 | Background worker `RateSyncWorker` (`backend/internal/currency/rate_sync_worker.go`) | P-02, U-01 |
| **W4** | W-02 | Deduplikasi kurs identik & dynamic sleep loop via `runtimeconfig` | W-01 |
| **W4** | W-03 | Daftarkan worker di `cmd/server/main.go` dengan graceful shutdown | W-02 |
| **W4** | W-04 | Unit test worker (`rate_sync_worker_test.go`) | W-02 |
| **W5** | F-01 | Update modal mata uang di `frontend/app/(dashboard)/super-admin/currency/page.tsx` | U-02 |
| **W5** | F-02 | Tambahkan tab/kontrol pengaturan kurs di Super Admin Settings | S-02 |
| **W6** | V-01 | Verifikasi penuh backend (`go vet`, `go test ./...`, `go build`) | All |
| **W6** | V-02 | Verifikasi penuh frontend (`pnpm test`, `pnpm build`) | All |
| **W6** | V-03 | Log aktivitas di `second_brain/` & git commit | V-01, V-02 |

---

## 3. Detail Implementasi per Langkah

### Wave 1: Skema Database & Pengaturan Dinamis

#### S-01 — Migrasi Pengaturan Kurs Dinamis
- **Target File:** `database/migrations/0043_currency_provider_settings.sql`
- **Tindakan:**
  Tambahkan baris setting baru ke tabel `settings`:
  ```sql
  INSERT INTO settings (key, value, description) VALUES
      ('currency_auto_sync_enabled',   'true'::jsonb,           'Aktifkan pembaruan kurs otomatis di latar belakang (true/false)'),
      ('currency_sync_interval_hours', '6'::jsonb,              'Frekuensi pembaruan kurs otomatis dalam jam (1-168)'),
      ('currency_rate_provider',       '"open_er_api"'::jsonb,  'Driver provider kurs aktif (open_er_api|floatrates|frankfurter|custom)'),
      ('currency_provider_base_url',   '""'::jsonb,             'URL kustom untuk provider kurs atau proxy internal'),
      ('currency_provider_api_key',    '""'::jsonb,             'Kunci API untuk provider kurs berbayar/privat'),
      ('currency_sync_timeout_seconds','10'::jsonb,             'Batas waktu request HTTP ke provider kurs (detik)'),
      ('currency_dedup_hours',         '24'::jsonb,             'Jendela waktu deduplikasi nilai kurs identik (jam)')
  ON CONFLICT (key) DO NOTHING;
  ```

#### S-02 — Daftarkan di `writableSettings`
- **Target File:** `backend/internal/admin/settings.go`
- **Tindakan:** Daftarkan spesifikasi validasi setting baru agar diizinkan diubah melalui `PUT /admin/settings`:
  ```go
  "currency_auto_sync_enabled":    {kind: "bool"},
  "currency_sync_interval_hours":  {kind: "int", min: 1, max: 168},
  "currency_rate_provider":        {kind: "enum", enum: []string{"open_er_api", "floatrates", "frankfurter", "custom"}},
  "currency_provider_base_url":    {kind: "string", maxLen: 256},
  "currency_provider_api_key":     {kind: "string", maxLen: 256},
  "currency_sync_timeout_seconds": {kind: "int", min: 2, max: 60},
  "currency_dedup_hours":          {kind: "int", min: 1, max: 72},
  ```

#### S-03 — Regenerate Database Dump
- **Tindakan:** Jalankan `./database/dump/dump.sh` untuk menyelaraskan skema dump database repository.

---

### Wave 2: Interface Adapter & Multi-Provider Engine

#### P-01 & P-02 — Provider Interface, Factory & Driver Adapters
- **Target File:** `backend/internal/currency/provider.go`
- **Tindakan:**
  1. Definisikan `RateProvider` interface:
     ```go
     type RateProvider interface {
         Name() string
         FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error)
     }
     ```
  2. Definisikan `ProviderFactory`:
     ```go
     type ProviderFactory struct {
         resolver *runtimeconfig.Resolver
         client   *http.Client
     }
     func (f *ProviderFactory) ResolveProvider(ctx context.Context) (RateProvider, error)
     ```
  3. Implementasikan driver `open_er_api` (endpoint: `open.er-api.com/v6/latest/IDR`), `floatrates` (`floatrates.com/daily/idr.json`), dan `custom`.
  4. Lakukan penanganan direct/inverse calculation: jika API memberikan base IDR (1 IDR = 0.00948 JPY), konversikan secara akurat ke rate to base (1 JPY = 105.48 IDR).

#### P-03 — Unit Test Provider
- **Target File:** `backend/internal/currency/provider_test.go`
- **Tindakan:**
  - Mock HTTP response menggunakan `httptest.Server`.
  - Uji kalkulasi rate, respon error 4xx/5xx, respon JSON tidak valid, dan timeout.

---

### Wave 3: On-Demand Auto-Fetch saat Input Mata Uang

#### U-01 & U-02 — Integrasi ke `CurrencyUsecase`
- **Target File:** `backend/internal/usecase/currency_usecase.go`
- **Tindakan:**
  1. Tambahkan dependency `providerFactory *currency.ProviderFactory` pada `CurrencyUsecase`.
  2. Pada method `UpsertCurrency`:
     - Cek apakah `m.IsActive` dan `m.RateSource == "provider"`.
     - Cek apakah kurs saat ini sudah ada (`u.svc.RateAsOf(ctx, code, time.Now())`).
     - Jika belum ada kurs:
       - Resolve active provider: `provider, err := u.providerFactory.ResolveProvider(ctx)`.
       - Ambil kurs: `rateVal, err := provider.FetchRate(ctx, code, currency.BaseCode)`.
       - Jika gagal, kembalikan error terformat: `fmt.Errorf("%w: gagal mengambil kurs otomatis dari %s: %v", ErrCurrencyInvalid, provider.Name(), err)`.
       - Jika berhasil, simpan baris kurs:
         ```go
         u.repo.InsertRate(ctx, currency.Rate{
             CurrencyCode: code,
             RateToBase:   rateVal,
             EffectiveAt:  time.Now(),
             Source:       "provider",
             Note:         toPtr("auto-fetch initial"),
         })
         ```
     - Lanjutkan simpan mata uang dalam status aktif ke `u.repo.UpsertCurrency(ctx, m)`.

#### U-03 — Unit Test Usecase
- **Target File:** `backend/internal/usecase/currency_usecase_test.go`
- **Tindakan:** Uji pembuatan mata uang baru dengan `rate_source = "provider"`, verifikasi pemanggilan provider dan pembuatan baris kurs awal di mock repo.

---

### Wave 4: Dynamic Scheduled Sync Worker

#### W-01 & W-02 — `RateSyncWorker`
- **Target File:** `backend/internal/currency/rate_sync_worker.go`
- **Tindakan:**
  1. Buat struct `RateSyncWorker`:
     ```go
     type RateSyncWorker struct {
         repo     AdminRepo
         factory  *ProviderFactory
         resolver *runtimeconfig.Resolver
         stopChan chan struct{}
     }
     ```
  2. Implementasikan loop dengan dynamic sleep:
     - Baca setting `currency_auto_sync_enabled`. Jika `false`, tidur sejenak lalu evaluasi ulang.
     - Baca `currency_sync_interval_hours`.
     - Query semua mata uang aktif dengan `rate_source = 'provider' AND is_base = false`.
     - Untuk setiap mata uang:
       - Fetch rate via provider aktif.
       - Deduplikasi: bandingkan dengan kurs terbaru di DB dalam `currency_dedup_hours`. Jika identik, lewati.
       - Jika berbeda, insert baris kurs baru (`source: "provider"`, `note: "scheduled-sync"`).
       - Tangani error per-mata uang dengan `logger.Warn` tanpa menghentikan worker.

#### W-03 — Registrasi di `cmd/server/main.go`
- **Target File:** `backend/cmd/server/main.go`
- **Tindakan:**
  - Inisialisasi `ProviderFactory` dan `RateSyncWorker`.
  - Jalankan worker dalam goroutine background: `go rateSyncWorker.Start(ctx)`.
  - Tambahkan cleanup saat server shutdown: `defer rateSyncWorker.Stop()`.

#### W-04 — Unit Test Worker
- **Target File:** `backend/internal/currency/rate_sync_worker_test.go`
- **Tindakan:**
  - Mock repo dan provider. Uji pembaruan kurs, deduplikasi, dan toleransi saat provider gagal.

---

### Wave 5: Frontend Super Admin UI

#### F-01 — Form Pilihan Sumber Kurs
- **Target File:** `frontend/app/(dashboard)/super-admin/currency/page.tsx`
- **Tindakan:**
  1. Tambahkan field `rate_source` pada `CurrencyForm` (opsi: `"provider"` | `"manual"`, default: `"provider"`).
  2. Tambahkan radio button / dropdown pilihan sumber kurs:
     - `Otomatis dari Provider` (rekomendasi).
     - `Manual`.
  3. Berikan teks penjelasan: *"Jika memilih otomatis, sistem akan langsung mengambil kurs pasar terbaru saat disimpan."*

#### F-02 — Kontrol Pengaturan Mata Uang Super Admin
- **Target File:** `frontend/components/super-admin/settings/`
- **Tindakan:** Pastikan grup setting `currency_*` dapat diedit langsung oleh Super Admin (interval sync, pilihan provider, timeout, api key).

---

### Wave 6: Verifikasi & Finalisasi

#### V-01 & V-02 — Uji Komprehensif
- **Backend:** `cd backend && go vet ./... && go test ./... && go build ./...`
- **Frontend:** `cd frontend && pnpm test && pnpm build`

#### V-03 — Dokumentasi & Commit
- Update `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.
- Commit seluruh perubahan ke git.
