# Log Aktivitas

> Audit trail operasional. Setiap aksi teknis signifikan dicatat di sini (urutan terbaru di atas).

---

## 2026-10-11 — Brainstorming & Analisis: Platform Berlangganan + AI Gateway Terpusat (P02–P08)

Ringkasan: sesi brainstorming (skill `brainstorming`, jalur **architectural**) atas 5 pilar permintaan pemilik
produk — (1) kunci AI terpusat via 9router, (2) subscription & sistem token, (3) dashboard/manajemen super
admin, (4) RBAC berbasis entitlement, (5) standar data-table/caching realtime/zero-downtime. Belum ada
kode yang ditulis; seluruh keluaran berupa dokumen.

### Keputusan terkunci (K-01..K-30)
- Tenancy per-user (B2C); role hanya `super_admin` + `user` (admin lama dipromosikan).
- Token: semua fitur AI; harga dinamis (base per 1k + markup kali/persen + pengali modality).
- Siklus **per-hari** (TZ Asia/Jakarta), reset tiap siklus, tanpa rollover; renewal **hybrid**.
- Upgrade **instan prorated**; downgrade **akhir siklus**, soft-limit (tanpa hapus data).
- Switch payment gateway: global + override per-plan (berlaku checkout baru; in-flight tetap gateway asalnya).
- Entitlement = fitur (bool) + limit numerik; penegakan **backend**; realtime via **cache berversi** (bukan JWT).
- AI gateway: pool multi-endpoint 9router + failover + circuit breaker + kill switch; embedding dikunci 768d.
- Ledger token **append-only** (reserve→commit/release) + idempotency; uang = **integer rupiah**.
- Infra: **tambah Redis** (cache bersama + pub/sub). DataTable: TanStack Table v8 + URL state.
- Zero downtime: migrasi **expand/contract** + rolling deploy + feature flag.
- Pendekatan: **hibrida A+B** (incremental Strangler Fig + modul baru bersih ber-interface).

### Dokumen yang ditulis
- `0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/` (6 file: ringkasan, analisis, kritik & saran best-practice, keputusan, pertanyaan terbuka, peta jalan).
- `4_Resources/Dokumen-Spesifikasi/` — BRD, PRD, dan Diagram (Use Case, Flowchart, Sequence, Activity, ERD).
- `2_Projects/` — 7 fase baru: P02 Fondasi Platform, P03 AI Gateway Terpusat, P04 Subscription & Token Core, P05 Payments & Billing, P06 Super Admin Dashboard, P07 Rollout RBAC/Realtime, P08 Hardening & Go-Live (masing-masing `analisis/planning/rules/teknis/tracking`).
- `3_Areas/` — diisi 5 area lama + tambah `billing-pembayaran/` & `ai-gateway/` (kebijakan + runbook).
- README indeks diperbarui: `0_Inbox`, `2_Projects`, `3_Areas`, `4_Resources`.

### Kritik & saran (diterima semua)
Uang integer; ledger transaksional; blast radius kunci terpusat; kualifikasi "realtime"; cache invalidation;
downgrade tidak hapus data; jangan percaya callback klien; PCI-DSS SAQ-A; konvensi prorata eksplisit;
audit append-only; UU PDP No. 27/2022; zero downtime realistis; observability; YAGNI.

### Langkah berikutnya
Spec self-review → review oleh pemilik produk → transisi ke rencana implementasi (writing-plans).

---

## 2026-10-11 — Eksekusi Migrasi FinTrack v2 SELESAI (Plan `fintrack-v2-migration`)

Ringkasan: repo baru `D:/project/finance/fintrack_v2` dibangun dari nol (Go + PostgreSQL murni + Next.js), data lama dimigrasi penuh, seluruh verifikasi hijau. Repo lama & volume Supabase lama **tidak disentuh**.

### Fase 0 — Scaffold monorepo & Second Brain ✅
- Monorepo `frontend/ backend/ database/ second_brain/ tester/`, `git init` (main).
- Skill SecondBrain global + struktur PARA + aturan multi-IDE + README.

