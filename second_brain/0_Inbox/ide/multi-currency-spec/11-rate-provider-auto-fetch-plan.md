# Implementation Plan — Multi-Currency Rate Provider & Auto-Fetch Sync (100% Dinamis)

**Tanggal:** 2026-10-20  
**Dokumen Induk:** `docs/superpowers/plans/2026-10-20-rate-provider-auto-fetch-plan.md`  
**Spec Desain:** `docs/superpowers/specs/2026-10-20-rate-provider-auto-fetch-design.md`

---

Lihat detail lengkap urutan wave dan instruksi implementasi di:
[docs/superpowers/plans/2026-10-20-rate-provider-auto-fetch-plan.md](../../../../docs/superpowers/plans/2026-10-20-rate-provider-auto-fetch-plan.md)

### Ringkasan Urutan Wave:
- **Wave 1 (W1)**: Migrasi `0043_currency_provider_settings.sql`, daftarkan key di `backend/internal/admin/settings.go`, dan regenerate dump database.
- **Wave 2 (W2)**: Interface `RateProvider`, `ProviderFactory`, driver adapters (`open_er_api`, `floatrates`, `frankfurter`, `custom`), dan unit test `httptest.Server`.
- **Wave 3 (W3)**: Integrasikan on-demand auto-fetch ke `currency_usecase.UpsertCurrency` dan unit test usecase.
- **Wave 4 (W4)**: Dynamic scheduled sync worker `RateSyncWorker`, deduplikasi, graceful sleep loop via `runtimeconfig`, dan registrasi di `main.go`.
- **Wave 5 (W5)**: Form UI pilihan sumber kurs di `super-admin/currency/page.tsx` dan panel pengaturan admin.
- **Wave 6 (W6)**: Verifikasi penuh backend (`go test`) & frontend (`pnpm build`), update log aktivitas, dan commit.
