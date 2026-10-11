# Aturan Implementasi Teknis yang Mengikat (Implementation Rules) - Fase P07

Dokumen ini memuat aturan rekayasa perangkat lunak yang **WAJIB (BINDING)** dipatuhi oleh seluruh pengembang selama fase implementasi P07 FinTrack v2. Pelanggaran terhadap aturan ini akan ditolak langsung pada tahap Code Review / Pull Request.

---

## 1. Aturan Otorisasi & Keamanan (RBAC & Entitlement)

- **RULE-P07-SEC-01**: **Backend adalah Otoritas Tunggal Mutlak**. Klien frontend tidak pernah dipercaya. Validasi kepemilikan data (`user_id`), kecukupan kuota, dan izin fitur wajib dijalankan di layer middleware dan handler Go.
- **RULE-P07-SEC-02**: **Dilarang Menyimpan Entitlement / Role di Payload JWT**. JWT hanya memuat identitas subjek (`sub`), issuer, tanggal kedaluwarsa (`exp`), dan versi sesi (`session_ver`). Data hak akses dan kuota wajib diambil dari Redis cache berversi atau database per-request.
- **RULE-P07-SEC-03**: **Format Response Error Terstandarisasi**:
  - Pelanggaran izin role: kembalikan `HTTP 403 Forbidden` dengan format JSON `{ "error": "FORBIDDEN", "message": "Akses memerlukan role super_admin" }`.
  - Pelanggaran izin fitur: kembalikan `HTTP 403 Forbidden` dengan kode spesifik `{ "error": "ENTITLEMENT_REQUIRED", "feature": "export_csv", "upgrade_url": "/settings/subscription" }`.
  - Batas kuota habis: kembalikan `HTTP 429 Too Many Requests` dengan header `Retry-After` dan payload `{ "error": "LIMIT_EXCEEDED", "limit_type": "monthly_transactions", "current": 50, "max": 50 }`.

---

## 2. Aturan Server-Sent Events (SSE) & Realtime

- **RULE-P07-SSE-01**: **Autentikasi SSE Eksklusif via HTTP-Only Cookie**. Dilarang menerima token melalui URL Query Parameter (misal `?token=...` atau `?jwt=...`). Endpoint `/api/v1/realtime/stream` wajib membaca cookie `fintrack_token`.
- **RULE-P07-SSE-02**: **Header Wajib Response SSE**:
  Setiap stream SSE wajib mengembalikan header:
  ```http
  Content-Type: text/event-stream
  Cache-Control: no-cache, no-transform
  Connection: keep-alive
  X-Accel-Buffering: no
  ```
- **RULE-P07-SSE-03**: **Heartbeat Ping Wajib 15 Detik**. Server wajib memancarkan komentar baris tunggal `: ping\n\n` tepat setiap 15 detik pada channel terbuka untuk mencegah idle timeout oleh Cloudflare / AWS ALB / Nginx.
- **RULE-P07-SSE-04**: **Idempotensi & Resumability Stream**. Setiap event SSE wajib memuat ID numerik menaik (`id: {sequence_id}`). Server wajib mengimplementasikan pembacaan header request `Last-Event-ID` untuk memancarkan ulang event tertinggal dari Redis buffer (retensi 10 menit).

---

## 3. Aturan Manajemen Cache & TanStack Query

- **RULE-P07-CACHE-01**: **Penghapusan Total Cache Memori Lokal di Backend**. Dilarang membuat variabel `sync.Map`, global slice, atau map Go untuk menyimpan state pengguna lintas HTTP request. Semua caching wajib menggunakan Redis dengan TTL terdefinisi.
- **RULE-P07-CACHE-02**: **Namespace Kunci Redis Wajib Standar**:
  Format kunci Redis wajib mengikuti:
  `ft:{env}:{module}:{entity}:{identifier}`
  Contoh: `ft:prod:entitlement:user:usr_99182`, `ft:prod:realtime:buffer:usr_99182`.
