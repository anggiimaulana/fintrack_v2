# P05 — Payments & Billing

Dokumentasi spesifikasi teknis, rancangan arsitektur, rencana kerja, aturan implementasi, dan pelacakan untuk fase **P05 (Payments & Billing)** FinTrack v2.

---

## 1. Konteks & Latar Belakang

FinTrack v2 bertransisi dari aplikasi pencatatan keuangan pribadi mandiri (*single-tier*) menjadi layanan berbasis langganan B2C berjenjang (*freemium to paid plans*). Untuk mendukung monetisasi token AI dan fitur premium, sistem membutuhkan infrastruktur pembayaran yang:
1. **Aman & Patuh Hukum**: Memenuhi standar PCI-DSS SAQ-A (tanpa menyimpan data sensitif kartu kredit) dan UU PDP No. 27/2022.
2. **Fleksibel**: Mendukung payment gateway multi-vendor (Midtrans Snap) dan verifikasi transfer manual (*bank transfer/QRIS direct*) dengan mekanisme switch gateway instan tanpa mengganggu transaksi berjalan (*in-flight*).
3. **Deterministik & Kebal Anomali**: Nominal rupiah selalu integer (`int64`, tanpa float/sen), ledger transaksi append-only, penanganan webhook idempoten dengan verifikasi ganda via API status, dan proration upgrade dibulatkan ke bawah (*floor*).

Modul ini diimplementasikan di `internal/payment/` dengan pola antarmuka Go yang bersih, terpisah dari modul `internal/billing/` dan `internal/entitlement/`.

---

## 2. Cakupan Fase (Scope)

### Di Dalam Cakupan (In-Scope)
1. **Modul `internal/payment/`**:
   - Abstraksi interface `Gateway` (`CreateTransaction`, `VerifyWebhook`, `CheckStatus`, `CancelTransaction`).
   - Implementasi driver `MidtransGateway` (Midtrans Snap redirect/popup).
   - Implementasi driver `ManualGateway` (aliran upload bukti transfer dan antrean verifikasi).
2. **Model Data Pembayaran & Tagihan**:
   - Tabel `payments`: `id`, `user_id`, `invoice_id`, `gateway`, `gateway_ref`, `amount_idr` (INT), `status`, `payment_method`, `proof_url`, `idempotency_key` (UNIQUE), `payload_snapshot` (JSONB), timestamps.
   - Tabel `invoices`: `id`, `invoice_number`, `user_id`, `subscription_id`, `subtotal_idr` (INT), `tax_idr` (INT), `total_idr` (INT), `status`, `issued_at`, `due_at`, `paid_at`, `voided_at`, timestamps.
3. **Midtrans Webhook & Security**:
   - Validasi signature HMAC SHA512 (`order_id + status_code + gross_amount + server_key`).
   - Verifikasi ulang independen via Midtrans GET Status API untuk mencegah spoofing webhook.
   - Mutasi status idempoten berbasis DB lock / status state machine.
4. **Alur Pembayaran Manual**:
   - Unggah bukti transfer ke storage privat / direktori aman dengan validasi MIME & size.
   - Endpoint review super admin: `POST /api/v1/super-admin/payments/manual/:id/approve` dan `reject`.
   - Audit trail setiap aksi persetujuan/penolakan manual.
5. **Switch Gateway & Resolver**:
   - Konfigurasi gateway default sistem di tabel `settings` (`payment_gateway = midtrans | manual`).
   - Override gateway per-plan di kolom `plans.payment_gateway_override`.
   - Transaksi *in-flight* tetap memproses webhook/verifikasi sesuai gateway pencatat awal (`payments.gateway`).
6. **Perhitungan Prorata Upgrade**:
   - Formula prorata presisi harian berbasis timezone `Asia/Jakarta`.
   - Pembulatan ke bawah (*floor*) ke rupiah penuh.
   - Penerbitan invoice penyesuaian instan.
7. **Dunning & Grace Period**:
   - Siklus reminder tagihan H-3, H-1, H-0 via notifikasi in-app / realtime SSE.
   - Masa tenggang (*grace period*) 3 hari kalender sebelum status langganan berubah menjadi `past_due` atau `canceled`.
   - Transisi status invoice: `draft` -> `issued` -> `paid` / `past_due` / `voided`.

### Di Luar Cakupan (Out-of-Scope)
- Penyimpanan nomor kartu kredit atau CVV (menggunakan Snap hosted token).
- Pajak multi-wilayah kompleks (PPN dikonfigurasi 0% pada tahap awal, kolom `tax_idr` disiapkan).
- Integrasi e-wallet selain channel yang disediakan oleh Midtrans Snap.

---

## 3. Hasil Kerja (Deliverables)

1. **Struktur Folder Dokumentasi Fase P05**:
   - `second_brain/2_Projects/P05_Payments-Billing/`
     - `README.md`
     - `analisis/hasil-analisis-teknis.md`
     - `planning/sprint-plan.md`
     - `planning/tasks.md`
     - `planning/timeline.md`
     - `rules/implementation-rules.md`
     - `tracking/tracking.md`
     - `teknis/payment-integration.md`
     - `teknis/midtrans-webhook.md`
     - `teknis/manual-payment.md`
     - `teknis/invoice.md`
     - `teknis/proration.md`
     - `teknis/dunning-grace.md`
     - `teknis/payment-gateway-switch.md`
2. **Artefak Kode Backend yang Dirancang**:
   - Migrasi SQL tabel `invoices`, `payments`, dan indeks pendukung.
   - Antarmuka `internal/payment/gateway.go` beserta concrete struct Midtrans dan Manual.
   - Engine kalkulasi prorata di `internal/billing/proration.go`.
   - Scheduler dunning & grace period worker di `internal/payment/dunning_worker.go`.
   - Endpoint publik webhook dan endpoint internal super admin.

---

## 4. Kriteria Keberhasilan (Success Criteria)

1. [x] **Integritas Moneter 100%**: tidak ada float untuk rupiah (BIGINT/int64 di DB, struct, JSON). *(pricing/tax floor; diuji)*
2. [x] **Anti Double-Charge**: webhook berulang untuk transaksi sama tidak menduplikasi kredit/aktivasi. *(diuji: 20 webhook paralel → 1 transisi & 1 aktivasi)*
3. [x] **PCI-DSS SAQ-A**: tidak ada data kartu disimpan; alur kartu via Midtrans Snap hosted. *(desain + Midtrans gateway)*
4. [x] **Isolasi In-Flight Gateway**: gateway dikunci pada baris payment saat checkout; ubah global tidak merusak transaksi berjalan. *(diuji resolver in-flight)*
5. [x] **Prorasi Akurat**: selisih harga harian deterministik (integer floor) + invoice. *(diuji)*
