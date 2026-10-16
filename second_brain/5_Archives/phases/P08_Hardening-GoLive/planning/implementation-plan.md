# Implementation Plan — P08 Hardening & Go-Live

| Field | Isi |
|---|---|
| **Fase** | P08 — Hardening & Go-Live |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | P01 s/d P07 selesai (Fondasi platform, AI Gateway, Billing Core, Payment, Super Admin, Multi-Agent Analytics siap di staging) |
| **Master plan** | `../../IMPLEMENTATION-PLAN.md` |
| **Status** | Siap dieksekusi — belum ada kode ditulis |

> **Koreksi path & integritas lingkungan:**  
> - Kode frontend berada di `frontend/` (bukan `frontend/src/`). Path nyata: `frontend/app/`, `frontend/components/`, `frontend/hooks/`, `frontend/lib/`.  
> - Backend berada di `backend/` (`backend/cmd/`, `backend/internal/`).  
> - Skrip pengujian, audit, dan otomasi berada di `scripts/loadtest/`, `scripts/chaos/`, `scripts/dr/`, `scripts/runbook/`, dan `monitoring/`.  
> - Dokumen ini merupakan **rencana pelaksanaan teknis**, bukan pelaporan hasil uji historis. Seluruh target performa, ambang toleransi, dan kepatuhan disajikan sebagai **kriteria penerimaan kuantitatif** yang wajib divalidasi sebelum pembukaan akses produksi.

---

## 1. Tujuan & Definition of Done

### 1.1 Tujuan Fase
Menjamin kehandalan ekstrem, integritas finansial, kekebalan keamanan, kepatuhan regulasi hukum (UU PDP No. 27/2022 & UU KUP Perpajakan 10 tahun), kesiapan pemulihan bencana (PITR), dan kelayakan operasional sistem FinTrack v2 sebelum peluncuran resmi ke lingkungan produksi publik (*General Availability*).

### 1.2 Definition of Done (DoD) Fase
- [ ] Seluruh 14 task (`TASK-P08-001` s/d `TASK-P08-014`) terencana tuntas tanpa ada komponen yang terlewat.
- [ ] Pengujian beban k6 membuktikan kekebalan *race condition* kuota token (20 VU vs saldo 5 token: tepat <= 5 sukses, >= 15 ditolak HTTP 429, saldo DB tidak pernah negatif).
- [ ] Pengujian replay webhook membuktikan idempotensi absolut (10 request identik < 50ms: tepat 1 mutasi transaksi, duplicate record = 0).
- [ ] Uji sustained AI Gateway 500 RPS selama 10 menit mencapai target error rate < 0.01% dan latensi p95 backend < 50ms.
- [ ] Uji chaos failover membuktikan pemutusan 9router primer beralih ke node cadangan dalam <= 2 detik tanpa error 5xx ke pengguna.
- [ ] Uji chaos Redis membuktikan graceful degradation ke in-memory fallback tanpa crash (*panic*) dan auto-reconnect saat online kembali.
- [ ] Audit enkripsi kredensial at-rest membuktikan 100% secret sensitif terenkripsi AES-256-GCM dengan nonce acak 12-byte dan master key dari environment variable.
- [ ] Verifikasi webhook signature menolak manipulasi payload dengan HTTP 401 dan perbandingan waktu-konstan (`crypto/subtle`).
- [ ] Postur PCI-DSS SAQ A terpenuhi 100%: server zero-contact terhadap data kartu kredit (PAN/CVV) dan security headers web aktif.
- [ ] Modul anonimisasi UU PDP Pasal 43 tervalidasi: PII disamarkan, histori chat & vector embedding 768d di-hard delete, dan cryptographic shredding tuntas.
- [ ] Rekonsiliasi retensi pajak 10 tahun membuktikan integritas pembukuan nominal Rupiah pada `invoices` dan `payments` tetap seimbang hingga pecahan terkecil.
- [ ] Drill Point-in-Time Recovery (PITR) membuktikan ketercapaian target RPO <= 1 menit dan RTO <= 15 menit.
- [ ] Uji runbook membuktikan keberhasilan rotasi kunci zero-downtime dan propagasi invalidasi cache via Redis Pub/Sub.
- [ ] Dasbor Four Golden Signals dan alerting SRE terkonfigurasi aktif di Grafana/Prometheus.
- [ ] 100% item Go-Live Checklist tertandatangani oleh pemangku kepentingan teknis (Tech Lead, Security, DBA, SRE, Compliance).

---

## 2. Urutan Eksekusi (Wave)

Rencana eksekusi diorganisasikan ke dalam 5 Wave bertahap untuk menjaga stabilitas ketergantungan dan isolasi kegagalan:

| Wave | Task ID | Deskripsi Task | Prasyarat Wajib |
|---|---|---|---|
| **W1: Load Test Scripts (k6)** | `TASK-P08-001`<br>`TASK-P08-002`<br>`TASK-P08-003` | - Skrip k6 race condition kuota token (20 VU vs saldo 5)<br>- Skrip k6 replay webhook pembayaran (10x identik < 50ms)<br>- Benchmark sustained AI Gateway 500 RPS (10 menit) | Staging environment aktif dengan skema P03 (AI Gateway), P04 (Billing), dan P05 (Payment) |
| **W2: Chaos Tests** | `TASK-P08-004`<br>`TASK-P08-005` | - Chaos failover endpoint 9router under load (kill port 20128)<br>- Chaos pemutusan koneksi Redis & validasi in-memory fallback | W1 selesai (baseline beban stabil) & Redis P02 |
| **W3: Security Audit** | `TASK-P08-006`<br>`TASK-P08-007`<br>`TASK-P08-008` | - Audit enkripsi database at-rest (AES-256-GCM)<br>- Validasi kriptografis signature webhook (Midtrans & Xendit)<br>- Verifikasi PCI-DSS SAQ A & audit security headers browser | Modul Payment P05, Crypto P02/P06, & Endpoint Frontend |
| **W4: Compliance (PDP, Pajak, DR)** | `TASK-P08-009`<br>`TASK-P08-010`<br>`TASK-P08-011` | - Modul anonimisasi subjek data UU PDP (Right to Erasure)<br>- Rekonsiliasi retensi data perpajakan 10 tahun (UU KUP)<br>- Disaster Recovery Drill (PITR RPO <= 1m, RTO <= 15m) | W3 selesai (enkripsi & integritas skema terverifikasi) |
| **W5: Observability & Go-Live** | `TASK-P08-012`<br>`TASK-P08-013`<br>`TASK-P08-014` | - Uji runbook operasional (rotasi master key & cache basi)<br>- Dasbor Observability Golden Signals & Alerting SRE<br>- Audit final dan penandatanganan Go-Live Checklist | Seluruh W1, W2, W3, W4 tuntas divalidasi |

