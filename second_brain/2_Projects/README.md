# 📁 2_Projects — Proyek Aktif & Planning

## Tujuan
Menyimpan perencanaan dan tracking implementasi aktif **FinTrack v2**, terorganisir per fase.

## Overview

| Proyek | Nama | Fase | Status | Deliverables Utama |
|--------|------|------|--------|-------------------|
| **P01** | [FinTrack v2 — Migrasi Stack](P01_FinTrack_v2/) | FASE 0–6 | ✅ Selesai | Monorepo Go + PostgreSQL + Next.js, migrasi data penuh, Compose lokal |
| **P02** | [Fondasi Platform](P02_Fondasi-Platform/) | — | ⬜ Belum mulai | Redis, feature flag, audit log, DataTable standar, observability |
| **P03** | [AI Gateway Terpusat](P03_AI-Gateway-Terpusat/) | — | ⬜ Belum mulai | Pool endpoint 9router, routing, circuit breaker, kill switch, metering |
| **P04** | [Subscription & Token Core](P04_Subscription-Token-Core/) | — | ⬜ Belum mulai | Plan, entitlement, subscription, token ledger, enforcement |
| **P05** | [Payments & Billing](P05_Payments-Billing/) | — | ⬜ Belum mulai | Midtrans + manual, switch gateway, invoice, prorata, dunning |
| **P06** | [Super Admin Dashboard](P06_Super-Admin-Dashboard/) | — | ⬜ Belum mulai | Manajemen plan/subscription/pembayaran/token/gateway/audit/settings |
| **P07** | [Rollout RBAC, Realtime & Standar](P07_Rollout-RBAC-Realtime/) | — | ⬜ Belum mulai | Entitlement + SSE + DataTable ke semua halaman & role |
| **P08** | [Hardening & Go-Live](P08_Hardening-GoLive/) | — | ⬜ Belum mulai | Load test, security review, chaos, compliance, runbook |

> **P01** adalah migrasi stack (selesai 100%). **P02–P08** adalah pengembangan platform berlangganan
> & AI gateway (lihat `../4_Resources/Dokumen-Spesifikasi/` dan
> `../0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/`).

## Urutan Dependensi

```
P02 ──▶ P03 ──▶ P04 ──▶ P05 ──▶ P06
 │                              │
 └──────────────▶ P07 ◀─────────┘
                  │
                  ▼
                 P08
```

## Rencana Implementasi

- **Master plan (semua fase):** [`IMPLEMENTATION-PLAN.md`](IMPLEMENTATION-PLAN.md)
- **Plan detail per fase:** `PXX_*/planning/implementation-plan.md`
  (mulai: [`P02_Fondasi-Platform/planning/implementation-plan.md`](P02_Fondasi-Platform/planning/implementation-plan.md))

## Struktur Proyek

```
2_Projects/
├── IMPLEMENTATION-PLAN.md      # Rencana implementasi tingkat program (P02–P08)
├── P01_FinTrack_v2/            # Migrasi stack (selesai)
├── P02_Fondasi-Platform/
├── P03_AI-Gateway-Terpusat/
├── P04_Subscription-Token-Core/
├── P05_Payments-Billing/
├── P06_Super-Admin-Dashboard/
├── P07_Rollout-RBAC-Realtime/
└── P08_Hardening-GoLive/
```

Setiap fase P02–P08 memiliki struktur:

```
PXX_Nama-Proyek/
├── README.md               # Overview, deliverables, success criteria
├── analisis/               # Keputusan teknis (K-PXX-NN)
├── planning/
│   ├── tasks.md            # Task breakdown (TASK-PXX-NNN)
│   ├── sprint-plan.md      # Rencana per sprint
│   └── timeline.md         # Timeline & dependensi
├── rules/                  # Aturan implementasi binding
├── teknis/                 # Dokumen teknis (bervariasi)
└── tracking/               # Status task (⬜/🔄/✅)
```

## Aturan
1. **Fase sequential** — fase berikut tidak mulai sebelum fase sebelumnya selesai (DoD terpenuhi).
2. **Task ID unik** — Format `TASK-PXX-NNN`, tidak boleh reuse.
3. **Keputusan teknis** — Format `K-PXX-NN`, dicatat di `analisis/`.
4. **Tracking real-time** — Update `tracking/tracking.md` setiap status task berubah.
5. **Referensi** — Selalu link ke `../4_Resources/` atau `../../database/README.md` untuk detail, jangan duplikasi.

## Referensi
- HUD & milestones P01: `P01_FinTrack_v2/roadmap.md`
- Spesifikasi (BRD/PRD/Diagram): `../4_Resources/Dokumen-Spesifikasi/`
- Ide & analisis awal: `../0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/`
- Standar area: `../3_Areas/`
