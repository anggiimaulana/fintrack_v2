# Kontrak API Super Admin (P06) — Frontend ⇄ Backend

Status: **beku** (sumber kebenaran untuk implementasi UI `/super-admin/*`).
Base URL: `/api/v1` (same-origin proxy Next.js; cookie `httpOnly` otomatis, tanpa header `Authorization`).

## 0. Konvensi Umum

Envelope respons (`pkg/response`):

```json
{ "success": true, "message": "...", "data": <payload|omitted>, "error": "<code|omitted>", "meta": { "total": 42, "page": 1, "per_page": 20 } }
```

- `meta` hanya ada pada endpoint daftar (paginated).
- Galat: `success:false`, HTTP 400/401/403/404/409/500, `error` berisi kode mesin
  (`validation`, `invalid_payload`, `invalid_id`, `setting_not_writable`, `INVALID_PAYMENT_STATE`,
  `NO_ACTIVE_PERIOD`, `forbidden`, `unauthorized`, …), `message` sudah berbahasa Indonesia dan boleh ditampilkan langsung.

### Parameter daftar (semua tabel `<DataTable>`)

| Param | Default | Catatan |
|---|---|---|
| `page` | 1 | absolut, 1-based |
| `per_page` | 20 | dipatok 1..200; nilai luar rentang → default 20 |
| `q` | – | pencarian teks bebas (kolom yang dicari lihat per endpoint) |
| `sort` | per endpoint | **whitelist**; nilai tak dikenal diabaikan (pakai default) |
| `order` | `desc` | `asc` \| `desc` |
| `filter_*` (URL frontend) | – | hook `useDataTableUrlState` menyimpan filter dengan prefix `filter_`; **setiap halaman wajib menerjemahkannya** ke nama param backend di bawah |
| `start_date` / `end_date` | – | RFC3339 atau `YYYY-MM-DD`; `end_date` inklusif (backend memakai `< end + 1 hari`) |

RBAC: seluruh `/admin/*` butuh `role = super_admin`.
- Tanpa sesi → **401** `unauthorized`
- Sesi non-super-admin → **403** `forbidden`
- Navigasi UI wajib menyembunyikan menu `super-admin` untuk role lain (guard di layout).

---

## 1. Dashboard metrik

### `GET /admin/metrics/overview`
Query: `refresh=true` memaksa bypass cache (60s). Cache default aktif.

```json
{
  "mrr_idr": 12500000,
  "mrr_growth_percentage": 8.5,
  "active_users_30d": 128,
  "active_users_growth_percentage": -2.0,
  "tokens_consumed_24h": 450000,
  "tokens_growth_percentage": 12.25,
  "payment_success_rate": 94.3,
  "pending_manual_payments": 3,
  "ai_endpoints_healthy": 4,
  "ai_endpoints_total": 5,
  "circuit_breaker_open_count": 1,
  "cached_at": "2026-10-11T09:30:00+07:00"
}
```

Catatan: MRR normalisasi bulanan (`yearly / 12`); `payment_success_rate` = settled / (settled+failed+expired) 30 hari, dalam persen 0..100.

### `GET /admin/metrics/timeseries?period=7d|30d|90d`
`period` tak dikenal → `30d`. `data` = array **flat**:

```json
[{ "date": "2026-10-01", "revenue_idr": 990000, "payments_settled": 4, "mrr_idr": 12500000,
   "tokens_chat": 12000, "tokens_vision": 300, "tokens_voice": 120, "tokens_embedding": 8000 }]
```

`mrr_idr` = nilai MRR saat ini (garis proyeksi datar), `revenue_idr`/`payments_settled` = nilai harian.

### `GET /admin/metrics/payment-methods`
Donut 30 hari terakhir: `[{ "gateway": "manual", "settled_count": 12, "amount_idr": 11880000 }]`.

---

## 2. Paket (`/super-admin/plans`)

### `GET /admin/plans` — tanpa pagination (jumlah paket kecil)
`data`: array `planDTO`:

```json
{ "id": "uuid", "code": "pro", "name": "Pro", "description": "…",
  "price_monthly_idr": 99000, "price_yearly_idr": 990000, "billing_period_days": 30,
  "is_active": true, "is_public": true, "fallback_gateway": "manual", "sort_order": 2,
  "entitlements": { "ai_chat": true }, "limits": { "ai_chat_monthly": 5000 } }
```

Urutan default: `sort_order ASC, price_monthly_idr ASC`. Sorting tabel dilakukan di klien (data kecil).

### `POST /admin/plans` · `PUT /admin/plans/:id` · `DELETE /admin/plans/:id`
Body create/update (semua field opsional saat `PUT`, minimal `code`+`name` saat `POST`):

```json
{ "code": "pro", "name": "Pro", "description": "…", "price_monthly_idr": 99000,
  "price_yearly_idr": 990000, "billing_period_days": 30, "is_active": true, "is_public": true,
  "fallback_gateway": "manual", "sort_order": 2,
  "entitlements": { "ai_chat": true }, "limits": { "ai_chat_monthly": 5000 } }
```

