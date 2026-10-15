# 03 — Spesifikasi API

[← Index](./README.md) · [← Skema & Migrasi](./02-skema-database-dan-migrasi.md)

---

## 1. Ringkasan perubahan API

| Method | Path | Status | Keterangan |
|---|---|---|---|
| `GET` | `/admin/plans` | **Berubah** | +`token_quota_ratio` per plan |
| `POST` | `/admin/plans` | **Berubah** | +`token_quota_ratio` opsional |
| `PUT` | `/admin/plans/:id` | **Berubah** | +`token_quota_ratio` opsional |
| `GET` | `/admin/entitlement-catalog` | **Berubah** | Kini dari **DB** (bukan kode) |
| `POST` | `/admin/plans/estimate` | **BARU** | Kalkulator kuota/biaya/margin |
| `GET` | `/admin/tokens/ledger` | **Berubah** | +`raw_tokens` per baris |
| `GET` | `/me/entitlements` | **Berubah (label)** | +`unit` pada usage; nilai = unit |
| `GET` | `/admin/setting-definitions` | **BARU** | Katalog setting (dari DB) |
| `PUT` | `/admin/setting-definitions/:key` | **BARU** | Ubah definisi setting |
| `GET` | `/admin/metric-definitions` | **BARU** | Katalog metrik (dari DB) |
| `POST` | `/admin/metric-definitions` | **BARU** | Tambah metrik baru |
| `PUT` | `/admin/metric-definitions/:key` | **BARU** | Ubah metrik |
| `DELETE` | `/admin/metric-definitions/:key` | **BARU** | Nonaktifkan metrik (soft) |
| `GET` | `/admin/feature-definitions` | **BARU** | Katalog fitur (dari DB) |
| `POST` | `/admin/feature-definitions` | **BARU** | Tambah fitur baru |
| `GET` | `/admin/resolvers` | **BARU** | Daftar resolver yang tersedia di kode |
| `POST` | `/admin/config/reload` | **BARU** | Paksa invalidasi katalog (untuk perubahan di luar API) |
| — | event SSE `config_changed` | **BARU** | Diberitakan ke klien saat katalog/plan berubah |

> **Prinsip:** seluruh katalog kini **data**, bukan kode. Detail lengkap di
> [09-lapisan-konfigurasi-dinamis.md](./09-lapisan-konfigurasi-dinamis.md).

Semua endpoint lama **tetap kompatibel**: field baru bersifat *additive*; klien lama
yang tidak membaca `token_quota_ratio` akan memakai default `1`.

---

## 2. `GET /admin/plans` — daftar plan

### 2.1 Response (contoh)

```json
{
  "success": true,
  "message": "Daftar paket berhasil dimuat",
  "data": [
    {
      "id": "a0000000-0000-0000-0000-000000000001",
      "code": "free",
      "name": "Free",
      "description": "Paket gratis untuk mencoba",
      "price_monthly_idr": 0,
      "price_yearly_idr": 0,
      "billing_period_days": 30,
      "is_active": true,
      "is_public": true,
      "fallback_gateway": "midtrans",
      "sort_order": 0,
      "token_quota_ratio": 5,
      "entitlements": { "ai_chat": true, "ocr_receipt": false },
      "limits": { "ai_tokens_per_cycle": 5000, "ai_requests_per_cycle": 50 }
    }
  ]
}
```

### 2.2 Perubahan

| Field | Tipe | Keterangan |
|---|---|---|
| `token_quota_ratio` | `integer` | **BARU.** Pembagi token→unit. `1` = 1:1. Selalu ≥ 1 |

---

## 3. `POST /admin/plans` — buat plan

### 3.1 Request

```json
{
  "code": "starter",
  "name": "Starter",
  "description": "Paket pemula",
  "price_monthly_idr": 49000,
  "price_yearly_idr": 490000,
  "billing_period_days": 30,
  "is_active": true,
  "is_public": true,
  "fallback_gateway": "midtrans",
  "sort_order": 10,
  "token_quota_ratio": 3,
  "entitlements": { "ai_chat": true, "ocr_receipt": true },
  "limits": { "ai_tokens_per_cycle": 50000, "ai_requests_per_cycle": 500 }
}
```

