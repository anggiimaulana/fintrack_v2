# P06 - Super Admin Dashboard

## Konteks & Deskripsi
Fase P06 menghadirkan antarmuka web terpusat dan backend API khusus untuk operasional administrator tingkat tinggi (`super_admin`). Modul ini mengendalikan seluruh siklus hidup produk FinTrack v2: manajemen katalog paket (Plan & Entitlement), intervensi langganan (Subscription Lifecycle), tinjauan dan rekonsiliasi transaksi pembayaran (Payment Review), kuota & ledger token AI, manajemen operasional AI Gateway (endpoint, routing, circuit breaker, kill switch), penampil jejak audit (Audit Log Viewer), serta konfigurasi platform global (Payment Gateway switch, grace period, default pricing, feature flags).

Sesuai arsitektur sistem, backend Go/Gin bertindak sebagai otoritas tunggal penegakan otorisasi (`super_admin` role gate), sedangkan frontend Next.js 16 (`app/(dashboard)/super-admin/*`) mengonsumsi data via REST API dan TanStack Query v5, menampilkan data tabular seragam menggunakan `<DataTable>` berbasis TanStack Table v8 dengan sinkronisasi status ke URL search params, serta visualisasi analitik interaktif berbasis Recharts.

## Ruang Lingkup (Scope)
1. **Frontend Super Admin Portal (`app/(dashboard)/super-admin/*`)**:
   - Layout khusus dengan navigation sidebar terproteksi role `super_admin`.
   - Ringkasan Metrik & KPI Dashboard (User aktif, MRR, token usage, payment success rate, AI status, antrean review).
   - Katalog & Manajemen Plan: CRUD plan, entitlement flags, limit numerik, pricing rule, AI markup, gateway override.
   - Manajemen Subscription: Monitoring status, perpanjangan manual (extend), force downgrade (soft-limit), terminasi/cancel.
   - Manajemen Pembayaran & Review Manual: Antrean verifikasi bukti transfer/manual, pencatatan refund, histori pembayaran.
   - Manajemen Token & Kuota: Saldo/pemakaian per user, penyesuaian kuota manual (adjust balance), log ledger token.
   - Operasional AI Gateway: Manajemen pool endpoint (health/latency/cost), routing use-case, status circuit breaker, kill switch darurat.
   - Penampil Jejak Audit (Audit Log Viewer): Tampilan terstruktur diff json `before` -> `after`, filter aktor/entitas/waktu, kepatuhan UU PDP No. 27/2022.
   - Pengaturan Sistem Global: Switch default gateway, konfigurasi grace period renewal, penetapan harga dasar token, toggle feature flags.
2. **Backend Admin Handlers & Endpoints (`internal/admin`, `internal/audit`, `internal/billing`, `internal/aigateway`)**:
   - Middleware otorisasi ketat `RequireRole("super_admin")`.
   - Admin Analytics Handler (agregasi KPI, metrik performa, grafik time-series).
   - Plan & Entitlement Admin Handler (CRUD terisolasi dengan audit trail).
   - Subscription Admin Handler (intervensi manual dengan idempotency).
   - Payment Admin Handler (review manual, approval/rejection, recording refund).
   - Token Admin Handler (manual ledger credit/debit dengan idempotency key).
   - AI Gateway Admin Handler (endpoint CRUD, route update, kill switch toggle).
   - Audit Log Query Handler (paged, filtered, sanitasi data sensitif PII).
   - System Settings & Feature Flag Admin Handler.
3. **Komponen Reusable UI**:
   - `<DataTable>` terstandarisasi: Pagination 20 item, server-side sorting, global/column search, facet filters, penomoran berurutan, sinkronisasi URL state (`?page=&sort=&order=&q=`).
   - Chart Komponen: Time-series revenue (MRR), token consumption breakdown, payment success funnel, AI latency distribution dengan tooltip, legend, dan drill-down.

## Batasan (Out of Scope)
- Fitur checkout atau self-service portal untuk user biasa (ditangani di P04/P05).
- Akses eksternal publik tanpa autentikasi session admin.
- Modifikasi langsung ke database bypass audit log (semua mutasi wajib lewat handler admin ber-audit).
- Multi-tenancy atau delegasi peran admin granular (sistem mengunci peran hanya `super_admin` dan `user` sesuai K-02).

## Deliverables
> 📦 **Arsip Detail P06:** Seluruh dokumen analisis keputusan, planning, rules, teknis, dan tracking tersimpan lengkap di [`../../5_Archives/phases/P06_Super-Admin-Dashboard/`](../../5_Archives/phases/P06_Super-Admin-Dashboard/).

1. **Dokumentasi Analisis, Keputusan & Perencanaan (Diarsipkan)**:
   - `analisis/hasil-analisis-teknis.md` (Keputusan K-P06-01 s/d K-P06-12).
   - `planning/sprint-plan.md` (Rencana sprint 2 minggu 4 milestone).
   - `planning/tasks.md` (Daftar rincian TASK-P06-001 s/d TASK-P06-020).
   - `planning/timeline.md` (Timeline sekuensial, critical path, matriks dependensi).
   - `rules/implementation-rules.md` (Aturan ketat arsitektur, UI/UX, audit, dan zero downtime).
   - `tracking/tracking.md` (Matriks status kesiapan task).
2. **Dokumentasi Teknis Arsitektur & Antarmuka (Diarsipkan)**:
   - `teknis/superadmin-architecture.md` (Arsitektur modul, flow otorisasi, keamanan session & CSRF).
   - `teknis/admin-menu.md` (Struktur navigasi, breadcrumb, routing hierarchy Next.js App Router).
   - `teknis/metrics-dashboard.md` (Spesifikasi query agregasi KPI, formula MRR, performa query & caching Redis).
   - `teknis/charts-interactive.md` (Desain komponen chart Recharts, responsivitas, drill-down interaction).
   - `teknis/audit-viewer.md` (Spesifikasi UI diff inspection, sanitasi PII UU PDP, audit search).
   - `teknis/payment-review-ui.md` (Alur verifikasi pembayaran manual, mitigasi race condition, dialog refund).
   - `teknis/plan-management-ui.md` (Form wizard/dialog katalog plan, konfigurasi entitlement checkbox, dynamic pricing rules).

## Kriteria Sukses (Success Criteria)
1. **Otorisasi Terisolasi**: Pengguna tanpa role `super_admin` menerima HTTP 403 Forbidden pada seluruh endpoint `/api/v1/admin/*` dan ter-redirect otomatis di frontend.
2. **Kesesuaian `<DataTable>`**: 100% tabel tabular di menu Super Admin mengimplementasikan TanStack Table v8 dengan sinkronisasi URL search params, navigasi bookmarkable, server-side pagination (default 20), dan penomoran absolut akurat.
3. **Integritas Audit Trail**: 100% aksi mutasi (Create, Update, Delete, Toggle, Adjust) di dashboard Super Admin menghasilkan baris baru pada tabel `audit_logs` secara append-only dengan metadata aktor, IP, user-agent, dan diff data `before`/`after`.
4. **Respon Cepat (< 250ms)**: Agregasi metrik dashboard utama memanfaatkan agregasi terindeks dan caching Redis ber-TTL pendek (30-60 detik) sehingga load time < 250ms.
5. **Zero Downtime & Idempoten**: Seluruh aksi manual (adjust kuota token, approve pembayaran, extend subscription) wajib menyertakan `idempotency_key` untuk mencegah eksekusi ganda saat network retries.
