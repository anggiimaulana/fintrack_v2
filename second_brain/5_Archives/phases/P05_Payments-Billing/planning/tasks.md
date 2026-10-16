# Rincian Tugas — Fase P05 (Payments & Billing)

Daftar tugas implementasi terperinci dengan konvensi format `TASK-P05-NNN`. Minimal 17 tugas terstruktur yang mencakup modul backend, database, keamanan, dan integrasi penagihan.

---

## Daftar Tugas (TASK-P05-001 .. TASK-P05-017)

### TASK-P05-001: Skema Migrasi Database Payments & Invoices
- **Deskripsi**: Menulis skrip migrasi SQL `0009_payments_invoices.sql` untuk membuat tabel `invoices` dan `payments` beserta indeks relasional dan kekangan unik.
- **Komponen**: `database/migrations/0009_payments_invoices.sql`
- **Kriteria Penerimaan**:
  - Tabel `invoices` memiliki kolom `id`, `invoice_number` (UNIQUE), `user_id`, `subscription_id`, `subtotal_idr` (BIGINT), `tax_idr` (BIGINT), `total_idr` (BIGINT), `status`, `issued_at`, `due_at`, `paid_at`, `voided_at`.
  - Tabel `payments` memiliki kolom `id`, `invoice_id`, `user_id`, `gateway`, `gateway_ref`, `amount_idr` (BIGINT), `status`, `payment_method`, `proof_url`, `idempotency_key` (UNIQUE), `payload_snapshot` (JSONB).
  - Tipe data nominal seluruhnya `BIGINT` atau `INTEGER`.
  - Skrip migrasi bersifat idempoten (`CREATE TABLE IF NOT EXISTS`) dan memiliki bagian rollback yang valid.

---

### TASK-P05-002: Abstraksi Interface Gateway di `internal/payment`
- **Deskripsi**: Mendefinisikan interface `Gateway` yang bersih pada package `internal/payment` beserta model DTO input dan output transaksi.
- **Komponen**: `internal/payment/gateway.go`, `internal/payment/models.go`
- **Kriteria Penerimaan**:
  - Interface memuat method: `Name() string`, `CreateTransaction(...)`, `VerifyWebhook(...)`, `CheckStatus(...)`, `CancelTransaction(...)`.
  - Semua model request/response menggunakan tipe data `int64` untuk seluruh representasi mata uang rupiah.

---

### TASK-P05-003: Implementasi Driver Midtrans Snap Client
- **Deskripsi**: Mengimplementasikan struct `MidtransGateway` yang mengonsumsi Midtrans Snap API untuk membuat transaksi pembayaran hosted.
- **Komponen**: `internal/payment/midtrans.go`
- **Kriteria Penerimaan**:
  - Mengirim payload transaksi ke API Snap Midtrans (`/snap/v1/transactions`) dengan basic auth menggunakan Server Key.
  - Mengembalikan `snap_token` dan `redirect_url` kepada klien.
  - Parameter `gross_amount` dikirim sebagai integer sesuai konversi rupiah murni.
  - Menangani error jaringan dan timeout dengan pembungkusan pesan error yang informatif.

---

### TASK-P05-004: Endpoint Publik Webhook Handler Midtrans
- **Deskripsi**: Membuat HTTP handler `POST /api/v1/payments/webhook/midtrans` untuk menerima callback notifikasi status pembayaran dari Midtrans.
- **Komponen**: `internal/payment/handler_webhook.go`
- **Kriteria Penerimaan**:
  - Menerima payload JSON notifikasi dari Midtrans tanpa otentikasi Bearer token (publik).
  - Melakukan validasi signature hash SHA512: `SHA512(order_id + status_code + gross_amount + server_key)`.
  - Mengembalikan HTTP status `400 Bad Request` atau `401 Unauthorized` bila signature tidak cocok.

---

### TASK-P05-005: Mekanisme Verifikasi Ulang (Pull) & Idempotensi DB Webhook
- **Deskripsi**: Menjamin penanganan webhook tidak menduplikasi kredit dan kebal terhadap pemalsuan payload dengan melakukan pengecekan balik ke API Midtrans.
- **Komponen**: `internal/payment/service.go`
- **Kriteria Penerimaan**:
  - Memanggil endpoint Midtrans Status API (`GET /v2/{order_id}/status`) untuk mencocokkan status asli sebelum memperbarui basis data.
  - Mengunci baris `payments` menggunakan `SELECT ... FOR UPDATE`.
  - Memeriksa `idempotency_key` dan status saat ini: jika sudah `settled`, abaikan pemrosesan berulang dan segera kembalikan response `200 OK`.
  - Memperbarui status `payments` dan `invoices` dalam satu transaksi atomik database.

