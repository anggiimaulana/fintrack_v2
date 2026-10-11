# Standar Manajemen Secret & Rotasi — FinTrack v2

## 1. Inventaris Kredensial

| Kredensial | Lokasi | Sensitivitas | Rotasi disarankan |
|---|---|---|---|
| `POSTGRES_PASSWORD` | `.env` | Tinggi | 6–12 bulan |
| `APP_DB_PASSWORD` | `.env` | Tinggi | 6–12 bulan |
| `JWT_SECRET` | `.env` (backend+frontend) | Kritis | 3–6 bulan (koordinasikan) |
| `ENCRYPTION_KEY` | `.env` | **Kritis** | **Hati-hati**: mengganti = data lama tak terbaca; butuh re-enkripsi |
| Kunci AI gateway (9router) | DB (`ai_endpoints.api_key_encrypted`) | Kritis | 3 bulan / saat dicurigai bocor |
| `MIDTRANS_SERVER_KEY` | `.env` / `settings` | Kritis | Saat dicurigai bocor |
| `MIDTRANS_CLIENT_KEY` | Frontend env | Sedang | Saat dicurigai bocor |

## 2. Aturan

1. **Tidak ada secret di git.** Hanya `.env.example` (nilai dummy).
2. Secret produksi disimpan di manajer secret (mis. Vault / secret manager), bukan file di server.
3. Akses secret = least privilege; setiap akses kunci AI tercatat audit.
4. Kunci AI dienkripsi dengan `ENCRYPTION_KEY` sebelum disimpan; hanya preview yang dikembalikan API.

## 3. Prosedur Rotasi Kunci AI (tanpa downtime)

1. Tambah endpoint/kunci baru di `ai_endpoints` (status `active`, prioritas lebih tinggi).
2. Health check → pastikan sehat.
3. Turunkan prioritas kunci lama / nonaktifkan (`status=inactive`).
4. Pantau metrik (error rate, failover count) → pastikan stabil.
5. Arsipkan kunci lama (jangan hapus) untuk audit.

> Karena gateway punya **pool + failover**, rotasi dapat dilakukan tanpa downtime: trafik otomatis
> pindah begitu kunci lama dinonaktifkan.

## 4. Prosedur Rotasi `JWT_SECRET`

- Mengganti `JWT_SECRET` **membatalkan semua access token** (user harus login ulang) kecuali
  menggunakan strategi kunci ganda (verifikasi dengan lama + baru selama masa transisi).
- Disarankan: rotasi saat jendela pemeliharaan atau implementasikan dual-key verification.

## 5. Prosedur Rotasi `ENCRYPTION_KEY`

- **Jangan** mengganti langsung: data lama akan tak terbaca.
- Prosedur aman: dekripsi dengan kunci lama → enkripsi dengan kunci baru → simpan versi kunci
  (key versioning). Wajib diuji di lingkungan non-produksi lebih dulu.

## 6. Deteksi & Respons Kebocoran

- Audit akses kunci AI & `last_used_at` anomali.
- Alert bila endpoint tiba-tiba banyak error/429 (indikasi kunci bocor/dipakai pihak lain).
- Saat bocor: nonaktifkan kunci → rotasi → audit → catat insiden (`RESPONS_INSIDEN.md`).

## Referensi

- `../ai-gateway/` (arsitektur pool & rotasi)
- BRD §11 (kepatuhan)
