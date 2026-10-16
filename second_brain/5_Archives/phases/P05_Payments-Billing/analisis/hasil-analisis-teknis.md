# Analisis Teknis — Fase P05 (Payments & Billing)

Dokumen ini mendokumentasikan keputusan teknis arsitektural (K-P05-01 s/d K-P05-15) untuk modul pembayaran dan penagihan FinTrack v2, menyelaraskan dengan keputusan tingkat proyek (K-01 s/d K-30).

---

## Daftar Keputusan Teknis (K-P05-01 .. K-P05-15)

### K-P05-01: Representasi Moneter Integer Murni
- **Konteks**: Nilai transaksi rupiah, PPN, dan diskon rentan terhadap kesalahan presisi IEEE 754 float binary.
- **Keputusan**: Semua nilai uang di basis data PostgreSQL menggunakan tipe `BIGINT` atau `INTEGER` dalam satuan rupiah penuh (IDR). Di Go menggunakan `int64`. Tidak ada sen, tidak ada float.
- **Alasan**: Mencegah kehilangan presisi atau anomali pembulatan pada kalkulasi saldo, invoice, dan pelaporan keuangan.

### K-P05-02: Kepatuhan PCI-DSS SAQ-A via Midtrans Snap Hosted
- **Konteks**: FinTrack v2 tidak memiliki sertifikasi PCI-DSS Level 1/2 untuk mengelola atau menyimpan Primary Account Number (PAN), CVV, atau tanggal kedaluwarsa kartu.
- **Keputusan**: Seluruh pengumpulan instrumen pembayaran menggunakan Midtrans Snap Token yang dirender melalui Snap JS (popup/redirect hosted). Server backend FinTrack v2 hanya menerima `snap_token` dan `redirect_url` dari API Midtrans dan menyimpannya sementara.
- **Alasan**: Menurunkan beban audit kepatuhan ke PCI-DSS SAQ-A; server backend tidak pernah menyentuh data kartu.

### K-P05-03: Abstraksi Provider melalui Interface Go
- **Konteks**: FinTrack v2 mendukung dua jenis pembayaran: Midtrans otomatis dan transfer manual. Di masa depan gateway lain dapat ditambahkan.
- **Keputusan**: Dibangun interface tunggal pada package `internal/payment`:
  ```go
  type Gateway interface {
      Name() string
      CreateTransaction(ctx context.Context, req CreateTxRequest) (*CreateTxResponse, error)
      VerifyWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookResult, error)
      CheckStatus(ctx context.Context, gatewayRef string) (*StatusResult, error)
      CancelTransaction(ctx context.Context, gatewayRef string) error
  }
  ```
  Driver Midtrans dan Manual mengimplementasikan interface ini secara independen.
- **Alasan**: Memisahkan logika domain billing dari detail protokol vendor pembayaran (*loose coupling*).

### K-P05-04: Idempotensi Webhook Ganda (Signature + State Machine + API Re-check)
- **Konteks**: Midtrans mengirimkan webhook berkali-kali jika terjadi latensi jaringan atau timeout. Serangan man-in-the-middle atau replay attack berisiko memalsukan status pembayaran.
- **Keputusan**: 
  1. Validasi SHA512 signature hash: `SHA512(order_id + status_code + gross_amount + server_key)`.
  2. Transaksi database berstatus `SELECT FOR UPDATE` pada baris `payments`.
  3. Verifikasi ulang (*pull validation*) ke endpoint Midtrans GET `/v2/{order_id}/status` sebelum melakukan state transition ke `settled`.
  4. Penyimpanan hash payload di `payments.idempotency_key` (format: `md5(order_id + transaction_status + status_code)`).
- **Alasan**: Mencegah race condition, double-crediting kuota langganan, dan pemalsuan payload webhook.

