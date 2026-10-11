# Runbook Penanganan Insiden Operasional Produksi - FinTrack v2

Runbook ini memuat panduan langkah demi langkah bagi tim On-Call SRE dan Software Engineer dalam menangani insiden operasional kritis pada lingkungan produksi FinTrack v2.

---

## Klasifikasi Tingkat Keparahan Insiden (Severity Levels)
- **P1 (Kritis)**: Sistem down total, kegagalan transaksi pembayaran massal, atau kebocoran data sensitif. Waktu respons: < 5 menit. Waktu mitigasi: < 30 menit.
- **P2 (Tinggi)**: Salah satu modul utama degradasi berat (misal AI Gateway down total meski fitur pembukuan aktif, webhook pembayaran delay > 15 menit). Waktu respons: < 15 menit. Waktu mitigasi: < 2 jam.
- **P3 (Sedang)**: Penurunan performa latensi parsial, metrik visual dasbor error. Waktu respons: < 1 jam.
- **P4 (Rendah)**: Masalah minor tanpa dampak ke pengguna akhir.

---

## RUNBOOK-01: AI Gateway Down Total / Upstream 9router Exhausted (P2)

### 1. Indikasi & Alert
- Alert PagerDuty: `ALERT: AIGatewayAllEndpointsDown` atau lonjakan HTTP 503 pada rute `/api/v1/ai/*`.
- Metrik Grafana: `ai_gateway_endpoint_circuit_state{state="OPEN"}` bernilai 1 untuk semua target endpoint.

### 2. Diagnosis Cepat
1. Cek kesehatan container 9router lokal di server produksi:
   ```bash
   docker ps | grep 9router
   curl -I http://localhost:20128/v1/models
   ```
2. Cek kuota upstream API provider atau GPU worker:
   ```bash
   curl -X GET http://localhost:20128/health
   ```
3. Cek log AI Gateway FinTrack:
   ```bash
   docker logs --tail 100 fintrack-backend | grep "aigateway"
   ```

### 3. Prosedur Mitigasi
1. **Langkah 1: Aktifkan Cadangan Darurat AI Route**:
   Jika 9router port 20128 mati atau out of quota, arahkan rute AI melalui panel Super Admin atau SQL langsung ke endpoint fallback darurat:
   ```sql
   UPDATE ai_routes 
   SET endpoint_id = (SELECT id FROM ai_endpoints WHERE name = 'Emergency-Fallback-Cloud' AND is_active = TRUE)
   WHERE is_active = TRUE;
   ```
2. **Langkah 2: Restart Service 9router**:
   ```bash
   docker restart fintrack-9router
   # Tunggu 10 detik lalu verifikasi heartbeat
   curl http://localhost:20128/v1/models
   ```
3. **Langkah 3: Reset Sirkuit Circuit Breaker Manual**:
   Kirim perintah reset circuit breaker via Redis CLI:
   ```bash
   redis-cli PUBLISH fintrack:circuit:reset "all"
   ```
4. **Langkah 4: Graceful Degradation (Jika seluruh provider mati)**:
   Aktifkan feature flag pembatasan fitur AI agar antarmuka Next.js menampilkan banner edukatif:
   ```sql
   UPDATE feature_flags SET is_enabled = FALSE WHERE name = 'ai_chat_enabled';
   ```

---

## RUNBOOK-02: Kegagalan Pembayaran Massal & Webhook Delayed (P1)

### 1. Indikasi & Alert
- Alert PagerDuty: `ALERT: PaymentWebhookDropSpike` atau tingkat kegagalan konversi pembayaran > 40% dalam 15 menit.
- Pelanggan melapor telah membayar via QRIS/VA namun status akun belum aktif.

### 2. Diagnosis Cepat
1. Periksa ketersediaan webhook endpoint:
   ```bash
   curl -I -X POST https://api.fintrack.id/api/v1/payment/webhook/midtrans
   ```
2. Periksa status antrian payment lock Redis:
   ```bash
   redis-cli KEYS "payment_lock:*"
   ```
3. Cek log verifikasi signature webhook:
   ```bash
   docker logs --tail 200 fintrack-backend | grep "SECURITY_WEBHOOK_SIGNATURE_MISMATCH"
   ```