---

## 3. Detail Task

### Wave 1: Load Test Scripts (k6)

#### TASK-P08-001 — Implementasi Skrip Load Test k6 Race Condition Kuota Token
- **Target File:**
  - `scripts/loadtest/token_quota_concurrency.js` (skrip k6 pengujian konkurensi)
  - `backend/internal/billing/token_service.go` (verifikasi mekanisme locking `SELECT ... FOR UPDATE`)
  - `backend/internal/billing/token_ledger.go` (pencatatan ledger transaksi token)
  - `reports/loadtest/token-quota-concurrency-result.json` (lokasi artefak luaran k6)
  - `reports/loadtest/token-quota-concurrency-summary.md` (resume evaluasi pengujian)
- **Langkah Rencana:**
  1. Siapkan endpoint seed data khusus staging (`POST /api/v1/test/seed-quota`) yang dilindungi header `X-Test-Secret` untuk mengeset saldo pengguna uji tepat bernilai 5 token.
  2. Susun skrip k6 skenario `per-vu-iterations` dengan 20 Virtual Users (VU) yang mengeksekusi request inferensi ke `/api/v1/ai/chat` secara serentak pada milidetik yang sama.
  3. Konfigurasi metrik k6 kustom (`Counter('quota_successful_requests')` dan `Counter('quota_rejected_requests')`) untuk mencatat distribusi status kode HTTP.
  4. Siapkan query audit pasca-uji untuk memeriksa konsistensi tabel `token_periods` dan tabel `token_ledger`.
- **Target Kriteria & Ambang Batas:**
  - Tepat `<= 5` request menghasilkan status HTTP 200 dengan pemotongan token yang sah.
  - Tepat `>= 15` request menghasilkan status HTTP 429 dengan kode error `insufficient_quota`.
  - Saldo akhir pada kolom `current_balance` tabel `token_periods` tepat bernilai `>= 0` (tidak boleh bernilai negatif dalam kondisi apapun).
  - Jumlah baris mutasi debit pada tabel `token_ledger` untuk request uji identik dengan jumlah HTTP 200.
- **Lokasi Artefak Bukti:** `reports/loadtest/token-quota-concurrency-result.json`, `reports/loadtest/token-quota-concurrency-summary.md`.
- **Catatan Zero-Downtime & Keamanan:** Isolasi akun pengujian menggunakan ID khusus (`usr_test_quota_race`) agar tidak memengaruhi data pengguna aktif staging; endpoint seed hanya aktif jika `APP_ENV != production`.

---

#### TASK-P08-002 — Implementasi Skrip Load Test k6 Replay Webhook Pembayaran
- **Target File:**
  - `scripts/loadtest/webhook_replay.js` (skrip pengujian k6 replay webhook)
  - `backend/internal/payment/webhook_handler.go` (handler webhook dengan validasi idempotensi)
  - `backend/internal/payment/idempotency.go` (mekanisme pengecekan kunci idempotency)
  - `reports/loadtest/webhook-replay-result.json` (lokasi artefak luaran k6)
  - `reports/loadtest/webhook-replay-summary.md` (resume evaluasi pengujian)
- **Langkah Rencana:**
  1. Buat skrip k6 yang menyimulasikan transaksi pembayaran sukses (status `settlement`) dengan generator signature HMAC SHA-512 valid dan `order_id` unik.
  2. Konfigurasi k6 skenario burst paralel: tembakkan 10 request secara serentak dalam selang waktu < 50ms dengan body JSON dan header `X-Idempotency-Key` yang identik ke endpoint `/api/v1/payment/webhook/midtrans`.
  3. Pantau respon HTTP dan ukur konsistensi eksekusi mutasi pada basis data PostgreSQL.
  4. Jalankan query pengecekan duplikasi pada tabel `payments`, `invoices`, dan `subscription_events`.
- **Target Kriteria & Ambang Batas:**
  - Tepat `1` request pertama yang memperoleh kunci lock berhasil memproses mutasi status invoice menjadi `PAID` dan menambahkan hak langganan pengguna.
  - Tepat `9` request konkuren lainnya ditolak atau ditanggapi secara idempotent (HTTP 200 dengan flag `already_processed` atau HTTP 409) tanpa memicu mutasi ganda.
  - Jumlah duplicate record pada tabel `payments` dengan `order_id` yang sama adalah tepat `0`.
  - Jumlah perpanjangan periode langganan pada tabel `subscriptions` bertambah tepat `1` kali.
- **Lokasi Artefak Bukti:** `reports/loadtest/webhook-replay-result.json`, `reports/loadtest/webhook-replay-summary.md`.
- **Catatan Zero-Downtime & Keamanan:** Pengujian memanfaatkan webhook mock gateway; tidak ada kredensial live Midtrans produksi yang digunakan pada fase ini.

---

#### TASK-P08-003 — Benchmark Beban Sustained AI Gateway (500 RPS) & Metrik Latensi
- **Target File:**
  - `scripts/loadtest/ai_gateway_sustained.js` (skrip k6 continuous load generator)
  - `backend/internal/aigateway/router.go` (modul perutean AI Gateway)
  - `backend/internal/aigateway/client.go` (HTTP client pooling ke upstream 9router)
  - `reports/loadtest/ai-gateway-sustained-result.json` (lokasi artefak luaran k6)
  - `reports/loadtest/ai-gateway-sustained-summary.md` (resume performa & latensi)
- **Langkah Rencana:**
  1. Susun skrip k6 dengan skenario `ramping-arrival-rate`: ramp-up dari 50 RPS ke 500 RPS dalam 2 menit, pertahankan sustained 500 RPS selama 10 menit penuh, ramp-down dalam 1 menit.
  2. Request diarahkan ke endpoint `/api/v1/ai/chat` dengan payload token embedding standar (menggunakan mock upstream atau instance 9router lokal terisolasi).
  3. Konfigurasi instrumen metrik latensi terpisah: latensi transit internal Go backend vs latensi pemrosesan inferensi upstream model.
  4. Pantau saturasi CPU host, pemakaian memori container, dan metrik `db_pool_active_connections`.
- **Target Kriteria & Ambang Batas:**
  - Error rate HTTP 5xx berada di bawah ambang batas `< 0.01%` (maksimal 30 error dari estimasi 300.000 total request).
  - Latensi internal Go backend persentil ke-95 (`p95`) berada di bawah ambang batas `< 50ms` (di luar durasi inferensi model upstream).
  - Latensi internal Go backend persentil ke-99 (`p99`) berada di bawah ambang batas `< 120ms`.
  - Database connection pool PostgreSQL tetap berada di bawah saturasi `< 80%` dari kapasitas pool maksimum (`max_open_conns`).
