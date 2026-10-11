# Work Breakdown Structure (Tasks) - Fase P08

Daftar tugas terstruktur untuk Fase P08 (Hardening & Go-Live). Format tugas: `TASK-P08-XXX`.

---

### TASK-P08-001: Implementasi Skrip Load Test k6 Race Condition Kuota Token
- **Deskripsi**: Buat skrip k6 untuk menyimulasikan 20 Virtual Users (VU) menembak endpoint `/api/v1/ai/chat` secara paralel pada saat sisa saldo pengguna tepat 5 token.
- **Kriteria Penerimaan**:
  - Tepat <= 5 request menghasilkan status HTTP 200 dengan pemotongan token valid.
  - Tepat >= 15 request menghasilkan status HTTP 429 (`insufficient_quota`).
  - Saldo token di database tidak pernah bernilai negatif.
  - Tercatat di tabel `token_ledger` secara akurat.

---

### TASK-P08-002: Implementasi Skrip Load Test k6 Replay Webhook Pembayaran
- **Deskripsi**: Buat skrip k6 untuk menembak endpoint webhook `/api/v1/payment/webhook/{provider}` dengan 10 request simultan yang membawa signature dan `idempotency_key` identik dalam selang waktu < 50ms.
- **Kriteria Penerimaan**:
  - Hanya 1 eksekusi yang memutasi status invoice dan menambahkan periode langganan pengguna.
  - 9 request paralel lainnya ditolak atau ditanggapi idempotently tanpa mutasi ganda.
  - Tidak ada duplicate record pada tabel `payments` dan `subscription_events`.

---

### TASK-P08-003: Benchmark Beban Sustained AI Gateway (500 RPS) & Metrik Latensi
- **Deskripsi**: Eksekusi pengujian beban berkesinambungan 500 RPS selama 10 menit ke routing inferensi AI FinTrack v2.
- **Kriteria Penerimaan**:
  - Error rate 5xx di bawah 0.01%.
  - Latensi p95 backend (di luar waktu inferensi model AI upstream) di bawah 50ms.
  - Database pool connection tetap stabil di bawah batas saturasi 80%.

---

### TASK-P08-004: Uji Chaos Failover Endpoint 9router Under Load
- **Deskripsi**: Jalankan load test 300 RPS ke AI Gateway, lalu matikan secara mendadak instans primer 9router (port 20128).
- **Kriteria Penerimaan**:
  - Sirkuit endpoint primer berpindah ke status OPEN dalam <= 2 detik.
  - Trafik otomatis beralih ke endpoint sekunder tanpa menghasilkan error 5xx ke sisi pengguna akhir.
  - Log sirkuit dan metrik Prometheus `ai_gateway_failover_total` tereskalasi dengan benar.

---

### TASK-P08-005: Uji Chaos Pemutusan Koneksi Redis & Validasi Fallback In-Memory
- **Deskripsi**: Putuskan jaringan Redis atau hentikan container Redis saat sistem sedang menerima lalu lintas produksi.
- **Kriteria Penerimaan**:
  - Backend tidak mengalami crash (panic).
  - Rate limiting dan caching sesi otomatis beralih ke in-memory cache lokal.
  - Data otoritatif dibaca langsung dari PostgreSQL dengan aman.
  - Setelah Redis online kembali, koneksi auto-reconnect tanpa restart container Go.

---

### TASK-P08-006: Audit Enkripsi Kredensial Database At-Rest (AES-256-GCM)
- **Deskripsi**: Verifikasi implementasi enkripsi at-rest untuk semua secret sensitif (kunci API gateway, token payment, token OAuth) di tabel database.
- **Kriteria Penerimaan**:
  - Data di kolom tabel tersimpan dalam format base64 dari `nonce (12 byte) + ciphertext + auth tag (16 byte)`.
  - Master key dibaca murni dari environment variable `APP_ENCRYPTION_MASTER_KEY`.
  - Dekripsi menghasilkan plaintext valid secara transparan pada aplikasi.

---

### TASK-P08-007: Validasi Kriptografis Webhook Signature (Midtrans SHA-512 & Xendit Token)
- **Deskripsi**: Lakukan security review dan penetrasi pengujian terhadap endpoint penerima webhook pembayaran.
- **Kriteria Penerimaan**:
  - Webhook Midtrans dengan signature hasil manipulasi ditolak mentah-mentah dengan HTTP 401.
  - Webhook Xendit diverifikasi menggunakan pembanding waktu-konstan (`crypto/subtle`).
  - Setiap upaya pemalsuan signature dicatat di tabel `audit_logs`.

