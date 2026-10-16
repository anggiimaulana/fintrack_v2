# 🧠 Second Brain — FinTrack v2 (Live Operations Dashboard)

```text
┌──────────────────────────────────────────────────────────────┐
│  FINTRACK v2 — MIGRATION HUD                                 │
│  [░░░░░░░░░░░░░░░░░░░░]  0% COMPLETED                        │
│  Active Milestone : FASE 0 — SCAFFOLD MONOREPO & SECOND BRAIN│
│  DB ⏳ | Backend ⏳ | Frontend ⏳ | Compose ⏳ | Migrate ⏳      │
└──────────────────────────────────────────────────────────────┘
```

## Peta Zona Workspace
| Zona | Path | Isi |
|------|------|-----|
| Kode produksi | `frontend/`, `backend/`, `database/` | Hanya kode & artefak produksi bersih |
| Sandbox testing | `tester/` | Script scratch, dump, hasil k6 — **tidak pernah di-commit** |
| Knowledge base | `second_brain/` | Dokumentasi, roadmap, log, keputusan — **di-commit** |

## Struktur PARA
| Folder | Fungsi |
|--------|--------|
| `0_Inbox/` | Quick capture zone (kredensial demo, catatan baru) |
| `1_Digest/Tugas-Rutin/` | Audit trail (`log_aktivitas.md`) |
| `2_Projects/` | Ringkasan fase proyek aktif (`P01`–`P08`) |
| `3_Areas/` | Standar & SOP jangka panjang |
| `4_Resources/` | Aset terdistilasi, kredensial, dokumentasi API |
| `5_Archives/` | Riwayat proyek selesai, detail fase, ide & audit (`phases/`, `ide/`, `sprint-logs/`) |

## Proyek Aktif
- **P01_FinTrack_v2** — Migrasi stack Go + PostgreSQL + Next.js (monorepo baru).
  - Ringkasan: `2_Projects/P01_FinTrack_v2/README.md`
  - Arsip Detail (Roadmap & Skenario): `5_Archives/phases/P01_FinTrack_v2/`
  - Brief awal: `5_Archives/ide/brief_awal.md`

## Aturan Emas
1. Kode hanya di `frontend/`, `backend/`, `database/`.
2. File scratch/testing AI → `tester/` (tidak pernah commit).
3. Dokumentasi & knowledge → `second_brain/`.
4. Jangan pernah commit `.env`, secret, atau dump berisi secret baru.
5. Perintah kanonik: `cd backend && go test ./...`, `cd frontend && pnpm build`.
