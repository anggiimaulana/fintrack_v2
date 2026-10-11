# Implementation Plan — P07 Rollout RBAC, Realtime & Standar

| Field | Isi |
|---|---|
| **Fase** | P07 — Rollout RBAC, Realtime & Standar |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | P06 selesai (Settings & Admin Dashboard; migrasi 0005–0010 selesai) |
| **Master plan** | `../../IMPLEMENTATION-PLAN.md` |
| **Status** | Siap dieksekusi — belum ada kode ditulis |

> **Koreksi path:** kode frontend berada di `frontend/` (bukan `frontend/src/`).
> Path nyata: `frontend/app/`, `frontend/components/`, `frontend/hooks/`, `frontend/lib/`, `frontend/providers/`.
> Backend: `backend/internal/realtime/`, `backend/internal/middleware/`, `backend/internal/cache/`, `backend/cmd/server/`.
> Database: migrasi kanonik opsional `database/migrations/0011_archive_byok_keys.sql`.

---

## 1. Tujuan & Definition of Done

Menerapkan penegakan hak akses terpadu (RBAC + Entitlement + Limit numerik) pada seluruh endpoint backend, menghadirkan infrastruktur Server-Sent Events (SSE) berkinerja tinggi dengan backbone Redis Pub/Sub untuk pembaruan realtime, standardisasi UI dengan TanStack Table v8 universal, pembersihan menyeluruh kode BYOK, perbaikan aksesibilitas (WCAG 2.1 AA) & kursor, serta memperkuat keandalan operasional melalui dual health gate, graceful shutdown 30 detik, dan prosedur migrasi expand/contract tanpa downtime (*zero downtime*).

**Definition of Done Fase:**
- [ ] Semua 18 task (`TASK-P07-001` s.d. `TASK-P07-018`) selesai (status ✅) dengan bukti verifikasi artefak/log yang lengkap.
- [ ] `go vet ./... && go test ./... && go build ./...` di folder `backend/` berhasil tanpa error/warning.
- [ ] `pnpm build && pnpm lint && pnpm test` di folder `frontend/` berhasil tanpa error.
- [ ] Migrasi kanonik `0011_archive_byok_keys.sql` teruji aditif, non-blocking, replayable, dan dump skema telah diperbarui via `database/scripts/dump.sh`.
- [ ] 100% dari 28 endpoint legacy P01 terproteksi middleware entitlement/limit tanpa regresi fungsional.
- [ ] Modul SSE `backend/internal/realtime/` beroperasi dengan heartbeat 15s, autentikasi HTTP-Only cookie, dan resume via `Last-Event-ID`.
- [ ] Cache TanStack Query terinvalidasi secara bedah (*surgical*) saat event domain diterima tanpa pemanggilan root cache liar.
- [ ] Seluruh halaman list legacy (`/transactions`, `/budgets`, `/categories`) telah dimigrasikan ke `<DataTable>` dengan state URL 2-arah.
- [ ] Rute dan kode BYOK bersih dari frontend dan backend; data lama terarsip aman.
- [ ] Health gate `/health/live` (< 1ms) dan `/health/ready` (validasi PostgreSQL + pgvector + Redis) lulus uji rolling restart.
- [ ] Zero packet loss dan zero connection dropped terkonfirmasi saat restart rolling container dengan batas graceful shutdown 30s.

---

## 2. Urutan Eksekusi (Wave)

