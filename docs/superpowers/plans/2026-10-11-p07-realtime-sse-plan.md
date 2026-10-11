# Implementation Plan — P07 Slice B: Realtime SSE & Cache Invalidation

| Field | Isi |
|---|---|
| **Fase** | P07 — Rollout RBAC, Realtime & Standar (**Slice B** dari 5) |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | Slice A selesai (commit `9500e4b`) |
| **Spec desain** | `docs/superpowers/specs/2026-10-11-p07-realtime-sse-design.md` |
| **Task terkait** | TASK-P07-004 … TASK-P07-009 + perbaikan cacat invalidasi |
| **Status** | Siap dieksekusi |

---

## 1. Tujuan & Definition of Done

Menghadirkan transport realtime SSE berstandar produksi (hub + fan-out Redis + resiliensi),
memancarkan lima event domain dari producer nyata, memasang surgical cache invalidation di
frontend, dan menutup tiga cacat invalidasi entitlement.

**Definition of Done:**
- [ ] Modul `internal/realtime/` lengkap: hub, client, buffer, publisher, subscriber, handler.
- [ ] `GET /api/v1/realtime/stream` terdaftar, auth via cookie `ft_at`, header SSE benar.
- [ ] Heartbeat `: ping` tiap 15s; resume via `Last-Event-ID` (header atau query param).
- [ ] Lima event dipancarkan dari producer nyata.
- [ ] Cacat 4a/4b/4c diperbaiki (invalidasi paket per-user, invalidasi plan, lintas-instance).
- [ ] Frontend: dispatcher + hook + provider; surgical invalidation tanpa key liar.
- [ ] Degradasi tanpa Redis aman; valve `REALTIME_ENABLED` berfungsi.
- [ ] `go vet`/`go test`/`go build` hijau; `pnpm lint`/`build`/`test` hijau.
- [ ] Dokumentasi dikoreksi; tracking + log diperbarui.

---

## 2. Urutan Eksekusi

| Wave | Langkah | Deskripsi | Prasyarat |
|---|---|---|---|
| **W1** | B-1 | Kontrak event di `pkg/contracts/events.go` | — |
| **W1** | B-2 | `realtime/hub.go` + `client.go` (registry, batas koneksi, cleanup) | B-1 |
| **W1** | B-3 | `realtime/buffer.go` (ring-buffer Redis, TTL 10m) | B-1 |
| **W1** | B-4 | `realtime/publisher.go` + `redis_sub.go` | B-2 |
| **W1** | B-5 | `realtime/handler.go` + route + valve `REALTIME_ENABLED` | B-4 |
| **W2** | B-6 | Perbaikan 4b: komponen versi plan pada kunci cache resolver | — |
| **W2** | B-7 | Perbaikan 4a/4c: admin memakai `entBroadcaster` | B-6 |
| **W2** | B-8 | Perbaikan 4b lanjutan: invalidasi saat plan diedit | B-7 |
| **W3** | B-9 | Producer `payment_received` (webhook + approve manual) | B-5 |
| **W3** | B-10 | Producer `manual_approval_needed` | B-5 |
| **W3** | B-11 | Producer `plan_updated` | B-7 |
| **W3** | B-12 | Producer `quota_reset` (periode baru dibuat) | B-5 |
| **W3** | B-13 | Producer `ai_endpoint_down` (hook breaker) | B-5 |
| **W4** | B-14 | Test unit backend | B-2…B-13 |
| **W4** | B-15 | Test integrasi (router + Redis nyata) | B-14 |
| **W5** | B-16 | `lib/realtime-dispatcher.ts` + test | B-5 |
| **W5** | B-17 | `hooks/use-realtime-stream.ts` (backoff) + test | B-16 |
| **W5** | B-18 | Provider + mount | B-17 |
| **W5** | B-19 | `pnpm lint`/`build`/`test` | B-18 |
| **W6** | B-20 | Koreksi dokumentasi + tracking + log | B-15, B-19 |
| **W6** | B-21 | Commit | B-20 |

**Jalur kritis:** B-1 → B-2 → B-4 → B-5 → B-15 (transport harus hidup sebelum producer diuji).

---

## 3. Detail Langkah

### W1 — Transport inti