### 3.2 Validasi

| Field | Aturan | Pesan error |
|---|---|---|
| `code` | wajib, unik, lowercase, `^[a-z0-9_-]{2,32}$` | `"kode dan nama paket wajib diisi"` / `"Kode paket sudah dipakai"` |
| `name` | wajib | `"kode dan nama paket wajib diisi"` |
| `price_*` | ≥ 0 | `"harga tidak boleh negatif"` |
| `token_quota_ratio` | opsional, default `1`, **≥ 1**, ≤ 1000 | `"rasio kuota harus antara 1 dan 1000"` |
| `limits.*` | integer; `-1` = unlimited | — |
| `entitlements.*` | key harus ada di katalog | **peringatan** (bukan error) |

### 3.3 Validasi kunci limit terhadap katalog

Karena editor kini berbasis dropdown, backend **tetap** memvalidasi:

```go
// admin/plans.go — validasi (konseptual)
for key := range derefInt64Map(req.Limits) {
    if !entitlement.IsKnownLimit(key) {
        // Bukan error fatal: plan boleh punya limit yang belum di-enforce.
        // Namun dicatat agar operator tahu.
        logger.Warn("plan uses unknown limit key", zap.String("limit_key", key))
    }
}
for key := range derefBoolMap(req.Entitlements) {
    if !entitlement.IsKnownFeature(key) {
        logger.Warn("plan uses unknown feature key", zap.String("feature_key", key))
    }
}
```

> **Keputusan:** unknown key **tidak** ditolak (agar operator bisa menyiapkan limit
> untuk fitur mendatang), tetapi **dicatat** dan **diperingatkan di UI** (sudah ada
> pola `enforcedByKey` di `PlanFormDialog`).

---

## 4. `POST /admin/plans/estimate` — kalkulator (BARU)

Endpoint ini yang menjawab kebutuhan user: *"di super admin ada perhitungannya"*.

### 4.1 Request

```json
{
  "token_quota_ratio": 5,
  "ai_tokens_per_cycle": 5000,
  "avg_tokens_per_chat": 9945,
  "price_rule_id": "b0000000-0000-0000-0000-000000000001"
}
```

| Field | Tipe | Wajib | Default | Keterangan |
|---|---|---|---|---|
| `token_quota_ratio` | integer | ✅ | — | Rasio yang disimulasikan |
| `ai_tokens_per_cycle` | integer | ✅ | — | Kuota plan (unit) |
| `avg_tokens_per_chat` | integer | ❌ | `8000` | Rata-rata token nyata per chat |
| `price_rule_id` | string | ❌ | rule aktif | Untuk menghitung biaya/margin |

### 4.2 Response

```json
{
  "success": true,
  "message": "Estimasi berhasil dihitung",
  "data": {
    "input": {
      "token_quota_ratio": 5,
      "ai_tokens_per_cycle": 5000,
      "avg_tokens_per_chat": 9945
    },
    "quota": {
      "unit_per_chat": 1989,
      "chats_per_cycle": 2,
      "chats_per_cycle_exact": 2.51,
      "leftover_units": 3022
    },
    "cost": {
      "price_unit": "per_10k",
      "sell_per_1k_idr": 300,
      "provider_cost_per_1k_idr": 150,
      "sell_per_chat_idr": 2984,
      "provider_cost_per_chat_idr": 1492,
      "margin_per_chat_idr": 1492,
      "margin_percent": 50.0,
      "cost_per_full_cycle_idr": 7480,
      "revenue_per_full_cycle_idr": 14960,
      "margin_per_full_cycle_idr": 7480
    },
    "comparison": {
      "without_ratio": { "chats_per_cycle": 0.50 },
      "with_ratio": { "chats_per_cycle": 2.51 }
    },
    "warnings": [
      "Rasio 5 memberi ~2,5 chat per siklus. Pertimbangkan menaikkan kuota bila ingin ≥ 5 chat."
    ]
  }
}
```

