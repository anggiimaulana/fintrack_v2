# Log Aktivitas

> Audit trail operasional. Setiap aksi teknis signifikan dicatat di sini (urutan terbaru di atas).

---

## 2026-10-11 — P07 Slice B: Realtime SSE + perbaikan 3 cacat invalidasi (1 bug koneksi)

Slice kedua P07. Membangun transport SSE dari nol dan menutup tiga cacat invalidasi
entitlement yang membuat perubahan paket baru terasa setelah 24 jam.

### Perubahan
- **Modul `backend/internal/realtime/`** (baru): `hub.go` (registry koneksi, batas 5/user,
  eviction terlama, cleanup saat ctx cancel), `buffer.go` (ring-buffer Redis ZSET
  `realtime:buf:{id}`, TTL 10 menit), `publisher.go` (+`redis_sub`), `handler.go`.
- **Kontrak event** di `pkg/contracts/events.go` — paket daun, tanpa impor internal, agar
  producer di modul mana pun bisa memancarkan event tanpa siklus impor.
- **Endpoint** `GET /api/v1/realtime/stream` (di luar grup `cached`); auth memakai
  `AuthMiddleware` yang sudah ada (cookie `ft_at`), bukan mekanisme token baru.
- **Fan-out Redis**: channel `realtime:u:{userID}` + `realtime:admins`; `SubscribePattern`
  ditambahkan ke pubsub. Tanpa Redis, publish jatuh ke hub lokal.
- **Lima producer**: `payment_received`, `manual_approval_needed`, `plan_updated`,
  `quota_reset`, `ai_endpoint_down`. Hook `breaker.Config.OnStateChange` baru, di-invoke
  **setelah** lock breaker dilepas.
- **Frontend**: `lib/realtime-dispatcher.ts`, `hooks/use-realtime-stream.ts`,
  `providers/realtime-provider.tsx`. Backoff manual 1s→16s + jitter (opsi A), `last_event_id`
  sebagai query param karena `EventSource` yang dibuat ulang tidak mengirim header otomatis.
- **Valve** `REALTIME_ENABLED` (default on).

### Bug yang ditemukan & diperbaiki
1. **Header SSE tidak di-flush saat connect.** `c.Status(200)` tidak mengirim header sampai
   penulisan pertama, jadi klien menunggu sampai heartbeat pertama — **15 detik** — sebelum
   melihat stream terbuka; `onopen` di browser juga tidak menyala. Ditemukan dari durasi test
   integrasi yang kelipatan tepat 15s. Setelah `flusher.Flush()`: **91.7s → 2.9s**.
2. **Invalidasi entitlement dari jalur admin memakai scope cache yang salah** (`"u:"` vs
   `"ent:"`), sehingga perubahan paket per-user tidak invalid.
3. **Mengubah fitur/limit sebuah plan tidak menginvalidasi apa pun** → seluruh subscriber plan
   memakai entitlement basi sampai 24 jam. Diperbaiki dengan komponen versi plan pada kunci
   cache: satu bump `ent:plan:{id}` menginvalidasi semua subscriber, O(1).
4. **`entBroadcaster` tidak pernah dipanggil** di `main.go` → tidak ada invalidasi lintas-instance.

### Verifikasi
- Backend: `go vet ./...` bersih, `go test ./...` hijau, `go build ./...` hijau.
- Integrasi: `tests/realtime_e2e_test.go` — 6 skenario lewat router + Redis nyata (401 tanpa
  token, 503 saat disabled, delivery event, isolasi antar user, event admin tidak bocor ke user
  biasa, resume backlog via `Last-Event-ID`).
- **Uji mutasi** pada perbaikan 4b: pemeriksaan versi plan dilepas → suite gagal
  ("still sees stale entitlement after plan edit") → terbukti menangkap regresi.
- Unit test baru: hub (isolasi, eviction, cleanup, slow-client), breaker hook (transisi,
  **anti-deadlock** — callback memanggil `State()` di dalam lock akan hang), invalidasi plan.
- Frontend: `pnpm lint` 0 error (62 warning = baseline), `pnpm build` hijau, `pnpm test`
  81 lolos (dari 66).

### Catatan
- **Gap:** `quota_reset` tidak terjadwal. Tidak ada worker rotasi siklus di codebase (cleaner
  hanya melepas reservasi kedaluwarsa); event dipancarkan saat periode baru benar-benar dibuat,
  yaitu ketika user berikutnya memakai AI. Siklus terjadwal = pekerjaan lanjutan.