### K-P05-05: Alur Pembayaran Manual dengan Penampungan Bukti Transfer
- **Konteks**: Sebagian pengguna UMKM/personal di Indonesia lebih memilih transfer langsung ke rekening admin atau QRIS statis tanpa biaya payment gateway pihak ketiga.
- **Keputusan**: Pengguna mengunggah gambar bukti transfer (`image/jpeg`, `image/png`, `image/webp` max 5MB). File disimpan di object storage lokal/S3 privat dengan nama teracak UUID. Status pembayaran disetel ke `pending_review`. Super admin memiliki antrean dashboard untuk `Approve` atau `Reject` disertai catatan penolakan.
- **Alasan**: Memberikan opsi murah bebas biaya gateway dengan akuntabilitas approval yang tercatat di audit log.

### K-P05-06: Hirarki Resolusi Gateway (Global vs Per-Plan)
- **Konteks**: FinTrack v2 memungkinkan admin mengubah gateway pembayaran secara global atau hanya untuk paket tertentu (misal: paket Enterprise wajib invoice manual, paket Starter via Midtrans).
- **Keputusan**:
  1. Gateway ditentukan saat pembuatan order pembayaran (`POST /api/v1/checkout`).
  2. Resolusi urutan prioritas: `plans.payment_gateway_override` (jika tidak null) -> `settings.payment_gateway` (default fallback).
  3. Gateway yang terpilih dikunci permanen pada baris `payments.gateway`.
- **Alasan**: Transaksi *in-flight* tidak terpengaruh jika admin mengubah setting gateway di tengah jalan.

### K-P05-07: Prorata Upgrade Dibulatkan ke Bawah (Floor)
- **Konteks**: User berpindah dari paket berbayar rendah ke tinggi di tengah siklus aktif.
- **Keputusan**: 
  - Formula:
    $$\text{Sisa Hari} = \max(0, \lceil(\text{expires\_at} - \text{now}) / 86400\rceil)$$
    $$\text{Kredit Plan Lama} = \lfloor(\text{Sisa Hari} / \text{Durasi Total}) \times \text{Harga Plan Lama}\rfloor$$
    $$\text{Beban Plan Baru} = \lfloor(\text{Sisa Hari} / \text{Durasi Total}) \times \text{Harga Plan Baru}\rfloor$$
    $$\text{Tagihan Upgrade} = \max(0, \text{Beban Plan Baru} - \text{Kredit Plan Lama})$$
  - Nilai desimal dibulatkan ke bawah (*floor*) ke rupiah penuh.
- **Alasan**: Adil bagi pengguna, tidak memicu komplain sisa selisih sen, dan mudah diverifikasi audit.

### K-P05-08: Siklus Downgrade Akhir Periode (Tanpa Refund)
- **Konteks**: User beralih dari paket tinggi ke rendah atau ke paket free.
- **Keputusan**: Tidak ada prorata pengembalian dana (*no refund*). Fitur paket tinggi tetap aktif hingga akhir siklus berjalan (`current_period_end`). Pemindahan ke paket baru dijadwalkan otomatis pada tanggal reset siklus berikutnya melalui `subscriptions.scheduled_plan_id`.
- **Alasan**: Menghindari sengketa penagihan balik (*chargeback*) dan penyalahgunaan kuota token yang sudah terpakai.

### K-P05-09: Penomoran Invoice Deterministik Berurutan
- **Konteks**: Invoice memerlukan format nomor unik legal untuk pembukuan dan perpajakan Indonesia.
- **Keputusan**: Format nomor invoice: `INV/YYYYMMDD/SEQ` (contoh: `INV/20261011/00042`). Dihasilkan menggunakan sequence PostgreSQL harian `invoice_number_seq_YYYYMMDD` atau sequence atomic berbasis tanggal.
- **Alasan**: Unik, terurut kronologis, tidak dapat ditebak (*tamper-resistant*), dan memenuhi standar akuntansi.

