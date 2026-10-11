# Rencana Implementasi: Fase P06 — Super Admin Dashboard

| Metadata | Deskripsi |
|---|---|
| **Fase** | P06 — Super Admin Dashboard |
| **Versi Dokumen** | 1.0.0 |
| **Tanggal Rilis** | 2026-10-11 |
| **Prasyarat** | P01 (Selesai), P02 (Core Audit & Flags), P03 (AI Gateway), P04 (Subscription Core), P05 (Payments & Invoices) |
| **Master Plan** | `../../IMPLEMENTATION-PLAN.md` |
| **Status Eksekusi** | Siap Dieksekusi — Belum Ada Kode Ditulis (Semua Task ⬜) |

---

## 1. Tujuan & Definition of Done (DoD)

### 1.1 Tujuan
Membangun portal kendali terpadu Super Admin FinTrack v2 berbasis Next.js 16 (`frontend/app/(dashboard)/super-admin/*`) dan Go+Gin (`backend/internal/admin/*`). Modul ini memberikan visibilitas penuh dan intervensi administratif aman terhadap:
1. Pemantauan metrik bisnis & kesehatan sistem (MRR, token, gateway status, alert failover).
2. Manajemen siklus katalog paket (Plan & Entitlement) tanpa rilis ulang kode.
3. Monitoring & penyesuaian langganan pengguna (Extend, Force Downgrade soft-limit, Cancel).
4. Verifikasi antrean pembayaran manual, review bukti transfer, mitigasi race condition antar-admin, dan pencatatan refund.
5. Audit saldo token pengguna dan penyesuaian kuota manual berbasis append-only ledger dengan idempotensi.
6. Pusat operasi AI Gateway: monitoring latensi pool endpoint 9router, reset circuit breaker, dan Emergency Kill Switch bertingkat.
7. Penampil jejak audit (*audit log viewer*) tersanitasi UU PDP No. 27/2022 dengan fitur visual diff JSON.
8. Konfigurasi sistem global (default payment gateway switcher, grace period, feature flags).

### 1.2 Definition of Done (Checklist)
- [ ] Otorisasi ketat backend via middleware `RequireSuperAdmin()` dan validasi versi hak akses via cache Redis berversi (K-14). Akses non-admin ditolak seketika (HTTP 401/403).
- [ ] Guard server component sisi frontend di `app/(dashboard)/super-admin/layout.tsx` mengalihkan pengguna non-admin ke `/dashboard` tanpa flash screen.
- [ ] Komponen generic reusable `<DataTable>` berbasis TanStack Table v8 terpasang di seluruh 8 tabel admin dengan penomoran absolut `(page - 1) * per_page + idx + 1`, baris berkursur pointer (`cursor-pointer`), debounce search 300ms, multi-column sorting, dan facet filters.
- [ ] Sinkronisasi dua arah penuh antara state `<DataTable>` dan URL Search Params Next.js via hook `useDataTableState()` (URL dapat dibookmark dan di-refresh tanpa kehilangan status filter).
- [ ] Visualisasi Recharts interaktif: AreaChart MRR, BarChart konsumsi token per modalitas, dan DonutChart rasio metode pembayaran dengan custom tooltip Rupiah (`Rp`), toggle legend, dan event klik drill-down ke tabel transaksi terkait.
- [ ] Migrasi database kanonik `0010_settings.sql` diterapkan secara aditif, idempotent, dan file `database/dump/` di-regenerate.
- [ ] Seluruh operasi penyesuaian token menggunakan `idempotency_key` (UUID unik) dan menulis baris append-only baru ke `token_ledger` (K-28).
- [ ] Seluruh tindakan mutasi admin (plan, subscription, payment review, refund, token adjust, AI route/kill-switch, system settings) tercatat otomatis di `audit_logs` (K-29) dengan sanitasi PII UU PDP No. 27/2022 (K-30).
- [ ] Seluruh pengujian otomatis (Go test & unit/E2E test frontend) lulus 100% tanpa regresi.

---

## 2. Urutan Eksekusi (Waves)

| Wave | Task ID | Nama Task | Prasyarat | Fokus Utama |
|---|---|---|---|---|
| **Wave 1** | `TASK-P06-001` | Backend Super Admin Route Group & Middleware Otorisasi | P01, P02 | Security & Routing Core |
| | `TASK-P06-002` | Frontend Super Admin Layout, Auth Guard & Navigasi Sidebar | TASK-P06-001 | Layout & SSR Guard |
| | `TASK-P06-003` | Komponen Reusable `<DataTable>` Berbasis TanStack Table v8 | TASK-P06-002 | UI Core Primitives |
| | `TASK-P06-004` | Sinkronisasi Status `<DataTable>` dengan URL Search Params | TASK-P06-003 | URL State Management |
| **Wave 2** | `TASK-P06-005` | Backend Endpoint Agregasi Metrik & KPI Dashboard | TASK-P06-001 | Aggregation & Redis Cache |
| | `TASK-P06-006` | Antarmuka Dashboard Ringkasan & Kartu KPI | TASK-P06-004, TASK-P06-005 | KPI Cards & Urgent Tasks |
| | `TASK-P06-007` | Visualisasi Analitik Interaktif Berbasis Recharts | TASK-P06-006 | Interactive Charts & Drill-Down |
| **Wave 3** | `TASK-P06-008` | Backend Plan & Entitlement Admin CRUD Handler | TASK-P06-001 | Plan Catalog API & Audit |
| | `TASK-P06-009` | Antarmuka Manajemen Katalog Plan & Form Konfigurasi | TASK-P06-004, TASK-P06-008 | Wizard Form & Plan DataTable |
| | `TASK-P06-010` | Backend Subscription Management & Intervensi Manual | TASK-P06-001, P04 | Subscription Mutations API |
| | `TASK-P06-011` | Antarmuka Monitoring & Intervensi Subscription | TASK-P06-004, TASK-P06-010 | Subscription DataTable & Dialogs |
| **Wave 4** | `TASK-P06-012` | Backend Payment Review, Settlement & Refund Handler | TASK-P06-001, P05 | Concurrency-Safe Payment API |
| | `TASK-P06-013` | Antarmuka Antrean Verifikasi Pembayaran & Dialog Refund | TASK-P06-004, TASK-P06-012 | Proof Viewer & Settlement UI |
| | `TASK-P06-014` | Backend Token Ledger Admin Handler & Idempotent Adjustment | TASK-P06-001, P04 | Idempotent Append-Only Ledger |
| | `TASK-P06-015` | Antarmuka Saldo Token Pengguna & Form Penyesuaian Manual | TASK-P06-004, TASK-P06-014 | Token DataTable & Adjustment Dialog |
| **Wave 5** | `TASK-P06-016` | Backend AI Gateway Operations & Kill Switch Handler | TASK-P06-001, P03 | Circuit Breaker & Pub/Sub Kill Switch |
| | `TASK-P06-017` | Antarmuka Pusat Kendali AI Gateway & Monitoring Realtime | TASK-P06-004, TASK-P06-016 | Live Endpoint Table & Emergency Switch |
| | `TASK-P06-018` | Backend Audit Log Query Handler & Sanitasi Data UU PDP | TASK-P06-001, P02 | PDP-Sanitized Query API |
| | `TASK-P06-019` | Antarmuka Audit Log Viewer dengan Diff JSON Inspeksi | TASK-P06-004, TASK-P06-018 | Audit DataTable & Side-by-Side Diff |
| | `TASK-P06-020` | Antarmuka Pengaturan Sistem Global, Feature Flags & E2E Validation | Wave 1–5 All | Global Settings, Flags & E2E Tests |

