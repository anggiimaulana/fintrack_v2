# P07 Slice D1 — Health Gates, Shutdown & Aturan Migrasi: Design

**Date:** 2026-10-11
**Status:** Approved (pending written-spec review)
**Phase:** P07 (Rollout RBAC, Realtime & Standar) — slice 4 of 5 (**D1**; BYOK → D2)
**Related tasks:** TASK-P07-016, TASK-P07-017, TASK-P07-018, TASK-P07-003 (verifikasi)

---

## 1. Problem

### 1.1 Health & shutdown

| Area | Kondisi terverifikasi |
|---|---|
| `/health` | satu endpoint; hanya `{"status":"ok"}`, **tidak memeriksa DB/Redis** |
| `/health/live`, `/health/ready` | tidak ada |
| Graceful shutdown | SIGTERM/SIGINT + `srv.Shutdown` ada, tapi **10s** (AC meminta 30s) |
| Healthcheck `backend` di compose | **tidak ada** |
| `deploy/` (k8s) | **tidak ada** di repo |

### 1.2 Aturan migrasi

`database/migrations/README.md` belum ada. Tidak ada aturan tertulis yang melarang
`DROP COLUMN` / `ALTER ... SET NOT NULL` tanpa fase perantara.

### 1.3 Temuan kritis: gateway terpusat tidak pernah aktif

```go
gatewayEnabled := flagSvc.IsEnabled(bgCtx, "feature_ai_gateway_centralized", nil)
```

`IsEnabled` mengembalikan **false** bila `userID == nil` dan rollout parsial. Flag bernilai
`enabled=true, rollout=25` → **selalu false**. Log runtime membuktikannya:
`"AI gateway centralized: disabled (feature flag off)"`.

Akibatnya **seluruh P03 tidak berjalan**: routing, breaker, metering, failover semuanya mati;
100% traffic AI lewat BYOK per-user.

**Flag itu artefak uji, bukan keputusan produk.** Riwayat di dump menunjukkan flag di-toggle via
`curl` saat pengujian P06 (`is_enabled: false→true`, `rollout: 0→25`).

**Infrastruktur gateway belum siap:**

| Pemeriksaan | Hasil |
|---|---|
| `ai_endpoints` | 1 baris: `9router-primary`, `base_url=http://localhost:20128/v1`, `health=unhealthy`, **tanpa API key** |
| Jangkauan endpoint | **connection refused** dari host; **timeout** dari container |
| `ai_routes` | **0 baris** |

**Konsekuensi:** menyalakan gateway pada kondisi ini akan mengalihkan seluruh chat ke endpoint
mati + route kosong, dan **mematikan AI untuk semua user**. Karena itu D1 memperbaiki
*kebenaran semantik flag* dan menambahkan **pengaman**, bukan memaksa gateway hidup.

---

## 2. Goals / Non-goals

**Goals**

1. `/health/live` + `/health/ready` dengan diagnostik dependensi; `/health` dipertahankan.
2. Graceful shutdown 30s (dapat dikonfigurasi) + healthcheck backend di compose.
3. Aturan migrasi expand/contract tertulis.
4. Verifikasi ulang TASK-P07-003 (`globalCache`).
5. Perbaiki kebenaran semantik flag gateway + pengaman agar tidak mematikan AI.

**Non-goals**

- **Tidak menghapus BYOK** (→ D2).
- **Tidak menyalakan gateway** pada infrastruktur yang belum siap.
- Tidak membuat manifes k8s (`deploy/` tidak ada di repo).

---

## 3. Design

### 3.1 Health gates

| Endpoint | Isi | Kriteria |
|---|---|---|
| `/health/live` | proses hidup, tanpa I/O | 200, < 1ms |
| `/health/ready` | PostgreSQL (`Ping` + ekstensi `vector`) + Redis (`Ping`) | 200 bila DB sehat; **503** bila DB mati; Redis mati → tetap 200 dengan `redis: degraded` |
| `/health` | alias `/health/live` | dipertahankan untuk kompatibilitas (dipakai `curl -i localhost:8080/health` di README) |

> **Koreksi (self-review):** draf awal menyebut `/health` dipakai `tester/p06_smoke.sh` dan k6.
> Setelah dicek, **tidak ada** yang memakainya — smoke/k6 tidak menyentuh `/health`. Alasan
> mempertahankannya adalah kompatibilitas dokumentasi (README mencontohkannya) dan kebiasaan
> probe eksternal, bukan ketergantungan yang terbukti.

**Mengapa Redis mati bukan 503:** Redis opsional di codebase ini — `REDIS_URL` boleh kosong dan
cache melakukan graceful degradation. Menjadikannya syarat readiness akan menandai pod tidak siap
padahal aplikasi tetap melayani. **DB adalah hard dependency**: tanpa DB tidak ada yang bisa
dilayani, jadi 503. Perbedaan ini disengaja dan dicatat.

Payload `/health/ready` memuat status + latensi tiap dependensi (JSON diagnostik), sesuai AC.

### 3.2 Graceful shutdown

- Timeout drain `10s → 30s`, dapat dikonfigurasi lewat `SHUTDOWN_TIMEOUT` (default 30s).
- Compose: tambah `healthcheck` untuk service `backend` (memakai `/health/ready`) dan
  `stop_grace_period: 30s` agar Docker memberi waktu drain sebelum SIGKILL.
- **Gap yang dicatat:** AC menyebut template k8s (`deploy/k8s/deployment.yaml`); repo tidak punya
  direktori `deploy/`. Tidak dibuat — mengarang manifes yang tidak dipakai tidak berguna.
