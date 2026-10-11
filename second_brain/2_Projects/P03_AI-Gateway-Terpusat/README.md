# P03 — AI Gateway Terpusat

## Context
FinTrack v2 mengalihkan seluruh pemrosesan AI (chat, vision, voice, embedding) dari model BYOK (Bring Your Own Key) per-user ke AI Gateway Terpusat berbasis pool multi-endpoint. Perubahan ini mendukung monetisasi token terpusat, SLA ketersediaan tinggi, failover otomatis, pemutusan ketergantungan API key perorangan, dan konsistensi dimensi embedding vektor (768d).

## Scope of Work
1. **Modul Bersih Baru (`internal/aigateway/`)**:
   - Abstraksi interface gateway (`ProviderClient`, `EndpointPool`, `Router`, `CircuitBreaker`, `HealthChecker`).
   - Pool multi-endpoint dengan default 9router (`http://localhost:20128`, OpenAI-compatible format).
   - Dynamic routing per-use-case (`chat`, `vision`, `voice`, `embedding`, `reasoning`).
   - Circuit breaker (Closed, Open, Half-Open) dengan cooldown 30s.
   - Passive & active periodic health check (interval 60s, timeout 5s).
   - Kill switch terpisah: per-endpoint dan per-use-case.
2. **Adapter Tipis di `internal/usecase/ai_usecase.go`**:
   - Menggantikan map hardcoded provider lama tanpa rewrite masif pada orchestration RAG dan SSE session streaming.
   - Refactoring pemanggilan LLM, vision, voice, dan embedder melalui `aigateway.Client`.
3. **Database & Migrasi (`0007_ai_gateway.sql`)**:
   - Pembuatan tabel `ai_endpoints` dan `ai_routes`.
   - Seed default endpoint 9router lokal + fallback cloud route.
   - Pengarsipan tabel lama `gemini_api_keys` (kolom `archived_at` / flag arsip, data tidak dihapus).
4. **Pembersihan Frontend**:
   - Penghapusan menu UI manajemen API Key user (`/settings/api-keys` atau modal kelola BYOK).
   - Penyesuaian instruksi error LLM ke pengguna (tidak mengarahkan ke menu kunci pribadi).
5. **Kekuncian Embedding**:
   - Memastikan semua embedding transaksi dan pencarian RAG terkunci pada model berdimensi 768d (`text-embedding-004` / kompatibel).
6. **Metering Token**:
   - Ekstraksi token usage dari respons OpenAI-compatible (`prompt_tokens`, `completion_tokens`, `total_tokens`).
   - Fallback tokenizer estimasi lokal ketika usage tidak dikembalikan oleh upstream provider.

## Deliverables
1. `backend/database/migrations/0007_ai_gateway.sql` (schema `ai_endpoints`, `ai_routes`, arsip `gemini_api_keys`).
2. `backend/internal/domain/aigateway.go` (definisi entitas, DTO, interface).
3. `backend/internal/aigateway/` (implementasi pool, client, router, circuit breaker, health check, fallback tokenizer).
4. `backend/internal/usecase/ai_usecase.go` (adapter tipis penghubung usecase lama ke gateway terpusat).
5. `backend/internal/infrastructure/embedding/embedder.go` (konektor embedder 768d via gateway).
6. `frontend/` (pembersihan halaman/komponen BYOK API key user).
7. Dokumen teknis dan operasional lengkap di `second_brain/2_Projects/P03_AI-Gateway-Terpusat/`.

## Success Criteria
1. [x] Tidak ada panggilan LLM langsung dari client atau via BYOK user; trafik AI melewati internal AI Gateway saat flag ON. *(adapter wired + diuji; BYOK endpoint → 410 Gone)*
2. [x] 9router lokal sebagai primary; failover otomatis < 2 detik tanpa crash. *(diuji: failover < 1s; live: 9router mati → pesan ramah, tanpa crash)*
3. [x] Circuit breaker memblokir endpoint setelah 5 kegagalan berturut (cooldown 30s). *(diuji)*
4. [x] Dimensi embedding konsisten 768 float. *(diuji: lock menolak non-768; cosine ops tetap valid)*
5. [x] Token tercatat akurat (provider reported / fallback estimation). *(metering diuji)*
6. [x] Zero downtime via expand-contract + feature flag `feature_ai_gateway_centralized`. *(flag wired default OFF; rollout doc)*
