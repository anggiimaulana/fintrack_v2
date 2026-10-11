# Hasil Analisis Teknis — Fase P06 (Super Admin Dashboard)

Dokumen ini mendokumentasikan keputusan teknis arsitektural dan implementasi (K-P06-01 s/d K-P06-12) untuk modul Super Admin Dashboard pada FinTrack v2. Seluruh keputusan konsisten dengan Keputusan Terkunci Global (K-01 s/d K-30).

---

## Daftar Keputusan Teknis (K-P06-01 .. K-P06-12)

### K-P06-01: Arsitektur Otorisasi dan Boundary Route Admin
- **Keputusan**: Semua endpoint backend Super Admin diletakkan di bawah namespace rute `/api/v1/admin/*` dan diproteksi oleh middleware rantai ganda: `AuthMiddleware()` (verifikasi session/JWT terinjeksi konteks pengguna) diikuti oleh `RequireSuperAdmin()` (verifikasi peran dari database berversi / Redis cache). Pada layer frontend, seluruh halaman diletakkan di dalam folder `app/(dashboard)/super-admin/*` dengan server-side layout guard yang memeriksa role pada server component sebelum merender view.
- **Rasional**: Mencegah kebocoran data administratif dan eskalasi hak istimewa (privilege escalation). Sesuai K-02 dan K-13, hak akses ditentukan secara otoritatif di backend. Frontend hanya memberikan proteksi navigasi awal (UX redirection ke `/dashboard` jika bukan super admin).
- **Dampak**: Tidak ada akses anonim atau pengguna berbayar reguler yang dapat menyentuh resource data admin. Setiap kegagalan otorisasi menghasilkan HTTP 403 Forbidden dengan payload JSON standar dan event keamanan dicatat ke logger.

### K-P06-02: Standarisasi Komponen `<DataTable>` dan Sinkronisasi URL Search Params
- **Keputusan**: Seluruh tampilan tabular pada antarmuka admin wajib menggunakan komponen bersama `<DataTable>` yang dibangun di atas TanStack Table v8. Parameter tabel disinkronkan dua arah ke URL Search Params:
  - `page`: nomor halaman aktif (integer, default 1).
  - `per_page`: batas ukuran halaman (integer tetap 20 sesuai K-22).
  - `sort`: kunci field sorting (string).
  - `order`: arah pengurutan (`asc` atau `desc`).
  - `q`: string pencarian bebas (search query).
  - `status`, `type`, `date_from`, `date_to`: facet filter kontekstual.
  Penomoran baris wajib bersifat absolut berdasarkan formula: `(page - 1) * per_page + index + 1`.
- **Rasional**: Menjamin konsistensi UX di seluruh 8 menu dashboard. Keadaan tabel menjadi *shareable* (dapat dibagikan via URL), *bookmarkable*, dan tetap bertahan saat pengguna menekan tombol Back/Forward browser.
- **Dampak**: Komponen frontend tidak menyimpan state pagination di local component memory terisolasi, melainkan di URL via `useSearchParams()` dan Next.js `useRouter()`.

### K-P06-03: Formula Agregasi Metrik Finansial (MRR) dan Performa Dashboard
- **Keputusan**: Metrik Monthly Recurring Revenue (MRR) dihitung pada backend dengan formula:
  $$\text{MRR} = \sum_{\text{active subscriptions}} \left( \frac{\text{plan.price\_idr} \times 30}{\text{plan.billing\_period\_days}} \right)$$
  Data metrik dashboard utama (Ringkasan MRR, Pengguna Aktif 30 Hari, Total Pemakaian Token 24 Jam, Payment Success Rate, Status Endpoint AI, Antrean Review Manual) diagregasikan secara terindeks dan disimpan di Redis dengan cache TTL 60 detik (`admin:metrics:overview`).
- **Rasional**: Query agregasi langsung terhadap jutaan baris data transaksi dan token ledger berisiko membebani database operasional PostgreSQL. Cache berdurasi singkat menjaga respon API tetap di bawah 200ms sambil tetap memberikan kesegaran data yang akurat bagi admin.
- **Dampak**: Tombol "Refresh Data" manual disediakan di dashboard untuk memotong (invalidate) cache Redis ketika admin membutuhkan data realtime seketika.