- **Gap yang dicatat:** AC menyebut uji rolling restart pada 200 RPS tanpa 5xx. Itu uji beban
  terkoordinasi, bukan unit test; saya tidak akan mengklaimnya tanpa menjalankannya.

### 3.3 Aturan migrasi expand/contract

`database/migrations/README.md` baru, memuat aturan yang mengikat:

| Dilarang | Wajib |
|---|---|
| `DROP COLUMN` langsung | fase perantara (berhenti dipakai → drop di rilis berikutnya) |
| `ALTER ... SET NOT NULL` tanpa default/backfill | `ADD COLUMN` nullable, backfill, lalu constraint terpisah |
| `CREATE INDEX` (blocking) | `CREATE INDEX CONCURRENTLY` |
| Edit migrasi yang sudah di-commit | migrasi bernomor baru; dump di-regenerate |

Semua migrasi wajib aditif & idempoten (pola `IF NOT EXISTS`/`OR REPLACE` yang sudah dipakai
0001–0010).

### 3.4 Verifikasi TASK-P07-003 (`globalCache`)

**Sudah diverifikasi (2026-10-11):** `grep` untuk `sync.Map` dan `globalCache` di seluruh
`backend/` (non-test) mengembalikan **nol hasil**. Klaim AC TASK-P07-003 ("nol referensi
`sync.Map` atau slice memori global untuk data pengguna antar-request") **terpenuhi** — cache
sudah Redis berversi sejak P04. Tidak ada pekerjaan tersisa; hanya ditandai selesai dengan bukti.

### 3.5 Flag gateway: perbaikan + pengaman

**Perbaikan kebenaran:** tambah `featureflag.IsEnabledForPlatform(ctx, key) bool` — membaca flag
tanpa bucketing user, sehingga `enabled=true, rollout=25` → **true**. Dipakai untuk flag yang
sifatnya kill switch platform, bukan rollout per-user.

**Pengaman di `main.go`:** gateway hanya diaktifkan bila **flag aktif DAN ada route aktif
untuk `chat` DAN endpoint-nya sehat**:

```go
gatewayEnabled := flagSvc.IsEnabledForPlatform(ctx, "feature_ai_gateway_centralized")
if gatewayEnabled {
    gatewayEnabled = aiGatewayRepo.HasHealthyEndpoint(ctx, domain.UseCaseChat)
}
```

**Mengapa "endpoint sehat" ikut disyaratkan (koreksi setelah recon):** rencana awal hanya
memeriksa *route ada*. Recon menemukan `ai_routes` **berisi 5 route aktif** (chat, vision, voice,
embedding, reasoning) yang menunjuk `9router-primary` — jadi pengaman berbasis route saja akan
**membuka** gateway. Sementara itu `chatViaGateway` **tidak punya fallback ke BYOK**: bila
gateway gagal, user melihat pesan error. Jadi mengarahkan seluruh platform ke endpoint yang
kesehatan-nya gagal akan **mematikan AI untuk semua user**.

`HasHealthyEndpoint` mensyaratkan `health == "healthy"`. Nilai `"unknown"` (endpoint baru yang
belum pernah diprobe) **tidak dihitung sehat** — belum diprobe bukan bukti bisa melayani.

**Status pada data saat ini (diverifikasi dengan menjalankan kode terhadap DB):**

```
HasActiveRoutes(chat)   = true
HasHealthyEndpoint(chat) = false      (endpoint berstatus "unhealthy")
=> gatewayEnabled = false             (BYOK tetap melayani)
```

Gateway akan terbuka sendiri begitu endpoint dilaporkan sehat oleh health checker — tanpa
perubahan kode.

**Konsekuensi yang dinyatakan terbuka:** setelah D1, gateway **masih mati**. "P03 tidak berjalan"
**belum terselesaikan** — butuh provisioning endpoint + route + 9router hidup. Dicatat sebagai
temuan prioritas tinggi di tracking dan log, **tidak diklaim beres**.

Perlu method baru `HasActiveRoutes(ctx, useCase)` pada `AIEndpointRepository` (memanfaatkan query
yang sudah ada di `loadRoutes`).

---

## 4. Testing

### 4.1 Backend
- `/health/live` → 200 tanpa menyentuh DB (diuji dengan pool nil).
- `/health/ready` → 200 saat DB sehat; **503** saat DB mati (pool diarahkan ke DSN tak valid).
- Redis mati → 200 dengan `redis: degraded` (bukan 503).
- Payload memuat latensi per dependensi.
- `IsEnabledForPlatform`: `enabled=true, rollout=25` → true; `enabled=false` → false;
  flag hilang → false (fail-closed).
- Pengaman gateway: flag aktif + route kosong → gateway **tidak** aktif.

### 4.2 Gerbang
`go vet`/`go test`/`go build` hijau.

---

## 5. Dokumentasi

`database/migrations/README.md` (baru), `tracking/tracking.md`, `log_aktivitas.md`, dan
`teknis/zero-downtime-deploy.md` (status implementasi + gap k8s).

---

## 6. Risks

| Risiko | Mitigasi |
|---|---|
| Mengaktifkan gateway ke infra mati → AI berhenti | Pengaman route aktif; diverifikasi test bahwa gateway tetap mati saat route kosong |
| Redis mati dianggap tidak siap | Redis = degraded, bukan 503 (disengaja, sesuai sifat opsionalnya) |
| AC k8s & uji 200 RPS tidak terpenuhi | Dicatat sebagai gap, tidak diklaim |
| Mengubah flag mengubah perilaku runtime | Flag tetap `enabled=true` tapi pengaman route menahannya; dicatat di log |
