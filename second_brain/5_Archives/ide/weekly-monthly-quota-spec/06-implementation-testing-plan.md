# Implementation & Testing Plan

Dokumen ini mendefinisikan urutan implementasi bertahap, dependensi antar-task, strategi pengujian, dan kriteria penerimaan untuk sistem kuota dwi-siklus.

---

## 1. Fase Implementasi

### Fase 1: Database & Core Backend (Hari 1-2)

| # | Task | File | Dependensi |
|---|------|------|-----------|
| 1.1 | Tulis migrasi `0036_weekly_monthly_quota.sql` | `database/migrations/0036_weekly_monthly_quota.sql` | — |
| 1.2 | Jalankan migrasi, verifikasi schema | `database/scripts/migrate.sh` | 1.1 |
| 1.3 | Regenerate dump | `database/dump/fintrack_v2.sql` | 1.2 |
| 1.4 | Implementasi `UsageCounterRepository` (Postgres) | `backend/internal/repository/postgres/usage_counter_repo.go` | 1.2 |
| 1.5 | Implementasi `CycleConfigRepository` (Postgres) | `backend/internal/repository/postgres/cycle_config_repo.go` | 1.2 |
| 1.6 | Unit test repository (GetOrCreate, Increment, race safety) | `backend/internal/repository/postgres/usage_counter_repo_test.go` | 1.4 |
| 1.7 | Implementasi `DualCycleEnforcer` service | `backend/internal/quota/enforcer.go` | 1.4, 1.5 |
| 1.8 | Unit test enforcer (lazy reset, monthly caps weekly, degraded modes) | `backend/internal/quota/enforcer_test.go` | 1.7 |

### Fase 2: Middleware & API Integration (Hari 2-3)

| # | Task | File | Dependensi |
|---|------|------|-----------|
| 2.1 | Modifikasi `EntitlementMiddleware.WithinLimit` untuk dual-cycle | `backend/internal/delivery/http/middleware/entitlement.go` | 1.7 |
| 2.2 | Modifikasi AI handler untuk degraded mode (model override, throttle context) | `backend/internal/delivery/http/handler/ai_handler.go` | 2.1 |
| 2.3 | Modifikasi transaction handler untuk weekly counter increment | `backend/internal/delivery/http/handler/transaction_handler.go` | 1.4 |
| 2.4 | Implementasi admin cycle config handler (GET/PUT cycles) | `backend/internal/admin/cycle_config.go` | 1.5 |
| 2.5 | Implementasi quota notification config handler | `backend/internal/admin/quota_notification.go` | 1.2 |
| 2.6 | Implementasi `GET /api/v1/me/quota` endpoint | `backend/internal/delivery/http/handler/quota_handler.go` | 1.7 |
| 2.7 | Integrasi di `router.go`: wire enforcer, register routes | `backend/internal/delivery/http/router.go` | 2.1-2.6 |
| 2.8 | Diperluas: `GET /api/v1/me/entitlements` response weekly info | `backend/internal/delivery/http/handler/entitlement_handler.go` | 1.7 |
| 2.9 | Modifikasi `QuotaSyncer` untuk handle weekly anchor pada upgrade | `backend/internal/billing/quota_syncer.go` | 1.4 |
| 2.10 | Integration test: full enforcement flow (reserve → commit → weekly increment) | `backend/tests/quota_dual_cycle_test.go` | 2.1-2.9 |

### Fase 3: SSE Events & Realtime (Hari 3-4)

| # | Task | File | Dependensi |
|---|------|------|-----------|
| 3.1 | Tambah event contracts (`quota_degraded`, `quota_blocked`, `cycle_config_changed`) | `backend/pkg/contracts/events.go` | — |
| 3.2 | Implementasi event emitting di `DualCycleEnforcer` | `backend/internal/quota/enforcer.go` | 3.1, 1.7 |
| 3.3 | Implementasi debounce pada usage_changed emit (5s per user per key) | `backend/internal/realtime/publisher.go` | 3.1 |
| 3.4 | Update `realtime-dispatcher.ts` — event routing baru | `frontend/lib/realtime-dispatcher.ts` | 3.1 |
| 3.5 | Implementasi `quota-toast-handler.ts` — toast notifications | `frontend/lib/quota-toast-handler.ts` | 3.4 |
| 3.6 | In-memory cache patch di SSE handler | `frontend/lib/realtime-dispatcher.ts` | 3.4 |

### Fase 4: Frontend — Admin UI (Hari 4-5)

