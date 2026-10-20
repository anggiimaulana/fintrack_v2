# Multi-Currency Rate Provider & Auto-Fetch Sync Design

**Tanggal:** 2026-10-20  
**Status:** In Review  
**Dokumen Induk:** `docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md`  
**Terkait:** `second_brain/0_Inbox/ide/multi-currency-spec/` (Keputusan D5: auto-fetch provider)

---

Lihat detail lengkap spesifikasi teknis arsitektur di:
[docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md](../../../../docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md)

### Ringkasan Desain:
1. **Interface Adapter `RateProvider` (`backend/internal/currency/provider.go`)**:
   - Fungsi `FetchRate(ctx context.Context, fromCode string, toCode string) (float64, error)`.
   - Implementasi default open provider (OpenER API / FloatRates) tanpa API key untuk base IDR.
2. **On-Demand Auto-Fetch saat Input Mata Uang**:
   - Super Admin memasukkan kode mata uang (misal `JPY`), memilih `rate_source = 'provider'`, dan centang `is_active = true`.
   - `currency_usecase.UpsertCurrency` otomatis memanggil provider untuk mengambil kurs hari ini dan menyimpannya ke `currency_rates`.
   - Mata uang langsung aktif tanpa perlu admin menginput kurs secara manual.
3. **Scheduled Sync Worker (`backend/internal/currency/rate_sync_worker.go`)**:
   - Background worker berjalan tiap 6 jam (`time.NewTicker`).
   - Mengambil kurs terbaru untuk semua mata uang aktif ber-`rate_source = 'provider'`.
   - Menyimpan nilai baru ke tabel `currency_rates` (append-only history) dengan deduplikasi jika rate belum berubah.
4. **Resilience & Fallback**:
   - Timeout ketat 8 detik.
   - Jika provider eksternal offline saat background sync, worker mencatat log `WARN` dan sistem otomatis fallback ke kurs terakhir di database (`RateAsOf`).