#### B-1 — Kontrak event
- **Target:** `backend/pkg/contracts/events.go`
- **Langkah:** `Event{Event, Timestamp, ID, Payload}`; konstanta `payment_received`,
  `plan_updated`, `ai_endpoint_down`, `manual_approval_needed`, `quota_reset`; struct payload
  per event. Diletakkan di `pkg/contracts` (paket daun) agar consumer tidak perlu mengimpor
  `internal/realtime` (menghindari siklus).
- **Verifikasi:** unit test marshal/unmarshal envelope.

#### B-2 — Hub & client
- **Target:** `backend/internal/realtime/hub.go`, `client.go`
- **Langkah:** registry `map[uuid.UUID]map[*client]struct{}` + `map[*client]struct{}` untuk admin;
  mutex; `Register`/`Unregister`/`SendToUser`/`SendToAdmins`; buffered channel per klien;
  **batas 5 koneksi/user** (ke-6 menutup yang tertua); cleanup saat `ctx.Done()`.
- **Verifikasi:** unit test fan-out, isolasi antar user, cleanup, batas koneksi.

#### B-3 — Ring-buffer
- **Target:** `backend/internal/realtime/buffer.go`
- **Langkah:** Redis ZSET `realtime:buf:{userID}` score = timestamp; `Append(ctx, userID, ev)`;
  `Since(ctx, userID, lastID)` → event setelah ID tsb; TTL 10 menit; no-op bila Redis mati.
- **Verifikasi:** unit test append/replay/urutan/TTL.

#### B-4 — Publisher & subscriber
- **Target:** `backend/internal/realtime/publisher.go`, `redis_sub.go`
- **Langkah:** `PublishUser`/`PublishAdmins` → channel `realtime:u:{id}` / `realtime:admins`;
  subscriber meneruskan ke hub lokal; bila Redis mati, publish langsung ke hub lokal (single
  instance) + log.
- **Verifikasi:** unit test channel benar; degradasi tanpa Redis.

#### B-5 — Handler & route
- **Target:** `backend/internal/realtime/handler.go`, `delivery/http/router.go`, `cmd/server/main.go`
- **Langkah:** handler SSE (header benar, flush, heartbeat 15s, resume header **atau**
  `last_event_id` query, param menang bila header kosong); daftarkan di luar grup `cached`;
  valve `REALTIME_ENABLED` (off → 503).
- **Verifikasi:** unit test 401/header/heartbeat/resume/503.

### W2 — Perbaikan cacat invalidasi

#### B-6 — Versi plan pada kunci cache
- **Target:** `backend/internal/entitlement/resolver.go`
- **Langkah:** kunci menjadi `entitlement:u:{id}:v{userVer}:p{planVer}`; tambah
  `InvalidatePlan(ctx, planID)` (bump `ent:plan:{id}`); `Resolve` membaca kedua versi.
- **Verifikasi:** unit test — bump versi plan mengubah kunci (cache miss).

#### B-7 — Admin memakai broadcaster
- **Target:** `backend/internal/admin/admin.go`, `subscriptions.go`, `cmd/server/main.go`
- **Langkah:** opsi `WithEntitlementBroadcaster`; `invalidateUser` memanggil
  `PublishInvalidate` (bump `"ent:"` + siar peer) **selain** `BumpUserVersion` yang ada
  (tetap untuk cache HTTP); wire `entBroadcaster` di `main.go`.
- **Verifikasi:** unit test broadcaster terpanggil.

#### B-8 — Invalidasi saat plan diedit
- **Target:** `backend/internal/admin/plans.go`
- **Langkah:** `CreatePlan`/`UpdatePlan` memanggil `InvalidatePlan(planID)` setelah menulis
  `plan_entitlements`/`plan_limits`.
- **Verifikasi:** unit/integrasi — user di plan itu mendapat entitlement baru tanpa menunggu TTL.

### W3 — Producer

#### B-9 — `payment_received`
- **Target:** `backend/internal/payment/realtime_publisher.go`, `webhook.go`, `manual_service.go`
- **Langkah:** ubah `UserEventsChannel` → `realtime:u:{id}` dan pancarkan envelope standar;
  panggil pada webhook settled dan approve manual.
- **Verifikasi:** unit test event terbit pada kedua jalur.

#### B-10 — `manual_approval_needed`
- **Target:** `backend/internal/payment/manual_service.go`
- **Langkah:** pancarkan ke channel admin di `UploadProof` setelah status jadi `pending_review`.
- **Verifikasi:** unit test.

