# Kebijakan Keamanan & Privasi — FinTrack v2

> Dokumen hidup. Perubahan standar harus di-review, bukan diubah unilateral.

## 1. Prinsip Dasar

| Prinsip | Penerapan |
|---|---|
| Defense in depth | Lapisan: TLS, auth, otorisasi, validasi input, rate limit, audit |
| Least privilege | Role DB `fintrack_app` tanpa DDL; akses minimum |
| Fail secure | Kegagalan menutup akses, bukan membukanya |
| Server-authoritative | Semua keputusan hak akses di backend |
| Non-repudiation | Audit append-only untuk aksi sensitif |
| Data minimization | Simpan hanya yang perlu; jangan simpan data kartu |

## 2. Autentikasi

- Password: **bcrypt cost 12** (`backend/internal/infrastructure/auth/password.go`).
- Access token: **JWT HS256**, umur **15 menit**, `JWT_SECRET` ≥ 32 karakter.
- Refresh token: rotasi di DB, hanya **sha256 hex** yang disimpan; deteksi reuse via `replaced_by`.
- Cookie: `httpOnly`, `SameSite`, `Secure` saat produksi (`COOKIE_SECURE=true`).
- Semua trafik browser lewat proxy same-origin Next.js; `OriginCheck` untuk CSRF dasar.

## 3. Otorisasi

- Role: `super_admin` (operator platform) & `user` (end-user).
- **Entitlement dipisah dari role** — hak akses fitur/limit diturunkan dari plan.
- Penegakan di backend (middleware guard); frontend hanya cermin UX.
- Entitlement di-resolve per-request dari cache berversi (tidak di JWT) → perubahan langsung berlaku.

## 4. Manajemen Secret

- **Jangan pernah** commit `.env`, secret, API key, atau dump berisi secret.
- Kunci AI disimpan **terenkripsi** (`ENCRYPTION_KEY`); API hanya menampilkan preview.
- Rotasi kredensial terjadwal (lihat `SECRET_MANAGEMENT.md`).

## 5. Keamanan Pembayaran

- **Jangan pernah** menyimpan/memproses data kartu → gunakan Midtrans Snap/hosted (PCI-DSS SAQ-A).
- Webhook wajib: verifikasi **signature** + verifikasi ulang via **Status API** + **idempoten**.
- Jangan percaya status dari klien; hanya webhook server-to-server yang sah.

## 6. Kepatuhan UU PDP No. 27/2022

- **Data pribadi** = nama, email, riwayat pembayaran, data transaksi keuangan.
- Wajib: dasar hukum pemrosesan, kebijakan retensi, hak akses/perbaikan/penghapusan subjek data.
- **Tegangan** retensi pajak (invoice, umumnya 10 tahun) vs hak hapus → selesaikan dengan
  **anonimisasi** (bukan hapus) bila konflik; catat kebijakan tertulis.

## 7. Logging & Audit

- `audit_logs` append-only: aktor, aksi, entitas, nilai sebelum→sesudah, IP, UA.
- **Jangan** mencatat secret, token, atau data kartu ke log.
- Log akses ke kunci AI & aksi billing wajib.

## 8. Insiden

Lihat `RESPONS_INSIDEN.md`. Setiap insiden keamanan dicatat di `../../1_Digest/Tugas-Rutin/log_aktivitas.md`.

## Referensi

- OWASP Top 10; NIST least privilege
- UU No. 27/2022 tentang Pelindungan Data Pribadi
- PCI-DSS SAQ-A (Midtrans Snap)
