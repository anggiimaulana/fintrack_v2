# Implementation Plan — P03 AI Gateway Terpusat

| Field | Isi |
|---|---|
| **Fase** | P03 — AI Gateway Terpusat |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | P01 selesai (monorepo jalan), P02 selesai (Redis pool, audit log, feature flag, metrics) |
| **Master plan** | `../../IMPLEMENTATION-PLAN.md` |
| **Status** | Siap dieksekusi — belum ada kode ditulis |

> **Catatan Alokasi Migrasi:** Sesuai urutan kanonik sistem, fase ini menggunakan migrasi **`0007_ai_gateway.sql`** (bukan 0006).  
> **Koreksi Path Frontend:** Seluruh kode frontend berada di `frontend/` (bukan `frontend/src/`). Path nyata: `frontend/app/`, `frontend/components/`, `frontend/hooks/`, `frontend/lib/`.  
> **Prinsip Modul:** Penambahan modul baru terisolasi di `backend/internal/aigateway/` dan adapter tipis di `backend/internal/usecase/ai_usecase.go`. Dilarang rewrite masif domain finansial.

---

## 1. Tujuan & Definisi Selesai

Membangun AI Gateway terpusat multi-endpoint (9router default `http://localhost:20128/v1`, failover, circuit breaker, kill switch, health checker) dan metering token, mengunci model embedding pada dimensi 768d, menghapus pola BYOK (Bring Your Own Key) lama secara elegan, serta membersihkan antarmuka frontend tanpa downtime.

**Definition of Done Fase:**
- [ ] Semua 14 task (TASK-P03-001 s.d. TASK-P03-014) ✅ dengan artefak bukti lengkap.
- [ ] Migrasi kanonik `database/migrations/0007_ai_gateway.sql` berhasil dieksekusi idempoten & file dump `database/dump/` diperbarui.
- [ ] Modul `backend/internal/aigateway/` terimplementasi bersih tanpa dependensi sirkular ke usecase.
- [ ] Circuit breaker mematuhi state machine: 5x error berturut-turut -> `Open` (cooldown 30s) -> `Half-Open` (1 probe) -> `Closed` / `Open`.
- [ ] Failover antar rute transparan saat upstream primer gagal dalam < 1 detik.
- [ ] Dimensi embedding runtime tervalidasi strictly 768 elemen (`vector(768)` PostgreSQL tetap konsisten).
- [ ] Metering token mengekstrak `TokenUsage` akurat (usage provider atau heuristik fallback).
- [ ] Handler BYOK lama (`/api/v1/gemini-keys`) merespons `HTTP 410 Gone`, data tabel `gemini_api_keys` terarsip aman (`is_archived = true`).
- [ ] Antarmuka frontend bebas dari form input API key pribadi dan bebas broken link.
- [ ] `go vet && go test ./... && go build ./...` hijau.
- [ ] `pnpm build && pnpm lint && pnpm test` hijau.

---

## 2. Urutan Eksekusi (Wave)

| Wave | Task | Deskripsi Ringkas | Prasyarat |
|---|---|---|---|
| **W1** | TASK-P03-001, TASK-P03-002, TASK-P03-003 | Skema database migrasi 0007, domain interface gateway, repository & route cache | P02 selesai |
| **W2** | TASK-P03-004, TASK-P03-005, TASK-P03-006, TASK-P03-007 | OpenAI HTTP client engine (SSE), circuit breaker, failover router, health checker | W1 (002, 003) |
| **W3** | TASK-P03-008, TASK-P03-009, TASK-P03-010 | Metering token extractor, adapter tipis AI usecase, embedder 768d terkunci | W2 (004, 006) |
| **W4** | TASK-P03-011, TASK-P03-012 | Penghentian & pengarsipan BYOK backend, pembersihan frontend UX BYOK | W3 (009) |
| **W5** | TASK-P03-013, TASK-P03-014 | Integrasi test failover/breaker, benchmark SSE & zero-downtime rollout | W4 (semua task) |

---