#### B-11 — `plan_updated`
- **Target:** `backend/internal/admin/subscriptions.go`, `plans.go`
- **Langkah:** pancarkan ke user (atau siar ke semua user plan terkait) setelah invalidasi.
- **Verifikasi:** unit test.

#### B-12 — `quota_reset`
- **Target:** `backend/internal/billing/ledger.go`
- **Langkah:** pancarkan **hanya** saat `GetOrCreatePeriod` benar-benar membuat periode baru.
- **Verifikasi:** unit test — pembuatan pertama memancarkan, pemakaian berikutnya tidak.

#### B-13 — `ai_endpoint_down`
- **Target:** `backend/internal/aigateway/breaker/circuit_breaker.go`, `gateway/service.go`
- **Langkah:** `Config.OnStateChange`; kumpulkan transisi, invoke **setelah lock dilepas**;
  `gateway.Config` meneruskan.
- **Verifikasi:** unit test transisi memicu callback; **test anti-deadlock** (callback memanggil
  `State()`).

### W4 — Verifikasi backend

#### B-14 — Unit test
Cakupan per komponen seperti di atas.

#### B-15 — Integrasi
- **Target:** `backend/tests/realtime_e2e_test.go`
- **Kasus:** buka stream → publish → terima; resume `Last-Event-ID` → backlog berurutan;
  isolasi user A vs B; non-admin tidak menerima event admin; `REALTIME_ENABLED=off` → 503.

### W5 — Frontend

#### B-16 — Dispatcher
- **Target:** `frontend/lib/realtime-dispatcher.ts` + test
- **Langkah:** peta event → query key **nyata**; tanpa `invalidateQueries()` tanpa key.

#### B-17 — Hook stream
- **Target:** `frontend/hooks/use-realtime-stream.ts` + test
- **Langkah:** `EventSource` ke `/api/v1/realtime/stream` (cookie ikut otomatis);
  backoff manual 1→2→4→8→16s + jitter; sertakan `last_event_id` saat reconnect; status koneksi.

#### B-18 — Provider
- **Target:** `frontend/providers/realtime-provider.tsx`, `app/providers.tsx`
- **Langkah:** pasang saat terautentikasi, lepas saat logout.

#### B-19 — Gerbang frontend
`pnpm lint && pnpm build && pnpm test`.

### W6 — Finalisasi

#### B-20 — Dokumentasi
`teknis/sse-strategy.md` (cookie `ft_at`, channel nyata), `teknis/cache-invalidation.md`
(query key nyata), `tracking/tracking.md`, `log_aktivitas.md`.

#### B-21 — Commit

---

## 4. Matriks Bukti

| Langkah | Artefak | Bukti |
|---|---|---|
| B-1..B-5 | `pkg/contracts/events.go`, `internal/realtime/*` | unit test per komponen |
| B-6..B-8 | `entitlement/resolver.go`, `admin/*` | unit test invalidasi |
| B-9..B-13 | `payment/*`, `billing/ledger.go`, `breaker/*` | unit test producer |
| B-15 | `tests/realtime_e2e_test.go` | `go test -run TestRealtime ./tests/...` |
| B-16..B-18 | `lib/realtime-dispatcher.ts`, `hooks/use-realtime-stream.ts` | `pnpm test` |

---

## 5. Perintah Verifikasi

```bash
cd backend && go vet ./... && go test ./... && go build ./...
cd backend && TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5435/fintrack_test?sslmode=disable" \
  go test -run TestRealtime -v ./tests/...
cd frontend && pnpm lint && pnpm build && pnpm test
```

---

## 6. Rollback

1. **Valve cepat:** `REALTIME_ENABLED=off` → endpoint 503 + publisher no-op, tanpa ubah kode.
2. **Rollback kode:** revert commit slice B. **Tanpa migrasi** — tidak ada perubahan skema.
3. **Perbaikan bug #4 tidak boleh ikut di-rollback sembarangan:** mematikannya mengembalikan
   bug "perubahan paket baru terasa 24 jam". Bila perlu rollback, pisahkan komit bila memungkinkan.

---

## 7. Handoff

Slice berikutnya: **C** (standar UI: DataTable, cursor-pointer, a11y WCAG 2.1 AA, optimistic
update) dan **D** (BYOK cleanup + health gates + graceful shutdown 30s + expand/contract).

**Gap yang dicatat (bukan dikerjakan di sini):** reset kuota tidak terjadwal (lazy saat
pemakaian AI berikutnya); tidak ada inbox notifikasi durabel.
