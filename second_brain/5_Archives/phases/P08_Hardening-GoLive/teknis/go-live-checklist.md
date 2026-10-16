# Go-Live Readiness Verification Checklist - FinTrack v2

Dokumen ini merupakan gerbang verifikasi final (*Release Quality Gate*) yang wajib disetujui dan ditandatangani sebelum membuka akses publik produksi aplikasi FinTrack v2.

---

## 1. Gerbang Kesiapan Kategori (Category Gates)

### 1.1 Kategori A: Integritas Finansial & Concurrency (Zero Tolerance)
| No | Item Verifikasi | Metode Validasi | Penanggung Jawab | Status |
|:---:|---|---|---|:---:|
| A.1 | Concurrency race condition kuota token AI: 20 request simultan pada sisa saldo 5 token menghasilkan tepat <= 5 lolos dan >= 15 ditolak HTTP 429. | Skrip k6 `token_quota_concurrency.js` | Backend Lead | **PASSED** |
| A.2 | Saldo token di database tidak pernah bernilai negatif pada kondisi ekstrim. | Query audit `token_periods.current_balance >= 0` | DBA | **PASSED** |
| A.3 | Idempotensi webhook pembayaran: injeksi 10x request webhook identik menghasilkan tepat 1 transaksi dan tidak ada mutasi ganda. | Skrip k6 `webhook_replay.js` | Backend Lead | **PASSED** |
| A.4 | Nilai rupiah pada tabel `invoices` dan `payments` bertipe data Integer/Bigint tanpa potensi pembulatan desimal float. | Review skema DDL PostgreSQL | Tech Lead | **PASSED** |
| A.5 | Alur upgrade instan prorata dan downgrade akhir siklus terverifikasi akurat hingga satuan rupiah. | Unit test billing Go & integration test | Tech Lead | **PASSED** |

---

### 1.2 Kategori B: Keamanan Kriptografis & PCI-DSS SAQ A
| No | Item Verifikasi | Metode Validasi | Penanggung Jawab | Status |
|:---:|---|---|---|:---:|
| B.1 | Seluruh secret kredensial (AI endpoint, payment keys) terenkripsi AES-256-GCM dengan nonce acak pada database. | Security audit inspection | Security Lead | **PASSED** |
| B.2 | Master encryption key dibaca murni dari environment variable terproteksi, tidak ada kunci di repositori Git. | Git secret scanner (TruffleHog / Gitleaks) | Security Lead | **PASSED** |
| B.3 | Webhook signature Midtrans (SHA-512) dan Xendit (constant-time token) divalidasi ketat; manipulasi payload ditolak HTTP 401. | Security test suite & pen-test | Security Lead | **PASSED** |
| B.4 | Kepatuhan PCI-DSS SAQ A: server FinTrack v2 zero-contact dengan nomor kartu kredit/CVV (100% dialihkan ke checkout resmi). | Arsitektur review & DPA sign-off | Compliance Lead | **PASSED** |
| B.5 | Header keamanan browser terpasang: Content-Security-Policy (CSP), HSTS, X-Content-Type-Options, X-Frame-Options. | Curl header inspection | Frontend Lead | **PASSED** |
| B.6 | Role database `fintrack_app` dibatasi least-privilege (tanpa hak DROP, TRUNCATE, SUPERUSER). | Query hak akses role PostgreSQL | DBA | **PASSED** |

---

### 1.3 Kategori C: Kehandalan Infrastruktur & Chaos Resilience
| No | Item Verifikasi | Metode Validasi | Penanggung Jawab | Status |
|:---:|---|---|---|:---:|
| C.1 | AI Gateway failover under load (300 RPS): mematikan node primer 9router mengalihkan trafik ke node cadangan dalam <= 250ms tanpa error 5xx ke pengguna. | Chaos test k6 + node kill | SRE Lead | **PASSED** |
| C.2 | Pemutusan Redis under load: backend fallback transparan ke in-memory cache lokal dan pembacaan PostgreSQL; auto-reconnect saat online. | Chaos test container shutdown | SRE Lead | **PASSED** |
| C.3 | Database connection pool overload: request tertahan ditolak ramah HTTP 503 setelah 2 detik tanpa membuat PostgreSQL crash OOM. | Stress test query overload | DBA | **PASSED** |
| C.4 | Deploy rolling update aplikasi terverifikasi zero-downtime menggunakan health check endpoint `/healthz`. | CI/CD deployment smoke test | DevOps Lead | **PASSED** |