## 3. Detail Task

### Wave 1 — Fondasi Skema, Domain & Repositori

#### TASK-P03-001 — Skema Database & Migrasi `0007_ai_gateway.sql`
- **Target File:** `database/migrations/0007_ai_gateway.sql`
- **Langkah:**
  1. Buat tabel `ai_endpoints` dengan kolom `id` (UUID PK), `name` (VARCHAR UNIQUE), `base_url` (TEXT), `api_key_encrypted` (BYTEA), `provider` (VARCHAR), `priority` (INT DEFAULT 1), `status` (VARCHAR DEFAULT 'active'), `health` (VARCHAR DEFAULT 'healthy'), `last_check_at` (TIMESTAMPTZ), `modalities` (TEXT[]), `weight` (INT DEFAULT 100), `created_at`, `updated_at`.
  2. Tambahkan constraints check `chk_endpoint_status` ('active', 'degraded', 'inactive', 'suspended') dan `chk_endpoint_health` ('healthy', 'unhealthy', 'unknown').
  3. Buat indeks `idx_ai_endpoints_status_priority` pada `(status, priority)`.
  4. Buat tabel `ai_routes` dengan kolom `id` (UUID PK), `use_case` (VARCHAR), `endpoint_id` (UUID FK REFERENCES `ai_endpoints` ON DELETE CASCADE), `model` (VARCHAR), `combo` (VARCHAR DEFAULT 'default'), `priority` (INT DEFAULT 1), `is_active` (BOOLEAN DEFAULT TRUE), `timeout_ms` (INT DEFAULT 30000), `max_retries` (INT DEFAULT 2), `created_at`, `updated_at`.
  5. Tambahkan constraint check `chk_use_case` ('chat', 'vision', 'voice', 'embedding', 'reasoning') dan indeks `idx_ai_routes_lookup` pada `(use_case, is_active, priority)`.
  6. Tambahkan kolom audit pengarsipan pada tabel lama: `ALTER TABLE gemini_api_keys ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT TRUE;` dan `ALTER TABLE gemini_api_keys ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ DEFAULT NOW();`.
  7. Masukkan seed endpoint default 9router lokal (`http://localhost:20128/v1`, provider `9router`, priority 1) dan rute default untuk `chat` (`gpt-4o-mini`), `vision` (`gpt-4o-mini`), `voice` (`whisper-1`), `embedding` (`text-embedding-004`), `reasoning` (`deepseek-r1`).
- **Verifikasi (Bukti):** Eksekusi migrasi UP dan DOWN tanpa error; `\d ai_endpoints`, `\d ai_routes`, dan `\d gemini_api_keys` menampilkan struktur tabel & constraint yang sesuai di PostgreSQL.
- **Catatan Zero-Downtime / Keamanan:** Operasi skema bersifat aditif (`ADD COLUMN IF NOT EXISTS`, `CREATE TABLE IF NOT EXISTS`), tidak memblokir query aktif aplikasi. Kunci API tersimpan dalam kolom biner terenkripsi.

#### TASK-P03-002 — Domain Modeling & Interface Gateway
- **Target File:** `backend/internal/domain/aigateway.go`
- **Langkah:**
  1. Definisikan enum tipe use case: `UseCaseChat`, `UseCaseVision`, `UseCaseVoice`, `UseCaseEmbedding`, `UseCaseReasoning`.
  2. Definisikan struct entitas `AIEndpoint` dan `AIRoute` lengkap dengan tag database/json.
  3. Definisikan struct DTO request & response: `AIChatRequest`, `AIChatMessage`, `AIChatStreamChunk`, `AIEmbeddingRequest`, `AIEmbeddingResponse`, `AIVisionRequest`, `AIAudioTranscriptionRequest`, serta `TokenUsage`.
  4. Definisikan antarmuka repositori: `AIEndpointRepository` dan `AIRouteRepository`.
  5. Definisikan antarmuka layanan gateway `AIGatewayService` (`ChatStream`, `ChatSync`, `CreateEmbedding`, `TranscribeAudio`, `AnalyzeVision`).
  6. Definisikan interface `CircuitBreaker` (`Allow(endpointID string) bool`, `RecordSuccess(endpointID string)`, `RecordFailure(endpointID string)`) dan `HealthChecker`.