- **RULE-P07-CACHE-03**: **Larangan Invalidation Liar di Frontend**. Pada TanStack Query, DILARANG memanggil `queryClient.invalidateQueries()` tanpa queryKey atau predicate spesifik. Setiap invalidasi harus surgical (contoh: `queryClient.invalidateQueries({ queryKey: queryKeys.billing.invoices })`).
- **RULE-P07-CACHE-04**: **Optimistic Updates Wajib Rollback Bersih**. Setiap mutasi optimistik wajib menyimpan snapshot konteks sebelumnya di `onMutate`, menerapkan rollback di `onError`, dan melakukan invalidasi konfirmasi di `onSettled`.

---

## 4. Aturan Komponen UI, DataTable & Interaktivitas

- **RULE-P07-UI-01**: **Standardisasi `<DataTable>` TanStack Table v8**. Seluruh daftar data berbasis tabel wajib menggunakan komponen `components/data-table/data-table.tsx`. Dilarang menulis implementasi tabel manual dari nol (HTML `<table>` mentah tanpa sorting/pagination URL).
- **RULE-P07-UI-02**: **State Tabel Wajib Sinkron dengan URL Search Params**. Perubahan nomor halaman (`page`), ukuran halaman (`limit`), kolom pengurutan (`sort`), arah pengurutan (`order`), filter facet, dan input pencarian (`q`) wajib terefleksikan di URL parameter secara otomatis.
- **RULE-P07-UI-03**: **Disiplin Kursor Pointer (Cursor Pointer Rule)**:
  - Setiap elemen yang dapat diklik (`<button>`, `<a href>`, `<input type="checkbox">`, baris tabel interaktif, trigger accordion/dropdown) WAJIB memiliki kelas Tailwind `cursor-pointer`.
  - Elemen yang dinonaktifkan (`disabled`) WAJIB memiliki kelas `cursor-not-allowed` dan `opacity-50`.
  - Elemen kartu statis, label, teks, atau pembungkus layout DILARANG KERAS memiliki kelas `cursor-pointer`.
- **RULE-P07-UI-04**: **Aksesibilitas Tombol Berbasis Ikon**: Seluruh elemen `<button>` yang hanya memuat elemen SVG / ikon Lucide wajib menyertakan atribut `aria-label` deskriptif (misal: `aria-label="Hapus transaksi"`).

---

## 5. Aturan Pembersihan Fitur Warisan (Legacy BYOK)

- **RULE-P07-BYOK-01**: **Peniadaan Menu Kunci AI Pengguna**. Dilarang menampilkan antarmuka input API Key perorangan di dashboard pengguna manapun. Seluruh interaksi AI dialihkan melalui gateway pool terpusat.
- **RULE-P07-BYOK-02**: **Integritas Arsip Basis Data**. Tabel basis data `gemini_api_keys` DILARANG di-DROP atau dihapus isinya selama fase ini. Kolom `is_archived = true` wajib disematkan pada seluruh baris yang ada untuk keperluan audit kepatuhan.

---

## 6. Aturan Zero-Downtime Deployment & Migrasi Skema

- **RULE-P07-OPS-01**: **Larangan Migrasi Skema Merusak (Breaking Migrations)**. Dilarang melakukan operasi `DROP COLUMN`, `RENAME COLUMN`, atau menambahkan `NOT NULL` tanpa nilai `DEFAULT` dalam satu langkah migrasi.
- **RULE-P07-OPS-02**: **Prosedur Migrasi Skema Dua Fase (Expand & Contract)**:
  - Deploy N: Tambahkan kolom baru (nullable). Kode menulis ke kolom lama dan kolom baru.
  - Deploy N+1: Backfill data selesai. Kode membaca hanya dari kolom baru.
  - Deploy N+2: Hapus kolom lama setelah kode versi lama sepenuhnya dimatikan.
- **RULE-P07-OPS-03**: **Graceful Shutdown Wajib Menghormati Sinyal OS**. Server Go Gin wajib mengimplementasikan `server.Shutdown(ctx)` dengan batas drain 30 detik untuk menuntaskan koneksi klien aktif dan stream SSE tanpa pemutusan paksa.
- **RULE-P07-OPS-04**: **Pemisahan Probe Liveness dan Readiness**:
  - `/health/live`: Hanya memvalidasi proses hidup.
  - `/health/ready`: Memvalidasi konektivitas basis data PostgreSQL, ekstensi pgvector, dan Redis ping.