### K-P06-04: Katalog Plan, Dynamic Pricing Rules, dan Gateway Override
- **Keputusan**: Formulir konfigurasi paket (`PlanForm`) mendukung konfigurasi komprehensif dalam satu transaksi atomik:
  1. Metadata plan (`name`, `slug`, `description`, `is_active`).
  2. Periode siklus hari (`billing_period_days`, misal 30 hari).
  3. Harga dasar rupiah (`price_idr`, tipe integer rupiah sesuai K-27).
  4. Matriks Entitlement: Centang boolean fitur (`can_use_budgeting`, `can_export_report`, `can_use_voice`) dan limit numerik (`max_wallets`, `max_budgets`, `ai_monthly_token_quota`).
  5. Konfigurasi harga token AI dinamis: Base per 1k token, multiplier modalitas (chat 1.0x, vision 1.5x, voice 2.0x, embedding 0.5x), serta markup multiplier & persentase sesuai K-04.
  6. Payment Gateway Override: Opsi dropdown menentukan gateway khusus (`midtrans`, `xendit`, dll.) atau mewarisi pengaturan global default.
- **Rasional**: Seluruh konfigurasi monetisasi terpusat dalam satu entitas yang terstruktur. Sesuai K-10, override per-plan memungkinkan rute pembayaran berbeda tanpa menyentuh kode aplikasi.
- **Dampak**: Modifikasi plan tidak merusak langganan aktif pengguna (K-15); langganan berjalan tetap mengacu pada snapshot kontrak plan saat transaksi terjadi, kecuali terjadi renewal berikutnya.

### K-P06-05: Penanganan Intervensi Subscription (Extend, Force Downgrade, Cancel)
- **Keputusan**: Tindakan intervensi manual terhadap langganan pengguna dibatasi pada tiga operasi eksplisit:
  1. `Extend Period`: Menambah durasi siklus aktif sebanyak $N$ hari kalender, memperbarui `current_period_end`.
  2. `Force Downgrade`: Menurunkan tingkatan langganan pengguna ke paket Free secara langsung dengan menerapkan prinsip **Soft-Limit** (K-09). Data keuangan historis user TIDAK BOLEH dihapus; sistem hanya mengunci pembuatan resource baru yang melebihi kuota free.
  3. `Cancel Subscription`: Menghentikan auto-renewal atau membatalkan langganan seketika sesuai pilihan admin, menandai `status = cancelled`.
  Semua tindakan wajib menyertakan alasan administratif (*reason*) minimal 10 karakter dan dicatat ke `subscription_events` serta `audit_logs`.
- **Rasional**: Menjamin kepatuhan terhadap prinsip integritas data (K-09 & K-29) serta menyediakan jejak audit yang dapat dipertanggungjawabkan apabila terjadi sengketa dengan pelanggan.
- **Dampak**: Mencegah inkonsistensi status langganan antara backend dan cache sesi pengguna. Realtime RBAC berversi (K-14) akan segera memperbarui hak akses user dalam hitungan milidetik.

### K-P06-06: Antrean Verifikasi Pembayaran Manual dan Pencatatan Refund
- **Keputusan**: Transaksi pembayaran yang membutuhkan verifikasi manual (status `pending_review` atau transfer bank manual):
  1. Ditampilkan dalam antrean prioritas khusus pada `/super-admin/payments/review`.
  2. Dilengkapi dialog preview bukti transfer (gambar/PDF) dengan sanitasi tampilan.
  3. Menggunakan mekanisme penguncian pesimis/optimis pada tombol verifikasi untuk mencegah *double-action* saat dua admin membuka tiket yang sama.
  4. Aksi persetujuan (`Approve`) memicu state transition `payments` menjadi `settled`, menerbitkan invoice resmi, dan mengaktifkan subscription pengguna secara idempoten.
  5. Aksi penolakan (`Reject`) mewajibkan admin mengisi alasan penolakan yang diteruskan ke notifikasi user.
  6. Aksi pencatatan refund (`Record Refund`): Digunakan untuk sengketa/pengembalian dana, mencatat jumlah refund (integer rupiah), referensi mutasi bank, dan status pembayaran menjadi `refunded`.