| Wave | Task ID | Deskripsi Task | Prasyarat |
|---|---|---|---|
| **W1** | TASK-P07-003 | Migrasi In-Memory globalCache ke Redis Berversi | — |
| **W1** | TASK-P07-001 | Middleware Otorisasi Entitlement & Limit di Backend Go | TASK-P07-003 |
| **W1** | TASK-P07-002 | Audit & Proteksi Entitlement 28 Endpoint Legacy P01 | TASK-P07-001 |
| **W2** | TASK-P07-004 | Pengembangan Hub SSE `internal/realtime/` | — |
| **W2** | TASK-P07-005 | Integrasi Redis Pub/Sub Distribusi SSE Lintas Instance | TASK-P07-003, TASK-P07-004 |
| **W2** | TASK-P07-006 | Resiliensi SSE: Cookie Auth, Heartbeat 15s, Resume Last-Event-ID | TASK-P07-004, TASK-P07-005 |
| **W2** | TASK-P07-007 | Standardisasi Skema JSON Event Domain SSE FinTrack v2 | TASK-P07-004 |
| **W3** | TASK-P07-008 | Hook Frontend `useRealtimeStream` & Auto-Reconnect | TASK-P07-006, TASK-P07-007 |
| **W3** | TASK-P07-009 | Surgical Cache Invalidation TanStack Query Frontend | TASK-P07-008, TASK-P07-002 |
| **W3** | TASK-P07-010 | Optimistic Update Mutasi Transaksi & Anggaran Pengguna | TASK-P07-009 |
| **W4** | TASK-P07-011 | Standardisasi Komponen `<DataTable>` TanStack Table v8 | — |
| **W4** | TASK-P07-012 | Refactor Halaman List Data Lama ke `<DataTable>` | TASK-P07-011 |
| **W4** | TASK-P07-013 | Pembersihan Tuntas Menu & Kode BYOK + Migrasi 0011 | — |
| **W4** | TASK-P07-014 | Audit Universal Cursor Pointer & State Interaktivitas | TASK-P07-012 |
| **W4** | TASK-P07-015 | Audit Kepatuhan Aksesibilitas (a11y) WCAG 2.1 Level AA | TASK-P07-014 |
| **W5** | TASK-P07-016 | Implementasi Dual Health Gate (`/health/live` & `/health/ready`) | TASK-P07-003 |
| **W5** | TASK-P07-017 | Konfigurasi Graceful Shutdown 30s & Zero Packet Loss | TASK-P07-016, TASK-P07-006 |
| **W5** | TASK-P07-018 | Standardisasi Prosedur Migrasi Skema Expand/Contract | — |

---

## 3. Detail Task

### Wave 1 — Fondasi RBAC, Entitlement & Cache Berversi

#### TASK-P07-001 — Implementasi Middleware Otorisasi Entitlement & Limit di Backend Go
- **Target:** `backend/internal/middleware/entitlement.go`, `backend/internal/middleware/rbac.go`
- **Langkah:**
  1. Buat middleware `RequireEntitlement(featureKey string)` yang mengekstrak identitas pengguna dari JWT context, memeriksa status aktif fitur pada model entitlement, dan memvalidasi izin terhadap versi cache Redis (`user:entitlement_ver:{uid}`).
  2. Buat middleware `RequireLimit(limitKey string, currentUsageFn func(ctx context.Context, uid string) (int, error))` untuk memvalidasi batas numerik kuota (misal: kuota transaksi bulanan, kuota akun bank).
  3. Kembalikan respons terstandarisasi: HTTP 403 Forbidden bila entitlement boolean false; HTTP 429 Too Many Requests bila batas numerik terlampaui.
  4. Optimasi evaluasi hak akses menggunakan Redis pipelined lookups untuk menjamin latensi < 2ms.
- **Verifikasi (bukti):** Unit test `entitlement_test.go` memverifikasi HTTP 403 saat flag false, HTTP 429 saat batas kuota habis, dan HTTP 200 saat pengguna memiliki hak akses penuh.
- **Catatan zero-downtime & keamanan:** Fail-closed design untuk operasi sensitif; fallback ke query database jika kunci Redis hilang sementara.

#### TASK-P07-002 — Audit & Pemasangan Proteksi Entitlement ke Seluruh Endpoint Legacy P01
- **Target:** `backend/internal/delivery/http/router.go`, `backend/internal/api/handler/*.go`
- **Langkah:**
  1. Identifikasi dan inventarisasi seluruh 28 endpoint peninggalan fase P01 (transaksi, anggaran, kategori, analitik, export, dan chat AI).
  2. Pasang middleware `RequireEntitlement` dan `RequireLimit` pada routing grup Gin di `router.go` sesuai matriks tier langganan (Free vs Pro vs Enterprise).
  3. Terapkan pembatasan limit transaksi gratis (`tx_monthly_limit`) dan kunci rute AI chat (`ai_chat`) ke verifikasi saldo token token_core (P04).
  4. Tambahkan bypass pengujian terisolasi untuk suite integration test yang ada.
- **Verifikasi (bukti):** Test suite integrasi API `router_protection_test.go` memverifikasi 28 endpoint menolak request token tanpa hak akses yang sesuai.
- **Catatan zero-downtime & keamanan:** Perubahan routing bersifat backwards-compatible bagi user bertier Pro yang sudah aktif.

