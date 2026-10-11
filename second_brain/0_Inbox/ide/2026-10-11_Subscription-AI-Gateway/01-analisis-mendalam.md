# 01 — Analisis Mendalam

Dokumen ini menganalisis kondisi FinTrack v2 saat ini terhadap lima pilar permintaan, dengan bukti dari
kode, serta menetapkan target desain. Setiap klaim disertai referensi file agar dapat diverifikasi.

---

## 1. Analisis Pilar 1 — API Key Terpusat (9router)

### 1.1 Kondisi saat ini

| Aspek | Temuan | Bukti |
|---|---|---|
| Model kunci | **Per-user** — kolom `user_id` wajib, `UNIQUE (user_id, alias)` | `database/migrations/0002_tables.sql` (`gemini_api_keys`) |
| Penyimpanan | Terenkripsi (`encrypted_key`), hanya `key_preview` yang tampil | `backend/internal/domain/gemini_key.go` |
| Seleksi kunci | Fallback by priority & by modality (`supports_text/audio/vision`) | `GetActiveKeysByPriority`, `GetActiveKeysByModality` |
| Endpoint provider | **Hardcoded** dalam map `providerEndpoints` (gemini, openai, groq, openrouter, qwen, deepseek, mistral, cohere, perplexity, claude) | `backend/internal/usecase/ai_usecase.go` baris ~310–335 |
| Kunci embedding | **Global server** via `GEMINI_EMBEDDING_API_KEY` (bukan per-user) | `backend/internal/infrastructure/embedding/*.go` |
| Rotasi/error | Saat gagal → `keyRepo.Update(status=error)`, coba key berikutnya | `ai_usecase.go` loop `for attempt := range len(keys)` |

### 1.2 Masalah dengan model per-user

1. **Biaya tak terkendali** — setiap user memakai kunci sendiri; tidak ada kontrol biaya platform.
2. **UX buruk** — user awam harus punya kunci LLM sendiri (berat, bukan model SaaS).
3. **Duplikasi konfigurasi** — provider & model diduplikasi di kode (map hardcoded), sulit diubah tanpa deploy.
4. **Tidak ada kuota/metering** — tidak mungkin membatasi pemakaian atau menagih.

### 1.3 Target desain

- **Pool kunci terpusat** di `ai_endpoints` (`base_url`, `api_key_encrypted`, `provider`, `priority`,
  `status`, `health`, `modalities`). Kunci dienkripsi dengan `ENCRYPTION_KEY` yang sudah ada.
- **Routing deklaratif** di `ai_routes`: `(use_case) → endpoint + model + combo`, dengan prioritas &
  fallback. Use-case: `chat`, `vision`, `voice`, `embedding`, `reasoning`.
- **9router sebagai endpoint utama**: `base_url` default `http://localhost:20128` (dapat diubah
  super admin). Karena 9router bersifat OpenAI-compatible, klien dapat disederhanakan.
- **Failover & circuit breaker**: pindah endpoint saat 5xx/timeout/429; cooldown; health check berkala;
  **kill switch** per endpoint & per use-case.
- **Embedding dikunci 768 dimensi** (cocok dengan kolom `vector(768)` yang ada). Mengganti model
  embedding non-768 = migrasi kolom + backfill, bukan perubahan diam-diam.
- **Hapus BYOK**: menu API-key user dihapus; baris lama diarsipkan (tidak di-`DELETE`) untuk audit.

### 1.4 Risiko & mitigasi

| Risiko | Mitigasi |
|---|---|
| Kunci terpusat = single point of failure | Pool multi-endpoint + failover + circuit breaker + kill switch |
| Kebocoran kunci | Enkripsi at-rest, `key_preview` saja di API, audit akses, rotasi terjadwal |
| 9router sendiri SPOF | Opsi fallback ke provider langsung via route cadangan |
| Perubahan model merusak embedding | Kunci dimensi 768; perubahan model = migrasi eksplisit |

