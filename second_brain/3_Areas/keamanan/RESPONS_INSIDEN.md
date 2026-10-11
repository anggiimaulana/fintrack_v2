# Prosedur Respons Insiden Keamanan — FinTrack v2

## 1. Tingkat Keparahan

| Level | Contoh | Respons |
|---|---|---|
| **P1 Kritis** | Kebocoran kunci AI, akses data billing tanpa otorisasi, webhook dipalsukan | Segera (menit), eskalasi ke pemilik produk |
| **P2 Tinggi** | Percobaan brute force berhasil, replay webhook, anomali pembayaran | < 1 jam |
| **P3 Sedang** | Lonjakan error gateway, endpoint bocor kuota | < 1 hari |
| **P4 Rendah** | Percobaan akses gagal, log mencurigakan | Rencanakan |

## 2. Langkah Umum

1. **Deteksi & konfirmasi** — dari alert metrik/audit/log.
2. **Batasi dampak** — kill switch endpoint, nonaktifkan akun, cabut token.
3. **Investigasi** — audit log, `last_used_at` kunci, jejak webhook.
4. **Pulihkan** — rotasi kredensial, aktifkan endpoint sehat.
5. **Catat** — di `../../1_Digest/Tugas-Rutin/log_aktivitas.md` + audit.
6. **Pembelajaran** — perbaiki kontrol yang gagal.

## 3. Runbook Cepat

### 3.1 Kunci AI bocor
1. Nonaktifkan endpoint/kunci (`status=inactive` atau kill switch).
2. Tambah kunci baru + health check.
3. Aktifkan kembali trafik; pantau failover.
4. Audit akses; rotasi; catat insiden.

### 3.2 Webhook dipalsukan
1. Verifikasi signature gagal → sudah ditolak; periksa apakah ada yang lolos.
2. Audit semua `payments` terbaru; batalkan aktivasi yang tidak sah.
3. Perkuat rate limit & logging.

### 3.3 Akses tidak sah ke data billing
1. Cabut sesi (rotasi `JWT_SECRET` / revoke refresh token).
2. Periksa cache entitlement basi.
3. Audit & laporkan bila melibatkan data pribadi (UU PDP).

### 3.4 Gateway AI down massal
1. Kill switch + failover ke endpoint cadangan.
2. Beri tahu user (pesan ramah, tanpa detail internal).
3. Pulihkan & catat metrik.

## 4. Komunikasi

- Internal: pemilik produk + tim teknis.
- Eksternal: hanya bila ada dampak data pribadi (kewajiban notifikasi UU PDP).

## 5. Pencegahan

- Review berkala kontrol keamanan; uji chaos failover; rotasi terjadwal;
  latihan respons insiden.

## Referensi

- `KEAMANAN_DAN_PRIVASI.md`, `SECRET_MANAGEMENT.md`
- `../../2_Projects/P08_Hardening-GoLive/teknis/runbook-incidents.md`
