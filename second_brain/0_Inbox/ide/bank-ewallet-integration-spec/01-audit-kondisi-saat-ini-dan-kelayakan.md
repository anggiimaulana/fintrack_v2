# 01 — Audit Kondisi Saat Ini & Analisis Kelayakan

**Topik:** Audit Teknis Codebase FinTrack v2 & Analisis Kelayakan Integrasi Finansial Indonesia  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Audit Kondisi Codebase FinTrack v2

### 1.1 Model Transaksi Saat Ini (`backend/internal/domain/transaction.go`)
Saat ini, tabel `transactions` menyimpan data keuangan secara flat per pengguna (`user_id`). Karakteristik model saat ini:
- **Tipe Transaksi:** Dibatasi oleh enum `income`, `expense`, dan `savings`. Belum ada tipe `transfer` untuk perpindahan dana antar-akun pengguna sendiri.
- **Ketiadaan Entitas Akun Keuangan:** Transaksi tidak memiliki referensi `account_id` atau `wallet_id`. Seluruh transaksi langsung diagregasikan menjadi satu dompet abstrak per pengguna.
- **Dukungan Multi-Kurs (P09):** Sudah tersedia kolom `amount`, `currency_code`, `exchange_rate`, `amount_base` (IDR), dan `rate_at`. Kolom `amount_base` selalu menjadi basis kalkulasi agregasi akuntansi FinTrack.
- **Sumber Transaksi (`source`):** Mendukung `manual`, `ai`, `voice`, dan `receipt`. Belum ada sumber `statement_import` atau `open_banking`.
- **Deduplikasi:** Belum ada mekanisme pendeteksi duplikasi transaksi mutasi perbankan.

### 1.2 Entitas Pembayaran yang Ada (`backend/internal/domain/payment_account.go`)
Codebase memiliki entitas `PaymentAccount`, tetapi entitas ini **bukan** milik pengguna untuk mencatat transaksi pribadi. Entitas tersebut adalah rekening tujuan transfer manual milik pengelola platform FinTrack v2 untuk menerima pembayaran langganan (checkout paket). Oleh karena itu, diperlukan entitas baru: `financial_accounts`.

### 1.3 Sistem RAG & Vektor Saat Ini (`backend/internal/domain/embedding.go`)
- FinTrack v2 menggunakan ekstensi PostgreSQL `pgvector` melalui tabel `transaction_embeddings`.
- Teks konten embedding saat ini berformat sederhana: `"[Jenis] [Kategori]: [Deskripsi] senilai [Amount] [Currency] pada tanggal [Date]"`.
- Ketiadaan informasi rekening/bank membuat asisten AI tidak dapat menjawab pertanyaan spesifik seperti: *"Berapa saldo tersisa di Mandiri?"* atau *"Tampilkan transaksi QRIS dari GoPay bulan lalu"*.

### 1.4 Sistem Langganan & Entitlement Dinamis (`backend/internal/admin/plans.go` & `billing`)
FinTrack v2 sudah memiliki infrastruktur langganan modern:
- Tabel `plans` dikaitkan dengan tabel relasional `plan_entitlements` (`feature_key`, `is_enabled`) dan `plan_limits` (`limit_key`, `limit_value`).
- Super Admin memiliki endpoint `PATCH /admin/plans/:id` untuk mengubah feature toggle dan batasan numerik secara dinamis tanpa restart aplikasi (*zero downtime*).
- Sistem token AI terpusat (`token_quota` & `token_wallets`) mencatat konsumsi token LLM.

---

## 2. Analisis Kelayakan Integrasi Perbankan & E-Wallet Indonesia

