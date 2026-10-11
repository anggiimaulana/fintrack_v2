# Task Breakdown — Fase P06 (Super Admin Dashboard)

Daftar tugas implementasi mendetail untuk fase P06 dengan format identitas `TASK-P06-NNN`. Terdiri dari 20 tugas terstruktur mencakup frontend, backend, keamanan, audit, dan pengujian.

---

## Daftar Tugas Rinci

### TASK-P06-001: Backend Super Admin Route Group & Middleware Otorisasi
- **Deskripsi**: Buat namespace routing `/api/v1/admin/*` di Gin engine. Implementasikan middleware `RequireSuperAdmin()` yang memeriksa apakah pengguna memiliki role `super_admin` dari token/session terverifikasi serta validasi versi hak akses via cache Redis.
- **File Terkait**: `backend/internal/api/routes.go`, `backend/internal/middleware/auth.go`, `backend/internal/admin/handler.go`
- **Acceptance Criteria**: Request dari pengguna dengan role `user` atau tanpa autentikasi wajib ditolak dengan status HTTP 403 Forbidden atau HTTP 401 Unauthorized. Unit test otorisasi lolos 100%.

### TASK-P06-002: Frontend Super Admin Layout, Auth Guard & Navigasi Sidebar
- **Deskripsi**: Bangun layout dasar Next.js di `app/(dashboard)/super-admin/layout.tsx`. Implementasikan pemeriksaan session di server component untuk memastikan role `super_admin` (redirect ke `/dashboard` bila unauthorized). Buat komponen sidebar admin dengan menu navigasi lengkap, badge notifikasi (pending payments & alert circuit breaker), dan breadcrumbs.
- **File Terkait**: `frontend/app/(dashboard)/super-admin/layout.tsx`, `frontend/components/admin/sidebar.tsx`, `frontend/components/admin/breadcrumbs.tsx`
- **Acceptance Criteria**: Akses langsung ke URL admin oleh non-admin dialihkan seketika tanpa flash screen; sidebar menampilkan seluruh 8 modul navigasi secara responsif.

### TASK-P06-003: Komponen Reusable `<DataTable>` Berbasis TanStack Table v8
- **Deskripsi**: Kembangkan komponen data-table modular di `frontend/components/data-table/data-table.tsx` dengan integrasi TanStack Table v8. Fitur wajib: server-side pagination (default 20 baris), multi-column sorting, input search dengan debounce 300ms, facet filtering, tombol column visibility, dan penomoran baris absolut `(page-1)*20 + idx + 1`.
- **File Terkait**: `frontend/components/data-table/data-table.tsx`, `frontend/components/data-table/pagination.tsx`, `frontend/components/data-table/toolbar.tsx`
- **Acceptance Criteria**: Komponen menerima generic data `<TData, TValue>`, render tabel tanpa layout shift, dan lulus rendering pada berbagai ukuran layar.

### TASK-P06-004: Sinkronisasi Status `<DataTable>` dengan URL Search Params
- **Deskripsi**: Hubungkan state tabel (page, per_page, sort, order, q, filter) secara dua arah ke URL Search Params Next.js 16 menggunakan custom hook `useDataTableState()`. Perubahan filter atau page otomatis memperbarui URL tanpa re-mount seluruh halaman.
- **File Terkait**: `frontend/components/data-table/use-data-table-state.ts`, `frontend/components/data-table/data-table.tsx`
- **Acceptance Criteria**: URL dapat dibagikan atau dibookmark; menekan refresh browser mempertahankan halaman, kata kunci pencarian, dan opsi filter yang sama persis.

### TASK-P06-005: Backend Endpoint Agregasi Metrik & KPI Dashboard
- **Deskripsi**: Implementasikan endpoint `GET /api/v1/admin/metrics/overview` dan `GET /api/v1/admin/metrics/timeseries`. Agregasikan data: User Aktif (30D), MRR (dalam Rupiah), Penggunaan Token (24H), Payment Success Rate (%), Status Endpoint AI, Jumlah Failover, dan Antrean Review Manual. Terapkan caching Redis dengan key `admin:metrics:overview` TTL 60 detik.
- **File Terkait**: `backend/internal/admin/metrics_handler.go`, `backend/internal/admin/metrics_service.go`, `backend/internal/admin/repository.go`
- **Acceptance Criteria**: Waktu respon endpoint < 200ms; cache Redis terpakai dengan benar; tersedia parameter query bypass cache untuk tombol refresh admin.

