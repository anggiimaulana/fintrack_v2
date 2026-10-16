# Sistem Kuota Mingguan + Bulanan — FinTrack v2

Spesifikasi teknis lengkap untuk implementasi sistem kuota dwi-siklus (mingguan + bulanan) di FinTrack v2. Sistem ini menambahkan limit mingguan yang dapat dikonfigurasi per fitur per plan, dengan mekanisme degraded mode saat kuota mingguan habis.

---

## Keputusan Desain Kunci

| # | Keputusan | Pilihan |
|---|-----------|---------|
| 1 | Definisi "minggu" | Rolling 7 hari dari timestamp subscribe (presisi menit) |
| 2 | Reset bulanan | Rolling, ikut siklus subscription — bukan awal bulan kalender |
| 3 | Saat weekly habis | Hard block default + configurable degraded mode per fitur per plan |
| 4 | Degraded modes | `hard_block` / `throttle` / `model_downgrade` — dinamis dari DB |
| 5 | Degraded params | `rate_limit_amount`, `rate_limit_window_secs`, `fallback_model`, `max_degraded_usage` |
| 6 | Weekly vs monthly | `remaining = min(weekly_limit - weekly_used, monthly_limit - monthly_used)` |
| 7 | Limit types | Dinamis dari DB (`metric_definitions`) — super admin tambah tanpa deploy |
| 8 | Reset mechanism | Lazy reset on-demand — no cron, atomik dalam satu transaksi |
| 9 | Upgrade/downgrade | Carry usage + extend limit. Weekly mulai fresh. Downgrade = block jika over |
| 10 | User visibility | Dual progress bar, countdown timer, status badge, toast via SSE |
| 11 | Warning threshold | Configurable oleh super admin, default 80% |

---

## Daftar Dokumen

| # | Dokumen | Isi |
|---|---------|-----|
| 01 | [Arsitektur & Data Model](01-arsitektur-data-model.md) | Skema DB, tabel baru (`plan_limit_cycles`, `usage_counters`, `quota_notification_config`), perubahan tabel existing, relasi ERD, strategi migrasi, lazy reset mekanisme detail, backward compatibility. |
| 02 | [Backend Enforcement & API](02-backend-enforcement-api.md) | `DualCycleEnforcer`, modifikasi `WithinLimit` middleware, degraded mode handling, API contracts (`GET /me/quota`, `PUT /admin/plans/:id/cycles`), token reserve/commit integration, subscription lifecycle, input validation & SQL injection prevention. |
| 03 | [Admin Configuration UI](03-admin-configuration-ui.md) | Tab "Kuota & Siklus" di `PlanFormDialog`, matrix konfigurasi per metrik per plan, conditional degraded params form, quota notification config UI, quota monitor enhancement, metric definitions management. |
| 04 | [User-Facing Quota Display](04-user-facing-quota-ux.md) | `DualQuotaCard` dual progress bars, `QuotaStatusBadge`, `QuotaCountdown` timer, toast notifications, halaman `/user/usage` dan `/user/subscription` enhancement, aksesibilitas, dark mode. |
| 05 | [Realtime & SSE Events](05-realtime-sse-events.md) | Event baru (`quota_degraded`, `quota_blocked`, `cycle_config_changed`), payload detail, debounce strategy, `realtime-dispatcher.ts` update, `quota-toast-handler.ts`, in-memory cache patch, cache invalidation strategy. |
| 06 | [Implementation & Testing Plan](06-implementation-testing-plan.md) | 6 fase implementasi (7 hari), dependensi antar-task, unit/integration/DOM/a11y test cases, kriteria penerimaan (Definition of Done). |
| 07 | [Mitigasi Risiko & Security](07-mitigasi-risiko-security.md) | 14 risiko teridentifikasi, SQL injection prevention, authorization matrix, zero downtime deployment (feature flag, gradual rollout), fallback strategies, performance analysis, observability. |

---

## Teknologi & Pola yang Digunakan

- **Backend**: Go, Gin, pgx (PostgreSQL), parameterized queries.
- **Frontend**: Next.js 16, React 19, TanStack Query, `sonner` toast, `lucide-react` ikon, `useSyncExternalStore` clock.
- **Realtime**: Server-Sent Events (SSE), surgical cache invalidation, in-memory patch.
- **Database**: PostgreSQL, `timestamptz`, CHECK constraints, UNIQUE constraints, `ON CONFLICT DO NOTHING`.
- **Design System**: Neobrutalist (border-2, shadow-[4px_4px_0px_0px]), CSS variables, dark mode adaptive.
- **Pola Enforcement**: Lazy reset, atomic increment, monthly caps weekly, fail-closed default.