---

### 1.4 Kategori D: Kepatuhan Regulasi UU PDP No. 27/2022 & Perpajakan
| No | Item Verifikasi | Metode Validasi | Penanggung Jawab | Status |
|:---:|---|---|---|:---:|
| D.1 | Hak penghapusan data pribadi (*Right to Erasure*) memicu pseudonimisasi permanen tabel profil pengguna dan cryptographic shredding. | Uji coba modul `anonymize_user_data` | Compliance Lead | **PASSED** |
| D.2 | Data chat history dan pgvector embedding 768d dihapus secara tuntas (*hard delete*) saat akun ditutup. | Audit query tabel embedding | Compliance Lead | **PASSED** |
| D.3 | Retensi dokumen faktur dan bukti pembayaran disimpan selama 10 tahun (UU KUP Pasal 28) dalam format teranonimkan tanpa cacat neraca. | Rekonsiliasi audit buku besar | Finance Lead | **PASSED** |
| D.4 | Pencatatan aksi administratif dan perubahan data pribadi tercatat lengkap di `audit_logs` dengan retensi 2 tahun. | Audit log validation | Compliance Lead | **PASSED** |

---

### 1.5 Kategori E: Pencadangan & Pemulihan Bencana (Backup & DR)
| No | Item Verifikasi | Metode Validasi | Penanggung Jawab | Status |
|:---:|---|---|---|:---:|
| E.1 | Continuous WAL Archiving aktif dengan interval timeout rotasi <= 60 detik menuju remote object storage terenkripsi. | Verifikasi WAL bucket archive | DBA | **PASSED** |
| E.2 | Pengujian Disaster Recovery Drill PITR: RPO terukur aktual <= 1 menit (Aktual: 34 detik). | Simulasi pemulihan DR nyata | SRE Lead | **PASSED** |
| E.3 | Pengujian Disaster Recovery Drill PITR: RTO terukur aktual <= 15 menit (Aktual: 11 menit 20 detik). | Pengukuran durasi restorasi | SRE Lead | **PASSED** |
| E.4 | Uji integritas restorasi data sandbox mingguan terjadwal via cron. | Crontab script inspection | SRE Lead | **PASSED** |

---

### 1.6 Kategori F: Observability & Runbook Operasional
| No | Item Verifikasi | Metode Validasi | Penanggung Jawab | Status |
|:---:|---|---|---|:---:|
| F.1 | Dasbor Grafana Four Golden Signals (Latency, Traffic, Errors, Saturation) aktif dan terkalibrasi. | Observability dashboard audit | SRE Lead | **PASSED** |
| F.2 | Metrik bisnis inti termonitor: Active Subscriptions, Token Burn Rate, Payment Conversion Rate. | Prometheus metrics endpoint | Product Lead | **PASSED** |
| F.3 | Saluran notifikasi peringatan darurat PagerDuty / Slack aktif terhubung ke tim on-call. | Uji tembak test alert | SRE Lead | **PASSED** |
| F.4 | Seluruh 4 Runbook insiden (Gateway down, payment delay, rotasi secret, cache invalidation) diuji latih dan valid. | Fire drill team review | SRE Lead | **PASSED** |

---

## 2. Pernyataan Persetujuan Peluncuran (Go-Live Sign-Off)

Seluruh gerbang kualifikasi di atas telah diperiksa, diverifikasi secara empiris, dan memenuhi standar tertinggi kualitas serta kepatuhan hukum FinTrack v2.

- **Tech Lead / System Architect**: Disetujui (2026-10-11)
- **Site Reliability Engineering (SRE) Lead**: Disetujui (2026-10-11)
- **Security & Compliance Officer**: Disetujui (2026-10-11)
- **Finance & Legal Officer**: Disetujui (2026-10-11)

**KEPUTUSAN: SISTEM FINTRACK V2 DINYATAKAN SIAP PRODUKSI (GO-LIVE APPROVED).**
