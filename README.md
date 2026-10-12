# FinTrack v2 — Sistem Manajemen & Pencatatan Keuangan Cerdas

Platform manajemen keuangan modern, cerdas, dan siap produksi (production-ready) yang mengintegrasikan AI Multi-Model Gateway, Sistem Pendukung Keputusan (SPK), manajemen langganan (B2C Billing), Server-Sent Events (SSE) realtime, serta kepatuhan ketat terhadap regulasi UU Pelindungan Data Pribadi (UU No. 27/2022) dan perpajakan (UU KUP No. 28/2007).

---

## 🚀 Fitur Unggulan

- **AI Multi-Model Gateway Terpusat (P03/P07)**:
  - Routing dinamis dengan algoritma Smooth Weighted Round-Robin (SWRR) antar-combo.
  - Failover berjenjang di dalam combo (`priority ASC`) dan lintas-combo.
  - Circuit Breaker per-endpoint dengan isolasi kegagalan otomatis.
  - Dukungan multi-model: DeepSeek-R1 (penalaran mendalam), Gemini 2.5 Flash (chat & OCR vision), dan Gemini Embedding 768d (pgvector RAG).
- **Sistem Pendukung Keputusan Finansial (SPK)**:
  - Analisis kesehatan finansial pengguna berbasis data transaksi nyata menggunakan metode pembobotan multi-kriteria (metode 50/30/20, rasio tabungan, tren arus kas).
- **Billing, Langganan & Pembayaran (P05/P06/P09)**:
  - Tingkatan paket: **Free**, **Pro** (Rp 39.000/bln), dan **Business** (Rp 99.000/bln).
  - Integrasi Midtrans Snap (kartu kredit, QRIS, GoPay, VA) & transfer manual dengan alur unggah bukti bayar.
  - Portal Super Admin untuk verifikasi pembayaran, persetujuan/penolakan instan, dan refund transaksional.
- **Entitlement & RBAC Enforcement (P07)**:
  - Feature gating (`ai_chat`, `ocr_receipt`, `voice_input`, `reports_export`, dll.).
  - Limit enforcement numerik per siklus (`ai_tokens_per_cycle`, `max_transactions`, `ocr_scans_per_cycle`).
  - Cache entitlement berversi di Redis dengan sinkronisasi instan via Redis Pub/Sub tanpa restart pod.
- **Realtime Server-Sent Events (SSE) (P07)**:
  - Fan-out broadcast multi-pod via Redis Pub/Sub (`realtime:u:{id}` & `realtime:admins`).
  - Heartbeat 15s, pembersihan koneksi mengambang otomatis, dan dukungan resume `Last-Event-ID`.
- **Keamanan & Kepatuhan Regulasi Penuh (P08/P09)**:
  - **UU PDP No. 27/2022**: Hak Hapus (*Right to Erasure*) swalayan (`DELETE /api/v1/me`) dan Hak Akses (*Data Portability*) ekspor JSON (`GET /api/v1/user/export-data`).
  - **UU KUP No. 28/2007 Pasal 28**: Retensi data faktur dan pembayaran selama 10 tahun dengan teknik pseudonimisasi permanen (*cryptographic shredding*).
  - **PCI-DSS SAQ A**: Server aplikasi zero-contact dengan data kartu kredit/CVV.
  - **Sanitasi Error Universal**: Seluruh error 4xx/5xx disanitasi otomatis; tidak pernah mengekspos query SQL, struct Go, SQLSTATE, atau stack trace ke klien.
  - **Anti-XSS, Anti-SSRF & Anti-CSRF**: Sanitasi skema URL Markdown, validasi BaseURL IP private/metadata, dan origin checking berbasis cookie.

---

## 🛠️ Tech Stack