---

## 3. Detail Task

### Wave 1: Fondasi Otorisasi, Layout, & Data-Table Engine

#### TASK-P06-001: Backend Super Admin Route Group & Middleware Otorisasi
- **Target File**:
  - `backend/internal/api/routes.go`
  - `backend/internal/middleware/auth.go`
  - `backend/internal/admin/handler.go`
- **Langkah**:
  1. Daftarkan grup rute baru `admin := r.Group("/api/v1/admin")` di `backend/internal/api/routes.go`.
  2. Implementasikan middleware `RequireSuperAdmin(cache *redis.Client, db *sql.DB)` di `backend/internal/middleware/auth.go`.
  3. Periksa klaim token JWT, lalu validasi role langsung terhadap cache Redis berversi (`user:role_version:<user_id>`). Bila versi cache mismatch atau role bukan `super_admin`, tolak dengan status HTTP 403 Forbidden (`FORBIDDEN_SUPER_ADMIN_REQUIRED`).
  4. Siapkan struktur modul `backend/internal/admin/` dan inisialisasi controller dasar.
- **Verifikasi / Bukti**:
  - Unit test `backend/internal/middleware/auth_test.go` memvalidasi skenario role `user` menghasilkan status 403 dan role `super_admin` menghasilkan status 200.
- **Catatan Keamanan**:
  - Menerapkan K-02 dan K-14. Revokasi hak akses instan saat versi role dinaikkan di Redis tanpa menunggu waktu kedaluwarsa JWT.