### 3. Prosedur Mitigasi & Rekonsiliasi
1. **Langkah 1: Periksa Status Gateway Eksternal**:
   Buka portal dashboard merchant Midtrans (`dashboard.midtrans.com`) atau Xendit (`dashboard.xendit.co`). Verifikasi apakah ada gangguan sistemik pada core gateway atau perbankan partner.
2. **Langkah 2: Eksekusi Tool Rekonsiliasi Pembayaran**:
   Jalankan background job sinkronisasi status pembayaran manual untuk invoice yang menggantung (*PENDING*):
   ```bash
   # Jalankan CLI rekonsiliasi batch
   docker exec -it fintrack-backend /app/fintrack-admin payments reconcile --hours=3
   ```
3. **Langkah 3: Re-trigger Notifikasi Webhook**:
   Pada portal Midtrans/Xendit, filter transaksi bersaldo sukses yang belum terkirim notifikasinya, lalu klik **"Resend Webhook"**. Mekanisme idempotensi FinTrack v2 akan memproses pembayaran dengan aman tanpa risiko dobel kredit.

---

## RUNBOOK-03: Rotasi Darurat Master Encryption Key (P1/P2)

### 1. Indikasi Kebutuhan
- Terjadi dugaan kebocoran variabel lingkungan server produksi atau perintah audit keamanan berkala (setiap 180 hari).

### 2. Prosedur Rotasi Tanpa Gangguan Layanan (Zero Downtime)
1. **Langkah 1: Generate Master Key Baru (32 Byte Base64)**:
   ```bash
   openssl rand -base64 32
   # Contoh hasil: 9uKjFw...==
   ```
2. **Langkah 2: Konfigurasi Nilai Dual-Key pada Environment**:
   Di server atau secrets manager (Docker Swarm / Kubernetes Secrets / `.env.production`):
   - Set `APP_ENCRYPTION_MASTER_KEY_PREVIOUS` = `<NILAI_KUNCI_LAMA>`
   - Set `APP_ENCRYPTION_MASTER_KEY_CURRENT` = `<NILAI_KUNCI_BARU>`
3. **Langkah 3: Rolling Deploy Backend**:
   Lakukan rolling update kontainer backend (`fintrack-backend`). Backend akan mulai mengenkripsi data baru dengan kunci baru, dan masih dapat membaca data lama dengan kunci previous.
4. **Langkah 4: Jalankan Migrasi Re-Enkripsi Database**:
   ```bash
   docker exec -it fintrack-backend /app/fintrack-admin secrets re-encrypt-all
   ```
   *Output yang diharapkan*: `Processed 124 records in ai_endpoints, 8 records in payment_configs. Zero failures.`
5. **Langkah 5: Lepaskan Kunci Lama**:
   Setelah seluruh data lama ter-enkripsi ulang, kosongkan variabel `APP_ENCRYPTION_MASTER_KEY_PREVIOUS` dan restart pod backend.

---

## RUNBOOK-04: Pembersihan & Resinkronisasi Cache Basi Lintas Pod (P3)

### 1. Indikasi
- Perubahan paket langganan, kuota limit, atau role super admin di database tidak terefleksi pada sebagian pengguna karena cache in-memory lokal pod tidak tersinkronisasi.

### 2. Diagnosis
Periksa apakah worker subscriber Redis Pub/Sub pada backend terputus:
```bash
docker logs fintrack-backend | grep "pubsub"
```

### 3. Tindakan Korektif
1. **Kirim Perintah Broadcast Flush Cache via Redis Pub/Sub**:
   ```bash
   redis-cli PUBLISH fintrack:cache:invalidate '{"scope":"all_entitlements","reason":"admin_force_resync"}'
   ```
2. **Verifikasi Respon Pod**:
   Log backend pada seluruh pod akan mencatat:
   `INFO [realtime/cache] Received broadcast cache invalidation for scope: all_entitlements. Flushed 480 local keys.`
3. Jika Redis tidak dapat diakses, restart pod backend secara bergantian (*rolling restart*) untuk membersihkan RAM cache lokal.
