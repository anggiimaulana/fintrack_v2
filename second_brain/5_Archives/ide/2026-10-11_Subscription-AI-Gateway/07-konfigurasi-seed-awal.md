# 07 — Konfigurasi Seed Awal (Siap Pakai)

Nilai konkret untuk **seed** awal. Dapat langsung dipakai saat implementasi P04/P05/P06, dan
sepenuhnya **dapat diubah super admin** dari dashboard (tanpa deploy). Angka bisnis = **[USULAN]**.

> Konvensi: uang = **integer rupiah**; token = integer; `-1` = unlimited.

---

## 1. `plans`

| code | name | price_idr | billing_period_days | is_active | sort | gateway_override |
|---|---|---:|---:|:--:|---:|---|
| `free` | Free | 0 | 30 | ✅ | 1 | NULL |
| `pro` | Pro | 39000 | 30 | ✅ | 2 | NULL |
| `business` | Business | 129000 | 30 | ✅ | 3 | NULL |

> `billing_period_days` = 30 hari **per-hari kalender** sejak aktivasi (K-05). Dapat diubah (mis. 365).

---

## 2. `plan_entitlements` (fitur bool)

| feature_key | free | pro | business |
|---|:--:|:--:|:--:|
| `ai_chat` | ✅ | ✅ | ✅ |
| `ocr_receipt` | ❌ | ✅ | ✅ |
| `voice_input` | ❌ | ✅ | ✅ |
| `rag_search` | ❌ | ✅ | ✅ |
| `reports_export` | ❌ | ✅ | ✅ |
| `advanced_budgeting` | ❌ | ✅ | ✅ |
| `api_access` | ❌ | ❌ | ✅ |
| `bank_sync` | ❌ | ❌ | ❌ (masa depan) |

---

## 3. `plan_limits` (numerik; `-1` = unlimited)

| limit_key | free | pro | business |
|---|---:|---:|---:|
| `ai_tokens_per_cycle` | 5000 | 150000 | 1000000 |
| `ai_requests_per_cycle` | 50 | 2000 | -1 |
| `max_transactions` | 100 | 5000 | -1 |
| `max_categories` | 5 | 30 | -1 |
| `max_wallets` | 1 | 5 | -1 |
| `ocr_scans_per_cycle` | 0 | 100 | -1 |

---

## 4. `pricing_rules`

| field | nilai |
|---|---|
| `rule_name` | `default` |
| `base_per_1k_idr` | 300 |
| `markup_multiplier` | 1.2500 |
| `markup_percent` | 0 |
| `modality_multipliers` | `{"chat":1.0,"vision":2.5,"voice":3.0,"embedding":0.5,"reasoning":2.0}` |
| `is_active` | ✅ |

**Contoh perhitungan:**
- 1.000 token chat: `300 × (1000/1000) × 1.25 × 1.0 = Rp375`
- 1.000 token vision: `300 × 1 × 1.25 × 2.5 = Rp938` (dibulatkan ke bawah dari 937,5)
- 10.000 token chat: `300 × 10 × 1.25 × 1.0 = Rp3.750`

---

## 5. `settings`

| key | value | Arti |
|---|---|---|
| `payment_gateway` | `"manual"` | Switch global (`midtrans`/`manual`) |
| `grace_period_days` | `3` | Masa tenggang |
| `reminder_days_before` | `[3,1,0]` | Reminder H-3/H-1/H-0 |
| `default_plan_code` | `"free"` | Plan saat registrasi |
| `pricing_rule_default` | `"default"` | Rule harga aktif |
| `timezone` | `"Asia/Jakarta"` | Zona waktu siklus |
| `manual_payment_review_sla_hours` | `24` | SLA review bukti transfer |
| `currency` | `"IDR"` | Mata uang |
| `invoice_tax_percent` | `0` | PPN (0 = nonaktif, siap diaktifkan) |

---

## 6. `feature_flags` (default OFF saat rilis bertahap)

| key | enabled | rollout | Arti |
|---|:--:|---:|---|
| `ff_billing` | ❌ | 0 | Modul billing/subscription |
| `ff_aigateway` | ❌ | 0 | AI gateway terpusat |
| `ff_sse` | ❌ | 0 | Event realtime SSE |
| `ff_datatable` | ❌ | 0 | DataTable standar |
| `ff_payment_midtrans` | ❌ | 0 | Gateway Midtrans |

> Dinyalakan bertahap saat rollout (P07); `rollout` = persentase user (0–100).

---

## 7. `ai_endpoints` (contoh awal)

| alias | base_url | provider | priority | status | modalities |
|---|---|---:|---|---|---|
| `9router-primary` | `http://localhost:20128` | `openai-compatible` | 1 | active | text,vision,audio |
| `9router-secondary` | `http://localhost:20129` | `openai-compatible` | 2 | active | text,vision,audio |
| `gemini-direct` | `https://generativelanguage.googleapis.com/v1beta` | `gemini` | 99 | inactive | text,vision,embedding |

> `gemini-direct` = **fallback darurat** (kunci terpisah). Kunci disimpan terenkripsi; hanya preview di API.

---

## 8. `ai_routes` (contoh awal)

| use_case | endpoint | model | combo | priority | is_active |
|---|---|---|---|---:|---|
| `chat` | `9router-primary` | (default 9router) | `fast` | 1 | ✅ |
| `reasoning` | `9router-primary` | (model reasoning) | `reasoning` | 1 | ✅ |
| `vision` | `9router-primary` | (model multimodal) | `vision` | 1 | ✅ |
| `voice` | `9router-primary` | (model audio) | `audio` | 1 | ✅ |
| `embedding` | `gemini-direct` | `text-embedding-004` | — | 1 | ✅ |

> Embedding diarahkan ke model **768d** (K-18). Sesuaikan `model`/`combo` setelah verifikasi 9router.

---

## 9. Urutan Seed

```
1. plans
2. plan_entitlements
3. plan_limits
4. pricing_rules
5. settings
6. feature_flags (semua OFF)
7. ai_endpoints (kunci dienkripsi ENCRYPTION_KEY)
8. ai_routes
9. (opsional) user demo → subscription free
```

Semua seed **idempoten** (`INSERT ... ON CONFLICT DO NOTHING/UPDATE`) agar aman diulang.

## Referensi

- `06-jawaban-rekomendasi-pertanyaan-terbuka.md`
- `../../2_Projects/P04_Subscription-Token-Core/teknis/entitlement-matrix.md`
- `../../2_Projects/P03_AI-Gateway-Terpusat/teknis/routing-table.md`