- **Lokasi Artefak Bukti:** `reports/loadtest/ai-gateway-sustained-result.json`, `reports/loadtest/ai-gateway-sustained-summary.md`.
- **Catatan Zero-Downtime & Keamanan:** Pengujian dijalankan pada node worker terdedikasi untuk mencegah resource starvation pada database server.

---

### Wave 2: Chaos Tests

#### TASK-P08-004 — Uji Chaos Failover Endpoint 9router Under Load
- **Target File:**
  - `scripts/chaos/chaos_9router_failover.sh` (otomasi simulasi terminasi proses)
  - `backend/internal/aigateway/circuitbreaker/breaker.go` (state machine sirkuit)
  - `backend/internal/aigateway/pool.go` (manajemen pool multi-endpoint)
  - `reports/chaos/9router-failover-drill.json` (catatan metrik saat failover)
  - `reports/chaos/9router-failover-summary.md` (resume evaluasi ketahanan failover)
- **Langkah Rencana:**
  1. Siapkan dua target endpoint 9router upstream: Node Primer (`localhost:20128`, bobot 100) dan Node Cadangan (`localhost:20129`, bobot 50).
  2. Alirkan beban konstan 300 RPS menggunakan k6 ke rute AI Gateway.
  3. Pada detik ke-30 beban berjalan stabil, jalankan skrip chaos untuk menghentikan paksa container/proses 9router primer (`docker stop fintrack-9router-primary` atau `kill -9`).
  4. Amati respons internal circuit breaker: deteksi 5 consecutive failure atau timeout, eskalasi status ke `OPEN`, dan pengalihan ke node sekunder.
  5. Nyalakan kembali container primer pada detik ke-90 untuk menguji transisi sirkuit dari `OPEN` -> `HALF-OPEN` -> `CLOSED`.
- **Target Kriteria & Ambang Batas:**
  - Sirkuit endpoint primer berpindah ke status `OPEN` dalam waktu `<= 2 detik` setelah pemutusan.
  - Kegagalan trafik dialihkan ke node cadangan dalam waktu `<= 250ms` (*failover switch latency*).
  - Ketersediaan layanan (*success rate*) ke sisi pengguna akhir tetap `>= 99.9%` selama proses transisi berlangsung.
  - Metrik Prometheus `ai_gateway_failover_total` bertambah dan tercatat di dashboard.
- **Lokasi Artefak Bukti:** `reports/chaos/9router-failover-drill.json`, `reports/chaos/9router-failover-summary.md`.
- **Catatan Zero-Downtime & Keamanan:** Circuit breaker mengisolasi node rusak tanpa memblokir thread HTTP handler pengguna; kegagalan tidak memicu panic goroutine.

---

#### TASK-P08-005 — Uji Chaos Pemutusan Koneksi Redis & Validasi Fallback In-Memory
- **Target File:**
  - `scripts/chaos/chaos_redis_disconnect.sh` (skrip pemutusan jaringan Redis via iptables/docker)
  - `backend/internal/infra/redis/client.go` (klien Redis dengan error handling toleran)
  - `backend/internal/infra/redis/cache.go` (abstraksi fallback cache)
  - `backend/internal/delivery/http/middleware/cache.go` (middleware cache dengan in-memory bypass)
  - `reports/chaos/redis-fallback-drill.log` (log audit respon sistem)
  - `reports/chaos/redis-fallback-summary.md` (resume evaluasi fallback)
- **Langkah Rencana:**
  1. Alirkan lalu lintas normal ke rute terproteksi yang memanfaatkan caching sesi dan rate limiting Redis.
  2. Eksekusi pemutusan jaringan Redis menggunakan isolasi network Docker (`docker network disconnect`) atau `docker stop fintrack_redis`.
  3. Evaluasi apakah backend Go tetap melayani request pengguna atau terjadi kegagalan sistem.
  4. Verifikasi bahwa sistem otomatis mengalihkan pembacaan data otorisasi/entitlement langsung ke database PostgreSQL, serta beralih ke cache memori lokal.
  5. Sambungkan kembali Redis ke jaringan dan pantau mekanisme auto-reconnect client Go Redis.
- **Target Kriteria & Ambang Batas:**
  - Backend Go tidak mengalami crash atau unhandled panic (`fatal error`).
  - Request pengguna tetap terlayani dengan respons status HTTP 200/401/403 yang valid (zero downtime pada core billing & chat).
  - Tingkat kegagalan request pengguna akibat putusnya Redis adalah `0%`.
  - Begitu Redis kembali online, koneksi pool otomatis terhubung ulang (*auto-reconnect*) dalam waktu `<= 5 detik` tanpa membutuhkan restart service backend.
- **Lokasi Artefak Bukti:** `reports/chaos/redis-fallback-drill.log`, `reports/chaos/redis-fallback-summary.md`.
- **Catatan Zero-Downtime & Keamanan:** Data otoritatif selalu bertumpu pada PostgreSQL; Redis difungsikan murni sebagai akselerator performa.

---

### Wave 3: Security Audit

#### TASK-P08-006 — Audit Enkripsi Kredensial Database At-Rest (AES-256-GCM)
- **Target File:**
  - `backend/internal/pkg/crypto/encrypt.go` (implementasi cipher AES-256-GCM)
  - `backend/internal/pkg/crypto/encrypt_test.go` (pengujian unit enkripsi/dekripsi)
  - `backend/internal/aigateway/repository.go` (penyimpanan kredensial API endpoint)
  - `backend/internal/payment/repository.go` (penyimpanan kredensial server gateway)
  - `reports/security/encryption-at-rest-audit.md` (laporan hasil audit struktur data ciphertext)
  - `reports/security/gitleaks-scan.json` (bukti scan secret repositori)
- **Langkah Rencana:**
  1. Lakukan audit kode terhadap implementasi modul `crypto.EncryptSecret` dan `crypto.DecryptSecret`.
  2. Verifikasi bahwa ukuran kunci master tepat 32 byte (256 bit), nonce dibangkitkan dari `crypto/rand` berukuran 12 byte secara acak per operasi, dan authentication tag berukuran 16 byte.
  3. Lakukan query inspeksi langsung terhadap kolom database penampung secret (`ai_endpoints.api_key`, `payment_gateways.server_key`, dll).
  4. Jalankan perkakas pemindai rahasia otomatis (Gitleaks / TruffleHog) pada seluruh riwayat commit repositori Git untuk memastikan tidak ada kunci privat atau master key yang bocor.
