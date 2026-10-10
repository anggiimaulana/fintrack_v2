# Workspace Rules — FinTrack v2 (Antigravity / Gemini)

Aturan ini mencerminkan struktur AKTUAL repo ini (bukan `Proyek/` bawaan tool SecondBrain).

## Zona & Batas
1. **Kode produksi hanya di** `frontend/`, `backend/`, `database/`. Jangan menaruh script test, dump, atau file scratch di sini.
2. **Semua file scratch/testing AI → `tester/`**. Folder ini di-`.gitignore` dan tidak pernah di-commit.
3. **Dokumentasi & knowledge → `second_brain/`** (PARA). Folder ini justru di-commit.
4. **Jangan pernah commit** `.env`, secret, API key, atau dump berisi secret baru. `.env.example` boleh.

## Perintah Kanonik
- Backend: `cd backend && go vet ./... && go test ./... && go build ./...`
- Frontend: `cd frontend && pnpm build && pnpm lint && pnpm test`
- Database: `database/scripts/migrate.sh` (apply migrasi), `database/scripts/dump.sh` (regenerate dump)

## Audit Trail
Setiap aksi teknis signifikan dicatat di `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`. Progress HUD diperbarui di `second_brain/2_Projects/P01_FinTrack_v2/roadmap.md`.

## Database
- Schema berubah = **migrasi bernomor baru** (`database/migrations/000N_*.sql`), lalu dump di-regenerate.
- Role aplikasi `fintrack_app` adalah least-privilege (tanpa DDL). Otorisasi di Go, bukan RLS.

## Archival
Jangan memindahkan proyek ke `5_Archives/` tanpa konfirmasi eksplisit user.
