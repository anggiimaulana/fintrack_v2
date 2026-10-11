# Rincian Task Backlog: Fase P07 — Rollout RBAC, Realtime & Standar

Berikut adalah rincian task teknis untuk fase P07 FinTrack v2 dengan format standar `TASK-P07-NNN`:

---

### TASK-P07-001: Implementasi Middleware Otorisasi Entitlement & Limit di Backend Go
- **Deskripsi**: Bangun middleware `RequireEntitlement(featureKey string)` dan `RequireLimit(limitKey string, currentUsageFn)` pada `internal/middleware/entitlement.go`. Middleware membaca konteks pengguna dari JWT, mengecek cache versi Redis `user:entitlement_ver:{uid}`, dan memvalidasi izin fitur atau batas kuota kuantitatif.
- **Tipe**: Backend / Security
- **File Target**: `backend/internal/middleware/entitlement.go`, `backend/internal/middleware/rbac.go`
- **Acceptance Criteria**:
  - Request diblokir dengan HTTP 403 Forbidden bila entitlement boolean bernilai false.
  - Request diblokir dengan HTTP 429 Too Many Requests bila batas numerik tercapai.
  - Latensi evaluasi middleware < 2ms menggunakan Redis pipelined lookups.

### TASK-P07-002: Audit & Pemasangan Proteksi Entitlement ke Seluruh Endpoint Legacy P01
- **Deskripsi**: Pasang middleware otorisasi baru ke seluruh rute transaksi, anggaran, analitik, dan chat AI warisan P01 pada `backend/internal/api/routes.go`.
- **Tipe**: Backend / Integration
- **File Target**: `backend/internal/api/routes.go`, `backend/internal/api/handler/*.go`
- **Acceptance Criteria**:
  - 100% dari 28 endpoint terdaftar memiliki pengawalan middleware yang sesuai dengan matriks hak akses.
  - Endpoint transaksi gratis dibatasi kuota siklus; transaksi pro/unlimited terbuka sesuai plan.
  - Seluruh rute chat/AI terikat verifikasi saldo token dan fitur `ai_chat`.

### TASK-P07-003: Migrasi In-Memory globalCache Lokal ke Redis Distributed Cache Berversi
- **Deskripsi**: Hapus variabel `globalCache` lokal berbasis memori tunggal pada backend Go. Gantikan dengan service Redis client terstandarisasi yang mendukung namespace berversi (`ft:{env}:{module}:{entity}:{id}`) dan invalidasi instan via bump counter versi.
- **Tipe**: Backend / Infra
- **File Target**: `backend/internal/cache/redis.go`, `backend/internal/cache/cache.go`
- **Acceptance Criteria**:
  - Nol referensi `sync.Map` atau slice memori global untuk data pengguna antar-request.
  - Fungsi `InvalidateUserCache(ctx, userID)` menaikkan versi integer Redis dan menghapus entri kadaluwarsa.
  - Lolos uji integrasi multi-instance tanpa stale cache.

