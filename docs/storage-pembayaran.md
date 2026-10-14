# Panduan Penyimpanan Bukti Pembayaran (Local & Cloudflare R2)

Modul `internal/storage/` menyimpan berkas bukti transfer (JPG/PNG/WEBP, maks 5 MB)
di salah satu dari dua driver:

| Driver | Deskripsi | Kapan dipakai |
|---|---|---|
| `local` | Berkas di disk server (`STORAGE_LOCAL_DIR`), disajikan lewat API. | Default pengembangan. |
| `r2` | Objek di bucket Cloudflare R2 (S3-compatible). | Produksi. |

Driver aktif **dapat diganti saat runtime** dari **Super Admin → Pengaturan →
Penyimpanan Bukti Transfer** (`settings.storage_driver`), tanpa restart.
Perubahan berlaku dalam ~5 detik (cache pemilihan driver) untuk unggahan baru.
Berkas yang sudah tersimpan tetap dapat dibaca lewat **fallback lintas-driver**:
jika objek tidak ada di driver aktif, sistem mencoba driver lain.

---

## 1. Alur Teknis

```
Upload  → Provider.Active() → (setting storage_driver) → LocalStore / R2Store
Serve   → Provider.Open(key) → driver aktif, lalu fallback ke driver lain
```

- Kunci objek deterministik per pembayaran: `payment-proofs/<payment_id>.png`.
  Unggah ulang menimpa berkas lama (tidak ada objek yatim).
- `payments.proof_url` menyimpan kunci tersebut (bukan URL absolut), sehingga
  berkas tetap dapat dilayani walau driver berganti.
- Penyajian: `GET /api/v1/payment-proofs/:id` — pemilik pembayaran atau
  `super_admin` saja.

---

## 2. Konfigurasi `local` (default)

```env
STORAGE_DRIVER=local
STORAGE_LOCAL_DIR=./uploads
```

Tidak ada dependensi eksternal. Berkas ditulis ke `./uploads/payment-proofs/`.

---

## 3. Konfigurasi Cloudflare R2

### 3.1 Buat bucket & API token

1. Cloudflare Dashboard → **R2** → **Create bucket** (mis. `fintrack-proofs`).
2. **R2 → Manage R2 API Tokens → Create API token** dengan izin
   *Object Read & Write* untuk bucket tersebut.
3. Catat **Access Key ID**, **Secret Access Key**, dan **Account ID**.
   Endpoint S3 default: `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`.

### 3.2 Set environment backend

```env
STORAGE_DRIVER=local            # default; bisa diganti runtime dari UI
STORAGE_LOCAL_DIR=./uploads

R2_ACCOUNT_ID=<cloudflare-account-id>
R2_ACCESS_KEY_ID=<access-key-id>
R2_SECRET_ACCESS_KEY=<secret-access-key>
R2_BUCKET=fintrack-proofs
R2_REGION=auto
R2_PREFIX=fintrack              # opsional, prefiks kunci di dalam bucket
# R2_ENDPOINT=                  # opsional; menimpa endpoint turunan
# R2_PUBLIC_BASE_URL=           # opsional; domain publik / r2.dev
```

> Jika variabel R2 tidak lengkap, backend tetap berjalan dengan driver `local`
> dan mencatat `R2 storage not configured (using local storage)`.

### 3.3 Aktifkan driver R2

- **Runtime (disarankan):** Super Admin → **Pengaturan** → *Penyimpanan Bukti
  Transfer* → pilih **r2** → Simpan. Status driver (aktif/sehat/bucket/endpoint)
  tampil langsung dari `GET /api/v1/admin/settings/storage`.
- **Saat boot:** set `STORAGE_DRIVER=r2`.

---

## 4. Endpoint Terkait

| Endpoint | Akses | Fungsi |
|---|---|---|
| `POST /api/v1/payments/:id/proof` | user (pemilik) | Unggah bukti (multipart `file`, atau raw body) |
| `GET /api/v1/payment-proofs/:id` | pemilik / super_admin | Sajikan berkas bukti |
| `GET /api/v1/admin/settings/storage` | super_admin | Status & kesehatan driver |
| `PUT /api/v1/admin/settings` | super_admin | Ubah `storage_driver` |

---

## 5. Catatan Operasional

- **Migrasi antar-driver:** mengganti driver hanya mengubah tujuan unggahan
  *baru*. Objek lama tetap terbaca lewat fallback. Untuk konsolidasi, salin
  objek lama ke driver tujuan secara terjadwal (di luar cakupan kode ini).
- **Keamanan:** validasi *magic-byte* (bukan sekadar ekstensi), batas 5 MB,
  kunci disanitasi dari traversal, dan akses dibatasi pemilik/admin.
- **Uji:** `go test ./internal/storage/...` mencakup driver lokal, driver R2
  (via server S3 tiruan in-process), dan perpindahan driver runtime.
