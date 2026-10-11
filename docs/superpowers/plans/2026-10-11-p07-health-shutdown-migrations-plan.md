# Implementation Plan — P07 Slice D1: Health Gates, Shutdown & Aturan Migrasi

| Field | Isi |
|---|---|
| **Fase** | P07 — Rollout RBAC, Realtime & Standar (**Slice D1**; BYOK → D2) |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | Slice A (`9500e4b`), B (`ea725e6`), C (`925abd8`) selesai |
| **Spec desain** | `docs/superpowers/specs/2026-10-11-p07-health-shutdown-migrations-design.md` |
| **Task terkait** | TASK-P07-016, 017, 018, 003 |
| **Status** | Siap dieksekusi |

---

## 1. Tujuan & Definition of Done

Menghadirkan health gate yang memeriksa dependensi nyata, drain shutdown 30s, aturan migrasi
expand/contract tertulis, dan memperbaiki kebenaran semantik flag gateway **dengan pengaman**
agar tidak mematikan AI.

**Definition of Done:**
- [ ] `/health/live` (tanpa I/O) + `/health/ready` (DB + pgvector + Redis) + `/health` alias.
- [ ] DB mati → 503; Redis mati → 200 `degraded`; payload memuat latensi dependensi.
- [ ] Drain shutdown 30s via `SHUTDOWN_TIMEOUT`; healthcheck backend + `stop_grace_period` di compose.
- [ ] `database/migrations/README.md` memuat aturan expand/contract.
- [ ] TASK-P07-003 ditandai selesai dengan bukti.
- [ ] `IsEnabledForPlatform` + pengaman `HasActiveRoutes`; **gateway tetap mati** pada state sekarang.
- [ ] `go vet`/`go test`/`go build` hijau.
- [ ] Dokumentasi + tracking diperbarui; gap AC dicatat.

---

## 2. Urutan Eksekusi

| Wave | Langkah | Deskripsi | Prasyarat |
|---|---|---|---|
| **W1** | D1-1 | `featureflag.IsEnabledForPlatform` + test | — |
| **W1** | D1-2 | `HasActiveRoutes` di repo + interface + fake | — |
| **W1** | D1-3 | Perbaiki `gatewayEnabled` di `main.go` + pengaman | D1-1, D1-2 |
| **W2** | D1-4 | Health handler (`live`/`ready`) + route | — |
| **W2** | D1-5 | Test health (200/503/degraded/latensi) | D1-4 |
| **W3** | D1-6 | Shutdown 30s via env | — |
| **W3** | D1-7 | Compose: healthcheck backend + `stop_grace_period` | D1-4 |
| **W4** | D1-8 | `database/migrations/README.md` (expand/contract) | — |
| **W4** | D1-9 | Verifikasi & tandai TASK-P07-003 | — |
| **W5** | D1-10 | Gerbang penuh `go vet`/`test`/`build` | semua |
| **W5** | D1-11 | Dokumentasi + tracking + log | D1-10 |
| **W5** | D1-12 | Commit | D1-11 |

**Jalur kritis:** D1-1 → D1-3 (pengaman harus benar sebelum menyentuh `main.go`), dan
D1-4 → D1-5 (handler harus ada sebelum diuji).

---

## 3. Detail Langkah

### W1 — Flag gateway + pengaman

#### D1-1 — `IsEnabledForPlatform`
- **Target:** `backend/internal/featureflag/featureflag.go` + test
- **Langkah:** method baru yang membaca flag **tanpa** bucketing user: `is_enabled=true` dan
  `rollout > 0` → true; `is_enabled=false` → false; flag hilang/error → false (fail-closed).
  Doc comment menjelaskan bedanya dengan `IsEnabled` (rollout per-user).
- **Verifikasi:** unit test — `enabled=true, rollout=25` → true; `enabled=false` → false;
  rollout 0 → false; flag hilang → false.

#### D1-2 — `HasActiveRoutes`
- **Target:** `backend/internal/domain/aigateway.go` (interface),
  `backend/internal/repository/postgres/ai_gateway_repo.go`, fake di test
- **Langkah:** method `HasActiveRoutes(ctx, useCase) (bool, error)` yang memakai query route yang
  sudah ada (`GetActiveRoutesByUseCase`) dan melaporkan apakah ada minimal satu route aktif.
  Enumerasi semua implementer lewat `grep`, bukan asumsi.
- **Verifikasi:** unit test ada/tidak ada route.

#### D1-3 — Pengaman di `main.go`
- **Target:** `backend/cmd/server/main.go`
- **Langkah:**
  ```go
  gatewayEnabled := flagSvc.IsEnabledForPlatform(bgCtx, "feature_ai_gateway_centralized")
  if gatewayEnabled {
      hasRoutes, err := aiGatewayRepo.HasActiveRoutes(bgCtx, domain.UseCaseChat)
      if err != nil { log warn; hasRoutes = false }
      gatewayEnabled = hasRoutes
  }
  ```
  Log jelas saat flag aktif tapi route kosong (menjelaskan mengapa gateway tidak dinyalakan).
