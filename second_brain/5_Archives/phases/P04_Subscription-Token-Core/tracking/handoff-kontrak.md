# Handoff Kontrak — P04 → P05 / P06

Kontrak antarmuka yang disediakan P04 (Subscription & Token Core) untuk dipakai
fase berikutnya. Perubahan pada kontrak ini bersifat **breaking**.

## 1. Untuk P05 (Payments & Billing)

### 1.1 Service yang tersedia

| Service | Lokasi | Fungsi |
|---|---|---|
| `billing.SubscriptionManager` | `internal/billing/subscription_management.go` | `Upgrade(ctx, userID, planCode)` (prorata), `ScheduleDowngrade`, `ApplyScheduledDowngrade` |
| `billing.LifecycleService` | `internal/billing/lifecycle.go` | `Transition`, `Renew` (aktivasi/perpanjangan setelah pembayaran) |
| `billing.PlanRepository` | `internal/billing/repository.go` | `GetPlanByCode`, `GetPlanByID`, `ListActivePlans` |
| `billing.PricingEngine` | `internal/billing/pricing.go` | `CostIDR(tokens, modality)` |

### 1.2 Alur pembayaran yang diharapkan

```
1. User pilih plan -> P05 buat invoice (integer IDR dari plans.price_monthly_idr)
2. Pembayaran sukses (webhook/approve manual)
3. P05 memanggil:
     - New user  -> LifecycleService: create subscription (status active) + token_period awal
     - Renewal   -> LifecycleService.Renew(sub, plan, now)
     - Upgrade   -> SubscriptionManager.Upgrade(userID, newPlanCode)  (prorata)
     - Downgrade -> SubscriptionManager.ScheduleDowngrade(userID, code) lalu ApplyScheduledDowngrade saat periode berakhir
4. Setiap perubahan status otomatis invalidate cache entitlement.
```

### 1.3 Kontrak data penting

- **Uang = integer Rupiah** (`price_monthly_idr`, `price_yearly_idr`, `amount_idr`).
- **Siklus = `billing_period_days`** (default 30) sejak `current_period_start`.
- `subscriptions.status` ∈ `{trialing, active, past_due, grace, canceled, expired}`.
- `subscription_events` append-only (audit transisi; `previous_state`/`new_state`).
- Upgrade prorata: `max(0, floor(newPrice×remain/total) − floor(oldPrice×remain/total))`.

## 2. Untuk P06 (Super Admin Dashboard)

### 2.1 Data yang dapat dibaca/dikelola

| Tabel | Aksi super admin |
|---|---|
| `plans` | CRUD paket (harga, periode, aktif) |
| `plan_entitlements` | Set centang fitur (bool) per plan |
| `plan_limits` | Set limit numerik per plan (-1 = unlimited) |
| `subscriptions` | Lihat/extend/force downgrade/cancel |
| `token_periods` | Lihat kuota/terpakai per user |
| `token_ledger` | Lihat mutasi (append-only; `adjust` untuk koreksi) |
| `pricing_rules` | Set base/markup/modality multipliers |

### 2.2 Setelah mengubah plan/entitlement

WAJIB memanggil invalidasi agar perubahan berlaku seketika (tanpa re-login):

```go
// Entitlement cache berversi (per user) + broadcast lintas instance.
entBroadcaster.PublishInvalidate(ctx, affectedUserID)
// atau, untuk perubahan massal pada sebuah plan: invalidate tiap user yang memakai plan tsb.
```

### 2.3 Endpoint yang tersedia

| Endpoint | Fungsi |
|---|---|
| `GET /api/v1/me/entitlements` | Hak akses efektif user (fitur+limit+plan+version) |

## 3. Middleware Penegakan

```go
// Backend otoritatif; super_admin bypass; fail-closed.
mw.RequireFeature("ocr_receipt")            // 403 bila fitur tak ada di plan
mw.WithinLimit("max_transactions", usageFn) // 402 bila limit tercapai
```

`usageFn(ctx, userID, limitKey) (int64, error)` menghitung pemakaian saat ini
(mis. jumlah transaksi) — diimplementasikan saat endpoint terkait dilindungi.

## 4. Metrik (Prometheus)

| Metrik | Arti |
|---|---|
| `fintrack_token_ledger_ops_total{entry_type,result}` | Operasi ledger (reserve/commit/release) |
| `fintrack_token_quota_used{user_id}` | Kuota terpakai per user |
| `fintrack_entitlement_resolve_total{source}` | Resolusi entitlement (cache L1/L2/db) |

## 5. Catatan Integrasi

- **Ledger reserve/commit/release** harus dipanggil membungkus panggilan AI
  (P03 gateway): reserve → call → commit/release.
- **Jangan** menghapus data user saat downgrade (soft-limit).
- Semua mutasi token **append-only**; koreksi memakai entri `adjust`.
