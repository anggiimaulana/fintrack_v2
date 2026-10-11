# Task Breakdown — P03 AI Gateway Terpusat

Daftar tugas implementasi terperinci untuk Fase P03. Format identifikasi tugas: `TASK-P03-NNN`.

---

### TASK-P03-001: Skema Database & Migrasi `0007_ai_gateway.sql`
- **Scope**:
  - Buat file migrasi `database/migrations/0007_ai_gateway.sql`.
  - Buat tabel `ai_endpoints`:
    - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
    - `name VARCHAR(100) NOT NULL`
    - `base_url TEXT NOT NULL`
    - `api_key_encrypted BYTEA NOT NULL`
    - `provider VARCHAR(50) NOT NULL`
    - `priority INT NOT NULL DEFAULT 1`
    - `status VARCHAR(20) NOT NULL DEFAULT 'active'`
    - `health VARCHAR(20) NOT NULL DEFAULT 'healthy'`
    - `last_check_at TIMESTAMPTZ`
    - `modalities TEXT[] NOT NULL DEFAULT '{"chat","vision","voice","embedding","reasoning"}'`
    - `weight INT NOT NULL DEFAULT 100`
    - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
    - `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - Buat tabel `ai_routes`:
    - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
    - `use_case VARCHAR(50) NOT NULL`
    - `endpoint_id UUID NOT NULL REFERENCES ai_endpoints(id) ON DELETE CASCADE`
    - `model VARCHAR(100) NOT NULL`
    - `combo VARCHAR(50) NOT NULL DEFAULT 'default'`
    - `priority INT NOT NULL DEFAULT 1`
    - `is_active BOOLEAN NOT NULL DEFAULT TRUE`
    - `timeout_ms INT NOT NULL DEFAULT 30000`
    - `max_retries INT NOT NULL DEFAULT 2`
    - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
    - `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - Tambah kolom arsip pada tabel lama `gemini_api_keys`:
    - `ALTER TABLE gemini_api_keys ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT TRUE;`
    - `ALTER TABLE gemini_api_keys ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ DEFAULT NOW();`
  - Seed awal:
    - 1 endpoint default 9router lokal (`http://localhost:20128/v1`, provider `9router`, priority 1).
    - Routes default untuk `chat` (model `gpt-4o-mini`), `vision` (`gpt-4o-mini`), `voice` (`whisper-1`), `embedding` (`text-embedding-004`), `reasoning` (`deepseek-r1`).
- **Acceptance Criteria**: Migrasi berjalan idempotent (UP dan DOWN berfungsi tanpa error), constraint valid.

---

### TASK-P03-002: Domain Modeling & Interface Gateway
- **Scope**:
  - Buat file `backend/internal/domain/aigateway.go`.
  - Definisikan tipe entitas `AIEndpoint`, `AIRoute`, `UseCaseType` (`chat`, `vision`, `voice`, `embedding`, `reasoning`).
  - Definisikan struct DTO request/response: `AIChatRequest`, `AIChatStreamChunk`, `AIEmbeddingRequest`, `AIEmbeddingResponse`, `TokenUsage`.
  - Definisikan interface:
    - `AIEndpointRepository`
    - `AIRouteRepository`
    - `AIGatewayService` (metode `ChatStream`, `ChatSync`, `CreateEmbedding`, `TranscribeAudio`, `AnalyzeVision`)
    - `CircuitBreaker`
    - `HealthChecker`
- **Acceptance Criteria**: Interface bersih, bebas import cycles, siap digunakan oleh usecase dan repository.

---

### TASK-P03-003: Repository SQL AI Gateway & Route Cache
- **Scope**:
  - Buat file `backend/internal/infrastructure/repository/ai_gateway_repository.go`.
  - Implementasikan query database untuk mengambil list active routes per use case diurutkan berdasarkan priority ASC.
  - Implementasikan enkripsi dan dekripsi `api_key_encrypted` menggunakan AES-GCM 256 (`crypto.Key`).
  - Implementasikan update status health, last_check_at, dan status endpoint.
  - Tambahkan in-memory thread-safe cache (`sync.RWMutex`) untuk active routes dengan TTL pendek / invalidation method.
- **Acceptance Criteria**: Query efisien (<2ms dari DB atau 0ms dari in-memory cache), dekripsi kunci aman.

---

