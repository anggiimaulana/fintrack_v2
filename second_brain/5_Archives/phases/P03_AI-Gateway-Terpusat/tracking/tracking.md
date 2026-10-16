# Tracking Status — P03 AI Gateway Terpusat

Status tracking tugas implementasi Fase P03.
- Format status: ⬜ (Belum Dimulai) | 🔄 (Sedang Dikerjakan) | ✅ (Selesai / Terverifikasi)

> **Status: SELESAI (14/14, 100%).** Semua task ✅ setelah dibuktikan test nyata.

---

## Tabel Pelacakan Tugas (TASK-P03-NNN)

| Task ID | Nama Tugas | Assignee | Status | Keterangan / Hasil Verifikasi |
|---|---|---|---|---|
| TASK-P03-001 | Skema Database & Migrasi `0007_ai_gateway.sql` | Backend | ✅ | Diterapkan ke DB; `ai_endpoints`/`ai_routes` + constraint + index + seed (1 endpoint, 5 route); kolom arsip `gemini_api_keys`; replay idempoten OK |
| TASK-P03-002 | Domain Modeling & Interface Gateway | Backend | ✅ | `internal/domain/aigateway.go` (entity, DTO, `UseCase`, `TokenUsage`, error sentinel, interface `AIGatewayService`/`CircuitBreaker`/`HealthChecker`); `go build ./internal/domain/...` OK |
| TASK-P03-003 | Repository SQL Endpoint & Route Cache | Backend | ✅ | `repository/postgres/ai_gateway_repo.go` (load routes join+priority, cache TTL 60s, list/get/update health, `DecryptKey`); 3 test PASS vs DB live |
| TASK-P03-004 | OpenAI-Compatible HTTP Client Engine | Backend | ✅ | `aigateway/client` (openai_client.go + stream_parser.go): chat stream/sync, embedding, transcribe multipart, pooled transport, `IsRetryable`. 6 test PASS (SSE, sync, embedding, transcribe, status class, auth) |
| TASK-P03-005 | State Machine Circuit Breaker | Backend | ✅ | `aigateway/breaker` (state machine + clock injection); 7 test PASS (trip 5x, half-open probe, reopen, isolation, concurrency). *-race tak tersedia (tanpa gcc)* |
| TASK-P03-006 | Router Multi-Endpoint & Failover Engine | Backend | ✅ | `aigateway/gateway/service.go` (failover chain, kill switch, breaker gate, embedding 768 lock); 6 test PASS (failover, all-exhausted, breaker skip, kill switch, dim lock) |
| TASK-P03-007 | Active & Passive Health Checker Worker | Backend | ✅ | `aigateway/gateway/health.go` (worker ticker, probe /models, update health); 4 test PASS (unhealthy, healthy, skip inactive, stop idempotent) |
| TASK-P03-008 | Metering Token Extractor & Heuristic Fallback | Backend | ✅ | `aigateway/metering` (FromProvider + Estimate + vision/audio); 8 test PASS (>0 selalu, buffer 10%, modality) |
| TASK-P03-009 | Adapter Tipis `AIUsecase` | Backend | ✅ | `ai_gateway_adapter.go` + `ai_gateway_vision_voice.go` (Strangler Fig: dispatch gateway di Chat/AnalyzeReceipt/ProcessVoice; jalur BYOK tetap fallback); 6 test PASS; usecase regression OK |
| TASK-P03-010 | Adapter Embedder 768d Terkunci | Backend | ✅ | `embedding/embedder.go` (interface + `validateDimensions`) + `gateway.go` (`GatewayEmbedder`); legacy `GeminiEmbedder` kini validasi 768; 5 test PASS |
| TASK-P03-011 | Deprecate Endpoint API Key Lama & Pengarsipan Kunci | Backend | ✅ | handler BYOK: mutasi→410 Gone (`BYOK_DEPRECATED`), list→200 `[]`; 2 test PASS; 30 kunci lama diarsipkan (`is_archived=true`, tidak dihapus) |
| TASK-P03-012 | Frontend BYOK Cleanup & UX Polish | Frontend | ✅ | nav "API Keys" dihapus dari Sidebar & Header; halaman `/user/api-keys` → redirect `/user/settings`; `pnpm lint` 0 error; `pnpm build` OK |
| TASK-P03-013 | Integration Test Failover, Timeout & Circuit Breaker | Backend | ✅ | `gateway/integration_test.go` — 5 skenario (normal stream, failover <1s, breaker reject ke-6, timeout 100ms→failover, half-open recovery) PASS |
| TASK-P03-014 | Benchmark & Zero-Downtime Rollout Verification | Backend | ✅ | `gateway/benchmark_test.go` — ChatStream ~1.18ms/op, ChatSync ~0.97ms/op; 50 stream paralel tanpa hang; flag `feature_ai_gateway_centralized` wired (default OFF); rollout doc dibuat |

