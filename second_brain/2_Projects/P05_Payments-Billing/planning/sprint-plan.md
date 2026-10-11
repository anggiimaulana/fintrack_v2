# Sprint Plan — Fase P05 (Payments & Billing)

Sprint plan untuk pelaksanaan implementasi fase P05 terbagi ke dalam 3 milestone/sprint mingguan dengan fokus modular, berlandaskan prinsip *test-driven* dan *zero-downtime*.

---

## Ringkasan Sprint

| Sprint | Nama Fokus | Sasaran Utama | Estimasi |
|---|---|---|---|
| **Sprint 1** | Skema Data, Abstraksi Gateway & Midtrans | Tabel `invoices` & `payments`, Interface `Gateway`, Integrasi Midtrans Snap & Webhook Idempoten | Hari 1 - 5 |
| **Sprint 2** | Alur Manual, Switch Gateway & Prorata | Unggah bukti transfer, approval super admin, proration engine, switch setting | Hari 6 - 10 |
| **Sprint 3** | Dunning, Grace Period, Realtime SSE & Hardening | Worker dunning, grace period enforcement, SSE live update, audit log & test suite | Hari 11 - 15 |

---

## Rincian Per Sprint

### Sprint 1: Skema Data, Abstraksi Gateway & Integrasi Midtrans
- **Tujuan**: Membangun fondasi database, arsitektur antarmuka Go, dan aliran otomatisasi pembayaran Midtrans Snap beserta verifikasi webhook yang aman.
- **Backlog Item**:
  1. `TASK-P05-001`: Penulisan skrip migrasi database `0007_create_payments_and_invoices.sql`.
  2. `TASK-P05-002`: Implementasi interface `Gateway` dan struct model di `internal/payment/gateway.go`.
  3. `TASK-P05-003`: Implementasi driver `MidtransGateway` (`CreateTransaction` via Snap API).
  4. `TASK-P05-004`: Endpoint publik webhook Midtrans dengan validasi HMAC SHA512 signature.
  5. `TASK-P05-005`: Mekanisme pull verification ke Midtrans Status API dan database concurrency lock (`SELECT FOR UPDATE`).
  6. `TASK-P05-006`: Pembuatan nomor invoice deterministik `INV/YYYYMMDD/SEQ` dan tabel `invoices`.
- **Keluaran Sprint 1**:
  - Migrasi skema database dieksekusi bersih.
  - Checkout plan menghasilkan Snap Token yang valid di sandbox Midtrans.
  - Webhook berhasil memverifikasi signature dan mengubah status invoice & payment menjadi `settled` secara idempoten.

---

### Sprint 2: Alur Manual, Resolver Switch Gateway & Engine Prorata
- **Tujuan**: Menyediakan opsi pembayaran transfer manual mandiri, mekanisme pergantian gateway global/per-plan, dan kalkulasi matematis prorata upgrade langganan.
- **Backlog Item**:
  1. `TASK-P05-007`: Implementasi driver `ManualGateway` dan modul upload bukti transfer (`proof_url`).
  2. `TASK-P05-008`: Endpoint review pembayaran manual bagi super admin (`approve` & `reject`).
  3. `TASK-P05-009`: Logika resolusi gateway (Global setting vs Plan override) di `internal/payment/resolver.go`.
  4. `TASK-P05-010`: Engine kalkulasi prorata upgrade (pembulatan ke bawah) di `internal/billing/proration.go`.
  5. `TASK-P05-011`: Penjadwalan downgrade akhir periode tanpa pengembalian dana (*no refund*).
  6. `TASK-P05-012`: Struktur penanganan PPN dasar (PPN 0% siap skala) di entitas invoice.
- **Keluaran Sprint 2**:
  - Alur manual berfungsi dari sisi upload user hingga validasi super admin.
  - Switch gateway dapat dilakukan di tabel settings tanpa merusak transaksi lama.
  - Unit test kalkulasi prorata upgrade lulus pengujian batas hari.

---

### Sprint 3: Dunning, Grace Period, Realtime SSE & Hardening
- **Tujuan**: Menjamin otomatisasi penagihan akhir siklus, masa tenggang keterlambatan, pembaruan antarmuka pengguna tanpa refresh, dan kepatuhan audit.
- **Backlog Item**:
  1. `TASK-P05-013`: Background worker siklus dunning (Reminder H-3, H-1, H-0).
  2. `TASK-P05-014`: Penegakan grace period 72 jam dan transisi status langganan ke `past_due`.
  3. `TASK-P05-015`: Integrasi realtime SSE via Redis Pub/Sub untuk notifikasi pembayaran sukses instan ke frontend.
  4. `TASK-P05-016`: Pencatatan audit trail menyeluruh ke tabel `audit_logs` untuk seluruh aksi finansial.
  5. `TASK-P05-017`: Penyusunan end-to-end integration test dan idempotency stress test.
- **Keluaran Sprint 3**:
  - Worker dunning berjalan sesuai interval cron.
  - Notifikasi realtime terkirim ke browser saat status invoice lunas.
  - Seluruh skenario audit dan idempotensi terverifikasi dengan hasil tes lulus 100%.

---

## Kriteria Masuk & Keluar Sprint (Entry & Exit Criteria)

### Entry Criteria
- Fase P02 (Fondasi & RBAC) dan migrasi P04 (Entitas Subscription dasar) telah terdefinisi.
- Kredensial sandbox Midtrans dan konfigurasi storage upload tersedia.

### Exit Criteria
- Seluruh task TASK-P05-001 s/d TASK-P05-017 selesai dengan status lolos verifikasi.
- Tidak ada floating point arithmetic dalam kode pembayaran.
- Tidak ada rekaman data sensitif kartu kredit di log atau database.
