# Hasil Analisis Teknis — P03 AI Gateway Terpusat

Dokumen ini mendefinisikan keputusan arsitektural dan teknis terkunci khusus untuk Fase P03 (AI Gateway Terpusat), selaras dengan keputusan global K-01 sampai K-30.

## Keputusan Teknis Terkunci (K-P03-01 s/d K-P03-16)

### K-P03-01: Arsitektur Modul AI Gateway Terpisah (`internal/aigateway`)
- **Keputusan**: Modul AI Gateway dibangun sebagai package tersendiri `backend/internal/aigateway/` dengan interface independen, bukan logic tambahan yang ditumpuk di `internal/usecase/ai_usecase.go`.
- **Rasional**: Memenuhi prinsip Strangler Fig (K-24). Memisahkan networking, pool balancing, circuit breaking, dan failover dari business logic chat/RAG keuangan.
- **Dampak**: `AIUsecase` bertindak hanya sebagai consumer/adapter terhadap `aigateway.Service`.

### K-P03-02: Endpoint Primer 9router Lokal & Kontrak OpenAI-Compatible
- **Keputusan**: Endpoint prioritas tertinggi (Priority 1) diarahkan ke 9router lokal (`http://localhost:20128/v1`), yang mengabstraksikan berbagai provider upstream ke format standar OpenAI `/chat/completions` dan `/embeddings`.
- **Rasional**: Mengurangi latensi, menyatukan format parsing stream SSE upstream, serta menyederhanakan mocking/testing lokal tanpa token cloud berbayar.
- **Dampak**: Semua payload request dan stream parser di internal aigateway mengimplementasikan spesifikasi OpenAI format (`delta.content`, `usage`).

### K-P03-03: Skema Tabel `ai_endpoints` dan Enkripsi Kunci
- **Keputusan**: Konfigurasi endpoint disimpan di database tabel `ai_endpoints` dengan skema:
  - `id` (UUID PK)
  - `name` (VARCHAR)
  - `base_url` (TEXT)
  - `api_key_encrypted` (BYTEA / TEXT, AES-GCM 256 menggunakan `cryptoKey` aplikasi)
  - `provider` (VARCHAR: `9router`, `openai`, `gemini`, `groq`, dll.)
  - `priority` (INT, 1 = tertinggi)
  - `status` (VARCHAR: `active`, `degraded`, `inactive`, `suspended`)
  - `health` (VARCHAR: `healthy`, `unhealthy`, `unknown`)
  - `last_check_at` (TIMESTAMPTZ)
  - `modalities` (TEXT[]: `['chat', 'vision', 'voice', 'embedding', 'reasoning']`)
  - `weight` (INT default 100)
  - `created_at`, `updated_at`
- **Rasional**: Menyimpan konfigurasi dinamis tanpa memerlukan restart service atau rebuild image container (K-17, K-44).

### K-P03-04: Skema Tabel `ai_routes` dan Pemetaan Dinamis Use-Case
- **Keputusan**: Pemetaan model dan provider per domain fitur dipisahkan ke tabel `ai_routes`:
  - `id` (UUID PK)
  - `use_case` (VARCHAR: `chat`, `vision`, `voice`, `embedding`, `reasoning`)
  - `endpoint_id` (UUID FK ke `ai_endpoints.id` ON DELETE CASCADE)
  - `model` (VARCHAR, misal `gpt-4o-mini`, `gemini-1.5-flash`, `text-embedding-004`)
  - `combo` (VARCHAR, misal `fast`, `balanced`, `precise`, default `default`)
  - `priority` (INT, fallback chain dalam satu use case)
  - `is_active` (BOOLEAN default true)
  - `timeout_ms` (INT default 30000)
  - `max_retries` (INT default 2)
  - `created_at`, `updated_at`
- **Rasional**: Menghapus map provider hardcoded di kode Go. Super admin dapat mengganti model atau mengalihkan beban jika provider tertentu mengalami lonjakan error/harga.