- Tidak ada inbox notifikasi durabel: SSE akselerator UX, bukan sumber kebenaran.
- `LedgerService` ternyata **tidak dikonstruksi di `main.go`** (hanya dipakai test/cleaner);
  jalur produksi yang memanggil `Reserve` belum ada. Dicatat sebagai temuan, bukan dikerjakan.

---

## 2026-10-11 — P07 Slice A: Rollout entitlement & RBAC + 3 bug ditemukan

Slice pertama P07 (Rollout RBAC/Realtime). Hasil sampingan penting: implementasi ini menemukan
**tiga bug nyata**, dua di antaranya lubang keamanan.

### Perubahan
- **Enforcement dinyalakan.** `NewEntitlementMiddleware` sebelumnya **tidak pernah dikonstruksi**
  (0 dari 63 rute ter-gate; hanya `RequireUser`, jadi autentikasi tanpa otorisasi). Kini
  dikonstruksi di `cmd/server/main.go`, disuntikkan via `RouterConfig`, dan dipasang per-rute.
  Rute ter-gate: `POST /transactions` (`max_transactions`), `POST /ai/chat` (`ai_chat` +
  `ai_tokens_per_cycle`), `POST /ai/receipt` (`ocr_receipt` + `ocr_scans_per_cycle`),
  `POST /ai/receipt/transaction` (`ocr_receipt`), `POST /ai/voice` (`voice_input`),
  `GET /reports` (`reports_export`). Rute baca dibiarkan agar cache 30s tidak berubah.
- **Fallback plan default.** `GetActivePlanForUser` kini jatuh ke plan `DEFAULT_PLAN_CODE`
  (default `free`) bila user tidak punya baris `subscriptions`, dengan sentinel
  `billing.StatusNone`. Fail-closed tetap bila plan default hilang.
- **Usage resolver baru** (`middleware/usage_resolvers.go`): `max_transactions` (non-soft-deleted),
  `ai_tokens_per_cycle` (`used + reserved`), `ocr_scans_per_cycle` (ledger `modality='vision'`).
  Ditambah `GetCurrentPeriod` read-only (non-locking) di `billing_repo` — `GetPeriodForUpdate`
  yang ada memakai row-lock, tidak cocok untuk pre-flight check.
- **Kontrak blok** 403 `feature_not_in_plan` / 402 `limit_exceeded`+`limit_not_in_plan`, keduanya
  memuat `upgrade_url`. Valve `ENTITLEMENT_ENFORCEMENT` (default on) untuk rollback insiden.
- **Mirror frontend (advisory-only):** `hooks/use-entitlements.ts`, `hooks/use-role.ts`,
  `lib/entitlement-errors.ts`, halaman read-only `/user/settings/subscription`, tombol
  OCR/export dinonaktifkan sesuai paket. `vitest.config.ts` diperluas ke `hooks/**`.

### Bug yang ditemukan & diperbaiki
1. **Lockout ~96% user (blocker).** Resolver mengembalikan deny-all untuk user tanpa baris
   `subscriptions`; DB dev punya 52 user vs 2 subscription. Menyalakan enforcement tanpa
   fallback akan meng-403/402 hampir semua user. → fallback plan `free`.
2. **`super_admin` terkunci dari rute user.** `RequireUser()` hanya menerima `user`/`admin`,
   sehingga pemilik platform ditolak 403 di transaksi, dashboard, AI, dan laporan. → menerima
   ketiga role, sejalan dengan `RequireAdmin()`.
3. **Langganan kedaluwarsa tetap dapat fitur berbayar.** Query subscription tidak memfilter
   `status`, jadi baris `expired`/`canceled` terus memberi plan Pro/Business selamanya — eskalasi
   hak akses begitu enforcement menyala. → `canceled`/`expired` jatuh ke plan default.

### Verifikasi
- Backend: `go vet ./...` bersih, `go test ./...` hijau, `go build ./...` hijau.
- Integrasi: `tests/entitlement_e2e_test.go` — 8 skenario lewat **router nyata** (Free 403 di
  `/reports`, Pro 200, limit 100 → 402, soft-deleted tidak dihitung, periode token tidak ada →
  diizinkan, tanpa subscription → diperlakukan Free, langganan expired → Free, super_admin bypass,
  `upgrade_url` ada). Ini penjaga kelas bug "middleware ada tapi tidak ter-wiring".
- **Uji mutasi:** gate `reports_export` dilepas sementara → suite gagal (200 bukan 403), lalu
  dikembalikan → hijau. Suite terbukti menangkap regresi, bukan lolos karena kebetulan.
- Frontend: `pnpm lint` 0 error (62 warning, sama seperti baseline), `pnpm build` hijau
  (rute `/user/settings/subscription` terdaftar), `pnpm test` 66 lolos (dari 54).