- **Verifikasi (Bukti):** File domain mengompilasi bersih tanpa import cycles (`go build ./backend/internal/domain/...`).
- **Catatan Zero-Downtime / Keamanan:** Interface domain murni tanpa dependensi eksternal pihak ketiga.

#### TASK-P03-003 — Repository SQL AI Gateway & Route Cache
- **Target File:** `backend/internal/infrastructure/repository/ai_gateway_repository.go`
- **Langkah:**
  1. Implementasikan query database `GetActiveRoutesByUseCase(ctx context.Context, useCase string) ([]*domain.AIRoute, error)` dengan sorting `priority ASC`.
  2. Implementasikan metode enkripsi & dekripsi `api_key_encrypted` menggunakan fungsi cipher AES-256-GCM berbasis kunci master sistem (`crypto.Key`).
  3. Implementasikan update status kesehatan `UpdateEndpointHealth(ctx context.Context, id uuid.UUID, health string, lastCheck time.Time) error`.
  4. Tambahkan in-memory thread-safe route cache (`sync.RWMutex`) dengan TTL 60 detik dan metode pembersihan manual `InvalidateCache()`.
- **Verifikasi (Bukti):** Unit test repository mengonfirmasi pengambilan rute terurut, enkripsi bolak-balik kredensial berhasil, dan cache serving rute < 1ms.
- **Catatan Zero-Downtime / Keamanan:** Plaintext API key tidak pernah dicatat ke log aplikasi; key hanya didekripsi saat hendak digunakan oleh worker runtime.

---

### Wave 2 — Engine Klien, Circuit Breaker, Router & Health Checker

#### TASK-P03-004 — OpenAI-Compatible HTTP Client Engine
- **Target File:** `backend/internal/aigateway/client/openai_client.go`, `backend/internal/aigateway/client/stream_parser.go`
- **Langkah:**
  1. Bangun HTTP client wrapper yang mematuhi format OpenAI REST API (`/chat/completions`, `/embeddings`, `/audio/transcriptions`).
  2. Konfigurasikan pooling koneksi HTTP transport (`MaxIdleConns: 100`, `IdleConnTimeout: 90s`, `KeepAlive: 30s`) dengan batas timeout per-request dinamis dari konfigurasi rute (`timeout_ms`).
  3. Implementasikan streaming reader untuk SSE (`text/event-stream`), memisahkan prefix baris `data: ` dan mengenali penanda akhir `[DONE]`.
  4. Ekstrak data usage dari SSE chunk saat upstream menyertakan `"stream_options": {"include_usage": true}`.
- **Verifikasi (Bukti):** Test stream parser dengan mock server SSE mengembalikan chunk token teks secara bertahap dan objek `TokenUsage` final.
- **Catatan Zero-Downtime / Keamanan:** Resource HTTP response body selalu ditutup menggunakan `defer resp.Body.Close()` guna menghindari socket leak.

#### TASK-P03-005 — State Machine Circuit Breaker
- **Target File:** `backend/internal/aigateway/breaker/circuit_breaker.go`
- **Langkah:**
  1. Definisikan tipe status: `StateClosed`, `StateOpen`, `StateHalfOpen`.
  2. Implementasikan struct per-endpoint state dengan proteksi `sync.Mutex`: failure count, consecutive failures, last failure timestamp, cooldown duration (30 detik), probe in-flight counter.
  3. Implementasikan fungsi `Allow(endpointID string) bool`:
     - `StateClosed`: izinkan request.
     - `StateOpen`: tolak seketika jika waktu cooldown belum habis (< 30s); alihkan ke `StateHalfOpen` jika cooldown terlampaui.
     - `StateHalfOpen`: izinkan tepat 1 probe concurrency. Tolak request lain selama probe berlangsung.
  4. Implementasikan `RecordSuccess`: reset consecutive failure ke 0, ubah state kembali ke `StateClosed`.
  5. Implementasikan `RecordFailure`:
     - Hitung kegagalan pada HTTP 5xx, gateway timeout, connection refused, 429 berulang.
     - Abaikan HTTP 400 dan 404 (kesalahan input/rute).
     - Jika consecutive failures >= 5 pada `StateClosed`, alihkan langsung ke `StateOpen`.
     - Jika terjadi kegagalan pada `StateHalfOpen`, langsung kembalikan ke `StateOpen` dengan cooldown baru 30 detik.