#### TASK-P07-003 — Migrasi In-Memory globalCache Lokal ke Redis Distributed Cache Berversi
- **Target:** `backend/internal/cache/redis.go`, `backend/internal/cache/cache.go`
- **Langkah:**
  1. Audit seluruh penggunaan variabel `globalCache`, `sync.Map`, atau in-memory map di backend Go peninggalan P01.
  2. Gantikan dengan client wrapper Redis terdistribusi menggunakan namespace standar berversi: `ft:{env}:{module}:{entity}:{id}`.
  3. Implementasikan fungsi `InvalidateUserCache(ctx context.Context, userID string) error` yang menaikkan counter versi integer `user:cache_ver:{uid}` dan menghapus kunci kadaluwarsa secara non-blocking.
  4. Pastikan deserialisasi cache tahan terhadap perubahan struct schema JSON.
- **Verifikasi (bukti):** `cache_versioned_test.go` membuktikan mutasi data pada Node 1 langsung menyebabkan cache miss dan fresh fetch data baru pada Node 2 via bump versi.
- **Catatan zero-downtime & keamanan:** Tidak ada state in-memory lokal yang tersisa; mendukung deployment multi-pod tanpa stale reads.

---

### Wave 2 — Infrastruktur Server-Sent Events (SSE) & Redis Pub/Sub

#### TASK-P07-004 — Pengembangan Modul Hub Server-Sent Events (SSE) `internal/realtime/`
- **Target:** `backend/internal/realtime/hub.go`, `backend/internal/realtime/client.go`
- **Langkah:**
  1. Bangun struct `Hub` berbasis concurrency Go (`sync.RWMutex`, channels) untuk mengelola map koneksi klien aktif terindeks oleh `userID`.
  2. Buat struct `Client` yang membungkus koneksi HTTP response stream, channel pengiriman buffer `chan []byte`, dan lifecycle context.
  3. Terapkan header streaming wajib W3C: `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-transform`, `Connection: keep-alive`, `X-Accel-Buffering: no`.
  4. Bersihkan alokasi memori koneksi dan channel secara bersih saat koneksi ditutup oleh klien (`ctx.Done()`).
- **Verifikasi (bukti):** Benchmark dan concurrent test `hub_test.go` membuktikan 1.000 koneksi simultan terdaftar dan dibersihkan tanpa goroutine leak.
- **Catatan zero-downtime & keamanan:** Buffer channel per-klien dibatasi (64 message) untuk mencegah pemborosan RAM bila klien mengalami lag pembacaan.

#### TASK-P07-005 — Integrasi Redis Pub/Sub sebagai Backbone Distribusi SSE Lintas Instance
- **Target:** `backend/internal/realtime/redis_sub.go`, `backend/internal/realtime/publisher.go`
- **Langkah:**
  1. Buat background subscriber yang mendengarkan channel Redis `realtime:events:*` menggunakan `redis.Client.Subscribe`.
  2. Implementasikan `EventPublisher` untuk mempublikasikan event domain dari modul mana pun (billing, gateway, transaksi) ke channel Redis.
  3. Tambahkan loop reconnect otomatis dengan exponential backoff jika koneksi Redis Pub/Sub terputus.
  4. Ketika pesan diterima dari Redis, parse payload dan teruskan ke subscriber lokal di `Hub` berdasarkan target `userID` atau broadcast.
- **Verifikasi (bukti):** Test multi-instance `pubsub_integration_test.go` mengonfirmasi event yang dipublikasikan pada mock instance A diterima oleh klien SSE di mock instance B dalam latensi < 50ms.
- **Catatan zero-downtime & keamanan:** Payload Redis diserialisasi dalam format JSON ringkas bertanda waktu dan ID unik.

#### TASK-P07-006 — Penguatan Resiliensi SSE: Cookie Auth, 15s Heartbeat, dan Resume Last-Event-ID
- **Target:** `backend/internal/realtime/handler.go`, `backend/internal/realtime/buffer.go`
- **Langkah:**
  1. Implementasikan handler `ServeHTTP` untuk rute `GET /api/v1/realtime/stream` yang memvalidasi otentikasi melalui HTTP-Only cookie `fintrack_token` (tolak dengan HTTP 401 jika absen).
  2. Buat background ticker goroutine per koneksi yang mengirimkan komentar ping `: ping\n\n` setiap 15 detik agar koneksi tidak diputus oleh proxy Nginx/load balancer.
  3. Sediakan penyimpanan ring-buffer event terbaru per pengguna di Redis (data type ZSET / Stream dengan TTL 10 menit).
  4. Periksa header `Last-Event-ID` saat request masuk; kirim ulang seluruh event yang terlewat secara sekuensial sebelum masuk ke live stream.
