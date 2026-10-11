# P07 Slice B — Realtime SSE & Cache Invalidation: Design

**Date:** 2026-10-11
**Status:** Approved (pending written-spec review)
**Phase:** P07 (Rollout RBAC, Realtime & Standar) — slice 2 of 5
**Related tasks:** TASK-P07-004 … TASK-P07-009 (+ perbaikan invalidasi lintas slice)

---

## 1. Problem

Tidak ada modul realtime sama sekali. Kondisi terverifikasi:

| Komponen | Kondisi |
|---|---|
| `backend/internal/realtime/` (hub, client, buffer, handler) | **tidak ada** |
| `payment/realtime_publisher.go` | ada, tapi **0 pemanggil** |
| Endpoint SSE | tidak ada |
| `payment.Notifier` (interface dunning) | interface ada, **implementasi tidak ada**, tidak di-wire |
| `EventSource` di frontend | tidak ada |
| Redis pub/sub | ada (`Publish`/`Subscribe`) |
| Cookie auth | `ft_at` (httpOnly) — `AuthMiddleware` sudah membacanya |

### 1.1 Tiga cacat invalidasi yang ditemukan (bug #4)

Ditemukan saat recon; semuanya membuat perubahan paket/fitur tidak terasa sampai
TTL cache entitlement (24 jam) habis.

| # | Lokasi | Masalah |
|---|---|---|
| 4a | `admin/subscriptions.go:326` | Membump scope `"u:"` (untuk cache HTTP — memang benar) tetapi resolver entitlement memakai scope `"ent:"`. Cache entitlement **tidak** terinvalidasi saat paket user berubah. |
| 4b | `admin/plans.go` (Create/UpdatePlan) | Menulis ulang `plan_entitlements`/`plan_limits` **tanpa invalidasi apa pun** → mengubah fitur sebuah plan membuat seluruh user di plan itu memakai entitlement basi hingga 24 jam. |
| 4c | `cmd/server/main.go:227` | `entBroadcaster` hanya dideklarasi (`_ = entBroadcaster // used by P05/P06 on plan changes`) — **tidak pernah dipanggil**, jadi tidak ada invalidasi lintas-instance. |

> **Koreksi klaim awal:** `BumpUserVersion` **bukan** dead code. `CacheMiddleware`
> memakainya sebagai komponen kunci cache HTTP (`cache.go:59`). Yang salah adalah
> scope-nya beda tujuan, bukan tidak terpakai. Panggilan `BumpUserVersion` yang ada
> **dipertahankan**.

---

## 2. Goals / Non-goals

**Goals**

1. Modul `internal/realtime/`: hub SSE + fan-out Redis + resiliensi (cookie auth,
   heartbeat 15s, resume `Last-Event-ID`).
2. Lima event domain dengan producer nyata.
3. Frontend: `EventSource` + backoff + surgical cache invalidation.
4. Perbaiki cacat invalidasi 4a/4b/4c.

**Non-goals**

- **Tidak ada inbox/notifikasi durabel.** Tidak ada tabel `notifications`. SSE adalah
  akselerator UX, bukan sumber kebenaran: bila admin offline, `manual_approval_needed`
  hilang, tetapi datanya tetap di DB dan muncul saat halaman dibuka.
- Tidak ada perubahan pada flow pembayaran/billing selain titik publish.
- Tidak ada UI baru selain indikator status koneksi.

---

## 3. Design

### 3.1 Struktur modul

| Berkas | Isi |
|---|---|
| `hub.go` | Registry koneksi thread-safe: `Register`/`Unregister`/`SendToUser`/`SendToAdmins`; cleanup saat `ctx.Done()` |
| `client.go` | Satu koneksi SSE: buffered channel, writer dengan flush per event |
| `events.go` | Envelope + 5 tipe event |
| `publisher.go` | `PublishUser(userID, ev)` / `PublishAdmins(ev)` → Redis |
| `redis_sub.go` | Listener yang meneruskan pesan Redis ke hub lokal |
| `buffer.go` | Ring-buffer per-user di Redis (ZSET, TTL 10 menit) untuk `Last-Event-ID` |
| `handler.go` | `GET /api/v1/realtime/stream` |

### 3.2 Channel

| Event | Channel | Penerima |
|---|---|---|
| `payment_received`, `plan_updated`, `quota_reset` | `realtime:u:{userID}` | user pemilik |
| `ai_endpoint_down`, `manual_approval_needed` | `realtime:admins` | semua `admin`/`super_admin` |