`DELETE` = nonaktifkan (`is_active=false`), tidak menghapus baris. Konflik kode → **409**.
Endpoint `PUT /admin/plans/:id/entitlements` dan `/limits` **tidak ada** — entitlements & limits selalu ikut payload plan di atas.

---

## 3. Langganan (`/super-admin/subscriptions`)

### `GET /admin/subscriptions`
Query: `q` (email/nama user), `user_id`, `plan_code`, `status`, `sort` ∈ {`user`,`plan`,`status`,`created_at`,`ends_at`}, `order`.

`data[]`:

```json
{ "id": "uuid", "user_id": "uuid", "user_email": "a@b.c", "user_full_name": "Nama",
  "plan_id": "uuid", "plan_code": "pro", "plan_name": "Pro", "status": "active",
  "current_period_start": "2026-10-01T00:00:00+07:00", "current_period_end": "2026-10-31T00:00:00+07:00",
  "cancel_at_period_end": false, "gateway": "manual", "created_at": "…" }
```

### Mutasi (wajib `reason`; dicatat ke `subscription_events` + `audit_logs`)
- `POST /admin/subscriptions/:id/extend` body `{ "days": 30, "reason": "kompensasi insiden" }`
- `POST /admin/subscriptions/:id/force-downgrade` body `{ "plan_code": "free", "reason": "…" }` (`plan_code` kosong → paket aktif termurah). Tidak pernah menghapus baris finansial (K-09).
- `POST /admin/subscriptions/:id/cancel` body `{ "reason": "…", "immediate": false }`

---

## 4. Pembayaran (`/super-admin/payments`, `/super-admin/payments/review`)

### `GET /admin/payments`
Query: `status`, `gateway`, `user_id`, `q` (email / nomor invoice), `sort` ∈ {`created_at`,`amount`,`status`,`user`}.

### `GET /admin/payments/review`
Antrean FIFO (`status = pending_review`, urut `created_at ASC`). Query hanya pagination.
Baris identik dengan di atas.

`paymentRow`:

```json
{ "id": "uuid", "invoice_id": "uuid", "invoice_number": "INV-2026-0001",
  "user_id": "uuid", "user_email": "a@b.c", "gateway": "manual", "gateway_ref": null,
  "amount_idr": 99000, "status": "pending_review", "payment_method": "BCA",
  "proof_url": "/uploads/proofs/….jpg", "review_reason": null, "reviewed_by": "uuid",
  "reviewed_at": "…", "refund_amount_idr": 0, "refund_reason": null, "refund_ref": null,
  "created_at": "…" }
```

### Mutasi
- `POST /admin/payments/:id/approve` — tanpa body. Sukses → `{ "id": …, "status": "settled" }`; invoice diterbitkan bernomor, langganan aktif, cache entitlement di-bump (K-08/K-11/K-14).
- `POST /admin/payments/:id/reject` body `{ "reason": "Nominal transfer tidak sesuai" }` → `{ "id": …, "status": "failed" }`.
- `POST /admin/payments/:id/refund` body `{ "amount_idr": 50000, "reason": "…", "reference": "TRF-123" }`.
  `amount_idr <= 0` (atau dihilangkan) = refund sisa penuh. Refund penuh → status `refunded`; parsial tetap `settled`.
- **Race antar-admin** → **409** `INVALID_PAYMENT_STATE`, `message`: `"Transaksi ini sudah diproses oleh admin lain."` Tampilkan sebagai toast dan refresh antrean.

---

## 5. Token (`/super-admin/tokens`)

### `GET /admin/tokens/users`
Query: `q` (email/nama), `sort` ∈ {`user`,`quota`,`used`}.

```json
{ "user_id": "uuid", "user_email": "a@b.c", "user_full_name": "Nama",
  "period_id": "uuid", "period_start": "…", "period_end": "…",
  "quota_tokens": 100000, "used_tokens": 12000, "reserved_tokens": 0, "available_tokens": 88000 }
```

### `GET /admin/tokens/ledger`
Query: `entry_type` (`commit`|`refund`|`adjust`|`grant`|`reset`), `modality` (`chat`|`vision`|`voice`|`embedding`|`admin`), `user_id`, `start_date`, `end_date`, `sort` ∈ {`created_at`,`amount`,`cost`,`user`,`type`}.

```json
{ "id": "uuid", "user_id": "uuid", "user_email": "a@b.c", "entry_type": "commit",
  "amount": -1200, "modality": "chat", "cost_idr": 340, "idempotency_key": "uuid",
  "reference_id": "…", "notes": "…", "created_at": "…" }
```

### `POST /admin/tokens/adjust`

```json
{ "user_id": "uuid", "amount": -5000, "reason": "koreksi audit", "idempotency_key": "uuid-v4" }
```