- **Verifikasi:** test bahwa flag aktif + route kosong → gateway tidak aktif; pada state
  sekarang (`ai_routes` kosong) `gatewayEnabled=false`.

### W2 — Health gates

#### D1-4 — Handler health
- **Target:** `backend/internal/delivery/http/health.go` (baru), `router.go`
- **Langkah:** `/health/live` (200 tanpa I/O); `/health/ready` memeriksa `pool.Ping` + ekstensi
  `vector` + `redis.Ping`; 503 bila DB gagal; Redis gagal → 200 `degraded`; payload memuat
  `status` + `latency_ms` tiap dependensi. `/health` tetap sebagai alias live.
- **Verifikasi:** D1-5.

#### D1-5 — Test health
- **Target:** `backend/internal/delivery/http/health_test.go`
- **Kasus:** live 200; ready 200 saat DB sehat; ready 503 saat pool tidak valid; Redis nil/mati →
  200 `degraded`; payload memuat latensi.

### W3 — Shutdown & compose

#### D1-6 — Drain 30s
- **Target:** `cmd/server/main.go`
- **Langkah:** `SHUTDOWN_TIMEOUT` (default 30s) untuk `srv.Shutdown`.
- **Verifikasi:** unit test parser timeout (default & override).

#### D1-7 — Compose
- **Target:** `docker-compose.yml`
- **Langkah:** healthcheck `backend` memakai `/health/ready`; `stop_grace_period: 30s`.
- **Verifikasi:** `docker compose config` valid (tanpa menjalankan stack).

### W4 — Aturan migrasi & verifikasi

#### D1-8 — `database/migrations/README.md`
- **Langkah:** tulis aturan expand/contract: larangan `DROP COLUMN` langsung & `SET NOT NULL`
  tanpa fase perantara; wajib `ADD COLUMN` nullable, `CREATE INDEX CONCURRENTLY`, migrasi
  aditif & idempoten, tidak mengedit migrasi yang sudah di-commit.
- **Verifikasi:** dokumen ada; contoh pola benar.

#### D1-9 — TASK-P07-003
- **Langkah:** catat hasil verifikasi (nol `sync.Map`/`globalCache`) sebagai bukti di tracking.
- **Verifikasi:** grep ulang.

### W5 — Finalisasi

#### D1-10 — Gerbang penuh
`go vet ./... && go test ./... && go build ./...`

#### D1-11 — Dokumentasi
`tracking/tracking.md`, `log_aktivitas.md`, `teknis/zero-downtime-deploy.md` (status + gap k8s).

#### D1-12 — Commit

---

## 4. Matriks Bukti

| Langkah | Artefak | Bukti |
|---|---|---|
| D1-1/D1-2 | `featureflag.go`, `ai_gateway_repo.go` | unit test |
| D1-3 | `main.go` | test pengaman; log runtime |
| D1-4/D1-5 | `health.go` + test | `go test -run TestHealth` |
| D1-6 | `main.go` | unit test parser |
| D1-7 | `docker-compose.yml` | `docker compose config` |
| D1-8 | `database/migrations/README.md` | dokumen |
| D1-9 | tracking | grep ulang |

---

## 5. Perintah Verifikasi

```bash
cd backend && go vet ./... && go test ./... && go build ./...
docker compose config >/dev/null && echo "compose ok"
```

---

## 6. Rollback

Revert commit D1. **Tanpa migrasi.** Pengaman gateway aman untuk di-rollback: mengembalikannya
mengembalikan kondisi "flag tidak pernah terbaca" (gateway mati) — tidak ada perubahan perilaku
AI yang bergantung padanya.

---

## 7. Gap yang dicatat (bukan diklaim)

- **Gateway tetap mati** setelah D1 (`ai_routes` kosong). "P03 tidak berjalan" **belum
  terselesaikan** — butuh provisioning endpoint + route + 9router hidup.
- **Template k8s** tidak dibuat (`deploy/` tidak ada di repo).
- **Uji rolling restart 200 RPS tanpa 5xx** tidak dijalankan (uji beban terkoordinasi).
- **BYOK belum dihapus** — slice D2.

---

## 8. Handoff

**D2 — BYOK cleanup**, dengan syarat yang sudah ditetapkan: pastikan gateway benar-benar
melayani semua user (endpoint hidup + route terisi + flag aktif) **sebelum** menghapus BYOK,
karena saat ini BYOK adalah satu-satunya jalur AI yang berfungsi.