### K-P05-10: Arsitektur Invoice Siap PPN (Tax-Ready)
- **Konteks**: K-11 menetapkan invoice dasar namun siap PPN bila diwajibkan regulasi.
- **Keputusan**: Tabel `invoices` menyimpan `subtotal_idr`, `tax_idr`, dan `total_idr` secara eksplisit, dengan relasi formula:
  $$\text{total\_idr} = \text{subtotal\_idr} + \text{tax\_idr}$$
  Konfigurasi `tax_rate_percent` disimpan di tabel `settings` (default awal `0`).
- **Alasan**: Aktivasi tarif PPN 11% atau 12% di masa mendatang hanya memerlukan update satu baris konfigurasi tanpa migrasi skema tabel.

### K-P05-11: Siklus Pengingat Dunning Multi-Tahap (H-3, H-1, H-0)
- **Konteks**: Metode pembayaran non-kartu (seperti VA atau transfer manual) tidak memiliki mekanisme auto-debit otomatis pada akhir siklus.
- **Keputusan**: Worker background berkala (cron per jam) memindai invoice atau langganan yang akan jatuh tempo:
  - H-3: Mengirim pengingat tagihan pertama via in-app notification & SSE.
  - H-1: Mengirim pengingat mendesak tagihan via in-app & SSE.
  - H-0: Penerbitan invoice penagihan baru dan permintaan tindakan user.
- **Alasan**: Mengurangi churn tidak disengaja (*involuntary churn*) bagi metode pembayaran manual/VA.

### K-P05-12: Kebijakan Grace Period 3 Hari Kalender
- **Konteks**: Pembayaran renewal mungkin terlambat karena libur perbankan atau kelalaian user.
- **Keputusan**: Masa tenggang diberikan selama 72 jam sejak `due_at`. Selama masa tenggang, status langganan menjadi `in_grace_period` dan akses fitur tetap dibuka. Jika lewat 72 jam belum lunas, status berubah menjadi `past_due` dan akses dikunci (*soft-limit*).
- **Alasan**: Pengalaman pengguna tetap baik tanpa langsung memutus akses di hari H keterlambatan.

### K-P05-13: Status State Machine Terisolasi untuk Payments & Invoices
- **Konteks**: Siklus hidup pembayaran dan faktur harus konsisten dan tidak ambigu.
- **Keputusan**: 
  - Status `invoices`: `draft` -> `issued` -> `paid` / `past_due` / `voided`.
  - Status `payments`: `pending` -> `pending_review` (manual) -> `settled` / `failed` / `expired` / `refunded`.
  - Transisi status hanya dapat berjalan satu arah berdasarkan matriks transisi resmi.
- **Alasan**: Mencegah race condition yang dapat mengubah transaksi yang sudah lunas kembali menjadi pending.

### K-P05-14: Audit Logging Wajib pada Semua Mutasi Pembayaran
- **Konteks**: K-29 dan regulasi UU PDP No. 27/2022 mewajibkan penelusuran aksi finansial.
- **Keputusan**: Setiap aksi penerbitan invoice, persetujuan pembayaran manual, pembatalan invoice, dan eksekusi webhook harus menyisipkan rekaman ke tabel `audit_logs` secara sinkron dalam satu transaksi database.
- **Alasan**: Menjamin non-repudiation dan jejak audit lengkap saat terjadi audit kepatuhan atau perselisihan pelanggan.

### K-P05-15: Pengiriman Event Realtime via SSE pada Perubahan Status Finansial
- **Konteks**: K-20 menetapkan integrasi SSE untuk perubahan status eksternal agar UI pengguna segera terbarui tanpa perlu refresh manual.
- **Keputusan**: Ketika status `payments` berubah menjadi `settled` (baik via webhook Midtrans maupun approval super admin), payment service memicu publish event ke Redis channel `user_events:{user_id}`, yang kemudian dialirkan ke klien Next.js melalui SSE endpoint yang sudah ada.
- **Alasan**: Pengalaman checkout langsung menampilkan centang hijau sukses seketika setelah pembayaran terkonfirmasi.