- **Rasional**: Memastikan aliran kas masuk valid secara operasional sebelum sistem membuka akses berbayar kepada pengguna, serta mencegah duplikasi persetujuan.
- **Dampak**: Setiap status settlement pembayaran memicu event internal bus untuk rekonsiliasi billing tanpa mengganggu thread utama HTTP.

### K-P06-07: Penyesuaian Saldo Token Manual dan Jaminan Idempoten Append-Only
- **Keputusan**: Penyesuaian kuota token AI per-user melalui antarmuka Super Admin (`/super-admin/tokens`) WAJIB menggunakan pendekatan ledger append-only (K-28). Form penyesuaian mengirimkan:
  - `user_id`: UUID target.
  - `amount`: Integer token bertanda (+ untuk penambahan kompensasi, - untuk koreksi kuota berlebih).
  - `reason`: Alasan administratif.
  - `idempotency_key`: UUID v4 unik yang dihasilkan oleh klien frontend saat dialog dibuka.
  Backend mengeksekusi mutasi melalui `INSERT INTO token_ledger (..., event_type='manual_adjustment', ...)` dan memperbarui agregat `token_periods`.
- **Rasional**: Menghilangkan race condition dan risiko double-credit/double-debit apabila koneksi jaringan terputus saat request dikirimkan. Sesuai K-28, tabel token ledger tidak pernah menjalankan operasi `UPDATE` atau `DELETE`.
- **Dampak**: Saldo pengguna selalu dapat diverifikasi melalui penjumlahan riwayat mutasi buku besar token.

### K-P06-08: Panel Operasional AI Gateway (Circuit Breaker & Emergency Kill Switch)
- **Keputusan**: Antarmuka AI Gateway menyediakan visualisasi status operasional real-time per endpoint (Host URL, status kesehatan `HEALTHY` / `DEGRADED` / `DEAD`, latensi p95, error rate 5 menit terakhir, serta sisa kuota/biaya).
  Fitur kendali operasional mencakup:
  1. Tabel CRUD pool endpoint AI (9router lokal / upstream provider) beserta bobot (*weight*) dan prioritas.
  2. Matriks rute use-case (`chat`, `vision`, `voice`, `embedding`, `reasoning`) yang memetakan jenis tugas ke endpoint tertentu (K-18).
  3. Tombol Reset Circuit Breaker manual jika endpoint pulih sebelum ambang timeout otomatis.
  4. Emergency Kill Switch (Global & Per Modality): Saklar darurat yang mematikan rute AI seketika (mengembalikan fallback error ramah ke user) bila terjadi anomali tagihan atau kegagalan upstream fatal.
- **Rasional**: Sesuai K-17, AI Gateway adalah infrastruktur terpusat berisiko tinggi. Super admin membutuhkan visibilitas seketika dan rem darurat untuk melindungi integritas biaya operasional.
- **Dampak**: Perubahan rute atau penekanan kill switch disinkronkan ke Redis Pub/Sub sehingga seluruh instance backend menerapkan aturan baru dalam < 100ms tanpa restart aplikasi.

### K-P06-09: Penampil Jejak Audit (Audit Log Viewer) dan Sanitasi UU PDP
- **Keputusan**: Antarmuka Audit Log Viewer (`/super-admin/audit`) menampilkan seluruh rekaman dari tabel `audit_logs` secara terperinci:
  - Kolom: Waktu (Asia/Jakarta), Aktor (Super Admin Email & ID), Aksi (`CREATE`, `UPDATE`, `DELETE`, `EXECUTE`), Entitas (`plan`, `subscription`, `payment`, `token`, `ai_route`, `settings`), Alamat IP, User Agent.
  - Detail Viewer: Modal dialog yang menampilkan visualisasi *diff side-by-side* antara `before_state` (JSON) dan `after_state` (JSON).
  - Kepatuhan UU PDP No. 27/2022 (K-30): Payload audit log WAJIB disanitasi sebelum ditampilkan. Field sensitif seperti nomor kartu, token rahasia, kata sandi, atau data identitas privat wajib disamarkan (*masked*, contoh: `****-****-****-1234`).
