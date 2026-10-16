# Chaos Engineering, Failover & Circuit Breaker - FinTrack v2

Dokumen ini mendokumentasikan metodologi chaos testing, arsitektur failover multi-endpoint AI Gateway (9router pool), ketahanan sirkuit pemutus (*circuit breaker*), serta pengujian kehandalan sistem FinTrack v2 saat terjadi anomali infrastruktur kritis.

---

## 1. Arsitektur Resiliensi AI Gateway & Multi-Endpoint Pool

### 1.1 Komponen & Topologi
Sistem FinTrack v2 beroperasi dengan pool multi-endpoint AI Gateway upstream yang dikelola melalui tabel `ai_endpoints`:
- **Endpoint Primer**: Instans 9router lokal (`http://localhost:20128/v1`), prioritas bobot 100.
- **Endpoint Sekunder / Cadangan**: Node 9router redundan atau cloud AI proxy cadangan, prioritas bobot 50.
- **Circuit Breaker Internal**: Modul Go `internal/aigateway/circuitbreaker` memonitor tingkat kegagalan dan latensi setiap endpoint secara independen.

```
       [ Client Request: AI Chat / Embedding ]
                        │
                        ▼
            [ FinTrack Go Backend ]
                        │
               [ AI Router Engine ]
           (Pilih endpoint aktif & CLOSED)
              ┌─────────┴─────────┐
              ▼                   ▼
     [ Endpoint Primer ]  [ Endpoint Sekunder ]
      (9router Node 1)     (9router Node 2)
      Port: 20128          Port: 20129
         [HEALTHY]            [HEALTHY]
```

### 1.2 State Machine Circuit Breaker
Circuit breaker mengimplementasikan Finite State Machine 3-state:
1. **CLOSED (Normal)**: Seluruh request dialirkan ke endpoint ini. Jika terjadi 5 error beruntun (5xx atau timeout > 10s), state berubah menjadi **OPEN**.
2. **OPEN (Isolasi Kegagalan)**: Tidak ada request yang dikirim ke endpoint ini selama cooldown 30 detik. Request langsung dialihkan ke endpoint cadangan berikutnya di dalam pool.
3. **HALF-OPEN (Pengujian Pemulihan)**: Setelah cooldown 30 detik, sistem mengizinkan 1 request uji coba (*canary test*). Jika sukses, sirkuit kembali ke **CLOSED**; jika gagal, sirkuit kembali **OPEN** selama 60 detik.

---

## 2. Skenario Chaos 1: Terminasi Mendadak Node Primer 9router Under Load

### 2.1 Prosedur Uji
1. Skrip k6 menginjeksi beban konstan 300 RPS ke endpoint `/api/v1/ai/chat`.
2. Pada detik ke-30 beban berjalan, proses node primer 9router (port 20128) dihentikan secara paksa (`kill -9`).
3. Sistem diamati untuk mengukur:
   - Waktu deteksi kegagalan.
   - Kecepatan pengalihan rute (*failover latency*).
   - Persentase request yang gagal dialirkan ke klien.

### 2.2 Hasil Pengamatan & Data Empiris
| Parameter | Hasil Observasi | Target / Ambang Batas | Status |
|---|---|---|:---:|
| Total Request selama Kegagalan | 9.000 request | - | - |
| Request Berhasil (HTTP 200) | 8.998 request (99.97%) | >= 99.9% | PASSED |
| Request Gagal (HTTP 502/504) | 2 request (0.02%) | <= 0.1% | PASSED |
| Waktu Transisi Sirkuit Primer ke OPEN | 180 ms | <= 500 ms | PASSED |
| Durasi Failover ke Node Sekunder | 12 ms | <= 50 ms | PASSED |
| Lonjakan Latensi p95 saat Transisi | +142 ms | <= 350 ms | PASSED |

### 2.3 Mekanisme Auto-Retry Transparan
Ketika endpoint primer mengembalikan error koneksi terputus (`connection refused`), interceptor client HTTP Go FinTrack secara otomatis mengulang request satu kali (*single retry*) ke endpoint sekunder dalam pool sebelum mengembalikan respons ke klien. Hal ini menjelaskan mengapa hanya 2 request yang sempat gagal (terjadi tepat saat socket TCP terputus di tengah streaming).

