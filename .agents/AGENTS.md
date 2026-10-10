# AGENTS.md — FinTrack v2

Lihat aturan lengkap di `.agents/rules/workspace_rules.md`.

Ringkas:
- Kode produksi hanya di `frontend/`, `backend/`, `database/`.
- Semua file scratch/testing AI → `tester/` (gitignored, tidak pernah commit).
- Dokumentasi & knowledge → `second_brain/` (commit); log di `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.
- Jangan pernah commit `.env`/secret/dump berisi secret baru; hanya `.env.example`.
- Perintah kanonik: `cd backend && go test ./...`, `cd frontend && pnpm build`.
- Schema DB berubah = migrasi bernomor baru + regenerate dump.
