# Implementation Plan — P05 Payments & Billing

| Field | Isi |
|---|---|
| **Fase** | P05 — Payments & Billing |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | P02 selesai (audit_logs, feature_flags, Redis pub/sub), P04 selesai (subscription core, plan, token balance) |
| **Master plan** | `../../IMPLEMENTATION-PLAN.md` |
| **Status** | Siap dieksekusi — belum ada kode ditulis |

> **Koreksi path & arsitektur:**
> - Seluruh kode backend payment berada di `backend/internal/payment/` dan prorata di `backend/internal/billing/`.
> - Frontend berada di `frontend/` (bukan `frontend/src/`).
> - Skema migrasi kanonik database: `database/migrations/0009_payments_invoices.sql` (mengalahkan nomor draf 0007).
> - Seluruh nominal uang WAJIB `BIGINT` / `int64` (rupiah integer murni, tanpa float/sen).
> - Standar PCI-DSS SAQ-A: sistem TIDAK PERNAH menyentuh, memproses, atau menyimpan nomor kartu kredit/CVV. Seluruh alur kartu didelegasikan ke Midtrans Snap.

---

## 1. Tujuan & Definition of Done

Menyediakan sistem pembayaran dan penagihan komprehensif, aman, dan berstandar enterprise yang mendukung multi-gateway (Midtrans Snap & Transfer Manual), penerbitan faktur deterministik, perhitungan prorata harian zona `Asia/Jakarta`, otomasi siklus pengingat dunning dan masa tenggang (grace period), notifikasi realtime via SSE, serta audit trail finansial bergaransi append-only.

**Definition of Done Fase:**
- [ ] Semua 17 task (TASK-P05-001 s/d TASK-P05-017) terimplementasi dengan pengujian unit dan integrasi.
- [ ] Skrip migrasi `database/migrations/0009_payments_invoices.sql` tereksekusi idempoten dan dump schema diperbarui.
- [ ] Interface `Gateway` bersih terpasang dengan driver Midtrans Snap dan Transfer Manual.
- [ ] Webhook Midtrans terverifikasi kriptografis SHA-512 dan tervalidasi ulang secara independen (*pull verification*) sebelum mutasi status DB.
- [ ] Penanganan webhook dan callback kebal terhadap kondisi balapan (*race condition*) serta replay attack dengan mekanisme `SELECT ... FOR UPDATE` dan unique constraint `idempotency_key`.
- [ ] Penomoran invoice deterministik `INV/YYYYMMDD/SEQ` berjalan atomik tanpa celah duplikasi nomor.
- [ ] Kalkulator prorata dan PPN menghitung nilai presisi dengan aturan pembulatan ke bawah (*floor*) ke integer rupiah terdekat.
- [ ] Seluruh aksi finansial dicatat ke tabel `audit_logs` secara append-only dalam transaksi database yang sama.
- [ ] Perintah verifikasi backend `go vet ./...`, `go test ./...`, dan `go build ./...` lulus tanpa galat.

---

## 2. Urutan Eksekusi (Wave)

Implementasi dibagi menjadi 5 gelombang terurut (Wave 1 s/d Wave 5) untuk memastikan ketergantungan modul tersambung secara linier dan stabil:

| Wave | Cakupan Fokus | Task Termasuk | Prasyarat |
|---|---|---|---|
| **W1** | Skema Database, Abstraksi Gateway, Driver Midtrans, & Webhook Receiver | TASK-P05-001, TASK-P05-002, TASK-P05-003, TASK-P05-004, TASK-P05-005 | P02, P04 |
| **W2** | Engine Invoice Deterministik, Driver Manual Transfer, & Panel Review Admin | TASK-P05-006, TASK-P05-007, TASK-P05-008 | W1 |
| **W3** | Dynamic Gateway Resolver, Kalkulator Prorata, Downgrade Policy, & Modul Pajak PPN | TASK-P05-009, TASK-P05-010, TASK-P05-011, TASK-P05-012 | W2 |
| **W4** | Otomasi Dunning Worker, Grace Period Worker, Realtime SSE, & Audit Trail Finansial | TASK-P05-013, TASK-P05-014, TASK-P05-015, TASK-P05-016 | W3 |
| **W5** | Pengujian Integrasi End-to-End, Concurrency Stress Test, & Verifikasi Idempotensi | TASK-P05-017 | W4 (Semua task fungsional) |

---

## 3. Detail Task

### Wave 1 — Fondasi Skema, Interface Gateway, Driver Midtrans, & Webhook Receiver