---

## 2. Analisis Pilar 2 — Subscription & Sistem Token

### 2.1 Kondisi saat ini

- **Tidak ada** tabel plan, subscription, payment, invoice, atau token ledger (dikonfirmasi: tidak ada
  kecocokan `midtrans|subscription|plan|quota|9router` di `backend/internal`, `database/migrations`,
  `frontend/`).
- `chat_messages.tokens_used` ada, tetapi **tidak dipakai untuk apa pun** (informasi saja, tidak
  terhubung ke kuota).
- Tidak ada konsep kuota/limit per user.

### 2.2 Target desain

**Plan & entitlement**

- `plans`: `code, name, price_idr, billing_period_days, is_active, sort, gateway_override`.
- `plan_entitlements`: `(plan_id, feature_key, enabled)` — centang fitur.
- `plan_limits`: `(plan_id, limit_key, value)` — batas numerik.
- Entitlement = **fitur (bool) + limit numerik**, keduanya diatur super admin.

**Langganan**

- `subscriptions`: `user_id, plan_id, status, current_period_start/end, gateway, cancel_at_period_end`.
- Status: `active`, `past_due`, `grace`, `canceled`, `expired`, `trialing`.
- `subscription_events` append-only: `upgrade|downgrade|renew|cancel|expire|grace_start` + before→after.

**Siklus per-hari (keputusan pemilik produk)**

- Siklus = `current_period_start + billing_period_days` (bukan bulan kalender) → adil untuk bulan 28/29/30/31.
- Contoh: mulai 5 Agu → berakhir 4 Sep 23:59:59 (TZ Asia/Jakarta).
- Renewal hybrid: auto-charge bila metode mendukung; selain itu reminder H-3/H-1/H-0 + grace period
  (mis. 3 hari) + dunning.

**Ledger token (append-only, transaksional)**

- `token_ledger`: `user_id, entry_type(reserve|commit|release|grant|adjust), amount, modality, ref_type,
  ref_id, idempotency_key UNIQUE, period_id, created_at`.
- `token_periods`: proyeksi cepat `(user_id, period_start, period_end, quota, used, reserved)`.
- **Alur wajib**: pre-check → **reserve** → call → **commit** (usage nyata) / **release** (gagal).
- Saldo tidak pernah `UPDATE`; hanya entri baru (audit & anti-race).

**Harga dinamis (keputusan pemilik produk)**

- `pricing_rules`: `scope(global|plan), base_per_1k_idr, markup_multiplier, markup_percent,
  modality_multipliers jsonb`.
- Rumus: `harga = base_per_1k × (token/1000) × markup_multiplier × (1 + markup_percent/100) × modality_multiplier`.
- Uang = **integer rupiah** (tanpa float). Token = integer.

### 2.3 Kenapa "reset harian" dan bukan "bulanan"

Benar secara fairness (bulan 28 hari ≠ 31 hari). Konsekuensi: **Midtrans tidak mendukung auto-recurring
untuk semua metode** → mayoritas user (QRIS/VA) harus bayar manual → wajib ada reminder + grace + dunning.
Ini bukan penghalang, tetapi menambah komponen operasional yang harus dibangun.

### 2.4 Risiko & mitigasi

| Risiko | Mitigasi |
|---|---|
| Concurrent over-spend | Reserve/commit + idempotency key + transaksi DB |
| Kuota bocor saat retry | Idempotency key unik pada ledger |
| Downgrade menghapus data | Soft-limit, **tidak pernah** hapus transaksi user |
| Sengketa tagihan | Ledger append-only + audit log + invoice |

---

## 3. Analisis Pilar 3 — Dashboard & Manajemen Super Admin

### 3.1 Kondisi saat ini

- Halaman admin yang ada: `/admin` (statistik dasar), `/admin/users`, `/admin/categories/*`,
  `/admin/providers`. **Belum ada** manajemen plan, subscription, pembayaran, token, gateway, audit,
  atau pengaturan.