### TASK-P06-006: Antarmuka Dashboard Ringkasan & Kartu KPI
- **Deskripsi**: Bangun halaman utama Super Admin di `app/(dashboard)/super-admin/page.tsx`. Tampilkan 6 kartu KPI utama dengan indikator persentase pertumbuhan dibanding periode sebelumnya, badge status kesehatan sistem, serta widget daftar antrean tugas darurat (pending review payments, degraded AI endpoints).
- **File Terkait**: `frontend/app/(dashboard)/super-admin/page.tsx`, `frontend/components/admin/metric-cards.tsx`, `frontend/components/admin/quick-actions.tsx`
- **Acceptance Criteria**: Data diambil via TanStack Query dengan revalidasi otomatis; skeleton loading state ditampilkan secara mulus sebelum data siap.

### TASK-P06-007: Visualisasi Analitik Interaktif Berbasis Recharts
- **Deskripsi**: Implementasikan grafik analitik interaktif di dashboard utama: AreaChart tren MRR & Pendapatan Harian, BarChart breakdown konsumsi token AI per-modality, Pie/Donut Chart rasio metode pembayaran. Sertakan custom tooltip dengan pemformatan IDR, toggle legend, dan event click-to-filter drill-down ke tabel transaksi terkait.
- **File Terkait**: `frontend/components/admin/charts/revenue-area-chart.tsx`, `frontend/components/admin/charts/token-bar-chart.tsx`, `frontend/components/admin/charts/payment-pie-chart.tsx`
- **Acceptance Criteria**: Grafik responsif terhadap perubahan lebar layar; tooltip menampilkan angka moneter terformat (`Rp XX.XXX.XXX`); klik pada elemen grafik mengarahkan admin ke halaman detail dengan query params terkait.

### TASK-P06-008: Backend Plan & Entitlement Admin CRUD Handler
- **Deskripsi**: Buat endpoint administratif untuk katalog paket: `GET /api/v1/admin/plans`, `POST /api/v1/admin/plans`, `PUT /api/v1/admin/plans/:id`, dan `DELETE /api/v1/admin/plans/:id` (soft-archive `is_active=false`). Kelola transaksi atomik untuk metadata plan, matriks entitlement boolean, batasan kuota numerik, dynamic pricing rules, dan gateway override. Catat seluruh perubahan ke `audit_logs`.
- **File Terkait**: `backend/internal/admin/plan_handler.go`, `backend/internal/admin/plan_service.go`
- **Acceptance Criteria**: Validasi payload ketat; harga wajib integer rupiah; penghapusan plan yang sedang digunakan user aktif ditolak dengan pesan error yang jelas.

### TASK-P06-009: Antarmuka Manajemen Katalog Plan & Form Konfigurasi
- **Deskripsi**: Bangun halaman `/super-admin/plans` menggunakan `<DataTable>`. Implementasikan modal formulir wizard untuk Create/Edit Plan: tab metadata, tab centang entitlement (boolean flags), tab batas numerik kuota (wallets, budgets, monthly tokens), tab dynamic pricing & AI token multiplier, serta dropdown gateway override.
- **File Terkait**: `frontend/app/(dashboard)/super-admin/plans/page.tsx`, `frontend/components/admin/plans/plan-form-dialog.tsx`, `frontend/components/admin/plans/columns.tsx`
- **Acceptance Criteria**: Admin dapat menambah atau mengubah paket; form memvalidasi input input angka non-negatif; status aktif/nonaktif dapat di-toggle langsung dari tabel.

### TASK-P06-010: Backend Subscription Management & Intervensi Manual
- **Deskripsi**: Buat endpoint admin untuk mengelola langganan: `GET /api/v1/admin/subscriptions` (dengan filter user, plan, status), `POST /api/v1/admin/subscriptions/:id/extend` (menambah $N$ hari siklus), `POST /api/v1/admin/subscriptions/:id/force-downgrade` (menerapkan soft-limit ke paket free), dan `POST /api/v1/admin/subscriptions/:id/cancel`. Setiap mutasi wajib menyimpan alasan ke `subscription_events` dan `audit_logs`.
- **File Terkait**: `backend/internal/admin/subscription_handler.go`, `backend/internal/admin/subscription_service.go`
- **Acceptance Criteria**: Sesuai K-09, force downgrade TIDAK menghapus baris data keuangan user; event tercatat akurat; hak akses user langsung terupdate di cache sesi.

