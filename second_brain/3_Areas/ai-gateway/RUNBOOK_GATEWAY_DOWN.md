# Runbook — Gateway / Endpoint AI Down

## 1. Deteksi

- Alert: error rate gateway > 5% (5 menit), semua endpoint `down`, atau lonjakan failover.
- Dashboard super admin menampilkan `health` tiap endpoint.

## 2. Tingkat Dampak

| Kondisi | Dampak | Respons |
|---|---|---|
| 1 endpoint down | Tidak terasa (failover otomatis) | Pantau |
| ≥ 2 endpoint down | Risiko gagal | Aktifkan cadangan/kill switch |
| Semua endpoint down | Fitur AI mati | Eskalasi + pesan ramah ke user |

## 3. Langkah Penanganan

```
1. Cek dashboard → endpoint mana yang down & mengapa (last_error)
2. Pastikan circuit breaker bekerja (endpoint bermasalah di-skip)
3. Jika 9router mati:
     - aktifkan route cadangan ke provider langsung (bila ada)
     - atau kill switch untuk endpoint bermasalah
4. Tambah endpoint baru bila perlu (super admin)
5. Health check → pastikan endpoint sehat sebelum trafik penuh
6. Beri pesan ramah ke user (tanpa detail internal)
7. Catat insiden + metrik failover
```

## 4. Komunikasi ke User

- Pesan generik: "Layanan AI sedang tidak tersedia sementara. Silakan coba lagi."
- **Jangan** membocorkan detail internal (URL, kunci, provider).

## 5. Pencegahan

- Selalu ada ≥ 2 endpoint sehat pada pool.
- Health check berkala & alerting.
- Uji chaos failover berkala (lihat P08).

## 6. Pasca-Insiden

- Post-mortem: penyebab, dampak, perbaikan.
- Perbarui runbook & tambah test bila relevan.

## Referensi

- `ARSITEKTUR_GATEWAY.md`
- `../operasional/RUNBOOK_INCIDENTS.md`
- `../../2_Projects/P08_Hardening-GoLive/teknis/chaos-failover.md`
