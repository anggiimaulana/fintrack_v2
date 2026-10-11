# Runbook — Insiden Operasional

## 1. Insiden: AI Gateway Semua Endpoint Down

**Gejala:** fitur AI mengembalikan 503; `ai_endpoints.health` down massal.
**Dampak:** chat, vision, voice, embedding tidak berfungsi.

1. Cek dashboard gateway → endpoint mana yang down.
2. Aktifkan **failover** ke endpoint cadangan (jika belum otomatis).
3. Jika 9router mati: aktifkan **kill switch** & arahkan ke provider langsung (bila route cadangan ada).
4. Beri pesan ramah ke user (tanpa detail internal).
5. Catat insiden + metrik failover.

## 2. Insiden: Pembayaran Gagal / Webhook Tidak Masuk

**Gejala:** user sudah bayar tapi langganan belum aktif.
1. Cek `payments` (status) & log webhook.
2. Verifikasi manual via Midtrans Status API.
3. Jika valid → tandai paid (idempoten) → aktivasi.
4. Jika webhook bermasalah → periksa URL & signature; uji replay.
5. Catat & beri tahu user.

## 3. Insiden: Cache Basi (Entitlement/Data)

**Gejala:** user melihat fitur yang seharusnya tidak boleh, atau data lama.
1. Bump `entitlement_version` → cache gugur.
2. Verifikasi Redis pub/sub berjalan (invalidasi lintas instance).
3. Periksa TTL cache — pastikan tidak terlalu panjang untuk billing/entitlement.
4. Catat; tambahkan test bila perlu.

## 4. Insiden: Over-spend Token / Kuota Bocor

**Gejala:** pemakaian melebihi kuota.
1. Periksa ledger (`reserve/commit/release`) — apakah ada reserve tanpa commit/release?
2. Cek idempotency key — apakah ada duplikasi?
3. Perbaiki transaksi; sesuaikan bila perlu (dengan audit).
4. Tambah test concurrency.

## 5. Insiden: Kunci AI Bocor

Lihat `../keamanan/RESPONS_INSIDEN.md` §3.1.

## 6. Insiden: DB Lambat / Pool Saturasi

1. Cek slow query & index.
2. Naikkan pool / tambah replika baca.
3. Cek cache hit (Redis) — perbaiki bila rendah.
4. Catat.

## 7. Eskalasi

| Level | Kontak | SLA |
|---|---|---|
| P1 | Pemilik produk + teknis | 15 menit |
| P2 | Teknis | 1 jam |
| P3 | Teknis (jadwal) | 1 hari |

## Referensi

- `DEPLOYMENT.md`, `RUNBOOK_ZERO_DOWNTIME.md`, `MONITORING_OBSERVABILITY.md`
- `../../2_Projects/P08_Hardening-GoLive/teknis/runbook-incidents.md`
