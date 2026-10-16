# Implementation Rules (Aturan Implementasi Mengikat) - Fase P08

Setiap kode, skrip pengujian, konfigurasi infrastruktur, dan prosedur operasional pada Fase P08 wajib mematuhi aturan baku berikut secara ketat.

---

### RULE-P08-01: Isolasi Saldo Finansial & Ledger Mutasi
- Tidak ada operasi pemotongan saldo kuota token AI atau pengaktifan langganan berbayar yang boleh dieksekusi tanpa transaksi basis data (`BEGIN ... COMMIT`).
- Pemotongan kuota token wajib menggunakan transaksi terisolasi dengan locking baris eksplisit (`SELECT ... FOR UPDATE`) pada periode aktif pengguna untuk mengeliminasi potensi race condition.
- Dilarang keras menggunakan tipe data floating-point untuk nilai mata uang rupiah maupun kuota token. Nilai nominal mata uang dan saldo token wajib berupa bilangan bulat (`BIGINT` atau `INTEGER`).

---

### RULE-P08-02: Kebijakan Idempotensi Mutlak
- Setiap endpoint mutasi atau callback webhook pembayaran wajib menyertakan constraint `UNIQUE(idempotency_key)`.
- Replay webhook atau pengiriman duplikat request pembayaran dengan token yang sama tidak boleh memicu mutasi status tambahan ataupun duplikasi entri pencatatan invoice.
- Handler webhook wajib mengembalikan respon sukses idempotent jika transaksi telah berada pada status final.

---

### RULE-P08-03: Standar Kriptografi & Manajemen Secret
- Algoritma enkripsi simetris data at-rest yang diizinkan hanyalah **AES-256-GCM**.
- Penggunaan Electronic Codebook (ECB) atau Cipher Block Chaining (CBC) tanpa autentikasi HMAC dilarang keras.
- Nonce enkripsi (12-byte) wajib di-generate secara acak kriptografis (`crypto/rand`) untuk setiap entri data. Dilarang mendaur ulang nonce.
- Master encryption key dilarang keras di-hardcode di dalam kode sumber, commit repository, atau disimpan dalam tabel database biasa. Master key wajib disuplai via environment variable sistem yang diamankan.
- Verifikasi tanda tangan digital webhook wajib menggunakan pembanding konstan (*constant-time comparison* via `subtle.ConstantTimeCompare`) guna menangkal side-channel timing attacks.

---

### RULE-P08-04: Prinsip PCI-DSS SAQ A (Zero Cardholder Data)
- FinTrack v2 dilarang menyimpan, memproses, mentransmisikan, atau mencatat data PAN (Primary Account Number), CVV/CVC, maupun tanggal kadaluarsa kartu kredit pada server backend maupun database aplikasi.
- Seluruh form input data kartu kredit wajib di-render secara langsung dari origin penyedia gateway pembayaran (Midtrans Snap / Xendit Checkout) melalui redirect atau iframe bersertifikat.
- Header log aplikasi wajib disanitasi: dilarang mencatat header authorization, raw payload kartu, atau token sensitif ke log stdout/file.

---

### RULE-P08-05: Rekonsiliasi Kepatuhan UU PDP vs Ketentuan Perpajakan
- Pemenuhan hak penghapusan data pribadi (*Right to Erasure*) berdasarkan UU PDP No. 27/2022 Pasal 43 dilarang menghapus riwayat transaksi keuangan pada tabel `invoices`, `payments`, dan `token_ledger` yang diwajibkan disimpan selama minimal 10 tahun berdasarkan UU KUP No. 28/2007.
- Prosedur penghapusan wajib menggunakan teknik pseudonimisasi dan anonimisasi data: identitas PII pada tabel `users` (nama, email, nomor kontak, hash kredensial) disamarkan secara ireversibel, riwayat percakapan AI dan vector embedding dihapus total (*hard delete*), sementara dokumen faktur pajak dan mutasi keuangan dipertahankan dengan atribut penagihan anonim yang sah.

---

### RULE-P08-06: Resiliensi Kegagalan Infrastruktur (Fail-Safe & Graceful Degradation)
- Kegagalan satu atau lebih node endpoint 9router tidak boleh menghasilkan respon error HTTP 5xx pada sisi pengguna selama masih terdapat node cadangan yang sehat di dalam pool.
- Sirkuit circuit breaker wajib beralih ke mode OPEN setelah 5 kegagalan beruntun atau timeout > 10 detik, dan secara bertahap memulihkan diri melalui mode HALF-OPEN.
- Putusnya koneksi ke Redis tidak boleh menyebabkan aplikasi backend mengalami crash (*panic*). Sistem wajib melakukan fallback otomatis ke in-memory cache lokal dan pembacaan langsung ke basis data PostgreSQL.

---

### RULE-P08-07: Integritas Cadangan Basis Data (RPO/RTO)
- Snapshot cadangan basis data wajib dienkripsi dan diuji secara berkala dengan prosedur Point-in-Time Recovery (PITR).
- Standar ketat RPO (Recovery Point Objective) untuk tier data finansial adalah maksimal 1 menit, dan RTO (Recovery Time Objective) maksimal 15 menit.
- Prosedur disaster recovery wajib dapat dijalankan secara terotomatisasi melalui skrip runbook tanpa intervensi manual yang rentan kesalahan.