### K-P03-05: State Machine Circuit Breaker (Closed, Open, Half-Open)
- **Keputusan**: Setiap tuple `(endpoint_id, use_case)` atau endpoint global memiliki state machine Circuit Breaker in-memory dengan spesifikasi:
  - **Threshold Kegagalan**: 5 kegagalan berturut-turut (HTTP 5xx, network timeout, rate limit 429 berulang) memicu transisi dari **Closed** ke **Open**.
  - **Cooldown**: 30 detik saat status **Open**. Selama cooldown, request langsung diarahkan ke fallback endpoint berikutnya tanpa menyentuh endpoint yang rusak.
  - **Half-Open**: Setelah 30 detik, 1 request probe diizinkan. Jika berhasil, state kembali ke **Closed** (reset counter kegagalan). Jika gagal, kembali ke **Open** selama 30 detik berikutnya.
  - HTTP 4xx (client error, misal invalid prompt) tidak dihitung sebagai kegagalan upstream circuit breaker.
- **Rasional**: Mencegah request starvation dan cascade failure ke seluruh sistem ketika salah satu provider AI mengalami gangguan eksternal.

### K-P03-06: Failover Chain Otomatis Berdasarkan Prioritas
- **Keputusan**: Panggilan request akan mengevaluasi rute aktif berdasarkan `priority` terendah (angka 1 = paling utama). Jika rute primer berada dalam status Circuit Breaker Open atau mengembalikan error 5xx/timeout/429:
  1. Circuit breaker mencatat kegagalan pada rute primer.
  2. Gateway secara transparan mencoba rute sekunder (Priority 2).
  3. Proses berlanjut hingga max retries atau rute habis.
  4. Jika semua rute gagal, gateway mengembalikan response fallback yang ramah (degraded graceful response).
- **Rasional**: Memastikan 99.9% availability fitur asisten finansial tanpa user merasakan error teknis server.

### K-P03-07: Health Check Proaktif dan Reaktif
- **Keputusan**: Gateway menggunakan sistem health check hibrida:
  - **Reaktif**: Hasil dari pemanggilan user request mengupdate status latency & error rate endpoint secara realtime.
  - **Proaktif (Background Worker)**: Ticker berjalan setiap 60 detik mengirimkan lightweight ping/probe (misal request embedding 1 kata atau model list call) ke endpoint aktif dengan timeout 5 detik.
- **Rasional**: Menemukan endpoint yang pulih dari gangguan tanpa harus mengorbankan latency request pengguna pertama.

### K-P03-08: Kill Switch Berjenjang (Endpoint Level & Use-Case Level)
- **Keputusan**: Mekanisme pemutusan darurat (Kill Switch) tersedia pada 2 tingkatan:
  1. **Level Endpoint**: Mengubah `ai_endpoints.status = 'inactive'` secara instan menghentikan seluruh trafik ke endpoint tersebut.
  2. **Level Use-Case**: Flag di database/Redis untuk mematikan kategori fitur tertentu (misal mematikan `voice` atau `vision` jika terjadi lonjakan biaya tak wajar) dengan balasan error informatif ke user.
- **Rasional**: Memberikan kontrol mitigasi instan bagi super admin terhadap anomali biaya atau abuse tanpa down time sistem.

### K-P03-09: Isolasi Adapter Tipis pada `AIUsecase` Tanpa Rewrite
- **Keputusan**: Logic chat orchestration, validasi kepemilikan sesi, context financial injection, dan streaming SSE di `internal/usecase/ai_usecase.go` dipertahankan. Hanya kode seleksi endpoint dan direct HTTP call client yang diganti memanggil `aigateway.Gateway`.
- **Rasional**: Menjaga stabilitas kode 2000+ baris yang sudah teruji fungsionalitasnya di P01, menghindari regresi flow chat dan RAG.

### K-P03-10: Penguncian Dimensi Embedding pada 768d
- **Keputusan**: Model embedding diwajibkan menghasilkan vektor dimensi 768 float (`vector(768)`). Default model: `text-embedding-004` (atau model kompatibel yang menghasilkan output 768d).
- **Rasional**: Tabel `transaction_embeddings` di PostgreSQL menggunakan pgvector dengan tipe `embedding vector(768)`. Mengganti dimensi membutuhkan re-embedding seluruh riwayat transaksi user yang mahal dan memicu downtime.
- **Dampak**: Model route dengan use_case `embedding` divalidasi ketat saat konfigurasi disimpan; jika dimensi bukan 768, sistem menolak konfigurasi tersebut.