- **Verifikasi (Bukti):** Unit test memverifikasi transisi state secara presisi: 5x error berturut-turut memutus sirkuit, cooldown 30s menguji 1 probe, dan pemulihan sukses menutup sirkuit kembali.
- **Catatan Zero-Downtime / Keamanan:** Thread-safe tanpa blocking operasi I/O di dalam critical section mutex.

#### TASK-P03-006 — Router Multi-Endpoint & Failover Execution Engine
- **Target File:** `backend/internal/aigateway/router/router.go`
- **Langkah:**
  1. Ambil daftar rute aktif terurut berdasarkan `priority ASC` untuk `use_case` terkait dari repository/cache.
  2. Iterasi setiap rute dalam rantai fallback:
     - Periksa status administratif endpoint (`status == 'active'`). Jika tidak aktif (kill switch aktif), lewati rute.
     - Periksa status circuit breaker endpoint via `breaker.Allow(endpointID)`. Jika sirkuit Open, lewati rute.
     - Eksekusi request melalui client engine.
     - Jika request berhasil: laporkan `breaker.RecordSuccess(endpointID)`, kembalikan hasil.
     - Jika request gagal (error jaringan/5xx/timeout): laporkan `breaker.RecordFailure(endpointID)`, catat log `WARN` (endpoint asal gagal, fallback ke endpoint berikutnya), dan iterasi ke rute berikutnya.
  3. Jika seluruh rute dalam pool gagal, kembalikan sentinel error `ErrAllEndpointsExhausted`.
- **Verifikasi (Bukti):** Test simulasi router: mock endpoint prioritas 1 dibuat mengembalikan HTTP 500, router secara otomatis mengalihkan request ke mock endpoint prioritas 2 dalam rentang < 100ms.
- **Catatan Zero-Downtime / Keamanan:** Failover berjalan transparan dari sisi pemanggil usecase bisnis.

#### TASK-P03-007 — Active & Passive Health Checker Background Worker
- **Target File:** `backend/internal/aigateway/health/health_checker.go`
- **Langkah:**
  1. Buat worker background berbasis Go goroutine dengan ticker periodik berdurasi 60 detik.
  2. Lakukan passive sync: baca ringkasan metrik dari circuit breaker memory state.
  3. Lakukan active probe: kirimkan ping request HTTP ringan ke setiap endpoint berstatus aktif dengan timeout ketat 5 detik.
  4. Perbarui status kolom `health` (`healthy` atau `unhealthy`) dan timestamp `last_check_at` pada database dan cache memori.
  5. Sediakan channel stop untuk graceful shutdown worker saat aplikasi menerima sinyal terminate.
- **Verifikasi (Bukti):** Background worker menandai endpoint mati sebagai `unhealthy` dalam maksimal 60 detik dan memulihkan label `healthy` ketika upstream kembali online.
- **Catatan Zero-Downtime / Keamanan:** Health check menggunakan timeout terisolasi sehingga tidak memperlambat lalu lintas request utama pengguna.

---

### Wave 3 — Metering Token, Adapter Usecase & Penguncian Embedder