### TASK-P06-011: Antarmuka Monitoring & Intervensi Subscription
- **Deskripsi**: Bangun halaman `/super-admin/subscriptions` menggunakan `<DataTable>`. Sediakan kolom: Pengguna, Paket, Status, Awal Siklus, Akhir Siklus, Auto-Renew, dan Aksi. Buat action menu dropdown untuk setiap baris dengan opsi dialog: `Perpanjang Periode (Extend)`, `Turunkan Paket (Force Downgrade)`, dan `Batalkan Langganan (Cancel)`.
- **File Terkait**: `frontend/app/(dashboard)/super-admin/subscriptions/page.tsx`, `frontend/components/admin/subscriptions/action-dialogs.tsx`, `frontend/components/admin/subscriptions/columns.tsx`
- **Acceptance Criteria**: Dialog mewajibkan pengisian alasan administratif; konfirmasi destruktif menampilkan peringatan merah; pembaruan data terjadi secara optimis di UI via TanStack Query.

### TASK-P06-012: Backend Payment Review, Settlement & Refund Handler
- **Deskripsi**: Kembangkan endpoint review pembayaran admin: `GET /api/v1/admin/payments/review` (antrean pending review), `POST /api/v1/admin/payments/:id/approve` (transisi ke settled, trigger aktivasi langganan, terbitkan invoice), `POST /api/v1/admin/payments/:id/reject` (transisi ke failed dengan alasan), dan `POST /api/v1/admin/payments/:id/refund` (mencatat refund parsial/penuh dalam integer rupiah).
- **File Terkait**: `backend/internal/admin/payment_handler.go`, `backend/internal/admin/payment_service.go`
- **Acceptance Criteria**: Gunakan row-locking untuk mencegah dua admin memproses transaksi yang sama secara paralel; event refund dicatat rapi tanpa menghapus record pembayaran asli.

### TASK-P06-013: Antarmuka Antrean Verifikasi Pembayaran & Dialog Refund
- **Deskripsi**: Bangun antarmuka `/super-admin/payments/review` dan riwayat `/super-admin/payments`. Fitur: tampilan antrean prioritas, dialog modal pratinjau bukti transfer (gambar/PDF viewer dengan fitur zoom/rotate), tombol cepat Approve/Reject dengan konfirmasi dialog, dan form pencatatan Refund dengan input nominal IDR dan nomor referensi bank.
- **File Terkait**: `frontend/app/(dashboard)/super-admin/payments/page.tsx`, `frontend/app/(dashboard)/super-admin/payments/review/page.tsx`, `frontend/components/admin/payments/proof-preview-modal.tsx`, `frontend/components/admin/payments/refund-dialog.tsx`
- **Acceptance Criteria**: Bukti bayar dapat dilihat dengan jelas; tombol tindakan terproteksi status loading untuk mencegah klik berulang; data ter-update seketika setelah persetujuan.

### TASK-P06-014: Backend Token Ledger Admin Handler & Idempotent Adjustment
- **Deskripsi**: Implementasikan endpoint manajemen token: `GET /api/v1/admin/tokens/users` (daftar ringkasan saldo token per user), `GET /api/v1/admin/tokens/ledger` (log riwayat ledger transaksi token), dan `POST /api/v1/admin/tokens/adjust` (menambah/mengurangi kuota token secara manual). Wajib memvalidasi header/payload `idempotency_key` (UUID unik) dan menulis baris append-only baru ke `token_ledger` (K-28).
- **File Terkait**: `backend/internal/admin/token_handler.go`, `backend/internal/admin/token_service.go`
- **Acceptance Criteria**: Operasi penyesuaian tidak pernah mengupdate baris lama; duplikasi request dengan idempotency_key yang sama mengembalikan respon tersimpan tanpa mutasi ganda.

### TASK-P06-015: Antarmuka Saldo Token Pengguna & Form Penyesuaian Manual
- **Deskripsi**: Bangun halaman `/super-admin/tokens` yang menampilkan dua tab: Tab Saldo Token Pengguna dan Tab Riwayat Ledger Global. Implementasikan modal formulir `Adjust Token Quota` dengan input: User Picker, Tipe Penyesuaian (Credit/Debit), Jumlah Token (integer), Alasan Administratif, serta generasi otomatis UUID `idempotency_key` pada saat modal dibuka.
- **File Terkait**: `frontend/app/(dashboard)/super-admin/tokens/page.tsx`, `frontend/components/admin/tokens/adjust-token-modal.tsx`, `frontend/components/admin/tokens/token-columns.tsx`
- **Acceptance Criteria**: Penyesuaian saldo langsung tercermin pada tabel saldo pengguna dan ledger; validasi mencegah pengurangan saldo melampaui batas wajar.