- **Target Kriteria & Ambang Batas:**
  - 100% nilai rahasia di kolom target tersimpan dalam format base64 dari struktur `nonce (12 byte) + ciphertext + auth tag (16 byte)`.
  - Plaintext tidak dapat dibaca secara kasat mata melalui `psql` tanpa master key.
  - Variabel `APP_ENCRYPTION_MASTER_KEY` dibaca murni dari environment variable terisolasi.
  - Jumlah secret/kredensial yang terdeteksi bocor di riwayat git commit adalah tepat `0` (Zero Leak).
- **Lokasi Artefak Bukti:** `reports/security/encryption-at-rest-audit.md`, `reports/security/gitleaks-scan.json`.
- **Catatan Zero-Downtime & Keamanan:** Penanganan error dekripsi wajib mengembalikan `ErrDecryptionFailed` generic tanpa membocorkan detail error internal cryptography ke klien.

---

#### TASK-P08-007 — Validasi Kriptografis Webhook Signature (Midtrans SHA-512 & Xendit Token)
- **Target File:**
  - `backend/internal/payment/webhook_validator.go` (logika validasi HMAC & constant time compare)
  - `backend/internal/payment/webhook_validator_test.go` (test suite pengujian keamanan signature)
  - `backend/internal/audit/audit.go` (pencatatan insiden keamanan)
  - `reports/security/webhook-signature-validation-test.log` (log pengujian penetrasi)
  - `reports/security/webhook-tampering-audit.md` (laporan verifikasi keamanan)
- **Langkah Rencana:**
  1. Buat skenario pengujian penetrasi dengan memanipulasi payload webhook Midtrans (mengubah nominal transaksi `gross_amount` atau `order_id`) dengan tetap menyertakan signature asli.
  2. Buat skenario manipulasi signature acak dan signature kosong.
  3. Verifikasi implementasi validasi token Xendit menggunakan fungsi `subtle.ConstantTimeCompare` untuk mencegah celah *timing attack*.
  4. Verifikasi bahwa setiap kegagalan validasi tanda tangan digital memicu pembuatan record log audit keamanan dengan kategori `SECURITY_VIOLATION`.
- **Target Kriteria & Ambang Batas:**
  - 100% request webhook dengan signature yang dimanipulasi atau tidak sah ditolak dengan status HTTP 401 Unauthorized.
  - Tidak ada perbedaan waktu pemrosesan (*timing discrepancy*) yang dapat dieksploitasi pada pengecekan token (terbukti menggunakan `crypto/subtle`).
  - Setiap upaya payload tampering menghasilkan entri tercatat pada tabel `audit_logs` dengan severity `HIGH`.
- **Lokasi Artefak Bukti:** `reports/security/webhook-signature-validation-test.log`, `reports/security/webhook-tampering-audit.md`.
- **Catatan Zero-Downtime & Keamanan:** Penolakan signature palsu terjadi di layer terluar sebelum proses pembacaan atau mutasi data transaksi keuangan dijalankan.

---

#### TASK-P08-008 — Verifikasi Postur Kepatuhan PCI-DSS SAQ A & Keamanan Browser
- **Target File:**
  - `frontend/app/(checkout)/checkout/page.tsx` (antarmuka checkout)
  - `frontend/components/billing/PaymentModal.tsx` (modal pemanggil snap.js)
  - `backend/internal/delivery/http/middleware/security_headers.go` (middleware HTTP headers)
  - `backend/internal/delivery/http/middleware/logging.go` (sanitasi log)
  - `reports/security/pci-dss-saq-a-attestation.md` (dokumen asesmen kepatuhan)
  - `reports/security/security-headers-scan.json` (hasil inspeksi header respon)