---

## 3. Skenario Chaos 2: Redis Network Partition & Crash Simulation

### 3.1 Hipotesis & Risiko
Redis digunakan FinTrack v2 untuk:
- Token rate limiting & transient locking idempotensi webhook.
- Invalidasi cache in-memory berversi lintas pod via Redis Pub/Sub.
- Sesi cache sementara.

*Risiko*: Jika Redis down atau mengalami partisi jaringan (*split-brain*), aplikasi tidak boleh berhenti beroperasi (*panic*), melainkan harus mendegradasi layanan secara elegan (*graceful degradation*).

### 3.2 Prosedur Pengujian
1. Container Redis dihentikan mendadak (`docker stop fintrack-redis`).
2. Backend menerima request autentikasi, pengecekan hak entitlement, dan pembuatan order pembayaran.
3. Jaringan Redis dipulihkan kembali (`docker start fintrack-redis`) setelah 60 detik.

### 3.3 Hasil Pengamatan & Log Sistem
1. **Fallback Caching**:
   - Backend mendeteksi hilangnya koneksi Redis dan memancarkan log:
     `WARN [realtime/cache] Redis connection lost. Falling back to local in-memory cache and direct PostgreSQL query.`
   - Pengecekan entitlement langsung membaca dari tabel `plan_entitlements` PostgreSQL. Latensi meningkat tipis dari 1.2ms (Redis) menjadi 4.8ms (PostgreSQL query), namun seluruh transaksi tetap berjalan 100% normal.
2. **Fallback Rate Limiting**:
   - Middleware rate limiter beralih ke struktur data *in-memory sliding window* lokal berbasis sync.Map per-instans.
3. **Auto-Reconnection**:
   - Background connection manager mencoba reconnect dengan exponential backoff setiap 5 detik.
   - Begitu Redis kembali online, koneksi tersambung ulang secara otomatis tanpa intervensi manual atau restart aplikasi:
     `INFO [realtime/cache] Redis connection restored. In-memory fallback deactivated.`

---

## 4. Skenario Chaos 3: Saturasi Connection Pool Database PostgreSQL

### 4.1 Prosedur Uji
- Konfigurasi database: `max_connections = 100`, backend pool `max_open = 80`.
- Diinjeksikan 1.000 VU melakukan query analisis laporan keuangan kompleks secara bersamaan selama 2 menit untuk menghabiskan pool koneksi.

### 4.2 Hasil Observasi & Proteksi
- Backend FinTrack menerapkan timeout query ketat via `context.WithTimeout(ctx, 3*time.Second)`.
- Saat pool mencapai 80 koneksi, request yang menunggu antrian pool melebihi 2 detik ditolak dengan respons ramah HTTP 503 (`database_busy_try_later`), mencegah penumpukan memory exhaust pada PostgreSQL.
- Server PostgreSQL tetap responsif dan tidak mengalami crash OOM (*Out Of Memory*).

---

## 5. Ringkasan Matriks Ketahanan Bencana

| Komponen Infrastruktur | Modus Kegagalan | Dampak ke Pengguna Akhir | Mekanisme Pemulihan |
|---|---|---|---|
| **AI Gateway Node 1** | Crash / Process Killed | Tidak ada gangguan (0.02% glitch) | Circuit Breaker OPEN -> Auto-failover ke Node 2 dalam 180ms |
| **Seluruh AI Gateway** | Upstream Provider Mati | Fitur AI nonaktif sementara (HTTP 503), Fitur Keuangan/Catatan tetap 100% aktif | Graceful degradation, pesan UI ramah pengguna |
| **Redis Cache** | Crash / Split-Brain | Latensi meningkat 3-5ms | Fallback otomatis ke in-memory cache lokal & direct PostgreSQL |
| **Payment Gateway Webhook** | Jaringan Flapping / Replay | Tidak ada mutasi ganda | Kunci unik idempotency PostgreSQL menolak callback duplikat |
