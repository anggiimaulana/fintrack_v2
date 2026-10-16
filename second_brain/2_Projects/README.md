# 📁 2_Projects — Ringkasan Eksekutif & Fase Proyek

## Tujuan
Menyimpan ringkasan eksekutif, ruang lingkup, dan kriteria sukses implementasi aktif **FinTrack v2** per fase.

> 📦 **Arsip Dokumen Detail:**
> Seluruh artefak detail (rencana implementasi program, analisis keputusan teknis, rincian task, sprint plan, timeline, aturan binding, dokumen teknis, dan lembar tracking) telah diarsipkan ke:
> [`../5_Archives/phases/`](../5_Archives/phases/)

---

## Overview Fase

| Proyek | Nama | Fase | Status | Dokumen Fase | Arsip Detail |
|--------|------|------|--------|--------------|--------------|
| **P01** | FinTrack v2 — Migrasi Stack | FASE 0–6 | ✅ Selesai | [README](P01_FinTrack_v2/README.md) | [Arsip P01](../5_Archives/phases/P01_FinTrack_v2/) |
| **P02** | Fondasi Platform | — | ⬜ Belum mulai | [README](P02_Fondasi-Platform/README.md) | [Arsip P02](../5_Archives/phases/P02_Fondasi-Platform/) |
| **P03** | AI Gateway Terpusat | — | ⬜ Belum mulai | [README](P03_AI-Gateway-Terpusat/README.md) | [Arsip P03](../5_Archives/phases/P03_AI-Gateway-Terpusat/) |
| **P04** | Subscription & Token Core | — | ⬜ Belum mulai | [README](P04_Subscription-Token-Core/README.md) | [Arsip P04](../5_Archives/phases/P04_Subscription-Token-Core/) |
| **P05** | Payments & Billing | — | ⬜ Belum mulai | [README](P05_Payments-Billing/README.md) | [Arsip P05](../5_Archives/phases/P05_Payments-Billing/) |
| **P06** | Super Admin Dashboard | — | ⬜ Belum mulai | [README](P06_Super-Admin-Dashboard/README.md) | [Arsip P06](../5_Archives/phases/P06_Super-Admin-Dashboard/) |
| **P07** | Rollout RBAC, Realtime & Standar | — | ⬜ Belum mulai | [README](P07_Rollout-RBAC-Realtime/README.md) | [Arsip P07](../5_Archives/phases/P07_Rollout-RBAC-Realtime/) |
| **P08** | Hardening & Go-Live | — | ⬜ Belum mulai | [README](P08_Hardening-GoLive/README.md) | [Arsip P08](../5_Archives/phases/P08_Hardening-GoLive/) |

> **P01** adalah migrasi stack (selesai 100%). **P02–P08** adalah pengembangan platform berlangganan
> & AI gateway (lihat `../4_Resources/Dokumen-Spesifikasi/` dan
> `../5_Archives/ide/2026-10-11_Subscription-AI-Gateway/`).

---

## Urutan Dependensi

```
P02 ──▶ P03 ──▶ P04 ──▶ P05 ──▶ P06
 │                              │
 └──────────────▶ P07 ◀─────────┘
                  │
                  ▼
                 P08
```

---

## Rencana Implementasi

- **Master Plan Program (P02–P08):** [`../5_Archives/phases/IMPLEMENTATION-PLAN.md`](../5_Archives/phases/IMPLEMENTATION-PLAN.md)
- **Detail Rencana per Fase:**
  - [P02 Plan](../5_Archives/phases/P02_Fondasi-Platform/planning/implementation-plan.md)
  - [P03 Plan](../5_Archives/phases/P03_AI-Gateway-Terpusat/planning/implementation-plan.md)
  - [P04 Plan](../5_Archives/phases/P04_Subscription-Token-Core/planning/implementation-plan.md)
  - [P05 Plan](../5_Archives/phases/P05_Payments-Billing/planning/implementation-plan.md)
  - [P06 Plan](../5_Archives/phases/P06_Super-Admin-Dashboard/planning/implementation-plan.md)
  - [P07 Plan](../5_Archives/phases/P07_Rollout-RBAC-Realtime/planning/implementation-plan.md)
  - [P08 Plan](../5_Archives/phases/P08_Hardening-GoLive/planning/implementation-plan.md)

---

## Struktur Folder 2_Projects

Folder ini dipertahankan ramping (*lean*) hanya berisi ringkasan fase (`README.md`):

```
2_Projects/
├── README.md                      # File ini (indeks eksekutif)
├── P01_FinTrack_v2/
│   └── README.md                  # Scope & kriteria migrasi stack
├── P02_Fondasi-Platform/
│   └── README.md                  # Scope & kriteria fondasi platform
├── P03_AI-Gateway-Terpusat/
│   └── README.md                  # Scope & kriteria AI gateway
├── P04_Subscription-Token-Core/
│   └── README.md                  # Scope & kriteria langganan & token
├── P05_Payments-Billing/
│   └── README.md                  # Scope & kriteria payment & billing
├── P06_Super-Admin-Dashboard/
│   └── README.md                  # Scope & kriteria super admin
├── P07_Rollout-RBAC-Realtime/
│   └── README.md                  # Scope & kriteria rollout RBAC & realtime
└── P08_Hardening-GoLive/
    └── README.md                  # Scope & kriteria hardening & go-live
```

Struktur arsip lengkap tersimpan di `../5_Archives/phases/PXX_*/`:
- `analisis/` — Keputusan teknis (`K-PXX-NN`)
- `planning/` — Breakdown task (`TASK-PXX-NNN`), sprint plan, timeline
- `rules/` — Aturan binding arsitektur & eksekusi
- `teknis/` — Spesifikasi teknis mendalam
- `tracking/` — Status pelacakan task

---

## Aturan
1. **Fase sequential** — fase berikut tidak mulai sebelum fase sebelumnya selesai (DoD terpenuhi).
2. **Task ID unik** — Format `TASK-PXX-NNN`, tidak boleh reuse.
3. **Keputusan teknis** — Format `K-PXX-NN`, dicatat di `analisis/` pada arsip fase terkait.
4. **Tracking real-time** — Update `tracking/tracking.md` di arsip fase setiap status task berubah.
5. **Referensi** — Selalu rujuk ke `../4_Resources/` atau arsip di `../5_Archives/phases/`.

---

## Referensi
- HUD & milestones P01: `../5_Archives/phases/P01_FinTrack_v2/roadmap.md`
- Spesifikasi (BRD/PRD/Diagram): `../4_Resources/Dokumen-Spesifikasi/`
- Ide & analisis awal: `../5_Archives/ide/2026-10-11_Subscription-AI-Gateway/`
- Brief awal & 7 keputusan terkunci: `../5_Archives/ide/brief_awal.md`
- Standar area: `../3_Areas/`