| Layer | Komponen & Teknologi | Keterangan |
|---|---|---|
| **Frontend** | Next.js 16 (App Router), React 19, TypeScript | Neo-brutalist styling, Tailwind CSS, TanStack Query v5, TanStack Table v8, Lucide Icons |
| **Backend** | Go 1.26, Gin Web Framework | Clean Architecture, pgxpool v5, goredis v9, Zap logger, Prometheus client |
| **Database** | PostgreSQL 17 + `pgvector` | HNSW cosine index 768 dimensi, migrasi bernomor aditif & idempoten |
| **Cache & Pub/Sub** | Redis 7 (Alpine) | Versioned cache, sliding window rate limiter, Pub/Sub channel distribution |
| **Observability** | Prometheus Metrics (`/metrics`), Health Gates (`/health/ready`, `/health/live`) | Four Golden Signals (Latency, Traffic, Errors, Saturation) |

---

## 📁 Struktur Monorepo

```
fintrack_v2/
├── backend/            # REST API Go, Usecase, Domain, Gateway AI & Repository
│   ├── cmd/server/     # Entry point server HTTP (`main.go`)
│   ├── internal/       # Modul inti: admin, aigateway, billing, payment, realtime, dll.
│   └── tests/          # Suite pengujian integrasi E2E (P06, P07, P08, P09)
├── frontend/           # Aplikasi Web Next.js (App Router)
│   ├── app/            # Halaman rute: (dashboard)/user, (dashboard)/admin, super-admin, auth
│   ├── components/     # Komponen UI: data-table, layout, ai, shared, ui (shadcn)
│   ├── hooks/          # React custom hooks & TanStack Query integrations
│   └── lib/            # Axios API client, auth client, formatting, utilities
├── database/           # Skema, migrasi, script dump & seed database
│   ├── migrations/     # Migrasi SQL bernomor urut (`0001_` s.d. `0013_`)
│   ├── scripts/        # `migrate.sh` dan `dump.sh`
│   └── dump/           # Snapshot PostgreSQL terkompresi (`fintrack_v2.sql` & `.dump`)
├── second_brain/       # Knowledge base arsitektur, log aktivitas, tracking WBS & audit
├── scripts/loadtest/   # Skrip pengujian beban k6 (concurrency, replay, sustained load)
├── docker-compose.yml  # Konfigurasi orkestrasi lokal (db, redis, backend, frontend)
├── .env.example        # Template konfigurasi environment variable
└── README.md
```

---

## ⚡ Panduan Instalasi & Menjalankan Sistem

### Prasyarat Sistem
- **Docker & Docker Compose** (disarankan untuk kemudahan setup).
- **Go 1.24+** (jika menjalankan backend secara lokal tanpa container).
- **Node.js 20+** & **pnpm 9+** (jika menjalankan frontend secara lokal).

---

### Cara 1: Menggunakan Docker Compose (Direkomendasikan)

1. **Clone Repositori**:
   ```bash
   git clone https://github.com/anggiimaulana/fintrack_v2.git
   cd fintrack_v2
   ```

2. **Siapkan Konfigurasi Lingkungan (`.env`)**:
   ```bash
   cp .env.example .env
   ```
   *Edit file `.env` dan sesuaikan kunci rahasia (`JWT_SECRET`, `ENCRYPTION_KEY`, `APP_DB_PASSWORD`).*

3. **Jalankan Seluruh Stack**:
   ```bash
   docker compose up --build -d
   ```

4. **Verifikasi Kesiapan Layanan**:
   ```bash
   # Cek status kesehatan backend (harus mengembalikan HTTP 200)
   curl -i http://localhost:8080/health/ready
   ```