### 4.3 Aturan perhitungan

```
unit_per_chat          = ceil(avg_tokens_per_chat / token_quota_ratio)
chats_per_cycle_exact  = ai_tokens_per_cycle / unit_per_chat
chats_per_cycle        = floor(chats_per_cycle_exact)
leftover_units         = ai_tokens_per_cycle - (chats_per_cycle × unit_per_chat)

sell_per_chat_idr      = PricingEngine.CostIDR(avg_tokens_per_chat, "chat")
provider_cost_per_chat = PricingEngine.ProviderCostIDR(avg_tokens_per_chat, "chat")
margin_per_chat        = sell − provider_cost
margin_percent         = margin / sell × 100  (0 bila sell = 0)
```

**Penting:** biaya dihitung dari `avg_tokens_per_chat` (RAW), **bukan** unit — konsisten
dengan K-1.

### 4.4 `warnings` — heuristik

| Kondisi | Pesan |
|---|---|
| `chats_per_cycle_exact < 1` | "Kuota tidak cukup untuk satu chat. Naikkan kuota atau rasio." |
| `1 ≤ chats_per_cycle_exact < 3` | "Rasio {r} memberi ~{n} chat per siklus. Pertimbangkan menaikkan kuota." |
| `margin_per_chat < 0` | "Harga jual di bawah biaya provider. Periksa aturan harga." |
| `margin_percent < 20` | "Margin di bawah 20%. Pertimbangkan menyesuaikan markup." |
| `token_quota_ratio > 20` | "Rasio sangat tinggi; user mungkin tidak menyadari biaya nyata." |

### 4.5 Error

| Status | Code | Kondisi |
|---|---|---|
| 400 | `invalid_payload` | Body bukan JSON valid |
| 400 | `validation` | `token_quota_ratio < 1` atau `ai_tokens_per_cycle < 1` |
| 404 | `rule_not_found` | `price_rule_id` tidak ditemukan |
| 500 | `internal` | Gagal membaca rule |

> Endpoint ini **read-only** (tidak mengubah apa pun) dan **idempoten** — aman
> dipanggil sesering apa pun saat operator menggeser slider.

---

## 5. Katalog konfigurasi (dari DB, bukan kode)

> **Revisi:** katalog kini **data**, bukan konstanta kode. Menambah setting/metrik/fitur
> baru tidak perlu deploy. Detail: [09](./09-lapisan-konfigurasi-dinamis.md).

### 5.0 `GET /admin/setting-definitions`

```json
{
  "success": true,
  "data": [
    {
      "key": "quota_rounding_mode",
      "kind": "enum",
      "enum_values": ["ceil", "floor", "round"],
      "default_value": "ceil",
      "group_key": "kuota",
      "label": "Mode pembulatan kuota",
      "description": "Cara membulatkan unit kuota.",
      "is_writable": true,
      "is_secret": false,
      "sort_order": 20
    },
    {
      "key": "quota_default_tokens_per_cycle",
      "kind": "int", "min_value": 1, "max_value": 1000000000,
      "default_value": 10000, "group_key": "kuota",
      "label": "Kuota default / siklus", "is_writable": true
    }
  ]
}
```

### 5.0.1 `PUT /admin/setting-definitions/:key`

Mengubah **definisi** (bukan nilai). Berguna untuk mengubah batas validasi, default,
atau label tanpa deploy.

```json
{ "min_value": 1, "max_value": 10000, "default_value": 15000, "label": "Kuota default" }
```

### 5.0.2 `GET /admin/metric-definitions`

```json
{
  "success": true,
  "data": [
    {
      "key": "ai_tokens_per_cycle",
      "label": "Kuota AI / siklus",
      "unit": "kuota",
      "description": "Kuota AI per siklus, diukur dalam unit kuota.",
      "is_enforced": true,
      "resolver_key": "token_period_usage",
      "ratio_adjusted": true,
      "per_cycle": true,
      "group_key": "kuota",
      "sort_order": 10
    }
  ]
}
```

