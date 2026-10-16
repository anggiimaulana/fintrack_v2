# Sprint Plan - Fase P08 (Hardening & Go-Live)

## 1. Ikhtisar Sprint
- **Nama Sprint**: P08 Hardening, Resilience, & Production Go-Live
- **Durasi**: 2 Minggu (10 Hari Kerja)
- **Tujuan Utama**: Memverifikasi secara tuntas seluruh kriteria performa, integritas finansial, kehandalan arsitektur failover, audit keamanan siber, dan kepatuhan hukum UU PDP No. 27/2022 FinTrack v2 sebelum membuka akses publik.
- **Prinsip**: Strict zero-downtime, zero-leakage, zero-unhandled-failover.

---

## 2. Struktur Tim & Pembagian Peran
1. **Tech Lead / SRE Lead**:
   - Memimpin orkestrasi skenario chaos engineering, load testing k6 skala tinggi, dan simulasi Disaster Recovery.
   - Mengawal pipeline release, zero-downtime rolling update, dan observability Golden Signals.
2. **Backend & Security Engineer**:
   - Menulis skrip k6 race condition kuota token dan replay webhook.
   - Melakukan peninjauan enkripsi AES-256-GCM, rotasi kredensial, verifikasi HMAC, dan audit OWASP API Security.
   - Mengimplementasikan alur anonimisasi subjek data UU PDP vs retensi pajak 10 tahun.
3. **Database Administrator (DBA) & Compliance Officer**:
   - Menguji WAL archiving PostgreSQL, backup PITR, simulasi RTO/RPO tier 1.
   - Mengaudit kesesuaian kebijakan retensi data, DPA (Data Processing Agreement) vendor, dan checklist PCI-DSS SAQ A.

---

## 3. Rencana Kerja Harian (Day-by-Day Sprint Cadence)

### Hari 1-2: Setup Baseline & Concurrency Load Testing
- Penyusunan skrip k6 untuk pengujian konkurensi kuota token AI (20 request paralel vs 5 sisa kuota).
- Pengujian idempotensi webhook (10x injeksi payload identik secara simultan).
- Eksekusi sustained load test 500 RPS pada backend dan AI Gateway.
- Verifikasi integritas tabel `token_ledger`, `payments`, dan ketiadaan race condition.

### Hari 3-4: Chaos Engineering & Failover Simulation
- Simulasi pemutusan koneksi salah satu endpoint pool 9router (port 20128) di bawah beban kerja aktif.
- Pengujian transisi state Circuit Breaker (Closed -> Open -> Half-Open).
- Simulasi Redis down / restart mendadak dan validasi in-memory cache fallback.
- Pengujian mitigasi database pool saturation under extreme load.

### Hari 5-6: Security Audit, Hardening, & PCI-DSS SAQ A Verification
- Audit enkripsi data sensitif at-rest (AES-256-GCM) pada tabel kredensial.
- Validasi mekanisme verifikasi webhook signature Midtrans (SHA-512) dan Xendit (constant-time token).
- Review konfigurasi CSP, CORS, cookie security (`HttpOnly`, `SameSite=Strict`, `Secure`), dan least privilege role database `fintrack_app`.
- Penyusunan dokumentasi kepatuhan PCI-DSS SAQ A (pembuktian nol kontak cardholder data).

### Hari 7: Kepatuhan UU PDP No. 27/2022 & Alur Anonimisasi Pajak
- Pengujian fungsional modul Right to Erasure / Account Deletion.
- Validasi pembersihan tuntas data PII, vector embedding, dan riwayat obrolan AI.
- Verifikasi integritas tabel `invoices` dan `payments` pasca anonimisasi: nomor faktur, nominal rupiah, rincian PPN, dan ledger pembukuan 10 tahun tetap konsisten.

### Hari 8: Simulasi Disaster Recovery & Uji Latih Insiden (Fire Drill)
- Uji coba pemulihan bencana (Disaster Recovery Drill) dengan PostgreSQL PITR.
- Pengukuran empiris RPO (harus <= 1 menit) dan RTO (harus <= 15 menit).
- Simulasi insiden operasional runbook: rotasi secret master key tanpa downtime dan pembersihan cache basi via Redis Pub/Sub.

### Hari 9: Observability Final & Monitoring Calibration
- Kalibrasi dashboard Grafana Four Golden Signals (Latency, Traffic, Errors, Saturation).
- Pengujian alert Slack/PagerDuty untuk metrik kritis: AI Gateway circuit breaker OPEN, payment webhook failure spike, kuota database pool > 80%.
- Verifikasi audit log pencatatan aksi super admin dan peristiwa keamanan.

### Hari 10: Tinjauan Checklist Go-Live Final & Rilis Produksi
- Eksekusi menyeluruh terhadap seluruh gerbang verifikasi `go-live-checklist.md`.
- Penandatanganan persetujuan rilis oleh Lead Developer, Security Officer, dan Product Owner.
- Eksekusi deployment rolling update dan pembukaan akses publik.

---

## 4. Manajemen Risiko Sprint

| Risiko | Dampak | Probabilitas | Mitigasi |
|---|---|---|---|
| Race condition kuota token bocor saat load test k6 | Tinggi | Sedang | Terapkan transaksi PostgreSQL `SELECT FOR UPDATE` pada baris periode token aktif sebelum memanggil inferensi AI Gateway. |
| Latensi failover 9router melampaui SLA (>500ms) | Sedang | Rendah | Konfigurasi timeout health check agresif (connect timeout 1s, fail threshold 3x berturut-turut). |
| Data invoice rusak saat eksekusi anonimisasi UU PDP | Kritis | Rendah | Pisahkan tabel PII pengguna dari tabel penagihan finansial; hanya anonymize kolom profil, jangan lakukan CASCADE DELETE ke invoice. |
| Replikasi WAL PostgreSQL gagal memenuhi RPO 1 menit | Kritis | Rendah | Gunakan `archive_timeout = 60s` dan verifikasi koneksi storage remote sebelum simulasi DR. |