- **Langkah Rencana:**
  1. Lakukan audit arsitektur frontend untuk memastikan implementasi pembayaran menggunakan Midtrans Snap Popup / Redirect atau Xendit Checkout Hosted Page secara penuh.
  2. Lakukan inspeksi menyeluruh terhadap kode backend, file log, dan database untuk memastikan tidak ada pemrosesan, transmisi, maupun penyimpanan data nomor kartu (PAN), CVV, atau tanggal kedaluwarsa kartu kredit (*zero cardholder data environment*).
  3. Konfigurasi middleware security headers pada Go backend dan Next.js:
     - `Content-Security-Policy`: membatasi eksekusi skrip murni ke origin sendiri dan domain payment gateway resmi (`app.midtrans.com`, `checkout.xendit.co`).
     - `Strict-Transport-Security`: `max-age=63072000; includeSubDomains; preload`.
     - `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, `Referrer-Policy: strict-origin-when-cross-origin`.
  4. Jalankan pemindaian otomatis terhadap header web staging menggunakan curl / scanner keamanan.
- **Target Kriteria & Ambang Batas:**
  - Kriteria PCI-DSS SAQ A terpenuhi 100% (FinTrack v2 memenuhi syarat pengisian instrumen SAQ A karena seluruh penanganan kartu didelegasikan ke entitas tervalidasi PCI-DSS Level 1).
  - Jumlah entri nomor kartu kredit (PAN) atau CVV yang ditemukan pada database dan log adalah tepat `0`.
  - 100% security header wajib terpasang aktif pada setiap respons HTTP.
- **Lokasi Artefak Bukti:** `reports/security/pci-dss-saq-a-attestation.md`, `reports/security/security-headers-scan.json`.
- **Catatan Zero-Downtime & Keamanan:** CSP dikonfigurasi secara presisi agar tidak memblokir render skrip analitik internal atau komponen visual Next.js.

---

### Wave 4: Compliance (PDP, Pajak, DR)

#### TASK-P08-009 — Implementasi & Pengujian Modul Anonimisasi Subjek Data UU PDP
- **Target File:**
  - `backend/internal/compliance/anonymizer.go` (layanan pemusnahan & anonimisasi akun)
  - `backend/internal/compliance/anonymizer_test.go` (pengujian alur pemusnahan data)
  - `backend/internal/delivery/http/handler/user_handler.go` (endpoint permintaan penutupan akun)
  - `reports/compliance/pdp-anonymization-test.log` (log pengujian penghapusan akun)
  - `reports/compliance/pdp-compliance-matrix.md` (matriks kesesuaian pasal UU PDP)
- **Langkah Rencana:**
  1. Rancang alur eksekusi fungsi `anonymize_user_data(userID)` di dalam transaksi atomik PostgreSQL:
     - Menyamarkan data PII pada tabel `users`: nama diganti `Anonymized User`, email diganti `deleted-<UUID>@anonymized.fintrack.id`, nomor telepon diganti hash acak, password hash dikosongkan.
     - Menghapus secara permanen (*hard delete*) seluruh data riwayat pesan AI pada tabel `chat_messages` dan `chat_sessions`.
     - Menghapus secara permanen seluruh representasi vektor pada tabel `transaction_embeddings` (pgvector 768-dimensi).
     - Menghancurkan kunci enkripsi personal pengguna (*cryptographic shredding*).
  2. Catat eksekusi proses anonimisasi ke dalam tabel `audit_logs` dengan kategori `PDP_DATA_ERASURE_FULFILLED`.
  3. Uji coba modul menggunakan akun tiruan dan periksa residu data pribadi di seluruh tabel sistem.
- **Target Kriteria & Ambang Batas:**
  - 100% kolom identitas pribadi (PII) pada tabel `users` untuk akun yang dihapus tersamarkan secara ireversibel.
  - Jumlah baris pesan chat dan embedding pgvector yang tersisa pasca-eksekusi untuk akun tersebut adalah tepat `0`.
  - Kunci enkripsi personal pengguna hancur tuntas (*cryptographically shredded*).
  - Proses pemusnahan tervalidasi memenuhi ketentuan Pasal 43 UU No. 27/2022.
- **Lokasi Artefak Bukti:** `reports/compliance/pdp-anonymization-test.log`, `reports/compliance/pdp-compliance-matrix.md`.
- **Catatan Zero-Downtime & Keamanan:** Seluruh operasi penghapusan dan pseudonimisasi berjalan dalam satu blok transaksi database terisolasi untuk mencegah kondisi inkonsistensi parsial.

---

#### TASK-P08-010 — Validasi Integritas Retensi Data Pajak 10 Tahun vs Anonimisasi
- **Target File:**
  - `backend/internal/compliance/tax_retention_validator.go` (modul verifikasi integritas pembukuan)
  - `backend/internal/compliance/tax_retention_validator_test.go` (skrip pengujian audit rekonsiliasi)
  - `backend/internal/billing/invoice_service.go` (penerbitan dan pengarsipan faktur)
  - `reports/compliance/tax-retention-reconciliation.json` (hasil data neraca pembukuan)
  - `reports/compliance/tax-audit-report.md` (laporan audit pemenuhan UU KUP)
- **Langkah Rencana:**
  1. Siapkan skenario pengujian di mana akun pengguna yang memiliki riwayat pembayaran langganan resmi meminta penghapusan akun (menjalankan `TASK-P08-009`).
  2. Lakukan audit struktur data pada tabel `invoices`, `payments`, dan `token_ledger` setelah proses anonimisasi selesai.
  3. Validasi bahwa record invoice tetap tersimpan dengan nomor faktur legal, rincian PPN, stempel waktu transaksi, dan nilai nominal Rupiah yang utuh.
  4. Jalankan prosedur rekonsiliasi keuangan menyeluruh: hitung total agregat pendapatan perusahaan sebelum dan sesudah anonimisasi pengguna.
- **Target Kriteria & Ambang Batas:**
  - 100% baris data pada tabel `invoices`, `payments`, dan `token_ledger` milik akun yang dihapus tetap tersimpan utuh selama masa retensi minimum 10 tahun (UU KUP No. 28/2007 Pasal 28).
  - Kolom nama dan alamat pada faktur menampilkan referensi entitas teranonimisasi yang sah tanpa merusak integritas nomor seri faktur.
  - Nilai selisih agregat total pendapatan perusahaan (*reconciliation discrepancy*) sebelum dan sesudah anonimisasi akun adalah tepat `Rp 0` (Zero Discrepancy).
- **Lokasi Artefak Bukti:** `reports/compliance/tax-retention-reconciliation.json`, `reports/compliance/tax-audit-report.md`.
- **Catatan Zero-Downtime & Keamanan:** Integritas relasi tabel dipertahankan dengan mekanisme nullable foreign key atau foreign key ke UUID anonim tanpa melanggar constraint integritas referensial database.

---

#### TASK-P08-011 — Disaster Recovery Drill (Pengujian Restorasi PITR PostgreSQL)
- **Target File:**
  - `scripts/dr/pitr_restore_drill.sh` (skrip otomasi simulasi bencana & restorasi)
  - `scripts/backup/daily_backup.sh` (skrip pencadangan berkala PostgreSQL)
  - `scripts/dr/verify_data_integrity.go` (skrip verifikasi checksum integritas pasca-restorasi)
  - `reports/dr/pitr-drill-execution.log` (catatan waktu dan proses pemulihan)
  - `reports/dr/pitr-rpo-rto-verification.md` (dokumen pembuktian pencapaian target RPO/RTO)
- **Langkah Rencana:**
  1. Konfigurasi continuous WAL archiving pada PostgreSQL 17 dengan interval pemotongan WAL maksimal 60 detik (`archive_timeout = 60`).
  2. Simulasikan skenario bencana: injeksikan transaksi keuangan aktif, catat stempel waktu tepat sebelum anomali buatan, lalu simulasikan kerusakan basis data (hentikan container dan hapus direktori data PG).
  3. Eksekusi prosedur pemulihan PITR ke lingkungan terisolasi menggunakan snapshot dasar harian dan pemutaran ulang arsip WAL hingga target waktu tepat sebelum kerusakan.
  4. Ukur durasi pemulihan aktual (RTO) dan selisih data transaksi yang hilang (RPO).
  5. Lakukan verifikasi checksum dan hitung total baris tabel billing.
- **Target Kriteria & Ambang Batas:**
  - Sasaran Pemulihan Data (*Recovery Point Objective* / RPO) tercapai dengan kehilangan data maksimal `<= 1 menit` untuk tier data finansial (Tier 1).
  - Sasaran Waktu Pemulihan (*Recovery Time Objective* / RTO) tercapai dengan durasi henti layanan total `<= 15 menit` hingga database pulih dan siap melayani trafik.
  - Verifikasi checksum integritas relasional database menunjukkan status 100% konsisten tanpa tabel atau index yang korup.
- **Lokasi Artefak Bukti:** `reports/dr/pitr-drill-execution.log`, `reports/dr/pitr-rpo-rto-verification.md`.
- **Catatan Zero-Downtime & Keamanan:** Pengujian pemulihan dieksekusi pada instans sandbox DR terisolasi; dilarang keras menimpa kluster database staging aktif secara langsung.

---

### Wave 5: Observability & Go-Live

#### TASK-P08-012 — Uji Latih Runbook Operasional (Rotasi Kunci & Invalidasi Cache Basi)
- **Target File:**
  - `scripts/runbook/rotate_master_key.go` (skrip rotasi kunci enkripsi data at-rest)
  - `scripts/runbook/flush_stale_cache.sh` (skrip invalidasi cache darurat lintas pod)
  - `backend/internal/infra/redis/pubsub.go` (penanganan invalidasi via Redis Pub/Sub)
  - `reports/runbook/key-rotation-drill.log` (log eksekusi rotasi kunci)
  - `reports/runbook/cache-invalidation-drill.log` (log pengujian invalidasi cache)
- **Langkah Rencana:**
  1. Simulasikan prosedur operasional rotasi master encryption key: sediakan kunci lama dan kunci baru melalui konfigurasi dual-key, dekripsi kolom terenkripsi dengan kunci lama, lalu re-enkripsi secara bertahap menggunakan kunci baru.
  2. Pantau ketersediaan aplikasi selama proses re-enkripsi berjalan untuk membuktikan tidak adanya gangguan layanan ke pengguna.
  3. Lakukan simulasi perubahan hak akses/entitlement pengguna melalui modul Super Admin, lalu amati transmisi pesan invalidasi cache via channel Redis Pub/Sub `cache:invalidate`.
  4. Ukur kecepatan propagasi pembersihan cache lokal pada seluruh pod backend yang sedang aktif.
- **Target Kriteria & Ambang Batas:**
  - Rotasi master key berjalan sukses 100% tanpa menimbulkan downtime aplikasi (Zero Downtime) dan seluruh ciphertext berhasil didekripsi dengan kunci baru.
  - Invalidasi cache hak pengguna tersinkronisasi ke seluruh pod backend dalam waktu `<= 50ms` via Redis Pub/Sub tanpa membutuhkan restart container backend.
  - Tidak ada data basi (*stale cache*) yang tersisa pada pengujian verifikasi akses lanjutan.
- **Lokasi Artefak Bukti:** `reports/runbook/key-rotation-drill.log`, `reports/runbook/cache-invalidation-drill.log`.
- **Catatan Zero-Downtime & Keamanan:** Skrip rotasi kunci beroperasi dengan mekanisme commit per batch kecil untuk mencegah table lock berkepanjangan pada PostgreSQL.

---

#### TASK-P08-013 — Konfigurasi Dasbor Observability Four Golden Signals & Alerting
- **Target File:**
  - `monitoring/dashboards/fintrack-golden-signals.json` (definisi dashboard Grafana Golden Signals)
  - `monitoring/dashboards/fintrack-business-metrics.json` (definisi dashboard metrik bisnis)
  - `monitoring/prometheus/alert_rules.yml` (aturan peringatan ambang batas Prometheus)
  - `backend/internal/delivery/http/middleware/metrics.go` (middleware instrumentasi metrik Prometheus)
  - `reports/observability/alert-rules-validation.md` (resume verifikasi notifikasi alerting)
- **Langkah Rencana:**
  1. Pasang konfigurasi visualisasi dashboard Grafana yang mencakup Four Golden Signals (Google SRE):
     - **Latensi**: Histogram latensi p50, p95, dan p99 per rute API.
     - **Trafik**: Tingkat request per detik (RPS) total dan per endpoint.
     - **Error**: Rasio error HTTP 4xx dan 5xx terhadap total request.
     - **Saturasi**: Utilisasi CPU, memori, pool koneksi database PostgreSQL (`active / max`), dan pool Redis.
  2. Pasang panel metrik bisnis spesifik: jumlah langganan aktif, AI token burn rate per menit, rasio keberhasilan webhook pembayaran.
  3. Konfigurasi aturan alerting Prometheus:
     - Alert P1 Kritis: `HTTP5xxRate > 1%` selama 2 menit, `DBPoolSaturation > 85%` selama 3 menit, `AIAllEndpointsDown == 1`.
     - Alert P2 Tinggi: `WebhookFailureRate > 5%`, `P95Latency > 200ms`.
  4. Lakukan pengujian pemicuan alert tiruan untuk memastikan notifikasi terkirim tepat sasaran ke channel operasional SRE.
- **Target Kriteria & Ambang Batas:**
  - Seluruh panel Four Golden Signals dan metrik bisnis terpasang dan menampilkan visualisasi data real-time yang akurat.
  - Aturan alerting tervalidasi aktif (`state: firing` saat kondisi anomali dipicu dalam uji coba).
  - Waktu transmisi notifikasi alert dari Prometheus ke channel komunikasi operasional berada dalam tempo `<= 30 detik`.
- **Lokasi Artefak Bukti:** `monitoring/dashboards/fintrack-golden-signals.json`, `monitoring/prometheus/alert_rules.yml`, `reports/observability/alert-rules-validation.md`.
- **Catatan Zero-Downtime & Keamanan:** Scraping metrik pada endpoint `/metrics` diamankan dengan autentikasi jaringan internal dan tidak dibuka ke internet publik.

---

#### TASK-P08-014 — Verifikasi Menyeluruh & Penandatanganan Go-Live Checklist
- **Target File:**
  - `second_brain/2_Projects/P08_Hardening-GoLive/teknis/go-live-checklist.md` (checklist verifikasi kesiapan rilis)
  - `second_brain/2_Projects/P08_Hardening-GoLive/tracking/tracking.md` (catatan status tracking final)
  - `reports/golive/signed-go-live-checklist.md` (salinan checklist tertandatangani resmi)
  - `reports/golive/operational-handover-signoff.md` (berita acara serah terima operasional)
- **Langkah Rencana:**
  1. Selenggarakan sidang audit kesiapan rilis (*Go-Live Readiness Review Meeting*) bersama tim arsitek, lead engineer, DBA, analis keamanan, dan compliance officer.
  2. Evaluasi bukti fisik dan laporan artefak dari seluruh 13 task pengujian sebelumnya terhadap gerbang verifikasi kategori A s/d F pada `teknis/go-live-checklist.md`.
  3. Bubuhkan konfirmasi validasi formal pada setiap baris item verifikasi.
  4. Terbitkan dokumen berita acara serah terima operasional ke tim pemeliharaan sistem (Day-2 Operations).
  5. Nyatakan otorisasi pembukaan gerbang rilis produksi publik FinTrack v2.
- **Target Kriteria & Ambang Batas:**
  - 100% item verifikasi pada lembar checklist (Kategori A Finansial, B Keamanan, C Resiliensi, D Kepatuhan, E DR, F Operasional) memperoleh status **PASSED** dengan bukti fisik yang sah.
  - Tidak ada tiket cacat kritis (*blocker / P1 / P2 bug*) yang tersisa dalam kondisi terbuka.
  - Seluruh pimpinan tim teknis (Tech Lead, Security Lead, DBA, SRE Lead, dan Compliance Officer) membubuhkan tanda tangan persetujuan resmi.
- **Lokasi Artefak Bukti:** `reports/golive/signed-go-live-checklist.md`, `reports/golive/operational-handover-signoff.md`.
- **Catatan Zero-Downtime & Keamanan:** Pembukaan akses publik dilakukan melalui prosedur DNS cutover / routing proxy bertahap dengan kemampuan rollback instan bila terdeteksi anomali pasca-rilis.

---

## 4. Matriks Bukti (Task vs Artefak)

Tabel berikut menetapkan artefak bukti wajib yang harus diproduksi dan diverifikasi untuk setiap task sebelum task dinyatakan selesai:

| Task ID | Nama Task | Tipe Bukti | Lokasi Path Artefak Wajib |
|---|---|---|---|
| `TASK-P08-001` | k6 Race Condition Kuota Token | Log & Metrik JSON | `reports/loadtest/token-quota-concurrency-result.json`<br>`reports/loadtest/token-quota-concurrency-summary.md` |
| `TASK-P08-002` | k6 Replay Webhook Pembayaran | Log & Metrik JSON | `reports/loadtest/webhook-replay-result.json`<br>`reports/loadtest/webhook-replay-summary.md` |
| `TASK-P08-003` | Benchmark Sustained AI Gateway | Log & Metrik JSON | `reports/loadtest/ai-gateway-sustained-result.json`<br>`reports/loadtest/ai-gateway-sustained-summary.md` |
| `TASK-P08-004` | Chaos Failover 9router | Log Drill & Metrik | `reports/chaos/9router-failover-drill.json`<br>`reports/chaos/9router-failover-summary.md` |
| `TASK-P08-005` | Chaos Redis Fallback | Log Sistem & Analisis | `reports/chaos/redis-fallback-drill.log`<br>`reports/chaos/redis-fallback-summary.md` |
| `TASK-P08-006` | Audit Enkripsi At-Rest AES-GCM | Laporan Audit Kriptografi | `reports/security/encryption-at-rest-audit.md`<br>`reports/security/gitleaks-scan.json` |
| `TASK-P08-007` | Validasi Signature Webhook | Log Uji Penetrasi & Audit | `reports/security/webhook-signature-validation-test.log`<br>`reports/security/webhook-tampering-audit.md` |
| `TASK-P08-008` | Verifikasi PCI-DSS SAQ A | Asesmen Kepatuhan & Scan | `reports/security/pci-dss-saq-a-attestation.md`<br>`reports/security/security-headers-scan.json` |
| `TASK-P08-009` | Modul Anonimisasi UU PDP | Log Pemusnahan Data | `reports/compliance/pdp-anonymization-test.log`<br>`reports/compliance/pdp-compliance-matrix.md` |
| `TASK-P08-010` | Integritas Retensi Pajak 10 Thn | Rekonsiliasi Neraca JSON | `reports/compliance/tax-retention-reconciliation.json`<br>`reports/compliance/tax-audit-report.md` |
| `TASK-P08-011` | Disaster Recovery Drill PITR | Log Eksekusi & Metrik RPO/RTO | `reports/dr/pitr-drill-execution.log`<br>`reports/dr/pitr-rpo-rto-verification.md` |
| `TASK-P08-012` | Uji Runbook Rotasi & Cache | Log Eksekusi Runbook | `reports/runbook/key-rotation-drill.log`<br>`reports/runbook/cache-invalidation-drill.log` |
| `TASK-P08-013` | Dasbor Golden Signals & Alerts | Konfigurasi JSON & YAML | `monitoring/dashboards/fintrack-golden-signals.json`<br>`monitoring/prometheus/alert_rules.yml`<br>`reports/observability/alert-rules-validation.md` |
| `TASK-P08-014` | Checklist Go-Live Final | Berita Acara Ditandatangani | `reports/golive/signed-go-live-checklist.md`<br>`reports/golive/operational-handover-signoff.md` |

---

## 5. Perintah Verifikasi

Perintah-perintah berikut disiapkan untuk memvalidasi seluruh rangkaian komponen tanpa toleransi kegagalan:

### 5.1 Integritas Kompilasi & Kode Bersih
```bash
# Verifikasi backend Go
cd /d D:/project/finance/fintrack_v2/backend
go vet ./...
go test -v -race ./...
go build -o /dev/null ./cmd/server