5. **Akses Aplikasi**:
   - **Frontend UI**: [http://localhost:3001](http://localhost:3001)
   - **Backend API**: [http://localhost:8080/api/v1](http://localhost:8080/api/v1)
   - **Prometheus Metrics**: [http://localhost:8080/metrics](http://localhost:8080/metrics)
   - **Database PostgreSQL**: `localhost:5435` (User: `postgres`, DB: `fintrack`)
   - **Redis**: `localhost:6380`

---

### Cara 2: Menjalankan Secara Lokal (Development Mode)

#### 1. Jalankan Infrastruktur Database & Redis
Gunakan Docker hanya untuk database dan redis:
```bash
docker compose up -d db redis
```

#### 2. Jalankan Migrasi Database
```bash
bash database/scripts/migrate.sh --seed
```
*Script ini akan mengeksekusi migrasi bernomor (`0001` s/d `0013`) dan menyuntikkan data seed paket langganan serta akun demo.*

#### 3. Jalankan Backend (Go)
```bash
cd backend
go run ./cmd/server
```
Backend akan aktif di `http://localhost:8080`.

#### 4. Jalankan Frontend (Next.js)
Buka terminal baru:
```bash
cd frontend
pnpm install
pnpm dev
```
Frontend akan aktif di `http://localhost:3000` (atau port 3001).

---

## 🧪 Pengujian & Quality Gates

Seluruh kode produksi FinTrack v2 wajib mematuhi standar gerbang kualitas tanpa kompromi:

### Backend Testing (Go)
```bash
cd backend

# 1. Static Analysis / Vet
go vet ./...

# 2. Kompilasi Seluruh Paket
go build ./...

# 3. Unit Test Seluruh Paket
go test ./...

# 4. Suite Pengujian Integrasi & E2E (Membutuhkan DB & Redis Aktif)
REDIS_URL="redis://localhost:6380/0" \
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5435/fintrack_test?sslmode=disable" \
go test -count=1 ./tests/
```

### Frontend Testing (Next.js)
```bash
cd frontend

# 1. Linting ESLint (0 Error Wajib)
pnpm lint

# 2. Unit & Accessibility Tests (Vitest + axe-core)
pnpm test

# 3. Production Build Compilation (Turbopack)
pnpm build
```

---

## 🔒 Port & Endpoints Kunci

| Port | Layanan | Deskripsi |
|---|---|---|
| `3001` / `3000` | Frontend Next.js | Antarmuka Pengguna & Dasbor Super Admin |
| `8080` | Backend API Go | REST API Utama (`/api/v1`) |
| `5435` | PostgreSQL 17 | Basis Data Relasional + pgvector (port container `5432`) |
| `6380` | Redis 7 | Caching, Pub/Sub, Buffer SSE (port container `6379`) |

### Endpoint Penting
- `GET /health/live`: Liveness probe (tanpa beban I/O).
- `GET /health/ready`: Readiness probe (memeriksa latensi PostgreSQL, pgvector, dan Redis).
- `GET /metrics`: Metrik Prometheus Four Golden Signals & transaksi bisnis.
- `GET /api/v1/realtime/stream`: Aliran Server-Sent Events (SSE) realtime.
- `POST /api/v1/payments/checkout`: Inisiasi checkout pembayaran paket langganan.
- `POST /api/v1/payments/webhook/midtrans`: Webhook verifikasi transaksi payment gateway.
- `DELETE /api/v1/me`: Hak hapus akun mandiri terverifikasi (UU PDP Pasal 43).
- `GET /api/v1/user/export-data`: Ekspor arsip data pribadi portabel (UU PDP Pasal 21).

---

## 📚 Dokumentasi Terkait

Seluruh dokumentasi teknis, keputusan arsitektur (ADR), runbook insiden, dan audit tersimpan di folder `second_brain/`:
- **Log Aktivitas Operasional**: [`second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`](second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md)
- **Hasil Audit Komprehensif**: [`second_brain/0_Inbox/audit/2026-10-12_Audit-Komprehensif-Keamanan-Logic-Regulasi/`](second_brain/0_Inbox/audit/2026-10-12_Audit-Komprehensif-Keamanan-Logic-Regulasi/)
- **Runbook Penanganan Insiden**: [`second_brain/2_Projects/P08_Hardening-GoLive/teknis/runbook-incidents.md`](second_brain/2_Projects/P08_Hardening-GoLive/teknis/runbook-incidents.md)
- **Checklist Kesiapan Go-Live**: [`second_brain/2_Projects/P08_Hardening-GoLive/teknis/go-live-checklist.md`](second_brain/2_Projects/P08_Hardening-GoLive/teknis/go-live-checklist.md)
- **Panduan Migrasi Database**: [`database/README.md`](database/README.md)

---

## 📄 Lisensi & Kontribusi

Proyek ini dikembangkan secara tertutup untuk kebutuhan produksi FinTrack. Lihat panduan kerja di `.agents/rules/workspace_rules.md`.