### 5.0.3 `POST /admin/metric-definitions` (metrik baru tanpa deploy)

```json
{
  "key": "budgets_created",
  "label": "Anggaran / siklus",
  "unit": "anggaran",
  "description": "Jumlah anggaran yang dapat dibuat.",
  "is_enforced": true,
  "resolver_key": "budgets_created",
  "per_cycle": true,
  "sort_order": 70
}
```

Validasi:

| Aturan | Error |
|---|---|
| `key` unik, `^[a-z0-9_]{2,64}$` | 409 `conflict` |
| `is_enforced = true` wajib punya `resolver_key` | 400 `validation` |
| `resolver_key` harus ada di `GET /admin/resolvers` | 400 `validation` |

### 5.0.3b Invalidasi katalog (P-13)

Setiap endpoint tulis di atas (`PUT /admin/settings`, `POST/PUT/DELETE /admin/*-definitions`,
`PUT /admin/plans/:id`, `PUT /admin/pricing-rules/:id`) melakukan **dua hal** dalam satu
transaksi:

1. `CacheEngine.BumpVersion(ctx, "cfg")` — invalidasi cache **lintas-instance** (O(1))
2. Publish event SSE `config_changed` — klien memuat ulang segera

```json
// Payload event SSE
{
  "event": "config_changed",
  "timestamp": 1760000000,
  "id": "evt_...",
  "payload": { "scope": "settings", "keys": ["quota_rounding_mode"] }
}
```

`scope` ∈ `settings | metrics | features | plans | pricing`. Bila `keys` kosong =
seluruh scope berubah.

### 5.0.3c `POST /admin/config/reload`

Memaksa invalidasi untuk perubahan yang dilakukan **di luar API** (mis. operator
mengubah DB langsung via psql). Menjalankan `BumpVersion` + publish `config_changed`.

```json
{ "scope": "settings" }
```

Response: `200 { "success": true, "message": "Katalog dimuat ulang" }`

> **Kenapa perlu?** Perubahan langsung di DB tidak memicu event apa pun. Endpoint ini
> adalah jalan keluar eksplisit, di samping TTL 5 menit sebagai jaring terakhir.

### 5.0.4 `GET /admin/resolvers`

```json
{
  "success": true,
  "data": {
    "resolvers": [
      { "key": "token_period_usage", "label": "Pemakaian kuota per siklus",
        "unit": "kuota", "requires_period": true },
      { "key": "ai_requests_in_period", "label": "Permintaan AI per siklus",
        "unit": "permintaan", "requires_period": true },
      { "key": "active_transactions", "label": "Transaksi aktif",
        "unit": "transaksi", "requires_period": false },
      { "key": "vision_scans_in_period", "label": "Pemindaian struk per siklus",
        "unit": "scan", "requires_period": true }
    ]
  }
}
```

> Resolver adalah **kemampuan** yang disediakan kode. Operator memilih salah satu via
> dropdown saat membuat metrik. Menambah **kemampuan** baru butuh deploy; memilih
> yang sudah ada **tidak**.

### 5.1 `GET /admin/entitlement-catalog` — katalog (kompatibilitas)

Endpoint lama tetap ada, tetapi datanya kini **berasal dari DB** (`feature_definitions`,
`metric_definitions`), bukan konstanta kode.

#### Response (contoh)

```json
{
  "success": true,
  "message": "Katalog entitlement",
  "data": {
    "features": [
      { "key": "ai_chat", "label": "Asisten AI (chat)", "description": "Chat keuangan dengan AI." }
    ],
    "limits": [
      {
        "key": "ai_tokens_per_cycle",
        "label": "Token AI / siklus",
        "unit": "kuota",
        "enforced": true,
        "description": "Kuota AI per siklus, diukur dalam unit kuota (bukan token mentah).",
        "ratioAdjusted": true
      },
      {
        "key": "ai_requests_per_cycle",
        "label": "Permintaan AI / siklus",
        "unit": "permintaan",
        "enforced": true,
        "ratioAdjusted": false
      }
    ]
  }
}
```

