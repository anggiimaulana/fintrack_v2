# Runbook — Rotasi Kunci AI (Tanpa Downtime)

## 1. Prinsip

Rotasi kunci **tanpa downtime** dimungkinkan karena gateway memiliki **pool + failover**.
Kunci lama tidak dihapus mendadak; trafik dipindahkan bertahap.

## 2. Prosedur Rotasi Terjadwal

```
1. Tambah endpoint/kunci BARU di ai_endpoints
     - status=active, health diuji, priority lebih tinggi dari yang lama
2. Health check → pastikan sehat
3. Arahkan trafik: naikkan prioritas kunci baru (atau turunkan yang lama)
4. Pantau metrik: error rate, latensi, failover count
5. Setelah stabil → nonaktifkan kunci lama (status=inactive)
6. Arsipkan kunci lama (JANGAN hapus) untuk audit
7. Catat di log aktivitas
```

## 3. Prosedur Rotasi Darurat (kunci bocor)

```
1. NONAKTIFKAN segera kunci/endpoint yang bocor (status=inactive / kill switch)
2. Aktifkan endpoint cadangan (pool) → trafik otomatis pindah
3. Tambah kunci pengganti + health check
4. Audit akses kunci (last_used_at, pola anomali)
5. Catat insiden (lihat ../keamanan/RESPONS_INSIDEN.md)
```

## 4. Checklist

- [ ] Selalu ada ≥ 1 endpoint sehat saat rotasi
- [ ] Kunci lama diarsipkan, bukan dihapus
- [ ] Metrik dipantau selama rotasi
- [ ] Audit dicatat
- [ ] Tidak ada downtime untuk user

## 5. Kesalahan yang Dihindari

- Menghapus kunci lama sebelum kunci baru terbukti sehat.
- Mengganti `ENCRYPTION_KEY` tanpa prosedur (data lama tak terbaca).
- Rotasi tanpa health check → trafik ke endpoint mati.

## Referensi

- `ARSITEKTUR_GATEWAY.md`
- `../keamanan/SECRET_MANAGEMENT.md`