- `AdminStats` hanya: total/active user, total transaksi, total income/expense
  (`backend/internal/domain/user.go`).
- `AdminSidebar.tsx` berisi 6 menu (dashboard, kategori income/expense/savings, user, provider AI).

### 3.2 Target desain

Menu super admin baru:

1. **Dashboard** — metrik: user aktif, MRR (pendapatan berulang bulanan), pemakaian token, success rate
   pembayaran, status endpoint AI, jumlah failover, antrean review pembayaran manual.
2. **Manajemen Plan** — CRUD plan, entitlement (centang fitur), limit numerik, harga, markup, gateway override.
3. **Manajemen Subscription** — daftar langganan, status, siklus, aksi (extend, force downgrade, cancel).
4. **Manajemen Pembayaran** — verifikasi bukti transfer manual, riwayat, refund (catat, tidak otomatis).
5. **Manajemen Token** — kuota & pemakaian per user, adjust manual (dengan audit), aturan harga.
6. **AI Gateway** — kelola endpoint 9router, kunci, routing, health, kill switch.
7. **Audit Log** — semua aksi super admin & billing (siapa, kapan, sebelum→sesudah).
8. **Pengaturan** — switch payment gateway, grace period, default pricing, feature flag.

Semua tabel memakai `<DataTable>` standar (pagination 20, sorting, search, filter, penomoran, URL state).

---

## 4. Analisis Pilar 4 — RBAC Berbasis Entitlement

### 4.1 Kondisi saat ini

- `middleware/rbac.go` hanya `RequireAdmin()` dan `RequireUser()` — cek string `role == "admin"`.
- Role: `admin`, `user` (`backend/internal/domain/user.go`).
- **Tidak ada** konsep entitlement/feature flag.

### 4.2 Target desain

- **Role = identitas**: `super_admin` (operator platform) & `user` (end-user). Admin lama → `super_admin`.
- **Entitlement = hak akses**: diturunkan dari `plan_entitlements` + `plan_limits` + `subscriptions`.
  **Dipisah dari role** — ini praktik yang benar (RBAC ≠ billing).
- **Penegakan di backend** (otoritatif); frontend hanya cermin UX (sembunyikan/nonaktifkan).
- **Realtime tanpa login/logout**: entitlement **tidak** di JWT; resolver per-request membaca cache
  berversi (`ent:<userID>:v<version>`). Super admin ubah centang → bump version → cache lama gugur.
- **Efek langsung**, tanpa migrasi data user (keputusan pemilik produk).

### 4.3 Matriks awal (contoh, sepenuhnya dinamis)

| Feature/Limit | Free | Pro | Business |
|---|:--:|:--:|:--:|
| `chat` | ✅ | ✅ | ✅ |
| `vision_receipt` | ❌ | ✅ | ✅ |
| `voice_input` | ❌ | ✅ | ✅ |
| `rag_search` | ❌ | ✅ | ✅ |
| `reports_export` | ❌ | ✅ | ✅ |
| `api_access` | ❌ | ❌ | ✅ |
| `max_transactions` | 100 | 5.000 | ∞ |
| `max_ai_requests_per_day` | 10 | 200 | ∞ |
| `token_quota_per_period` | 5.000 | 100.000 | 1.000.000 |
| `max_categories` | 5 | 30 | ∞ |

### 4.4 Risiko & mitigasi

| Risiko | Mitigasi |
|---|---|
| Cache salah invalidasi → akses ilegal | Kunci berversi + TTL pendek; mutasi bump version |
| Frontend jadi otoritas | Backend selalu mengecek ulang |
| Perubahan plan mengejutkan user | Notifikasi via SSE saat entitlement berubah |

---

## 5. Analisis Pilar 5 — Standar Wajib Semua Halaman

### 5.1 Data table (pagination 20, sorting, search, filter, penomoran, URL state)