#### TASK-P05-001 — Skema Migrasi Database Payments & Invoices
- **Target File:** `database/migrations/0009_payments_invoices.sql`, `database/dump/schema.sql`
- **Langkah:**
  1. Buat tipe enum PostgreSQL: `invoice_status_enum` ('draft', 'issued', 'paid', 'past_due', 'voided') dan `payment_status_enum` ('pending', 'pending_review', 'settled', 'failed', 'expired', 'refunded').
  2. Buat tabel `invoices` dengan kolom `id` (UUID PK), `invoice_number` (VARCHAR(64) UNIQUE NOT NULL), `user_id` (UUID FK NOT NULL), `subscription_id` (UUID FK), `subtotal_idr` (BIGINT CHECK >= 0), `tax_idr` (BIGINT DEFAULT 0 CHECK >= 0), `total_idr` (BIGINT CHECK >= 0), `status` (invoice_status_enum), `issued_at`, `due_at`, `paid_at`, `voided_at`, `notes`, `created_at`, `updated_at`.
  3. Buat tabel `payments` dengan kolom `id` (UUID PK), `invoice_id` (UUID FK NOT NULL), `user_id` (UUID FK NOT NULL), `gateway` (VARCHAR(32) NOT NULL), `gateway_ref` (VARCHAR(128)), `amount_idr` (BIGINT NOT NULL CHECK > 0), `status` (payment_status_enum), `payment_method` (VARCHAR(64)), `proof_url` (TEXT), `idempotency_key` (VARCHAR(128) UNIQUE NOT NULL), `payload_snapshot` (JSONB NOT NULL DEFAULT '{}'), `created_at`, `updated_at`.
  4. Pasang relasi indeks: `idx_invoices_user_id`, `idx_invoices_status`, `idx_invoices_due_at`, `idx_payments_invoice_id`, `idx_payments_user_id`, `idx_payments_status`, `idx_payments_gateway_ref`.
  5. Sediakan skrip rollback terstruktur yang bersih (`DROP TABLE IF EXISTS ... CASCADE; DROP TYPE IF EXISTS ...;`).
- **Verifikasi (Bukti):** Eksekusi migrasi sukses di DB lokal/staging; tabel dan constraint terdaftar di `information_schema`; `database/dump/schema.sql` terbarui.
- **Catatan Keamanan & Zero-Downtime:** Semua pembuatan tabel menggunakan `IF NOT EXISTS` dan `CREATE INDEX CONCURRENTLY` (atau transaksional aman pada migrasi baru); tidak ada *table lock* yang mengganggu modul lain.

#### TASK-P05-002 — Abstraksi Interface Gateway di `internal/payment`
- **Target File:** `backend/internal/payment/gateway.go`, `backend/internal/payment/models.go`
- **Langkah:**
  1. Definisikan tipe alias `PaymentStatus` dan `InvoiceStatus` beserta konstanta terkait.
  2. Susun interface `Gateway` standar:
     - `Name() string`
     - `CreateTransaction(ctx context.Context, req CreateTxRequest) (*CreateTxResponse, error)`
     - `VerifyWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error)`
     - `CheckStatus(ctx context.Context, gatewayRef string) (*StatusResult, error)`
     - `CancelTransaction(ctx context.Context, gatewayRef string) error`
  3. Definisikan struct input/output DTO: `CreateTxRequest` (memuat `AmountIDR int64`), `CreateTxResponse` (`SnapToken`, `RedirectURL`, `GatewayRef`), `WebhookResult`, `StatusResult`.
- **Verifikasi (Bukti):** Unit test kompilasi interface; validasi model memastikan seluruh properti mata uang bertipe `int64`.
- **Catatan Desain:** Mencegah kebocoran implementasi vendor ke domain penagihan (`internal/billing/`).

#### TASK-P05-003 — Implementasi Driver Midtrans Snap Client
- **Target File:** `backend/internal/payment/midtrans.go`, `backend/internal/config/config.go`
- **Langkah:**
  1. Tambahkan konfigurasi `MIDTRANS_SERVER_KEY`, `MIDTRANS_CLIENT_KEY`, `MIDTRANS_IS_PRODUCTION`, dan `MIDTRANS_SNAP_URL` pada modul config.
  2. Implementasikan struct `MidtransGateway` yang memenuhi antarmuka `Gateway`.
  3. Implementasikan fungsi `CreateTransaction`:
     - Buat HTTP client dengan timeout 15 detik dan retry terukur (maksimal 2x dengan backoff eksponensial).
     - Format basic authentication header: `Basic base64(MIDTRANS_SERVER_KEY + ":")`.
     - Susun request Snap: parameter `gross_amount` dikirim dalam integer rupiah murni, `item_details` dan `customer_details` tervalidasi.
     - Simpan respon lengkap Midtrans ke DTO `CreateTxResponse`.
  4. Tangani error jaringan, status kode HTTP bukan 2xx, dan format respon Snap secara terstruktur.
