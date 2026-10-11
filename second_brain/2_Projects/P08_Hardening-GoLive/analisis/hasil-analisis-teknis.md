# Analisis Teknis & Keputusan Arsitektur Fase P08 (Hardening & Go-Live)

Dokumen ini memuat keputusan teknis mengikat (K-P08-01 sampai K-P08-16) yang menjadi landasan operasional, pengujian, keamanan, dan kehandalan FinTrack v2 sebelum fase produksi aktif.

---

### K-P08-01: Arsitektur Eksekusi Load Test Menggunakan k6 Distributed Headless
- **Konteks**: Perlu menguji ketahanan konkurensi ledger token AI, isolasi transaksi PostgreSQL, dan integritas webhook pembayaran tanpa bias latensi lokal mesin pengembang.
- **Keputusan**: Seluruh skrip pengujian beban ditulis dalam JavaScript k6 standar industri. Skenario k6 dibagi menjadi 3 grup terisolasi:
  1. *Spike Test Quota Token*: 20 Virtual Users (VU) menembak endpoint `/api/v1/ai/chat` serentak pada saat kuota pengguna tersisa tepat 5 token. Validasi wajib: tepat 5 response HTTP 200, 15 response HTTP 429 (`insufficient_quota`), tidak terjadi saldo token negatif pada database.
  2. *Webhook Idempotency Stress*: 10 VU mengirimkan payload webhook transaksi Midtrans/Xendit yang sama dengan header signature valid secara simultan dalam jendela waktu < 50ms. Validasi wajib: tepat 1 request mengembalikan HTTP 200/204 (proses status update invoice/payment), 9 request lainnya mengembalikan HTTP 200/204 atau 409 dengan payload `duplicate_event_ignored`, tabel `payments` dan `invoices` tidak memuat duplikasi record, saldo pengguna tidak bertambah ganda.
  3. *Sustained AI Gateway Load & Failover*: 500 Virtual Users menghasilkan beban 300-500 RPS ke endpoint inferensi, sementara salah satu endpoint upstream 9router di-kill secara paksa. Validasi: tingkat kesalahan klien < 0.05%, p95 latensi meningkat tidak lebih dari 350ms selama proses failover.

---

### K-P08-02: Isolasi Transaksi & Atomisitas Ledger Token AI
- **Konteks**: Race condition pada pemotongan kuota token dapat menyebabkan eksploitasi over-usage yang merugikan biaya operasional perusahaan.
- **Keputusan**: Alur token ledger menggunakan pola *Two-Phase Reserve & Commit* yang diatur dengan transaksi database PostgreSQL level `READ COMMITTED` disertai penguncian baris eksplisit:
  ```sql
  SELECT current_balance FROM token_periods 
  WHERE user_id = $1 AND period_end >= NOW() 
  FOR UPDATE;
  ```
  Pengurangan saldo token dan penulisan baris reservasi di tabel `token_ledger` terjadi dalam satu unit transaksi atomik. Jika sisa saldo kurang dari token estimasi request, transaksi di-rollback seketika dan middleware melepaskan lock sebelum memanggil endpoint AI eksternal, mengembalikan respons status HTTP 429.

---

### K-P08-03: Jaminan Idempotensi Mutasi Pembayaran & Webhook
- **Konteks**: Penyedia payment gateway (Midtrans, Xendit) menerapkan kebijakan retry webhook otomatis jika respons jaringan lambat, memicu risiko pencatatan ganda.
- **Keputusan**: 
  - Kunci idempotensi `idempotency_key` pada tabel `payments` dan `subscription_events` didefinisikan dengan constraint `UNIQUE`.
  - Pola handling webhook:
    1. Verifikasi tanda tangan digital HMAC.
    2. Cek eksistensi event ID atau transaction ID di tabel `subscription_events` dalam transaksi terisolasi.
    3. Jika kunci sudah tercatat dengan status terminal, tanggapi langsung dengan HTTP 200 (ACK) tanpa memicu ulang pemrosesan bisnis.
    4. Kunci Redis sementara berumur 60 detik (`SET payment_lock:<event_id> 1 NX EX 60`) digunakan untuk menyaring request paralel sebelum menyentuh PostgreSQL.

---