# Verifikasi frontend Next.js 16
cd /d D:/project/finance/fintrack_v2/frontend
pnpm lint
pnpm test
pnpm build
```

### 5.2 Skema Basis Data & Migrasi Replayable
```bash
# Validasi konsistensi migrasi database
cd /d D:/project/finance/fintrack_v2
# Pastikan migrasi 0001 s/d 0010 dapat diulang secara deterministik
docker compose exec backend ./fintrack-migrate up
docker compose exec db pg_dump -U postgres -d fintrack_v2 --schema-only > database/dump/schema.sql
```

### 5.3 Eksekusi Pengujian Beban & Konkurensi (k6)
```bash
# Pengujian race condition kuota token (20 VU vs 5 token)
k6 run --out json=reports/loadtest/token-quota-concurrency-result.json \
  scripts/loadtest/token_quota_concurrency.js

# Pengujian replay webhook (10x request konkuren < 50ms)
k6 run --out json=reports/loadtest/webhook-replay-result.json \
  scripts/loadtest/webhook_replay.js

# Benchmark sustained AI Gateway 500 RPS (10 menit)
k6 run --out json=reports/loadtest/ai-gateway-sustained-result.json \
  scripts/loadtest/ai_gateway_sustained.js
```

### 5.4 Eksekusi Pengujian Chaos & Resiliensi
```bash
# Uji chaos failover node 9router
bash scripts/chaos/chaos_9router_failover.sh