**Saat ini:** hanya `transactions` yang punya pagination server-side (default `PerPage=20`, cap 100) di
`backend/internal/repository/postgres/transaction_repo.go`. `admin/users` **tidak** berpaginasi
(mengembalikan seluruh user). Tidak ada komponen DataTable generik (`frontend/components/ui/table.tsx`
saja). Sorting/filter/search hanya sebagian.

**Target:** satu komponen `<DataTable>` di atas **TanStack Table v8**, state tersinkron ke URL
(`?page=2&sort=created_at&order=desc&q=...&status=active`), dipakai **semua** tabel. Backend
menstandarkan parameter & mengembalikan `meta{total,page,per_page}` (envelope sudah ada di
`pkg/response`).

### 5.2 Caching realtime

**Saat ini:** TanStack Query + `queryKeys` + invalidation helper sudah ada
(`frontend/lib/cache-keys.ts`, `cache-invalidation.ts`). Namun pola saat ini **meng-invalidate seluruh
grup** (mis. semua `transactions`) — belum "fetch hanya data yang berubah". Backend cache
(`middleware/cache.go`) **in-memory per-proses** → tidak sinkron lintas instance.

**Target:**
- Frontend: **surgical update** — `setQueryData` pada entri spesifik; optimistic update untuk aksi sendiri.
- Backend: **Redis** sebagai cache bersama + **pub/sub** untuk invalidasi lintas instance.
- Kunci cache berversi; data billing/entitlement tidak di-cache naif.

### 5.3 TanStack Query

Sudah terpasang (v5) di `frontend/app/providers.tsx`. Akan diperluas: devtools, error boundary, retry
policy per-domain, dan konvensi query key yang konsisten untuk entitas baru.

### 5.4 Chart interaktif

Sudah ada komponen chart (`frontend/components/charts/*`). Target: tooltip, legend interaktif, drill-down,
dan data realtime. Pastikan aksesibel (alt/aria, kontras).

### 5.5 Cursor pointer

Belum diaudit menyeluruh. Target: setiap tombol/nav/interaktif memakai `cursor-pointer` (dan elemen
non-interaktif **tidak**). Ini bagian dari audit UI/UX.

### 5.6 Realtime & zero downtime

**Saat ini:** SSE sudah dipakai untuk streaming AI chat (`frontend/hooks/useAiChat.ts` memakai
`fetch` + `getReader`). Belum ada SSE untuk event domain (pembayaran, plan, dll). Cache in-memory
membatasi skala ke satu instance.

**Target:**
- SSE per-user untuk perubahan eksternal (webhook bayar, admin ubah plan, endpoint AI down) dengan
  auth cookie, `Last-Event-ID` (resume), heartbeat, reconnect.
- Redis pub/sub sebagai jembatan lintas instance.
- Zero downtime: migrasi **expand → migrate → contract**, rolling deploy, feature flag, health gate.

### 5.7 Risiko & mitigasi

| Risiko | Mitigasi |
|---|---|
| URL state kacau (deep link rusak) | Satu konvensi param + parser bersama |
| Invalidate berlebihan | Kunci granular per-entitas + surgical update |
| Reconnect SSE hilang event | `Last-Event-ID` + replay dari cache/DB |
| Migrasi memutus sesi berjalan | Expand/contract + kolom nullable dulu |

---

## 6. Kesimpulan analisis

Semua lima pilar dapat dibangun **tanpa rewrite**, dengan memanfaatkan fondasi yang sudah kuat
(auth Go-native, envelope response, TanStack Query, SSE chat, optimistic locking, migrasi bernomor).
Komponen baru dibangun sebagai **modul bersih ber-interface** (`aigateway`, `billing`, `entitlement`,
`payment`, `realtime`, `audit`) lalu diadaptasi oleh kode lama. Prioritas dibagi ke 7 fase P02–P08
(lihat `05-peta-jalan-awal.md`).
