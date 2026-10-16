# Analisis Arsitektur & Skalabilitas Masa Depan: Bottlenecks & Penguatan Jangka Panjang

**Kode Dokumen**: AUDIT-ARCH-01  
**Klasifikasi**: Arsitektur Sistem, Kinerja Tinggi & Skalabilitas (Infrastructure Architecture & High-Scale Resilience)  

---

## 1. Konteks Skalabilitas FinTrack v2

FinTrack v2 dirancang dengan stack modern: Go 1.26 + Gin, pgxpool PostgreSQL 17 + pgvector (768d), Redis 7 (Pub/Sub & Versioned Caching), serta Next.js 16 + React 19.

Setelah pengujian beban Fase P08 membuktikan throughput 3.924 RPS pada AI Gateway dan latensi p95 sub-50ms pada skenario 500 RPS sustained, audit ini mengidentifikasi **bottleneck struktural yang berpotensi muncul ketika sistem bertumbuh ke skala 100.000+ pengguna aktif (MAU)** dan multi-pod horizontal scaling di Kubernetes/Docker Swarm.

---

## 2. Analisis Bottlenecks & Risiko Masa Depan

### 2.1 Bottleneck 1: Database Connection Pool Tanpa PgBouncer
- **Kondisi Saat Ini**:
  Di `backend/internal/infrastructure/db/postgres.go`, setiap instans backend Go menginisialisasi `pgxpool.Pool` dengan ukuran pool default 20–25 koneksi. PostgreSQL container dikonfigurasi dengan `max_connections = 100`.
- **Risiko Masa Depan**:
  Jika trafik meningkat dan pod backend di-scale horizontal menjadi 6–10 replika pod (misal via HPA di Kubernetes), total koneksi idle dari seluruh pod akan mencapai `10 * 25 = 250` koneksi. Hal ini akan **melampaui `max_connections` PostgreSQL**, menyebabkan error `FATAL: sorry, too many clients already` dan kegagalan seluruh pod baru yang melakukan startup.
- **Solusi Rekomendasi**:
  Pasang **PgBouncer** sebagai middleware connection pooler di depan PostgreSQL dalam mode `transaction pooling`.
  - Aplikasi Go membuka ratusan koneksi virtual ke PgBouncer.
  - PgBouncer hanya membuka 20–30 koneksi fisik tetap ke server PostgreSQL riil.
  - Skalabilitas pod backend dapat ditingkatkan hingga 50+ pod tanpa membebani memori proses PostgreSQL backend.

---

### 2.2 Bottleneck 2: Siklus Hidup Koneksi Long-Lived SSE (Ghost Connections)
- **Kondisi Saat Ini**:
  Modul Server-Sent Events (`internal/realtime/hub.go`) mengelola koneksi aktif pengguna ke channel broadcast.
- **Risiko Masa Depan**:
  Pada perangkat seluler atau jaringan yang mengalami pemutusan jaringan tiba-tiba (*airplane mode*, sinyal hilang tanpa paket TCP FIN/RST), koneksi SSE di sisi server dapat menggantung sebagai "koneksi hantu" (*ghost connection*) hingga heartbeat ticker gagal menulis beberapa siklus kemudian.
  Jika ribuan tab browser tertinggal dalam status ini, memori goroutine dan channel buffer dapat menumpuk (*slow memory leak*).
- **Solusi Rekomendasi**:
  1. Tambahkan waktu henti baca mutlak (*ReadTimeout / IdleTimeout*) pada level HTTP server Gin untuk koneksi streaming.
  2. Tambahkan background worker `Hub.ReapStaleConnections(interval 5m)` yang memindai koneksi yang tidak pernah merespons heartbeat ping lebih dari 45 detik dan menutup channel secara proaktif.

---

### 2.3 Bottleneck 3: Rate Limiting Terdistribusi (Multi-Pod Drift)
- **Kondisi Saat Ini**:
  `RateLimitMiddlewareByPath` di `backend/internal/delivery/http/middleware/rate_limit.go` menggunakan driver penyimpanan lokal `memory.NewStore()`.
- **Risiko Masa Depan**:
  Ketika sistem beroperasi di balik load balancer dengan 4 pod backend:
  - Batas rate limit login (misal 5 request/menit) dihitung secara independen di setiap pod.
  - Seorang penyerang *brute-force* dapat menyebarkan request ke 4 pod berbeda dan berhasil mengirimkan `4 * 5 = 20` percobaan per menit sebelum diblokir.
- **Solusi Rekomendasi**:
  Manfaatkan koneksi Redis yang sudah ada untuk mengaktifkan driver `redis.NewStore(client)` pada `RateLimitMiddlewareByPath` di lingkungan multi-pod produksi, dengan fallback ke `memory.NewStore()` hanya jika Redis mengalami kegagalan (*graceful degradation*).

---

### 2.4 Bottleneck 4: Pertumbuhan Index HNSW pgvector pada Skala 1 Juta+ Transaksi
- **Kondisi Saat Ini**:
  Tabel `transaction_embeddings` menggunakan indeks HNSW (`embedding vector_cosine_ops`) dengan dimensi 768.
- **Risiko Masa Depan**:
  Indeks HNSW menyimpan grafik ketetanggaan berdimensi tinggi langsung di dalam RAM PostgreSQL. Ketika jumlah transaksi mencapai 1.000.000 baris, memori indeks dapat melampaui alokasi `shared_buffers` / RAM container, menyebabkan penurunan tajam performa query pencarian vektor dari 8ms menjadi ratusan milidetik akibat disk I/O thrashing.
- **Solusi Rekomendasi**:
  1. Partisi tabel `transaction_embeddings` berdasarkan rentang waktu (`PARTITION BY RANGE (created_at)`), misalnya per kuartal atau per tahun.
  2. Batasi pencarian semantik vektor pengguna hanya pada data 12 bulan terakhir secara default, dengan opsi query arsip terpisah.
  3. Konfigurasi parameter HNSW PostgreSQL: `m = 16`, `ef_construction = 64` untuk efisiensi rasio memori-ke-akurasi.

---

### 2.5 Bottleneck 5: Partisi & Rotasi Tabel `audit_logs`
- **Kondisi Saat Ini**:
  Tabel `audit_logs` bertumbuh secara append-only setiap ada aksi sensitif, perubahan data, atau mutasi akun.
- **Risiko Masa Depan**:
  Dalam 1–2 tahun operasional, tabel audit logs dapat mencapai puluhan juta baris. Query filter audit logs di portal Super Admin akan mengalami degradasi kecepatan jika scanning index btree meluas.
- **Solusi Rekomendasi**:
  Terapkan partisi tabel berbasis waktu bulanan di PostgreSQL (`PARTITION BY RANGE (created_at)`) dan buat background job pemindahan log berusia > 2 tahun ke penyimpanan arsip dingin (*cold storage / S3 Parquet*).
