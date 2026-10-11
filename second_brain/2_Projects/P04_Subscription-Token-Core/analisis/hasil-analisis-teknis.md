# Hasil Analisis Teknis — P04: Subscription & Token Core

Dokumen ini mendokumentasikan keputusan teknis arsitektural (*Architectural Decision Records*) untuk Fase P04 (Subscription & Token Core) FinTrack v2, selaras dengan keputusan terkunci K-01 hingga K-30.

---

## Daftar Keputusan Teknis (K-P04-01 .. K-P04-20)

### K-P04-01: Skema Moneter dan Token Berbasis Integer Mutlak
- **Konteks:** Perhitungan nilai rupiah dan kuota token AI rentan terhadap galat pembulatan desimal (*floating-point inaccuracies*) pada kalkulasi berulang atau rekonsiliasi audit.
- **Keputusan:**
  - Seluruh nominal moneter disimpan dalam satuan terkecil Rupiah (IDR) bulat menggunakan tipe data PostgreSQL `BIGINT` (Go: `int64`).
  - Seluruh kuota dan mutasi token AI disimpan sebagai bilangan bulat murni (`BIGINT`, Go: `int64`).
  - Dilarang keras menggunakan tipe data `FLOAT`, `DOUBLE`, atau `REAL` pada kolom keuangan, kuota, dan harga di database maupun representasi Go struct.
- **Dampak:** Integritas keuangan konsisten 100%, rekonsiliasi audit ledger selalu seimbang tanpa selisih sen/pecahan.

### K-P04-02: Isolasi Transaksi Ledger Token (Reserve-Commit-Release)
- **Konteks:** Pemanggilan model AI memerlukan kuota token yang cukup, namun jumlah token aktual hanya diketahui setelah pemanggilan API selesai. Kegagalan sistem di tengah proses berisiko menimbulkan *double-spending* atau hilangnya kuota user tanpa layanan.
- **Keputusan:**
  - Mengadopsi pola 2-Phase Ledger Mutation:
    1. `Reserve`: Mengunci estimasi token maksimal dari kuota aktif (`reserved_tokens += est`). Menghasilkan `reservation_id` dengan status pending.
    2. `Commit`: Dipanggil saat streaming/respons AI selesai sukses dengan token aktual `act`. Menyesuaikan kuota (`reserved_tokens -= est`, `used_tokens += act`) dan mencatat mutasi `commit` di `token_ledger`.
    3. `Release`: Dipanggil jika pemanggilan AI gagal atau dibatalkan. Mengembalikan kuota yang terkunci (`reserved_tokens -= est`) dan mencatat mutasi `release` di `token_ledger`.
  - Concurrency Control menggunakan transaksi PostgreSQL dengan klausa `SELECT ... FOR UPDATE` pada baris `token_periods`.
- **Dampak:** Kuota tidak pernah bocor atau negatif meskipun terjadi crash di tengah eksekusi AI.

### K-P04-03: Idempotensi Mutasi Ledger Token
- **Konteks:** Retry jaringan atau pengiriman ulang request oleh klien/gateway dapat menyebabkan pemotongan kuota ganda (*double-deduction*).
- **Keputusan:**
  - Tabel `token_ledger` memiliki kolom `idempotency_key VARCHAR(64) NOT NULL UNIQUE`.
  - Format `idempotency_key` terstandarisasi:
    - Reserve: `res_{user_id}_{trace_id}_{attempt}`
    - Commit: `com_{reservation_id}`
    - Release: `rel_{reservation_id}`
  - Repository Go menerapkan penanganan *unique constraint violation* (error code `23505` PostgreSQL): jika kunci sudah ada, kembalikan entitas yang telah tercatat sebelumnya tanpa melempar kegagalan.
- **Dampak:** Eksekusi ulang aman dari pemotongan kuota ganda.