### TASK-P07-004: Pengembangan Modul Hub Server-Sent Events (SSE) `internal/realtime/`
- **Deskripsi**: Bangun modul `internal/realtime/` yang bertindak sebagai broker koneksi klien HTTP SSE. Menyediakan manajemen thread-safe connection map per-user, pengiriman stream text/event-stream, dan isolasi buffer per-klien.
- **Tipe**: Backend / Realtime
- **File Target**: `backend/internal/realtime/hub.go`, `backend/internal/realtime/client.go`
- **Acceptance Criteria**:
  - Handler SSE menetapkan header: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no`.
  - Klien dapat tersambung dan menerima pesan instan saat event ditujukan untuk userID tersebut.
  - Resource connection dibersihkan dengan benar saat koneksi TCP terputus (`ctx.Done()`).

### TASK-P07-005: Integrasi Redis Pub/Sub sebagai Backbone Distribusi SSE Lintas Instance
- **Deskripsi**: Hubungkan broker SSE `internal/realtime/` dengan listener Redis Pub/Sub pada channel `realtime:events:{channel}`. Saat instance A mempublikasikan event, listener pada instance B, C, D menerima pesan dan mendistribusikannya ke klien SSE lokal masing-masing.
- **Tipe**: Backend / Distributed Systems
- **File Target**: `backend/internal/realtime/redis_sub.go`, `backend/internal/realtime/publisher.go`
- **Acceptance Criteria**:
  - Broadcast event dari pod A sampai ke klien yang tersambung di pod B dalam waktu < 50ms.
  - Reconnect subscriber Redis otomatis aktif bila koneksi Redis terputus sementara.

### TASK-P07-006: Penguatan Resiliensi SSE: Cookie Auth, 15s Heartbeat, dan Resume Last-Event-ID
- **Deskripsi**: Terapkan proteksi autentikasi SSE via HTTP-Only cookie `fintrack_token`. Tambahkan background goroutine yang mengirim ping heartbeat komentar (`: ping\n\n`) setiap 15 detik. Simpan ring-buffer event di Redis (TTL 10 menit) untuk melayani resume stream via header `Last-Event-ID`.
- **Tipe**: Backend / Reliability
- **File Target**: `backend/internal/realtime/handler.go`, `backend/internal/realtime/buffer.go`
- **Acceptance Criteria**:
  - Akses SSE tanpa valid cookie ditolak dengan HTTP 401 Unauthorized.
  - Klien tidak mengalami pemutusan timeout oleh reverse proxy saat idle berkat heartbeat 15s.
  - Klien yang terputus selama 60 detik dapat menyambung kembali dengan `Last-Event-ID` dan menerima seluruh event yang terlewat secara berurutan.

### TASK-P07-007: Standardisasi Skema JSON Event Domain SSE FinTrack v2
- **Deskripsi**: Definisikan tipe data Go dan skema event payload untuk event sistem: `payment_received`, `plan_updated`, `ai_endpoint_down`, `manual_approval_needed`, dan `quota_reset`.
- **Tipe**: Backend / Shared Contracts
- **File Target**: `backend/internal/realtime/events.go`, `backend/pkg/contracts/events.go`
- **Acceptance Criteria**:
  - Seluruh payload tervalidasi skema JSON dan menyertakan field `event`, `timestamp`, `id`, serta objek data `payload`.
  - Terintegrasi langsung dengan pemanggilan di modul billing, payment webhook, dan AI gateway.

### TASK-P07-008: Pembangunan Frontend Hook `useRealtimeStream` & Auto-Reconnect
- **Deskripsi**: Buat React hook `useRealtimeStream` pada frontend Next.js 16. Hook mengelola lifecycle `EventSource`, menangani pemutusan koneksi dengan strategi exponential backoff jitter (1s, 2s, 4s, maks 16s), dan mendistribusikan event ke TanStack Query subscriber.
- **Tipe**: Frontend / Realtime
- **File Target**: `frontend/src/hooks/use-realtime-stream.ts`, `frontend/src/providers/realtime-provider.tsx`
- **Acceptance Criteria**:
  - Koneksi otomatis terinisialisasi saat pengguna login dan terputus saat pengguna logout.
  - Reconnect otomatis berfungsi transparan saat server restart tanpa intervensi pengguna.
  - Browser console bersih dari pesan error unhandled stream termination.

### TASK-P07-009: Implementasi Surgical Cache Invalidation TanStack Query pada Frontend
- **Deskripsi**: Susun peta registri query key terpusat dan kaitkan dispatcher SSE ke invalidasi bedah. Saat event `payment_received` masuk, hanya query key `['billing', 'invoices']` dan `['subscriptions', 'current']` yang di-invalidate, bukan seluruh root cache.
- **Tipe**: Frontend / State Management
- **File Target**: `frontend/src/lib/query-keys.ts`, `frontend/src/lib/realtime-dispatcher.ts`
- **Acceptance Criteria**:
  - Tidak ada pemanggilan liar `queryClient.invalidateQueries()` tanpa queryKey filter spesifik.
  - Perubahan data transaksi instan muncul di layar pengguna lain yang memonitor dalam < 500ms.
  - Meniadakan refetch berulang untuk data yang tidak terpengaruh.

### TASK-P07-010: Pola Optimistic Update pada Mutasi Transaksi & Anggaran Pengguna
- **Deskripsi**: Terapkan metode `onMutate`, `onError`, dan `onSettled` pada TanStack Query mutation transaksi (`useCreateTransaction`, `useUpdateTransaction`, `useDeleteTransaction`). Data di antarmuka langsung diperbarui sebelum respon server selesai.
- **Tipe**: Frontend / UX Performance
- **File Target**: `frontend/src/hooks/use-transactions.ts`, `frontend/src/hooks/use-budgets.ts`
- **Acceptance Criteria**:
  - Latensi persepsi pengguna 0ms saat menambah atau menyunting transaksi.
  - Bila server mengembalikan error (misal HTTP 409 Optimistic Lock conflict atau HTTP 429), rollback state UI ke kondisi semula secara mulus dengan notifikasi toast error.

### TASK-P07-011: Standardisasi Komponen `<DataTable>` TanStack Table v8 Universal
- **Deskripsi**: Lengkapi komponen `components/data-table/data-table.tsx` dengan fitur pagination 20 baris, sorting kolom, multi-filter, search debounce 300ms, serta sinkronisasi state dua arah ke URL query params Next.js App Router (`useSearchParams`).
- **Tipe**: Frontend / UI Components
- **File Target**: `frontend/src/components/data-table/data-table.tsx`, `frontend/src/components/data-table/data-table-pagination.tsx`, `frontend/src/components/data-table/data-table-toolbar.tsx`
- **Acceptance Criteria**:
  - URL browser otomatis merefleksikan filter dan halaman saat ini (contoh: `?page=2&sort=created_at&order=desc`).
  - Halaman yang di-refresh atau link yang dibagikan mempertahankan posisi tabel dan filter yang sama persis.
  - Mendukung empty state, skeleton loading, dan error boundary.

### TASK-P07-012: Refactor Seluruh Halaman List Data Lama Menggunakan `<DataTable>`
- **Deskripsi**: Ganti implementasi tabel kustom atau list statis lama pada halaman Transaksi (`/transactions`), Anggaran (`/budgets`), dan Kategori (`/categories`) dengan komponen baru `<DataTable>`.
- **Tipe**: Frontend / Refactor
- **File Target**: `frontend/src/app/(dashboard)/transactions/page.tsx`, `frontend/src/app/(dashboard)/budgets/page.tsx`, `frontend/src/app/(dashboard)/categories/page.tsx`
- **Acceptance Criteria**:
  - Seluruh halaman list mengimplementasikan search, pagination 20 baris, dan sorting yang konsisten.
  - Desain visual selaras dengan tabel super admin dashboard P06.
  - Lolos uji integrasi form filter dan penomoran baris global.

### TASK-P07-013: Pembersihan Tuntas Menu & Kode Bring Your Own Key (BYOK)
- **Deskripsi**: Hapus halaman `/settings/ai-keys`, link navigasi sidebar/navbar, serta dialog form input API Key Gemini user. Pada backend, hapus rute registrasi user key dan buat migrasi database untuk mengarsipkan tabel `gemini_api_keys` (`is_archived = true`).
- **Tipe**: Fullstack / Deprecation
- **File Target**: `frontend/src/components/sidebar.tsx`, `frontend/src/app/(dashboard)/settings/ai-keys/`, `backend/internal/api/routes.go`, `backend/database/migrations/0009_archive_byok_keys.sql`
- **Acceptance Criteria**:
  - 0 kemunculan antarmuka input API Key pengguna di seluruh frontend.
  - Rute lama `/settings/ai-keys` mengembalikan HTTP 404 atau redirect permanen HTTP 301 ke `/settings/subscription`.
  - Data kunci lama pengguna tersimpan aman dalam status arsip tanpa kebocoran data.

### TASK-P07-014: Audit Universal Cursor Pointer & State Interaktivitas UI
- **Deskripsi**: Lakukan penelusuran seluruh file TSX di folder `components/` dan `app/`. Terapkan kelas `cursor-pointer` pada seluruh elemen interaktif, `cursor-not-allowed` pada elemen disabled, dan pastikan container statis berstatus `cursor-default`.
- **Tipe**: Frontend / UI Styling
- **File Target**: Seluruh berkas `.tsx` pada `frontend/src/components/` dan `frontend/src/app/`
- **Acceptance Criteria**:
  - Seluruh tombol, link, tab navigasi, checkbox, radio button, dan collapsible header memiliki kursor pointer saat diarahkan pointer mouse.
  - Komponen dinonaktifkan menampilkan kursor not-allowed dan opacity tereduksi.
  - Lolos pengujian manual visual hover review di seluruh 14 halaman dashboard.

### TASK-P07-015: Audit Kepatuhan Aksesibilitas (a11y) WCAG 2.1 Level AA
- **Deskripsi**: Pasang atribut `aria-label` pada seluruh tombol berbasis ikon saja (tanpa teks). Terapkan focus trap pada dialog/modal dan pastikan navigasi keyboard (Tab, Shift+Tab, Enter, Space, Escape) berfungsi sempurna. Validasi rasio kontras warna pada tema gelap dan terang.
- **Tipe**: Frontend / Accessibility
- **File Target**: Komponen modal/dialog, button, dropdown, dan form input
- **Acceptance Criteria**:
  - Skor Lighthouse Accessibility >= 95 di semua rute utama.
  - Dialog modal dapat ditutup dengan menekan tombol Escape dan fokus keyboard terkurung di dalam modal saat terbuka.
  - Semua tombol icon-only memiliki accessible name yang terbaca oleh screen reader.

### TASK-P07-016: Implementasi Dual Health Gate Endpoint (`/health/live` & `/health/ready`)
- **Deskripsi**: Bangun handler khusus di backend Go untuk probe orchestrator. `/health/live` memvalidasi proses hidup. `/health/ready` memvalidasi koneksi aktif ke PostgreSQL (termasuk verifikasi ekstensi pgvector) dan koneksi Redis ping.
- **Tipe**: Backend / DevOps
- **File Target**: `backend/internal/api/health.go`, `backend/cmd/server/main.go`
- **Acceptance Criteria**:
  - `/health/live` merespons HTTP 200 OK secara instan (< 1ms).
  - `/health/ready` merespons HTTP 200 OK jika PostgreSQL dan Redis sehat; merespons HTTP 503 Service Unavailable bila salah satu dependensi mati.
  - Output berupa JSON diagnostik waktu respon dependensi.

### TASK-P07-017: Konfigurasi Graceful Shutdown & Zero Packet Loss Rolling Deploy
- **Deskripsi**: Konfigurasikan penanganan sinyal OS (`SIGTERM`, `SIGINT`) pada backend Gin dengan batas waktu drain 30 detik. Sesuaikan file konfigurasi Docker Compose dan template Kubernetes ingress/deployment dengan parameter `terminationGracePeriodSeconds` dan readiness probe.
- **Tipe**: DevOps / Reliability
- **File Target**: `backend/cmd/server/main.go`, `docker-compose.yml`, `deploy/k8s/deployment.yaml`
- **Acceptance Criteria**:
  - Saat container menerima sinyal stop, server menolak request baru dengan drain mulus dan menyelesaikan transaksi/SSE yang sedang aktif sampai selesai.
  - Uji rolling restart container di bawah beban konstan tidak menghasilkan connection dropped atau HTTP 502/504 pada reverse proxy Nginx.

### TASK-P07-018: Standardisasi Prosedur Migrasi Skema Database Expand/Contract
- **Deskripsi**: Tulis dokumen panduan operasional dan terapkan template migrasi expand/contract untuk skema database masa depan FinTrack v2. Pastikan seluruh migrasi SQL bersifat non-blocking (`ADD COLUMN` nullable, pembuatan indeks concurrent `CREATE INDEX CONCURRENTLY`).
- **Tipe**: Database / DevOps
- **File Target**: `backend/database/migrations/README.md`, `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/zero-downtime-deploy.md`
- **Acceptance Criteria**:
  - Aturan tertulis melarang eksekusi `DROP COLUMN` atau `ALTER TABLE ... NOT NULL` tanpa fase perantara default.
  - Skrip migrasi terbukti dapat dijalankan maju (up) dan mundur (down) tanpa merusak aplikasi versi lama yang sedang berjalan paralel.