| # | Task | File | Dependensi |
|---|------|------|-----------|
| 4.1 | Implementasi `usePlanCycles` hook | `frontend/hooks/usePlanCycles.ts` | 2.4 |
| 4.2 | Implementasi `CycleConfigMatrix` + `CycleConfigRow` | `frontend/components/super-admin/plans/CycleConfigMatrix.tsx` | 4.1 |
| 4.3 | Implementasi `DegradedParamsForm` | `frontend/components/super-admin/plans/DegradedParamsForm.tsx` | 4.2 |
| 4.4 | Integrasi tab "Kuota & Siklus" di `PlanFormDialog` | `frontend/components/super-admin/plans/PlanFormDialog.tsx` | 4.2, 4.3 |
| 4.5 | Implementasi quota notification config UI | `frontend/app/(dashboard)/super-admin/plans/page.tsx` | 2.5 |
| 4.6 | Perluasan quota monitor table (kolom weekly, status, countdown) | `frontend/components/super-admin/quota/QuotaMonitorTable.tsx` | 2.6 |
| 4.7 | DOM test: `CycleConfigMatrix.dom.test.tsx` | `frontend/components/super-admin/plans/CycleConfigMatrix.dom.test.tsx` | 4.2 |

### Fase 5: Frontend — User UI (Hari 5-6)

| # | Task | File | Dependensi |
|---|------|------|-----------|
| 5.1 | Implementasi `useQuota` hook | `frontend/hooks/use-quota.ts` | 2.6 |
| 5.2 | Implementasi `DualQuotaBar` — satu metrik, dual bars | `frontend/components/billing/DualQuotaBar.tsx` | 5.1 |
| 5.3 | Implementasi `DualQuotaCard` — container semua metrik | `frontend/components/billing/DualQuotaCard.tsx` | 5.2 |
| 5.4 | Implementasi `QuotaStatusBadge` | `frontend/components/billing/QuotaStatusBadge.tsx` | — |
| 5.5 | Implementasi `QuotaCountdown` (dual weekly+monthly) | `frontend/components/billing/QuotaCountdown.tsx` | — |
| 5.6 | Integrasi `DualQuotaCard` di `/user/usage` page | `frontend/app/(dashboard)/user/usage/page.tsx` | 5.3 |
| 5.7 | Perluasan `/user/subscription` card dengan weekly info | `frontend/app/(dashboard)/user/subscription/page.tsx` | 5.1 |
| 5.8 | Perluasan `Entitlements` TypeScript types (weekly field) | `frontend/hooks/use-entitlements.ts` | 2.8 |
| 5.9 | DOM tests: `DualQuotaCard.dom.test.tsx`, `QuotaCountdown.dom.test.tsx` | `frontend/components/billing/` | 5.3, 5.5 |
| 5.10 | A11y test: `DualQuotaCard.a11y.test.tsx` | `frontend/components/billing/` | 5.3 |

### Fase 6: Verification & Cleanup (Hari 6-7)

| # | Task | Dependensi |
|---|------|-----------|
| 6.1 | `cd backend && go test -v -race ./...` — semua test lulus | Fase 1-3 |
| 6.2 | `cd frontend && pnpm test` — semua test lulus | Fase 4-5 |
| 6.3 | `cd frontend && pnpm build` — TypeScript + build bersih | Fase 4-5 |
| 6.4 | Manual smoke test: subscribe, use quota, observe weekly reset, degraded mode, blocked state | Semua |
| 6.5 | Update log_aktivitas.md | 6.1-6.4 |

---

## 2. Strategi Pengujian

### 2.1. Unit Tests (Backend)

| Area | Test Case | File |
|------|-----------|------|
| UsageCounterRepo | GetOrCreate — window aktif dikembalikan | `usage_counter_repo_test.go` |
| UsageCounterRepo | GetOrCreate — window baru dibuat saat expired | `usage_counter_repo_test.go` |
| UsageCounterRepo | GetOrCreate — concurrent insert, satu menang | `usage_counter_repo_test.go` |
| UsageCounterRepo | IncrementUsage — atomik, fail saat melebihi limit | `usage_counter_repo_test.go` |
| DualCycleEnforcer | monthly_only plan — perilaku identik existing | `enforcer_test.go` |
| DualCycleEnforcer | weekly_monthly — weekly habis, degraded mode aktif | `enforcer_test.go` |
| DualCycleEnforcer | weekly_monthly — monthly caps weekly | `enforcer_test.go` |
| DualCycleEnforcer | weekly_monthly — hard_block saat weekly habis | `enforcer_test.go` |
| DualCycleEnforcer | weekly_monthly — throttle mode, rate check | `enforcer_test.go` |
| DualCycleEnforcer | weekly_monthly — model_downgrade, fallback model set | `enforcer_test.go` |
| DualCycleEnforcer | lazy reset — window baru dibuat, counter=0 | `enforcer_test.go` |
| DualCycleEnforcer | lazy reset — event emitted | `enforcer_test.go` |
| DualCycleEnforcer | unlimited (-1) — bypass semua checks | `enforcer_test.go` |
| DualCycleEnforcer | max_degraded_usage — block saat cap tercapai | `enforcer_test.go` |
| QuotaSyncer | upgrade — weekly anchor set, weekly counter fresh | `quota_syncer_test.go` |
| QuotaSyncer | downgrade — usage melebihi limit baru, blocked | `quota_syncer_test.go` |
| CycleConfigHandler | validasi: weekly_limit > monthly → reject | `cycle_config_test.go` |
| CycleConfigHandler | validasi: throttle tanpa params → reject | `cycle_config_test.go` |