Mengintegrasikan perbankan dan e-wallet di Indonesia memiliki tantangan teknis dan regulasi yang unik:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SPEKTRUM KELAYAKAN INTEGRASI PERBANKAN                   │
├───────────────────────┬──────────────────────┬──────────────────────────────┤
│ METODE                │ ASPEK LEGALITAS      │ ASPEK TEKNIS & KEBERLANJUTAN │
├───────────────────────┼──────────────────────┼──────────────────────────────┤
│ 1. SNAP BI (Resmi)    │ Legal (Bank Indonesia│ Tertutup untuk publik.       │
│                       │ PADG 23/15/PADG/2021)│ Wajib PT, audit ISO 27001,   │
│                       │                      │ deposit B2B, SLA enterprise. │
├───────────────────────┼──────────────────────┼──────────────────────────────┤
│ 2. Web Scraping /     │ Berisiko Tinggi      │ Sangat rapuh. Captcha, OTP   │
│    Reverse-Engineered │ Potensi pelanggaran  │ SMS/WA, device biometric, IP │
│    Private API        │ UU ITE Pasal 30-32.  │ ban, sesi hangus berkala.    │
├───────────────────────┼──────────────────────┼──────────────────────────────┤
│ 3. Agregator Open     │ Legal & Berlisensi   │ Sangat stabil (SDK/Widget).  │
│    Finance (Brick dll)│ Terdaftar OJK/BI.    │ Biaya per koneksi aktif.     │
│                       │                      │ Cocok untuk Fase 2.          │
├───────────────────────┼──────────────────────┼──────────────────────────────┤
│ 4. Universal E-       │ 100% Legal & Aman    │ Menggunakan parser PDF/CSV.  │
│    Statement Parsing  │ Pengguna pegang data,│ Nol biaya API pihak ketiga.  │
│                       │ patuh UU PDP.        │ Pilihan terbaik Fase 1.      │
└───────────────────────┴──────────────────────┴──────────────────────────────┘
```

### 2.1 Mengapa Bukan Web Scraping / Sniffing Kredensial?
1. **Risiko Pidana UU ITE:** Mengakses sistem perbankan dengan memotong mekanisme otentikasi atau menyimpan kredensial internet banking pihak ketiga tanpa izin tertulis melanggar Pasal 30 UU ITE.
2. **Kerapuhan Operasional:** Bank besar (BCA, Mandiri, BRI) menggunakan proteksi web tingkat lanjut (Akamai, Cloudflare Bot Management, F5 WAF). Script scraping sering rusak saat UI internet banking diperbarui sedikit saja.
3. **Penyimpanan Kredensial Berbahaya:** Menyimpan username dan password perbankan pengguna di server FinTrack akan menciptakan liability keamanan yang sangat besar. Sekali terjadi kebocoran data, konsekuensi hukumnya fatal.

### 2.2 Solusi Fase 1: Universal E-Statement Ingestion
Setiap bank dan e-wallet di Indonesia menyediakan fitur unduh mutasi resmi:
- **BCA:** e-Statement PDF bulanan atau mutasi CSV dari KlikBCA / myBCA.
- **Mandiri:** e-Statement PDF dari Livin' by Mandiri.
- **Bank Jago:** Download laporan mutasi bulanan format CSV/PDF langsung dari aplikasi.
- **SeaBank:** Unduh riwayat transaksi bulanan PDF/CSV via aplikasi mobile.
- **GoPay:** Riwayat transaksi bulanan yang dikirim ke email terdaftar (PDF/CSV).
- **DANA:** Unduh riwayat transaksi di menu e-Statement DANA.
- **ShopeePay:** Unduh rekap transaksi via aplikasi Shopee.

**Keunggulan Pendekatan Ini:**
- Nol biaya vendor/agregator pihak ketiga.
- Nol risiko pemblokiran rekening pengguna oleh perbankan.
- 100% patuh UU No. 27 Tahun 2022 (UU PDP) karena pengguna memiliki kendali penuh atas file data miliknya sendiri.

### 2.3 Solusi Fase 2: Agregator Open Finance Berlisensi
Setelah traksi paket Business terbukti, FinTrack v2 dapat membuka kanal otomatisasi menggunakan penyedia Open Finance berizin (misal Brick by OneFinance atau Ayoconnect):
- Pengguna menghubungkan rekening melalui widget otentikasi terenkripsi.
- FinTrack v2 hanya menerima webhook atau query API read-only data mutasi tanpa pernah menyentuh password bank.

---

## 3. Kepatuhan Regulasi UU Pelindungan Data Pribadi (UU PDP)

Dalam merancang fitur ini, FinTrack v2 wajib mematuhi ketentuan UU PDP:
1. **Prinsip Minimisasi Data (Data Minimization):**
   - FinTrack v2 dilarang menyimpan nomor rekening atau nomor kartu secara utuh. Sistem hanya menyimpan mask 4 digit terakhir (contoh: `****1234` atau `0812****8899`).
2. **Persetujuan Eksplisit (Explicit Consent):**
   - Sebelum mengunggah e-statement, pengguna harus menyetujui disclaimer bahwa file mutasi diproses semata-mata untuk pencatatan keuangan dan analitik akun yang bersangkutan.
3. **Hak untuk Dihapus (Right to Erasure):**
   - Jika pengguna menghapus akun bank (`financial_accounts`), seluruh riwayat mutasi terkait, batch upload, dan embedding vektor RAG yang bersangkutan harus dihapus secara tuntas (*cascading delete* / *hard purge*).
4. **Isolasi Multi-Tenant:**
   - Setiap query ingestion, pencarian, dan RAG wajib memvalidasi `user_id` di tingkat paling bawah (SQL level) untuk mencegah kebocoran data antar-pengguna (*cross-tenant data leak*).