### TASK-P03-004: OpenAI-Compatible HTTP Client Engine
- **Scope**:
  - Buat package `backend/internal/aigateway/client/`.
  - Bangun HTTP client berbasis standar OpenAI REST API (`/v1/chat/completions`, `/v1/embeddings`, `/v1/audio/transcriptions`).
  - Dukung HTTP Keep-Alive connection pooling, custom timeouts per rute.
  - Parsing streaming respons SSE (`data: {"choices":[{"delta":{"content":"..."}}],"usage":{...}}`).
  - Parsing token usage dari field `usage` upstream.
- **Acceptance Criteria**: Client berhasil mengirim dan menerima chunk SSE dari 9router lokal dan standard OpenAI endpoints.

---

### TASK-P03-005: State Machine Circuit Breaker
- **Scope**:
  - Buat package `backend/internal/aigateway/breaker/`.
  - Implementasikan state machine:
    - `StateClosed`: Semua request diteruskan; counter error di-reset saat sukses. Jika consecutive errors >= 5, transisi ke `StateOpen`.
    - `StateOpen`: Semua request ditolak langsung (fast failover ke rute berikutnya). Mulai timer cooldown 30 detik.
    - `StateHalfOpen`: Setelah 30 detik cooldown, 1 request diperbolehkan melintas (probe). Jika berhasil, kembali ke `StateClosed`. Jika gagal, kembali ke `StateOpen` selama 30 detik lagi.
  - Error yang dihitung kegagalan: status HTTP 5xx, gateway timeout, connection refused, 429 berulang. HTTP 400/404 (client error) diabaikan.
- **Acceptance Criteria**: Unit test membuktikan transisi Closed -> Open -> Half-Open -> Closed atau Half-Open -> Open berjalan presisi sesuai timing.

---

### TASK-P03-006: Router Multi-Endpoint & Failover Execution Engine
- **Scope**:
  - Buat package `backend/internal/aigateway/router/`.
  - Ambil daftar route terurut berdasarkan `priority ASC` untuk `use_case` tertentu.
  - Iterasi melalui fallback chain:
    1. Cek status Circuit Breaker endpoint rute. Jika Open, lewati ke rute berikutnya.
    2. Cek status Kill Switch (apakah endpoint inactive). Jika inactive, lewati.
    3. Jalankan request ke client rute terpilih.
    4. Jika sukses, laporkan sukses ke circuit breaker, return result/stream.
    5. Jika gagal (timeout/5xx), catat error ke circuit breaker, log warning, lalu coba rute berikutnya (Priority 2).
  - Jika seluruh rute gagal, kembalikan sentinel error `ErrAllEndpointsExhausted`.
- **Acceptance Criteria**: Simulasi error pada rute primer otomatis memindahkan eksekusi ke rute sekunder dalam <1 detik tanpa crash.

---

### TASK-P03-007: Active & Passive Health Checker Background Worker
- **Scope**:
  - Buat package `backend/internal/aigateway/health/`.
  - Background worker berjalan via Go goroutine ticker (default interval 60s).
  - Menguji setiap endpoint aktif dengan ping ringan (health endpoint atau lightweight embedding/chat ping).
  - Timeout health check: 5 detik.
  - Update kolom `health` (`healthy` / `unhealthy`) dan `last_check_at` di database dan in-memory state.
- **Acceptance Criteria**: Endpoint mati terdeteksi dalam maksimal 60 detik; endpoint pulih ditandai healthy kembali secara otomatis.

---

### TASK-P03-008: Metering Token Extractor & Heuristic Fallback
- **Scope**:
  - Buat package `backend/internal/aigateway/metering/`.
  - Ekstrak nilai `PromptTokens`, `CompletionTokens`, `TotalTokens` dari response upstream jika tersedia.
  - Implementasikan fallback heuristic tokenizer bila provider upstream tidak mengirimkan data usage:
    - Hitung karakter prompt dan response.
    - Rasio: 1 token ≈ 4 karakter untuk teks latin/Indonesia, ditambah safety margin buffer 10%.
  - Emit event telemetri token untuk konsumsi downstream (persiapan ledger Fase P04).
- **Acceptance Criteria**: Setiap request AI mengembalikan struct `TokenUsage` yang valid dan tidak bernilai 0.

---

