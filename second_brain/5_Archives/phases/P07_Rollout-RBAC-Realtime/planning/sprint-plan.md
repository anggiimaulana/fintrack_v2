# Sprint Plan: Fase P07 — Rollout RBAC, Realtime & Standar

## Durasi & Kapasitas Sprint
- **Durasi**: 2 Minggu (10 Hari Kerja)
- **Fokus Utama**: Rollout menyeluruh RBAC & Entitlement, implementasi Hub SSE terdistribusi via Redis, standardisasi DataTable & UI UX, serta orkestrasi Zero-Downtime Deployment.
- **Kapasitas Tim**: 3 Backend Engineer (Go), 3 Frontend Engineer (Next.js/React), 1 QA / DevOps Engineer.

---

## Pembagian Milestone Sprint

### Minggu 1 (Hari 1 - 5): Fondasi Backend, Hub SSE & Standardisasi Komponen

#### Milestone 1.1: Backend Middleware RBAC & Entitlement (Hari 1 - 3)
- Pembaruan struktur middleware Go pada `internal/middleware/`.
- Penggantian cache memori lokal `globalCache` dengan Redis distributed versioned cache.
- Audit dan pengetatan seluruh 28 endpoint legacy (transaksi, anggaran, sasaran finansial, analitik, chat).
- Unit test & benchmark otorisasi middleware (< 2ms per-evaluasi).

#### Milestone 1.2: Hub Realtime SSE & Redis Pub/Sub (Hari 2 - 4)
- Pengembangan modul backend `internal/realtime/` (broker SSE, event broadcaster, redis pub/sub subscriber).
- Implementasi autentikasi cookie, heartbeat ping interval 15 detik, dan mekanisme recovery `Last-Event-ID`.
- Format penyiaran event domain standar (`payment_received`, `plan_updated`, `ai_endpoint_down`, `manual_approval_needed`, `quota_reset`).

#### Milestone 1.3: Standardisasi Frontend DataTable & Cleanup Legacy (Hari 3 - 5)
- Finalisasi komponen reusable `components/data-table/data-table.tsx` dengan integrasi TanStack Table v8.
- Migrasi tabel transaksi, anggaran, dan kategori lama ke komponen DataTable baru dengan URL query state.
- Penghapusan tuntas halaman, form, dan rute menu BYOK user (`/settings/ai-keys`).
- Pengarsipan skema database tabel `gemini_api_keys`.

---

### Minggu 2 (Hari 6 - 10): Integrasi Frontend Realtime, UI/UX Audit & Zero-Downtime Deployment

#### Milestone 2.1: Integrasi SSE Frontend & Surgical Cache Updates (Hari 6 - 8)
- Pembangunan hook `useRealtimeStream` dengan koneksi resilient, auto-reconnect exponential backoff, dan filter event.
- Integrasi penanganan event SSE ke TanStack Query cache via surgical `queryClient.invalidateQueries`.
- Implementasi optimistic update pada mutasi transaksi dan alokasi anggaran user.
- Pengujian simulasi koneksi drop, server restart, dan pemulihan event tertinggal.

#### Milestone 2.2: Audit Aksesibilitas (a11y) & Kepatuhan Cursor Pointer (Hari 7 - 9)
- Audit komprehensif elemen interaktif di seluruh halaman dashboard user dan super admin.
- Penegakan utilitas class `cursor-pointer`, `cursor-not-allowed`, dan pembersihan pointer pada container statis.
- Validasi WCAG 2.1 AA: kontras warna teks/badge, keyboard navigation (tabbing), ARIA label pada tombol ikon, dan focus trapping modal.

#### Milestone 2.3: Zero-Downtime Deployment & Operasional Readiness (Hari 8 - 10)
- Konfigurasi graceful shutdown 30 detik pada server Gin backend.
- Pembuatan dual health gate endpoint `/health/live` dan `/health/ready`.
- Pembuatan template pipeline CI/CD rolling update dengan zero packet loss.
- Simulasi skenario migrasi Expand/Contract pada skema database produksi.
- Uji beban end-to-end simulasi rilis versi N ke N+1 di bawah beban 200 RPS.

---

## Definisi Selesai Sprint (Sprint Definition of Done)
1. Seluruh 18 task backlog (TASK-P07-001 s/d TASK-P07-018) selesai dan berstatus closed.
2. Seluruh endpoint backend lulus verifikasi pengujian unit & integrasi RBAC/entitlement.
3. Koneksi SSE terbukti mampu pulih otomatis dan menerima kembali event tertinggal via `Last-Event-ID`.
4. Tidak ada query invalidation liar tanpa filter kunci spesifik pada TanStack Query frontend.
5. Halaman pengaturan BYOK user hilang 100% dari codebase antarmuka pengguna.
6. Seluruh halaman list data mengadopsi `<DataTable>` dengan state URL yang dapat di-bookmark.
7. Tidak ada elemen interaktif tanpa class `cursor-pointer` di seluruh aplikasi web.
8. Uji rolling deployment membuktikan ketersediaan sistem 100% tanpa error 5xx pada beban aktif.
