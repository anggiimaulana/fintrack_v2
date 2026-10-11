# Analisis Teknis & Keputusan Arsitektur Fase P07

## Latar Belakang & Masalah
Fase P02 sampai P06 berhasil membangun subsistem fondasi: skema multi-role, billing engine, AI gateway, dan dashboard manajemen super admin. Namun, terdapat celah arsitektural sebelum sistem dapat dinyatakan siap produksi:
1. **Disparitas Proteksi Antar-Endpoint**: Modul lama P01 (transaksi, budgeting, sasaran finansial) hanya memeriksa autentikasi token JWT dasar tanpa verifikasi entitlement paket dinamis atau batasan kuota numerik.
2. **Keterbatasan Cache In-Memory Lokal**: Implementasi cache sebelumnya mengandalkan peta memori proses tunggal (`globalCache`). Pada arsitektur multi-instance Docker/k8s, invalidasi di instance A tidak menjangkau instance B, memicu status basi (*stale data*).
3. **Respon Lambat terhadap Event Eksternal**: Perubahan status transaksi oleh webhook Midtrans/Xendit atau penyesuaian plan oleh super admin memerlukan refresh halaman manual oleh pengguna.
4. **Residu Fitur BYOK**: Komponen frontend P01 masih memuat antarmuka input Gemini API Key pengguna perorangan, bertentangan dengan K-16 (penghapusan BYOK terpusat via AI Gateway).
5. **Inkonsistensi Komponen Tabel & Interaktivitas UI**: Sebagian tabel lama masih memakai rendering statis tanpa dukungan query URL parameter dan memiliki elemen klik tanpa atribut kursor pointer standar.

## Keputusan Teknis Terkunci (K-P07-01 s/d K-P07-16)

### K-P07-01: Arsitektur Otorisasi Dual-Layer (Middleware Backend Otoritatif + Client Mirror)
- **Keputusan**: Penegakan hak akses 100% diputuskan di backend menggunakan middleware Go (`RequireRole`, `RequireEntitlement`, `RequireLimit`). Frontend hanya berfungsi sebagai cermin visual (UX helper) via `useEntitlements()` dan `<EntitlementGuard>`.
- **Rasional**: Memenuhi prinsip K-13 & K-14. Manipulasi state di sisi browser tidak boleh membuka akses ke data atau operasi berbayar.
- **Implementasi**: Middleware membaca identitas pengguna dari JWT, lalu mengecek cache berversi entitlement Redis (`user:entitlements:{uid}:{ver}`). Jika cache kosong, fallback ke query database terindeks.

### K-P07-02: Versioned Invalidation Cache RBAC & Entitlement
- **Keputusan**: Hak akses dan limit pengguna tidak disimpan dalam klaim token JWT, melainkan direferensikan melalui pointer versi integer (`entitlement_version`) pada Redis.
- **Rasional**: Perubahan paket atau role oleh super admin harus berefek instan tanpa memaksa pengguna logout atau menunggu token JWT kedaluwarsa (K-14 & K-15).
- **Mekanisme**: Setiap mutasi langganan atau role memicu increment `INCR user:entitlement_ver:{uid}` dan menghapus payload cache lama via Redis pipeline.

### K-P07-03: Realtime Engine Terpusat via Server-Sent Events (SSE) & Jembatan Redis Pub/Sub
- **Keputusan**: Menggunakan protokol SSE murni berbasis HTTP/2 streaming untuk komunikasi server-ke-klien satu arah, dikombinasikan dengan Redis Pub/Sub (`realtime:events:{channel}`) untuk distribusi multi-pod backend.
- **Rasional**: FinTrack v2 tidak membutuhkan komunikasi duplex berbobot tinggi seperti WebSockets. SSE kompatibel dengan HTTP proxy standar, ringan, hemat konsumsi baterai/koneksi, dan memiliki fitur native reconnect di browser (`EventSource`).