#### TASK-P03-008 — Metering Token Extractor & Heuristic Fallback
- **Target File:** `backend/internal/aigateway/metering/extractor.go`, `backend/internal/aigateway/metering/estimator.go`
- **Langkah:**
  1. Ekstrak nilai `PromptTokens`, `CompletionTokens`, dan `TotalTokens` dari respons upstream jika tersedia, set `Source = "provider_reported"`.
  2. Bangun fallback estimasi heuristik jika upstream tidak mengembalikan field usage:
     - Hitung karakter prompt dan response.
     - Terapkan rasio konversi: 1 token ≈ 4 karakter untuk teks latin/Indonesia, ditambah safety buffer 10% (ceil division).
     - Untuk multimodal vision: tambahkan flat estimate per gambar (misal 85 token per low-res patch).
     - Untuk audio: estimasikan token berdasarkan durasi audio (misal 1 detik ≈ 25 token).
     - Set label `Source = "heuristic_estimated"`.
  3. Kembalikan struct `TokenUsage` yang lengkap pada setiap output pemanggilan gateway.
- **Verifikasi (Bukti):** Unit test memastikan output `TokenUsage` selalu bernilai positif (`TotalTokens > 0`), baik respons menyertakan usage resmi maupun mengandalkan fallback heuristik.
- **Catatan Zero-Downtime / Keamanan:** Format siap dikonsumsi langsung oleh skema billing ledger Fase P04 tanpa perlu perubahan kontrak data.

#### TASK-P03-009 — Adapter Tipis di `AIUsecase` (Chat, Vision, Voice)
- **Target File:** `backend/internal/usecase/ai_usecase.go`
- **Langkah:**
  1. Hapus ketergantungan pencarian kunci pribadi pengguna dari repositori `gemini_api_keys`.
  2. Inject dependency `aigateway.AIGatewayService` ke dalam struct `aiUsecase`.
  3. Ganti pemanggilan client lama pada method chat streaming dengan `u.aiGateway.ChatStream(ctx, req)`.
  4. Ganti pemanggilan chat sinkron dengan `u.aiGateway.ChatSync(ctx, req)`.
  5. Delegasikan pemrosesan analisa struk/gambar ke `u.aiGateway.AnalyzeVision(ctx, req)`.
  6. Delegasikan transkripsi audio ke `u.aiGateway.TranscribeAudio(ctx, req)`.
  7. Pertahankan seluruh logika domain finansial yang ada (agregasi riwayat transaksi, ringkasan anggaran bulanan, deteksi anomali, serta sanitasi teks RAG).
- **Verifikasi (Bukti):** Unit test usecase chat, vision, dan voice berhasil dieksekusi dengan mock gateway service; format balasan finansial tetap identik.
- **Catatan Zero-Downtime / Keamanan:** Dilarang merombak prompt engineering finansial yang telah tervalidasi di P01; adaptasi dilakukan strictly pada level integrasi transport.

#### TASK-P03-010 — Adapter Embedder 768d Terkunci
- **Target File:** `backend/internal/infrastructure/embedding/embedder.go`
- **Langkah:**
  1. Hubungkan embedder sistem ke rute gateway `aigateway.CreateEmbedding(ctx, text)` dengan model target `text-embedding-004`.
  2. Pasang validasi runtime ketat terhadap dimensi array vektor:
     ```go
     if len(vector) != 768 {
         return nil, fmt.Errorf("invalid embedding dimension: expected 768, got %d", len(vector))
     }
     ```
  3. Pastikan vektor hasil embedding diteruskan ke repository pencarian transaksi pgvector.
  4. Uji kalkulasi cosine similarity terhadap data vektor di tabel `transaction_embeddings`.
- **Verifikasi (Bukti):** Test eksekusi embedding menghasilkan tepat 768 float array; query pgvector `vector_cosine_ops` berhasil mengembalikan data tanpa memicu database error `SQLSTATE 22000`.
- **Catatan Zero-Downtime / Keamanan:** Mencegah kontaminasi dimensi yang berpotensi merusak indeks IVFFlat / HNSW pada database operasional.

---

### Wave 4 — Penghentian BYOK & Pembersihan Frontend UX