- **Rasional**: Memenuhi standar kepatuhan regulasi perlindungan data pribadi dan asas akuntabilitas administratif (non-repudiation) tanpa mengekspos data kredensial di antarmuka internal.
- **Dampak**: Log bersifat read-only (append-only dari sisi database; tidak ada API endpoint yang mengizinkan update/delete terhadap tabel `audit_logs`).

### K-P06-10: Pengaturan Global Platform dan Dynamic Gateway Switching
- **Keputusan**: Halaman Settings Super Admin (`/super-admin/settings`) mengelola parameter global sistem:
  1. Default Payment Gateway (Midtrans, Xendit, Manual Transfer) dengan logika K-10: perubahan default hanya berlaku untuk checkout sesi baru; transaksi berjalan tetap terikat pada gateway inisial.
  2. Grace Period Renewal: Durasi hari toleransi pembayaran jatuh tempo sebelum subscription disuspend (default 3 hari).
  3. Pricing Rules Default: Parameter dasar perhitungan kuota token global.
  4. Feature Flags: Daftar saklar fitur platform (`enable_voice_input`, `enable_crypto_payment`, `maintenance_mode`) yang tersimpan di tabel `feature_flags`.
- **Rasional**: Memungkinkan penyesuaian strategi bisnis dan penanganan insiden pihak ketiga (misalnya saat salah satu payment gateway mengalami gangguan layanan) secara instan tanpa proses rilis kode baru.
- **Dampak**: Nilai pengaturan dicache di Redis dan diinvalasi seketika saat form pengaturan disimpan.

### K-P06-11: Spesifikasi Chart Visualisasi Interaktif (Recharts)
- **Keputusan**: Seluruh visualisasi grafik dibangun menggunakan Recharts yang dibungkus dalam komponen responsif Tailwind/CSS:
  1. Area/Line Chart: Tren Pendapatan MRR dan Pertumbuhan Pengguna (sumbu X: tanggal, sumbu Y: Rupiah/Jumlah User) dengan tooltip formatting IDR (`Intl.NumberFormat`).
  2. Bar/Column Chart: Konsumsi Token Berdasarkan Modality (Chat, Vision, Voice, Embedding) harian dengan filter rentang 7D, 30D, 90D.
  3. Donut/Pie Chart: Distribusi Metode Pembayaran dan Rasio Sukses/Gagal.
  4. Interaktivitas: Wajib mendukung hover tooltip informatif, click-to-filter (drill-down ke rute `<DataTable>` terkait dengan search params terisi), dan toggle legend untuk menyembunyikan seri data.
- **Rasional**: Memberikan kejelasan analitik visual yang intuitif bagi pengambil keputusan dengan performa render ringan di sisi klien.
- **Dampak**: Komponen grafik hanya memuat data agregat (maksimal beberapa ratus titik data), bukan raw event transaksi.

### K-P06-12: Strategi Deployment Zero-Downtime dan Backward Compatibility API
- **Keputusan**: Penyebaran antarmuka Super Admin dan handler backend mengikuti pola Strangler Fig (K-24) dan Zero Downtime (K-23):
  1. Handler backend baru diimplementasikan secara modular pada package `internal/admin` tanpa mengganggu endpoint user publik yang telah ada.
  2. Endpoint admin diproteksi oleh feature flag internal `feature_super_admin_portal`. Jika flag dinonaktifkan, endpoint merespons HTTP 503 Maintenance Mode dengan aman.
  3. Rilis frontend Next.js dapat berjalan berdampingan; jika backend versi baru belum aktif, navigasi menu admin menampilkan banner pemeliharaan secara graceful.
- **Rasional**: Memastikan rilis modul baru tidak pernah menimbulkan regresi pada fitur budgeting, pencatatan transaksi, atau AI chat pengguna reguler yang sedang aktif.
- **Dampak**: Proses migrasi dan rilis aman dijalankan kapan pun tanpa perlu menjadwalkan jendela pemadaman sistem (*maintenance window*).
