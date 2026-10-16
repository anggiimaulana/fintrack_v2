# Dokumen Teknis — Observability Baseline & Four Golden Signals

## 1. Ikhtisar Arsitektur
Observability pada FinTrack v2 dirancang untuk memberikan transparansi total terhadap performa runtime, utilisasi sumber daya, kesehatan layanan, dan anomali sistem secara instan.

Standar pemantauan mengadopsi prinsip **Four Golden Signals (Google SRE)**:
1. **Latency**: Durasi penanganan HTTP request dan waktu respons dependensi eksternal (Redis/PostgreSQL/AI).
2. **Traffic**: Volume permintaan yang masuk (HTTP Requests Per Second / RPS).
3. **Errors**: Rasio kegagalan permintaan (HTTP 5xx, koneksi timeout, database error).
4. **Saturation**: Tingkat utilisasi antrean buffer, pool koneksi, dan kapasitas memori Redis.

---

## 2. Metrik Standar Prometheus di Backend Go

Instrumentasi backend menggunakan package resmi `github.com/prometheus/client_golang/prometheus`.

### 2.1. Taksonomi Metrik

| Nama Metrik | Tipe | Label | Deskripsi |
|---|---|---|---|
| `http_requests_total` | Counter | `method`, `path`, `status` | Jumlah kumulatif request HTTP yang diproses. |
| `http_request_duration_seconds` | Histogram | `method`, `path` | Distribusi latensi HTTP request (bucket: 0.005s s/d 10s). |
| `redis_operations_total` | Counter | `command`, `status` | Total eksekusi perintah Redis (success/error). |
| `audit_queue_depth` | Gauge | - | Jumlah event audit yang sedang mengantre di buffer Go channel. |
| `audit_events_processed_total` | Counter | `status` | Total event audit yang berhasil atau gagal di-flush ke DB. |
| `feature_flag_evaluations_total` | Counter | `flag_key`, `result` | Frekuensi evaluasi feature flag runtime (true/false). |
| `token_consumption_total` | Counter | `user_id`, `model`, `modality` | Akumulasi penggunaan token AI (disiapkan untuk P03/P04). |
| `payment_webhook_events_total` | Counter | `gateway`, `status` | Event webhook pembayaran yang diterima (disiapkan untuk P05). |

---

## 3. Implementasi Middleware Instrumentasi Go

```go
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total jumlah permintaan HTTP yang masuk.",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Distribusi latensi respons HTTP dalam satuan detik.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	AuditQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "audit_queue_depth",
			Help: "Jumlah event audit saat ini dalam antrean memori.",
		},
	)
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		HttpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HttpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
```

---

## 4. Keamanan Endpoint `/metrics`
Endpoint metrik diekspos di rute `/metrics` menggunakan `promhttp.Handler()`.
**Aturan Keamanan Akses**:
- Endpoint `/metrics` **DILARANG** diekspos ke internet publik.
- Rute ini hanya dapat diakses melalui antarmuka localhost (`127.0.0.1`) atau jaringan internal Docker (`fintrack_net`) oleh Prometheus scraper container.
- Middleware otorisasi menyaring IP pengakses dan melempar `403 Forbidden` jika diakses dari IP non-internal.

---

## 5. Ambang Batas Peringatan (Alerting Thresholds)

Berdasarkan Four Golden Signals, didefinisikan alert rekomendasi operasional:

| Sinyal | Kondisi Peringatan | Tingkat Keparahan | Tindakan Operasional |
|---|---|---|---|
| **Latency** | p95 HTTP duration > 1.500ms selama 5 menit | Warning | Periksa slow query PostgreSQL & thread Redis. |
| **Errors** | HTTP 5xx error rate > 1% dari total traffic | Critical | Cek error log backend, potensi database connection leak. |
| **Saturation** | `audit_queue_depth` > 1.800 (90% kapasitas buffer) | High | Worker database kewalahan, cek disk I/O PostgreSQL. |
| **Saturation** | Redis Memory > 220MB (85% batas 256MB) | High | Evaluasi TTL key atau naikkan alokasi memori Redis. |