- **Verifikasi (bukti):** `handler_resilience_test.go` memverifikasi replay event setelah simulasi putus koneksi selama 60 detik serta validasi ping frame tiap 15 detik.
- **Catatan zero-downtime & keamanan:** Dilarang menerima token lewat query parameter URL untuk mencegah credential leaking di log akses proxy.

#### TASK-P07-007 — Standardisasi Skema JSON Event Domain SSE FinTrack v2
- **Target:** `backend/internal/realtime/events.go`, `backend/pkg/contracts/events.go`
- **Langkah:**
  1. Definisikan tipe data Go untuk envelope event domain: `ID`, `Event` (nama tipe), `Timestamp` (RFC3339), dan `Payload` (`json.RawMessage`).
  2. Buat struct payload terstandarisasi untuk event kunci:
     - `payment_received` (invoice_id, amount, status)
     - `plan_updated` (tier, valid_until, quota_limits)
     - `ai_endpoint_down` (endpoint_id, provider, reason)
     - `manual_approval_needed` (tx_id, requester, amount)
     - `quota_reset` (cycle_start, new_balance)
  3. Sediakan fungsi factory helper untuk marshaling event dengan validasi skema ketat.
- **Verifikasi (bukti):** Unit test `events_schema_test.go` memvalidasi encoding dan decoding JSON dari semua tipe event domain tanpa error unmarshal.
- **Catatan zero-downtime & keamanan:** Skema bersifat aditif; field baru opsional tidak memecahkan parser versi klien lama.

---

### Wave 3 — Integrasi Frontend: Realtime Stream & Surgical Cache Invalidation

#### TASK-P07-008 — Pembangunan Frontend Hook `useRealtimeStream` & Auto-Reconnect
- **Target:** `frontend/hooks/use-realtime-stream.ts`, `frontend/providers/realtime-provider.tsx`
- **Langkah:**
  1. Bangun context provider `RealtimeProvider` yang menginisialisasi streaming saat pengguna terautentikasi dan membersihkannya saat logout.
  2. Implementasikan custom hook `useRealtimeStream` yang mengelola native `EventSource` dengan opsi `withCredentials: true`.
  3. Tambahkan algoritma reconnection exponential backoff dengan jitter acak (1s, 2s, 4s, hingga maksimal 16s) saat koneksi terputus.
  4. Simpan event ID terakhir di state lokal untuk diteruskan otomatis via `Last-Event-ID` saat peramban mencoba menyambung ulang.
- **Verifikasi (bukti):** Vitest `use-realtime-stream.test.tsx` menguji siklus hidup koneksi, penanganan error jaringan, dan pemicuan reconnect dengan backoff yang tepat.
- **Catatan zero-downtime & keamanan:** Tidak ada memory leak pada unmount komponen; listener event dihapus secara rapi.

#### TASK-P07-009 — Implementasi Surgical Cache Invalidation TanStack Query pada Frontend
- **Target:** `frontend/lib/query-keys.ts`, `frontend/lib/realtime-dispatcher.ts`
- **Langkah:**
  1. Standardisasi hierarki query key di `query-keys.ts` (misal: `transactions.list`, `transactions.detail(id)`, `billing.invoices`, `subscription.current`).
  2. Bangun dispatcher `dispatchRealtimeEvent(event, queryClient)` yang memetakan event domain ke operasi invalidasi TanStack Query terarah (*surgical*).
  3. Pastikan event `payment_received` hanya menginvalidasi `['billing']` dan `['subscription']`, bukan seluruh cache query.
  4. Larang pemanggilan wildcard `queryClient.invalidateQueries()` tanpa filter key yang presisi.
- **Verifikasi (bukti):** Vitest `realtime-dispatcher.test.ts` memverifikasi pemanggilan `invalidateQueries` hanya menyentuh key spesifik yang terasosiasi dengan tipe event yang diterima.
- **Catatan zero-downtime & keamanan:** Mencegah refetch cascade storm yang dapat membebani backend saat terjadi broadcast event serentak.

#### TASK-P07-010 — Pola Optimistic Update pada Mutasi Transaksi & Anggaran Pengguna
- **Target:** `frontend/hooks/use-transactions.ts`, `frontend/hooks/use-budgets.ts`
- **Langkah:**
  1. Perbarui mutation hook transaksi (`useCreateTransaction`, `useUpdateTransaction`, `useDeleteTransaction`) dengan siklus `onMutate`, `onError`, dan `onSettled`.
  2. Di `onMutate`: batalkan query aktif (`cancelQueries`), ambil snapshot data cache sebelumnya, dan suntikkan data baru secara instan ke cache UI.
  3. Di `onError`: rollback data cache ke snapshot sebelumnya dan tampilkan toast notifikasi error kepada pengguna.
  4. Di `onSettled`: lakukan invalidasi query terkait untuk memastikan sinkronisasi akhir dengan backend.