### TASK-P03-009: Adapter Tipis di `AIUsecase` (Chat, Vision, Voice)
- **Scope**:
  - Perbarui `backend/internal/usecase/ai_usecase.go`.
  - Hapus dependensi pemanggilan langsung ke provider eksternal per-user key (`gemini_api_keys`).
  - Ganti pemanggilan LLM dengan `u.aiGateway.ChatStream(ctx, req)` atau `u.aiGateway.ChatSync(ctx, req)`.
  - Delegasikan vision prompt ke `u.aiGateway.AnalyzeVision(ctx, req)`.
  - Delegasikan voice transcription ke `u.aiGateway.TranscribeAudio(ctx, req)`.
  - Pertahankan seluruh flow context financial gathering (transaksi, kategori, ringkasan saldo) dan sanitasi prompt.
- **Acceptance Criteria**: Chatbot, analisa struk via vision, dan input suara tetap berfungsi mulus dari perspektif pengguna.

---

### TASK-P03-010: Adapter Embedder 768d Terkunci
- **Scope**:
  - Perbarui `backend/internal/infrastructure/embedding/embedder.go`.
  - Integrasikan embedder ke `aigateway.CreateEmbedding(ctx, text)`.
  - Wajibkan validasi dimensi:
    - `if len(vector) != 768 { return nil, ErrInvalidEmbeddingDimension }`.
  - Pastikan pencarian RAG transaksi berbasis cosine similarity tetap menghasilkan skor kecocokan valid terhadap tabel `transaction_embeddings`.
- **Acceptance Criteria**: Nilai vektor tetap 768-dimensi float; query pencarian kemiripan vektor pgvector berjalan normal.

---

### TASK-P03-011: Deprecate Endpoint API Key Lama & Pengarsipan Kunci
- **Scope**:
  - Nonaktifkan rute HTTP handler manajemen Gemini Key user (`/api/v1/gemini-keys`).
  - Ubah handler untuk mengembalikan `HTTP 410 Gone` dengan pesan informatif bahwa FinTrack kini menggunakan AI Gateway terpusat.
  - Eksekusi script pengarsipan data kunci lama (`UPDATE gemini_api_keys SET is_archived = true, archived_at = NOW();`).
  - Kunci lama tidak dihapus secara fisik untuk kepentingan riwayat audit.
- **Acceptance Criteria**: Pengguna tidak dapat lagi menambah, mengubah, atau menghapus API key mandiri.

---

### TASK-P03-012: Frontend BYOK Cleanup & UX Polish
- **Scope**:
  - Hapus menu / navigasi ke "API Keys" di sidebar dashboard dan settings frontend.
  - Hapus komponen formulir input API key pribadi.
  - Perbaiki pesan sistem pada komponen Chat: hilangkan peringatan seperti "Kunci API Anda habis, masukkan kunci baru di menu Settings".
  - Ganti dengan pesan kesalahan server yang ramah jika terjadi gangguan upstream.
- **Acceptance Criteria**: UI bersih dari elemen BYOK; tidak ada link mati atau error 404 pada navigasi frontend.

---

### TASK-P03-013: Integration Test Failover, Timeout & Circuit Breaker
- **Scope**:
  - Buat skenario pengujian otomatis di `tester/` atau unit test backend:
    - Test 1: Skenario normal via 9router lokal.
    - Test 2: Simulasi 9router down (503 Service Unavailable) -> gateway transparan memanggil Endpoint Priority 2.
    - Test 3: Simulasi 5 error beruntun -> Circuit breaker berpindah ke Open -> request ke-6 di-reject tanpa network call ke endpoint 1.
    - Test 4: Simulasi setelah 30 detik -> Probe half-open berhasil -> Circuit breaker pulih ke Closed.
    - Test 5: Validasi token usage reporting pada chat streaming.
- **Acceptance Criteria**: Seluruh test case lolos tanpa flaky errors.

---

### TASK-P03-014: Benchmark & Zero-Downtime Rollout Verification
- **Scope**:
  - Uji beban ringan (load test 50 concurrent SSE streaming requests) ke AI Gateway.
  - Pastikan tidak ada memory leak pada SSE connection handling dan goroutine pooling.
  - Pasang Feature Flag `AI_GATEWAY_CENTRALIZED_ENABLED`.
  - Verifikasi deployment rolling update backend: layanan chat tidak terputus bagi pengguna aktif.
- **Acceptance Criteria**: Memory footprint stabil, latency overhead gateway < 15ms, zero downtime deployment terkonfirmasi.