#### TASK-P06-002: Frontend Super Admin Layout, Auth Guard & Navigasi Sidebar
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/layout.tsx`
  - `frontend/components/admin/sidebar.tsx`
  - `frontend/components/admin/breadcrumbs.tsx`
  - `frontend/components/admin/header.tsx`
- **Langkah**:
  1. Buat layout Server Component di `frontend/app/(dashboard)/super-admin/layout.tsx`. Ekstrak session pengguna di sisi server; jika sesi kosong atau `role !== 'super_admin'`, panggil `redirect('/dashboard')`.
  2. Rancang sidebar admin responsif (`sidebar.tsx`) dengan 8 navigasi utama: Overview Dashboard, Paket Langganan, Pelanggan, Review Pembayaran, Saldo Token, AI Gateway, Jejak Audit, dan Pengaturan Sistem.
  3. Tambahkan badge notifikasi dinamis pada item menu Pembayaran (menampilkan jumlah pending review) dan AI Gateway (menampilkan status circuit breaker).
  4. Buat breadcrumb navigasi otomatis berbasis segment rute Next.js.
- **Verifikasi / Bukti**:
  - Akses URL langsung `/super-admin` menggunakan akun biasa teralihkan ke `/dashboard` secara instan tanpa glitch tampilan.
- **Catatan Zero-Downtime & Keamanan**:
  - Pengamanan berlapis: SSR Guard di layout Next.js mencegah rendering UI, dan backend Gin menolak semua panggilan data.

#### TASK-P06-003: Komponen Reusable `<DataTable>` Berbasis TanStack Table v8
- **Target File**:
  - `frontend/components/data-table/data-table.tsx`
  - `frontend/components/data-table/pagination.tsx`
  - `frontend/components/data-table/toolbar.tsx`
  - `frontend/components/data-table/column-header.tsx`
- **Langkah**:
  1. Kembangkan komponen generic `<DataTable<TData, TValue>>` mengadopsi `@tanstack/react-table` v8.
  2. Implementasikan server-side pagination dengan default 20 baris per halaman dan pilihan ukuran halaman (10, 20, 50, 100).
  3. Pasang nomor baris absolut terhitung: `(page - 1) * per_page + index + 1`.
  4. Tambahkan styling baris interaktif: `cursor-pointer hover:bg-muted/50 transition-colors` pada setiap baris tabel.
  5. Sediakan toolbar terintegrasi: input teks pencarian dengan debounce 300ms, filter facet dropdown (status/kategori), dan dropdown visibilitas kolom.
- **Verifikasi / Bukti**:
  - Unit test komponen me-render tabel dengan mock data; baris memiliki kelas CSS `cursor-pointer`, pagination menampilkan indeks absolut yang tepat.
- **Catatan Desain**:
  - Seluruh 8 tabel administratif pada P06 WAJIB menggunakan komponen tunggal ini tanpa membuat tabel ad-hoc.

#### TASK-P06-004: Sinkronisasi Status `<DataTable>` dengan URL Search Params
- **Target File**:
  - `frontend/components/data-table/use-data-table-state.ts`
  - `frontend/components/data-table/data-table.tsx`
- **Langkah**:
  1. Buat custom hook `useDataTableState()` yang membaca dan menulis parameter URL: `page`, `per_page`, `sort`, `order`, `q`, dan facet filters via `useSearchParams()` dan `useRouter()`.
  2. Terapkan debounced update untuk input pencarian teks agar tidak membebani browser history (`router.replace` dengan opsi `scroll: false`).
  3. Hubungkan hook ke instance TanStack Table pada `data-table.tsx`.
- **Verifikasi / Bukti**:
  - Pengubahan filter atau halaman memperbarui query string di URL browser; melakukan reload halaman mengembalikan posisi tabel dan filter yang sama persis.
- **Catatan UX**:
  - Admin dapat membagikan link URL terfilter secara langsung kepada sesama admin untuk koordinasi investigasi.

---

### Wave 2: Dashboard Eksekutif, Agregasi Metrik & Visualisasi Recharts

#### TASK-P06-005: Backend Endpoint Agregasi Metrik & KPI Dashboard
- **Target File**:
  - `backend/internal/admin/metrics_handler.go`
  - `backend/internal/admin/metrics_service.go`
  - `backend/internal/admin/repository.go`
- **Langkah**:
  1. Buat handler `GET /api/v1/admin/metrics/overview` untuk menghitung: Total Pengguna Aktif (30 Hari), MRR Berjalan (integer Rupiah), Konsumsi Token AI (24 Jam), Rasio Sukses Pembayaran (%), Status Kesehatan AI Endpoint, dan Jumlah Antrean Pending Review Pembayaran.
  2. Buat handler `GET /api/v1/admin/metrics/timeseries` untuk menyajikan tren pendapatan harian dan konsumsi token per modalitas (rentang 7D, 30D, 90D).
  3. Bungkus hasil query dengan caching Redis pada key `admin:metrics:overview` berdurasi TTL 60 detik. Sediakan parameter `?force_refresh=true` untuk bypass cache.
- **Verifikasi / Bukti**:
  - Benchmark endpoint menunjukkan respon < 200ms; key Redis terisi dan ter-invalasi saat parameter bypass dikirimkan.
- **Catatan Performa**:
  - Menggunakan agregasi SQL terindeks untuk mencegah *full table scan* pada tabel ledger dan pembayaran bernilai jutaan baris.

#### TASK-P06-006: Antarmuka Dashboard Ringkasan & Kartu KPI
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/page.tsx`
  - `frontend/components/admin/metric-cards.tsx`
  - `frontend/components/admin/quick-actions.tsx`
  - `frontend/components/admin/urgent-tasks-banner.tsx`
- **Langkah**:
  1. Rancang halaman utama `/super-admin/page.tsx` dengan layout grid modern.
  2. Buat komponen `metric-cards.tsx` yang menampilkan 6 kartu KPI: MRR, Active Users, Token Usage, Success Rate, AI Endpoints, Pending Reviews. Setiap kartu menampilkan persentase delta dibanding bulan lalu dan badge status tren.
  3. Tambahkan banner peringatan tugas darurat (`urgent-tasks-banner.tsx`) bila terdapat pembayaran transfer manual yang pending > 2 jam atau endpoint AI mengalami status degraded/circuit-open.
  4. Sediakan tombol refresh manual dengan indikator animasi putar.
- **Verifikasi / Bukti**:
  - Visualisasi kartu KPI menampilkan skeleton loading saat data dimuat, lalu terisi nilai numerik berformat mata uang Rupiah (`Rp XX.XXX.XXX`).

#### TASK-P06-007: Visualisasi Analitik Interaktif Berbasis Recharts
- **Target File**:
  - `frontend/components/admin/charts/revenue-area-chart.tsx`
  - `frontend/components/admin/charts/token-bar-chart.tsx`
  - `frontend/components/admin/charts/payment-pie-chart.tsx`
- **Langkah**:
  1. Bangun `RevenueAreaChart`: Area chart gradien ganda (Pendapatan Harian vs Garis Tren MRR) warna Emerald (`#10B981`) di dalam `<ResponsiveContainer width="100%" height={320}>`.
  2. Bangun `TokenConsumptionBarChart`: Stacked Bar Chart yang membagi pemakaian token ke dalam 4 modalitas (Chat `#3B82F6`, Vision `#8B5CF6`, Voice `#F59E0B`, Embedding `#10B981`) lengkap dengan selektor rentang waktu (7D/30D/90D).
  3. Bangun `PaymentMethodDonutChart`: Donut chart rasio gateway pembayaran (Midtrans, Xendit, Manual Transfer) dengan teks persentase sukses di tengah lingkaran.
  4. Pasang kustom tooltip terformat mata uang IDR (`Intl.NumberFormat('id-ID')`) pada seluruh grafik.
  5. Implementasikan event klik drill-down interaktif: klik pada titik tanggal grafik mengarahkan admin ke `/super-admin/payments?date=YYYY-MM-DD` atau `/super-admin/tokens?date=YYYY-MM-DD`.
- **Verifikasi / Bukti**:
  - Tooltip menampilkan format Rupiah tepat; klik batang grafik menavigasikan browser ke URL tujuan dengan query params tanggal yang sesuai.

---

### Wave 3: Tata Kelola Paket Langganan & Siklus Pengguna

#### TASK-P06-008: Backend Plan & Entitlement Admin CRUD Handler
- **Target File**:
  - `backend/internal/admin/plan_handler.go`
  - `backend/internal/admin/plan_service.go`
  - `backend/internal/admin/repository.go`