### K-P03-11: Penghapusan BYOK dan Pengarsipan Tabel `gemini_api_keys`
- **Keputusan**: Kolom/fitur BYOK (Bring Your Own Key) dicabut dari akses user.
  - Tabel `gemini_api_keys` di PostgreSQL tidak di-DROP fisik, melainkan ditambah kolom `is_archived BOOLEAN DEFAULT TRUE`, `archived_at TIMESTAMPTZ`.
  - Endpoint HTTP user untuk CRUD API key di-deprecate (mengembalikan 410 Gone atau dinonaktifkan di router).
  - UI frontend komponen kunci user dihapus.
- **Rasional**: Menjaga audit compliance data historis tanpa menyisakan lubang pemanggilan LLM mandiri yang membypass sistem tokenisasi terpusat.

### K-P03-12: Standardisasi Metering Token (Provider Usage + Fallback Estimator)
- **Keputusan**: Setiap eksekusi AI menghasilkan metadata konsumsi token yang terstruktur (`PromptTokens`, `CompletionTokens`, `TotalTokens`).
  - **Sumber Utama**: Mengambil nilai dari field `usage` pada respons OpenAI-compatible payload.
  - **Sumber Fallback**: Jika provider upstream tidak mengembalikan field `usage` (terutama pada streaming SSE chunked response yang terputus di tengah), gateway menghitung token menggunakan heuristic tokenizer estimator (rata-rata 1 token ≈ 4 karakter teks bahasa Indonesia/Inggris atau 0.75 kata) dengan toleransi batas atas.
- **Rasional**: Memberikan angka konsumsi yang deterministik dan konsisten untuk pencatatan di Token Ledger (P04).

### K-P03-13: Penanganan Modality Khusus (Vision dan Voice)
- **Keputusan**:
  - Untuk `vision`, gateway menerima image base64 atau data URL, meneruskannya ke model multimodal yang dikonfigurasi pada route `vision`.
  - Untuk `voice`, audio diproses melalui endpoint OpenAI-compatible `/audio/transcriptions` (Whisper) atau multimodal prompt audio jika didukung 9router.
- **Rasional**: Menyatukan kontrak request multimedia ke satu pipeline gateway yang sama.

### K-P03-14: In-Memory Caching Konfigurasi Route dengan Invalidation Pub/Sub
- **Keputusan**: Daftar route dan endpoint aktif di-cache di in-memory gateway service Go. Jika ada perubahan konfigurasi oleh super admin di database, gateway mendengarkan signal invalidasi via Redis Pub/Sub (atau local sync timer fallback 30s) untuk me-reload tabel route.
- **Rasional**: Lookup rute AI terjadi pada setiap pesan user. Cache in-memory mencegah overhead query SQL berulang pada setiap interaksi LLM.

### K-P03-15: Zero-Downtime Deployment & Feature Flagging
- **Keputusan**: Fungsionalitas gateway terpusat dikawal feature flag `AI_GATEWAY_CENTRALIZED_ENABLED`.
  - Jika flag bernilai `false`, fallback legacy tetap dapat berjalan (selama masa transisi migrasi).
  - Begitu verifikasi smoke test lolos, flag diaktifkan `true` secara global tanpa restart pod/container.
- **Rasional**: Memastikan transisi migrasi dari P01 ke P03 berjalan tanpa gangguan layanan bagi pengguna aktif.

### K-P03-16: Logging & Telemetri Gateway Append-Only
- **Keputusan**: Setiap invocation gateway dicatat dengan level trace/info berisi: `request_id`, `use_case`, `endpoint_id`, `model`, `latency_ms`, `http_status`, `prompt_tokens`, `completion_tokens`, `is_failover`.
- **Rasional**: Menyediakan data observabilitas untuk audit log, debugging performa upstream, dan rekonsiliasi billing token.
