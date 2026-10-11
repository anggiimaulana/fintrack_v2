# Runbook — Backup & Restore

## 1. Prinsip

- **Data billing & finansial = prioritas tertinggi.** Backup lebih ketat dari data lain.
- **RPO** (Recovery Point Objective) target: ≤ 1 jam untuk data billing; ≤ 24 jam untuk data lain.
- **RTO** (Recovery Time Objective) target: ≤ 2 jam.
- Backup **diuji** (restore drill), bukan hanya dibuat.

## 2. Ruang Lingkup Backup

| Data | Frekuensi | Retensi | Catatan |
|---|---|---|---|
| PostgreSQL (full) | Harian | 30 hari | Wajib |
| PostgreSQL (WAL/inkremental) | Per jam | 7 hari | Untuk RPO ≤ 1 jam |
| `payments`, `invoices`, `token_ledger`, `subscriptions`, `audit_logs` | Harian + snapshot sebelum rilis | 1 tahun | Data uang & audit |
| Dump schema final | Setiap perubahan schema | Selamanya (git) | `database/dump/` |
| Kunci AI terenkripsi | Ikut DB | — | Tidak pernah plaintext |

## 3. Prosedur Backup

```bash
# Dump penuh
database/scripts/dump.sh
# Backup terverifikasi (custom format)
pg_dump -Fc -f backup_$(date +%F).dump "$DATABASE_URL"
# Simpan ke penyimpanan terpisah (bukan server yang sama)
```

- Simpan di lokasi **terpisah** dari DB (mencegah kehilangan bersamaan).
- Enkripsi backup at-rest.
- Catat metadata (waktu, ukuran, checksum).

## 4. Prosedur Restore

```bash
# 1) Siapkan DB target (scratch) — JANGAN langsung ke produksi
createdb fintrack_restore
# 2) Restore
pg_restore -d fintrack_restore backup_YYYY-MM-DD.dump
# 3) Verifikasi
#    - count & sum(amount) per user identik dengan sumber
#    - jumlah payments/invoices/token_ledger sesuai
# 4) Tukar (bila perlu) — dengan prosedur terkontrol & downtime minimal
```

## 5. Drill Berkala

- **Bulanan**: restore ke DB scratch → verifikasi count & `sum(amount)` → catat hasil.
- **Sebelum rilis besar**: snapshot tambahan.
- Hasil drill dicatat di `../../1_Digest/Tugas-Rutin/log_aktivitas.md`.

## 6. Integritas Data Uang

- Verifikasi: `SELECT sum(amount_idr) FROM payments` & `invoices` cocok antar-backup.
- `token_ledger` append-only: jumlah entri tidak boleh berkurang.
- Audit log tidak boleh hilang (non-repudiation).

## Referensi

- `DEPLOYMENT.md`
- `../../2_Projects/P08_Hardening-GoLive/teknis/backup-dr.md`
- `../../../database/README.md`
