# Claude Code Project Guidelines — FinTrack v2

Aturan ini mencerminkan struktur AKTUAL repo (bukan `Proyek/` bawaan tool SecondBrain).

## Batas Zona
- **Kode produksi:** hanya di `frontend/`, `backend/`, `database/`. Jangan mencemari dengan script test/dump.
- **Sandbox testing:** semua script debug/test AI → `tester/` (di-`.gitignore`, tidak pernah di-commit).
- **Knowledge & logging:** dokumentasi & knowledge → `second_brain/` (di-commit); catat perubahan di `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.
- **Secret:** jangan pernah commit `.env`, secret, API key, atau dump berisi secret baru. Hanya `.env.example` yang di-commit.

## Perintah Kanonik
- `cd backend && go vet ./... && go test ./... && go build ./...`
- `cd frontend && pnpm build && pnpm lint && pnpm test`
- `database/scripts/migrate.sh`, `database/scripts/dump.sh`

## Database
- Perubahan schema → migrasi bernomor baru (`database/migrations/000N_*.sql`) → regenerate dump.
- Role aplikasi `fintrack_app` least-privilege (tanpa DDL). Otorisasi di Go, bukan RLS.

## Archival Rule
Never move projects to `5_Archives/` without explicit user confirmation.