### K-P04-04: Siklus Kuota Harian Berbasis Zona Waktu Asia/Jakarta
- **Konteks:** Siklus kuota berbasis bulan kalender tidak adil pada bulan dengan durasi 28, 29, 30, atau 31 hari, serta menyulitkan perhitungan prorata. Sesuai K-05, siklus langganan dan kuota ditentukan per-HARI dalam TZ Asia/Jakarta (WIB, UTC+7).
- **Keputusan:**
  - Batas harian dihitung dari `00:00:00 WIB` sampai `23:59:59 WIB` (dikonversi ke UTC pada penyimpanan database: `17:00:00 UTC` hari sebelumnya hingga `16:59:59 UTC`).
  - Siklus langganan bulanan dihitung tepat $N$ hari kalender (default: 30 hari untuk siklus bulanan, 365 hari untuk tahunan) sejak jam aktivasi.
  - Tabel `token_periods` memiliki rentang `period_start TIMESTAMPTZ` dan `period_end TIMESTAMPTZ`.
  - Reset kuota terjadi otomatis saat masuk rentang baru; sisa kuota yang tidak terpakai hangus (*no rollover*, K-06).
- **Dampak:** Transparansi kuota pengguna jelas, perhitungan kuota deterministik terbebas dari anomali perbedaan hari dalam bulan.

### K-P04-05: State Machine Status Subscription
- **Konteks:** Siklus langganan harus mencakup berbagai skenario pembayaran, penagihan, masa tenggang (*grace period*), dan pembatalan.
- **Keputusan:**
  - Status langganan didefinisikan sebagai enum:
    - `trialing`: Masa uji coba aktif tanpa penagihan langsung.
    - `active`: Langganan aktif dan terbayar penuh.
    - `past_due`: Pembayaran perpanjangan gagal, sedang dalam antrean retry otomatis/reminder.
    - `grace`: Masa tenggang (3 hari kalender) setelah `past_due`, fitur Pro masih dapat diakses secara terbatas.
    - `canceled`: Pengguna memilih batal, langganan tetap aktif hingga `current_period_end`.
    - `expired`: Periode berakhir dan masa tenggang habis; akun kembali ke plan Free.
  - Setiap perubahan status WAJIB mencatat event ke tabel append-only `subscription_events` lengkap dengan snapshot state sebelum dan sesudah.
- **Dampak:** Jejak audit status langganan lengkap, transisi status terprediksi dan konsisten.

### K-P04-06: Arsitektur Entitlement: Pemisahan Fitur Boolean dan Limit Numerik
- **Konteks:** Paket langganan membutuhkan pembatasan akses berupa saklar fitur biner (misal: "Bisa Export PDF", "Bisa Bank Sync") dan kuota terukur (misal: "Maksimal 5 Akun Dompet", "100.000 Token AI/hari").
- **Keputusan:**
  - Dibagi menjadi 2 tabel berelasi ke `plans`:
    - `plan_entitlements`: Menyimpan `(plan_id, feature_key, is_enabled)`. `feature_key` bertipe string konstan (`export_csv`, `ai_chat`, `bank_sync`, `ocr_receipt`, `advanced_analytics`).
    - `plan_limits`: Menyimpan `(plan_id, limit_key, limit_value)`. `limit_key` bertipe string (`max_wallets`, `max_categories`, `daily_ai_tokens`, `daily_ocr_scans`), `limit_value` bertipe integer (`-1` merepresentasikan *unlimited*).
- **Dampak:** Fleksibel dalam penambahan fitur dan kuota tanpa perlu migrasi kolom tabel `plans`.

### K-P04-07: Versioned Cache Entitlement Multi-Level
- **Konteks:** Pemeriksaan hak akses dilakukan di setiap request HTTP ke backend. Direct query ke PostgreSQL di setiap request akan membebani database utama. Penyimpanan hak akses dalam payload JWT ditolak (K-14) karena JWT tidak dapat diinvalidasi seketika tanpa mekanisme refresh token yang lambat.
- **Keputusan:**
  - Entitlement diambil per-request dengan mekanisme Cache Berversi:
    - Format Key Redis: `entitlement:u:{user_id}:v:{version}` dengan TTL 24 jam.
    - Counter Versi Pengguna: `entitlement:ver:{user_id}` bertipe integer.
    - L1 Cache: In-memory go-cache / LRU di setiap instance Go (TTL 60 detik).
    - L2 Cache: Redis Cluster / Standalone (TTL 24 jam).
  - Mekanisme Invalidasi:
    - Saat status subscription atau plan user berubah, backend mengeksekusi `INCR entitlement:ver:{user_id}` di Redis dan mempublikasikan event ke channel Redis Pub/Sub `entitlement:invalidations`.
    - Semua instance Go yang mendengarkan pub/sub menghapus L1 cache untuk `user_id` tersebut.
    - Request berikutnya membaca versi baru dan melakukan *cache warming* dari PostgreSQL.