- **Langkah**:
  1. Implementasikan endpoint CRUD katalog paket:
     - `GET /api/v1/admin/plans`: daftar paket beserta rincian entitlement dan batas kuota.
     - `POST /api/v1/admin/plans`: pembuatan paket baru dengan validasi harga integer non-negatif.
     - `PUT /api/v1/admin/plans/:id`: pembaruan metadata, entitlement matriks, kuota token, dan gateway override.
     - `DELETE /api/v1/admin/plans/:id`: soft-archive (`is_active = false`). Tolak jika masih ada pengguna aktif terikat.
  2. Terapkan transaksi atomik DB saat menyimpan konfigurasi entitlement boolean dan limit numerik.
  3. Rekam setiap aksi mutasi ke tabel `audit_logs` lengkap dengan snapshot `before_state` dan `after_state`.
- **Verifikasi / Bukti**:
  - Unit test memvalidasi penolakan harga negatif; audit log terbuat setiap kali paket diubah; percobaan menghapus paket berpelanggan aktif menghasilkan pesan galat terproteksi.
- **Catatan Keuangan**:
  - Seluruh harga wajib integer rupiah tanpa desimal (K-27). Durasi siklus disimpan dalam hitungan hari kalender (K-05).

#### TASK-P06-009: Antarmuka Manajemen Katalog Plan & Form Konfigurasi
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/plans/page.tsx`
  - `frontend/components/admin/plans/plan-form-dialog.tsx`
  - `frontend/components/admin/plans/columns.tsx`
- **Langkah**:
  1. Bangun halaman `/super-admin/plans` menggunakan komponen standar `<DataTable>` dengan baris `cursor-pointer` untuk membuka detail paket.
  2. Rancang modal formulir wizard bertab (`PlanFormDialog`):
     - Tab 1: Informasi Dasar & Harga (Nama, slug, deskripsi, harga IDR, durasi siklus hari).
     - Tab 2: Fitur Entitlement Boolean (Checkbox: `can_use_budgeting`, `can_export_report`, `can_use_voice`, `can_use_vision`, `priority_ai_routing`).
     - Tab 3: Batasan Kuota Numerik (Max wallets, max budgets, kuota token per siklus; nilai `-1` untuk unlimited).
     - Tab 4: Dynamic Pricing Rules AI & Token Multiplier.
     - Tab 5: Gateway Override & Status Publikasi.
  3. Sediakan switch toggle aktif/nonaktif langsung pada kolom tabel dengan konfirmasi optimis.
- **Verifikasi / Bukti**:
  - Form memvalidasi input; paket baru berhasil ditambahkan dan langsung muncul di `<DataTable>` tanpa reload halaman.

#### TASK-P06-010: Backend Subscription Management & Intervensi Manual
- **Target File**:
  - `backend/internal/admin/subscription_handler.go`
  - `backend/internal/admin/subscription_service.go`
- **Langkah**:
  1. Implementasikan endpoint query dan intervensi:
     - `GET /api/v1/admin/subscriptions`: daftar langganan dengan pagination, filter status (`active`, `past_due`, `canceled`), dan pencarian user.
     - `POST /api/v1/admin/subscriptions/:id/extend`: menambah $N$ hari pada `current_period_end`.
     - `POST /api/v1/admin/subscriptions/:id/force-downgrade`: transisi instan ke paket Free dengan menerapkan aturan soft-limit (K-09).
     - `POST /api/v1/admin/subscriptions/:id/cancel`: membatalkan pembaruan otomatis pada akhir siklus.
  2. Pastikan `force-downgrade` TIDAK PERNAH menghapus data transaksi/anggaran milik pengguna di database, melainkan mengunci pembuatan data baru yang melampaui limit paket Free.
  3. Catat seluruh kejadian intervensi ke `subscription_events` dan `audit_logs` disertai `admin_reason`.
  4. Invalasi cache hak akses pengguna di Redis agar perubahan langsung berlaku di sesi klien.
- **Verifikasi / Bukti**:
  - Integrasi test memverifikasi penambahan hari siklus dan penegakan soft-limit tanpa kehilangan baris data user.

#### TASK-P06-011: Antarmuka Monitoring & Intervensi Subscription
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/subscriptions/page.tsx`
  - `frontend/components/admin/subscriptions/action-dialogs.tsx`
  - `frontend/components/admin/subscriptions/columns.tsx`
- **Langkah**:
  1. Bangun halaman `/super-admin/subscriptions` dengan `<DataTable>` (kolom: Pengguna, Paket, Status Badge, Awal Siklus, Akhir Siklus, Auto-Renew, Aksi; baris `cursor-pointer`).
  2. Buat komponen dropdown menu aksi pada setiap baris:
     - Opsi `Perpanjang Periode`: Dialog input jumlah hari tambahan dan teks alasan admin.
     - Opsi `Turunkan Paket (Force Downgrade)`: Dialog konfirmasi bahaya berkode warna merah yang menerangkan efek soft-limit pada akun user.
     - Opsi `Batalkan Langganan`: Dialog konfirmasi penghentian perpanjangan otomatis.
  3. Hubungkan mutasi dengan TanStack Query untuk pembaruan data real-time di antarmuka.
- **Verifikasi / Bukti**:
  - Intervensi perpanjangan siklus memperbarui tanggal akhir pada tabel secara instan dan memicu notifikasi toast sukses.

---

### Wave 4: Verifikasi Pembayaran & Rekonsiliasi Token Ledger

#### TASK-P06-012: Backend Payment Review, Settlement & Refund Handler
- **Target File**:
  - `backend/internal/admin/payment_handler.go`
  - `backend/internal/admin/payment_service.go`
  - `backend/internal/admin/repository.go`