---

### TASK-P08-008: Verifikasi Postur Kepatuhan PCI-DSS SAQ A & Keamanan Browser
- **Deskripsi**: Evaluasi arsitektur frontend dan backend terhadap kriteria kepatuhan PCI-DSS SAQ A.
- **Kriteria Penerimaan**:
  - Terverifikasi 100% data kartu kredit dimasukkan pada form terisolasi milik payment gateway.
  - Tidak ada log backend atau frontend yang mencatat nomor PAN kartu kredit.
  - Header keamanan web terpasang ketat: Content-Security-Policy (CSP), Strict-Transport-Security (HSTS), X-Content-Type-Options, X-Frame-Options.

---

### TASK-P08-009: Implementasi & Pengujian Modul Anonimisasi Subjek Data UU PDP
- **Deskripsi**: Bangun dan uji modul `anonymize_user_data` untuk memenuhi hak penghapusan data pribadi (UU PDP Pasal 43).
- **Kriteria Penerimaan**:
  - Kolom nama, email, nomor telepon, dan password hash pada tabel `users` disamarkan permanen menjadi hash acak.
  - Data chat history dan pgvector embedding 768d dihapus secara tuntas (*hard delete*).
  - Kunci enkripsi personal pengguna dihancurkan (*cryptographic shredding*).

---

### TASK-P08-010: Validasi Integritas Retensi Data Pajak 10 Tahun vs Anonimisasi
- **Deskripsi**: Uji konsistensi tabel pembukuan keuangan (`invoices`, `payments`, `token_ledger`) setelah proses anonimisasi akun pengguna.
- **Kriteria Penerimaan**:
  - Record faktur pajak dan pembayaran tetap ada dengan nominal, PPN, dan timestamp tidak berubah.
  - Kolom identitas faktur menampilkan referensi anonim yang sah untuk audit pajak (UU KUP Pasal 28).
  - Laporan rekonsiliasi total pendapatan perusahaan tetap balance hingga satuan rupiah terkecil.

---

### TASK-P08-011: Disaster Recovery Drill (Pengujian Restorasi PITR PostgreSQL)
- **Deskripsi**: Lakukan simulasi kerusakan basis data total dan lakukan prosedur Point-in-Time Recovery (PITR) dari WAL archive dan snapshot cold backup.
- **Kriteria Penerimaan**:
  - Data billing berhasil dipulihkan dengan kehilangan data (RPO) <= 1 menit.
  - Sistem pulih dan siap melayani request (RTO) dalam waktu <= 15 menit.
  - Integritas data diverifikasi dengan checksum database.

---

### TASK-P08-012: Uji Latih Runbook Operasional (Rotasi Kunci & Invalidasi Cache Basi)
- **Deskripsi**: Lakukan uji coba operasional (fire drill) mengikuti runbook resmi penanganan insiden.
- **Kriteria Penerimaan**:
  - Rotasi master encryption key berhasil dilakukan tanpa downtime dan data lama berhasil didekripsi/dire-enkripsi.
  - Perubahan hak entitlement di super admin tersinkronisasi instan ke seluruh pod backend via Redis Pub/Sub tanpa restart pod.

---

### TASK-P08-013: Konfigurasi Dasbor Observability Four Golden Signals & Alerting
- **Deskripsi**: Setup dashboard Prometheus/Grafana dan konfigurasi notifikasi alerting untuk metrik sistem dan bisnis.
- **Kriteria Penerimaan**:
  - Panel terpasang untuk Golden Signals: Latensi (p50/p95/p99), Trafik (RPS), Error (5xx/4xx), Saturasi (CPU, RAM, DB Pool).
  - Panel metrik bisnis terpasang: Active subscriptions, AI Token Burn Rate, Webhook Success Rate.
  - Alert routing aktif ke channel peringatan darurat SRE.

---

### TASK-P08-014: Verifikasi Menyeluruh & Penandatanganan Go-Live Checklist
- **Deskripsi**: Melakukan audit menyeluruh terhadap 100% item pada lembar verifikasi go-live.
- **Kriteria Penerimaan**:
  - Seluruh kriteria fungsional, performa, keamanan, hukum, dan backup bertanda centang lolos (*PASSED*).
  - Dokumen serah terima operasional disahkan oleh tim pengembang dan arsitek.
  - Sistem dinyatakan siap untuk pembukaan akses publik FinTrack v2.
