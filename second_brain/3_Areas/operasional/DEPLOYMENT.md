# Deployment — FinTrack v2

## 1. Lingkungan

| Lingkungan | Tujuan | Catatan |
|---|---|---|
| Lokal | Pengembangan | Docker Compose (db 5435, backend 8080, frontend 3001) |
| Staging | Verifikasi | Mirip produksi; Midtrans **sandbox** |
| Produksi | Live | Midtrans produksi; Redis; multi-instance backend |

## 2. Stack

```
db        PostgreSQL 17 + pgvector   (5435 -> 5432)
redis     Redis 7                    (6379)   [BARU di P02]
backend   Go API (Gin)               (8080)
frontend  Next.js                    (3001 -> 3000)
```

## 3. Prasyarat Rilis

- [ ] `go vet && go test && go build` hijau
- [ ] `pnpm build && pnpm lint && pnpm test` hijau
- [ ] Migrasi baru replayable; dump di-regenerate
- [ ] Fitur baru di balik feature flag
- [ ] Rencana rollback disiapkan
- [ ] Backup terbaru tersedia

## 4. Prosedur Deploy (zero downtime)

```
1. Expand      → jalankan migrasi ADITIF (tambah tabel/kolom nullable)
2. Deploy      → rolling restart backend (health gate per instance)
3. Verifikasi  → /health 200, smoke test kritis
4. Aktifkan    → nyalakan feature flag bertahap (rollout %)
5. Pantau      → metrik error/latency/failover
6. Contract    → setelah stabil, migrasi pembersihan terpisah (opsional)
```

> Kode lama tetap kompatibel sepanjang transisi → tidak ada jeda layanan.

## 5. Rollback

- **Feature flag OFF** — cara tercepat menonaktifkan fitur baru.
- **Revert image** — kembali ke image sebelumnya (migrasi aditif tidak merusak kode lama).
- **Jangan** rollback migrasi yang sudah destruktif; desain migrasi agar aditif.

## 6. Konfigurasi Wajib Produksi

| Variabel | Catatan |
|---|---|
| `COOKIE_SECURE=true` | Wajib HTTPS |
| `JWT_SECRET` | ≥ 32 char, rotasi berkala |
| `ENCRYPTION_KEY` | Jangan diganti sembarangan (lihat keamanan) |
| `DATABASE_URL` | Role least-privilege `fintrack_app` |
| `REDIS_URL` | Cache + pub/sub |
| `MIDTRANS_SERVER_KEY` | Hanya server |
| `RATE_LIMIT_*` | Sesuaikan produksi |

## Referensi

- `RUNBOOK_ZERO_DOWNTIME.md`
- `../../2_Projects/P07_Rollout-RBAC-Realtime/teknis/zero-downtime-deploy.md`