- **Verifikasi (Bukti):** Unit test dengan `httptest.Server` mock Midtrans Snap API; pengujian pembentukan payload request dan otentikasi basic auth.
- **Catatan Keamanan (PCI-DSS SAQ-A):** Sistem hanya menerima `token` dan `redirect_url` dari Midtrans. Pengguna mengisi detail pembayaran langsung di iframe/halaman Midtrans Snap. Data kartu kredit tidak menyentuh server backend FinTrack v2 sama sekali.

#### TASK-P05-004 — Endpoint Publik Webhook Handler Midtrans
- **Target File:** `backend/internal/payment/handler_webhook.go`, `backend/internal/delivery/http/router.go`
- **Langkah:**
  1. Daftarkan rute publik `POST /api/v1/payments/webhook/midtrans` (tanpa middleware auth JWT).
  2. Baca raw body request dan batasi ukuran pembacaan (maksimal 1 MB via `http.MaxBytesReader`).
  3. Ekstraksi field `order_id`, `status_code`, `gross_amount`, `signature_key` dari JSON notifikasi.
  4. Hitung hash SHA-512: `SHA512(order_id + status_code + gross_amount + MIDTRANS_SERVER_KEY)`.
  5. Bandingkan hasil hash menggunakan perbandingan konstan (`subtle.ConstantTimeCompare`) terhadap `signature_key` payload.
  6. Kembalikan respons `401 Unauthorized` atau `400 Bad Request` bila hash tidak valid. Jika valid, teruskan payload ke `payment.Service`.
- **Verifikasi (Bukti):** Unit test `handler_webhook_test.go` menguji skenario signature cocok (sukses 200), signature salah (gagal 401), dan body corrupt (gagal 400).
- **Catatan Keamanan:** Penggunaan perbandingan konstan mencegah serangan *timing attack* pada verifikasi kriptografi signature.

#### TASK-P05-005 — Mekanisme Verifikasi Ulang (Pull) & Idempotensi DB Webhook
- **Target File:** `backend/internal/payment/service.go`, `backend/internal/payment/repository.go`
- **Langkah:**
  1. Setelah signature valid, lakukan verifikasi independen (*pull verification*) dengan memanggil endpoint resmi Midtrans `GET /v2/{order_id}/status`.
  2. Buka transaksi database PostgreSQL (`BEGIN`).
  3. Kunci baris pembayaran yang bersangkutan: `SELECT * FROM payments WHERE gateway_ref = $1 FOR UPDATE`.
  4. Validasi idempotensi:
     - Jika status baris pembayaran sudah `settled` atau `paid`, catat log duplikasi dan segera lakukan `COMMIT` serta return HTTP `200 OK`.
     - Jika status belum final, lakukan transisi status sesuai hasil pull verification (`settled`, `failed`, atau `expired`).
  5. Dalam transaksi yang sama, perbarui baris invoice terkait menjadi `paid` dan isi kolom `paid_at = NOW()`.
  6. Panggil service langganan (`internal/billing`) untuk mengaktifkan paket atau menambah token balance.
  7. Lakukan `COMMIT` transaksi database.
- **Verifikasi (Bukti):** Unit test mock repository menguji race condition; eksekusi 2 webhook berurutan memastikan pembaruan invoice hanya terjadi sekali.
- **Catatan Keamanan:** Penguncian baris (`FOR UPDATE`) mencegah *double crediting* ketika Midtrans mengirimkan notifikasi ganda dalam milidetik yang sama.

---

### Wave 2 — Invoice Deterministik, Transfer Manual, & Panel Review Admin

#### TASK-P05-006 — Generator Nomor Invoice & Service Penagihan
- **Target File:** `backend/internal/payment/invoice_service.go`, `backend/internal/payment/invoice_repo.go`
- **Langkah:**
  1. Buat mekanisme sekuens harian berbasis tabel atau sequence PostgreSQL `invoice_seq_YYYYMMDD` yang di-reset per hari atau format atomik dengan locking:
     - Format: `INV/YYYYMMDD/SEQ` (misal `INV/20261011/00001`).
     - Sequence menggunakan padding 5 digit angka (`%05d`).
  2. Buat fungsi `CreateInvoice(ctx, req CreateInvoiceDTO)`:
     - Hitung `total_idr = subtotal_idr + tax_idr`.
     - Tentukan batas waktu `due_at` (standar 24 jam untuk pesanan baru, atau konfigurasi spesifik paket).
     - Simpan ke tabel `invoices` dengan status awal `issued`.
  3. Sediakan fungsi pencarian invoice berdasarkan ID dan nomor invoice dengan proteksi kepemilikan `user_id`.
- **Verifikasi (Bukti):** Unit test membuktikan eksekusi pembuatan 50 invoice berurutan pada tanggal yang sama menghasilkan nomor sekuensial yang unik dan urut tanpa celah.
- **Catatan Integritas:** Penggunaan fungsi atomik di level DB memastikan tidak terjadi tabrakan nomor invoice meskipun dijalankan pada multi-instance backend.