- **Dampak:** Zero logout/relogin bagi user, latensi pengecekan hak akses < 1ms, perubahan plan aktif dalam < 50ms di seluruh node backend.

### K-P04-08: Penegakan Hak Akses Bersifat Backend-Otoritatif
- **Konteks:** Frontend (Next.js) dapat direkayasa oleh pengguna tingkat lanjut. Sesuai K-13, hak akses wajib diverifikasi di sisi backend.
- **Keputusan:**
  - Penegakan dilakukan melalui middleware Gin HTTP di layer router:
    - `RequireFeature(featureKey string)`: Memastikan boolean feature bernilai `true`.
    - `WithinLimit(limitKey string, countFn UsageCounter)`: Memastikan pemakaian saat ini < `limit_value` (atau `limit_value == -1`).
  - Frontend hanya menerima DTO dari endpoint `GET /api/v1/me/entitlements` untuk menyembunyikan/menonaktifkan komponen UI.
- **Dampak:** Sistem tahan terhadap serangan manipulasi antarmuka klien.

### K-P04-09: Kebijakan Downgrade: Soft-Limit Tanpa Penghapusan Data
- **Konteks:** Saat pengguna turun kelas langganan (misal: Business ke Free), jumlah entitas eksisting (misal: 10 dompet aktif) mungkin melebihi batas paket baru (Free: 2 dompet).
- **Keputusan:**
  - Diterapkan prinsip *Soft-Limit* (K-09):
    - Tidak ada data transaksi, dompet, atau kategori yang dihapus dari basis data.
    - Pengguna tetap dapat melihat (*read-only*) seluruh data historis dan dompet yang ada.
    - Operasi penambahan data baru (*write/create*) diblokir oleh middleware `WithinLimit` sampai jumlah entitas berada di bawah batas baru atau pengguna melakukan upgrade kembali.
- **Dampak:** Data privasi finansial user aman, reputasi produk terjaga, friksi emosional pengguna minimal.

### K-P04-10: Kebijakan Upgrade Instan Prorata
- **Konteks:** Pengguna yang melakukan upgrade di pertengahan siklus berhak mendapatkan akses paket baru seketika dengan penyesuaian biaya yang adil.
- **Keputusan:**
  - Upgrade langsung aktif seketika (*instant provision*):
    - Sisa hari pada paket lama dihitung sebagai kredit prorata:
      $$\text{Credit} = \left\lfloor \frac{\text{Hari Tersisa}}{\text{Total Hari Siklus}} \times \text{Harga Paket Lama} \right\rfloor$$
    - Tagihan baru adalah harga paket baru dikurangi sisa kredit paket lama (dibulatkan ke bawah ke integer IDR).
    - Kuota token harian langsung dinaikkan ke kuota paket baru untuk sisa hari berjalan.
- **Dampak:** Pengalaman pengguna instan, tidak ada kelebihan pembayaran, konversi upgrade mulus.

### K-P04-11: Pricing Engine Dinamis untuk Biaya Token AI
- **Konteks:** Konsumsi AI melibatkan model teks biasa, multimodal vision, pemrosesan audio/voice, dan embedding dengan biaya modal yang bervariasi.
- **Keputusan:**
  - Tabel `pricing_rules` menyimpan parameter pricing global dan per-modality:
    - `base_per_1k_idr`: Biaya dasar per 1.000 token dalam Rupiah bulat (e.g. Rp 300).
    - `markup_multiplier`: Faktor pengali markup (e.g. 1.25 untuk margin 25%).
    - `markup_percent`: Tambahan markup persentase integer jika diperlukan.
    - `modality_multipliers`: JSONB menyimpan bobot per modalitas:
      `{"chat": 1.0, "vision": 2.5, "voice": 3.0, "embedding": 0.5, "reasoning": 2.0}`.
  - Formula Biaya Rupiah per transaksi:
    $$\text{Biaya IDR} = \left\lceil \frac{\text{Tokens}}{1000} \times \text{base\_per\_1k\_idr} \times \text{markup\_multiplier} \times \text{modality\_multiplier} \right\rceil$$