- **Verifikasi (bukti):** Vitest `use-transactions-optimistic.test.tsx` membuktikan UI langsung terupdate sebelum response server kembali, dan kembali ke kondisi awal saat server mengembalikan simulasi HTTP 500/409.
- **Catatan zero-downtime & keamanan:** Optimistic updates memberikan latensi persepsi 0ms bagi interaksi pengguna harian.

---

### Wave 4 — Standardisasi UI, Pembersihan BYOK & Audit Aksesibilitas

#### TASK-P07-011 — Standardisasi Komponen `<DataTable>` TanStack Table v8 Universal
- **Target:** `frontend/components/data-table/data-table.tsx`, `frontend/components/data-table/data-table-pagination.tsx`, `frontend/components/data-table/data-table-toolbar.tsx`
- **Langkah:**
  1. Bangun wrapper komponen generic `<DataTable<TData, TValue>>` menggunakan `@tanstack/react-table` v8.
  2. Implementasikan sinkronisasi state 2-arah ke URL search parameters via hook `useSearchParams` Next.js (`page`, `per_page`, `sort`, `order`, `search`).
  3. Sediakan komponen toolbar terintegrasi dengan input pencarian ber-debounce 300ms, filter faceted status/kategori, dan pengatur visibilitas kolom.
  4. Tambahkan komponen paginasi 20 baris per halaman, tombol navigasi halaman pertama/terakhir, dan tampilan jumlah total baris.
- **Verifikasi (bukti):** Vitest `data-table.test.tsx` menguji interaksi rendering data, perubahan sorting, filter pencarian debounce, dan update URL query parameters.
- **Catatan zero-downtime & keamanan:** State URL memungkinkan tautan halaman tabel dapat di-bookmark dan dibagikan secara akurat.

#### TASK-P07-012 — Refactor Seluruh Halaman List Data Lama Menggunakan `<DataTable>`
- **Target:** `frontend/app/(dashboard)/transactions/page.tsx`, `frontend/app/(dashboard)/budgets/page.tsx`, `frontend/app/(dashboard)/categories/page.tsx`
- **Langkah:**
  1. Ganti implementasi tabel kustom atau elemen list statis lama pada halaman Transaksi (`/transactions`) dengan `<DataTable>` terstandarisasi.
  2. Ganti tampilan tabel pada halaman Anggaran (`/budgets`) dan Kategori (`/categories`) dengan kolom definition TanStack Table.
  3. Hubungkan pagination dan sorting server-side dengan TanStack Query hooks yang relevan.
  4. Pertahankan empty state, skeleton placeholder saat loading, dan penanganan visual error.
- **Verifikasi (bukti):** Uji fungsional visual dan tes komponen Next.js membuktikan ketiga halaman me-render data tabel secara seragam dengan URL filtering yang bekerja.
- **Catatan zero-downtime & keamanan:** Tidak ada perubahan logika bisnis transaksi, murni peremajaan antarmuka.

#### TASK-P07-013 — Pembersihan Tuntas Menu & Kode Bring Your Own Key (BYOK)
- **Target:** `frontend/components/sidebar.tsx`, `frontend/app/(dashboard)/settings/ai-keys/`, `backend/internal/delivery/http/router.go`, `database/migrations/0011_archive_byok_keys.sql`
- **Langkah:**
  1. Buat skrip migrasi kanonik non-blocking `database/migrations/0011_archive_byok_keys.sql` yang menambahkan kolom `is_archived BOOLEAN DEFAULT TRUE` pada tabel `gemini_api_keys` (jika ada) dan menonaktifkan pemakaian kuncinya.
  2. Hapus tautan navigasi menu "AI API Keys" pada sidebar dan pengaturan akun frontend.
  3. Hapus folder halaman `frontend/app/(dashboard)/settings/ai-keys/` dan buat konfigurasi redirect permanen HTTP 301 ke `/settings/subscription` pada `next.config.ts`.
  4. Hapus rute endpoint backend registrasi kunci user di `router.go` dan handler terkait.
- **Verifikasi (bukti):** Eksekusi migrasi 0011 sukses; pencarian string kode membuktikan 0 kemunculan referensi form BYOK di frontend; rute lama mengembalikan redirect 301.
- **Catatan zero-downtime & keamanan:** Pengarsipan database tidak menghapus baris secara fisik (*non-destructive*), menjamin integritas audit log masa lalu.

