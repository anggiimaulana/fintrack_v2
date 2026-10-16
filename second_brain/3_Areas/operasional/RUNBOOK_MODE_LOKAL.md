# Runbook — Mode Run Lokal (Hybrid vs Docker)

FinTrack v2 punya **dua cara menjalankan stack**, dan keduanya berebut port yang
sama. Berpindah mode tanpa mematikan mode sebelumnya adalah penyebab error
paling sering ("error docker" / 404 misterius). Runbook ini mencatat kedua mode,
cara berpindah, dan gejalanya.

## 1. Dua Mode

| | **hybrid** (Cara 2 di README) | **docker** (Cara 1 di README) |
|---|---|---|
| db + redis | container | container |
| backend | host — `go run ./cmd/server` | container |
| frontend | host — `pnpm dev` | container |
| Port :3001 | proses host (`node`) | `fintrack_v2_frontend` |
| Port :8081 | proses host (`main`) | `fintrack_v2_backend` |
| Kelebihan | hot-reload, cepat untuk dev | konsisten, mirip produksi |
| Kekurangan | perlu Go + Node di host | build lambat, tanpa hot-reload |

**Peta port (sama di kedua mode):**

| Port | Layanan |
|---|---|
| 3001 | frontend |
| 8081 | backend |
| 5435 | PostgreSQL |
| 6380 | Redis (6379 dipakai stack lain di mesin ini) |

> **Port 8080 sengaja dibiarkan bebas** untuk proyek lokal lain. FinTrack memakai
> **8081** di kedua mode (kode default `main.go`, `PORT` di `backend/.env`,
> `ports`/`PORT` compose, dan `BACKEND_ORIGIN` frontend sudah seragam 8081).
> Jangan hardcode 8080 di skrip baru.

## 2. Gejala & Arti

| Gejala | Penyebab |
|---|---|
| `ports are not available ... Only one usage of each socket address` saat `docker compose up` | host dev (`pnpm dev` / `go run`) masih memegang :3001 / :8081 |
| `/login` 404 padahal file `app/(auth)/login/page.tsx` ada | dev server host punya route table basi (bukan masalah Docker) |
| `/user` → 307 ke `/login`, lalu 404 | belum login (memang benar) + `/login` 404 karena sebab di atas |
| Auth gagal / logout terus setelah pindah mode | `JWT_SECRET` beda antar mode |
| Fitur baru tidak muncul di mode docker | image basi — perlu `docker compose build` |
| `Docker Desktop is unable to start` | Docker Desktop belum siap; db/redis akan naik sendiri setelahnya (mereka `restart: unless-stopped`) |

## 3. Cek Dulu: `preflight.sh`

Selalu jalankan ini sebelum start/berpindah mode:

```bash
# Git Bash (JANGAN pakai `bash` telanjang bila itu WSL bash)
"C:\Program Files\Git\bin\bash.exe" scripts/preflight.sh

# atau lewat wrapper Windows
scripts\preflight.cmd

# validasi khusus satu mode
scripts\preflight.cmd --mode hybrid    # "boleh start dev host sekarang?"
scripts\preflight.cmd --mode docker    # "boleh docker compose up sekarang?"
```

Exit code `0` = siap, `1` = ada blocker. Script memeriksa: pemilik tiap port,
status container, konsistensi `JWT_SECRET`, kesegaran image, dan kesiapan mode.

> **Penting:** `bash` di PATH mesin ini bisa resolve ke **WSL bash**
> (`C:\Windows\System32\bash.exe`) yang tidak bisa melihat proses Windows atau
> Docker Desktop — dan akan melaporkan semua port "free" (menyesatkan). Script
> menolak jalan di shell itu; pakai Git Bash atau `preflight.cmd`.

## 4. Berpindah Mode

**Host → Docker:**

```bash
# 1. Pastikan preflight hijau
scripts\preflight.cmd --mode docker

# 2. Matikan dev host: Ctrl+C di terminal `pnpm dev` dan `go run`
#    (db & redis tetap jalan, tidak perlu disentuh)

# 3. Build ulang image (WAJIB bila ada commit baru) lalu up
docker compose up -d --build backend frontend

# 4. Verifikasi
curl -i http://localhost:8081/health/ready
```

**Docker → Host:**

```bash
# 1. Hentikan hanya app container (biarkan db & redis)
docker compose stop backend frontend

# 2. Jalankan dev host
cd backend && go run ./cmd/server      # terminal 1
cd frontend && pnpm dev                # terminal 2
```

> Bila kamu `docker compose down`, db & redis ikut berhenti — jalankan
> `docker compose up -d db redis` lagi (inilah alur hybrid normal).

## 5. Rahasia: `JWT_SECRET` per Mode

Tiap mode membaca file berbeda, jadi nilainya **harus sama di ketiganya**:

| File | Dibaca oleh |
|---|---|
| `.env` (root) | `docker compose` (mode docker) |
| `backend/.env` | `godotenv.Load()` (mode hybrid) |
| `frontend/.env.local` | `next dev` (mode hybrid) |

Aturan:

- Frontend memverifikasi cookie `ft_at` yang ditandatangani backend → **wajib sama**.
- Access token TTL **1 jam**, refresh token bersifat opaque (hash di DB, tidak
  memakai JWT secret) → mengganti secret aman; hanya sesi < 1 jam yang perlu
  login ulang. Tidak perlu menghapus baris `refresh_tokens`.
- Jangan pernah menaruh secret berbeda hanya karena beda mode tanpa alasan.

## 6. Pencegahan

1. **Satu mode pada satu waktu** untuk :3001 / :8081. db & redis selalu boleh
   jalan di container.
2. **`preflight.cmd --mode <x>`** sebelum start/berpindah.
3. **Selalu `--build`** saat naikkan mode docker (image 2026-10-11 pernah
   tertinggal 52 commit).
4. **`backend/.dockerignore`** menjaga `.env`/`.tmp` keluar dari image — jangan
   dihapus. `.env` yang ter-bake akan menimpa `DATABASE_URL`/`REDIS_URL` compose
   (karena `godotenv.Load()`) sehingga container menunjuk ke `localhost` (dirinya
   sendiri), bukan service `db`/`redis`.
5. Bila 404 halaman tetap muncul setelah kode benar → restart dev server atau
   `rm -rf .next && pnpm dev` (route table basi).

## 7. Referensi

- `../../../docker-compose.yml`
- `../../../scripts/preflight.sh` — implementasi cek
- `../README.md` — README root, bagian "Cara 1 / Cara 2"
- `../keamanan/AUTENTIKASI_OTORISASI.md` — detail cookie & token