#### TASK-P05-007 — Implementasi Driver Pembayaran Manual & Unggah Bukti
- **Target File:** `backend/internal/payment/manual.go`, `backend/internal/payment/handler_manual.go`, `backend/internal/infra/storage/storage.go`
- **Langkah:**
  1. Implementasikan struct `ManualGateway` yang memenuhi interface `Gateway`:
     - Method `CreateTransaction` mengembalikan instruksi rekening bank (nomor rekening, nama pemilik, nominal unik) dan status `pending`.
  2. Buat endpoint `POST /api/v1/payments/manual/:payment_id/upload-proof`:
     - Autentikasi JWT user pemilik pembayaran.
     - Terima file via form-data multipart (field `proof_image`).
     - Validasi MIME type menggunakan deteksi header byte (`image/jpeg`, `image/png`, `image/webp`). Tolak file binary/executable.
     - Validasi ukuran file (maksimum 5 MB).
     - Simpan berkas ke storage aman lokal/S3 dengan nama file berbasis UUID acak (`uploads/payment-proofs/{uuid}.{ext}`).
     - Simpan URL bukti ke `payments.proof_url` dan ubah status payment menjadi `pending_review`.
- **Verifikasi (Bukti):** Test upload file dengan file valid (berhasil 200, status `pending_review`), file PDF/EXE (ditolak 400), dan file > 5 MB (ditolak 413).
- **Catatan Keamanan:** Validasi *magic bytes* file mencegah pengunggahan webshell berbahaya; direktori penyimpanan file tidak boleh memiliki izin eksekusi script.

#### TASK-P05-008 — Endpoint Review Persetujuan Manual oleh Super Admin
- **Target File:** `backend/internal/payment/handler_admin.go`, `backend/internal/payment/service_admin.go`
- **Langkah:**
  1. Daftarkan endpoint admin yang dilindungi middleware otorisasi role `super_admin`:
     - `POST /api/v1/super-admin/payments/manual/:id/approve`
     - `POST /api/v1/super-admin/payments/manual/:id/reject`
  2. Logika `Approve`:
     - Buka transaksi DB: kunci baris payment `FOR UPDATE`.
     - Validasi status saat ini harus `pending_review`.
     - Ubah status payment menjadi `settled`, update invoice menjadi `paid` (`paid_at = NOW()`).
     - Aktifkan langganan / kreditkan token kuota ke akun user.
     - Catat log persetujuan ke tabel audit.
  3. Logika `Reject`:
     - Terima alasan penolakan (`reason` string wajib) dari body request.
     - Ubah status payment menjadi `failed`, perbarui invoice atau batalkan.
     - Kirimkan notifikasi penolakan ke user terkait.
     - Catat log penolakan ke tabel audit.
- **Verifikasi (Bukti):** Test handler admin membuktikan non-admin menerima `403 Forbidden`, admin valid berhasil menyetujui, dan percobaan menyetujui payment yang sudah `settled` ditolak.
- **Catatan Keamanan:** Proteksi RBAC ketat dan pencatatan identitas super admin yang mengeksekusi mutasi ke `audit_logs`.

---

### Wave 3 — Gateway Resolver, Prorata, Downgrade, & Tax Engine

#### TASK-P05-009 — Resolver Pemilihan Gateway (Global vs Per-Plan)
- **Target File:** `backend/internal/payment/resolver.go`, `backend/internal/payment/factory.go`
- **Langkah:**
  1. Buat gateway registry/factory yang mendaftarkan instance gateway aktif (`midtrans`, `manual`).
  2. Implementasikan fungsi `ResolveGateway(ctx context.Context, planID string) (Gateway, error)`:
     - Ambil data paket dari database/cache: periksa kolom `plans.payment_gateway_override`.
     - Jika terisi dan terdaftar di factory, gunakan gateway override tersebut.
     - Jika tidak terisi/NULL, baca konfigurasi default sistem dari `settings` (key `payment_gateway_default`).
     - Jika konfigurasi default kosong atau bermasalah, gunakan fallback darurat `manual`.
  3. Saat inisiasi checkout, kunci nama gateway yang terpilih secara permanen pada kolom `payments.gateway`.
- **Verifikasi (Bukti):** Unit test resolusi: plan dengan override "manual" menghasilkan `ManualGateway`, plan tanpa override menghasilkan gateway dari `settings`, konfigurasi diubah tidak mempengaruhi pembayaran yang sudah terbit.
- **Catatan Arsitektur:** Menjamin transaksi *in-flight* tetap dapat diproses oleh handler yang sesuai meskipun konfigurasi global diubah oleh admin di tengah jalan.

