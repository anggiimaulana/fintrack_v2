# Runbook — Zero Downtime Deploy & Migrasi

## 1. Prinsip

Zero downtime = **tidak ada request gagal** selama deploy. Dicapai lewat:
**Parallel Change (expand → migrate → contract) + rolling deploy + feature flag + health gate.**

## 2. Aturan Migrasi (WAJIB)

| Perubahan | Cara aman |
|---|---|
| Tambah tabel | `CREATE TABLE IF NOT EXISTS` — aman |
| Tambah kolom | **nullable** dulu (atau DEFAULT); jangan `NOT NULL` tanpa default |
| Rename kolom | Tambah kolom baru → backfill → pindahkan kode → hapus lama (migrasi terpisah) |
| Ubah tipe | Tambah kolom baru bertipe baru → backfill → tukar → hapus |
| Hapus kolom | Hanya setelah kode lama benar-benar mati (migrasi contract) |
| Tambah index | `CREATE INDEX CONCURRENTLY` (tanpa lock panjang) |

> **Dilarang** dalam satu langkah rilis: kolom `NOT NULL` tanpa default pada tabel besar,
> `ALTER TYPE` langsung, atau `DROP` yang dipakai kode berjalan.

## 3. Langkah Eksekusi

```bash
# 1) Expand (aditif)
database/scripts/migrate.sh            # migrasi bernomor baru

# 2) Deploy rolling
docker compose up -d --no-deps --build backend
docker compose up -d --no-deps --build frontend

# 3) Health gate
curl -fsS localhost:8081/health        # harus 200

# 4) Aktifkan fitur bertahap
# set feature_flags.rollout naik perlahan

# 5) Pantau metrik; jika anomali → flag OFF (rollback instan)
```

## 4. Checklist Pra-Deploy

- [ ] Migrasi bersifat aditif & replayable
- [ ] Fitur baru di balik flag (default OFF)
- [ ] Backup terbaru tersedia
- [ ] Health endpoint berfungsi
- [ ] Rollback plan jelas
- [ ] Alert aktif (error rate, latency)

## 5. Checklist Pasca-Deploy

- [ ] `/health` 200
- [ ] Smoke test: login, catat transaksi, chat AI
- [ ] Error rate tidak naik
- [ ] Cache terisi normal (Redis)
- [ ] Tidak ada lonjakan failover gateway

## 6. Rollback

1. **Flag OFF** (paling cepat, tanpa deploy).
2. Revert image ke versi sebelumnya.
3. Jika migrasi aditif → tidak perlu mundur; biarkan (aman).
4. Catat insiden & penyebab di log aktivitas.

## Referensi

- Martin Fowler — *ParallelChange*, *StranglerFigApplication*
- `DEPLOYMENT.md`, `RUNBOOK_INCIDENTS.md`