Satu channel untuk admin (bukan per-admin) karena himpunan penerimanya berubah-ubah;
publish per-admin berarti harus mengetahui daftar admin setiap kali. Filter dilakukan di
hub: hanya klien ber-role admin yang dihubungkan ke sana.

### 3.3 Endpoint

- `GET /api/v1/realtime/stream`
- Auth: **`AuthMiddleware` yang sudah ada** (membaca cookie httpOnly `ft_at`, fallback
  `Authorization: Bearer`). Dokumen menyebut cookie `fintrack_token` — **usang**.
  Tidak ada token di query param.
- Didaftarkan **di luar** grup `cached` agar tidak tersentuh cache 30s.
- Header: `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-transform`,
  `Connection: keep-alive`, `X-Accel-Buffering: no`.
- Format W3C: `id:`, `event:`, `data:` (envelope `{event, timestamp, id, payload}`).
- ID: `evt_<unixnano>_<seq>`; monotonik per koneksi.

### 3.4 Perbaikan invalidasi (bug #4)

**4a — perubahan paket per-user.** `admin/subscriptions.go` memanggil
`entBroadcaster.PublishInvalidate(userID)` (membump `"ent:"` + menyiarkan ke peer).
`BumpUserVersion` yang ada dipertahankan untuk cache HTTP.

**4b — perubahan fitur/limit sebuah plan.** Menyentuh banyak user; bump per-user akan
O(jumlah user). Ditambahkan **komponen versi plan** pada kunci cache resolver:

```
entitlement:u:{userID}:v{userVersion}:p{planVersion}
```

Satu bump `ent:plan:{planID}` menginvalidasi semua user plan itu, O(1). Bump per-user
tetap ada untuk perubahan spesifik user.

**4c — lintas instance.** `entBroadcaster` disuntikkan ke admin Service (pola seperti
`WithCache`) dan dipanggil pada kedua jalur; subscriber yang sudah ada menangani sisi
penerima.

### 3.5 Producer kelima event

| Event | Dipicu di | Channel |
|---|---|---|
| `payment_received` | webhook sukses + approve manual | user |
| `plan_updated` | setelah 4a/4b | user |
| `quota_reset` | `LedgerService` saat `GetOrCreatePeriod` **membuat periode baru** | user |
| `ai_endpoint_down` | circuit breaker `OnStateChange` | admin |
| `manual_approval_needed` | `ManualService.UploadProof` | admin |

> **Koreksi (self-review):** draf awal menyebut `quota_reset` dipicu di cleaner worker.
> Itu **salah** — `CleanerWorker.sweep` hanya melepas reservasi kedaluwarsa
> (`ListExpiredReservations` → `Release`); **tidak ada mekanisme rotasi siklus** di codebase.
> `GetOrCreatePeriod` dipanggil *lazy* oleh `LedgerService` (`ledger.go:220`) ketika user
> memakai AI; di situlah periode baru dibuat, dan **itu** momen reset kuota yang sebenarnya.
> Event dipancarkan hanya ketika periode benar-benar baru dibuat (bukan pada setiap
> pemanggilan), agar tidak mengirim `quota_reset` palsu.
>
> Konsekuensi yang perlu diketahui: reset terdeteksi saat user berikutnya memakai AI,
> bukan pada tengah malam tanggal siklus. Untuk siklus yang benar-benar dijadwalkan perlu
> worker rotasi tersendiri — itu **di luar cakupan slice ini** dan dicatat sebagai gap.

**Hook circuit breaker.** `breaker.Config` mendapat `OnStateChange func(id string, from, to State)`.
**Penting:** `RecordFailure`/`RecordSuccess` memegang `b.mu` saat transisi, jadi callback
harus dipanggil **setelah lock dilepas** (transisi dikumpulkan lalu di-invoke di luar lock).
Memanggil di dalam lock akan deadlock bila callback menyentuh breaker (mis. `State()`).
`gateway.Config` meneruskan hook ini. Breaker tetap dapat diuji tanpa Redis (callback opsional).

### 3.6 Frontend

| Berkas | Isi |
|---|---|
| `lib/realtime-dispatcher.ts` | Fungsi murni: event → daftar query key |
| `hooks/use-realtime-stream.ts` | Lifecycle `EventSource`, backoff, status koneksi |
| `providers/realtime-provider.tsx` | Pasang saat login, lepas saat logout |
| `app/providers.tsx` | Mount provider |

