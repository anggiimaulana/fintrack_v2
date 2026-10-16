# Kredensial Login Akun Demo & Lokal — FinTrack v2

Dokumen ini memuat daftar akun demo hasil seeder database (`database/seeds/0001_demo.sql`) yang dapat digunakan untuk login pengujian lokal maupun development.

---

## 1. Daftar Akun & Password

| Peran (Role) | Email | Password | Nama Lengkap | Route Frontend | Keterangan |
|---|---|---|---|---|---|
| **Super Admin** | `superadmin@fintrack.id` | `Admin@123456` | Super Admin FinTrack | `/super-admin` | Akses penuh panel super admin (katalog paket, pembayaran, AI Gateway) |
| **Admin** | `admin@fintrack.id` | `Admin@123456` | Admin FinTrack | `/admin` | Akses panel admin platform |
| **User (Demo 1)** | `budi@example.com` | `User@123456` | Budi Santoso | `/user` | Akun aktif dengan data transaksi & kategori lengkap |
| **User (Demo 2)** | `siti@example.com` | `User@123456` | Siti Rahayu | `/user` | Akun kedua untuk uji isolasi data & otorisasi |
| **User (Demo 3)** | `andi@example.com` | `User@123456` | Andi Wijaya | `/user` | Akun ketiga untuk uji konkurensi / multi-user |

---

## 2. Akses Super Admin (`/super-admin`)

Role `super_admin` digunakan untuk modul manajemen katalog paket, tinjauan manual pembayaran, harga & kuota token AI, serta operasional AI Gateway.

Bila ingin menggunakan akun `admin@fintrack.id` sebagai `super_admin` di lingkungan lokal, jalankan query berikut:

```sql
UPDATE users SET role = 'super_admin' WHERE email = 'admin@fintrack.id';
```

---

## 3. Contoh Login via cURL / API

Endpoint autentikasi backend menggunakan cookie `HttpOnly` (`ft_at` untuk Access Token dan `ft_rt` untuk Refresh Token):

```bash
# Login sebagai User (Budi)
curl -i -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"budi@example.com","password":"User@123456"}'

# Login sebagai Admin
curl -i -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@fintrack.id","password":"Admin@123456"}'
```

---

## 4. Referensi Sumber Kode

- File Seed Data: `database/seeds/0001_demo.sql`
- Dokumentasi Database: `database/README.md`
- Skenario Validasi Auth: `second_brain/2_Projects/P01_FinTrack_v2/skenario_pengujian.md`