### 5.2 Perubahan pada `entitlement.LimitKey` (kini dari DB)

> **Revisi:** `LimitCatalog`/`FeatureCatalog` **tidak lagi konstanta kode**. Struct
> tetap ada sebagai bentuk DTO, tetapi isinya dibaca dari `metric_definitions` /
> `feature_definitions`.

```go
// entitlement/catalog.go — SESUDAH
type LimitKey struct {
    Key         string `json:"key"`
    Label       string `json:"label"`
    Unit        string `json:"unit"`
    Enforced    bool   `json:"enforced"`
    // Description menjelaskan makna limit untuk tooltip di UI.
    Description string `json:"description,omitempty"`
    // RatioAdjusted menandai limit yang nilainya dipengaruhi rasio token,
    // sehingga UI dapat menampilkan catatan khusus.
    RatioAdjusted bool `json:"ratioAdjusted,omitempty"`
}
```

Dan katalog diperbarui:

```go
var LimitCatalog = []LimitKey{
    {
        Key: "ai_tokens_per_cycle", Label: "Kuota AI / siklus", Unit: "kuota",
        Enforced: true, RatioAdjusted: true,
        Description: "Kuota AI per siklus, diukur dalam unit kuota. " +
            "Token provider dibagi rasio plan sebelum ditagih.",
    },
    {Key: "ai_requests_per_cycle", Label: "Permintaan AI / siklus", Unit: "permintaan", Enforced: true,
        Description: "Jumlah panggilan AI (chat, OCR, suara) per siklus."},
    {Key: "max_transactions", Label: "Transaksi", Unit: "transaksi", Enforced: true,
        Description: "Transaksi aktif yang dapat disimpan."},
    {Key: "ocr_scans_per_cycle", Label: "Pindai struk / siklus", Unit: "scan", Enforced: true,
        Description: "Pemindaian struk (OCR) per siklus."},
    {Key: "max_categories", Label: "Kategori", Unit: "kategori", Enforced: false},
    {Key: "max_wallets", Label: "Dompet", Unit: "dompet", Enforced: false},
}
```

> **Catatan label:** `ai_tokens_per_cycle` tetap memakai key yang sama (agar tidak
> memutus data), tetapi **label** berubah dari "Token AI / siklus" → "Kuota AI /
> siklus" dan **unit** dari "token" → "kuota" (keputusan P-3).

---

## 6. `GET /me/entitlements` — kuota user

### 6.1 Response (contoh)

```json
{
  "success": true,
  "data": {
    "plan_code": "free",
    "plan_name": "Free",
    "features": { "ai_chat": true, "ocr_receipt": false },
    "limits": { "ai_tokens_per_cycle": 5000, "ai_requests_per_cycle": 50 },
    "usage": {
      "ai_tokens_per_cycle": {
        "used": 1989,
        "limit": 5000,
        "remaining": 3011,
        "unlimited": false,
        "unit": "kuota"
      },
      "ai_requests_per_cycle": {
        "used": 1, "limit": 50, "remaining": 49, "unlimited": false, "unit": "permintaan"
      }
    },
    "period": { "start": "2026-10-15T00:00:00Z", "end": "2026-11-14T00:00:00Z" },
    "version": 7
  }
}
```

### 6.2 Perubahan

| Field | Perubahan |
|---|---|
| `usage.*.unit` | **BARU** — satuan untuk label di UI |
| `usage.*.used/limit/remaining` | Nilai kini **unit** untuk `ai_tokens_per_cycle` |

**Backward compatibility:** klien lama yang hanya menampilkan angka tetap bekerja
(angkanya saja yang berubah makna). Tidak ada field yang dihapus.

---

## 7. `GET /admin/tokens/ledger` — audit ledger

### 7.1 Response (contoh)

