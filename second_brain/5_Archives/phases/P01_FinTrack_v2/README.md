# P01 — FinTrack v2

## Context
Migrasi stack FinTrack dari Go + Supabase ke Go + PostgreSQL murni + Next.js dalam repo baru `D:/project/finance/fintrack_v2`. Repo lama & DB lama utuh.

## Scope of Work
Membangun monorepo baru (`frontend/`, `backend/`, `database/`, `second_brain/`, `tester/`) dengan:
- DB schema final replayable (tanpa RLS, tanpa `auth.*`), migrasi bernomor, dump di-commit.
- Backend Go: auth Go-native (bcrypt + JWT + refresh rotation), otorisasi di Go, context propagation penuh, SPK race-free, RAG aktif (pgvector + Gemini embedding), cache per-user, optimistic locking.
- Migrasi penuh data lama (user, transaksi, kategori, SPK, chat, API key terenkripsi).
- Frontend Next.js: proxy same-origin, session via jose, audit UI/UX menyeluruh.
- Docker compose lokal + struktur siap prod.

## Keputusan Terkunci
Lihat `second_brain/0_Inbox/brief_awal.md` (7 keputusan terkunci).

## Deliverables
1. `database/` — schema final + migrasi + dump + scripts.
2. `backend/` — Go API (build/vet/test hijau).
3. `frontend/` — Next.js (build/lint/test hijau, tanpa Supabase).
4. `docker-compose.yml` + Dockerfiles + `.env.example`.
5. `database/migration_from_supabase/` + `verify_output.txt`.

## Referensi
- Plan lengkap: `local://fintrack-v2-migration-plan.md` (durable copy).
- Roadmap & HUD: `roadmap.md`.
- Skenario pengujian: `skenario_pengujian.md`.