- **Langkah**:
  1. Implementasikan endpoint peninjauan transaksi:
     - `GET /api/v1/admin/payments/review`: antrean transaksi transfer manual berstatus `pending_review`.
     - `POST /api/v1/admin/payments/:id/approve`: menyetujui bukti bayar, mengubah status ke `settled`, mengaktifkan hak langganan user, dan menerbitkan invoice resmi.
     - `POST /api/v1/admin/payments/:id/reject`: menolak transaksi dengan mencantumkan alasan penolakan.
     - `POST /api/v1/admin/payments/:id/refund`: mencatat pengembalian dana administratif (penuh atau sebagian dalam integer Rupiah).
  2. Cegah race condition antar-admin (double review) menggunakan optimistic locking:
     `UPDATE payments SET status = 'settled', reviewed_by = $1, version = version + 1 WHERE id = $2 AND status = 'pending_review' AND version = $3`.
     Bila `RowsAffected == 0`, kembalikan HTTP 409 Conflict.
  3. Catat riwayat aksi ke `payment_events` dan `audit_logs`.
- **Verifikasi / Bukti**:
  - Uji konkurensi (simultaneous requests) membuktikan hanya 1 request yang berhasil mengeksekusi approve/reject; request kedua menerima HTTP 409.
- **Catatan Keuangan**:
  - Nilai refund dicatat secara append-only tanpa menghapus record pembayaran asli (K-28).

#### TASK-P06-013: Antarmuka Antrean Verifikasi Pembayaran & Dialog Refund
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/payments/page.tsx`
  - `frontend/app/(dashboard)/super-admin/payments/review/page.tsx`
  - `frontend/components/admin/payments/proof-preview-modal.tsx`
  - `frontend/components/admin/payments/refund-dialog.tsx`
  - `frontend/components/admin/payments/columns.tsx`
- **Langkah**:
  1. Bangun halaman antrean verifikasi di `/super-admin/payments/review` dan log riwayat transaksi di `/super-admin/payments` menggunakan `<DataTable>`. Seluruh baris diberi styling `cursor-pointer`.
  2. Implementasikan `ProofPreviewModal`: Komponen pratinjau bukti transfer yang mendukung pembesaran (zoom in/out), rotasi gambar 90 derajat, dan inspeksi dokumen PDF berdampingan dengan rincian nominal tagihan.
  3. Buat tombol tindakan `Approve` dan `Reject` yang terlindungi dari double-click dengan disable state otomatis saat mutasi berjalan.
  4. Buat dialog modal `RefundDialog` dengan input nominal Rupiah yang dapat dikembalikan, alasan pengembalian dana, dan nomor referensi transfer bank pengembalian.
- **Verifikasi / Bukti**:
  - Gambar bukti bayar dapat dirotasi dan di-zoom; konfirmasi approve berhasil mengubah status baris di tabel seketika.

#### TASK-P06-014: Backend Token Ledger Admin Handler & Idempotent Adjustment
- **Target File**:
  - `backend/internal/admin/token_handler.go`
  - `backend/internal/admin/token_service.go`
  - `backend/internal/admin/repository.go`
- **Langkah**:
  1. Implementasikan endpoint:
     - `GET /api/v1/admin/tokens/users`: ringkasan saldo token aktif per pengguna.
     - `GET /api/v1/admin/tokens/ledger`: log riwayat transaksi token append-only dengan filter user, waktu, dan modalitas.
     - `POST /api/v1/admin/tokens/adjust`: menambah (credit) atau memotong (debit) kuota token pengguna secara manual.
  2. Wajibkan penerimaan header/payload `idempotency_key` (UUID v4). Simpan key di PostgreSQL/Redis untuk mencegah eksekusi ganda penyesuaian saldo token.
  3. Catat penyesuaian sebagai baris baru di `token_ledger` (append-only, tidak ada mutasi baris masa lalu) dengan tipe transaksi `ADMIN_ADJUSTMENT`.
- **Verifikasi / Bukti**:
  - Pemanggilan ganda endpoint adjust dengan `idempotency_key` yang sama hanya menambahkan 1 baris di tabel `token_ledger` dan mengembalikan payload respon yang identik.

#### TASK-P06-015: Antarmuka Saldo Token Pengguna & Form Penyesuaian Manual
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/tokens/page.tsx`
  - `frontend/components/admin/tokens/adjust-token-modal.tsx`
  - `frontend/components/admin/tokens/token-columns.tsx`
  - `frontend/components/admin/tokens/ledger-columns.tsx`
- **Langkah**:
  1. Bangun antarmuka `/super-admin/tokens` dengan dua tab utama:
     - Tab 1: Ringkasan Saldo Pengguna (Tabel `<DataTable>` dengan kolom Nama, Email, Saldo Kuota Siklus, Saldo Tambahan, Total Token, dan Tombol Penyesuaian). Baris berkursur pointer.
     - Tab 2: Riwayat Ledger Global (Tabel `<DataTable>` log audit append-only pemakaian & penambahan token).
  2. Bangun formulir dialog modal `AdjustTokenModal`:
     - Pemilihan Pengguna (Combobox / User Picker).
     - Pilihan Tipe Penyesuaian: Tambah Kuota (Kredit) atau Kurangi Kuota (Debit).
     - Input Jumlah Token: Angka integer positif.
     - Teks Alasan Administratif (Wajib diisi).
     - Otomatis menghasilkan `idempotency_key` (UUID v4) saat modal dibuka di browser.
- **Verifikasi / Bukti**:
  - Kuota token pengguna bertambah atau berkurang sesuai input form; baris baru muncul seketika pada tab Riwayat Ledger.

---

### Wave 5: Operasi AI Gateway, Audit PDP, Pengaturan Sistem & E2E

#### TASK-P06-016: Backend AI Gateway Operations & Kill Switch Handler
- **Target File**:
  - `backend/internal/admin/aigateway_handler.go`
  - `backend/internal/admin/aigateway_service.go`
  - `backend/internal/aigateway/circuit_breaker.go`
  - `backend/internal/aigateway/kill_switch.go`