### Fase 1 — Database (schema final) ✅
- Migrasi bernomor idempoten: `0001_extensions` (vector), `0002_tables` (13 tabel, tanpa RLS/`auth.*`), `0003_indexes`, `0004_roles` (`fintrack_app` least-privilege, DML-only).
- `seeds/0001_demo.sql` (4 user bcrypt cost 12, kategori, 390 tx, SPK, chat) — idempoten.
- Scripts: migrate/dump/restore/backup/verify (autodetect psql Windows). DB `fintrack_test` untuk integrasi.
- Catatan: jumlah tabel = **13** (plan menulis "14"; `ai_conversations` sengaja tidak dibuat sebagai dead code, jadi daftar eksplisit plan = 13).

### Fase 2 — Backend (Go) ✅
- Modul → `github.com/fintrack/backend`; deps Supabase/genai dibuang (`go mod tidy`).
- Context dipropagasi ke seluruh layer (0 `context.Background()` di `internal/`).
- Auth Go-native: bcrypt cost 12 + JWT HS256 (15m) + refresh token rotasi (deteksi reuse → revoke semua); cookie `ft_at`/`ft_rt` httpOnly; `OriginCheck` menggantikan CSRF no-op.
- Repo `users` baru; perbaikan kontrak admin (`PUT /admin/users/:id/status`, `GET /admin/subcategories?category_id=`, `GET /admin/users` = array).
- SPK: queue dedupe + debounce + advisory lock + self-heal `GetLatest`.
- RAG: embedder Gemini REST (`text-embedding-004`) + `embedding_repo` + retrieval di `Chat` + `cmd/backfill`.
- Cache per-user + invalidasi prefix nyata; optimistic lock (409); dashboard errgroup.
- Test: unit (password/token/spk_queue/embedding text/cache) + integrasi env-gated (auth reuse, isolasi, optimistic lock, SPK race). `go vet/test/build` hijau.

### Fase 3 — Migrasi data dari Supabase lama ✅
- `supabase start` (port 54322); data lama: 9 user, 479 tx, 30 key, 58 SPK, 33 sesi, 114 pesan.
- `01_extract.sh` → 12 CSV; `02_transform.sql` → staging + map `auth.users`+`profiles` → `users`.
- `03_verify.sql` + `verify_output.txt`: count & `sum(amount)` per user **identik**.
- Login hash migrasi (`budi`, `admin`) → 200 + cookie HttpOnly. `supabase stop` (volume utuh).

### Fase 4 — Frontend (Next.js) ✅
- Copy `apps/web` → `frontend`; buang `@supabase/*`, `react-dropzone`, `zustand`; tambah `jose`, `next-themes`, `vitest`.
- `lib/session.ts`, `lib/auth-client.ts`, `lib/format.ts`, `lib/session-context.tsx`; `proxy.ts` (jose, guard role); `lib/api.ts` single-flight refresh; `next.config.ts` rewrites `/api → backend`.
- UI/UX: font Inter + JetBrains Mono; `Money` component (font-mono); token warna (hapus hex hardcoded di landing/auth); konsolidasi RupiahInput; EmptyState/ErrorState; a11y (aria-label, alt, label, focus-visible); dark mode toggle; media query via `useSyncExternalStore`.
- Hapus semua `: any` (0 lint error); `pnpm build` + `pnpm lint` + `pnpm test` (8 test) hijau; 0 referensi `supabase`.

### Fase 5 — Compose & env root ✅
- `docker-compose.yml` 3 service (db 5435 / backend 8080 / frontend 3001) + healthcheck.
- Dockerfile backend (`golang:1.27-alpine`) & frontend (`node:22` + pnpm 11).
- Rate limit overridable via env (untuk load test). `proxy.ts` matcher mengecualikan `/api`.
- k6: p95 23.6ms, checks 100%, exit 0 (hasil di `tester/k6-result.txt`).

### Fase 6 — Finalisasi ✅
- Dump final (`database/dump/fintrack_v2.{dump,sql}`, 260KB) + `verify_output.txt` di-commit.
- Backup drill: restore ke DB scratch → count identik → drop.
- Commit bertahap per fase.

### Catatan / deviasi kecil
- `GEMINI_EMBEDDING_API_KEY` tidak tersedia → RAG dijalankan dalam mode disabled-graceful; plumbing diuji dengan embedding sintetis (cosine 1.0) dan `cmd/backfill` memverifikasi kebutuhan key. Jalankan `cd backend && go run ./cmd/backfill` setelah key diisi untuk backfill penuh.
- Rate limit login dinaikkan via env khusus untuk profil k6 (default produksi tetap 5/menit).