- `amount` bertanda (+kredit / −debit), tidak boleh 0; `reason` wajib; `idempotency_key` **wajib UUID valid** (tombol submit harus membangkitkan UUID sekali per percobaan dan memakainya ulang saat retry).
- Sukses: `{ "ledger_id": "uuid", "user_id": "uuid", "amount": -5000, "quota_tokens": 95000, "replayed": false }`; replay key sama → `replayed: true` tanpa mutasi kedua.
- **409** `NO_ACTIVE_PERIOD` bila user tidak punya periode token aktif; debit melebihi saldo → **400** `validation`.

---

## 6. AI Gateway (`/super-admin/ai-gateway`)

### `GET /admin/ai-gateway/endpoints`
```json
{ "id": "uuid", "name": "router-1", "base_url": "https://…", "provider": "9router",
  "priority": 1, "status": "active", "health": "healthy",
  "modalities": ["chat","vision","voice","embedding","reasoning"], "weight": 100,
  "last_check_at": "…", "last_error": null, "has_api_key": true, "breaker_state": "closed" }
```
`breaker_state` ∈ `closed` | `open` | `half_open`. `api_key` **tidak pernah** dikirim balik.

### Mutasi endpoint
- `POST /admin/ai-gateway/endpoints` body `{ "name","base_url","provider","priority","weight","status","modalities","api_key" }` → **201** `{ "id": "uuid" }`; nama duplikat → **409**.
- `PUT /admin/ai-gateway/endpoints/:id` — field yang sama, semuanya opsional; `api_key` kosong/absen = pertahankan kunci lama.
- `DELETE /admin/ai-gateway/endpoints/:id`
- `POST /admin/ai-gateway/circuit-breaker/reset` body `{ "endpoint_id": "uuid" }` → `{ "endpoint_id": …, "state": "closed" }`

### `GET /admin/ai-gateway/routes`
```json
[{ "id": "uuid", "use_case": "chat", "endpoint_id": "uuid", "model": "gpt-4o-mini",
   "priority": 1, "timeout_ms": 30000, "max_retries": 2, "is_active": true }]
```

### `PUT /admin/ai-gateway/routes`
Body = **array penuh** `routeDTO` (penggantian atomik seluruh matriks). `use_case` wajib salah satu dari `chat|vision|voice|embedding|reasoning`; `endpoint_id` harus UUID; `timeout_ms`/`max_retries` ≥ 0.

### Kill switch
- `GET /admin/ai-gateway/kill-switch` → `{ "global": false, "chat": false, "vision": false, "voice": false, "embedding": false, "reasoning": false }`
- `POST /admin/ai-gateway/kill-switch` body `{ "target": "global|chat|vision|voice|embedding|reasoning", "enabled": true }` → snapshot baru.

---

## 7. Audit (`/super-admin/audit`)

### `GET /admin/audit-logs`
Query: `start_date`, `end_date`, `user_id` (alias `actor_id`), `email`, `entity`
(`plan|subscription|payment|token|ai_route|settings`), `action`, `entity_id`, `sort` ∈ {`created_at`,`action`,`entity`,`actor`}.
Semua filter teks case-insensitive. `user_id` non-UUID → **400** `validation`.

```json
{ "id": "uuid", "actor_id": "uuid", "actor_name": "Nama", "actor_email": "a@b.c",
  "action": "update", "entity": "settings", "entity_id": "bulk",
  "before": { … }, "after": { … }, "ip": "1.2.3.4", "user_agent": "…",
  "request_id": "…", "created_at": "…" }
```

`before`/`after` sudah dimasking (PDP/UU 27/2022) — render apa adanya di diff viewer.
Read-only: tidak ada endpoint export/CSV.

---

## 8. Pengaturan (`/super-admin/settings`)

### `GET /admin/settings`
```json
[{ "key": "tax_rate_percent", "value": 11, "description": "…", "updated_by": "uuid", "updated_at": "…" }]
```

### `PUT /admin/settings`
Body `{ "settings": { "<key>": <value> } }` — bulk upsert, minimal satu key.
Key yang boleh ditulis **hanya**: `payment_gateway_default` (`manual|midtrans`), `grace_period_hours` (0..720), `tax_rate_percent` (0..100), `invoice_due_hours` (1..720).
Key lain (mis. `ai_kill_switch`) → **400** `setting_not_writable`. Nilai di luar rentang → **400** `validation`.

### Feature flags
- `GET /admin/feature-flags` → array flag `{ "key","name","description","enabled","rollout","rules" }` (absen = `[]`).
- `PUT /admin/feature-flags` body `flagRequest` (field di atas) — toggle + rollout persen + rules.

---

## 9. Catatan implementasi UI

- `lib/api.ts` (axios, `withCredentials`) sudah menangani refresh token single-flight dan redirect ke `/login` saat 401.
- Hook `hooks/use-data-table-url-state.ts` + `lib/table-params.ts` = satu-satunya sumber state tabel (URL). Filter dikirim ke API sebagai `filter_<name>` di URL → diterjemahkan per halaman.
- Komponen `components/data-table/*` bersifat server-driven: kirim `sort`/`order` ke API dan render `meta.total`.
- Warna/format uang: `components/shared/Money.tsx`; format angka/IDR: `lib/format.ts`.