**Pemetaan event → query key (hanya key yang benar-benar ada).** Dokumen
`cache-invalidation.md` memakai `billing.subscription`/`billing.invoices`/`billing.tokens`
yang **tidak ada**; dipetakan ke key nyata:

| Event | Invalidasi |
|---|---|
| `payment_received` | `entitlements.me`, `dashboard.all` |
| `plan_updated` | `entitlements.me`, `dashboard.all` |
| `quota_reset` | `entitlements.me`, `transactions.all`, `dashboard.all`, `reports.all` |
| `ai_endpoint_down` | `superAdmin.gatewayEndpoints`, `superAdmin.metricsOverview` |
| `manual_approval_needed` | `superAdmin.paymentsReview`, `superAdmin.metricsOverview` |

Semua **surgical** — tidak ada `invalidateQueries()` tanpa key (memenuhi TASK-P07-009).

**Reconnect: opsi A (manual backoff).** Tutup dan buat ulang `EventSource` dengan backoff
1s→2s→4s→…→cap 16s + jitter. Karena `EventSource` yang dibuat ulang tidak lagi mengirim
`Last-Event-ID` otomatis, ID terakhir dikirim sebagai **query param `?last_event_id=`**.
Server menerima header **atau** query param (header menang bila keduanya ada). Param berisi
ID event, bukan token — tidak melanggar larangan token di query.

### 3.7 Degradasi & proteksi

- **Tanpa Redis:** publish jatuh ke hub lokal saja (single-instance tetap jalan), dicatat di
  log. Ring-buffer tidak tersedia → resume `Last-Event-ID` dilewati; klien hanya menerima
  event baru. Keterbatasan ini **didokumentasikan**, bukan error.
- **Valve `REALTIME_ENABLED`** (default on). `off` → endpoint 503 + publisher no-op.
  Konsisten dengan pola `ENTITLEMENT_ENFORCEMENT`.
- **Batas koneksi per user** (5). Koneksi ke-6 menutup yang tertua. Tanpa ini, klien yang
  membuka tab berulang menghabiskan goroutine + memori; heartbeat 15s per koneksi
  memperbesar tekanan.

---

## 4. Testing

### 4.1 Backend unit

- `hub`: register/unregister, fan-out ke user tepat, tidak bocor ke user lain, cleanup saat
  context cancel, `SendToAdmins` hanya ke klien admin.
- `buffer`: append + replay by ID, TTL, urutan terjaga.
- `handler`: 401 tanpa cookie; header benar; heartbeat `: ping` muncul; resume mengirim backlog.
- `publisher`: channel yang benar (user vs admins).
- `events`: validasi envelope kelima tipe.
- `breaker`: `OnStateChange` terpanggil pada transisi Open, dan **tidak deadlock** (uji
  callback yang memanggil `State()`).

### 4.2 Integrasi (`tests/realtime_e2e_test.go`)

Pola P06/P07-A: router nyata + Redis nyata, env-gated `TEST_DATABASE_URL`.

- buka stream → publish → terima
- putus → sambung ulang dengan `Last-Event-ID` → terima backlog berurutan
- user A tidak menerima event user B
- non-admin tidak menerima event channel admin
- `REALTIME_ENABLED=off` → 503

### 4.3 Frontend (Vitest)

- dispatcher: setiap event → key yang benar; **tidak ada** invalidasi tanpa key
- backoff: urutan 1→2→4→8→16s, cap di 16s, jitter ada

### 4.4 Gerbang

`go vet`/`go test`/`go build` hijau; `pnpm lint`/`build`/`test` hijau.

---

## 5. Dokumentasi yang dikoreksi

- `teknis/sse-strategy.md`: cookie `fintrack_token` → `ft_at`; channel
  `realtime:events:*` → `realtime:u:{id}` + `realtime:admins`; status implementasi.
- `teknis/cache-invalidation.md`: query key yang tidak ada → key nyata.
- `tracking/tracking.md`, `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.

---

## 6. Risks

| Risiko | Mitigasi |
|---|---|
| Deadlock callback breaker | Callback di-invoke **setelah** lock dilepas; ada test khusus |
| Kebocoran koneksi/goroutine | Batas 5 koneksi/user + cleanup `ctx.Done()` + heartbeat |
| Event lintas-user | Test isolasi user A vs B di integrasi |
| Redis mati | Degradasi single-instance + log; resume dilewati (didokumentasikan) |
| Event hilang saat admin offline | Diterima sebagai non-goal; DB tetap sumber kebenaran |
| `Last-Event-ID` hilang karena EventSource dibuat ulang | Dikirim sebagai query param, server menerima header atau param |
