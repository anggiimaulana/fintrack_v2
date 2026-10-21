# 02 — Kebutuhan Bisnis & Ruang Lingkup

**Topik:** Spesifikasi Kebutuhan Fungsional, Persona, Cakupan Institusi, & Batasan Sistem  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Latar Belakang & Nilai Bisnis

Pengguna modern di Indonesia mengelola keuangan mereka di banyak tempat sekaligus: rekening gaji di BCA atau Mandiri, rekening tabungan bunga tinggi di Bank Jago atau SeaBank, dan berbagai dompet digital seperti GoPay, DANA, serta ShopeePay untuk transaksi harian, belanja online, dan pesan makanan.

Saat ini, pencatatan manual memerlukan waktu dan disiplin tinggi, mengakibatkan:
1. **Pencatatan yang Terbengkalai:** Pengguna sering melewatkan transaksi-transaksi kecil di e-wallet (QRIS, biaya parkir, subscription mikro).
2. **Ketiadaan Visibilitas Saldo Riil:** Pengguna tidak mengetahui total kekayaan bersih (*Net Worth*) secara instan karena saldo terpecah di berbagai aplikasi.
3. **Double Counting saat Top-Up:** Pengguna sering mencatat top-up e-wallet dari bank sebagai pengeluaran (*expense*), dan saat saldo e-wallet dibelanjakan dicatat kembali sebagai pengeluaran, sehingga laporan keuangan menjadi terdistorsi.

Dengan menghadirkan integrasi multi-bank/e-wallet melalui **Universal Statement Ingestion** dan **Open Finance Ready Architecture**, FinTrack v2 memberikan solusi pencatatan otomatis tanpa batas, rekonsiliasi transfer internal tanpa bias, dan konteks data kaya untuk asisten AI (RAG).

---

## 2. Target Persona Pengguna

### Persona 1: Pemilik Bisnis & UMKM (Paket Business)
- Menggunakan rekening BCA/Mandiri untuk operasional usaha, dan akun e-wallet (GoPay/ShopeePay) untuk transaksi operasional harian.
- Butuh mengunggah mutasi bulanan dalam hitungan detik, mengetahui arus kas keluar-masuk, serta melihat vendor/supplier terbesar tanpa input manual satu per satu.

### Persona 2: Digital Freelancer & Remote Worker (Paket Business / Pro)
- Menggunakan Bank Jago (karena kantong/pockets) dan SeaBank untuk investasi/tabungan likuid.
- Membutuhkan asisten AI yang dapat ditanya: *"Berapa total pemasukan dari klien X ke Bank Jago selama kuartal ini?"* atau *"Apakah pengeluaran e-wallet saya melebihi budget jajan bulan ini?"*.

---

## 3. Cakupan Institusi Keuangan (Fase 1)

Sistem wajib menyediakan parser bawaan (*native adapter*) untuk institusi perbankan dan dompet digital terbesar di Indonesia:

| Institusi | Tipe Akun | Format File Masukan | Karakteristik Khas Mutasi |
|---|---|---|---|
| **Bank Central Asia (BCA)** | Bank | CSV (KlikBCA) & PDF (myBCA) | Deskripsi multi-line, kode cabang, indikator `CR`/`DB`, format nominal koma/titik Indonesia (`10.000,00`). |
| **Bank Mandiri** | Bank | PDF (Livin' by Mandiri) | Format tabel dengan kolom Tanggal, Keterangan, Mutasi (Debit/Kredit), dan Saldo Akhir. |
| **Bank Jago** | Bank Digital | CSV & PDF (Jago App) | Header CSV modern, kolom terpisah untuk *Transaction Type*, *Amount*, *Recipient/Sender*, dan *Pockets*. |
| **SeaBank** | Bank Digital | CSV & PDF (SeaBank App) | Laporan mutasi bulanan terstruktur dengan timestamp detail dan referensi nomor transaksi. |
| **GoPay** | E-Wallet | PDF e-Statement & CSV Email | Identifikasi transaksi QRIS, GoFood, GoCar, GoSend, dan biaya top-up via bank transfer. |
| **DANA** | E-Wallet | PDF e-Statement bulanan | Rincian transaksi Merchant DANA, Kirim Uang, dan Tarik Saldo. |
| **ShopeePay** | E-Wallet | PDF e-Statement / Rekap CSV | Rincian transaksi belanja marketplace, ShopeeFood, dan QRIS merchant offline. |
| **Generic Financial CSV** | Universal | CSV / Excel (.xlsx) | Parser fleksibel dengan mapping kolom dinamis (`date`, `description`, `amount`, `type`) untuk bank lain (BRI, BNI, CIMB, OVO). |

---

## 4. Ruang Lingkup Fungsional

### 4.1 Manajemen Akun Finansial (`financial_accounts`)
- Pengguna dapat membuat akun baru dengan memilih institusi (BCA, Mandiri, Jago, SeaBank, GoPay, DANA, ShopeePay, atau Custom/Tunai).
- Memasukkan nama akun personal (misal: "BCA Tabungan Pribadi", "GoPay Belanja").
- Memasukkan nomor rekening bertopeng (*masked number* max 4 digit akhir, contoh: `****5678`).
- Mengatur saldo awal (*initial balance*) dan mata uang (default `IDR`).
- Sistem otomatis menghitung saldo berjalan (*current balance*) setiap kali transaksi ditambahkan, diedit, atau dihapus.

### 4.2 Ingestion File Mutasi & Deduplikasi Fingerprint
- Pengguna mengunggah berkas mutasi melalui antarmuka drag-and-drop.
- Sistem mengekstrak baris mutasi dan menghitung fingerprint unik SHA-256 untuk setiap baris:
  $$\text{Fingerprint} = \text{SHA256}(\text{user\_id} \parallel \text{account\_id} \parallel \text{date} \parallel \text{amount} \parallel \text{normalized\_description})$$
- Jika fingerprint sudah ada di database, baris dilewati (*skipped duplicate*) tanpa melempar error, mencegah penghitungan ganda saat berkas mutasi dengan rentang tanggal beririsan diunggah ulang.

### 4.3 Rekonsiliasi Otomatis Transfer Antar-Rekening (Smart Transfer Pairing)
- Sistem memindai transaksi keluar dari Akun A dan transaksi masuk di Akun B milik pengguna yang sama.
- Jika terdapat kecocokan nominal persis dan selisih tanggal $\le 36\text{ jam}$ serta pola deskripsi mengindikasikan transfer/top-up, sistem:
  - Mengubah tipe transaksi menjadi `transfer`.
  - Mengaitkan kedua baris dengan `transfer_pair_id`.
  - Mengeliminasi kedua transaksi tersebut dari perhitungan total pengeluaran (*expense*) dan total pendapatan (*income*) bersih pengguna.

### 4.4 Riwayat Batch & Pembatalan Satu Klik (Batch Rollback)
- Setiap kali file diimpor, sebuah baris dibuat di tabel `import_batches`.
- Menampilkan metrik: Total baris, baris berhasil diimpor, baris duplikat yang diabaikan.
- Pengguna memiliki tombol "Batalkan Impor (Undo)": seluruh transaksi dalam batch tersebut di-soft delete dan saldo akun dikembalikan ke posisi sebelum impor file dilakukan.

### 4.5 Pengayaan Vektor RAG & Visualisasi Analitik
- Transaksi yang berhasil masuk otomatis dibuatkan vector embedding di `transaction_embeddings` dengan metadata terstruktur (`account_id`, `institution`, `merchant`, `is_transfer`).
- Visualisasi analitik khusus:
  - Kartu Saldo Multi-Akun & Total Kekayaan Bersih (*Net Worth Card*).
  - Grafik Aliran Kas (*Sankey / Cash Flow Chart*).
  - Grafik Vendor/Merchant Terbesar (*Top Merchants Breakdown*).

### 4.6 Pembatasan Paket & Entitlement Dinamis
- Fitur integrasi ini dikendalikan oleh switch `plan_entitlements.bank_integration`.
- Pada rilis awal, switch aktif (`true`) hanya pada paket **Business**.
- Pengguna pada paket Free atau Pro akan melihat banner promosi/upsell dengan penjelasan keunggulan fitur.
- Batasan jumlah akun dan frekuensi impor dikendalikan oleh `plan_limits`:
  - `max_financial_accounts`
  - `monthly_statement_imports`
- Super Admin dapat mengubah konfigurasi ini kapan saja via dashboard admin dengan efek langsung tanpa redeploy (*zero downtime*).

---

## 5. Batasan Sistem & Non-Goals (Out of Scope)

Untuk menjaga kejelasan arsitektur dan kepatuhan hukum, hal-hal berikut **bukan** bagian dari sistem:
1. **Bukan Sarana Pembayaran:** FinTrack v2 tidak menyediakan tombol untuk mentransfer uang keluar dari rekening bank pengguna ke pihak ketiga.
2. **Tidak Menyimpan Kata Sandi/PIN:** Sistem tidak pernah meminta atau menyimpan kredensial login internet banking / e-wallet pengguna.
3. **Tidak Menggunakan Scraping Agresif:** Sistem tidak menjalankan browser otomatis untuk membongkar portal e-banking bank.
4. **Fase 2 Agregator Ditangguhkan Hingga Fase 1 Stabil:** Integrasi SDK Brick / Ayoconnect dirancang antarmukanya (*interface contract*), namun aktivasi koneksi B2B menunggu lisensi operasional.
