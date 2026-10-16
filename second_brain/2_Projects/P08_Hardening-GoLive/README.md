# Fase P08: Hardening & Go-Live

## Konteks & Deskripsi Proyek
Fase P08 merupakan fase penutupan dan validasi operasional skala penuh sebelum rilis publik (Go-Live) sistem FinTrack v2. Fokus utama fase ini adalah pembuktian empiris atas seluruh jaminan performa, ketahanan (resilience), integritas finansial, keamanan siber, dan kepatuhan regulasi perlindungan data pribadi (UU PDP No. 27/2022).

FinTrack v2 mengelola data transaksi keuangan sensitif, model langganan berbayar (B2C), billing token AI (chat/vision/voice/embedding), integrasi payment gateway Midtrans/Xendit, serta routing AI gateway via pool multi-endpoint 9router. Seluruh arsitektur harus tahan uji terhadap lonjakan beban ekstrim, kegagalan infrastruktur tiba-tiba, serangan siber, serta insiden operasional riil di lingkungan produksi.

## Ruang Lingkup (Scope)
1. **Load & Concurrency Testing (k6)**:
   - Pengujian batas kuota paralel (*race condition attack*): 20 request konkuren dengan sisa kuota 5 token harus menghasilkan tepat <= 5 request lolos dan >= 15 request ditolak HTTP 429.
   - Idempotensi webhook pembayaran: injeksi 10x request webhook identik secara paralel harus menghasilkan tepat 1 transaksi tercatat dan 1 mutasi saldo.
   - AI Gateway failover under load: simulasi terminasi instan node aktif 9router saat 500 RPS tanpa kegagalan request yang terekspos ke pengguna.
2. **Security & Vulnerability Assessment**:
   - Enkripsi data saat istirahat (*at rest*) via AES-256-GCM untuk kredensial gateway dan token pembayaran.
   - Rotasi secret zero-downtime, prinsip hak akses minimum (*least privilege*) role PostgreSQL & IAM.
   - Verifikasi tanda tangan digital webhook (HMAC-SHA512/SHA256).
   - Penilaian postur kepatuhan PCI-DSS SAQ A (karena FinTrack v2 menggunakan redirect/tokenization UI pihak ketiga dan tidak menyentuh data mentah kartu kredit).
   - Mitigasi OWASP Top 10 API & Anti-fraud transaksi finansial.
3. **Chaos Engineering & Resilience**:
   - Pengujian circuit breaker dan failover multi-endpoint 9router (default `http://localhost:20128`).
   - Simulasi Redis split-brain/restart dan validasi fallback in-memory cache/database authoritatif.
   - Pengujian degradasi graceful ketika seluruh AI Gateway eksternal mengalami timeout.
4. **Kepatuhan UU PDP No. 27/2022**:
   - Penataan hak subjek data: hak akses, perbaikan, pembatasan, dan hak penghapusan data pribadi (*right to erasure*).
   - Rekonsiliasi ketegangan hukum: retensi pencatatan pajak/keuangan 10 tahun (UU KUP & Dokumen Perusahaan) versus hak hapus data PDP diselesaikan melalui teknik pseudonimisasi/anonimisasi data transaksi permanen tanpa menghapus catatan buku besar.
5. **Incident Response & Operational Runbooks**:
   - Runbook mitigasi AI Gateway down total / kuota upstream habis.
   - Runbook rekonsiliasi pembayaran gagal massal & webhook delayed.
   - Runbook rotasi darurat secret/kunci enkripsi database.
   - Runbook pembersihan cache basi (*stale cache*) lintas instans.
6. **Backup & Disaster Recovery (DR)**:
   - Standar RPO (Recovery Point Objective) <= 1 menit untuk data billing/ledger dan RTO (Recovery Time Objective) <= 15 menit.
   - Prosedur continuous WAL archiving dan cold backup terenkripsi.
   - Prosedur uji restorasi mandiri secara terjadwal.
7. **Observability & Checklist Go-Live**:
   - Implementasi Four Golden Signals (Latency, Traffic, Errors, Saturation) dan metrik bisnis inti.
   - Checklist gerbang peluncuran (*gate criteria*) tanpa toleransi kompromi.

## Deliverables
> 📦 **Arsip Detail P08:** Seluruh dokumen analisis, planning, rules, teknis (load-test, security-review, chaos-failover, pdp-compliance, runbooks, backup-dr, go-live checklist), dan tracking tersimpan lengkap di [`../../5_Archives/phases/P08_Hardening-GoLive/`](../../5_Archives/phases/P08_Hardening-GoLive/).

- Dokumen load test suite k6 dan laporan benchmark performa (`teknis/load-test.md`).
- Dokumen security review, postur PCI-DSS SAQ A, dan arsitektur enkripsi (`teknis/security-review.md`).
- Dokumen metodologi chaos engineering dan matriks failover (`teknis/chaos-failover.md`).
- Kerangka kerja kepatuhan UU PDP No. 27/2022 dan skema anonimisasi pajak (`teknis/pdp-compliance.md`).
- Kumpulan runbook penanganan insiden P1-P4 siap pakai (`teknis/runbook-incidents.md`).
- Kebijakan backup, strategi DR, simulasi RPO/RTO (`teknis/backup-dr.md`).
- Lembar verifikasi final Go-Live readiness checklist (`teknis/go-live-checklist.md`).
- Rencana sprint, task WBS terperinci, timeline, aturan implementasi, dan tabel tracking.

## Kriteria Sukses (Success Criteria)
1. **Concurrency Enforcement**: 0% over-quota leakage pada pengujian k6 kuota konkuren (20 request serentak pada sisa kuota 5 token menghasilkan tepat 5 lolos dan 15 blokir 429).
2. **Webhook Idempotency**: Replay 10x webhook serentak menghasilkan tepat 1 status update dan 1 catatan ledger tanpa duplicate charging.
3. **Failover Transparency**: Pindahnya beban AI Gateway saat node aktif mati diselesaikan dalam <= 250ms tanpa error 5xx diterima oleh client.
4. **Keamanan**: Zero temuan critical/high pada audit OWASP dan konfigurasi PCI-DSS SAQ A terverifikasi.
5. **Kepatuhan Hukum**: Alur *Right to Erasure* tervalidasi dengan data PII pengguna dihilangkan, namun integritas invoice dan pembukuan finansial 10 tahun tetap balance (*anonymized retention*).
6. **Disaster Recovery**: Uji coba restorasi database point-in-time recovery (PITR) berhasil memenuhi RPO < 1 menit dan RTO < 15 menit.
7. **Observability**: Dashboard Grafana/Prometheus aktif memonitor Golden Signals + metrik bisnis (Active Subscriptions, Token Burn Rate, Payment Success Rate, Route Health).
