# 05 — Peta Jalan Awal (Fase P02–P08)

Hasil analisis dibagi menjadi **7 fase** di `second_brain/2_Projects/`. Setiap fase punya folder
`analisis/`, `planning/`, `rules/`, `teknis/`, dan `tracking/` (mengikuti pola `simbot/second_brain`).

> **Prinsip:** setiap fase harus memenuhi Definition of Done sendiri sebelum fase berikutnya dimulai
> (sequential), dan setiap fase harus tetap memenuhi **zero downtime** (expand/contract + feature flag).

---

## Ringkasan fase

| Fase | Nama | Fokus utama | Bergantung pada |
|---|---|---|---|
| **P02** | Fondasi Platform | Redis, feature flag, audit log, observability, standar DataTable + URL state, SSE hub dasar | — |
| **P03** | AI Gateway Terpusat | Pool endpoint 9router, tabel routing, circuit breaker, kill switch, metering token | P02 |
| **P04** | Subscription & Token Core | Plan, entitlement, limit, subscription, token ledger, enforcement | P03 |
| **P05** | Payments & Billing | Midtrans + manual, switch gateway, invoice, prorasi, dunning, webhook | P04 |
| **P06** | Super Admin Dashboard | Manajemen user/plan/subscription/pembayaran/token/gateway/audit/settings | P04, P05 |
| **P07** | Rollout RBAC, Realtime & Standar | Terapkan entitlement + SSE + DataTable ke SEMUA halaman & role | P02–P06 |
| **P08** | Hardening & Go-Live | Load test, security review, chaos (failover), compliance, dokumentasi | semua |

---

## P02 — Fondasi Platform

**Kenapa lebih dulu:** semua fase lain bergantung pada cache bersama, feature flag, audit, dan standar
tabel. Tanpa ini, perubahan tidak bisa "zero downtime".

- **Analisis:** dampak Redis pada arsitektur; strategi cache berversi; desain feature flag & audit.
- **Teknis:** `redis-cache.md`, `feature-flags.md`, `audit-log.md`, `datatable-standard.md`,
  `observability.md`, `url-state-convention.md`.
- **Deliverable:** Redis di compose; `internal/audit/`; middleware cache Redis + versi; komponen
  `<DataTable>` (TanStack Table v8) + hook URL state; metrik dasar.

## P03 — AI Gateway Terpusat

**Kenapa:** prasyarat penagihan kuota (harus bisa mengukur & mengarahkan trafik AI).

- **Analisis:** peta provider lama → tabel route; strategi failover & circuit breaker; dimensi embedding.
- **Teknis:** `gateway-architecture.md`, `endpoint-pool.md`, `routing-table.md`, `circuit-breaker.md`,
  `key-rotation.md`, `embedding-strategy.md`.
- **Deliverable:** `internal/aigateway/`; tabel `ai_endpoints`, `ai_routes`; adapter di `ai_usecase.go`;
  kill switch; health check; metrik token & failover.

## P04 — Subscription & Token Core

**Kenapa:** inti monetisasi; butuh metering dari P03.

- **Analisis:** model plan/entitlement/limit; alur ledger reserve/commit/release; harga dinamis.
- **Teknis:** `billing-architecture.md`, `entitlement-matrix.md`, `token-ledger.md`, `pricing-engine.md`,
  `subscription-lifecycle.md`, `enforcement-middleware.md`.
- **Deliverable:** `internal/billing/`, `internal/entitlement/`; tabel `plans`, `plan_entitlements`,
  `plan_limits`, `subscriptions`, `subscription_events`, `token_ledger`, `token_periods`, `pricing_rules`;
  guard entitlement + cache berversi.

## P05 — Payments & Billing

**Kenapa:** mengaktifkan aliran uang; butuh subscription dari P04.

- **Analisis:** perbandingan gateway; keamanan webhook; kebijakan prorata/downgrade; dunning.
- **Teknis:** `payment-integration.md`, `midtrans-webhook.md`, `manual-payment.md`,
  `invoice.md`, `proration.md`, `dunning-grace.md`, `payment-gateway-switch.md`.
- **Deliverable:** `internal/payment/`; tabel `payments`, `invoices`; webhook terverifikasi + idempoten;
  UI checkout + upload bukti; prorata; reminder/grace.

## P06 — Super Admin Dashboard

**Kenapa:** menyatukan kendali; butuh data model P04 & P05.

- **Analisis:** peta menu; kebutuhan metrik; alur approval pembayaran manual.
- **Teknis:** `superadmin-architecture.md`, `admin-menu.md`, `audit-viewer.md`, `metrics-dashboard.md`,
  `charts-interactive.md`.
- **Deliverable:** `app/(dashboard)/super-admin/*`; manajemen plan/subscription/pembayaran/token/gateway/
  audit/settings; chart interaktif; semua tabel pakai `<DataTable>`.

## P07 — Rollout RBAC, Realtime & Standar

**Kenapa:** konsolidasi; memastikan standar berlaku di SEMUA halaman & role.

- **Analisis:** audit kepatuhan halaman lama; peta invalidation; desain event SSE.
- **Teknis:** `rbac-rollout.md`, `sse-strategy.md`, `cache-invalidation.md`, `cursor-pointer-audit.md`,
  `zero-downtime-deploy.md`.
- **Deliverable:** entitlement aktif di semua endpoint; SSE event domain + Redis pub/sub; semua tabel
  standar; audit cursor pointer & a11y; hapus menu API-key user.

## P08 — Hardening & Go-Live

**Kenapa:** membuktikan sistem siap uang nyata.

- **Analisis:** hasil load test, security review, chaos test, gap compliance.
- **Teknis:** `load-test.md`, `security-review.md`, `chaos-failover.md`, `pdp-compliance.md`,
  `runbook-incidents.md`, `backup-dr.md`.
- **Deliverable:** laporan k6 (kuota concurrency, failover, replay webhook); checklist keamanan;
  runbook insiden (gateway down, pembayaran gagal, rotasi kunci); dokumentasi operasional.

---

## Definition of Done umum (semua fase)

- [ ] `go vet && go test && go build` hijau; `pnpm build && pnpm lint && pnpm test` hijau.
- [ ] Migrasi baru replayable dari nol; dump di-regenerate.
- [ ] Fitur baru di balik feature flag; dapat dimatikan tanpa deploy ulang.
- [ ] Ada test yang membuktikan setiap klaim kritis (ledger, failover, idempotensi, entitlement).
- [ ] Audit log mencatat mutasi sensitif.
- [ ] Tidak ada data finansial user yang hilang.
- [ ] `tracking/tracking.md` diperbarui; `log_aktivitas.md` dicatat.

## Urutan dependensi

```
P02 ──▶ P03 ──▶ P04 ──▶ P05 ──▶ P06
 │                              │
 └──────────────▶ P07 ◀─────────┘
                  │
                  ▼
                 P08
```

## Dokumen terkait

- Analisis lengkap: `01-analisis-mendalam.md`
- Kritik & saran: `02-kritik-saran-best-practice.md`
- Keputusan: `03-keputusan-terkunci.md`
- BRD/PRD: `../../../4_Resources/Dokumen-Spesifikasi/`