#### TASK-P07-014 — Audit Universal Cursor Pointer & State Interaktivitas UI
- **Target:** Seluruh berkas `.tsx` pada `frontend/components/` dan `frontend/app/`
- **Langkah:**
  1. Lakukan audit otomatis dan manual terhadap seluruh elemen interaktif pada berkas antarmuka UI.
  2. Pasang kelas Tailwind `cursor-pointer` pada seluruh komponen tombol (`<Button>`), link navigasi (`<Link>`), baris tabel yang dapat diklik, tab switcher, dan checkbox.
  3. Pasang kelas `cursor-not-allowed` pada elemen tombol/input yang berada dalam kondisi `disabled`.
  4. Pastikan kartu statis atau container teks biasa tetap berstatus default (`cursor-default` / `cursor-auto`).
- **Verifikasi (bukti):** Laporan checklist audit cursor di `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/cursor-pointer-audit.md` diverifikasi lengkap di 14 halaman dashboard.
- **Catatan zero-downtime & keamanan:** Meningkatkan kejelasan affordance UI bagi pengguna desktop.

#### TASK-P07-015 — Audit Kepatuhan Aksesibilitas (a11y) WCAG 2.1 Level AA
- **Target:** Komponen modal dialog, dropdown menu, tombol icon, dan input form pada `frontend/components/`
- **Langkah:**
  1. Pasang atribut `aria-label` deskriptif pada seluruh tombol yang hanya menampilkan ikon tanpa label teks (contoh: tombol delete, icon edit, close dialog).
  2. Terapkan mekanisme focus trap pada modal dialog Radix UI / Shadcn UI dan pastikan tombol `Escape` menutup dialog secara konsisten.
  3. Validasi urutan navigasi keyboard (`Tab`, `Shift+Tab`, `Enter`, `Space`) pada tabel dan form input.
  4. Audit rasio kontras warna teks dan latar belakang (minimal 4.5:1 untuk teks normal) pada tema terang dan gelap.
- **Verifikasi (bukti):** Audit audit automated tool / Lighthouse Accessibility menghasilkan skor >= 95 pada seluruh rute utama dashboard.
- **Catatan zero-downtime & keamanan:** Kepatuhan WCAG 2.1 AA menjamin aplikasi ramah screen reader dan keyboard-only navigation.

---

### Wave 5 — Reliability, Zero-Downtime Deploy & Standardisasi Migrasi

#### TASK-P07-016 — Implementasi Dual Health Gate Endpoint (`/health/live` & `/health/ready`)
- **Target:** `backend/internal/api/health.go`, `backend/cmd/server/main.go`
- **Langkah:**
  1. Bangun handler endpoint `GET /health/live` yang merespons status proses Go secara langsung dengan HTTP 200 OK (< 1ms).
  2. Bangun handler endpoint `GET /health/ready` yang melakukan pengecekan dependensi secara konkuren:
     - `db.PingContext()` dan query verifikasi pgvector (`SELECT 1 FROM pg_extension WHERE extname = 'vector'`).
     - `redis.Ping()`.
  3. Jika semua dependensi sehat, kembalikan HTTP 200 OK dengan payload waktu latensi (JSON). Jika salah satu gagal, kembalikan HTTP 503 Service Unavailable.
  4. Daftarkan kedua rute di luar middleware otorisasi agar dapat diakses probe Docker/Kubernetes.
- **Verifikasi (bukti):** Unit test `health_test.go` memvalidasi `/health/live` selalu 200, dan `/health/ready` mengembalikan 503 saat mock koneksi Redis atau PostgreSQL dimatikan.
- **Catatan zero-downtime & keamanan:** Mencegah orchestrator mengarahkan lalu lintas pengguna ke pod backend yang belum selesai inisialisasi koneksi pool.

#### TASK-P07-017 — Konfigurasi Graceful Shutdown & Zero Packet Loss Rolling Deploy
- **Target:** `backend/cmd/server/main.go`, `docker-compose.yml`, `deploy/k8s/deployment.yaml`
- **Langkah:**
  1. Tangkap sinyal OS `SIGTERM` dan `SIGINT` menggunakan channel `os.Notify` pada `main.go`.
  2. Saat sinyal diterima, buat timeout context selama 30 detik untuk memanggil `httpServer.Shutdown(ctx)`.
  3. Hentikan penerimaan request HTTP baru, tutup listener SSE secara graceful dengan notifikasi penutupan, dan tunggu goroutine transaksi yang sedang berjalan selesai.
  4. Konfigurasikan parameter container `stop_grace_period: 35s` pada `docker-compose.yml` atau `terminationGracePeriodSeconds: 35` di manifest deployment.