### K-P07-04: Autentikasi Koneksi SSE via Secure HTTP-Only Cookie
- **Keputusan**: Endpoint SSE (`GET /api/v1/realtime/stream`) diautentikasi secara eksklusif melalui HTTP-Only cookie `fintrack_token` dengan flag `SameSite=Lax` dan `Secure`. Pengiriman token lewat URL query parameter (`?token=...`) dilarang keras.
- **Rasional**: Mencegah kebocoran token pada log server, riwayat browser, dan monitoring jaringan (mitigasi OWASP A01/A02).

### K-P07-05: Mekanisme Reconnect & Resiliensi SSE dengan Heartbeat 15 Detik dan Last-Event-ID
- **Keputusan**: Server mengirim event komentar heartbeat (`: ping\n\n`) setiap 15 detik untuk mencegah pemutusan idle timeout oleh Nginx/Cloudflare/AWS ALB. Setiap event domain diberi nomor urut monotoonik per-user/sistem (`id: {sequence_id}`).
- **Rasional**: Saat koneksi terputus dan browser menyambung kembali dengan header `Last-Event-ID`, modul `internal/realtime/` mengambil delta event yang belum diterima dari ring buffer Redis (retensi 10 menit) lalu mengirimkannya kembali ke klien.

### K-P07-06: Domain Event SSE Standar FinTrack v2
- **Keputusan**: Seluruh event realtime diseragamkan dengan format JSON terstruktur:
  ```json
  {
    "event": "payment_received | plan_updated | ai_endpoint_down | manual_approval_needed | quota_reset",
    "timestamp": 1773294800,
    "payload": { ... }
  }
  ```
- **Rasional**: Memastikan konsistensi penanganan event oleh TanStack Query di sisi frontend dan memudahkan audit event stream.

### K-P07-07: Strategi Surgical Cache Update Frontend (Direct Action vs SSE Event)
- **Keputusan**:
  1. **Direct User Action**: Menggunakan TanStack Query `queryClient.setQueryData()` untuk mutasi optimistik instan pada layar pengguna yang bersangkutan.
  2. **External Event (via SSE)**: Handler SSE memicu invalidasi surgical granular (`queryClient.invalidateQueries({ queryKey: [...] })`) hanya pada entitas yang terpengaruh, bukan `invalidateQueries()` global tanpa filter.
- **Rasional**: Menjaga aplikasi tetap responsif (sub-100ms UI update) sekaligus menghindari lonjakan query serentak (*thundering herd / cache storm*) ke backend saat event eksternal tiba.

### K-P07-08: Terminasi Total `globalCache` Lokal Menuju Redis Cluster / Replica
- **Keputusan**: Seluruh struct cache memori Go lokal (`sync.Map` atau slice global) di backend dihapus. Seluruh penyimpanan cache dialihkan ke Redis dengan key namespace standar `ft:{env}:{module}:{entity}:{id}` dan TTL wajib.
- **Rasional**: Memenuhi K-21 dan K-23. Memungkinkan backend dijalankan secara horizontal scaling (N instance) tanpa inkonsistensi cache.

### K-P07-09: Deprecate & Clean-up Total Menu/Fitur BYOK (Bring Your Own Key)
- **Keputusan**:
  - Hapus tuntas menu, halaman, dan form pengaturan BYOK di frontend (`/settings/ai-keys`).
  - Hapus endpoint publik input kunci API user di backend (`POST /api/v1/user/gemini-keys`).
  - Arsipkan tabel database `gemini_api_keys` dengan mengubah flag `is_archived = true` (K-16); jangan pernah drop tabel ini sebelum audit P08 selesai.
  - Alihkan 100% lalu lintas AI (chat, vision, voice, embedding) melalui modul AI Gateway terpusat (`internal/aigateway`).

### K-P07-10: Standardisasi Komponen `<DataTable>` TanStack Table v8 Universal
- **Keputusan**: Seluruh halaman list data di FinTrack v2 (transaksi, anggaran, audit log, invoice, users, plans) wajib menggunakan komponen bersama `components/data-table/data-table.tsx`.
- **Rasional**: Menyediakan UX konsisten: pagination default 20 baris, sorting multi-kolom, pencarian instan debounce 300ms, filter faceted, dan sinkronisasi dua arah ke URL search params (`?page=1&sort=amount&order=desc&q=makan`). Bookmarkable dan shareable.

