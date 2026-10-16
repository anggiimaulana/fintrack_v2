# 📥 0_Inbox — Quick Capture Zone

Zona tangkap-cepat untuk FinTrack v2. Taruh apapun di sini dulu, rapikan nanti.

## Aturan
1. **Tangkap cepat** — Taruh apapun di sini: ide mentah, catatan meeting, pertanyaan teknis, screenshot error, errata spec.
2. **Format bebas** — Tidak perlu struktur. File `.md`, `.txt`, gambar, log, apapun.
3. **Proses rutin** — Triage isi folder ini secara berkala (akhir sprint / sesi review):
   - Actionable → pindah ke `2_Projects/P01_FinTrack_v2/` sebagai task
   - Referensi terdistilasi → pindah ke `4_Resources/`
   - Standar/SOP jangka panjang → pindah ke `3_Areas/`
   - Selesai / tidak relevan → pindah ke `5_Archives/` atau hapus
4. **Jangan biarkan menumpuk** — Inbox yang membengkak = tanda triage mandek.

## Struktur
```
0_Inbox/
├── README.md      # File ini
├── ide/           # Ide mentah & eksperimen yang belum jadi task
└── *.md           # Catatan lepas, brief, errata
```

## Konvensi Penamaan
```
YYYY-MM-DD_deskripsi-singkat.md
```
Contoh: `2026-10-15_ide-optimasi-cache-per-user.md`

## 🔑 Kredensial Login Demo / Lokal (Quick Reference)

Akun bawaan hasil seed database (`database/seeds/0001_demo.sql`):

| Peran (Role) | Email | Password | Nama Pengguna | Keterangan Akses |
|---|---|---|---|---|
| **Admin** | `admin@fintrack.id` | `Admin@123456` | Admin FinTrack | Akses dashboard manajemen (`/admin`) |
| **User (Demo 1)** | `budi@example.com` | `User@123456` | Budi Santoso | Akun reguler dengan data transaksi aktif |
| **User (Demo 2)** | `siti@example.com` | `User@123456` | Siti Rahayu | Akun pengujian isolasi data |
| **User (Demo 3)** | `andi@example.com` | `User@123456` | Andi Wijaya | Akun user reguler |

> 💡 **Akses Super Admin (`/super-admin`):**
> Bila memerlukan akses Super Admin (manajemen plan, gateway AI, pricing, audit), jalankan query berikut di PostgreSQL:
> ```sql
> UPDATE users SET role = 'super_admin' WHERE email = 'admin@fintrack.id';
> ```

## Isi Saat Ini
- `2026-10-14_kredensial-login-demo.md` — catatan lengkap akun login demo & instruksi pengujian.
- `ide/` — folder penampung ide mentah (saat ini kosong; file lama diarsipkan).

> 📦 **Arsip Dokumen Mentah & Ide:**
> Seluruh analisis awal, ide, audit komprehensif, dan brief awal proyek telah diarsipkan ke:
> [`../5_Archives/ide/`](../5_Archives/ide/)
> - Brief awal & keputusan migrasi: [`../5_Archives/ide/brief_awal.md`](../5_Archives/ide/brief_awal.md)
> - Ide & analisis platform: [`../5_Archives/ide/2026-10-11_Subscription-AI-Gateway/`](../5_Archives/ide/2026-10-11_Subscription-AI-Gateway/)
> - Dokumen audit keamanan: [`../5_Archives/ide/2026-10-12_Audit-Komprehensif-Keamanan-Logic-Regulasi/`](../5_Archives/ide/2026-10-12_Audit-Komprehensif-Keamanan-Logic-Regulasi/)

## Status
Folder ini sekarang **bersih** setelah sesi triage. File baru yang masuk dapat ditampung di sini sebelum diproses.
