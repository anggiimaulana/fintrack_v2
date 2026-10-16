# Spesifikasi Metrik & Agregasi Dashboard Super Admin

Dokumen ini mendefinisikan seluruh formula matematika, query agregasi database PostgreSQL, arsitektur caching Redis, dan spesifikasi API untuk dashboard metrik eksekutif FinTrack v2.

---

## 1. Definisi Formula KPI Utama

### 1.1 Monthly Recurring Revenue (MRR)
MRR mengukur pendapatan berulang bulanan yang dinormalisasi dari seluruh langganan aktif pengguna:
$$\text{MRR} = \sum_{s \in S_{\text{active}}} \left( \frac{\text{price\_idr}(s.\text{plan\_id}) \times 30}{\text{billing\_period\_days}(s.\text{plan\_id})} \right)$$
*Catatan*: Menggunakan pembagian integer rupiah sesuai K-27 (hasil dibulatkan ke bawah / floor).

### 1.2 Pengguna Aktif Bulanan (MAU)
Jumlah pengguna unik yang melakukan login atau mencatat transaksi dalam 30 hari terakhir:
$$\text{MAU} = \text{COUNT(DISTINCT user\_id)} \quad \text{dimana} \quad \text{last\_activity\_at} \ge \text{NOW}() - \text{INTERVAL '30 days'}$$

### 1.3 Pemakaian Token AI 24 Jam
Total token AI yang dikonsumsi (committed) oleh seluruh pengguna dalam 24 jam terakhir:
$$\text{Total Token}_{24\text{h}} = \sum_{l \in L_{24\text{h}}} l.\text{tokens\_used}$$

### 1.4 Payment Success Rate
Persentase transaksi pembayaran yang berhasil diselesaikan dibandingkan total upaya transaksi checkout (tidak termasuk checkout yang sengaja dibatalkan):
$$\text{Success Rate} = \left( \frac{N_{\text{settled}}}{N_{\text{settled}} + N_{\text{failed}} + N_{\text{expired}}} \right) \times 100\%$$

---

## 2. Spesifikasi Endpoint API & Caching Redis

### 2.1 Endpoint Ringkasan Eksekutif
- **Rute**: `GET /api/v1/admin/metrics/overview`
- **Otorisasi**: `Bearer Token` dengan role `super_admin`
- **Strategi Caching**: Redis Key `admin:metrics:overview`, TTL **60 detik**. Mendukung query param `?refresh=true` untuk bypass cache.
- **Contoh Respon JSON**:
```json
{
  "status": "success",
  "data": {
    "mrr_idr": 48500000,
    "mrr_growth_percentage": 12.4,
    "active_users_30d": 1845,
    "active_users_growth_percentage": 8.1,
    "tokens_consumed_24h": 4235800,
    "tokens_growth_percentage": -3.2,
    "payment_success_rate": 96.8,
    "pending_manual_payments": 7,
    "ai_endpoints_healthy": 3,
    "ai_endpoints_total": 3,
    "circuit_breaker_open_count": 0,
    "cached_at": "2026-10-11T14:30:00+07:00"
  }
}
```

### 2.2 Endpoint Data Deret Waktu (Time-Series)
- **Rute**: `GET /api/v1/admin/metrics/timeseries?period=30d`
- **Output**: Array data harian untuk plotting grafik Recharts:
```json
{
  "status": "success",
  "data": [
    {
      "date": "2026-10-01",
      "revenue_idr": 1650000,
      "tokens_chat": 85000,
      "tokens_vision": 42000,
      "tokens_voice": 18000,
      "tokens_embedding": 12000
    }
  ]
}
```

---

## 3. Query PostgreSQL Teroptimasi

Query backend memanfaatkan indeks terkelola untuk mencegah *full table scan*:

```sql
-- Query Perhitungan MRR
SELECT COALESCE(SUM(FLOOR((p.price_idr * 30.0) / p.billing_period_days)), 0)::BIGINT AS mrr_idr
FROM subscriptions s
JOIN plans p ON s.plan_id = p.id
WHERE s.status = 'active';

-- Query Konsumsi Token 24 Jam Terakhir
SELECT COALESCE(SUM(amount), 0)::BIGINT AS tokens_used
FROM token_ledger
WHERE event_type IN ('ai_chat', 'ai_vision', 'ai_voice', 'ai_embedding')
  AND created_at >= NOW() - INTERVAL '24 hours';
```
Semua query dijalankan dengan timeout maksimal 3 detik untuk menjaga keandalan database produksi.
