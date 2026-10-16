# Roadmap — P01 FinTrack v2

```text
[████████████████████] 100% COMPLETED
Active Milestone : FASE 6 — FINALISASI (selesai)
System Status    : DB ✅ | Backend ✅ | Frontend ✅ | Compose ✅ | Data Migration ✅ | Tester Verified ✅
```

## Definition of Done (DoD)
- [x] DB final replayable dari nol (`fintrack_test`) → 13 tabel, 3 constraint kondisional.
- [x] Backend `go vet && go build && go test` hijau; 0 `context.Background()` di `internal/`.
- [x] Integrasi auth + isolasi user + optimistic lock + SPK ≤2 baris hijau.
- [x] Data lama tervalidasi identik (count & `sum(amount)` per user).
- [x] Login kredensial migrasi (budi/admin) → 200 + cookie HttpOnly.
- [x] RAG plumbing aktif (embedder REST + repo + chat retrieval + backfill); runtime penuh butuh `GEMINI_EMBEDDING_API_KEY`.
- [x] Cache per-user benar (tidak bocor); race SPK terkendali (growth 1).
- [x] k6 p95 23.6ms, checks 100%.
- [x] Backup drill (dump → restore → count sama).
- [x] Frontend build/lint/test hijau; 0 `supabase` di `app/components/hooks/lib`.
- [x] UI walkthrough (token/font/a11y/dark mode/responsif).
- [x] Compose end-to-end (health 200, login browser, rewrite same-origin).

## Milestones

### FASE 0 — Scaffold monorepo & Second Brain ✅
- [x] 0.1 Monorepo dirs + `git init` (main)
- [x] 0.2 Root `.gitignore` + `tester/.gitkeep`
- [x] 0.3 SecondBrain skill + scaffold PARA
- [x] 0.4 Multi-IDE rules
- [x] 0.5 Root `README.md`

### FASE 1 — Database (schema final baru) ✅
- [x] 1.1 `0001_extensions.sql`
- [x] 1.2 `0002_tables.sql`
- [x] 1.3 `0003_indexes.sql`
- [x] 1.4 `0004_roles.sql`
- [x] 1.5 `seeds/0001_demo.sql`
- [x] 1.6 `scripts/` (migrate/dump/restore/backup/verify)
- [x] 1.7 `database/README.md`
- [x] 1.8 Compose `db` + migrate `--seed` + verify
- [x] 1.9 `fintrack_test` + migrate

### FASE 2 — Backend (Go) ✅
- [x] 2.1 Copy `apps/api` → `backend`, rename module, build
- [x] 2.2 Hapus dead code + `go mod tidy`
- [x] 2.3 Propagasi context (0 `context.Background()`)
- [x] 2.4 Pool config dari env
- [x] 2.5 Auth Go-native (password/token/refresh/usecase/handler/middleware/origin/router/main)
- [x] 2.6 Perbaikan kontrak admin
- [x] 2.7 SPK race fix (queue + advisory lock + self-heal)
- [x] 2.8 Aktivasi RAG (embedding REST + repo + chat + backfill)
- [x] 2.9 Cache fix (per-user + invalidation nyata)
- [x] 2.10 Optimistic locking (409)
- [x] 2.11 Dashboard errgroup paralel
- [x] 2.12 Test (unit + integrasi)
- [x] 2.13 `backend/.env.example`
- [x] 2.14 `go vet/test/build` + smoke `/health`

### FASE 3 — Migrasi data dari Supabase lama ✅
- [x] 3.1 `supabase start` + verifikasi data
- [x] 3.2 `01_extract.sh`
- [x] 3.3 `02_transform.sql`
- [x] 3.4 `03_verify.sql` + `verify_output.txt`
- [x] 3.5 Backfill embeddings
- [x] 3.6 Uji login hash migrasi
- [x] 3.7 `supabase stop` (volume utuh)

### FASE 4 — Frontend (Next.js) ✅
- [x] 4.1 Copy `apps/web` → `frontend`, update deps
- [x] 4.2 Hapus lib Supabase, tambah session/auth-client/format
- [x] 4.3 `proxy.ts` (jose)
- [x] 4.4 `lib/api.ts` single-flight refresh
- [x] 4.5 `next.config.ts` rewrites
- [x] 4.6 Ganti semua pemakaian Supabase
- [x] 4.7 `frontend/.env.example`
- [x] 4.8 Audit UI/UX (font, token, komponen, a11y, responsif, state, dark mode)
- [x] 4.9 Vitest + test
- [x] 4.10 `pnpm build && pnpm lint && pnpm test`

### FASE 5 — Compose & env root ✅
- [x] 5.1 `docker-compose.yml` (db/backend/frontend)
- [x] 5.2 `.env.example` root
- [x] 5.3 Dockerfiles
- [x] 5.4 `docker compose up --build -d` + verify

### FASE 6 — Finalisasi ✅
- [x] 6.1 Dump final + commit
- [x] 6.2 Update roadmap 100% + log aktivitas
- [x] 6.3 Commit bertahap
