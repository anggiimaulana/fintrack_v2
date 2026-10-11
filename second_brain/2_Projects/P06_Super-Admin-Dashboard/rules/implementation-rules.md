# Implementation Rules — Fase P06 (Super Admin Dashboard)

Dokumen ini mendefinisikan aturan implementasi mengikat (*binding implementation rules*) bagi para perekayasa perangkat lunak yang mengerjakan modul Super Admin Dashboard. Pelanggaran terhadap aturan ini akan menggagalkan proses tinjauan kode (*code review*) dan integrasi otomatis.

---

## 1. Aturan Keamanan & Otorisasi (Security & Access Control)

1. **Backend Sebagai Otoritas Tunggal (K-13)**:
   - Frontend TIDAK PERNAH dipercaya untuk menentukan hak akses. Logika seperti `if (user.role === 'super_admin')` di browser hanya berguna untuk kenyamanan antarmuka (UX), bukan pengaman data.
   - Setiap endpoint di bawah `/api/v1/admin/*` WAJIB mengikat middleware otorisasi `RequireSuperAdmin()`.
2. **Realtime RBAC Lookup & Verifikasi Versi (K-14)**:
   - Role pengguna TIDAK BOLEH dibaca secara statis dari klaim JWT lawas. Middleware wajib melakukan validasi cepat terhadap versi hak akses atau mengecek status aktif pengguna di Redis cache ber-TTL pendek.
   - Jika role pengguna dicabut, akses administratifnya wajib terputus seketika pada request berikutnya tanpa menunggu token kedaluwarsa.
3. **Proteksi Layout SSR Frontend**:
   - Layout `app/(dashboard)/super-admin/layout.tsx` adalah Server Component. Pengecekan sesi wajib terjadi di sisi server sebelum HTML dikirim ke browser. Pengguna tanpa peran `super_admin` dialihkan seketika via `redirect('/dashboard')`.

---

## 2. Aturan Standarisasi UI & Komponen `<DataTable>` (K-22)

1. **Kepatuhan Terhadap TanStack Table v8**:
   - Seluruh 8 modul tabular (Plans, Subscriptions, Payments, Reviews, Tokens, Endpoints, Routes, Audit Logs) WAJIB menggunakan komponen bersama `<DataTable>`. Dilarang membuat implementasi tabel ad-hoc atau tabel HTML mentah.
2. **Sinkronisasi Dua Arah URL State**:
   - Setiap interaksi pagination, sorting, search filter, dan tab view WAJIB tercermin pada URL Search Params (`?page=&sort=&order=&q=&status=`).
   - Ukuran halaman (*page size*) dikunci tetap pada **20 baris per halaman**.
   - Penomoran baris wajib konsisten dan absolut: `nomor = (page - 1) * 20 + baris_ke + 1`.
3. **Debounce Pencarian**:
   - Input pencarian teks wajib menggunakan teknik *debounce* minimal 300ms untuk mencegah banjir request HTTP ke backend saat pengguna mengetik.
4. **Respon Kosong & Loading State**:
   - Setiap tabel wajib memiliki *skeleton loading* yang mempertahankan dimensi layout, serta komponen *Empty State* informatif bila hasil pencarian/filter menghasilkan 0 baris data.

---

## 3. Aturan Integritas Data Finansial, Token, dan Soft-Limit

1. **Format Angka Moneter (K-27)**:
   - Semua nilai uang (harga paket, pembayaran, tagihan, refund) WAJIB bertipe data **Integer Rupiah**.
   - DILARANG menggunakan tipe data floating-point (`float32`, `float64`, atau tipe desimal JavaScript) untuk kalkulasi saldo uang. Pembagian prorata wajib dibulatkan ke bawah (*floor integer*).
2. **Buku Besar Token Append-Only (K-28)**:
   - DILARANG mengeksekusi perintah SQL `UPDATE` atau `DELETE` pada tabel `token_ledger`.
   - Penyesuaian saldo token manual WAJIB berupa baris mutasi kredit (`+N`) atau debit (`-N`) baru dengan menyertakan `idempotency_key` (UUID v4) dan keterangan alasan operasional.
3. **Prinsip Soft-Limit pada Downgrade / Pembatalan (K-09)**:
   - Tindakan administratif berupa pembatalan atau penurunan paksa paket pengguna (`Force Downgrade`) TIDAK BOLEH menghapus baris data keuangan, rekening dompet, atau riwayat transaksi pengguna.
   - Sistem hanya mengubah level entitlement langganan pengguna; pembatasan akses resource baru dieksekusi oleh backend saat user mencoba membuat data baru.

---

## 4. Aturan Audit Trail & Kepatuhan Hukum (K-29 & K-30)

1. **Pencatatan Wajib Seluruh Aksi Mutasi**:
   - Setiap aksi administratif yang mengubah status data (`CREATE`, `UPDATE`, `DELETE`, `EXTEND`, `APPROVE`, `REJECT`, `REFUND`, `KILL_SWITCH_TOGGLE`, `SETTING_SAVE`) WAJIB menghasilkan baris baru di tabel `audit_logs` dalam transaksi yang sama.
2. **Metadata Audit yang Lengkap**:
   - Setiap baris audit log wajib merekam: `id` (UUID), `actor_id` (UUID admin), `actor_email`, `action`, `entity_type`, `entity_id`, `before_state` (JSON), `after_state` (JSON), `ip_address`, `user_agent`, dan `created_at` (Asia/Jakarta).
3. **Sanitasi Kepatuhan UU PDP No. 27/2022**:
   - Data pribadi sensitif, kata sandi, secret key gateway, dan token otentikasi DILARANG disimpan dalam teks mentah pada kolom `before_state` / `after_state`.
   - Field sensitif wajib disamarkan (*masked*) dengan format `***` sebelum diserialisasi ke log.

---

## 5. Aturan Operasional AI Gateway & Keamanan Darurat (K-17 & K-18)

1. **Konfirmasi Tindakan Berisiko Tinggi**:
   - Operasi penekanan tombol `Emergency Kill Switch` atau pemutusan endpoint wajib menampilkan dialog modal konfirmasi berkode bahaya (warna merah) dengan pengetikan kata kunci konfirmasi manual ("MATIKAN" atau "KILL").
2. **Sinkronisasi Redis Pub/Sub**:
   - Setiap perubahan status endpoint, perutean use-case, atau toggle kill switch wajib langsung mempublikasikan event ke channel Redis `aigateway:config:events` agar seluruh instance worker backend mengetahui perubahan seketika (< 100ms) tanpa reload.
3. **Integritas Embedding Vector 768d**:
   - Model AI untuk use-case `embedding` DILARANG diubah ke model dengan dimensi output selain 768 dimensi (karena terikat skema PostgreSQL `vector(768)`).

---

## 6. Aturan Zero Downtime Deployment & Backward Compatibility (K-23 & K-24)

1. **Pendekatan Strangler Fig**:
   - Kode admin dibangun di package baru `internal/admin` dan tidak boleh merusak fungsi routing pengguna reguler yang sudah berjalan stabil sejak fase P01.
2. **Feature Flag Boundary**:
   - Fitur Super Admin dikendalikan oleh flag platform `feature_super_admin_portal`. Jika flag dimatikan, endpoint admin merespons HTTP 503 yang aman tanpa merusak ketersediaan layanan publik aplikasi.
3. **Replayability Pengujian**:
   - Setiap skrip migrasi database atau test runner wajib bersifat *idempotent* dan *replayable* (dapat dijalankan berulang kali tanpa memicu galat duplikasi skema).