### TASK-P06-016: Backend AI Gateway Operations & Kill Switch Handler
- **Deskripsi**: Buat endpoint manajemen operasional AI Gateway: `GET /api/v1/admin/ai-gateway/endpoints` (daftar endpoint 9router/upstream dengan metrik health/latency/error), `POST /api/v1/admin/ai-gateway/endpoints` (CRUD pool endpoint), `PUT /api/v1/admin/ai-gateway/routes` (pemetaan rute use-case chat/vision/voice/embedding), `POST /api/v1/admin/ai-gateway/circuit-breaker/reset` (reset circuit breaker), dan `POST /api/v1/admin/ai-gateway/kill-switch` (toggle saklar darurat global atau per modalitas).
- **File Terkait**: `backend/internal/admin/aigateway_handler.go`, `backend/internal/admin/aigateway_service.go`
- **Acceptance Criteria**: Status kill switch dan pembaruan rute dipublikasikan via Redis Pub/Sub ke seluruh instance backend dalam waktu < 100ms.

### TASK-P06-017: Antarmuka Pusat Kendali AI Gateway & Monitoring Realtime
- **Deskripsi**: Bangun antarmuka `/super-admin/ai-gateway`. Tampilkan: (1) Panel status darurat dengan tombol Emergency Kill Switch (Global, Chat, Vision, Voice, Embedding) berkode warna merah, (2) Tabel Pool Endpoint dengan indikator status lampu (Hijau/Kuning/Merah), latensi p95, dan tombol reset circuit breaker, (3) Editor Matriks Routing Use-Case untuk mengubah target endpoint tanpa deploy.
- **File Terkait**: `frontend/app/(dashboard)/super-admin/ai-gateway/page.tsx`, `frontend/components/admin/ai-gateway/kill-switch-panel.tsx`, `frontend/components/admin/ai-gateway/endpoint-table.tsx`, `frontend/components/admin/ai-gateway/route-matrix.tsx`
- **Acceptance Criteria**: Indikator status terupdate secara berkala; aksi kill switch mewajibkan pengetikan teks konfirmasi ("MATIKAN") untuk mencegah klik tidak sengaja.

### TASK-P06-018: Backend Audit Log Query Handler & Sanitasi Data UU PDP
- **Deskripsi**: Implementasikan endpoint `GET /api/v1/admin/audit-logs` dengan parameter pencarian: waktu (`start_date`, `end_date`), aktor (`user_id`, `email`), entitas (`plan`, `subscription`, `payment`, `token`, `ai_route`, `settings`), dan tipe aksi (`CREATE`, `UPDATE`, `DELETE`, `EXECUTE`). Wajib melakukan masking pada field sensitif (token API, nomor kartu, password, PII rahasia) sesuai UU PDP No. 27/2022 sebelum payload dikirimkan.
- **File Terkait**: `backend/internal/admin/audit_handler.go`, `backend/internal/admin/audit_sanitizer.go`, `backend/internal/audit/repository.go`
- **Acceptance Criteria**: Query terindeks waktu dan entitas; data sensitif tersanitasi 100%; tidak ada endpoint backend untuk update atau delete record audit log (tabel read-only di level admin).

### TASK-P06-019: Antarmuka Audit Log Viewer dengan Diff JSON Inspeksi
- **Deskripsi**: Bangun halaman `/super-admin/audit` menggunakan `<DataTable>`. Sediakan filter rentang tanggal, filter aktor, dan filter entitas. Sediakan modal inspeksi detail rekaman audit yang menampilkan visualisasi perbedaan (*diff viewer*) antara `before_state` dan `after_state` dengan penandaan warna hijau (penambahan) dan merah (penghapusan).
- **File Terkait**: `frontend/app/(dashboard)/super-admin/audit/page.tsx`, `frontend/components/admin/audit/audit-diff-dialog.tsx`, `frontend/components/admin/audit/columns.tsx`
- **Acceptance Criteria**: Admin dapat meneliti setiap perubahan konfigurasi sistem hingga ke tingkat properti JSON terkecil dengan mudah dan cepat.

### TASK-P06-020: Antarmuka Pengaturan Sistem Global, Feature Flags & E2E Validation
- **Deskripsi**: Bangun halaman `/super-admin/settings`. Kelola konfigurasi: Payment Gateway Default Switcher (K-10), Grace Period Days, Default Pricing Rules Token, dan Matriks Feature Flags. Tulis pengujian E2E/integrasi komprehensif yang memvalidasi bahwa seluruh flow admin (login, pembatasan akses, operasi tabel, dan audit logging) berjalan tanpa galat.
- **File Terkait**: `frontend/app/(dashboard)/super-admin/settings/page.tsx`, `frontend/components/admin/settings/feature-flags-card.tsx`, `frontend/components/admin/settings/gateway-config-card.tsx`, `backend/tests/admin_e2e_test.go`
- **Acceptance Criteria**: Seluruh perubahan pengaturan tersimpan di database dan cache Redis; tes E2E skenario super admin lulus 100%.
