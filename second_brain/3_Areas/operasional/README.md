# ⚙️ 3_Areas — Operasional

Area tanggung jawab berkelanjutan untuk **keandalan & operasional** FinTrack v2.

## Scope

- Deployment & rilis (rolling, zero downtime, feature flag).
- Backup & pemulihan (disaster recovery) — termasuk data billing.
- Monitoring & observability (four golden signals + metrik bisnis).
- Runbook insiden operasional (gateway down, pembayaran gagal, cache basi).
- Kapasitas & performa (load test).

## Dokumen

| Dokumen | Isi |
|---|---|
| `DEPLOYMENT.md` | Prosedur deploy & rilis |
| `RUNBOOK_ZERO_DOWNTIME.md` | Deploy & migrasi tanpa downtime |
| `RUNBOOK_BACKUP_RESTORE.md` | Backup & restore |
| `MONITORING_OBSERVABILITY.md` | Metrik, alert, dashboard |
| `RUNBOOK_INCIDENTS.md` | Penanganan insiden operasional |
| `RUNBOOK_MODE_LOKAL.md` | Mode run lokal (hybrid vs Docker) + pencegahan bentrok port |

## Prinsip

1. **Zero downtime** — migrasi expand/contract + rolling deploy + health gate.
2. **Data billing paling dijaga** — backup lebih ketat, RPO/RTO tertulis.
3. **Observability dulu** — tanpa metrik, insiden tak terdeteksi.
4. **Rollback siap** — setiap rilis punya rencana mundur.

## Referensi

- Compose: `../../../docker-compose.yml`
- Runbook detail: `../../2_Projects/P08_Hardening-GoLive/teknis/`
- Area keamanan: `../keamanan/`