- **Langkah**:
  1. Implementasikan endpoint operasional AI:
     - `GET /api/v1/admin/ai-gateway/endpoints`: daftar status endpoint 9router upstream (latensi p95, error rate, status circuit breaker: Closed, Half-Open, Open).
     - `POST /api/v1/admin/ai-gateway/endpoints`: registrasi dan konfigurasi pool endpoint baru.
     - `PUT /api/v1/admin/ai-gateway/routes`: pemetaan rute dinamis per-use-case (chat, vision, voice, embedding).
     - `POST /api/v1/admin/ai-gateway/circuit-breaker/reset`: me-reset circuit breaker endpoint yang macet.
     - `POST /api/v1/admin/ai-gateway/kill-switch`: mengaktifkan saklar darurat global atau per modalitas.
  2. Siarkan perubahan kill switch dan rute via Redis Pub/Sub (`channel:ai_gateway_events`) agar diserap instan (< 100ms) oleh seluruh node worker Go.
  3. Kunci dimensi embedding pada 768 dimensi; tolak konfigurasi embedding model non-768d.
- **Verifikasi / Bukti**:
  - Publikasi kill-switch mematikan rute AI pada seluruh worker backend dalam hitungan milidetik; permintaan AI yang masuk saat saklar aktif segera ditolak dengan pesan maintenance.