- **Dampak:** Super admin memiliki kontrol margin laba menyeluruh tanpa perlu deploy ulang kode backend.

### K-P04-12: Desain Endpoint `/api/v1/me/entitlements`
- **Konteks:** Frontend Next.js memerlukan payload yang ringkas, deterministik, dan mudah dikonsumsi oleh TanStack Query v5 untuk mengatur status visual komponen dan navigasi.
- **Keputusan:**
  - Endpoint mengembalikan DTO:
    ```json
    {
      "plan": {
        "code": "pro",
        "name": "Pro Tier",
        "status": "active",
        "current_period_end": "2026-11-10T23:59:59+07:00",
        "cancel_at_period_end": false
      },
      "features": {
        "export_csv": true,
        "ai_chat": true,
        "bank_sync": true,
        "ocr_receipt": true,
        "advanced_analytics": false
      },
      "limits": {
        "max_wallets": { "limit": 10, "used": 4, "remaining": 6 },
        "daily_ai_tokens": { "limit": 250000, "used": 45000, "remaining": 205000 }
      },
      "version": 4
    }
    ```
- **Dampak:** Rendering kondisional komponen UI di client-side konsisten dengan aturan otorisasi server.

### K-P04-13: Struktur Migrasi Database Expand/Contract
- **Konteks:** Database produksi PostgreSQL 17 harus menerapkan perubahan skema tanpa penghentian layanan (*zero downtime*).
- **Keputusan:**
  - Migrasi `0007_subscription_token_core.sql` hanya menambahkan tabel-tabel baru tanpa mengubah kolom tabel pengguna yang sedang berjalan secara destruktif.
  - Penambahan foreign key ke tabel `users(id)` menggunakan klausa `ON DELETE RESTRICT` untuk mencegah cascade delete yang tidak disengaja.
  - Index komposit dibuat secara `CONCURRENTLY` jika dioperasikan pada tabel transaksi besar.
- **Dampak:** Tidak ada table-lock berdurasi panjang, aplikasi versi lama dan baru dapat berjalan berdampingan selama *rolling deployment*.

### K-P04-14: Audit Trail Append-Only untuk Mutasi Billing & Subscription
- **Konteks:** Kepatuhan regulasi UU PDP No. 27/2022 dan kebutuhan resolusi sengketa penagihan mewajibkan pencatatan historis yang tidak dapat diubah (*immutable audit logs*).
- **Keputusan:**
  - Tabel `subscription_events` dan `token_ledger` tidak memiliki operasi `UPDATE` atau `DELETE`. Hak akses PostgreSQL untuk role aplikasi dibatasi hanya `INSERT` dan `SELECT` pada kedua tabel ini.
  - Segala bentuk koreksi saldo token dicatat sebagai mutasi baru dengan `entry_type = 'adjust'` dan referensi admin penanggung jawab.
- **Dampak:** Non-repudiation terjamin, investigasi anomali transaksi dapat diaudit secara forensik.

### K-P04-15: Default Plan Provisioning untuk Pengguna Baru
- **Konteks:** Pengguna baru yang mendaftar harus langsung mendapatkan akses fitur dasar tanpa intervensi manual super admin.
- **Keputusan:**
  - Hook pendaftaran user memicu pembuatan entri `subscriptions` baru dengan `plan_id` menunjuk ke paket `free`, `status = 'active'`, dan `current_period_end = NULL` (tidak kedaluwarsa).
  - Inisialisasi entri `token_periods` untuk hari berjalan otomatis dibuat pada transaksi pertama kali pemanggilan AI atau saat registrasi.
- **Dampak:** Onboarding pengguna otomatis, tidak terjadi error *nil pointer* saat pengecekan entitlement user baru.