### K-P07-11: Standar Interaktivitas UI & Kepatuhan Universal Cursor Pointer
- **Keputusan**:
  - Semua elemen interaktif (`<button>`, `<a href>`, input radio/checkbox, collapsible trigger, pagination item, baris tabel ber-klik) WAJIB memiliki kelas CSS `cursor-pointer`.
  - Tombol dalam kondisi disabled WAJIB memiliki kelas `cursor-not-allowed` dan `opacity-50`.
  - Kontainer statis, kartu informasi non-klik, dan label biasa dilarang keras memakai `cursor-pointer`.
- **Rasional**: Mengatasi friksi UX utama di mana pengguna tidak dapat membedakan elemen statis dan interaktif secara visual.

### K-P07-12: Kepatuhan Aksesibilitas Web (WCAG 2.1 Level AA)
- **Keputusan**:
  - Rasio kontras teks terhadap latar belakang minimal 4.5:1 untuk teks normal dan 3.0:1 untuk teks besar/badge.
  - Semua tombol berbasis ikon wajib menyertakan atribut `aria-label` deskriptif.
  - Komponen modal dan sheet dialog wajib menerapkan *focus trap* dan dapat ditutup via tombol Escape (`keydown: Escape`).

### K-P07-13: Arsitektur Zero-Downtime Deployment (Expand / Contract Schema Pattern)
- **Keputusan**: Perubahan skema basis data PostgreSQL wajib melalui 2 fase deploy terpisah:
  1. **Fase Expand**: Tambahkan kolom baru (nullable atau bernilai default) tanpa menghapus kolom lama. Deploy backend versi N+1 yang membaca kolom baru dan menulis ke kedua kolom (dual-write).
  2. **Fase Contract**: Jalankan migrasi data latar belakang, lalu jalankan deploy backend versi N+2 yang hanya membaca/menulis kolom baru. Terakhir, jalankan skrip migrasi untuk drop kolom lama.
- **Rasional**: Meniadakan lock table eksklusif dan downtime aplikasi saat migrasi database dijalankan (K-23).

### K-P07-14: Orchestration Rolling Deploy & Graceful Shutdown
- **Keputusan**: Backend Go wajib menangani sinyal `SIGINT` dan `SIGTERM` dengan `server.Shutdown(ctx)` berbatas waktu drain 30 detik.
- **Rasional**: Koneksi HTTP aktif dan streaming SSE yang sedang berjalan diselesaikan secara rapi tanpa pemutusan paksa (connection reset by peer). Load balancer mendeteksi status shutting down dan mengalihkan traffic ke pod sehat.

### K-P07-15: Dual Health Gate (`/health/live` vs `/health/ready`)
- **Keputusan**:
  - `/health/live`: Mengembalikan status HTTP 200 selama proses runtime Go tidak deadlock.
  - `/health/ready`: Memeriksa konektivitas koneksi pool PostgreSQL, pgvector extension, dan Redis ping. Jika salah satu dependensi kritis gagal, kembalikan HTTP 503.
- **Rasional**: Memastikan orchestrator container (Docker Compose / Kubernetes) tidak mengalirkan trafik pengguna ke pod yang belum selesai inisialisasi atau kehilangan koneksi database.

### K-P07-16: Dynamic Feature Flag Engine Terpusat
- **Keputusan**: Fitur-fitur baru dan rute rollout P07 dikontrol oleh feature flag dinamis dari tabel `feature_flags` yang di-cache di Redis. Flag dapat diaktifkan/dinonaktifkan secara instan oleh Super Admin tanpa perlu deployment ulang kode.
- **Rasional**: Memungkinkan canary rollout, A/B testing, dan instant emergency kill-switch bila ditemukan anomali performa.
