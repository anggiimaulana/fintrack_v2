# Roadmap — P01 FinTrack v2

```text
[░░░░░░░░░░░░░░░░░░░░] 0% COMPLETED
Active Milestone : FASE 0 — SCAFFOLD MONOREPO & SECOND BRAIN
System Status    : DB ⏳ | Backend ⏳ | Frontend ⏳ | Compose ⏳ | Data Migration ⏳ | Tester Verified ⏳
```

## Definition of Done (DoD)
- [ ] DB final replayable dari nol (`fintrack_test`) → 14 tabel, 3 constraint kondisional.
- [ ] Backend `go vet && go build && go test` hijau; 0 `context.Background()` di `internal/`.
- [ ] Integrasi auth + isolasi user + optimistic lock + SPK ≤2 baris hijau.
- [ ] Data lama tervalidasi identik (count & `sum(amount)` per user).
- [ ] Login kredensial migrasi (budi/admin) → 200 + cookie HttpOnly.
- [ ] RAG hidup (embedding bertambah + chat memakai transaksi relevan).
- [ ] Cache per-user benar (tidak bocor); race SPK terkendali.
- [ ] k6 p95 < 300ms, checks > 99%.
- [ ] Backup drill (dump → restore → count sama).
- [ ] Frontend build/lint/test hijau; 0 `supabase` di `app/components/hooks/lib`.
- [ ] UI walkthrough 375/768/1440 + dark mode OK.
- [ ] Compose end-to-end (health 200, login browser, chat SSE streaming).

## Milestones

### FASE 0 — Scaffold monorepo & Second Brain
- [ ] 0.1 Monorepo dirs + `git init` (main)
- [ ] 0.2 Root `.gitignore` + `tester/.gitkeep`
- [ ] 0.3 SecondBrain skill + scaffold PARA
- [ ] 0.4 Multi-IDE rules
- [ ] 0.5 Root `README.md`

### FASE 1 — Database (schema final baru)
- [ ] 1.1 `0001_extensions.sql`
- [ ] 1.2 `0002_tables.sql`
- [ ] 1.3 `0003_indexes.sql`
- [ ] 1.4 `0004_roles.sql`
- [ ] 1.5 `seeds/0001_demo.sql`
- [ ] 1.6 `scripts/` (migrate/dump/restore/backup/verify)
- [ ] 1.7 `database/README.md`
- [ ] 1.8 Compose `db` + migrate `--seed` + verify (14 tabel / 4 user)
- [ ] 1.9 `fintrack_test` + migrate

### FASE 2 — Backend (Go)
- [ ] 2.1 Copy `apps/api` → `backend`, rename module, build
- [ ] 2.2 Hapus dead code + `go mod tidy`
- [ ] 2.3 Propagasi context (0 `context.Background()`)
- [ ] 2.4 Pool config dari env
- [ ] 2.5 Auth Go-native (password/token/refresh/usecase/handler/middleware/origin/router/main)
- [ ] 2.6 Perbaikan kontrak admin
- [ ] 2.7 SPK race fix (queue + advisory lock + self-heal)
- [ ] 2.8 Aktivasi RAG (embedding REST + repo + chat + backfill)
- [ ] 2.9 Cache fix (per-user + invalidation nyata)
- [ ] 2.10 Optimistic locking (409)
- [ ] 2.11 Dashboard errgroup paralel
- [ ] 2.12 Test (unit + integrasi)
- [ ] 2.13 `backend/.env.example`
- [ ] 2.14 `go vet/test/build` + smoke `/health`

### FASE 3 — Migrasi data dari Supabase lama
- [ ] 3.1 `supabase start` + verifikasi data
- [ ] 3.2 `01_extract.sh`
- [ ] 3.3 `02_transform.sql`
- [ ] 3.4 `03_verify.sql` + `verify_output.txt`
- [ ] 3.5 Backfill embedding
- [ ] 3.6 Uji login hash migrasi
- [ ] 3.7 `supabase stop` (volume utuh)

### FASE 4 — Frontend (Next.js)
- [ ] 4.1 Copy `apps/web` → `frontend`, update deps
- [ ] 4.2 Hapus lib Supabase, tambah session/auth-client/format
- [ ] 4.3 `proxy.ts` (jose)
- [ ] 4.4 `lib/api.ts` single-flight refresh
- [ ] 4.5 `next.config.ts` rewrites
- [ ] 4.6 Ganti semua pemakaian Supabase
- [ ] 4.7 `frontend/.env.example`
- [ ] 4.8 Audit UI/UX (font, token, komponen, a11y, responsif, state, dark mode)
- [ ] 4.9 Vitest + test
- [ ] 4.10 `pnpm build && pnpm lint && pnpm test` + walkthrough

### FASE 5 — Compose & env root
- [ ] 5.1 `docker-compose.yml` (db/backend/frontend)
- [ ] 5.2 `.env.example` root
- [ ] 5.3 Dockerfiles
- [ ] 5.4 `docker compose up --build -d` + verify

### FASE 6 — Finalisasi
- [ ] 6.1 Dump final + commit
- [ ] 6.2 Update roadmap 100% + log aktivitas
- [ ] 6.3 Commit bertahap