### K-P08-04: Enkripsi Kredensial At-Rest & Manajemen Kunci (AES-256-GCM)
- **Konteks**: Kunci API endpoint AI 9router, secret payment gateway, dan kredensial sensitif lainnya tersimpan di tabel database dan berisiko jika backup bocor.
- **Keputusan**:
  - Semua secret disimpan dalam bentuk ciphertext menggunakan AES-256-GCM.
  - Vektor inisialisasi (Nonce 12-byte) dibuat secara acak kriptografis (`crypto/rand`) untuk setiap operasi enkripsi dan digabungkan di depan ciphertext (`nonce || ciphertext || tag`).
  - Kunci enkripsi master (*Master Key*) disuplai murni melalui environment variable sistem (`APP_ENCRYPTION_MASTER_KEY`), tidak pernah ditulis ke git repository atau database.
  - Disediakan mekanisme rotasi kunci zero-downtime: sistem mendukung `APP_ENCRYPTION_MASTER_KEY_CURRENT` dan `APP_ENCRYPTION_MASTER_KEY_PREVIOUS`. Proses re-enkripsi berjalan di latar belakang via perintah CLI migrasi data.

---

### K-P08-05: Verifikasi Kriptografis Webhook Payment Gateway
- **Konteks**: Webhook yang tidak diverifikasi memungkinkan penyerang menyuntikkan callback pembayaran palsu dan mengaktifkan langganan tanpa pembayaran riil.
- **Keputusan**:
  - Webhook Midtrans wajib diverifikasi menggunakan algoritma SHA-512 dengan rumus: `SHA512(order_id + status_code + gross_amount + ServerKey)`. Nilai hash wajib identik dengan header `signature_key`.
  - Webhook Xendit wajib diverifikasi menggunakan token callback rahasia yang dicocokkan melalui komparasi konstan (*constant-time comparison* `subtle.ConstantTimeCompare`) terhadap header `x-callback-token` guna mencegah serangan side-channel timing attack.
  - Request webhook yang gagal verifikasi signature langsung ditolak HTTP 401 dan dicatat di `audit_logs` sebagai indikasi insiden keamanan `SECURITY_WEBHOOK_SIGNATURE_MISMATCH`.

---

### K-P08-06: Arsitektur Kepatuhan PCI-DSS SAQ A
- **Konteks**: FinTrack v2 memproses pembayaran kartu kredit pengguna B2C namun tidak memiliki infrastruktur bersertifikasi PCI-DSS Level 1/2.
- **Keputusan**:
  - FinTrack v2 mengadopsi postur **PCI-DSS SAQ A (Self-Assessment Questionnaire A)**.
  - Seluruh alur input data kartu kredit, CVV, dan tanggal kadaluarsa didelegasikan 100% ke halaman pembayaran ter-hosting aman milik Payment Gateway (Snap Midtrans / Checkout Xendit) melalui redirect browser atau modal iframe terisolasi yang di-serve langsung dari origin payment gateway.
  - Server FinTrack v2 (Go backend) dan Frontend Next.js TIDAK PERNAH menerima, memproses, mentransmisikan, atau menyimpan *Cardholder Data* (PAN) maupun *Sensitive Authentication Data* (CVV).
  - Kebijakan HTTP Content-Security-Policy (CSP) membatasi `frame-src` dan `script-src` hanya ke domain resmi Midtrans dan Xendit.

---

### K-P08-07: Prinsip Least Privilege Role Database & Container
- **Konteks**: Akses database dan runtime kontainer yang berlebihan memperbesar dampak eskalasi hak akses (*privilege escalation*).
- **Keputusan**:
  - Aplikasi Go backend berjalan di bawah user PostgreSQL `fintrack_app` yang hanya memiliki hak `SELECT`, `INSERT`, `UPDATE`, `DELETE` pada tabel operasional. Dilarang memiliki hak `SUPERUSER`, `CREATEROLE`, atau `DROP TABLE`.
  - DDL / Eksekusi migrasi database berjalan dalam pipeline deploy terpisah menggunakan user khusus migrasi `fintrack_migrator`.
  - Container Docker backend dan frontend wajib dijalankan dengan user non-root (`UID 10001`), root filesystem read-only di mana memungkinkan, dan `no-new-privileges: true`.

---