#### TASK-P05-010 — Engine Perhitungan Prorata Upgrade
- **Target File:** `backend/internal/billing/proration.go`, `backend/internal/billing/proration_test.go`
- **Langkah:**
  1. Implementasikan fungsi kalkulasi prorata matematis berdasarkan zona waktu `Asia/Jakarta`:
     - Hitung sisa hari aktif: $S = \max(0, \lceil (T_{\text{end}} - T_{\text{now}}) / 86400 \rceil)$.
     - Hitung sisa kredit paket lama: $C_A = \lfloor (S \times P_A) / D \rfloor$.
     - Hitung biaya pemakaian paket baru: $C_B = \lfloor (S \times P_B) / D \rfloor$.
     - Hitung nominal tagihan upgrade: $P_{\text{upgrade}} = \max(0, C_B - C_A)$.
  2. Semua operasi perhitungan menggunakan integer arithmetic (`int64`) dengan fungsi `math.Floor` setelah pembagian.
  3. Terbitkan invoice penyesuaian instan untuk nominal $P_{\text{upgrade}}$.
  4. Bila $P_{\text{upgrade}} == 0$ (misal upgrade di menit-menit akhir siklus), langsung mutasikan paket tanpa tagihan.
- **Verifikasi (Bukti):** Unit test berbasis tabel (*table-driven test*) mencakup kasus Starter ke Pro (hari ke-10 sisa 20 hari menghasilkan Rp 66.667), kasus upgrade hari terakhir, dan kasus durasi 0 hari.
- **Catatan Finansial:** Pembulatan ke bawah (*floor*) menjamin pengguna tidak pernah ditagih melebihi nilai prorata riil dalam integer rupiah.

#### TASK-P05-011 — Manajemen Downgrade Siklus Akhir Periode
- **Target File:** `backend/internal/billing/downgrade_service.go`, `backend/internal/billing/downgrade_worker.go`
- **Langkah:**
  1. Buat endpoint/service penjadwalan downgrade: `ScheduleDowngrade(ctx, userID, targetPlanID)`.
  2. Validasi kebijakan *No-Refund*: tegaskan bahwa downgrade tidak memicu pengembalian dana dalam bentuk apa pun.
  3. Simpan target paket pada kolom `subscriptions.scheduled_plan_id`.
  4. Pertahankan hak akses, kuota, dan status paket saat ini hingga waktu `current_period_end` tercapai.
  5. Buat background worker yang mengecek langganan yang mencapai akhir siklus dengan `scheduled_plan_id != NULL`:
     - Terapkan perubahan paket ke target paket baru.
     - Reset `scheduled_plan_id = NULL`.
     - Terapkan kebijakan *soft-limit* (data lama tidak dihapus, kuota baru disesuaikan).
- **Verifikasi (Bukti):** Unit test simulasi downgrade memastikan paket tidak langsung berubah saat request dipanggil, dan worker berhasil mengubah paket ketika `current_period_end` terlampaui.
- **Catatan UX:** Mencegah frustrasi pengguna dengan membiarkan fasilitas paket tinggi tetap dinikmati hingga hak hari yang sudah dibayar habis.

#### TASK-P05-012 — Modul Kalkulator Pajak (Tax / PPN Engine)
- **Target File:** `backend/internal/payment/tax.go`, `backend/internal/payment/tax_test.go`
- **Langkah:**
  1. Buat interface `TaxCalculator` dan implementasi `StandardTaxCalculator`.
  2. Baca nilai persentase pajak dari tabel `settings` (key `tax_rate_percent`, default integer `0`, disiapkan untuk PPN 11% atau 12%).
  3. Hitung besaran pajak: $\text{tax\_idr} = \lfloor (\text{subtotal\_idr} \times \text{tax\_rate\_percent}) / 100 \rfloor$.
  4. Hitung total akhir: $\text{total\_idr} = \text{subtotal\_idr} + \text{tax\_idr}$.
  5. Sediakan struktur rincian pajak (*tax breakdown*) untuk dicatat pada invoice metadata dan tampilan bukti bayar.
- **Verifikasi (Bukti):** Unit test perhitungan pajak: tarif 0% menghasilkan tax 0, tarif 11% dari Rp 100.000 menghasilkan Rp 11.000, pembulatan floor pada angka pecahan desimal.
- **Catatan Kepatuhan:** Pajak dipisahkan secara transparan dari subtotal untuk mematuhi regulasi faktur pajak Indonesia.

---

### Wave 4 — Dunning Worker, Grace Period, Realtime SSE, & Audit Trail

#### TASK-P05-013 — Background Worker Siklus Penagihan Dunning
- **Target File:** `backend/internal/payment/dunning_worker.go`, `backend/cmd/server/main.go`
- **Langkah:**
  1. Implementasikan worker berkala menggunakan interval `time.Ticker` (berjalan setiap 1 jam).
  2. Pindai langganan yang akan kedaluwarsa pada interval H-3 (72 jam) dan H-1 (24 jam) sebelum `current_period_end`.
  3. Cek tabel penanda pengingat agar tidak mengirim notifikasi ganda pada rentang yang sama.
  4. Buat entri notifikasi sistem (`notifications`) bertipe `SUBSCRIPTION_EXPIRING_SOON` atau `SUBSCRIPTION_EXPIRING_URGENT`.
  5. Terbitkan pesan realtime ke Redis pub/sub untuk memicu toast/banner di antarmuka klien yang sedang aktif.
  6. Pada H-0, terbitkan invoice penagihan baru (`INV/...`) dengan status `issued`.