---

### TASK-P05-006: Generator Nomor Invoice & Service Penagihan
- **Deskripsi**: Membuat layanan pembuatan invoice deterministik dengan pola format nomor `INV/YYYYMMDD/SEQ` serta pencatatan rincian tagihan.
- **Komponen**: `internal/payment/invoice_service.go`, `internal/payment/invoice_repo.go`
- **Kriteria Penerimaan**:
  - Nomor invoice dihasilkan secara atomik dan terurut harian (misal `INV/20261011/00001`).
  - Menyimpan rincian tagihan, menghitung `total_idr = subtotal_idr + tax_idr`.
  - Status awal invoice adalah `issued` dengan batas waktu jatuh tempo `due_at` yang terdefinisi.

---

### TASK-P05-007: Implementasi Driver Pembayaran Manual & Unggah Bukti
- **Deskripsi**: Mengimplementasikan struct `ManualGateway` dan endpoint bagi pengguna untuk mengunggah berkas bukti transfer (`proof_url`).
- **Komponen**: `internal/payment/manual.go`, `internal/payment/handler_manual.go`
- **Kriteria Penerimaan**:
  - Endpoint `POST /api/v1/payments/manual/:payment_id/upload-proof` menerima berkas multipart.
  - Validasi tipe MIME file (hanya `image/jpeg`, `image/png`, `image/webp`) dan batas ukuran berkas maksimum 5 MB.
  - Berkas disimpan di storage privat aman dengan penamaan UUID acak.
  - Status payment diperbarui menjadi `pending_review`.

---

### TASK-P05-008: Endpoint Review Persetujuan Manual oleh Super Admin
- **Deskripsi**: Membuat endpoint khusus super admin untuk meninjau, menyetujui, atau menolak pembayaran transfer manual.
- **Komponen**: `internal/payment/handler_admin.go`, `internal/payment/service_admin.go`
- **Kriteria Penerimaan**:
  - Endpoint `POST /api/v1/super-admin/payments/manual/:id/approve` mengubah status payment dan invoice menjadi `settled`/`paid` serta mengaktifkan hak langganan terkait.
  - Endpoint `POST /api/v1/super-admin/payments/manual/:id/reject` menerima alasan penolakan (`reason`), mengubah status menjadi `failed`, dan mengirim notifikasi penolakan ke user.
  - Wajib dilindungi middleware autentikasi role `super_admin`.

---

### TASK-P05-009: Resolver Pemilihan Gateway (Global vs Per-Plan)
- **Deskripsi**: Mengimplementasikan logika resolusi pemilihan gateway pembayaran saat inisiasi transaksi checkout.
- **Komponen**: `internal/payment/resolver.go`
- **Kriteria Penerimaan**:
  - Membaca override pada `plans.payment_gateway_override`. Jika nilainya terisi dan aktif, gunakan gateway tersebut.
  - Jika tidak ada override, baca konfigurasi default dari `settings.payment_gateway`.
  - Mengunci nama gateway yang terpilih ke kolom `payments.gateway`.
  - Transaksi yang sudah terbit (*in-flight*) tidak terpengaruh oleh perubahan konfigurasi di masa depan.

---

### TASK-P05-010: Engine Perhitungan Prorata Upgrade
- **Deskripsi**: Mengembangkan fungsi matematis kalkulasi biaya prorata saat pengguna melakukan upgrade paket di tengah masa aktif langganan.
- **Komponen**: `internal/billing/proration.go`
- **Kriteria Penerimaan**:
  - Menghitung sisa hari kalender aktif berdasarkan zona waktu `Asia/Jakarta`.
  - Menghitung sisa nilai kredit paket lama dan biaya sisa hari paket baru.
  - Hasil selisih pembagian dibulatkan ke bawah (*floor*) ke integer rupiah terdekat.
  - Menerbitkan invoice penyesuaian instan untuk selisih pembayaran yang harus diselesaikan.

---

### TASK-P05-011: Manajemen Downgrade Siklus Akhir Periode
- **Deskripsi**: Mengatur perilaku perubahan paket ke tier yang lebih rendah atau gratis (*downgrade*).
- **Komponen**: `internal/billing/downgrade_service.go`
- **Kriteria Penerimaan**:
  - Memastikan tidak ada pengembalian dana (*no refund*) yang dilakukan pada sistem.
  - Status dan hak akses fitur paket saat ini tetap aktif hingga `current_period_end`.
  - Menyimpan ID paket target pada `subscriptions.scheduled_plan_id` untuk dieksekusi saat siklus berakhir.

