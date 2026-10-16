# Antarmuka Review Pembayaran & Refund — FinTrack v2

Dokumen ini mendokumentasikan antarmuka antrean verifikasi pembayaran manual, mitigasi race condition antar-admin, dialog pratinjau bukti bayar, serta alur pencatatan refund administratif pada FinTrack v2.

---

## 1. Antrean Verifikasi Pembayaran Manual (`/super-admin/payments/review`)

Sesuai **K-07** dan alur operasional di Indonesia, transaksi yang menggunakan metode transfer bank manual atau verifikasi bukti bayar memerlukan tinjauan operasional sebelum hak akses pengguna diaktifkan.

### 1.1 Tampilan Antrean & Informasi Kolom
Antrean review ditampilkan menggunakan komponen `<DataTable>` dengan urutan prioritas waktu tunggu terlama (*First-In, First-Out*):
- **Waktu Submit**: Durasi tunggu relatif (`15 menit yang lalu`, `2 jam yang lalu`).
- **Pengguna**: Nama lengkap, email, dan avatar pengguna.
- **Paket & Siklus**: Paket yang dibeli (misal `Pro Bulanan - 30 Hari`).
- **Nominal Tagihan**: Nilai rupiah berformat moneter (`Rp 99.000`).
- **Metode Pembayaran**: Bank tujuan (BCA / Mandiri / BNI Manual).
- **Pratinjau Bukti**: Thumbnail gambar bukti transfer dengan tombol zoom.
- **Tombol Aksi**: Tombol cepat `Setujui (Approve)` dan `Tolak (Reject)`.

---

## 2. Mitigasi Race Condition & Double-Processing

Ketika dua atau lebih admin membuka antrean review pembayaran secara simultan:
1. **Status Kunci Pesimis / Optimis (Backend Protection)**:
   - Handler persetujuan backend mengeksekusi:
     ```sql
     UPDATE payments 
     SET status = 'settled', reviewed_by = $1, reviewed_at = NOW(), version = version + 1
     WHERE id = $2 AND status = 'pending_review' AND version = $3;
     ```
   - Jika admin kedua mencoba menekan tombol persetujuan pada transaksi yang sama, backend mendeteksi benturan versi (`RowsAffected == 0`) dan mengembalikan galat HTTP 409 Conflict dengan pesan ramah: *"Transaksi ini telah disetujui oleh admin lain."*
2. **Kunci UI Klien (Frontend Mutex)**:
   - Tombol tindakan langsung memasuki status *loading/disabled* seketika saat diklik untuk mencegah duplikasi penekanan (*double-click*).

---

## 3. Komponen Pratinjau Bukti Bayar (Proof Viewer Modal)

Dialog modal pratinjau bukti bayar menyediakan:
- **Tampilan Beresolusi Penuh**: Gambar bukti transfer atau dokumen PDF ditampilkan dengan rasio aspek optimal.
- **Fitur Rotasi & Zoom**: Tombol putar 90 derajat searah jarum jam dan zoom in/out untuk memudahkan verifikasi keaslian teks mutasi rekening.
- **Pemeriksaan Nilai Nominal**: Komponen menyoroti nominal yang harus dibayar pengguna di sebelah gambar bukti bayar untuk mempercepat verifikasi kecocokan angka oleh admin.

---

## 4. Alur Persetujuan & Penolakan

### 4.1 Aksi Persetujuan (Approve)
1. Admin menekan tombol `Setujui Pembayaran`.
2. Dialog konfirmasi meminta kepastian admin bahwa dana telah masuk ke mutasi rekening bank FinTrack.
3. Setelah dikonfirmasi:
   - Status payment berubah menjadi `settled`.
   - Invoice resmi bernomor seri otomatis diterbitkan (K-11).
   - Langganan user diaktifkan atau diperpanjang secara otomatis (K-08).
   - Hak akses entitlement pengguna diperbarui seketika via cache berversi (K-14).
   - Aksi dicatat ke `audit_logs` (K-29).

### 4.2 Aksi Penolakan (Reject)
1. Admin menekan tombol `Tolak Pembayaran`.
2. Dialog modal mewajibkan admin memilih atau mengetik alasan penolakan:
   - Nominal transfer tidak sesuai.
   - Bukti bayar buram / tidak terbaca.
   - Mutasi bank tidak ditemukan / terindikasi fiktif.
3. Status payment berubah menjadi `failed` dan notifikasi alasan penolakan dikirimkan ke pengguna.

---

## 5. Dialog Pencatatan Refund Administratif (Record Refund)

Pada halaman riwayat pembayaran (`/super-admin/payments`), admin dapat mencatat pengembalian dana (*refund*) untuk transaksi yang telah diselesaikan (`settled`):
- **Input Nominal Refund**: Nilai integer rupiah (mendukung refund penuh atau refund parsial).
- **Nomor Referensi Bank Pengembalian**: Bukti transfer mutasi balik dana ke rekening pelanggan.
- **Alasan Administratif**: Alasan pengembalian (misal kompensasi insiden layanan, salah transfer oleh user).
- **Hasil Aksi**:
  - Catatan pembayaran diperbarui statusnya menjadi `refunded` atau `partially_refunded`.
  - Nilai pengembalian dana mengurangi metrik pendapatan kotor di dashboard.
  - Sesuai K-09, bila pengembalian dana membatalkan langganan, langganan pengguna diturunkan ke paket Free dengan prinsip soft-limit.
  - Riwayat refund dicatat secara append-only di `audit_logs`.
