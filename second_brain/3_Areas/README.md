# 🔄 3_Areas — Area Tanggung Jawab Berkelanjutan

## Tujuan
Menyimpan standar, SOP, dan panduan operasional yang berlaku sepanjang proyek **FinTrack v2**.
Berbeda dengan `2_Projects/` (ada awal & akhir), area ini bersifat **ongoing** — terus dipelihara dan diperbarui.

## Struktur Area

| Folder | Domain | Relevansi |
|--------|--------|-----------|
| `kualitas-kode/` | Standar kode, testing, review, batas modul | `STANDAR_KODE.md`, `TESTING_STRATEGY.md`, `REVIEW_GUIDELINE.md`, `MODULE_BOUNDARIES.md` |
| `keamanan/` | Kebijakan keamanan & privasi, secret, auth/otorisasi, keamanan pembayaran, respons insiden | `KEAMANAN_DAN_PRIVASI.md`, `SECRET_MANAGEMENT.md`, `AUTENTIKASI_OTORISASI.md`, `KEAMANAN_PEMBAYARAN.md`, `RESPONS_INSIDEN.md` |
| `dokumentasi/` | Standar dokumentasi (PARA), decision log (ADR) | `STANDAR_DOKUMENTASI.md`, `DECISION_LOG.md` |
| `operasional/` | Deployment, zero-downtime, backup/restore, monitoring, insiden | `DEPLOYMENT.md`, `RUNBOOK_ZERO_DOWNTIME.md`, `RUNBOOK_BACKUP_RESTORE.md`, `MONITORING_OBSERVABILITY.md`, `RUNBOOK_INCIDENTS.md` |
| `kolaborasi/` | Workflow, git, code review | `WORKFLOW.md`, `CODE_REVIEW.md` |
| `billing-pembayaran/` 🆕 | Kebijakan billing, dunning, pembayaran manual, refund | `KEBIJAKAN_BILLING.md`, `RUNBOOK_DUNNING.md`, `RUNBOOK_PEMBAYARAN_MANUAL.md`, `RUNBOOK_REFUND.md` |
| `ai-gateway/` 🆕 | Arsitektur gateway AI, runbook down, rotasi kunci, metering | `ARSITEKTUR_GATEWAY.md`, `RUNBOOK_GATEWAY_DOWN.md`, `RUNBOOK_ROTASI_KUNCI.md`, `KEBIJAKAN_METERING.md` |

## Aturan
1. **Dokumen di sini adalah panduan hidup** — update seiring proyek berkembang.
2. **Jangan duplikasi sumber asli** — referensikan `4_Resources/`, `database/README.md`, atau rules file alih-alih menyalin.
3. **Setiap area punya README.md** — menjelaskan scope dan daftar dokumen.
4. **Perubahan standar harus di-review** — jangan ubah unilateral.
5. **Area billing & AI gateway bersifat mengikat** (menyangkut uang nyata & keandalan AI).

## Referensi
- Aturan workspace: `../../CLAUDE.md`, `../../.agents/rules/workspace_rules.md`
- Spesifikasi (BRD/PRD): `../4_Resources/Dokumen-Spesifikasi/`
- Dokumentasi DB: `../../database/README.md`
- Rencana fase: `../2_Projects/`