---

### TASK-P05-012: Modul Kalkulator Pajak (Tax / PPN Engine)
- **Deskripsi**: Membangun modul kalkulasi PPN yang modular pada penerbitan invoice dengan tarif yang dapat dikonfigurasi.
- **Komponen**: `internal/payment/tax.go`
- **Kriteria Penerimaan**:
  - Membaca setting `tax_rate_percent` dari tabel `settings` (default awal `0`).
  - Menghitung nilai `tax_idr = floor(subtotal_idr * tax_rate_percent / 100)`.
  - Mengisi rincian `tax_idr` secara transparan pada invoice sebelum total diakumulasikan.

---

### TASK-P05-013: Background Worker Siklus Penagihan Dunning
- **Deskripsi**: Mengembangkan background worker berkala untuk memonitor masa kedaluwarsa langganan dan mengirim peringatan tagihan.
- **Komponen**: `internal/payment/dunning_worker.go`
- **Kriteria Penerimaan**:
  - Berjalan setiap jam memindai langganan yang akan jatuh tempo pada H-3, H-1, dan H-0.
  - Menghasilkan event pengingat ke tabel notifikasi dan memancarkan sinyal realtime ke pengguna terkait.
  - Tidak mengirim pengingat ganda untuk tahapan yang sama pada hari yang sama.

---

### TASK-P05-014: Penegakan Masa Tenggang (Grace Period) & Past Due
- **Deskripsi**: Mengatur transisi otomatis status langganan saat invoice melewati batas jatuh tempo pembayaran.
- **Komponen**: `internal/payment/grace_worker.go`
- **Kriteria Penerimaan**:
  - Memberikan masa tenggang 72 jam terhitung sejak `due_at` invoice penagihan baru.
  - Selama masa tenggang, status langganan berubah menjadi `in_grace_period` namun hak akses tetap aktif.
  - Melewati 72 jam tanpa pembayaran, status otomatis diperbarui menjadi `past_due` dan akses fitur dibatasi (*soft-limit*).

---

### TASK-P05-015: Integrasi Pembaruan Realtime Status Pembayaran via SSE
- **Deskripsi**: Menghubungkan mutasi sukses pembayaran dengan sistem event realtime SSE untuk memperbarui antarmuka pengguna secara instan.
- **Komponen**: `internal/payment/realtime_publisher.go`
- **Kriteria Penerimaan**:
  - Menerbitkan pesan ke Redis channel `user_events:{user_id}` ketika pembayaran berstatus `settled`.
  - Klien web menerima event tipe `payment.settled` dengan payload nomor invoice dan status langganan baru.
  - Antarmuka kasir/checkout pada frontend otomatis bertransisi ke tampilan sukses tanpa reload halaman.

---

### TASK-P05-016: Audit Trail Komprehensif Seluruh Aksi Finansial
- **Deskripsi**: Mencatat setiap aktivitas pembayaran dan penagihan ke tabel `audit_logs` sesuai mandat regulasi UU PDP.
- **Komponen**: `internal/payment/audit_interceptor.go`
- **Kriteria Penerimaan**:
  - Mencatat aktor (user ID atau super admin ID), aksi (`PAYMENT_CREATE`, `WEBHOOK_RECEIVE`, `MANUAL_APPROVE`, `MANUAL_REJECT`, `INVOICE_VOID`), payload sebelum dan sesudah perubahan (*before -> after*), serta alamat IP dan User-Agent.
  - Pencatatan bersifat append-only dan terikat dalam transaksi database yang sama dengan mutasi status.

---

### TASK-P05-017: Pengujian Integrasi End-to-End & Idempotensi Concurrency
- **Deskripsi**: Membangun pengujian otomatis end-to-end untuk memvalidasi alur pembayaran lengkap serta ketahanan terhadap kondisi balapan (*race condition*).
- **Komponen**: `tests/integration/payment_test.go`
- **Kriteria Penerimaan**:
  - Pengujian simulasi 20 webhook konkruen untuk order yang sama hanya menghasilkan satu eksekusi pembaruan kuota.
  - Pengujian transisi status ilegal (misal dari `settled` kembali ke `pending`) ditolak oleh sistem.
  - Pengujian verifikasi signature Midtrans palsu mengembalikan error 401.
