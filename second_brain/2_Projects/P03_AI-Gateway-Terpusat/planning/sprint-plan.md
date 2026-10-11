# Sprint Plan — P03 AI Gateway Terpusat

## Ringkasan Sprint
- **Nama Sprint**: Sprint P03 — Centralized AI Gateway & Routing Engine
- **Fokus Utama**: Membangun modul terpusat `internal/aigateway/`, skema database `ai_endpoints` dan `ai_routes`, circuit breaker, failover engine, adapter tipis `AIUsecase`, pengarsipan BYOK, dan penguncian embedding 768d.
- **Durasi / Alokasi**: 2 Minggu (10 Hari Kerja)
- **Kapasitas**: 1 Backend Senior, 1 Fullstack Engineer (FE Cleanup & Admin Config), 1 QA / Performance Tester
- **Prasyarat**: P01 Selesai; skema PostgreSQL pgvector 768d aktif; 9router container terpasang pada port 20128.

---

## Pembagian Sprint Wave

### Wave 1: Fondasi Skema Database & Domain Modeling (Hari 1 - 2)
- **Tujuan**: Mempersiapkan skema persistensi, enkripsi kunci provider, dan kontrak antarmuka di Go domain.
- **Deliverables**:
  - Migrasi `0007_ai_gateway.sql` (tabel `ai_endpoints`, `ai_routes`, alter `gemini_api_keys`).
  - Definisi interface dan model di `internal/domain/aigateway.go`.
  - Repository layer untuk operasi CRUD endpoint dan route.

### Wave 2: Core AI Gateway Engine (`internal/aigateway/`) (Hari 3 - 5)
- **Tujuan**: Membangun client OpenAI-compatible, pool balancer, circuit breaker, dan router multi-provider.
- **Deliverables**:
  - Client HTTP OpenAI-compatible dengan stream SSE parser (`client.go`).
  - State machine Circuit Breaker Closed/Open/Half-Open (`circuit_breaker.go`).
  - Router seleksi model & failover by priority (`router.go`).
  - Health check scheduler background worker (`health.go`).
  - Token usage extractor & heuristic fallback estimator (`metering.go`).

### Wave 3: Adapter Refactoring di `AIUsecase` & Embedder 768d (Hari 6 - 7)
- **Tujuan**: Menghubungkan gateway ke business logic chat, vision, voice, dan RAG tanpa merusak flow lama.
- **Deliverables**:
  - Refactor `internal/usecase/ai_usecase.go` untuk mendelegasikan chat, vision, dan voice ke gateway interface.
  - Refactor `internal/infrastructure/embedding/embedder.go` untuk memanggil endpoint embedding terpusat via gateway.
  - Validasi ketat output dimensi embedding (wajib 768 float).

### Wave 4: Frontend Cleanup & Deprecate BYOK (Hari 8 - 9)
- **Tujuan**: Menghilangkan kontrol API key perorangan di user space dan merapikan pesan user-facing.
- **Deliverables**:
  - Hapus halaman/modal API Key user di Frontend Next.js.
  - Perbarui error boundaries dan pesan sistem (tidak lagi menampilkan peringatan "ganti API key di menu pengaturan").
  - Redirect route lama `/settings/api-keys` ke halaman profile atau notifikasi fitur terpusat.

### Wave 5: Testing, Hardening & Zero-Downtime Deployment (Hari 10)
- **Tujuan**: Verifikasi ketahanan failover, benchmarking circuit breaker, dan deployment aman.
- **Deliverables**:
  - Test skenario failover: simulasi 9router down -> auto failover ke Cloud fallback.
  - Test circuit breaker 5 errors -> 30s cooldown.
  - Verifikasi RAG embedding cosine distance pencarian transaksi.
  - Sign-off Definition of Done dan aktivasi feature flag.

---

## Matriks Tanggung Jawab & Dependensi

| ID Task | Judul Task | Assignee | Dependensi | Status |
|---|---|---|---|---|
| TASK-P03-001 | Skema Database & Migrasi `0007_ai_gateway.sql` | Backend | - | ⬜ Ready |
| TASK-P03-002 | Domain Modeling & Entity Gateway di Go | Backend | TASK-P03-001 | ⬜ Ready |
| TASK-P03-003 | Repository SQL Endpoint & Routing Table | Backend | TASK-P03-002 | ⬜ Ready |
| TASK-P03-004 | OpenAI-Compatible HTTP Client & Streaming Parser | Backend | TASK-P03-002 | ⬜ Ready |
| TASK-P03-005 | State Machine Circuit Breaker (30s Cooldown) | Backend | TASK-P03-004 | ⬜ Ready |
| TASK-P03-006 | Router Multi-Endpoint & Failover Chain | Backend | TASK-P03-003, TASK-P03-005 | ⬜ Ready |
| TASK-P03-007 | Active & Passive Health Checker Worker | Backend | TASK-P03-006 | ⬜ Ready |
| TASK-P03-008 | Metering Token Extractor & Heuristic Fallback | Backend | TASK-P03-004 | ⬜ Ready |
| TASK-P03-009 | Adapter Tipis `AIUsecase` (Chat, Vision, Voice) | Backend | TASK-P03-006, TASK-P03-008 | ⬜ Ready |
| TASK-P03-010 | Adapter Embedder 768d Terkunci | Backend | TASK-P03-006 | ⬜ Ready |
| TASK-P03-011 | Deprecate Endpoint API Key Lama & Pengarsipan | Backend | TASK-P03-001, TASK-P03-009 | ⬜ Ready |
| TASK-P03-012 | Frontend BYOK Cleanup & UX Polish | Frontend | TASK-P03-011 | ⬜ Ready |
| TASK-P03-013 | Integration Test Failover & Circuit Breaker | QA/Backend | TASK-P03-009, TASK-P03-010 | ⬜ Ready |
| TASK-P03-014 | Benchmark & Zero-Downtime Rollout Verification | Team | TASK-P03-012, TASK-P03-013 | ⬜ Ready |

---

## Risiko dan Rencana Mitigasi

1. **Risiko**: Upstream 9router lokal mengalami restart mendadak saat user streaming chat.
   - **Mitigasi**: Client gateway mendeteksi `io.ErrUnexpectedEOF` atau status 5xx, langsung mengeksekusi rute sekunder (cloud direct fallback) secara transparan sebelum stream chunk pertama dikirim ke user jika memungkinkan.
2. **Risiko**: Model embedding baru menghasilkan dimensi bukan 768 sehingga pgvector menolak insert.
   - **Mitigasi**: Pre-check dimensi pada unit test dan inisialisasi runtime gateway; jika `len(vector) != 768`, gateway menghentikan flow dengan error eksplisit dan menolak konfigurasi disimpan di database.
3. **Risiko**: Token usage tidak dikembalikan oleh provider third-party tertentu saat streaming.
   - **Mitigasi**: Heuristic token estimator (`pkg/tokenizer`) otomatis aktif menghitung akumulasi buffer karakter teks respons.