- **Verifikasi (Bukti):** Unit test mock repository worker dunning mendeteksi langganan yang memenuhi kriteria H-3 dan H-1 tanpa memproses entri yang sudah dikirim pengingat.
- **Catatan Operasional:** Menekan tingkat kegagalan perpanjangan tak disengaja (*involuntary churn*) pada metode pembayaran berbasis transfer/QRIS.

#### TASK-P05-014 — Penegakan Masa Tenggang (Grace Period) & Past Due
- **Target File:** `backend/internal/payment/grace_worker.go`
- **Langkah:**
  1. Implementasikan worker pemantauan masa tenggang yang dieksekusi secara periodik (setiap 30 menit).
  2. Identifikasi invoice yang melewati `due_at` tanpa pembayaran terkonfirmasi.
  3. Transisi 1: Jika waktu berada dalam rentang $\le 72$ jam sejak `due_at`, perbarui status langganan menjadi `in_grace_period`. Akses fitur tetap diizinkan dengan peringatan visual.
  4. Transisi 2: Jika waktu melewati 72 jam (> 72 jam) sejak `due_at`, ubah status langganan menjadi `past_due` dan status invoice menjadi `past_due`.
  5. Saat status `past_due` aktif, terapkan kebijakan *soft-limit*: kunci fitur premium/AI gateway dan batasi kapasitas token, namun jangan hapus riwayat data finansial pengguna.
- **Verifikasi (Bukti):** Test worker membuktikan langganan bertransisi tepat waktu dari `active` -> `in_grace_period` -> `past_due`, dan entri event tercatat di audit log.
- **Catatan Keamanan Bisnis:** Mencegah eksploitasi layanan AI berbiaya API tinggi oleh akun yang menunggak pembayaran setelah grace period usai.

#### TASK-P05-015 — Integrasi Pembaruan Realtime Status Pembayaran via SSE
- **Target File:** `backend/internal/payment/realtime_publisher.go`, `backend/internal/realtime/sse_handler.go`
- **Langkah:**
  1. Buat abstraksi event publisher pembayaran `PaymentRealtimePublisher`.
  2. Ketika pembayaran berubah status menjadi `settled`, terbitkan payload JSON ke Redis channel `user_events:{user_id}`:
     - Event type: `payment.settled`.
     - Payload memuat: `invoice_number`, `amount_idr`, `payment_id`, `subscription_status`, `updated_at`.
  3. Hubungkan dengan SSE stream handler yang meneruskan event ke koneksi browser aktif pengguna.
  4. Frontend menerima event dan otomatis mengalihkan antarmuka checkout ke layar sukses (*payment success state*) tanpa memerlukan reload manual (*F5*).
- **Verifikasi (Bukti):** Integrasi test pub/sub: pemanggilan `PublishPaymentSettled` memicu pesan pada subscriber Redis channel yang sesuai dalam waktu < 50 milidetik.
- **Catatan User Experience:** Menghilangkan kebingungan pengguna setelah menyelesaikan pembayaran QRIS/Snap di ponsel mereka.

#### TASK-P05-016 — Audit Trail Komprehensif Seluruh Aksi Finansial
- **Target File:** `backend/internal/payment/audit_interceptor.go`
- **Langkah:**
  1. Buat interceptor audit log khusus domain pembayaran yang mengonsumsi modul `internal/audit` (P02).
  2. Catat event finansial penting:
     - `PAYMENT_CREATE`: Inisiasi transaksi checkout.
     - `WEBHOOK_RECEIVE`: Notifikasi masuk dari Midtrans beserta signature validity.
     - `PAYMENT_SETTLED`: Pembayaran terkonfirmasi lunas.
     - `MANUAL_PROOF_UPLOAD`: User mengunggah berkas transfer.
     - `MANUAL_APPROVE` / `MANUAL_REJECT`: Super admin meninjau transaksi manual.
     - `INVOICE_VOID`: Pembatalan tagihan.
  3. Tangkap konteks mutasi: `actor_id` (User ID / Admin ID / System Worker), IP address, User-Agent, state `before` dan `after`.
  4. Pastikan pencatatan audit berjalan dalam transaksi atomik yang sama dengan mutasi status finansial.
- **Verifikasi (Bukti):** Test transaksi memvalidasi bahwa jika mutasi pembayaran di-commit, baris di `audit_logs` wajib terbentuk; jika transaksi di-rollback, log audit ikut ter-rollback.
- **Catatan Kepatuhan Regulasi:** Memenuhi mandat UU PDP dan standar audit keuangan untuk integritas data transaksi yang tidak dapat dimanipulasi (*tamper-evident*).

