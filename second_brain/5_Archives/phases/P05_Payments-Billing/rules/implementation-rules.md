# Implementation Rules — Fase P05 (Payments & Billing)

Aturan implementasi ini bersifat **mengikat (*binding*)** dan wajib dipatuhi oleh seluruh pengembang yang menyentuh modul `internal/payment/`, `internal/billing/`, serta skema database penagihan.

---

## 1. Aturan Presisi Moneter (Anti-Float Rule)

1. **Dilarang Keras Menggunakan Float**:
   - Tipe data `float32`, `float64`, atau tipe floating point lainnya **TIDAK BOLEH** digunakan untuk merepresentasikan nilai mata uang dalam bentuk apapun (rupiah, pajak, diskon, saldo token).
   - Seluruh nilai nominal mata uang wajib bertipe `int64` di kode Go dan bertipe `BIGINT` di skema PostgreSQL.
2. **Pembulatan Operasi Pembagian**:
   - Setiap operasi pembagian (seperti kalkulasi prorata atau perhitungan persentase pajak) wajib dibulatkan ke bawah (*floor*) menggunakan operasi aritmatika integer integer division murni:
     ```go
     // Benar:
     taxAmount := (subtotal * taxRate) / 100
     
     // Dilarang:
     // taxAmount := int64(float64(subtotal) * 0.11)
     ```

---

## 2. Kepatuhan PCI-DSS SAQ-A & Kerahasiaan Data

1. **Larangan Penyimpanan Data Kartu**:
   - Dilarang menyimpan, mencatat di log (*logging*), atau menerima Primary Account Number (PAN), CVV/CVC, tanggal kedaluwarsa kartu kredit/debit, atau PIN perbankan pada server FinTrack v2.
   - Semua interaksi pengumpulan kartu wajib didelegasikan sepenuhnya ke halaman hosted Midtrans Snap (`snap.js` atau Snap redirect URL).
2. **Sanitasi Log Pembayaran**:
   - Payload JSON yang disimpan di `payload_snapshot` atau dicetak ke log sistem wajib disanitasi dari field token autentikasi rahasia atau informasi PII sensitif yang tidak diperlukan.

---

## 3. Idempotensi & State Machine Transaksi

1. **Kekangan Idempotensi Unik**:
   - Setiap baris pada tabel `payments` wajib memiliki `idempotency_key` yang dilindungi indeks `UNIQUE`.
   - Webhook yang masuk dengan `order_id` dan hash status yang sama tidak boleh mengeksekusi mutasi ganda terhadap kuota langganan atau saldo token.
2. **Kunci Baris Konkuren (*Pessimistic Concurrency Lock*)**:
   - Pemrosesan mutasi status pembayaran wajib diawali dengan:
     ```sql
     SELECT id, status, invoice_id FROM payments WHERE gateway_ref = $1 FOR UPDATE;
     ```
   - Jika status telah mencapai status akhir (`settled`, `failed`, `expired`, `voided`), pemrosesan wajib langsung berhenti dan mengembalikan response sukses idempoten tanpa melempar error.
3. **Pola Transisi Status Satu Arah**:
   - Transisi status dilarang mundur (contoh: `settled` dilarang berubah kembali menjadi `pending` atau `pending_review`).

---

## 4. Keamanan Verifikasi Webhook

1. **Verifikasi Hash Signature Wajib**:
   - Webhook Midtrans wajib divalidasi keasliannya dengan mencocokkan header/payload signature:
     $$\text{signature} = \text{SHA512}(\text{order\_id} + \text{status\_code} + \text{gross\_amount} + \text{server\_key})$$
   - Permintaan dengan signature tidak valid wajib langsung ditolak dengan status HTTP 401 Unauthorized.
2. **Verifikasi Ganda (*Pull Verification*)**:
   - Sebelum menandai transaksi sebagai lunas (`settled`), backend wajib memanggil API status Midtrans secara independen via HTTPS untuk memastikan status yang dilaporkan webhook identik dengan rekaman di server Midtrans.

---

## 5. Alur Bukti Transfer Manual

1. **Validasi File Ketat**:
   - File bukti transfer yang diunggah wajib divalidasi tipe konten aslinya (*magic number sniffing*, bukan hanya ekstensi berkas).
   - Hanya tipe `image/jpeg`, `image/png`, dan `image/webp` yang diizinkan dengan ukuran file maksimal 5 MB.
2. **Isolasi Penyimpanan**:
   - File bukti transfer tidak boleh ditaruh di folder publik web server yang dapat diakses sembarang pihak tanpa otentikasi.
   - URL yang diberikan untuk peninjauan admin harus berupa URL privat atau ditayangkan melalui handler terotentikasi.

---

## 6. Audit Trail & Kepatuhan UU PDP

1. **Pencatatan Wajib Sinkron**:
   - Setiap perubahan status pada `payments` dan `invoices` wajib mencatat entri baru ke tabel `audit_logs` di dalam blok transaksi database (`tx.Commit()`) yang sama.
   - Kolom audit mencakup: `actor_id`, `actor_role`, `action`, `entity_type`, `entity_id`, `before_state`, `after_state`, `ip_address`, `user_agent`.
2. **Append-Only**:
   - Tabel `audit_logs`, `payments`, dan `invoices` tidak boleh memiliki operasi `DELETE` pada kode aplikasi. Pembatalan tagihan menggunakan status `voided`.

---

## 7. Prinsip Zero Downtime & Feature Flag

1. **Skema Database Expand-Contract**:
   - Penambahan kolom baru pada tabel penagihan wajib bersifat nullable atau memiliki nilai default yang aman.
   - Dilarang melakukan migrasi destruktif yang mengunci tabel utama dalam durasi panjang (*long table locks*).
2. **Feature Flags**:
   - Integrasi Midtrans dan fitur pembayaran mandiri wajib berada di bawah kendali feature flag `FEATURE_BILLING_PAYMENT_ENABLED`. Jika flag nonaktif, sistem fallback ke mode pembatasan graceful.