#### TASK-P06-017: Antarmuka Pusat Kendali AI Gateway & Monitoring Realtime
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/ai-gateway/page.tsx`
  - `frontend/components/admin/ai-gateway/kill-switch-panel.tsx`
  - `frontend/components/admin/ai-gateway/endpoint-table.tsx`
  - `frontend/components/admin/ai-gateway/route-matrix.tsx`
- **Langkah**:
  1. Bangun halaman kendali `/super-admin/ai-gateway`.
  2. Buat panel darurat `KillSwitchPanel`: Saklar sakral berkode warna merah menyala untuk pemadaman Global, Chat, Vision, Voice, dan Embedding. Wajibkan pengetikan teks konfirmasi manual (misal ketik kata `"MATIKAN"`) sebelum dialog eksekusi dapat disetujui.
  3. Tampilkan tabel pool endpoint menggunakan `<DataTable>` dengan baris `cursor-pointer`, badge lampu lalu lintas (Hijau/Kuning/Merah), latensi waktu nyata, dan tombol aksi `Reset Circuit Breaker`.
  4. Buat antarmuka `RouteMatrix`: Tabel pemetaan model dan target endpoint untuk tiap modalitas yang dapat disunting langsung di browser.
- **Verifikasi / Bukti**:
  - Saklar darurat memerlukan konfirmasi teks; tombol reset circuit breaker berhasil memulihkan endpoint yang sedang berstatus open ke half-open/closed.

#### TASK-P06-018: Backend Audit Log Query Handler & Sanitasi Data UU PDP
- **Target File**:
  - `backend/internal/admin/audit_handler.go`
  - `backend/internal/admin/audit_sanitizer.go`
  - `backend/internal/audit/repository.go`
- **Langkah**:
  1. Implementasikan endpoint `GET /api/v1/admin/audit-logs` dengan filter: rentang waktu (`start_date`, `end_date`), aktor (`user_id`, `actor_email`), jenis entitas (`plan`, `subscription`, `payment`, `token`, `ai_route`, `settings`), dan aksi (`CREATE`, `UPDATE`, `DELETE`, `EXECUTE`).
  2. Bangun layer sanitasi data `audit_sanitizer.go` sesuai UU PDP No. 27/2022 (K-30):
     - Sensor informasi kredensial (`api_key`, `token`, `secret`, `password`) menjadi string `[REDACTED]`.
     - Masking data kartu dan rekening (`4111-XXXX-XXXX-1234`).
     - Masking nomor identitas dan nomor telepon pribadi.
  3. Tegaskan bahwa tabel audit log bersifat *append-only* murni (tidak ada handler untuk delete atau update).
- **Verifikasi / Bukti**:
  - Respons JSON audit logs tidak mengandung kredensial mentah atau data pribadi terbuka; query filter berjalan cepat melalui indeks database.

#### TASK-P06-019: Antarmuka Audit Log Viewer dengan Diff JSON Inspeksi
- **Target File**:
  - `frontend/app/(dashboard)/super-admin/audit/page.tsx`
  - `frontend/components/admin/audit/audit-diff-dialog.tsx`
  - `frontend/components/admin/audit/columns.tsx`
- **Langkah**:
  1. Bangun halaman `/super-admin/audit` menggunakan `<DataTable>` (kolom: Waktu WIB, Aktor Email, Aksi Badge, Entitas, IP Address, dan Tombol Inspeksi; baris diberi `cursor-pointer` yang membuka dialog inspeksi).
  2. Pasang facet filter: DateRangePicker, dropdown tipe entitas, dan multi-select aksi.
  3. Kembangkan dialog modal inspeksi diff (`AuditDiffDialog`): Tampilan *side-by-side* yang membandingkan objek `before_state` dan `after_state`. Baris properti baru diberi latar hijau (`bg-emerald-500/10`), properti terhapus diberi latar merah (`bg-rose-500/10`), dan properti termodifikasi diberi latar kuning.
  4. Tambahkan tombol salin format JSON mentah ke clipboard.
- **Verifikasi / Bukti**:
  - Mengklik baris tabel membuka dialog diff; visualisasi warna hijau/merah membedakan perubahan konfigurasi dengan akurat.

#### TASK-P06-020: Antarmuka Pengaturan Sistem Global, Feature Flags & E2E Validation
- **Target File**:
  - `database/migrations/0010_settings.sql`
  - `backend/internal/admin/settings_handler.go`
  - `frontend/app/(dashboard)/super-admin/settings/page.tsx`
  - `frontend/components/admin/settings/feature-flags-card.tsx`
  - `frontend/components/admin/settings/gateway-config-card.tsx`
  - `backend/tests/admin_e2e_test.go`
- **Langkah**:
  1. Buat file migrasi kanonik `database/migrations/0010_settings.sql` untuk membuat tabel `system_settings` (key-value JSONB, deskripsi, updated_by, updated_at). Terapkan secara aditif dan perbarui `database/dump/`.
  2. Implementasikan handler backend `GET /api/v1/admin/settings` dan `PUT /api/v1/admin/settings`.
  3. Bangun halaman `/super-admin/settings` di Next.js:
     - Kartu Saklar Payment Gateway Global (Midtrans Snap vs Manual Transfer switch, K-10).
     - Kartu Konfigurasi Grace Period (Input angka hari toleransi sebelum freeze fitur).
     - Kartu Matriks Feature Flags (Toggle switch aktivasi fitur eksperimental).
  4. Tulis suite pengujian integrasi E2E komprehensif di `backend/tests/admin_e2e_test.go`:
     - Skenario otorisasi super admin vs user biasa.
     - Skenario pembuatan paket dan inspeksi jejak audit.
     - Skenario review pembayaran dan mitigasi race condition.
     - Skenario penyesuaian saldo token idempoten.
     - Skenario aktivasi kill switch AI Gateway.
- **Verifikasi / Bukti**:
  - Skrip migrasi tereksekusi tanpa error; perubahan switch tersimpan di database dan cache Redis; seluruh tes E2E di `admin_e2e_test.go` lulus 100%.

---

## 4. Matriks Bukti & Artefak Task

| Task ID | Artefak Kode Utama | Artefak Bukti / Pengujian |
|---|---|---|
| `TASK-P06-001` | `backend/internal/middleware/auth.go`<br>`backend/internal/api/routes.go` | `backend/internal/middleware/auth_test.go` (Status 401/403/200 test) |
| `TASK-P06-002` | `frontend/app/(dashboard)/super-admin/layout.tsx`<br>`frontend/components/admin/sidebar.tsx` | SSR redirect test (Playwright/Jest mock session role non-admin) |
| `TASK-P06-003` | `frontend/components/data-table/data-table.tsx`<br>`frontend/components/data-table/pagination.tsx` | Unit test render data-table & pemeriksaan index penomoran absolut |
| `TASK-P06-004` | `frontend/components/data-table/use-data-table-state.ts` | Cypress/Playwright URL param bookmarking & refresh test |
| `TASK-P06-005` | `backend/internal/admin/metrics_handler.go`<br>`backend/internal/admin/metrics_service.go` | Unit test handler metrik & Redis benchmark respon < 200ms |
| `TASK-P06-006` | `frontend/app/(dashboard)/super-admin/page.tsx`<br>`frontend/components/admin/metric-cards.tsx` | Pratinjau visual kartu KPI & skeleton loading state |
| `TASK-P06-007` | `frontend/components/admin/charts/revenue-area-chart.tsx`<br>`frontend/components/admin/charts/token-bar-chart.tsx` | Test drill-down event klik chart yang mengubah URL search params |
| `TASK-P06-008` | `backend/internal/admin/plan_handler.go`<br>`backend/internal/admin/plan_service.go` | Unit test transaksi atomik CRUD plan & validasi integer Rupiah |
| `TASK-P06-009` | `frontend/app/(dashboard)/super-admin/plans/page.tsx`<br>`frontend/components/admin/plans/plan-form-dialog.tsx` | Snapshot wizard dialog 5 tab & render tabel plan berbasis `<DataTable>` |
| `TASK-P06-010` | `backend/internal/admin/subscription_handler.go`<br>`backend/internal/admin/subscription_service.go` | Test intervensi extend siklus & soft-limit downgrade K-09 |
| `TASK-P06-011` | `frontend/app/(dashboard)/super-admin/subscriptions/page.tsx`<br>`frontend/components/admin/subscriptions/action-dialogs.tsx` | Render tabel subscription & modal dialog konfirmasi perpanjangan |
| `TASK-P06-012` | `backend/internal/admin/payment_handler.go`<br>`backend/internal/admin/payment_service.go` | Test optimistic concurrency locking (HTTP 409 Conflict check) |
| `TASK-P06-013` | `frontend/app/(dashboard)/super-admin/payments/review/page.tsx`<br>`frontend/components/admin/payments/proof-preview-modal.tsx` | Verifikasi fungsional zoom/rotate bukti transfer & modal refund |
| `TASK-P06-014` | `backend/internal/admin/token_handler.go`<br>`backend/internal/admin/token_service.go` | Idempotency test: duplicate UUID menghasilkan single append ledger |
| `TASK-P06-015` | `frontend/app/(dashboard)/super-admin/tokens/page.tsx`<br>`frontend/components/admin/tokens/adjust-token-modal.tsx` | Uji penyesuaian kuota token credit/debit & refresh tab ledger |
| `TASK-P06-016` | `backend/internal/admin/aigateway_handler.go`<br>`backend/internal/aigateway/kill_switch.go` | Redis Pub/Sub broadcast test latensi < 100ms & penolakan embedding non-768d |
| `TASK-P06-017` | `frontend/app/(dashboard)/super-admin/ai-gateway/page.tsx`<br>`frontend/components/admin/ai-gateway/kill-switch-panel.tsx` | Validasi safety text confirmation ("MATIKAN") pada saklar darurat |
| `TASK-P06-018` | `backend/internal/admin/audit_handler.go`<br>`backend/internal/admin/audit_sanitizer.go` | UU PDP test sanitasi data PII (verifikasi tidak ada password/token mentah) |
| `TASK-P06-019` | `frontend/app/(dashboard)/super-admin/audit/page.tsx`<br>`frontend/components/admin/audit/audit-diff-dialog.tsx` | Render side-by-side JSON diff dialog dengan highlight hijau/merah |
| `TASK-P06-020` | `database/migrations/0010_settings.sql`<br>`frontend/app/(dashboard)/super-admin/settings/page.tsx`<br>`backend/tests/admin_e2e_test.go` | Eksekusi migrasi 0010 berhasil + Laporan kelulusan 100% `admin_e2e_test.go` |

---

## 5. Perintah Verifikasi Operasional

Saat fase implementasi dieksekusi, gunakan perintah standar berikut untuk memverifikasi kebersihan kode, kebenaran kompilasi, dan kelulusan pengujian:

### 5.1 Verifikasi Backend (Go)
```bash
# Pindah ke direktori backend
cd D:/project/finance/fintrack_v2/backend

