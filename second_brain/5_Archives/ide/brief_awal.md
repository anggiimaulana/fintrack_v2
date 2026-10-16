# Brief Awal — FinTrack v2

## Ringkasan
Migrasi penuh FinTrack dari stack **Go + Supabase (Auth, PostgREST/RLS, CLI lokal PG17) + Next.js** ke stack **Go + PostgreSQL murni + Next.js** dalam **repo baru** `D:/project/finance/fintrack_v2`.

Repo lama `D:/project/finance/fintrack` **tidak disentuh**; DB lama (volume `supabase_db_fintrack`) **tidak dihapus** dan tetap sebagai backup.

## Struktur monorepo target
```
fintrack_v2/
├── frontend/      # Next.js (UI neo-brutalist, Inter + JetBrains Mono)
├── backend/       # Go API (auth Go-native, otorisasi di Go + DB role least-privilege)
├── database/      # schema final + migrasi bernomor + dump + scripts
├── second_brain/  # knowledge base & audit trail (PARA)
└── tester/        # sandbox testing AI (tidak pernah di-commit)
```

## Motivasi
- RAG maksimal (embedding Gemini `text-embedding-004` 768d + pgvector HNSW + backfill).
- Bebas race condition (SPK queue + advisory lock + transaksi DB).
- Ketahanan & keamanan DB optimal (tanpa RLS, role `fintrack_app` least-privilege).
- Performa optimal (index valid, pool terkonfigurasi, query paralel, cache per-user).
- Audit UI/UX menyeluruh (neo-brutalist konsisten, a11y, responsif, dark mode).

## Keputusan terkunci (disetujui user)
1. Auth Go-native: bcrypt cost 12 + JWT HS256 access 15 menit + refresh token rotasi di DB; cookie httpOnly; semua trafik browser lewat proxy same-origin Next.js.
2. RLS dihapus; otorisasi di Go + DB role least-privilege `fintrack_app` (tanpa DDL).
3. `database/` = dump final + migrasi bernomor tetap dipelihara; dump lengkap di-commit ke git.
4. Migrasi semua data: user, profil, transaksi, kategori, SPK, chat, API key terenkripsi — `ENCRYPTION_KEY` lama dipertahankan.
5. RAG: key embedding global server + Gemini `text-embedding-004` (768d) + backfill.
6. UI/UX: audit & perbaikan menyeluruh, neo-brutalist konsisten, Inter + JetBrains Mono.
   > **Revisi 2026-10-15 (ADR-011):** border radius kini **aktif** (`--radius-*` ≠ 0).
   > Ciri brutalist yang dipertahankan: border tebal 2px + hard shadow `Npx Npx 0`.
7. Deploy: lokal dulu (docker compose), struktur siap jalur prod.

## Pointer
- Plan lengkap eksekusi: lihat roadmap & dokumen plan migrasi (path durable `local://fintrack-v2-migration-plan.md`).
- Proyek aktif: `second_brain/2_Projects/P01_FinTrack_v2/`.
- Log aktivitas: `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.