---

### Wave 5 — Pengujian Integrasi End-to-End & Stress Testing

#### TASK-P05-017 — Pengujian Integrasi End-to-End & Idempotensi Concurrency
- **Target File:** `backend/tests/integration/payment_test.go`, `backend/tests/integration/concurrency_test.go`
- **Langkah:**
  1. Siapkan pengujian integrasi berbasis docker database test yang mengisolasi state transaksi.
  2. Skenario Uji 1 (End-to-End Midtrans Flow): Inisiasi checkout -> generate invoice -> mock Midtrans Snap -> kirim webhook sukses -> pull verification valid -> mutasi invoice `paid` -> aktivasi langganan.
  3. Skenario Uji 2 (End-to-End Manual Flow): Inisiasi checkout manual -> unggah bukti bayar dummy -> status `pending_review` -> approval super admin -> aktivasi paket.
  4. Skenario Uji 3 (Concurrency Stress Test Webhook): Kirim 20 request webhook secara serentak (*goroutine concurrent*) untuk nomor order yang sama. Verifikasi bahwa database hanya memproses satu pembaruan dan 19 request lainnya menerima respons 200 OK secara idempoten tanpa duplikasi kredit paket.
  5. Skenario Uji 4 (Negative & Security Test): Webhook dengan signature palsu ditolak 401; transisi status ilegal (misal mengubah `settled` menjadi `pending`) ditolak oleh state machine.
- **Verifikasi (Bukti):** `go test -v -race ./tests/integration/...` lulus 100% tanpa data race dan tanpa kebocoran transaksi.
- **Catatan Stabilitas:** Menjamin sistem kebal terhadap anomali jaringan, *burst traffic*, dan percobaan penipuan status transaksi.

---

## 4. Matriks Bukti (Evidence)

Setiap task memiliki artefak keluaran terukur yang harus diverifikasi sebelum status tugas dapat dinyatakan selesai:

| Task ID | Nama Tugas | Artefak Bukti yang Dihasilkan |
|---|---|---|
| **TASK-P05-001** | Skema Migrasi Payments & Invoices | Berkas `0009_payments_invoices.sql`, tabel terdaftar di DB, `dump/schema.sql` terbarui |
| **TASK-P05-002** | Abstraksi Interface Gateway | Berkas `gateway.go` & `models.go`, seluruh model uang tervalidasi `int64` |
| **TASK-P05-003** | Driver Midtrans Snap Client | Berkas `midtrans.go`, unit test mock snap client lulus |
| **TASK-P05-004** | Webhook Handler Midtrans | Berkas `handler_webhook.go`, test validasi signature SHA-512 lulus |
| **TASK-P05-005** | Pull Verification & Idempotensi | Berkas `service.go`, test `FOR UPDATE` & penanganan idempotent callback sukses |
| **TASK-P05-006** | Generator Nomor Invoice | Berkas `invoice_service.go`, unit test keunikan sekuens `INV/YYYYMMDD/SEQ` |
| **TASK-P05-007** | Driver Pembayaran Manual | Berkas `manual.go` & `handler_manual.go`, unit test upload validasi MIME type |
| **TASK-P05-008** | Review Persetujuan Super Admin | Berkas `handler_admin.go`, unit test approval & rejection dengan audit log |
| **TASK-P05-009** | Resolver Gateway Dinamis | Berkas `resolver.go`, unit test hierarki override plan vs global settings |
| **TASK-P05-010** | Engine Prorata Upgrade | Berkas `proration.go`, unit test matematis pembulatan floor rupiah |
| **TASK-P05-011** | Manajemen Downgrade Akhir Periode | Berkas `downgrade_service.go`, test kebijakan no-refund dan penjadwalan siklus |
| **TASK-P05-012** | Modul Kalkulator Pajak (PPN) | Berkas `tax.go`, unit test kalkulasi tarif pajak dari settings |
| **TASK-P05-013** | Dunning Background Worker | Berkas `dunning_worker.go`, test pengingat H-3, H-1, H-0 tanpa duplikasi |
| **TASK-P05-014** | Penegakan Grace Period & Past Due | Berkas `grace_worker.go`, test transisi status 72 jam dan soft-limit |
| **TASK-P05-015** | Integrasi Realtime SSE Pembayaran | Berkas `realtime_publisher.go`, test pub/sub redis channel `user_events` |
| **TASK-P05-016** | Audit Trail Finansial Komprehensif | Berkas `audit_interceptor.go`, baris `audit_logs` terbentuk transaksional |
| **TASK-P05-017** | E2E & Concurrency Test | Berkas `tests/integration/payment_test.go`, log eksekusi test concurrent hijau |