### K-P04-16: Pengelolaan Kegagalan AI Gateway dalam Reservasi Token
- **Konteks:** Jika AI Gateway mengalami timeout (504) atau kegagalan internal (500/502) setelah reservasi kuota dilakukan, kuota pengguna tidak boleh tergantung selamanya dalam status `reserved`.
- **Keputusan:**
  - Diterapkan mekanisme Auto-Release Timeout:
    - Service layer membungkus pemanggilan AI dalam blok `defer` Go. Jika terjadi panic atau error kembalian dari AI Gateway, fungsi `Release` otomatis dipanggil.
    - Ditambahkan background cleaner worker yang berjalan tiap 15 menit untuk membatalkan (*auto-release*) reservasi dengan status `pending` yang berusia lebih dari 5 menit.
- **Dampak:** Kuota pengguna tidak terkunci secara permanen akibat gangguan jaringan sesaat.

### K-P04-17: Mekanisme Grace Period Penagihan
- **Konteks:** Kegagalan kartu debit/kredit atau keterlambatan transfer manual tidak boleh langsung memutus akses pengguna setia secara mendadak.
- **Keputusan:**
  - Saat tanggal `current_period_end` tercapai dan perpanjangan otomatis belum berhasil, status berubah menjadi `past_due` dan masuk ke masa `grace` selama 3 hari kerja (72 jam).
  - Selama masa `grace`, fitur Pro tetap aktif, namun di antarmuka web dimunculkan banner peringatan urgensi pembayaran.
  - Jika setelah 72 jam pembayaran belum diterima, worker beralih status ke `expired` dan mendowngrade hak akses ke Free.
- **Dampak:** Mengurangi churn akibat insiden teknis perbankan, meningkatkan retensi pengguna.

### K-P04-18: Integrasi Metrik Observabilitas (Prometheus & OpenTelemetry)
- **Konteks:** Operasional sistem langganan dan konsumsi token membutuhkan pemantauan beban kuota secara real-time.
- **Keputusan:**
  - Modul billing mengekspos metrik:
    - Counter: `billing_token_reservations_total{status="success|failed"}`
    - Counter: `billing_token_commits_total{modality="chat|vision|voice|embedding"}`
    - Histogram: `entitlement_middleware_duration_seconds{cache="hit|miss"}`
    - Gauge: `subscription_active_users_total{plan="free|pro|business"}`
- **Dampak:** Deteksi anomali kuota, lonjakan biaya token, dan performa cache dapat dipantau langsung pada Grafana.

### K-P04-19: Isolasi Kegagalan Redis (Graceful Degradation)
- **Konteks:** Jika layanan Redis mengalami gangguan atau downtime sesaat, aplikasi backend tidak boleh mengalami crash total pada setiap endpoint yang memerlukan verifikasi hak akses.
- **Keputusan:**
  - Cache manager mengimplementasikan circuit breaker/fallback:
    - Jika pembacaan Redis timeout (> 50ms) atau mengembalikan connection error, sistem langsung membaca PostgreSQL sebagai fallback (*direct query*).
    - Kegagalan publish ke Redis Pub/Sub dicatat dalam logger dengan level `ERROR`, dan L1 cache lokal tetap diinvalir di instance saat ini.
- **Dampak:** Ketersediaan tinggi (*high availability*) sistem FinTrack tetap terjaga meskipun terjadi degradasi infrastruktur cache.

### K-P04-20: Pembatasan Concurrency dan Lock Contention pada Token Periods
- **Konteks:** Pengguna yang menjalankan beberapa request AI secara paralel (misal: chat multi-tab) dapat memicu *lock contention* tinggi pada baris `token_periods` pengguna tersebut.
- **Keputusan:**
  - Transaksi `Reserve` membatasi durasi lock DB sekecil mungkin: lock baris dibuka segera setelah pembaruan `reserved_tokens` dan pencatatan entri ledger `reserve` selesai.
  - Pemanggilan AI dilakukan di luar konteks transaksi database DB.
  - Transaksi `Commit`/`Release` membuka transaksi terpisah yang singkat saat respons AI diterima.
- **Dampak:** Menghilangkan risiko transaksi database berdurasi panjang (*long-running transactions*) yang menahan koneksi pool PostgreSQL selama streaming AI.
