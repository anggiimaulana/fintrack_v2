# Arsitektur & Strategi Realtime: Server-Sent Events (SSE)

## 1. Topologi Arsitektur Realtime FinTrack v2

FinTrack v2 mengadopsi protokol Server-Sent Events (SSE) murni melalui modul backend `internal/realtime/` dengan backbone pesan terdistribusi Redis Pub/Sub. Arsitektur ini menjamin pesan server sampai ke peramban pengguna secara instan (< 100ms) di lingkungan multi-pod container.

```
 [ Browser Klien A ]         [ Browser Klien B ]
          │                           │
  (GET /stream Cookie)        (GET /stream Cookie)
          │                           │
          ▼                           ▼
  [ Backend Pod 1 ]           [ Backend Pod 2 ]
   (Broker SSE Hub)            (Broker SSE Hub)
          │                           │
          ▲                           ▲
          └───────────┬───────────────┘
                      │ Redis SUBSCRIBE
                      │ "realtime:events:*"
                      ▼
             [ Redis Cluster / PubSub ]
                      ▲
                      │ Redis PUBLISH
       ┌──────────────┴──────────────┐
       │                             │
 [ Payment Webhook ]        [ Super Admin Action ]
 (Status Sukses)            (Ubah Kuota / Plan)
```

---

## 2. Spesifikasi Endpoint Stream SSE

- **URL Endpoint**: `GET /api/v1/realtime/stream`
- **Metode**: `GET`
- **Autentikasi**: Wajib melalui HTTP-Only Cookie bernama `fintrack_token` (K-P07-04). Dilarang menyertakan token via query param string.
- **Header Response Wajib**:
  ```http
  HTTP/1.1 200 OK
  Content-Type: text/event-stream
  Cache-Control: no-cache, no-transform
  Connection: keep-alive
  X-Accel-Buffering: no
  Access-Control-Allow-Credentials: true
  ```

---

## 3. Format Penyiaran Pesan & Spesifikasi Event Domain

Seluruh event SSE FinTrack v2 mematuhi spesifikasi standar W3C Server-Sent Events dengan format envelope JSON terstruktur:

```http
id: evt_1773294801_001
event: payment_received
data: {"event":"payment_received","timestamp":1773294801,"payload":{"invoice_id":"INV-2026-0091","amount":150000,"plan_id":"plan_pro_monthly","status":"settled"}}

```

### Daftar Event Domain Terdaftar:

#### 1. `payment_received`
Dipancarkan saat webhook payment gateway berhasil memverifikasi pembayaran faktur pengguna.
```json
{
  "event": "payment_received",
  "timestamp": 1773294801,
  "payload": {
    "invoice_id": "INV-2026-0091",
    "payment_id": "pay_midtrans_8819",
    "amount": 150000,
    "plan_id": "plan_pro_monthly",
    "status": "settled",
    "period_end": "2026-11-11T23:59:59Z"
  }
}
```

#### 2. `plan_updated`
Dipancarkan saat super admin atau sistem mengubah status langganan atau kuota pengguna.
```json
{
  "event": "plan_updated",
  "timestamp": 1773294850,
  "payload": {
    "user_id": "usr_99182",
    "previous_plan": "free",
    "current_plan": "pro",
    "new_version": 4,
    "effective_immediately": true
  }
}
```

#### 3. `ai_endpoint_down`
Dipancarkan ke dashboard super admin saat circuit breaker mendeteksi kegagalan upstream 9router endpoint.
```json
{
  "event": "ai_endpoint_down",
  "timestamp": 1773294900,
  "payload": {
    "endpoint_id": "ep_local_20128",
    "endpoint_url": "http://localhost:20128",
    "failure_count": 5,
    "status": "circuit_open",
    "fallback_target": "ep_backup_openai"
  }
}
```

#### 4. `manual_approval_needed`
Dipancarkan ke panel super admin saat ada pembayaran transfer manual baru yang diunggah pengguna.
```json
{
  "event": "manual_approval_needed",
  "timestamp": 1773295100,
  "payload": {
    "proof_id": "prf_8827",
    "user_email": "budi@example.com",
    "amount": 500000,
    "bank_name": "BCA",
    "submitted_at": "2026-10-11T14:32:00Z"
  }
}
```

#### 5. `quota_reset`
Dipancarkan saat siklus langganan pengguna berakhir dan kuota direset (awal siklus baru, TZ Asia/Jakarta).
```json
{
  "event": "quota_reset",
  "timestamp": 1773295200,
  "payload": {
    "user_id": "usr_99182",
    "new_cycle_start": "2026-10-11T00:00:00+07:00",
    "new_cycle_end": "2026-11-10T23:59:59+07:00",
    "allocated_tokens": 50000
  }
}
```

---

## 4. Mekanisme Heartbeat & Pencegahan Timeout

Protokol HTTP streaming rentan terhadap pemutusan koneksi sepihak oleh reverse proxy (Nginx, AWS ALB, Cloudflare) bila tidak ada lalu lintas data selama 30-60 detik.
- **Heartbeat Ping**: Goroutine latar belakang server memancarkan komentar baris tunggal setiap **15 detik**:
  ```http
  : ping
  
  ```
- **Karakteristik Komentar SSE**: Baris yang diawali titik dua (`:`) diabaikan oleh parser `EventSource` standar peramban, tidak memicu handler `onmessage`, namun menjaga status koneksi TCP tetap hidup (*alive*).

---

## 5. Resiliensi Reconnect & Dukungan `Last-Event-ID`

Bila jaringan klien terputus (misalnya koneksi WiFi tidak stabil atau laptop memasuki mode sleep):
1. Browser otomatis mencoba menyambung kembali (`EventSource` bawaan).
2. Browser mengirimkan header request HTTP:
   ```http
   Last-Event-ID: evt_1773294801_001
   ```
3. Modul `internal/realtime/` menangkap header tersebut.
4. Server membaca daftar ring-buffer event pengguna dari Redis:
   - Key: `ft:prod:realtime:buffer:{user_id}` (Redis ZSET dengan score timestamp, TTL 10 menit).
5. Server mengirimkan kembali seluruh event yang memiliki ID lebih besar dari `Last-Event-ID` secara berurutan (*replay backlog*) sebelum melanjutkan penyiaran pesan baru.
6. Hal ini menjamin status UI tidak kehilangan event penting (*zero event drop*) selama durasi putus di bawah 10 menit.