#### TASK-P03-011 — Deprecate Endpoint API Key Lama & Pengarsipan Kunci
- **Target File:** `backend/internal/delivery/http/handler/gemini_key_handler.go`, `backend/internal/delivery/http/router.go`
- **Langkah:**
  1. Modifikasi HTTP handler lama manajemen kunci mandiri user (`/api/v1/gemini-keys`).
  2. Endpoint mutasi (`POST`, `PUT`, `DELETE`) diubah mengembalikan response `HTTP 410 Gone` dengan payload JSON:
     ```json
     {
       "status": 410,
       "error": "BYOK_DEPRECATED",
       "message": "Fitur API Key mandiri telah dihentikan. FinTrack v2 kini menggunakan AI Gateway terpusat."
     }
     ```
  3. Endpoint pembacaan (`GET /api/v1/gemini-keys`) diubah mengembalikan array kosong `[]` dengan status `HTTP 200 OK` guna mencegah crash pada klien lawas yang belum disegarkan.
  4. Eksekusi pengarsipan data kunci lama pada database: `UPDATE gemini_api_keys SET is_archived = true, archived_at = NOW() WHERE is_archived = false;`.
- **Verifikasi (Bukti):** Pengujian curl ke endpoint `/api/v1/gemini-keys`: method POST mengembalikan `410 Gone`; query PostgreSQL memastikan seluruh record `gemini_api_keys` berstatus `is_archived = true`.
- **Catatan Zero-Downtime / Keamanan:** Kunci lama tidak dihapus secara permanen (hard delete) demi kepatuhan audit & jejak forensik.

#### TASK-P03-012 — Frontend BYOK Cleanup & UX Polish
- **Target File:** `frontend/components/sidebar.tsx`, `frontend/app/settings/page.tsx`, `frontend/components/chat/chat-interface.tsx`, `frontend/hooks/use-gemini-key.ts`
- **Langkah:**
  1. Hapus entri menu "API Keys" pada navigasi sidebar dan settings overview.
  2. Hapus halaman atau modal formulir input API key mandiri.
  3. Buat redirect route dari `/settings/api-keys` menuju `/settings/general` jika diakses langsung oleh URL pengguna.
  4. Bersihkan komponen Chat: hapus dialog peringatan "Kunci API Anda habis / belum diisi".
  5. Perbarui pesan penanganan error pada antarmuka chat: tampilkan pesan server yang informatif dan ramah jika upstream AI mengalami gangguan temporer.
- **Verifikasi (Bukti):** Linter dan build frontend lulus (`pnpm build`); navigasi sidebar tidak memuat menu API Keys; akses URL `/settings/api-keys` teralihkan dengan benar; komponen chat berfungsi mulus.
- **Catatan Zero-Downtime / Keamanan:** Menghilangkan kebingungan pengguna dan meniadakan permukaan serangan penyimpanan API key di sisi klien/browser.

---

### Wave 5 — Integrasi Pengujian, Benchmark & Rollout

#### TASK-P03-013 — Integration Test Failover, Timeout & Circuit Breaker
- **Target File:** `backend/internal/aigateway/gateway_test.go`
- **Langkah:**
  1. Bangun suite integration test komprehensif menggunakan mock HTTP server upstream:
     - **Test 1 (Normal Stream):** Request chat SSE terkirim ke 9router lokal, token streaming diterima lengkap beserta `TokenUsage`.
     - **Test 2 (Transparent Failover):** Endpoint Priority 1 disimulasikan 503 Service Unavailable -> gateway otomatis meneruskan request ke Endpoint Priority 2 dan mengembalikan hasil sukses ke user.
     - **Test 3 (Circuit Breaker Tripping):** Picu 5 kali error berurutan pada endpoint -> verifikasi status sirkuit menjadi `StateOpen` -> request ke-6 ditolak seketika (fast failover) tanpa mengirim traffic jaringan ke endpoint primer.
     - **Test 4 (Half-Open Recovery):** Majukan waktu clock melewati cooldown 30 detik -> status sirkuit menjadi `StateHalfOpen` -> kirim 1 request probe sukses -> verifikasi sirkuit pulih menjadi `StateClosed`.
     - **Test 5 (Timeout Enforcement):** Rute dengan setting `timeout_ms = 100` memutus koneksi upstream lambat (> 200ms) dan memicu failover tepat waktu.