# Uji chaos pemutusan jaringan Redis
bash scripts/chaos/chaos_redis_disconnect.sh
```

### 5.5 Eksekusi Audit Keamanan & Kepatuhan
```bash
# Pindai kebocoran kredensial di git
gitleaks detect --source=. --report-path=reports/security/gitleaks-scan.json

# Uji unit cipher AES-256-GCM & signature validator
cd /d D:/project/finance/fintrack_v2/backend
go test -v -run TestEncryptSecret ./internal/pkg/crypto/...
go test -v -run TestWebhookSignature ./internal/payment/...

# Uji kepatuhan anonimisasi UU PDP & integritas pajak
go test -v -run TestAnonymizeUserData ./internal/compliance/...
go test -v -run TestTaxRetentionReconciliation ./internal/compliance/...
```

### 5.6 Eksekusi Latih Pemulihan Bencana (PITR Drill)
```bash
# Eksekusi simulasi pemulihan PITR ke database sandbox
bash scripts/dr/pitr_restore_drill.sh
```

---

## 6. Prosedur Rollback

Jika saat pengujian hardening atau saat pembukaan gerbang rilis produksi ditemukan anomali berbobot P1 (kritis), tim wajib mengeksekusi rencana kontinjensi berikut:

### 6.1 Anomali Beban & Concurrency Finansial (W1)
- **Kondisi:** Terjadi saldo negatif pada tabel `token_periods` atau duplikasi transaksi invoice pada uji beban k6.
- **Tindakan Rollback:**
  1. Hentikan eksekusi pengujian beban seketika (`Ctrl+C` / kill process k6).
  2. Kembalikan state database pengujian menggunakan database transaction rollback atau restore dump staging terakhir.
  3. Lakukan isolasi ke modul Go `token_service.go` atau `webhook_handler.go` untuk memperketat locking query (`SELECT ... FOR UPDATE`) dan memastikan constraint database `UNIQUE(idempotency_key)` aktif.

### 6.2 Kegagalan Resiliensi & Panic Sistem (W2)
- **Kondisi:** Backend mengalami crash fatal/panic saat Redis dimatikan, atau AI Gateway menghasilkan lonjakan HTTP 5xx saat 9router primer mati.
- **Tindakan Rollback:**
  1. Hidupkan kembali container dependensi (`docker start fintrack_redis`, `docker start fintrack-9router-primary`).
  2. Perbaiki fallback logic di `cache.go` dan `breaker.go` dengan memastikan pointer error nil diperiksa secara defensif sebelum digunakan.
  3. Re-run uji chaos lokal hingga terbukti *zero panic*.

### 6.3 Kegagalan Validasi Kepatuhan & PCI-DSS (W3 / W4)
- **Kondisi:** Ditemukan pencatatan PAN/CVV pada log aplikasi atau proses anonimisasi merusak keseimbangan neraca pembukuan pajak.
- **Tindakan Rollback:**
  1. Sanitasi log darurat: jalankan skrip pembersih file log untuk menghapus jejak data sensitif.
  2. Rollback skrip modul anonimisasi; pulihkan snapshot data finansial dari backup cold harian.
  3. Perbaiki klausul pseudonimisasi agar kolom finansial pada `invoices` tidak terhapus atau berubah nilai nominalnya.

### 6.4 Abort Go-Live Produksi (W5)
- **Kondisi:** Ditemukan ketidaksesuaian kritis pada salah satu item verifikasi Kategori A s/d E saat verifikasi akhir checklist.
- **Tindakan Rollback:**
  1. Status rilis seketika diturunkan menjadi **ABORTED / POSTPONED**.
  2. Pembukaan domain publik dibatalkan pada ingress/reverse proxy (tetap arahkan trafik ke maintenance page resmi FinTrack).
  3. Tim engineering menyusun perbaikan darurat dan menjadwalkan ulang audit verifikasi minimal 24 jam berikutnya.

---

## 7. Handoff ke Operasional (Day-2 Operations & GA)

Karena P08 merupakan **fase final** dari seluruh siklus implementasi platform FinTrack v2, alur serah terima dialihkan langsung ke Tim Operasional Berkelanjutan (*Day-2 Operations / SRE / Business Operations*):

1. **Penyerahan Kredensial Produksi & Master Keys:**
   - Kunci enkripsi master `APP_ENCRYPTION_MASTER_KEY` dan kredensial live payment gateway diserahterimakan secara aman melalui vault manajer rahasia institusional (bukan via chat/dokumen terbuka).
2. **Aktivasi Jadwal Operasional Rutin (Crontab & Automation):**
   - Pencadangan basis data snapshot penuh terenkripsi setiap hari pukul 02:00 WIB (`scripts/backup/daily_backup.sh`).
   - Uji coba integritas pemulihan data berkala setiap hari Minggu pukul 04:00 WIB ke database sandbox otomatis.
   - Rotasi dan kompresi file audit log setiap 30 hari.
3. **Pemberian Akses Dashboard & PagerDuty Alerting:**
   - Hak akses dasbor Grafana diserahkan ke tim SRE on-call dengan rotasi jadwal jaga 24/7.
   - Nomor eskalasi darurat SRE terhubung langsung ke Webhook notifikasi Alertmanager.
4. **Pemberian Akses Portal Super Admin:**
   - Kredensial akun role `super_admin` pertama diserahkan kepada pimpinan operasional bisnis untuk pengelolaan paket langganan, kuota token, dan pemantauan pembayaran real-time.
5. **Deklarasi Kesiapan Publik (GA Status):**
   - Sistem FinTrack v2 resmi dinyatakan berstatus **General Availability (GA)** dan siap menerima pendaftaran pengguna publik secara global.

---

## 8. Referensi

- `README.md` — Ringkasan Fase P08 Hardening & Go-Live
- `planning/tasks.md` — Work Breakdown Structure (TASK-P08-001 s/d TASK-P08-014)
- `planning/timeline.md` — Timeline, Matriks Ketergantungan, dan Analisis Jalur Kritis
- `planning/sprint-plan.md` — Alokasi Kapasitas & Jadwal Sprint Hardening
- `rules/implementation-rules.md` — Aturan Implementasi Mengikat (Kriptografi, Finansial, Regulasi)
- `teknis/load-test.md` — Spesifikasi Pengujian Beban k6 & Hasil Pengujian Baseline
- `teknis/chaos-failover.md` — Metodologi Chaos Engineering, Failover, & Circuit Breaker
- `teknis/security-review.md` — Audit Keamanan AES-256-GCM, HMAC Webhook, & PCI-DSS SAQ A
- `teknis/pdp-compliance.md` — Kepatuhan Regulasi UU PDP No. 27/2022 & Retensi Pajak 10 Tahun
- `teknis/backup-dr.md` — Kebijakan Backup, PITR WAL Archiving, & Metrik RPO/RTO
- `teknis/runbook-incidents.md` — Panduan Prosedur Penanganan Insiden Operasional Produksi
- `teknis/go-live-checklist.md` — Lembar Verifikasi Kesiapan Go-Live (Kategori A s/d F)
- `../../IMPLEMENTATION-PLAN.md` — Master Implementation Plan FinTrack v2