---

## 5. Perintah Verifikasi

Perintah standar yang dijalankan oleh agen verifikator untuk memastikan seluruh sistem pembayaran memenuhi standar mutu kode:

```bash
# 1. Verifikasi Kode Backend (Go)
cd backend
go vet ./internal/payment/... ./internal/billing/...
go test -v -race ./internal/payment/... ./internal/billing/...
go test -v -race ./tests/integration/... -run Payment
go build ./cmd/server

# 2. Verifikasi Kode Frontend (Next.js)
cd ../frontend
pnpm lint
pnpm test
pnpm build

# 3. Verifikasi Migrasi Database & Schema Dump
cd ..
APP_DB_PASSWORD=secret database/scripts/migrate.sh
database/scripts/dump.sh
git diff --exit-code database/dump/schema.sql
```

---

## 6. Prosedur Rollback

Strategi pengembalian keadaan bila ditemukan anomali kritis saat peluncuran:

1. **Pengalihan Gateway Darurat (Runtime Switch):**
   - Jika integrasi Midtrans mengalami gangguan masif di sisi vendor, ubah pengaturan default sistem tanpa restart server:
     ```sql
     UPDATE settings SET value = 'manual' WHERE key = 'payment_gateway_default';
     ```
   - Seluruh transaksi baru otomatis dialihkan ke jalur transfer bank manual.
2. **Rollback Fitur via Feature Flag:**
   - Fitur-fitur baru (seperti modul PPN atau kalkulator upgrade otomatis) dikontrol via tabel `feature_flags`:
     - Ubah `is_enabled = false` untuk flag `ff_tax_engine` atau `ff_instant_upgrade`. Sistem kembali ke alur penagihan standar.
3. **Rollback Migrasi Database:**
   - Skrip migrasi bersifat aditif dan tidak merusak skema eksisting.
   - Bila diperlukan pembersihan total modul pembayaran:
     ```sql
     DROP TABLE IF EXISTS payments CASCADE;
     DROP TABLE IF EXISTS invoices CASCADE;
     DROP TYPE IF EXISTS payment_status_enum;
     DROP TYPE IF EXISTS invoice_status_enum;
     ```

---

## 7. Handoff ke Fase Berikutnya (P06 — Pengaturan & Kuota)

Setelah Fase P05 selesai dan terverifikasi penuh, modul berikut siap diserahkan untuk integrasi di Fase P06:
1. **Invoice & Billing Data:** Modul P06 (Settings & Admin Dashboard) dapat langsung mengonsumsi entitas `invoices` dan `payments` untuk visualisasi riwayat pembayaran user dan rekapitulasi omset platform.
2. **Payment Gateway Settings:** Antarmuka konfigurasi gateway global di P06 dapat secara langsung memperbarui key `payment_gateway_default` dan `tax_rate_percent` di tabel `settings`.
3. **Proration & Entitlement Sync:** Sinyal lunas pembayaran dari P05 menjadi pemicu penambahan saldo token AI dan pembaruan kuota fitur di modul entitlement P06.

---

## 8. Referensi Teknis

- `second_brain/2_Projects/P05_Payments-Billing/README.md` — Gambaran umum arsitektur pembayaran.
- `second_brain/2_Projects/P05_Payments-Billing/planning/tasks.md` — Rincian 17 tugas spesifik P05.
- `second_brain/2_Projects/P05_Payments-Billing/planning/timeline.md` — Jadwal dan dependensi sprint.
- `second_brain/2_Projects/P05_Payments-Billing/teknis/midtrans-webhook.md` — Spesifikasi teknis webhook & validasi signature Midtrans.
- `second_brain/2_Projects/P05_Payments-Billing/teknis/payment-integration.md` — Desain antarmuka `Gateway` & model DTO.
- `second_brain/2_Projects/P05_Payments-Billing/teknis/payment-gateway-switch.md` — Mekanisme switch gateway global dan per-plan override.
- `second_brain/2_Projects/P05_Payments-Billing/teknis/proration.md` — Rumus matematis prorata upgrade dan kebijakan no-refund downgrade.
- `second_brain/2_Projects/P05_Payments-Billing/teknis/invoice.md` — Spesifikasi penomoran sekuensial invoice dan skema perpajakan.
- `second_brain/2_Projects/P05_Payments-Billing/teknis/dunning-grace.md` — Timeline pengingat penagihan dunning dan penegakan batas 72 jam grace period.
- `second_brain/2_Projects/P05_Payments-Billing/teknis/manual-payment.md` — Prosedur upload bukti transfer dan alur review admin.
- `second_brain/2_Projects/IMPLEMENTATION-PLAN.md` — Master implementation plan lintas fase FinTrack v2.
- `second_brain/3_Areas/operasional/RUNBOOK_ZERO_DOWNTIME.md` — Standar operasional penerapan migrasi zero-downtime.
