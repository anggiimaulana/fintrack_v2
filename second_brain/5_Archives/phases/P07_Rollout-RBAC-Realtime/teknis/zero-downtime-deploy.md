# Panduan & Prosedur Zero-Downtime Deployment

## 1. Definisi & Jaminan Ketersediaan (Zero-Downtime SLA)

FinTrack v2 melayani data pencatatan transaksi keuangan, sinkronisasi webhook pembayaran real-time, dan asisten AI. Setiap penghentian layanan terencana (*scheduled downtime*) menimbulkan risiko hilangnya webhook gateway pembayaran dan merusak pengalaman pengguna.

**Target Operasional P07**:
- **Downtime Selama Deployment**: **0 detik** (0 detik HTTP 5xx).
- **Packet Loss**: **0%** saat proses transisi pod/container dari versi N ke versi N+1.
- **Integritas Transaksi In-Flight**: Seluruh koneksi streaming SSE dan transaksi yang sedang diproses diselesaikan hingga tuntas sebelum pod lama dimatikan.

---

## 2. Pola Migrasi Skema Basis Data: Expand and Contract

Perubahan skema PostgreSQL tidak boleh memicu exclusive table lock yang mengunci pembacaan atau penulisan data. Perubahan skema wajib mengikuti pola 2-fase bertahap:

```
[ Waktu T0: Kondisi Awal ]
  Skema: [ id | name | old_column ]
  Aplikasi N: Membaca & Menulis 'old_column'
            │
            ▼
[ Waktu T1: Fase Expand ]
  Migrasi DB: ALTER TABLE tbl ADD COLUMN new_column (Nullable);
  Aplikasi N+1: Membaca 'old_column', Menulis ke 'old_column' DAN 'new_column' (Dual-Write)
            │
            ▼
[ Waktu T2: Backfill Data Latar Belakang ]
  Worker: UPDATE tbl SET new_column = transform(old_column) WHERE new_column IS NULL;
            │
            ▼
[ Waktu T3: Fase Transisi Baca ]
  Aplikasi N+2: Membaca 'new_column', Menulis ke 'new_column'
            │
            ▼
[ Waktu T4: Fase Contract ]
  Migrasi DB: ALTER TABLE tbl DROP COLUMN old_column;
  Aplikasi N+3: Skema final bersih, tanpa sisa kolom legacy
```

### Aturan Skrip Migrasi SQL:
1. **Penambahan Kolom**: Wajib `NULLABLE` atau memiliki `DEFAULT`. Jangan pernah menambahkan kolom `NOT NULL` tanpa nilai default pada tabel besar.
2. **Pembuatan Indeks**: Wajib menggunakan klausa konkurensi:
   ```sql
   CREATE INDEX CONCURRENTLY idx_transactions_user_cycle 
   ON transactions (user_id, created_at);
   ```
3. **Penghapusan Kolom**: Hanya boleh dijalankan setelah seluruh pod versi lama yang mengakses kolom tersebut telah berhenti beroperasi.

---

## 3. Graceful Shutdown & Drain Koneksi Backend Go

Pada aplikasi Gin backend, server wajib menangkap sinyal terminasi sistem operasi (`SIGTERM` dan `SIGINT`) untuk melakukan *graceful connection draining* selama maksimal 30 detik:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func runServer(router http.Handler) {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Goroutine latar belakang untuk mendengarkan port
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server gagal dijalankan: %v", err)
		}
	}()

	// Channel untuk menangkap sinyal shutdown OS
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[SHUTDOWN] Menerima sinyal terminasi, memulai graceful shutdown...")

	// Memberikan batas waktu 30 detik untuk menyelesaikan request aktif & koneksi SSE
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[SHUTDOWN] Server dipaksa berhenti dengan error: %v", err)
	}

	log.Println("[SHUTDOWN] Seluruh koneksi selesai, server berhenti dengan aman.")
}
```

---

## 4. Dual Health Gate Orchestrator (`/health/live` vs `/health/ready`)

Container orchestrator (Docker Swarm / Kubernetes / Reverse Proxy) memerlukan dua pemeriksaan terpisah untuk membedakan antara container yang sedang booting dan container yang siap menerima beban:

```go
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// /health/live: Liveness Probe
// Digunakan oleh orchestrator untuk menentukan apakah container perlu di-restart.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// /health/ready: Readiness Probe
// Digunakan untuk menentukan apakah container boleh dialiri trafik oleh load balancer.
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	// 1. Cek Konektivitas PostgreSQL & Ekstensi pgvector
	if err := h.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unready",
			"error": "database connection pool offline",
		})
		return
	}

	// 2. Cek Konektivitas Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unready",
			"error": "redis cache offline",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"db": "connected",
		"redis": "connected",
	})
}
```

---

## 5. Konfigurasi Rolling Deploy (Docker Compose / Ingress)

Untuk memastikan zero packet drop selama transisi deployment, konfigurasi service pada deployment manifest menggunakan parameter rolling update:

```yaml
version: '3.8'