### K-P08-08: Resiliensi AI Gateway & Multi-Endpoint Circuit Breaker
- **Konteks**: Salah satu endpoint 9router (misal GPU worker lokal atau upstream proxy) dapat mengalami degradasi, hang, atau kehabisan memori secara mendadak.
- **Keputusan**:
  - AI Gateway internal (`internal/aigateway`) mengimplementasikan *Circuit Breaker* berbasis Finite State Machine (Closed, Open, Half-Open) per-endpoint (`ai_endpoints`).
  - Ambang batas: Jika suatu endpoint mengalami 5 kegagalan berturut-turut atau latensi timeout > 10 detik, sirkuit berstatus OPEN selama 30 detik.
  - Mekanisme failover otomatis mengarahkan trafik ke endpoint sekunder yang memiliki status `is_active = true` dan sirkuit `CLOSED`.
  - Endpoint `ai_endpoints` memiliki health checker background yang memvalidasi ketersediaan endpoint via heartbeat `/v1/models` setiap 15 detik.

---

### K-P08-09: Resiliensi Redis Split-Brain & In-Memory Fallback
- **Konteks**: Redis digunakan untuk token rate-limiting, event pub/sub, dan sinkronisasi cache berversi. Jika Redis crash, sistem tidak boleh mati total.
- **Keputusan**:
  - Driver Redis menggunakan pola *Circuit Fallback*: Jika koneksi Redis terputus, backend secara otomatis beralih (*fallback*) ke in-memory cache lokal (go-cache / sync.Map) untuk caching sesi dan langsung membaca ke PostgreSQL untuk data otoritatif.
  - Rate limiting beralih ke in-memory sliding window lokal per-instance dengan batas aman sementara.
  - Log peringatan level WARN diekspor ke monitoring dan reconnect pool berjalan di latar belakang setiap 5 detik.

---

### K-P08-10: Rekonsiliasi UU PDP No. 27/2022 vs Kewajiban Retensi Pajak (Anonimisasi Pajak)
- **Konteks**: UU PDP Pasal 43 memberikan subjek data hak untuk meminta penghapusan data pribadi (*Right to Erasure*). Namun, UU No. 28/2007 (KUP) Pasal 28 dan UU No. 8/1997 tentang Dokumen Perusahaan mewajibkan penyimpanan catatan keuangan, faktur, dan bukti transaksi selama minimal 10 (sepuluh) tahun.
- **Keputusan**:
  - Permintaan penghapusan akun pengguna (`account_deletion_request`) TIDAK AKAN menghapus baris tabel `invoices`, `payments`, dan mutasi `token_ledger` dari database finansial.
  - Solusi implementasi: **Pseudonimisasi dan Anonimisasi Permanen**:
    1. Tabel `users`: data nama, email, nomor HP, password hash, dan profil dihapus atau ditimpa dengan string acak SHA-256 (`deleted-user-<uuid>@anonymized.fintrack.internal`). Kunci kriptografis personal dihancurkan (*cryptographic shredding*).
    2. Data transaksi non-finansial (AI chat history, memory RAG vector 768d di pgvector) dihapus secara tuntas (*hard delete*).
    3. Tabel `invoices` dan `payments`: status user ditandai `is_anonymized = true`. Kolom penagihan publik diganti `Subjek Teranonimisasi - UU PDP`, namun nomor faktur, nominal rupiah, rincian PPN, dan stempel waktu transaksi tetap dipertahankan 100% utuh untuk kepentingan audit kepatuhan perpajakan Indonesia.

---

### K-P08-11: Kebijakan Retensi & Pembersihan Data Otomatis (Data Minimization)
- **Konteks**: Penyimpanan data log mentah dan riwayat chat AI yang tidak terbatas melanggar asas pembatasan penyimpanan UU PDP Pasal 16.
- **Keputusan**:
  - Jadwal retensi data FinTrack v2:
    - Log audit keamanan (`audit_logs`): disimpan selama 2 tahun (kepatuhan regulasi forensik).
    - Log inferensi mentah & telemetry AI Gateway: disimpan 30 hari, kemudian diagregasi dan data mentah dihapus.
    - Sesi cache in-memory & Redis transient keys: TTL maksimal 24 jam.
    - Riwayat chat dan embedding pengguna aktif: disimpan selama langganan aktif atau sampai pengguna menekan tombol hapus chat di aplikasi.