# Pemeriksaan sintaksis dan statis
go vet ./...

# Menjalankan unit tests seluruh paket administratif
go test -v -race ./internal/admin/... ./internal/middleware/...

# Menjalankan pengujian E2E integrasi admin
go test -v -run TestAdminE2E ./tests/...

# Memastikan biner backend dapat dikompilasi tanpa error
go build -o /dev/null ./cmd/server
```

### 5.2 Verifikasi Frontend (Next.js 16)
```bash
# Pindah ke direktori frontend
cd D:/project/finance/fintrack_v2/frontend

# Pemeriksaan tipe TypeScript statis
pnpm typecheck # atau pnpm tsc --noEmit

# Pemeriksaan linting ESLint
pnpm lint

# Menjalankan pengujian unit komponen antarmuka
pnpm test

# Menjalankan build produksi Next.js untuk memastikan tidak ada kesalahan SSR
pnpm build
```

### 5.3 Verifikasi Database & Migrasi
```bash
# Menerapkan migrasi 0010 secara aditif
docker compose exec backend /app/migrate up

# Regenerasi schema dump PostgreSQL setelah migrasi
docker compose exec db pg_dump -U postgres -d fintrack_v2 --schema-only > database/dump/schema.sql
```

---

## 6. Prosedur Rollback & Kontinjensi

Bila terjadi anomali kritis saat tahap penggelaran atau eksekusi fase P06:

### 6.1 Rollback Kode Frontend
- Rute super admin di Next.js terisolasi di dalam route group `app/(dashboard)/super-admin/*`.
- Bila ditemukan regresi antarmuka fatal, super admin route group dapat dinonaktifkan sementara melalui feature flag global `ENABLE_SUPER_ADMIN_PORTAL=false` di environment atau file konfigurasi tanpa perlu melakukan rebuild/redeploy menyeluruh.

### 6.2 Rollback Endpoint Backend
- Endpoint grup admin berada di prefix terpisah `/api/v1/admin/*`.
- Jika ditemukan kelemahan logika pada salah satu handler admin, middleware `RequireSuperAdmin()` dapat disetel untuk mengembalikan status HTTP 503 Maintenance Mode khusus untuk rute admin, sementara endpoint pengguna umum (`/api/v1/*`) tetap berjalan 100% normal tanpa downtime.

### 6.3 Rollback Database (Migrasi 0010)
- Migrasi kanonik `0010_settings.sql` bersifat aditif (membuat tabel baru `system_settings` dan nilai default tanpa mengubah struktur tabel bawaan).
- Skrip pembatalan aditif:
  ```sql
  -- Rollback migrasi 0010
  DROP TABLE IF EXISTS system_settings;
  DELETE FROM schema_migrations WHERE version = 10;
  ```
- Tidak ada data tabel operasional utama (users, transactions, budgets, wallets) yang terpengaruh selama rollback migrasi ini.

---

## 7. Handoff ke Fase Berikutnya

Setelah seluruh kriteria Definition of Done (DoD) pada Fase P06 terpenuhi:
1. **Fase P07 — Advanced Reporting & Export**:
   - Memanfaatkan endpoint metrik MRR dan data agregasi yang sudah distandardisasi pada P06 untuk menghasilkan laporan analitik tingkat lanjut dan export CSV/PDF otomatis.
2. **Fase P08 — Hardening, Observability & Production Readiness**:
   - Menghubungkan metrik kesehatan AI Gateway dan log audit P06 ke dashboard Prometheus/Grafana dan OpenTelemetry tracing global.
3. **Dokumentasi Operasional**:
   - Siapkan SOP manual bagi Super Admin mengenai penanganan sengketa pembayaran transfer manual dan panduan pengaktifan Emergency Kill Switch saat terjadi lonjakan anomali kuota upstream AI.

---

## 8. Dokumen Referensi

1. `second_brain/2_Projects/P06_Super-Admin-Dashboard/README.md` — Ikhtisar modul dan tujuan bisnis.
2. `second_brain/2_Projects/P06_Super-Admin-Dashboard/planning/tasks.md` — Daftar rincian 20 tugas administratif (`TASK-P06-001` hingga `TASK-P06-020`).
3. `second_brain/2_Projects/P06_Super-Admin-Dashboard/planning/timeline.md` — Estimasi jadwal dan dependensi antar wave kerja.
4. `second_brain/2_Projects/P06_Super-Admin-Dashboard/teknis/superadmin-architecture.md` — Arsitektur keamanan, RBAC, dan integrasi backend Go/Gin.
5. `second_brain/2_Projects/P06_Super-Admin-Dashboard/teknis/charts-interactive.md` — Desain teknis Recharts interaktif dan drill-down.
6. `second_brain/2_Projects/P06_Super-Admin-Dashboard/teknis/plan-management-ui.md` — Spesifikasi UI katalog plan dan form wizard entitlement.
7. `second_brain/2_Projects/P06_Super-Admin-Dashboard/teknis/payment-review-ui.md` — Alur verifikasi pembayaran manual dan mitigasi race condition.
8. `second_brain/2_Projects/P06_Super-Admin-Dashboard/teknis/audit-viewer.md` — Spesifikasi penampil audit trail dan kepatuhan UU PDP No. 27/2022.
9. `second_brain/2_Projects/P06_Super-Admin-Dashboard/teknis/metrics-dashboard.md` — Formula agregasi metrik finansial dan KPI sistem.
10. `second_brain/2_Projects/P06_Super-Admin-Dashboard/teknis/admin-menu.md` — Matriks navigasi sidebar dan struktur breadcrumbs.
11. `second_brain/IMPLEMENTATION-PLAN.md` — Master implementation plan FinTrack v2.