- **Verifikasi (bukti):** Pengujian skrip `test_rolling_restart.sh` di bawah beban sintetis 200 RPS membuktikan 0 request mengalami HTTP 502/504 selama proses restart container.
- **Catatan zero-downtime & keamanan:** Menghindari pemutusan paksa koneksi database atau transaksi finansial yang sedang dieksekusi.

#### TASK-P07-018 — Standardisasi Prosedur Migrasi Skema Database Expand/Contract
- **Target:** `database/migrations/README.md`, `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/zero-downtime-deploy.md`
- **Langkah:**
  1. Susun panduan resmi migrasi skema database berpola Expand/Contract untuk seluruh fase FinTrack v2.
  2. Tetapkan aturan baku: dilarang melakukan `DROP COLUMN`, `RENAME COLUMN`, atau menambah constraint `NOT NULL` tanpa nilai default secara langsung.
  3. Wajibkan pembuatan indeks baru menggunakan klausa non-blocking (`CREATE INDEX CONCURRENTLY`).
  4. Sediakan template migrasi dua langkah: Fase Expand (tambah kolom nullable/view baru) diikuti Fase Contract (hapus kolom usang setelah kode baru live).
- **Verifikasi (bukti):** Dokumen panduan `database/migrations/README.md` dan `zero-downtime-deploy.md` disetujui, dan migrasi 0011 divalidasi mematuhi aturan ini secara presisi.
- **Catatan zero-downtime & keamanan:** Skema database selalu kompatibel ke belakang (*backward-compatible*) dengan versi aplikasi n-1.

---

## 4. Matriks Bukti (Evidence)

| Task ID | Artefak Bukti yang Dihasilkan |
|---|---|
| **TASK-P07-001** | Hasil unit test `backend/internal/middleware/entitlement_test.go` (lulus HTTP 403 & 429) |
| **TASK-P07-002** | Log pengujian integrasi `router_protection_test.go` melingkupi 28 endpoint legacy |
| **TASK-P07-003** | Log test `cache_versioned_test.go` memverifikasi invalidasi cache via bump counter versi |
| **TASK-P07-004** | Hasil benchmark concurrency `hub_test.go` dengan 1.000 subscriber SSE tanpa leak |
| **TASK-P07-005** | Log integrasi `pubsub_integration_test.go` membuktikan distribusi event lintas node < 50ms |
| **TASK-P07-006** | Test output `handler_resilience_test.go` membuktikan replay `Last-Event-ID` & ping frame 15s |
| **TASK-P07-007** | Unit test `events_schema_test.go` memvalidasi JSON schema 5 domain events |
| **TASK-P07-008** | Laporan Vitest `use-realtime-stream.test.tsx` (lifecycle & backoff reconnect) |
| **TASK-P07-009** | Vitest output `realtime-dispatcher.test.ts` (surgical cache invalidation) |
| **TASK-P07-010** | Laporan test `use-transactions-optimistic.test.tsx` (optimistic render + rollback) |
| **TASK-P07-011** | Vitest coverage report `data-table.test.tsx` (pagination, debounce, URL sync) |
| **TASK-P07-012** | Screenshot & Vitest render check untuk `/transactions`, `/budgets`, dan `/categories` |
| **TASK-P07-013** | Output migrasi `0011_archive_byok_keys.sql`, dump diff, dan tes HTTP 301 redirect |
| **TASK-P07-014** | Checklist audit visual pointer yang terisi di `teknis/cursor-pointer-audit.md` |
| **TASK-P07-015** | Laporan skor Lighthouse Accessibility >= 95 dan verifikasi focus trap |
| **TASK-P07-016** | Output curl JSON `/health/live` (200) dan `/health/ready` (200 OK & 503 saat simulasi down) |
| **TASK-P07-017** | Laporan eksekusi beban rolling restart 200 RPS dengan 0 failed connection |
| **TASK-P07-018** | Panduan operasional tertulis `database/migrations/README.md` bertema Expand/Contract |

---

## 5. Perintah Verifikasi

Perintah berikut dijalankan untuk memvalidasi kesiapan kode sebelum dan sesudah rilis:

```bash
# 1. Verifikasi Backend Go (Lint, Test, Compile)
cd D:/project/finance/fintrack_v2/backend
go vet ./...
go test -v -race ./internal/middleware/... ./internal/realtime/... ./internal/cache/... ./internal/api/...
go build -v ./cmd/server

# 2. Verifikasi Frontend Next.js 16 (Typecheck, Lint, Unit Test, Build)
cd D:/project/finance/fintrack_v2/frontend
pnpm lint
pnpm test
pnpm build

# 3. Verifikasi Migrasi Database (Replayable & Dump)
cd D:/project/finance/fintrack_v2
APP_DB_PASSWORD=postgres database/scripts/migrate.sh
database/scripts/dump.sh
git diff --stat database/dump/

# 4. Verifikasi Health Gates & Graceful Shutdown
curl -s -i http://localhost:8080/health/live
curl -s -i http://localhost:8080/health/ready
```

---

## 6. Prosedur Rollback

Jika ditemukan regresi kritis selama rollout fase P07, lakukan prosedur rollback berikut:

1. **Penegakan Entitlement / Limit (TASK-P07-001 & 002):**
   - Nonaktifkan enforcement ketat dengan mengaktifkan feature flag `ff_permissive_entitlements=true` di Redis/env tanpa perlu redeploy container.
   - Jika terjadi kegagalan Redis versi cache, fallback otomatis ke database query langsung dengan mode read-through.
2. **Realtime SSE & Redis Pub/Sub (TASK-P07-004 s.d. 006):**
   - Bila koneksi SSE mengalami lonjakan beban atau instabilitas proxy, set feature flag frontend `ff_enable_sse=false`.
   - Frontend akan secara otomatis kembali ke mekanisme interval polling berkala standar (fallback mode) tanpa memutus fungsionalitas UI.
3. **Komponen UI DataTable (TASK-P07-011 & 012):**
   - Komponen list lama disimpan dalam cabang fallback sementara atau diatur melalui feature flag komponen `ff_legacy_tables=true` bila ditemukan isu kompabilitas browser.
4. **Migrasi Skema Database 0011 (TASK-P07-013):**
   - Karena migrasi bersifat aditif (`is_archived BOOLEAN DEFAULT TRUE`), kolom ini tidak memblokir query aplikasi lama.
   - Bila dibutuhkan pembalikan status, jalankan script perbaikan `UPDATE gemini_api_keys SET is_archived = FALSE;`.
5. **Rollback Container:**
   - Gunakan image tag container sebelumnya (`fintrack-backend:p06` dan `fintrack-frontend:p06`) pada docker-compose atau cluster orchestrator.

---

## 7. Handoff ke Fase P08 (Hardening, E2E & Production Readiness)

Setelah seluruh task P07 tuntas, fondasi berikut siap diserahkan ke **P08**:
- **Proteksi Akses Terpusat:** Seluruh endpoint P01–P07 telah memiliki pengawalan RBAC dan entitlement yang siap diuji secara penetratif (security pentest) di P08.
- **Kanal Realtime Stabil:** Infrastruktur SSE dan Redis Pub/Sub siap menangani push notifikasi transaksi real-time, status pembayaran Midtrans, dan pembaruan AI token balance di lingkungan produksi.
- **Standar UI & a11y:** Antarmuka aplikasi telah seragam menggunakan `<DataTable>`, memiliki affordance kursor yang presisi, dan lolos verifikasi aksesibilitas WCAG 2.1 AA.
- **Kesiapan Zero-Downtime:** Dual health gate (`/live` dan `/ready`) serta mekanisme graceful shutdown 30s telah aktif, memungkinkan pengujian E2E load test, simulasi disaster recovery, dan rolling deployment production readiness di Fase P08.

---

## 8. Referensi

- Dokumen Spesifikasi Fase: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/README.md`
- Daftar Rincian Task: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/planning/tasks.md`
- Analisis Timeline & Dependensi: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/planning/timeline.md`
- Dokumen Teknis Arsitektur SSE: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/sse-strategy.md`
- Dokumen Teknis Cache Invalidation: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/cache-invalidation.md`
- Dokumen Teknis Rollout RBAC: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/rbac-rollout.md`
- Dokumen Teknis Zero-Downtime Deploy: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/zero-downtime-deploy.md`
- Dokumen Teknis Audit Kursor: `second_brain/2_Projects/P07_Rollout-RBAC-Realtime/teknis/cursor-pointer-audit.md`
- Master Plan Proyek: `second_brain/2_Projects/IMPLEMENTATION-PLAN.md`