### 2.2. Integration Tests (Backend)

| Skenario | Deskripsi |
|----------|-----------|
| Full flow: token reserve+commit | Reserve → commit → weekly counter increment → monthly counter increment |
| Concurrent token commits | 10 goroutines commit bersamaan, total = sum individual commits |
| Weekly exhaustion → degraded | Reserve saat weekly habis → context berisi degraded info |
| Monthly exhaustion → 402 | Reserve saat monthly habis → 402 response |
| Lazy reset across window | Request di t=0 (window 1), t=8d (window 2) → counter reset |
| Upgrade mid-cycle | Subscribe Free → upgrade to Pro → weekly counter fresh, monthly carried |
| Plan config change | Admin ubah cycle config → existing users terdampak |

### 2.3. DOM Tests (Frontend)

| Komponen | Test Case |
|----------|-----------|
| `DualQuotaCard` | Renders dual progress bars dengan persentase yang benar |
| `DualQuotaCard` | Renders monthly-only metrik tanpa weekly bar |
| `DualQuotaCard` | Warna warning saat ≥ threshold |
| `DualQuotaCard` | Warna destructive saat blocked |
| `QuotaStatusBadge` | Renders label dan ikon sesuai status |
| `QuotaCountdown` | Format countdown: hari+jam, jam+menit, menit saja |
| `QuotaCountdown` | Warna merah saat < 1 jam |
| `CycleConfigMatrix` | Renders semua metrik dari metric_definitions |
| `CycleConfigMatrix` | Toggle siklus → tampilkan/sembunyikan field weekly |
| `CycleConfigMatrix` | Validasi: weekly > monthly → error inline |
| `DegradedParamsForm` | Conditional render: throttle vs model_downgrade vs hard_block |

### 2.4. Accessibility Tests (Frontend)

| Komponen | Test Case |
|----------|-----------|
| `DualQuotaCard` | aria-valuenow, aria-valuemax pada progress bar |
| `DualQuotaCard` | aria-live pada status badge |
| `QuotaCountdown` | aria-label statis pada countdown |
| `CycleConfigMatrix` | Form elements memiliki label |

---

## 3. Perintah Uji Kanonik

```bash
# Backend (Go)
cd backend && go test -v -race ./...

# Frontend unit & DOM (Vitest)
cd frontend && pnpm test

# Frontend type check & build (Next.js)
cd frontend && pnpm build
```

---

## 4. Kriteria Penerimaan (Definition of Done)

### 4.1. Backend

- [X] Migrasi `0036` berhasil tanpa error.
- [X] `DualCycleEnforcer` menegakkan `remaining = min(weekly, monthly)`.
- [X] Lazy reset berfungsi: window baru dibuat saat pertama kali diakses setelah expired.
- [X] Degraded mode: `hard_block`, `throttle`, `model_downgrade` berfungsi sesuai config.
- [X] `max_degraded_usage` di-enforce — block saat cap tercapai.
- [X] `GET /api/v1/me/quota` mengembalikan snapshot lengkap.
- [X] `GET /api/v1/me/entitlements` diperluas dengan weekly info.
- [X] `PUT /api/v1/admin/plans/:planId/cycles` validasi lengkap.
- [X] SSE events emitted pada state transitions.
- [X] `cd backend && go test ./...` → semua PASS.

### 4.2. Frontend

- [X] Tab "Kuota & Siklus" di `PlanFormDialog` berfungsi: save/load cycle config.
- [X] `DualQuotaCard` menampilkan dual bars, countdown, badge.
- [X] Toast notifications muncul pada event degraded/blocked/reset.
- [X] In-memory cache patch untuk usage_changed — zero flicker.
- [X] `cd frontend && pnpm test` → semua PASS.
- [X] `cd frontend && pnpm build` → bersih tanpa error.

### 4.3. End-to-End

- [X] User dengan plan weekly+monthly: request normal → counter increment.
- [X] Weekly habis → degraded mode aktif (throttle/downgrade sesuai config).
- [X] Monthly habis → hard block 402.
- [X] Lazy reset: setelah 7 hari → weekly counter 0.
- [X] Upgrade mid-cycle: usage carried, weekly fresh.
- [X] Admin ubah cycle config → efek langsung pada user (via SSE).
- [X] Countdown timer realtime (per menit tick, warna berubah < 1 jam).
- [X] Halaman `/user/usage` menampilkan DualQuotaCard + history.
- [X] Halaman `/user/subscription` menampilkan info weekly.
- [X] Tidak ada polling berlebih: hanya 1 SSE connection + surgical invalidation.
- [X] Nol karakter emoji di seluruh halaman. Semua ikon SVG `lucide-react`.
