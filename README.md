# FinTrack v2

Migrasi FinTrack dari **Go + Supabase** ke **Go + PostgreSQL murni + Next.js** dalam satu monorepo.

> Repo lama (`../fintrack`) dan DB lama (volume `supabase_db_fintrack`) **tidak disentuh** dan tetap sebagai backup.

## Layout Monorepo

```
fintrack_v2/
├── frontend/      # Next.js (App Router) — UI neo-brutalist, Inter + JetBrains Mono
├── backend/       # Go API — auth Go-native, otorisasi di Go, RAG (pgvector + Gemini)
├── database/      # schema final + migrasi bernomor + dump + scripts
├── second_brain/  # knowledge base & audit trail (PARA) — di-commit
├── tester/        # sandbox testing AI — gitignored, tidak pernah di-commit
├── docker-compose.yml
└── .env.example
```

| Zona | Isi | Commit? |
|------|-----|---------|
| `frontend/`, `backend/`, `database/` | Kode & artefak produksi | ✅ |
| `second_brain/` | Dokumentasi, roadmap, log, keputusan | ✅ |
| `tester/` | Script scratch, dump, hasil k6 | ❌ (gitignored) |

## Arsitektur Singkat

- **Auth**: Go-native — bcrypt cost 12 + JWT HS256 (access 15 menit) + refresh token rotasi di DB. Cookie `httpOnly` (`ft_at`, `ft_rt`). Semua trafik browser lewat proxy same-origin Next.js.
- **Otorisasi**: di Go + DB role least-privilege `fintrack_app` (tanpa DDL). Tanpa RLS, tanpa `auth.*` Supabase.
- **Database**: PostgreSQL 17 + `pgvector` (embedding 768d, HNSW). Migrasi bernomor replayable; dump final di-commit.
- **RAG**: `text-embedding-004` (Gemini REST) + pencarian vektor cosine; backfill via `cmd/backfill`.
- **Resilience**: context propagation penuh, SPK queue (dedupe + advisory lock), optimistic locking (409), cache per-user.

## Cara Start (Docker Compose)

```bash
cp .env.example .env          # isi APP_DB_PASSWORD, JWT_SECRET, ENCRYPTION_KEY, GEMINI_EMBEDDING_API_KEY
docker compose up --build -d  # db (5435) + backend (8080) + frontend (3001)
curl -i localhost:8080/health # 200
open http://localhost:3001    # UI
```

## Cara Migrasi Database

Lihat detail di [`database/README.md`](database/README.md). Ringkas:

```bash
# 1) DB via compose
docker compose up -d db

# 2) Apply migrasi (idempoten) + seed demo
APP_DB_PASSWORD=<password> database/scripts/migrate.sh --seed

# 3) Regenerate dump final
database/scripts/dump.sh
```

Aturan: **schema berubah = migrasi bernomor baru** (`database/migrations/000N_*.sql`), lalu dump di-regenerate.

## Development Lokal

```bash
# Backend
cd backend && go vet ./... && go test ./... && go build ./...

# Frontend
cd frontend && pnpm install && pnpm build && pnpm lint && pnpm test
```

## Pointer

- Database: [`database/README.md`](database/README.md)
- Second Brain: [`second_brain/README.md`](second_brain/README.md)
- Roadmap & HUD: [`second_brain/2_Projects/P01_FinTrack_v2/roadmap.md`](second_brain/2_Projects/P01_FinTrack_v2/roadmap.md)
- Skenario pengujian: [`second_brain/2_Projects/P01_FinTrack_v2/skenario_pengujian.md`](second_brain/2_Projects/P01_FinTrack_v2/skenario_pengujian.md)

## Aturan Kontribusi

Lihat `.agents/rules/workspace_rules.md`, `CLAUDE.md`, `.cursorrules`, `.windsurfrules` (isi sama).