- `fintrack_test` tanpa seed, jadi suite integrasi menyediakan katalog plan sendiri (idempoten).

### Dokumen dikoreksi
- `teknis/rbac-rollout.md`: matriks "28 endpoint" diganti matriks rute nyata; 429 → 402; nama
  kunci nyata (`max_transactions`); perilaku tanpa subscription; daftar kunci yang belum ditegakkan.
- `planning/tasks.md` + `planning/implementation-plan.md`: AC 429 → 402; status Slice A.
- `tracking/tracking.md`, `teknis/frontend-entitlement-mirror.md`: status implementasi.
- Spec & plan: `docs/superpowers/specs/2026-10-11-p07-entitlement-rollout-design.md`,
  `docs/superpowers/plans/2026-10-11-p07-entitlement-rollout-plan.md`.

### Catatan
- Belum ditegakkan (butuh pekerjaan lanjutan): `ai_requests_per_cycle` (belum ada penghitung
  request), `max_categories` (kategori dikelola admin), `max_wallets` (fitur belum ada),
  `rag_search` (menyatu di `/ai/chat`), `advanced_budgeting`, `api_access`, `bank_sync`.
- User dengan transaksi sudah di atas batas Free (mis. demo 479 vs 100) ditolak pada
  `POST /transactions`; rute baca tetap terbuka. Enforcement yang benar, bukan bug.

---

## 2026-10-11 — P06-020: Suite E2E integrasi super admin + perbaikan race review pembayaran

Penutupan P06-020 ("E2E Validation"). Hasil sampingan penting: suite ini **menemukan bug
data-integrity nyata** pada jalur persetujuan pembayaran manual, yang lalu diperbaiki.

### Perubahan
- **`backend/tests/admin_e2e_test.go` (baru)** — suite integrasi 6 skenario memakai HTTP asli
  (httptest) di atas middleware produksi (`AuthMiddleware` + `RequireSuperAdmin`), handler
  `internal/admin`, repository Postgres, dan DB nyata. Env-gated `TEST_DATABASE_URL` sehingga
  `go test ./...` tanpa infrastruktur tetap hijau (skip).
  1. Otorisasi: anon 401, user biasa 403, klaim `super_admin` palsu 403 (re-check peran hidup, K-14), demosi saat token masih valid 403.
  2. Pembuatan plan + inspeksi jejak audit (aktor, `entity=plan`, `after.code`).
  3. Race review pembayaran (8 approve bersamaan → tepat 1×200, sisanya 409).
  4. Penyesuaian token idempoten (replay `idempotency_key` → `replayed=true`, kuota tidak dobel, 1 baris ledger).
  5. Kill switch AI Gateway (persistensi ke `settings`, target tidak valid 400).
  6. Pengaturan global + feature flags (allowlist key, `updated_by`, validasi rollout 0–100).
- **Perbaikan race (3 berkas sumber + 1 fake test)**:
  - `payment/service.go` — `GetPaymentByIDForUpdate` ditambahkan ke interface `Repository`.
  - `repository/postgres/payment_repo.go` — implementasi `SELECT ... FOR UPDATE`.
  - `payment/manual_service.go` — `review()` kini memakai varian `ForUpdate` di dalam transaksi.
  - `payment/webhook_test.go` — `memRepo` memenuhi method baru.

### Temuan bug (bukti)
- Sebelum perbaikan: 8 approve bersamaan pada satu pembayaran → `[200 200 200 409 200 409 409 409]`,
  dan tanpa penjadwalan acak bisa `[200 ×8]`. Akar masalah: `review()` membaca dengan
  `GetPaymentByID` (tanpa lock); di bawah READ COMMITTED semua transaksi melihat `pending_review`
  dan semuanya menulis `settled` → event/audit ganda untuk satu pembayaran.
- Jalur webhook sudah punya lock (`GetPaymentByGatewayRefForUpdate`, dibuktikan
  `TestPayment_ConcurrentWebhookIdempotent`, 20 webhook bersamaan); jalur admin tidak pernah diberi.