services:
  backend:
    image: fintrack-backend:latest
    deploy:
      replicas: 2
      update_config:
        parallelism: 1
        delay: 15s
        order: start-first  # Menyalakan pod baru sebelum mematikan pod lama
        failure_action: rollback
        monitor: 10s
        max_failure_ratio: 0.1
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health/ready"]
      interval: 5s
      timeout: 3s
      retries: 3
      start_period: 10s
    stop_grace_period: 35s
```

### Mekanisme `order: start-first`:
1. Container `backend_v2` dibuat dan dinyalakan berdampingan dengan `backend_v1`.
2. Load balancer menguji endpoint `/health/ready` milik `backend_v2`.
3. Setelah lolos health check selama 10 detik, load balancer mulai mengalirkan trafik baru ke `backend_v2`.
4. Sinyal `SIGTERM` dikirim ke `backend_v1`.
5. `backend_v1` menuntaskan koneksi aktif dalam jendela 30 detik (`stop_grace_period`), lalu berhenti dengan aman.
6. **Hasil Akhir**: Klien tidak mengalami gangguan HTTP 502/504 Bad Gateway sepanjang proses rilis.

---

## Status Implementasi (Slice D1, 2026-10-11)

### Health gates — implementasi

| Endpoint | Perilaku | Alasan |
|---|---|---|
| `/health/live` | 200, **tanpa I/O** | Proses hidup tidak boleh bergantung pada dependensi; kalau tidak, orchestrator me-restart proses sehat hanya karena DB blip |
| `/health/ready` | 200 bila DB sehat; **503** bila DB gagal | PostgreSQL hard dependency: tanpa DB instance tidak bisa melayani apa pun |
| `/health/ready` (Redis) | Redis mati → **200** dengan `redis: degraded` | Redis **opsional** di codebase ini (`REDIS_URL` boleh kosong, cache graceful degradation); menandai pod tidak siap karena cache outage akan membuang kapasitas yang tidak diperlukan |
| `/health` | alias `/health/live` | Kompatibilitas dokumentasi & probe ad-hoc |

`/health/ready` memeriksa **pgvector** juga (`pg_extension`), karena jalur embedding bergantung
padanya. Payload memuat `status` + `latency_ms` per dependensi.

### Graceful shutdown

- Drain `10s → 30s`, dapat dikonfigurasi `SHUTDOWN_TIMEOUT` (detik).
- Compose: `healthcheck` untuk service `backend` (memakai `/health/ready`) dan
  `stop_grace_period: 30s` — harus **≥** `SHUTDOWN_TIMEOUT`, jika tidak Docker akan SIGKILL
  sebelum drain selesai.

### Gap yang belum dikerjakan (dicatat, tidak diklaim)

1. **Template Kubernetes** (`deploy/k8s/deployment.yaml`, ingress, readiness probe) **tidak
   dibuat** — direktori `deploy/` tidak ada di repo. AC TASK-P07-017 menyebutnya.
2. **Uji rolling restart pada 200 RPS tanpa 5xx** tidak dijalankan. Itu uji beban terkoordinasi
   (butuh k6/vegeta + stack hidup), bukan unit test; mengklaimnya tanpa menjalankan akan keliru.
   Perintah k6 yang ada (`tester/k6-smoke.js`) menguji jalur auth, bukan rolling restart.
3. **`ReadTimeout: 15s` pada `http.Server`** tidak diubah. Untuk endpoint SSE yang berumur
   panjang, `ReadTimeout` mengikat pembacaan request (bukan penulisan respons), jadi stream tidak
   terpotong — tetapi bila nanti ada kebutuhan menahan koneksi lebih lama, ini perlu ditinjau
   bersama `WriteTimeout`.