---

### K-P08-12: Target RPO & RTO Disaster Recovery FinTrack v2
- **Konteks**: Bencana infrastruktur (kerusakan disk, korupsi data pusat data) menuntut standar pemulihan yang jelas agar bisnis tidak kehilangan rekam jejak finansial.
- **Keputusan**:
  - Kategori Data Tier 1 (Finansial & Keanggotaan: `invoices`, `payments`, `subscriptions`, `token_ledger`, `users`):
    - **RPO (Recovery Point Objective)**: <= 1 Menit.
    - **RTO (Recovery Time Objective)**: <= 15 Menit.
  - Kategori Data Tier 2 (AI Chat Memory & Embedding: `transaction_embeddings`, `chat_sessions`):
    - RPO: <= 1 Jam.
    - RTO: <= 2 Jam.
  - Implementasi melalui Continuous Archiving PostgreSQL Write-Ahead Log (WAL) ke object storage terpisah yang terenkripsi dan backup snapshot fisik basis data harian.

---

### K-P08-13: Arsitektur Observability Four Golden Signals & Metrik Bisnis
- **Konteks**: Tim operasional memerlukan visibilitas instan terhadap anomali sistem dan kesehatan operasional FinTrack v2 di produksi.
- **Keputusan**:
  - Implementasi Four Golden Signals:
    1. *Latency*: Histogram latensi request HTTP per handler (khususnya AI Gateway p50, p95, p99 dan DB query latency).
    2. *Traffic*: HTTP Requests per second (RPS) dikelompokkan berdasarkan endpoint dan status HTTP.
    3. *Errors*: Error rate 5xx / 4xx, AI upstream timeout count, webhook validation failure count.
    4. *Saturation*: Utilisasi CPU/Memory kontainer, connection pool usage PostgreSQL (`max_connections = 100`), Redis memory fragmentation ratio.
  - Metrik Bisnis Kritis:
    1. Active Subscriptions Count (per plan).
    2. Token Burn Rate per jam/hari.
    3. Payment Conversion & Failure Rate per Gateway (Midtrans vs Xendit).
    4. 9router Endpoint Availability Percentage & Failover Counter.

---

### K-P08-14: Invalidation Cache Basi Lintas Instans Menggunakan Redis Pub/Sub
- **Konteks**: Perubahan data konfigurasi plan, pricing rules, hak entitlement, dan feature flag oleh super admin harus terefleksi instan di seluruh instans backend tanpa jeda polling.
- **Keputusan**:
  - Channel khusus Redis `fintrack:cache:invalidate` digunakan sebagai media broadcast event invalidasi.
  - Setiap instans backend berlangganan (*subscribe*) ke channel ini pada saat boot.
  - Saat ada mutasi entitlement/flag di super admin, pesan JSON berisi `{ "scope": "entitlements", "user_id": "...", "version": 42 }` disiarkan. Seluruh instans membersihkan cache in-memory lokal masing-masing secara bedah tanpa perlu restart aplikasi.

---

### K-P08-15: Prosedur Rotasi Secret & Kunci Akses Tanpa Downtime
- **Konteks**: Kebocoran kunci atau audit keamanan rutin mewajibkan rotasi kredensial (JWT secret, DB password, Payment server key, AI Endpoint key) tanpa mengganggu sesi pengguna yang sedang bertransaksi.
- **Keputusan**:
  - JWT Signature Key mendukung key array dengan `kid` (Key ID). Token lama diverifikasi dengan kunci lama sampai kedaluwarsa, token baru ditandatangani dengan kunci baru.
  - Kunci Webhook Payment Gateway didaftarkan ganda di portal payment gateway (secondary key) sebelum primary key dicabut.

---

### K-P08-16: Gerbang Rilis Keras (Strict Go-Live Quality Gate)
- **Konteks**: Menghindari peluncuran prematur dengan risiko kompromi data atau bug finansial.
- **Keputusan**:
  - Rilis ke produksi (Go-Live) HANYA diizinkan jika seluruh 100% item pada `teknis/go-live-checklist.md` bertanda status LULUS (Passed). Tidak ada dispensasi untuk toleransi kegagalan pada kategori Finansial, Keamanan Kriptografis, dan Kepatuhan Hukum UU PDP.