- Sesuai mitigasi yang diminta `P06/planning/sprint-plan.md` (risk register: "pessimistic row-lock
  `SELECT ... FOR UPDATE`") dan `implementation-plan.md` ("cegah race condition antar-admin").

### Verifikasi
- `go vet ./...` bersih; `go test ./...` hijau; `go build ./...` hijau.
- `TEST_DATABASE_URL=... go test -count=30 -run TestAdminE2E_PaymentReviewRace ./tests/...` → hijau (90 burst).
- Uji mutasi: lock dilepas sementara → suite gagal di ronde pertama (`[200 ×8]`), lalu dikembalikan → hijau. Tes terbukti menangkap regresi, bukan lolos karena keberuntungan.
- `TEST_DATABASE_URL=... go test -run TestAdminE2E ./tests/...` → 6/6 hijau.
- **Prasyarat**: `fintrack_test` perlu dimigrasi penuh dulu (sebelumnya hanya tabel dari `0002`; kolom
  `super_admin` belum ada). Diterapkan `0001–0010` idempoten via `psql` (dokumentasi: `database/README.md`).
- **Bukan regresi**: `TestLedger_ReserveCommitRelease` gagal karena `fintrack_test` tanpa seed
  (`pricing_rules` kosong). Diverifikasi gagal juga saat perubahan ini di-`git stash` — kondisi pra-ada.

---

## 2026-10-11 — P06: Bersih-bersih lint + hapus `setState` di dalam `useEffect`

Lanjutan penutupan P06 (fitur sudah 20/20). Fokus: menghilangkan pola sinkronisasi state
anti-pattern dan menyisakan 0 error lint.

### Perubahan
- **Hook baru `frontend/hooks/use-reset-on-open.ts`** — reset form dialog saat `open` berubah
  `false → true` memakai pola React "adjust state during render" (tanpa render pass tambahan);
  param `signature` opsional untuk reset saat record yang disunting berganti.
  Dipakai di `ForceDowngradeDialog`, `CancelSubscriptionDialog`, `PaymentActionDialogs`
  (approve/reject/refund), `FeatureFlagDialog`.
- **Adopsi nilai server saat render** (bukan effect) di `AuditFilters` (4 nilai filter),
  `GlobalSettingsTab` (hidrasi form) dan `RoutingMatrixPanel` (matriks rute). Perbandingan
  memakai signature nilai, sehingga refetch latar tidak lagi menghapus suntingan lokal.
- Impor/ekspor mati dibuang: `AreaChartView`, 6 ikon lucide di `GatewayPanels`, `CardHeader/
  CardTitle/CardDescription` di `FeatureFlagsTab`, `AlertCircle`, tipe `TokenLedgerRow`,
  deklarasi `variant` di `JsonDiffViewer`; kutip ganda pada teks JSX di-escape (`&quot;`).

### Verifikasi
- `pnpm lint` → **0 error** (3 error sebelumnya: `react-hooks/set-state-in-effect` ×2,
  `prefer-const`), 62 warning sisa (mayoritas `no-unused-vars` di berkas di luar P06).
- `pnpm build` hijau; `pnpm test` → 4 berkas / 54 tes lolos.
- Browser (Next build lokal di port 3001 + backend/DB docker, login super admin):
  - Reset dialog: extend (30 hari/'' → terisi → buka ulang kembali 30/''), feature flag
    (key/nama/rollout kembali ke nilai server), paksa turunkan paket (paket default +
    alasan kosong, tombol submit `disabled`), refund (3 field kosong).
  - Matriks rute: sunting prioritas → tombol "Simpan Matriks" aktif; "Segarkan Data" →
    suntingan **tetap** (dulu terhapus); "Batalkan perubahan" → kembali ke nilai server & tombol nonaktif.
  - Filter audit: ketik email → URL `?filter_email=...` setelah debounce; "Reset Filter" → input & URL bersih.
  - Pengaturan: muat ulang halaman → form terhidrasi dari server, tombol "Simpan Pengaturan" `disabled`.

### Catatan / deviasi kecil
- Tidak ada tes unit baru untuk hook: `vitest` proyek berjalan di environment `node` tanpa
  `@testing-library/react`/jsdom; menambah dependensi hanya untuk 15 baris hook dinilai tidak
  sepadan. Perilaku dibuktikan lewat walkthrough browser di atas.


---

## 2026-10-11 — Implementasi P06 Super Admin Dashboard SELESAI (20/20, 100%)

Eksekusi `P06_Super-Admin-Dashboard/planning/implementation-plan.md` (4 milestone). Semua task ✅
setelah dibuktikan lewat suite API + walkthrough browser.

### Backend — Otorisasi, Admin API, Kill Switch
- **TASK-P06-001** ✅ middleware `super_admin.go` (+test) + grup `/api/v1/admin/*`; anon 401, user 403 (dibuktikan di smoke).
- **TASK-P06-005/008/010/012/014/018** ✅ paket `internal/admin` (12 berkas): metrics (overview, timeseries, payment-methods), plans CRUD + entitlement, subscriptions (extend/cancel/force-downgrade), payments (review/settle/refund), tokens (users/ledger/adjust 409 `NO_ACTIVE_PERIOD`), audit-logs (filter + sanitasi PDP).
- **TASK-P06-016** ✅ aigateway admin (endpoints/routes/kill-switch) + migrasi `0010_settings.sql`; enforcement kill switch di `gateway/service.go`, `breaker/circuit_breaker.go`, `embedding/killswitch.go`.
- **TASK-P06-006/013/015/017/019** ✅ feature flag repository + test; `pkg/response` dan `ai_usecase` diselaraskan.

### Frontend — Dashboard Admin
- **TASK-P06-002/003/004** ✅ layout + guard role di `proxy.ts`, sidebar, `<DataTable>` TanStack v8 + URL state (Suspense boundary ditambahkan — sebelumnya `pnpm build` gagal).
- **TASK-P06-006/007/009/011/013/015/017/019/020** ✅ 8 halaman `(dashboard)/super-admin/*` (dashboard + Recharts, plans, subscriptions, payments + review, tokens, ai-gateway, audit viewer + diff JSON, settings + feature flag).

### Verifikasi (bukti nyata)
- `bash tester/p06_smoke.sh` → seluruh endpoint admin 200; negatif sesuai kontrak: 400 `target tidak valid`, 401 anon, 403 user, 409 `NO_ACTIVE_PERIOD`.
- Browser (super admin): 8 halaman render, **0 error konsol**.
- Kill switch lewat UI: klik switch global → dialog "Ya, Aktifkan Kill Switch" → `data-state=checked`, banner "Layanan Global Dimatikan" muncul, API `global=true`, audit `entity_id=ai_kill_switch after={"enabled":true,"target":"global"}`; toggle OFF langsung tanpa dialog → API semua `false` (state dikembalikan).
- RBAC: user biasa membuka `/super-admin` → dialihkan ke `/user`.
- `cd frontend && pnpm build` hijau.

### Catatan / deviasi kecil
- Tombol "Simpan Matriks" hanya untuk perubahan matriks rute; toggle kill switch tersimpan otomatis setelah dialog konfirmasi (dibuktikan: tombol tetap `disabled` saat toggle berubah, namun API sudah terbarui).

---

## 2026-10-11 — Implementasi P05 Payments & Billing SELESAI (17/17, 100%)

Eksekusi `P05_Payments-Billing/planning/implementation-plan.md` (5 wave). Semua task ✅ setelah
dibuktikan test nyata.

### W1 — Skema, Gateway, Midtrans, Webhook
- **TASK-P05-001** ✅ migrasi `0009_payments_invoices.sql` (invoices, payments, settings, invoice_counters + seed); CHECK amount>0 terbukti; replay OK.
- **TASK-P05-002** ✅ `payment/gateway.go` (interface Gateway + models; uang int64).
- **TASK-P05-003** ✅ `payment/midtrans.go` (Snap create + basic auth, verify SHA-512 constant-time, check status, cancel); 7 test PASS.
- **TASK-P05-004** ✅ `handler/payment_handler.go` webhook (MaxBytes 1MB).
- **TASK-P05-005** ✅ `payment/webhook.go` (pull verification + row-lock idempoten); replay 5x → aktivasi 1x.

### W2 — Invoice, Manual, Review
- **TASK-P05-006** ✅ invoice_service + NextInvoiceNumber (INV/YYYYMMDD/NNNNN atomic upsert).
- **TASK-P05-007** ✅ manual.go (transfer instructions + magic-byte proof validation 5MB) + manual_service upload.
- **TASK-P05-008** ✅ approve/reject (row-lock idempoten, reason wajib) + AdminService. 6 test PASS.

### W3 — Resolver, Prorata, Downgrade, Tax
- **TASK-P05-009** ✅ registry.ResolveGateway (per-plan > global > manual) + checkout locks gateway; 4 test PASS.
- **TASK-P05-010** ✅ prorata (billing) + invoice via checkout.
- **TASK-P05-011** ✅ downgrade schedule/apply (soft-limit).
- **TASK-P05-012** ✅ tax.go TaxCalculator (settings rate, floor).

### W4 — Dunning, Grace, SSE, Audit
- **TASK-P05-013** ✅ dunning_worker (H-3/H-1/H-0 + dedup).
- **TASK-P05-014** ✅ grace_worker (grace → past_due soft-limit).
- **TASK-P05-015** ✅ realtime_publisher (user_events:{uid}, payment.settled).
- **TASK-P05-016** ✅ audit_interceptor (named financial events). 6 test PASS.

### W5 — E2E & Concurrency
- **TASK-P05-017** ✅ **Anti double-credit terbukti:** 20 webhook paralel order yang sama → **tepat 1 transisi & 1 aktivasi** (19 idempoten); invoice `paid`, payment `settled`. Nomor invoice 20× unik.
- Wiring: payment handler + routes (webhook publik + admin manual approve/reject/pending), registry, services, dunning/grace workers, `SubscriptionActivator` (bridge ke billing + invalidate entitlement), `SettingsRepo`.

### Verifikasi akhir
- `go vet`/`go build`/`go test ./...` hijau (22 paket ok).
- Backend rebuild; webhook endpoint terdaftar.

**Catatan deviasi:** `-race` tidak tersedia (tanpa gcc). Integrasi Midtrans diuji dengan mock Snap server (kredensial produksi belum ada).

---

## 2026-10-11 — Implementasi P04 Subscription & Token Core SELESAI (17/17, 100%)

Eksekusi `P04_Subscription-Token-Core/planning/implementation-plan.md` (5 wave). Semua task ✅
setelah dibuktikan test nyata.

### W1 — Skema, Seeder, Domain, Repositori
- **TASK-P04-001** ✅ migrasi `0008_subscription_token_core.sql` (8 tabel; trigger append-only `token_ledger`; constraint & idempotency unique terbukti; replay OK).
- **TASK-P04-002** ✅ `seeds/0002_subscription_token_seeds.sql` (3 plan, 24 entitlement, 18 limit, pricing; idempoten).
- **TASK-P04-003** ✅ `internal/billing` domain + interface repo.
- **TASK-P04-004** ✅ `repository/postgres/billing_repo.go` (row-lock FOR UPDATE, idempotency); 2 test PASS.

### W2 — Token Ledger 2-Fase
- **TASK-P04-005/006** ✅ `billing/ledger.go` Reserve/Commit/Release (row-lock, idempoten, release sisa reserve).
- **TASK-P04-007** ✅ `billing/cleaner.go` auto-release reservation basi.
- **TASK-P04-008** ✅ `billing/pricing.go` (integer IDR, floor). 10 test PASS.

### W3 — Pricing, Entitlement, Middleware
- **TASK-P04-009/010/011** ✅ `entitlement` resolver + cache berversi L1/L2 + broadcaster pub/sub.
- **TASK-P04-012** ✅ middleware `RequireFeature` (403) & `WithinLimit` (402), super_admin bypass, fail-closed.
- **TASK-P04-013** ✅ `GET /api/v1/me/entitlements`. 12 test PASS.

### W4 — Subscription Lifecycle
- **TASK-P04-014** ✅ `billing/lifecycle.go` state machine + audit event.
- **TASK-P04-015** ✅ `billing/subscription_management.go` prorata (integer floor) + downgrade soft-limit. 5 test PASS.

### W5 — Testing & Metrics
- **TASK-P04-016** ✅ **Anti double-spend terbukti:** 40 reserve paralel pada kuota 1000 → tepat 10 sukses, 30 ditolak, `reserved=1000` (tanpa over-spend); 10 commit ganda → `used=1000`, `reserved=0`, tanpa saldo negatif.
- **TASK-P04-017** ✅ Metrik Prometheus (`token_ledger_ops_total`, `token_quota_used`, `entitlement_resolve_total`) + handoff kontrak P05/P06.

### Verifikasi akhir
- `go vet`/`go build`/`go test ./...` hijau (semua paket ok; 21 paket).
- Backend rebuild; endpoint `/api/v1/me/entitlements` tersedia.

**Catatan deviasi:** `-race` tidak tersedia (tanpa gcc); uji race harus dijalankan di CI. Uji konkurensi dijalankan tanpa race detector namun dengan verifikasi invarian DB (reserved ≤ quota, tidak negatif).

---

## 2026-10-11 — Implementasi P03 AI Gateway Terpusat SELESAI (14/14, 100%)

Eksekusi `second_brain/2_Projects/P03_AI-Gateway-Terpusat/planning/implementation-plan.md` (5 wave).
Semua task ✅ hanya setelah dibuktikan test nyata.

### W1 — Skema, Domain, Repositori
- **TASK-P03-001** ✅ migrasi `0007_ai_gateway.sql`: `ai_endpoints` + `ai_routes` (+constraint, index, trigger), kolom arsip `gemini_api_keys`, seed 9router-primary + 5 route. Replay idempoten.
- **TASK-P03-002** ✅ `internal/domain/aigateway.go`: entity, DTO, `UseCase`, `TokenUsage`, error sentinel, interface.
- **TASK-P03-003** ✅ `repository/postgres/ai_gateway_repo.go`: load routes (join+priority), cache TTL 60s, `DecryptKey`. 3 test PASS vs DB.

### W2 — Client, Breaker, Router, Health, Metering
- **TASK-P03-004** ✅ `aigateway/client` (OpenAI-compatible: SSE parser, chat sync/stream, embedding, audio multipart). 6 test PASS.
- **TASK-P03-005** ✅ `aigateway/breaker` (closed/open/half_open + clock injection). 7 test PASS.
- **TASK-P03-006** ✅ `aigateway/gateway/service.go` (failover, kill switch, breaker gate, embedding 768 lock). 6 test PASS.
- **TASK-P03-007** ✅ `aigateway/gateway/health.go` (worker probe). 4 test PASS.
- **TASK-P03-008** ✅ `aigateway/metering` (provider + heuristic fallback). 8 test PASS.

### W3 — Adapter, Embedder
- **TASK-P03-009** ✅ Adapter Strangler Fig (`ai_gateway_adapter.go`, `ai_gateway_vision_voice.go`); dispatch gateway di Chat/AnalyzeReceipt/ProcessVoice; jalur BYOK tetap fallback. 6 test PASS.
- **TASK-P03-010** ✅ `embedding.Embedder` interface + `GatewayEmbedder` + validasi 768. 5 test PASS.

### W4 — Deprecate BYOK & Frontend
- **TASK-P03-011** ✅ handler BYOK: mutasi → 410 Gone, list → 200 `[]`; 30 kunci lama diarsipkan. 2 test PASS.
- **TASK-P03-012** ✅ nav API Keys dihapus; `/user/api-keys` → redirect `/user/settings`. `pnpm lint` 0 error; build OK.

### W5 — Integrasi & Benchmark
- **TASK-P03-013** ✅ `gateway/integration_test.go` — 5 skenario PASS (normal, failover <1s, breaker reject ke-6, timeout, half-open recovery).
- **TASK-P03-014** ✅ `gateway/benchmark_test.go` — ChatStream ~1.18ms/op, ChatSync ~0.97ms/op; 50 stream paralel tanpa hang; flag `feature_ai_gateway_centralized` wired (default OFF).

### Verifikasi akhir (live)
- `go vet`/`go build`/`go test ./...` hijau (semua paket ok).
- Backend rebuild & restart: `/health` 200; log "AI gateway centralized: disabled (feature flag off)".
- Flag ON diuji: chat lewat gateway → log mencoba `localhost:20128/v1/chat/completions`, gagal (9router mati) → user menerima pesan ramah (tanpa crash). Flag dikembalikan OFF.

**Catatan deviasi:** `-race` tidak tersedia (tanpa gcc); uji race harus dijalankan di CI. Benchmark dijalankan lokal dengan mock upstream.

---

## 2026-10-11 — Implementasi P02 Fondasi Platform SELESAI (14/14, 100%)

Eksekusi `second_brain/2_Projects/P02_Fondasi-Platform/planning/implementation-plan.md` dalam 5 wave.
Semua task ditandai ✅ hanya setelah dibuktikan test nyata (tanpa klaim palsu).

### W1 — Infrastruktur Redis
- **TASK-P02-001** ✅ Redis 7-alpine di `docker-compose.yml` (host port **6380**; 6379 dipakai stack lain), `maxmemory 256mb` + `allkeys-lru`, healthcheck `redis-cli ping` → healthy; `depends_on: redis: healthy` di backend.
- **TASK-P02-002** ✅ `internal/infrastructure/redis` (client/config/errors) dengan graceful degradation; di-wire di `main.go` + `defer Close()`. Test 4 PASS (EmptyURL/InvalidURL/Unreachable/Live).

### W2 — Cache berversi, Pub/Sub, Observability
- **TASK-P02-003** ✅ `redis/cache.go` (CacheEngine berversi + fallback memory) + rewrite `middleware/cache.go` (key `cache:u:{uid}:v{ver}:g{gver}:{hash}`); invalidasi O(1) via version bump. Test HIT/MISS/version-bump/isolasi-user/live-redis PASS.
- **TASK-P02-004** ✅ `redis/pubsub.go` (channel `cache:invalidate`); publish saat invalidate + subscriber di main (stop on shutdown). Test disabled + live PASS.
- **TASK-P02-013** ✅ `internal/infrastructure/metrics` (four golden signals + counter bisnis); `/metrics` wired. Test PASS; **live `/metrics` 200**.
- **TASK-P02-005** ✅ migrasi `0005_create_audit_logs.sql`; trigger append-only menolak UPDATE & DELETE (terbukti via psql); replay idempoten.

### W3 — Audit & Feature Flag
- **TASK-P02-006** ✅ `internal/audit` (Service async + worker batch + masker + repo pgx); wired di main. Test 9 PASS (non-blocking <10ms, flush-on-close, batching, masking); integrasi live DB PASS.
- **TASK-P02-007** ✅ migrasi `0006_create_feature_flags.sql`; unique key menolak duplikat; replay idempoten.
- **TASK-P02-008** ✅ `internal/featureflag` (evaluator fail-closed, rollout deterministik, cache Redis/memory, repo pgx). Test 8 PASS.

### W4 — DataTable & URL State
- **TASK-P02-009** ✅ `pkg/pagination` (parser + envelope); clamp per_page 10-100, whitelist sort anti-injection. Test 9 PASS.
- **TASK-P02-010** ✅ `lib/table-params.ts` (murni) + `hooks/use-data-table-url-state.ts`. Test 13 PASS.
- **TASK-P02-011** ✅ `components/data-table/*` (TanStack Table v8.21.3, server-side, empty/error/skeleton).
- **TASK-P02-012** ✅ toolbar (search debounce 300ms + filter chips) & pagination (10/20/50/100, numbering, cursor-pointer). `pnpm lint` **0 error**; `pnpm build` OK; `pnpm test` 21 PASS.

### W5 — Verifikasi
- **TASK-P02-014** ✅ `internal/integration/foundation_test.go` (4 skenario: cache versioning, audit persist+mask, feature flag, pagination) — PASS terhadap Redis+PostgreSQL live.
- **Verifikasi akhir:** `go vet`/`go build`/`go test ./...` hijau; `pnpm lint` 0 error/build/test hijau; **smoke test live**: `/health` 200, `/metrics` 200, login 200 + cookie, `X-Cache: HIT` (cache berversi Redis aktif).

**Catatan deviasi:** Redis host port = **6380** (6379 dipakai stack lain di mesin dev). TanStack Table dipin ke **v8** (bukan v9) sesuai plan. `psql` tidak di PATH → migrasi diterapkan via `docker exec`.

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
- `0_Inbox/ide/2026-10-11_Subscription-AI-Gateway/` (8 file: ringkasan, analisis, kritik & saran best-practice, keputusan, pertanyaan terbuka, peta jalan, **jawaban+rekomendasi**, **konfigurasi seed awal**).
- `4_Resources/Dokumen-Spesifikasi/` — BRD, PRD, dan Diagram (Use Case, Flowchart, Sequence, Activity, ERD).
- `2_Projects/` — 7 fase baru: P02 Fondasi Platform, P03 AI Gateway Terpusat, P04 Subscription & Token Core, P05 Payments & Billing, P06 Super Admin Dashboard, P07 Rollout RBAC/Realtime, P08 Hardening & Go-Live (masing-masing `analisis/planning/rules/teknis/tracking`).
- `3_Areas/` — diisi 5 area lama + tambah `billing-pembayaran/` & `ai-gateway/` (kebijakan + runbook).
- README indeks diperbarui: `0_Inbox`, `2_Projects`, `3_Areas`, `4_Resources`.

### Kritik & saran (diterima semua)
Uang integer; ledger transaksional; blast radius kunci terpusat; kualifikasi "realtime"; cache invalidation;
downgrade tidak hapus data; jangan percaya callback klien; PCI-DSS SAQ-A; konvensi prorata eksplisit;
audit append-only; UU PDP No. 27/2022; zero downtime realistis; observability; YAGNI.

### Rencana implementasi (writing-plans)
- `2_Projects/IMPLEMENTATION-PLAN.md` — master plan P02–P08 (prinsip, aturan migrasi, alokasi nomor migrasi 0005–0012, DoD, gate kualitas, risiko).
- Plan detail semua fase: `P0X_*/planning/implementation-plan.md` (P02: 196 baris, P03: 350, P04: 403, P05: 403, P06: 537, P07: 366, P08: 555). Masing-masing: tujuan & DoD, wave, detail task (target file, langkah, verifikasi, bukti), matriks bukti, perintah verifikasi, rollback, handoff.
- **Koreksi status:** `tracking.md` fase P02/P04/P06/P08 yang keliru menandai "100% selesai" & memuat hasil uji fiktif → diperbaiki menjadi ⬜ belum dimulai (P02 sprint-plan checkbox juga di-uncheck).
- **Konsistensi:** nomor migrasi diseragamkan ke alokasi kanonik master plan (0007 ai_gateway, 0008 subscription_token_core, 0009 payments_invoices, 0010 settings, 0011 archive_byok); path `frontend/src/` → `frontend/`; `backend/database/migrations/` → `database/migrations/`.

### Langkah berikutnya
Eksekusi P02 dimulai setelah konfirmasi pemilik produk; fase P03–P08 mendapat plan detail saat dimulai.

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