---

## Log Aktivitas Proyek

- **2026-10-11**: Inisialisasi dokumentasi perencanaan Fase P03 (AI Gateway Terpusat). Perumusan 16 keputusan teknis (K-P03-01 s/d K-P03-16), breakdown 14 tugas rinci (TASK-P03-001 s/d TASK-P03-014), serta penulisan spesifikasi teknis mendalam.
- **2026-10-13 (Contract Phase — tuntas)**: Dead code BYOK dihapus total dari backend (`ai_usecase.go` 2.183→~980 baris; `domain/gemini_key.go` dihapus) — gateway kini satu-satunya transport AI. Ditambah **bootstrap dinamis** (`internal/aigateway/bootstrap`): env menyemai `ai_endpoints`/`ai_routes`/flag hanya saat tabel kosong, DB menimpa selamanya; health checker `RunOnce` sinkron saat boot. `GatewayEmbedder` tidak lagi hardcode model (mengikuti route `embedding`). Gateway 9router **aktif dan sehat** (chat `gemini-2.5-flash`, reasoning `deepseek-r1`, embedding `gemini/gemini-embedding-001` 768d). Detail: `second_brain/3_Areas/ai-gateway/RUNBOOK_AKTIVASI_GATEWAY.md`, `teknis/routing-table.md` §5, `1_Digest/Tugas-Rutin/log_aktivitas.md` 2026-10-13.
| 2026-10-14 | TASK-P03 (Breaker) | **Tuning circuit breaker dinamis.** Threshold/cooldown breaker endpoint (`breaker_failure_threshold`/`breaker_cooldown_seconds`/`breaker_half_open_probes`) & breaker rute (`route_breaker_threshold`/`route_breaker_cooldown_secs`) dibaca dari `settings` saat boot dan diterapkan ulang seketika saat diubah via Pengaturan (hook `WithBreakerReload`). `Breaker.SetConfig`/`Config` (lock-safe) + `Service.SetBreakerConfig`/`SetRouteBreakerConfig` tanpa menghapus state. Bukti: `TestBreaker_SetConfigAppliesAtRuntime`, `TestRouteHealth_SetThresholdsAtRuntime`, `TestAdminE2E_BreakerAndDunningSettingsWritable` — PASS | Backend |
| 2026-10-14 | TASK-P03 (Metering) | **Heuristik estimasi token dinamis.** Parameter estimasi token lokal (`metering_chars_per_token`, `metering_safety_buffer_pct`, `metering_image_tokens`, `metering_audio_tokens_per_sec`) kini dibaca dari `settings` saat boot dan diaplikasikan seketika tanpa restart (hook `WithMeteringReload`). `metering.Estimator` thread-safe dengan `SetConfig`/`Config`, kompatibel penuh dengan package-level helpers. UI Super Admin `GlobalSettingsTab` menyediakan form tuning terintegrasi. Bukti: `TestEstimator_DynamicConfig`, `TestPackage_SetConfig`, `TestAdminE2E_MeteringSettingsWritable` — PASS | Fullstack |