- **Verifikasi (Bukti):** Seluruh suite pengujian di `backend/internal/aigateway/` lulus tanpa flaky failure (`go test -v -race ./backend/internal/aigateway/...`).
- **Catatan Zero-Downtime / Keamanan:** Memastikan keandalan penanganan kegagalan jaringan secara mandiri tanpa bergantung pada layanan pihak ketiga saat CI pipeline berjalan.

#### TASK-P03-014 — Benchmark & Zero-Downtime Rollout Verification
- **Target File:** `backend/internal/aigateway/benchmark_test.go`, `second_brain/2_Projects/P03_AI-Gateway-Terpusat/tracking/rollout-verification.md`
- **Langkah:**
  1. Jalankan benchmark konkurensi: uji beban 50 concurrent SSE streaming connections untuk memverifikasi kestabilan goroutine pooling dan memori heap.
  2. Ukur overhead latensi gateway internal: pastikan latensi routing + breaker checking < 15ms di luar waktu respon model LLM.
  3. Konfigurasikan feature flag integrasi `ff_ai_gateway_centralized` (default ON di staging, dapat di-toggle instan via Redis/DB).
  4. Lakukan simulasi rolling update deployment backend: pastikan koneksi HTTP chat streaming yang sedang berjalan tidak terputus secara mendadak.
  5. Catat hasil pengujian beban, konsumsi CPU, dan alokasi memori ke dalam artefak verifikasi.
- **Verifikasi (Bukti):** Laporan benchmark mengonfirmasi latensi internal < 15ms; tidak terdeteksi memory leak (`pprof` / heap profile stabil); zero-downtime deployment terverifikasi.
- **Catatan Zero-Downtime / Keamanan:** Layanan asisten finansial tetap aktif 100% selama proses transisi migrasi infrastruktur.

---

## 4. Matriks Bukti (Evidence)

| Task | Artefak Bukti yang Wajib Dihasilkan |
|---|---|
| **TASK-P03-001** | Log migrasi SQL `0007_ai_gateway.sql` sukses + output psql `\d ai_endpoints` & `\d ai_routes` |
| **TASK-P03-002** | File domain terkompilasi bersih (`go build ./backend/internal/domain/...`) |
| **TASK-P03-003** | Log unit test enkripsi AES-GCM + test repository route cache (`go test ./backend/internal/infrastructure/repository/...`) |
| **TASK-P03-004** | Unit test OpenAI client & SSE chunk parsing (`go test ./backend/internal/aigateway/client/...`) |
| **TASK-P03-005** | Unit test transisi Circuit Breaker (Closed -> Open -> Half-Open -> Closed) |
| **TASK-P03-006** | Unit test Router failover multi-endpoint (< 1s peredam kegagalan) |
| **TASK-P03-007** | Unit test worker health checker (deteksi endpoint sehat & tidak sehat dalam 60s) |
| **TASK-P03-008** | Unit test metering token (ekstraksi upstream & formula heuristik fallback) |
| **TASK-P03-009** | Log pengujian chat streaming, vision, voice melalui adapter `ai_usecase.go` |
| **TASK-P03-010** | Log verifikasi dimensi embedding: array tepat 768 float & validasi query cosine pgvector |
| **TASK-P03-011** | Output curl handler BYOK mengembalikan `HTTP 410 Gone` + query bukti pengarsipan `gemini_api_keys` |
| **TASK-P03-012** | Log build frontend `pnpm build` sukses + screenshot verifikasi UI bebas menu API Key |
| **TASK-P03-013** | Log test terpadu `go test -v -race ./backend/internal/aigateway/...` seluruhnya PASS |
| **TASK-P03-014** | Dokumen hasil benchmark konkurensi (latensi < 15ms) & verifikasi zero downtime |