```json
{
  "data": [
    {
      "id": "…",
      "entry_type": "commit",
      "amount": 1989,
      "raw_tokens": 9945,
      "modality": "chat",
      "cost_idr": 2984,
      "created_at": "2026-10-15T10:20:00Z"
    },
    {
      "id": "…",
      "entry_type": "reserve",
      "amount": 762,
      "raw_tokens": 0,
      "modality": "chat",
      "cost_idr": 0,
      "created_at": "2026-10-15T10:19:58Z"
    }
  ]
}
```

### 7.2 Kolom baru

| Field | Keterangan |
|---|---|
| `raw_tokens` | Token provider nyata. `0` untuk reserve/release. Untuk baris lama (sebelum migrasi), `0` berarti "sama dengan `amount`" |

> UI admin **wajib** memakai `GREATEST(raw_tokens, ABS(amount))` saat menjumlahkan
> token nyata (lihat [02 §6.5](./02-skema-database-dan-migrasi.md)).

---

## 8. Kontrak error (tidak berubah)

| Status | Code | Dipakai untuk |
|---|---|---|
| 400 | `invalid_payload` | JSON tidak valid |
| 400 | `validation` | Aturan bisnis dilanggar |
| 402 | `limit_exceeded` | Kuota plan tercapai (middleware) |
| 402 | `limit_not_in_plan` | Limit tidak ada di plan (fail-closed) |
| 403 | `feature_not_in_plan` | Fitur tidak tersedia di plan |
| 404 | `not_found` | Resource tidak ada |
| 409 | `conflict` | Duplikat (mis. kode plan) |
| 429 | `insufficient_quota` | Sisa kuota < reserve (ledger) |
| 500 | `internal` | Kesalahan server |

Tidak ada perubahan pada kontrak ini.

---

## 9. Contoh alur lengkap (curl)

```bash
# 1. Operator membuka editor plan → ambil katalog (dari DB)
curl -s "$API/admin/entitlement-catalog" -H "Cookie: $COOKIE" | jq '.data.limits'
curl -s "$API/admin/metric-definitions" -H "Cookie: $COOKIE" | jq '.data'
curl -s "$API/admin/setting-definitions" -H "Cookie: $COOKIE" | jq '.data[] | select(.group_key=="kuota")'
curl -s "$API/admin/resolvers" -H "Cookie: $COOKIE" | jq '.data.resolvers'

# 2. Operator menggeser slider rasio → kalkulator
curl -s -X POST "$API/admin/plans/estimate" \
  -H "Cookie: $COOKIE" -H "Content-Type: application/json" \
  -d '{"token_quota_ratio":5,"ai_tokens_per_cycle":5000,"avg_tokens_per_chat":9945}' | jq '.data'

# 3. Operator menyimpan plan
curl -s -X PUT "$API/admin/plans/a0000000-0000-0000-0000-000000000001" \
  -H "Cookie: $COOKIE" -H "Content-Type: application/json" \
  -d '{"token_quota_ratio":5,"limits":{"ai_tokens_per_cycle":5000}}' | jq

# 4. User melihat kuotanya (siklus berikutnya memakai rasio baru)
curl -s "$API/me/entitlements" -H "Cookie: $USER_COOKIE" | jq '.data.usage'

# 5. Operator mengubah kebijakan dinamis (tanpa deploy)
curl -s -X PUT "$API/admin/settings" \
  -H "Cookie: $COOKIE" -H "Content-Type: application/json" \
  -d '{"settings":{"quota_rounding_mode":"ceil","quota_default_tokens_per_cycle":10000}}' | jq

# 6. Operator menambah metrik baru (tanpa deploy, resolver sudah ada)
curl -s -X POST "$API/admin/metric-definitions" \
  -H "Cookie: $COOKIE" -H "Content-Type: application/json" \
  -d '{"key":"budgets_created","label":"Anggaran / siklus","unit":"anggaran","is_enforced":true,"resolver_key":"budgets_created"}' | jq
```

---

**Navigasi:** [← Skema & Migrasi](./02-skema-database-dan-migrasi.md) · [Desain UI →](./04-desain-ui.md)
