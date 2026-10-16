# Fase P07: Rollout RBAC, Realtime & Standar

## Konteks & Deskripsi Proyek
Fase P07 merupakan fase konsolidasi, standardisasi arsitektural, dan penyebaran menyeluruh (*rollout*) kapabilitas platform FinTrack v2 ke seluruh modul, endpoint, dan antarmuka pengguna. Setelah fondasi dasar (P02), AI gateway (P03), subscription & billing (P04), sistem pembayaran (P05), dan super admin dashboard (P06) selesai dibangun, P07 memastikan seluruh sistem beroperasi secara harmonis di bawah satu tata kelola akses terpadu dan transport realtime berstandar produksi.

FinTrack v2 adalah aplikasi keuangan pribadi (B2C) berbasis Go + Gin di backend, PostgreSQL 17 + pgvector + Redis di basis data/cache, dan Next.js 16 + TanStack Query v5 di frontend. Fase ini menghapus disparitas fitur warisan (legacy) P01, meniadakan menu/fitur Bring Your Own Key (BYOK) user secara tuntas, menggantikan cache in-memory lokal instans dengan arsitektur Redis pub/sub + versioned cache, serta menegakkan zero-downtime deployment menggunakan pola expand/contract dan rolling deploy dengan health gates.

## Ruang Lingkup (Scope)
1. **Penegakan RBAC & Entitlement Universal**:
   - Audit kepatuhan seluruh endpoint backend lama (transaksi, anggaran, sasaran tabungan, analitik, RAG/chat).
   - Penegakan otoritatif di layer middleware Go (`RequireRole`, `RequireEntitlement`, `RequireLimit`).
   - Cermin UX konsisten di frontend via hooks (`useEntitlements`, `useRole`) tanpa pernah memercayai klien untuk otorisasi final.
2. **Realtime Transport & Event Domain (Hub SSE + Redis Pub/Sub)**:
   - Modul `internal/realtime/` sebagai hub Server-Sent Events (SSE) terdistribusi.
   - Jembatan Redis pub/sub untuk sinkronisasi multi-instance.
   - Format standar event SSE: `payment_received`, `plan_updated`, `ai_endpoint_down`, `manual_approval_needed`, `quota_reset`.
   - Fitur resilience SSE: autentikasi via secure HTTP-only cookie, resume event hilang dengan `Last-Event-ID`, heartbeat ping 15 detik, dan exponential backoff reconnect pada frontend.
3. **Surgical Cache Update & Invalidation Strategy**:
   - Pola mutasi frontend: aksi pengguna langsung menggunakan TanStack Query `setQueryData` / optimistic update.
   - Event eksternal (SSE / webhook) memicu invalidasi surgical granular berdasarkan query key hierarkis, mencegah refetch global massal (*cache storm*).
   - Penggantian total `globalCache` in-memory lokal backend dengan Redis distributed cache berversi (K-14 & K-21).
4. **Pembersihan Fitur Warisan & Standar UX/A11y**:
   - Penghapusan tuntas menu Bring Your Own Key (BYOK) user di frontend; pengarsipan data kunci lama di database (`gemini_api_keys`).
   - Standardisasi tabel ke komponen `<DataTable>` berbasis TanStack Table v8 dengan URL state query (`?page=&sort=&order=&q=&filter`).
   - Audit universal cursor pointer: elemen klik (`button`, `a`, checkbox, baris tabel interaktif) wajib `cursor-pointer`; elemen non-klik/disabled wajib `cursor-default`/`cursor-not-allowed`.
   - Audit aksesibilitas (a11y) WCAG 2.1 AA: kontras warna, keyboard navigation, focus trap modal, ARIA attributes.
5. **Zero-Downtime Deployment & Operasional**:
   - Pola migrasi database Expand/Contract (multi-phase schema migration).
   - Rolling update orchestration di Docker/k8s environment dengan zero packet drop.
   - Readiness & Liveness health gates terpisah (`/health/live`, `/health/ready`).
   - Feature flags dinamis berbasis Redis untuk gradual rollout dan instan circuit kill switch.

## Deliverables
> 📦 **Arsip Detail P07:** Seluruh dokumen analisis keputusan (K-P07-01 s/d K-P07-16), planning, rules, teknis (RBAC, SSE, cache invalidation, cursor pointer, zero-downtime, entitlement mirror), dan tracking tersimpan lengkap di [`../../5_Archives/phases/P07_Rollout-RBAC-Realtime/`](../../5_Archives/phases/P07_Rollout-RBAC-Realtime/).

- **Dokumentasi Teknis Inti (Diarsipkan)**:
  - `teknis/rbac-rollout.md`: Spesifikasi audit endpoint, middleware Go, dan matriks role/entitlement seluruh rute.
  - `teknis/sse-strategy.md`: Arsitektur hub SSE, protokol koneksi, reconnect, auth cookie, dan spesifikasi event domain.
  - `teknis/cache-invalidation.md`: Desain invalidasi Redis pub/sub, TanStack Query cache key registry, dan pola surgical update.
  - `teknis/cursor-pointer-audit.md`: Laporan audit interaktivitas UI, perbaikan pointer class, dan standar interaktivitas token styling.
  - `teknis/zero-downtime-deploy.md`: Prosedur migrasi expand/contract, rolling deploy, health checks, dan arsitektur feature flag.
  - `teknis/frontend-entitlement-mirror.md`: Panduan implementasi client mirror UX, guard component, menu filtering, dan fallback banner.
- **Dokumentasi Perencanaan & Kontrak (Diarsipkan)**:
  - `analisis/hasil-analisis-teknis.md`: Keputusan teknis arsitektur P07 (K-P07-01 hingga K-P07-16).
  - `planning/sprint-plan.md`: Alokasi sprint 2 minggu, milestone, dan resource plan.
  - `planning/tasks.md`: Rincian task backlog komprehensif (TASK-P07-001 hingga TASK-P07-018).
  - `planning/timeline.md`: Gantt-flow, dependensi antar-task, dan jalur kritis (critical path).
  - `rules/implementation-rules.md`: Aturan teknis absolut yang mengikat seluruh developer selama rollout.
  - `tracking/tracking.md`: Status pelacakan real-time pengerjaan deliverable.

## Kriteria Sukses (Success Criteria)
1. **Otorisasi 100% Otoritatif**: Seluruh 28 endpoint legacy & modul baru terlindungi middleware RBAC & entitlement berversi; tidak ada celah bypass URL atau manipulasi token JWT.
2. **Transisi Realtime Mulus**: Event pembayaran eksternal atau pembaruan paket dari super admin terefleksikan di UI klien dalam waktu < 500ms via SSE tanpa refresh manual.
3. **Resilience Koneksi SSE**: Koneksi SSE yang terputus (jaringan mati / server restart) otomatis tersambung kembali via backoff dengan pemulihan backlog event menggunakan `Last-Event-ID` tanpa kehilangan data.
4. **Zero-Cache Inconsistency**: Multi-instance backend sinkron 100% via Redis pub/sub invalidation; tidak ada anomali stale state antar instans.
5. **Kebersihan Legacy**: 0 referensi aktif menu BYOK di antarmuka pengguna; data lama diarsipkan aman; semua tabel mengadopsi `<DataTable>` dengan bookmarkable URL parameters.
6. **Zero-Downtime Deployment**: Uji simulasi migrasi skema dan rolling restart backend menghasilkan 0 request HTTP 5xx pada beban aktif 200 RPS.
7. **Kepatuhan UI/UX**: Lolos audit cursor-pointer dan a11y di 100% halaman modul dashboard pengguna dan super admin.