---

## 5. Perintah Verifikasi

```bash
# ==============================================================================
# 1. VERIFIKASI BACKEND (GO)
# ==============================================================================
# Analisis statis dan deteksi kesalahan tipe data
cd D:/project/finance/fintrack_v2/backend && go vet ./...

# Eksekusi unit test & integration test paket aigateway dengan race detector
go test -v -race ./internal/aigateway/...

# Eksekusi seluruh test backend untuk memastikan tidak ada regresi finansial
go test -v ./internal/usecase/... ./internal/infrastructure/...

# Verifikasi kompilasi binary backend
go build -v -o /dev/null ./cmd/server

# ==============================================================================
# 2. VERIFIKASI FRONTEND (NEXT.JS)
# ==============================================================================
# Masuk ke direktori frontend, jalankan linter, test, dan build aplikasi
cd D:/project/finance/fintrack_v2/frontend
pnpm lint
pnpm test
pnpm build

# ==============================================================================
# 3. VERIFIKASI MIGRASI DATABASE & DUMP
# ==============================================================================
cd D:/project/finance/fintrack_v2
# Jalankan migrasi kanonik secara idempoten
APP_DB_PASSWORD=postgres database/scripts/migrate.sh

# Verifikasi konsistensi dump skema database
database/scripts/dump.sh
```

---

## 6. Prosedur Rollback

1. **Rollback Melalui Feature Flag (Tanpa Downtime):**
   - Jika endpoint gateway mengalami anomali tidak terduga di tahap awal rollout, ubah status feature flag `ff_ai_gateway_centralized` menjadi `false` via Redis atau dashboard admin.
   - Sistem secara aman menolak fitur AI sementara dengan pesan pemeliharaan ramah pengguna tanpa merusak alur transaksi keuangan inti.
2. **Rollback Administrasi Endpoint / Rute (Runtime Zero Deploy):**
   - Jika salah satu model atau upstream provider bermasalah: set `status = 'inactive'` pada tabel `ai_endpoints` atau `is_active = false` pada tabel `ai_routes`.
   - Router gateway secara otomatis mengabaikan rute tersebut tanpa perlu restart service aplikasi backend.
3. **Rollback Skema Database:**
   - Karena migrasi `0007_ai_gateway.sql` bersifat aditif (penambahan tabel baru dan kolom boolean arsip), skema database baru tidak merusak query versi aplikasi sebelumnya.
   - Tidak diperlukan rollback skema yang destruktif (`DROP TABLE`).

---

## 7. Handoff ke Fase Berikutnya (P04)

Setelah Fase P03 selesai dan dinyatakan lolos verifikasi:
1. **Modul Metering Token Siap Pakai:** Struct `TokenUsage` (`PromptTokens`, `CompletionTokens`, `TotalTokens`, `UseCase`, `Model`) telah terstandarisasi di setiap interaksi AI, siap dihubungkan langsung ke modul **Billing & Token Ledger** di Fase P04.
2. **Ketersediaan Kuota Terpusat:** FinTrack v2 memiliki kontrol penuh atas konsumsi AI, memungkinkan pembatasan kuota berbasis langganan (Free Tier vs Pro/Enterprise) dan pencegahan defisit token via reserve-commit pattern.
3. **Stabilitas Embedding 768d:** Basis data vektor pgvector terjamin integritasnya, mempermudah ekspansi fitur pencarian cerdas pada fase lanjutan.

---

## 8. Referensi

- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/README.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/planning/tasks.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/planning/timeline.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/gateway-architecture.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/circuit-breaker.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/routing-table.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/endpoint-pool.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/metering-token.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/embedding-strategy.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/teknis/key-rotation.md`
- `second_brain/2_Projects/P03_AI-Gateway-Terpusat/rules/implementation-rules.md`
- `second_brain/IMPLEMENTATION-PLAN.md`
- `second_brain/3_Areas/operasional/RUNBOOK_ZERO_DOWNTIME.md`
