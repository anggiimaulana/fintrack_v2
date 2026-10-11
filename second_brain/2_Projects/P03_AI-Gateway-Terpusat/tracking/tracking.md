# Tracking Status — P03 AI Gateway Terpusat

Status tracking tugas implementasi Fase P03.
- Format status: ⬜ (Belum Dimulai) | 🔄 (Sedang Dikerjakan) | ✅ (Selesai / Terverifikasi)

> **Status saat ini: BERJALAN.** W1 (skema, domain, repo) selesai. Status ✅ hanya diberikan setelah dibuktikan test nyata.

---

## Tabel Pelacakan Tugas (TASK-P03-NNN)

| Task ID | Nama Tugas | Assignee | Status | Keterangan / Hasil Verifikasi |
|---|---|---|---|---|
| TASK-P03-001 | Skema Database & Migrasi `0007_ai_gateway.sql` | Backend | ✅ | Diterapkan ke DB; `ai_endpoints`/`ai_routes` + constraint + index + seed (1 endpoint, 5 route); kolom arsip `gemini_api_keys`; replay idempoten OK |
| TASK-P03-002 | Domain Modeling & Interface Gateway | Backend | ✅ | `internal/domain/aigateway.go` (entity, DTO, `UseCase`, `TokenUsage`, error sentinel, interface `AIGatewayService`/`CircuitBreaker`/`HealthChecker`); `go build ./internal/domain/...` OK |
| TASK-P03-003 | Repository SQL Endpoint & Route Cache | Backend | ✅ | `repository/postgres/ai_gateway_repo.go` (load routes join+priority, cache TTL 60s, list/get/update health, `DecryptKey`); 3 test PASS vs DB live |
| TASK-P03-004 | OpenAI-Compatible HTTP Client Engine | Backend | ⬜ | HTTP client standard, SSE stream reader, usage extractor |
| TASK-P03-005 | State Machine Circuit Breaker | Backend | ⬜ | Closed/Open/Half-Open state, 5 error threshold, 30s cooldown |
| TASK-P03-006 | Router Multi-Endpoint & Failover Engine | Backend | ⬜ | Priority fallback chain, kill switch check, routing logic |
| TASK-P03-007 | Active & Passive Health Checker Worker | Backend | ⬜ | Goroutine worker 60s, timeout 5s, update status DB & memory |
| TASK-P03-008 | Metering Token Extractor & Heuristic Fallback | Backend | ⬜ | Ekstraksi field usage, fallback karakter/4 tokenizer |
| TASK-P03-009 | Adapter Tipis `AIUsecase` | Backend | ⬜ | Refactor `ai_usecase.go` chat, vision, voice tanpa rewrite RAG |
| TASK-P03-010 | Adapter Embedder 768d Terkunci | Backend | ⬜ | Integrasi embedder ke gateway, strict validation vector len = 768 |
| TASK-P03-011 | Deprecate Endpoint API Key Lama & Pengarsipan | Backend | ⬜ | Return 410 Gone pada user API key, soft-archive tabel DB |
| TASK-P03-012 | Frontend BYOK Cleanup & UX Polish | Frontend | ⬜ | Hapus menu API key user, pembersihan modal dan pesan error |
| TASK-P03-013 | Integration Test Failover & Circuit Breaker | QA/Backend | ⬜ | Skenario failover otomatis, test transisi circuit breaker |
| TASK-P03-014 | Benchmark & Zero-Downtime Rollout Verification | Team | ⬜ | Load testing SSE, zero downtime deployment via feature flag |

---

## Log Aktivitas Proyek

- **2026-10-11**: Inisialisasi dokumentasi perencanaan Fase P03 (AI Gateway Terpusat). Perumusan 16 keputusan teknis (K-P03-01 s/d K-P03-16), breakdown 14 tugas rinci (TASK-P03-001 s/d TASK-P03-014), serta penulisan spesifikasi teknis mendalam.
