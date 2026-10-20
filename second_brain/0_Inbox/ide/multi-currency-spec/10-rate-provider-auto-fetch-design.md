# Multi-Currency Rate Provider & Auto-Fetch Sync Design (Fully Dynamic & Admin-Configured)

**Tanggal:** 2026-10-20  
**Status:** In Review  
**Dokumen Induk:** `docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md`  
**Terkait:** `second_brain/0_Inbox/ide/multi-currency-spec/` (Keputusan D5: auto-fetch provider)

---

Lihat detail lengkap spesifikasi teknis arsitektur di:
[docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md](../../../../docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md)

### Ringkasan Desain (100% Dinamis & Super Admin Configured):
1. **Zero Hardcoded Parameters**:
   - Seluruh knob operasional terdaftar di tabel `settings` dan dikelola via Super Admin Panel (`GET/PUT /admin/settings`):
     - `currency_auto_sync_enabled`: sakelar on/off sinkronisasi otomatis.
     - `currency_sync_interval_hours`: interval background sync (1-168 jam).
     - `currency_rate_provider`: pilihan driver provider aktif (`open_er_api`, `floatrates`, `frankfurter`, `custom`).
     - `currency_provider_base_url` & `currency_provider_api_key`: endpoint kustom dan secret API key.
     - `currency_sync_timeout_seconds`: batas waktu timeout HTTP.
     - `currency_dedup_hours`: jendela deduplikasi nilai kurs.
2. **Interface Adapter & Factory Dinamis**:
   - `RateProvider` interface di Go backend.
   - `ProviderFactory` me-resolve driver provider aktif secara instan berdasarkan database setting melalui `runtimeconfig.Resolver` (DB-over-env).
3. **On-Demand Auto-Fetch saat Input Mata Uang**:
   - Super Admin memasukkan kode mata uang (misal `JPY`), memilih `rate_source = 'provider'`, dan centang `is_active = true`.
   - `currency_usecase.UpsertCurrency` otomatis memanggil provider aktif untuk mengambil kurs hari ini dan menyimpannya ke `currency_rates`.
   - Mata uang langsung aktif tanpa perlu admin menginput kurs secara manual.
4. **Scheduled Sync Worker Dinamis (`backend/internal/currency/rate_sync_worker.go`)**:
   - Background worker berjalan membaca interval dan flag enabled dinamis.
   - Mengambil kurs terbaru untuk semua mata uang aktif ber-`rate_source = 'provider'`.
   - Menyimpan nilai baru ke tabel `currency_rates` (append-only history) dengan deduplikasi jika rate belum berubah.
5. **Resilience & Fallback**:
   - Timeout dinamis via `currency_sync_timeout_seconds`.
   - Jika provider eksternal offline saat background sync, worker mencatat log `WARN` dan sistem otomatis fallback ke kurs terakhir di database (`RateAsOf`). Super Admin dapat mengganti provider kapan saja dari panel dashboard jika salah satu provider bermasalah.
