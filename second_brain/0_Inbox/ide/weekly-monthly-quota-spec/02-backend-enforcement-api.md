# Backend Enforcement & API Contracts

Dokumen ini mendefinisikan mekanisme enforcement kuota dwi-siklus di backend Go, termasuk middleware, service layer, API endpoint, dan kontrak respons.

---

## 1. Enforcement Flow Overview

```
HTTP Request
  ↓
  AuthMiddleware (existing)
  ↓
  EntitlementMiddleware.WithinLimit(limitKey, usageResolver)
  ↓ (DIMODIFIKASI)
  DualCycleEnforcer.Evaluate(ctx, userID, limitKey)
  ↓
  ┌──────────────────────────────────────────────────────┐
  │ 1. Resolve plan_limit_cycles untuk (plan_id, key)    │
  │ 2. Lazy reset: cari/buat usage_counters              │
  │ 3. Hitung remaining:                                 │
  │    monthly_remaining = monthly_limit - monthly_used   │
  │    weekly_remaining  = weekly_limit  - weekly_used    │
  │    effective = min(weekly_remaining, monthly_remaining)│
  │ 4. Tentukan status:                                  │
  │    ├── effective > 0 → ALLOW                         │
  │    ├── weekly_exhausted && degraded_mode → DEGRADED  │
  │    └── monthly_exhausted → DENY (402)                │
  └──────────────────────────────────────────────────────┘
```

---

## 2. Komponen Backend Baru & Modifikasi

### 2.1. `DualCycleEnforcer` (BARU)

```go
// Package quota implements the dual-cycle (weekly + monthly) enforcement.
package quota

type CycleConfig struct {
    CycleType            string // "weekly_monthly" | "monthly_only"
    WeeklyLimit          int64
    MonthlyLimit         int64  // dari plan_limits.limit_value
    DegradedMode         string // "hard_block" | "throttle" | "model_downgrade"
    ThrottleRateAmount   int    // request per window
    ThrottleRateWindowS  int    // window dalam detik
    FallbackModel        string
    MaxDegradedUsage     int64  // max usage dalam mode degraded; 0 = unlimited
}

type EvalResult struct {
    Allowed          bool
    Degraded         bool    // true jika weekly habis tapi degraded mode aktif
    DegradedMode     string  // "" | "throttle" | "model_downgrade"
    FallbackModel    string  // hanya saat DegradedMode = "model_downgrade"
    WeeklyUsed       int64
    WeeklyLimit      int64
    WeeklyRemaining  int64
    MonthlyUsed      int64
    MonthlyLimit     int64
    MonthlyRemaining int64
    Effective        int64   // min(weekly_remaining, monthly_remaining)
    ResetAt          time.Time // kapan window terdekat reset
    BlockReason      string  // "weekly_exhausted" | "monthly_exhausted" | "degraded_cap"
}
```

### 2.2. Interface `CycleConfigResolver` (BARU)

```go
type CycleConfigResolver interface {
    // ResolveCycle mengembalikan konfigurasi cycle untuk (planID, limitKey).
    // Mengembalikan nil jika plan tidak punya cycle config → monthly_only default.
    ResolveCycle(ctx context.Context, planID uuid.UUID, limitKey string) (*CycleConfig, error)
}
```

### 2.3. Interface `UsageCounterRepository` (BARU)

```go
type UsageCounterRepository interface {
    // GetOrCreateCounter mendapatkan atau membuat counter untuk window aktif.
    // Atomik: ON CONFLICT DO NOTHING + SELECT.
    GetOrCreateCounter(ctx context.Context, userID uuid.UUID, limitKey, cycleType string,
        cycleStart, cycleEnd time.Time) (*UsageCounter, error)

    // IncrementUsage menambah counter secara atomik. Mengembalikan error jika
    // used + delta > hardLimit (fail-closed).
    IncrementUsage(ctx context.Context, counterID uuid.UUID, delta int64, hardLimit int64) error

    // IncrementDegradedUsage menambah degraded_used counter.
    IncrementDegradedUsage(ctx context.Context, counterID uuid.UUID, delta int64, maxDegraded int64) error

    // GetCurrentCounters mengembalikan semua counter aktif untuk user (untuk display).
    GetCurrentCounters(ctx context.Context, userID uuid.UUID, now time.Time) ([]UsageCounter, error)
}
```

### 2.4. Modifikasi `EntitlementMiddleware.WithinLimit`

`WithinLimit` diperluas untuk menangani dual-cycle:

```go
func (m *EntitlementMiddleware) WithinLimit(limitKey string, usage UsageResolver) gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... existing auth + resolve code ...

        // BARU: Cek dual-cycle jika enforcer terpasang.
        if m.dualCycle != nil {
            result, err := m.dualCycle.Evaluate(ctx, userID, limitKey, planID)
            if err != nil {
                response.InternalError(c, "Failed to evaluate quota")
                c.Abort()
                return
            }
            if !result.Allowed && !result.Degraded {
                // Hard block: 402.
                c.JSON(402, gin.H{
                    "success":           false,
                    "error":             "limit_exceeded",
                    "message":           "Anda telah mencapai batas paket Anda.",
                    "limit_key":         limitKey,
                    "block_reason":      result.BlockReason,
                    "weekly_used":       result.WeeklyUsed,
                    "weekly_limit":      result.WeeklyLimit,
                    "monthly_used":      result.MonthlyUsed,
                    "monthly_limit":     result.MonthlyLimit,
                    "reset_at":          result.ResetAt.Format(time.RFC3339),
                    "upgrade_hint":      true,
                    "upgrade_url":       UpgradeURL,
                })
                c.Abort()
                return
            }
            if result.Degraded {
                // Set context values untuk handler downstream.
                c.Set("quota_degraded", true)
                c.Set("quota_degraded_mode", result.DegradedMode)
                if result.FallbackModel != "" {
                    c.Set("quota_fallback_model", result.FallbackModel)
                }
            }
            c.Next()
            return
        }

        // Fallback: existing single-cycle enforcement.
        // ... existing code ...
    }
}
```

### 2.5. Modifikasi AI Handler untuk Degraded Mode

```go
func (h *AIHandler) Chat(c *gin.Context) {
    // ... existing code ...

    // BARU: Cek degraded mode dari context.
    if degraded, _ := c.Get("quota_degraded"); degraded == true {
        mode, _ := c.GetString("quota_degraded_mode")
        switch mode {
        case "model_downgrade":
            fallbackModel, _ := c.GetString("quota_fallback_model")
            if fallbackModel != "" {
                req.Model = fallbackModel  // Override model ke fallback.
            }
        case "throttle":
            // Throttle di-enforce oleh rate limiter terpisah
            // (lihat §2.6).
        }
    }

    // ... lanjut existing flow ...
}
```

### 2.6. Rate Limiter untuk Throttle Mode (BARU)

```go
// ThrottleGuard menerapkan rate-limit saat user dalam mode degraded.
// Menggunakan sliding window counter di Redis (atau in-memory fallback).
type ThrottleGuard struct {
    store ThrottleStore
}

type ThrottleStore interface {
    // AllowRequest mengembalikan true jika request diizinkan
    // berdasarkan rate limit (amount request per windowSecs detik).
    AllowRequest(ctx context.Context, userID uuid.UUID, limitKey string,
        amount int, windowSecs int) (bool, error)
}
```

Implementasi: sliding window counter menggunakan `usage_counters.degraded_used` + timestamp check. Tidak memerlukan Redis — cukup atomik DB query.

---

## 3. API Endpoints

### 3.1. Existing Endpoints yang Dimodifikasi

#### `GET /api/v1/me/entitlements` (DIPERLUAS)

Response ditambah field baru:

```json
{
  "plan_code": "pro",
  "plan_name": "Pro",
  "status": "active",
  "features": { "ai_chat": true, "ocr_receipt": true },
  "limits": {
    "ai_tokens_per_cycle": 150000,
    "ai_requests_per_cycle": 2000
  },
  "usage": {
    "ai_tokens_per_cycle": {
      "used": 42000,
      "limit": 150000,
      "remaining": 108000,
      "unlimited": false,
      "unit": "kuota",
      "weekly": {
        "used": 12000,
        "limit": 50000,
        "remaining": 38000,
        "cycle_start": "2026-10-16T14:30:00+07:00",
        "cycle_end": "2026-10-23T14:30:00+07:00",
        "degraded": false,
        "degraded_mode": "model_downgrade",
        "degraded_used": 0,
        "max_degraded_usage": 10000,
        "fallback_model": "gpt-4o-mini"
      }
    },
    "ai_requests_per_cycle": {
      "used": 150,
      "limit": 2000,
      "remaining": 1850,
      "unlimited": false,
      "unit": "permintaan",
      "weekly": null
    }
  },
  "period": {
    "start": "2026-10-01T14:30:00+07:00",
    "end": "2026-10-31T14:30:00+07:00"
  },
  "weekly_anchor": "2026-10-01T14:30:00+07:00",
  "version": 42,
  "quota_ratio": 5
}
```

**Perubahan key**:
- `usage[key].weekly` — objek baru, `null` jika `monthly_only`.
- `weekly_anchor` — timestamp anchor untuk weekly cycle.

#### `402` Response (DIPERLUAS)

Saat limit tercapai, respons 402 diperkaya:

```json
{
  "success": false,
  "error": "limit_exceeded",
  "message": "Kuota mingguan token AI Anda telah habis. Reset dalam 2 hari 5 jam.",
  "limit_key": "ai_tokens_per_cycle",
  "block_reason": "weekly_exhausted",
  "weekly_used": 50000,
  "weekly_limit": 50000,
  "monthly_used": 142000,
  "monthly_limit": 150000,
  "reset_at": "2026-10-23T14:30:00+07:00",
  "degraded_available": true,
  "degraded_mode": "model_downgrade",
  "fallback_model": "gpt-4o-mini",
  "upgrade_hint": true,
  "upgrade_url": "/user/subscription"
}
```

### 3.2. Endpoint Baru: Admin Cycle Config

#### `GET /api/v1/admin/plans/:planId/cycles`

Mengembalikan konfigurasi cycle untuk semua limit keys plan.

```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "limit_key": "ai_tokens_per_cycle",
      "cycle_type": "weekly_monthly",
      "weekly_limit": 50000,
      "degraded_mode": "model_downgrade",
      "throttle_rate_amount": null,
      "throttle_rate_window_secs": null,
      "fallback_model": "gpt-4o-mini",
      "max_degraded_usage": 10000
    }
  ]
}
```

#### `PUT /api/v1/admin/plans/:planId/cycles`

Upsert batch konfigurasi cycle.

```json
{
  "cycles": [
    {
      "limit_key": "ai_tokens_per_cycle",
      "cycle_type": "weekly_monthly",
      "weekly_limit": 50000,
      "degraded_mode": "model_downgrade",
      "fallback_model": "gpt-4o-mini",
      "max_degraded_usage": 10000
    },
    {
      "limit_key": "ocr_scans_per_cycle",
      "cycle_type": "weekly_monthly",
      "weekly_limit": 30,
      "degraded_mode": "hard_block"
    }
  ]
}
```

**Validasi server-side**:
- `weekly_limit` wajib ada dan > 0 jika `cycle_type = 'weekly_monthly'`.
- `degraded_mode` wajib ada jika `cycle_type = 'weekly_monthly'`.
- `fallback_model` wajib ada dan non-empty jika `degraded_mode = 'model_downgrade'`.
- `throttle_rate_amount` > 0 dan `throttle_rate_window_secs` > 0 jika `degraded_mode = 'throttle'`.
- `weekly_limit` harus ≤ `plan_limits.limit_value` (monthly) untuk key yang sama. Weekly lebih besar dari monthly tidak masuk akal.

#### `GET /api/v1/admin/quota-notification-config`

#### `PUT /api/v1/admin/quota-notification-config`

```json
{
  "global_threshold": 80,
  "plan_overrides": [
    { "plan_id": "uuid", "threshold": 90 }
  ]
}
```

### 3.3. Endpoint Baru: User Quota Detail

#### `GET /api/v1/me/quota`

Endpoint khusus yang mengembalikan snapshot lengkap kuota user — dual progress bars, countdown, status.

```json
{
  "success": true,
  "data": {
    "plan_code": "pro",
    "weekly_anchor": "2026-10-01T14:30:00+07:00",
    "monthly_period": {
      "start": "2026-10-01T14:30:00+07:00",
      "end": "2026-10-31T14:30:00+07:00"
    },
    "quotas": {
      "ai_tokens_per_cycle": {
        "label": "Kuota AI",
        "unit": "kuota",
        "icon": "brain",
        "monthly": { "used": 42000, "limit": 150000, "remaining": 108000, "unlimited": false },
        "weekly": { "used": 12000, "limit": 50000, "remaining": 38000, "unlimited": false },
        "effective_remaining": 38000,
        "status": "normal",
        "degraded_mode": "model_downgrade",
        "degraded_config": {
          "fallback_model": "gpt-4o-mini",
          "max_degraded_usage": 10000,
          "degraded_used": 0
        },
        "weekly_resets_at": "2026-10-23T14:30:00+07:00",
        "monthly_resets_at": "2026-10-31T14:30:00+07:00"
      }
    },
    "warning_threshold": 80
  }
}
```

**Field `status`** dihitung server-side:
- `"normal"` — `effective_remaining / effective_limit > warning_threshold%`
- `"warning"` — `effective_remaining / effective_limit ≤ warning_threshold%`
- `"degraded_throttle"` — weekly habis, mode throttle aktif.
- `"degraded_downgrade"` — weekly habis, mode model_downgrade aktif.
- `"blocked"` — monthly habis atau degraded cap tercapai.

---

## 4. Enforcement: Token Khusus

Token menggunakan reserve → commit flow di `token_ledger`. Weekly enforcement untuk token:

### 4.1. Reserve Phase

```
1. Hitung monthly_remaining dari token_periods (existing).
2. Hitung weekly_remaining dari usage_counters (weekly, ai_tokens_per_cycle).
3. effective_remaining = min(monthly_remaining, weekly_remaining).
4. Jika effective_remaining < estimated_cost:
   a. Cek degraded_mode:
      - hard_block → 402
      - model_downgrade → override model, reserve dengan estimasi model murah
      - throttle → cek rate limit, jika allowed → reserve normal
5. Reserve dari token_periods (existing flow).
```

### 4.2. Commit Phase

```
1. Commit ke token_periods (existing flow) → monthly counter terupdate.
2. Increment usage_counters(weekly, ai_tokens_per_cycle, committed_tokens).
   → Atomik, satu transaksi.
```

### 4.3. Release Phase

Tidak ada perubahan pada release — token yang di-release kembali ke `token_periods`, dan weekly counter tidak di-decrement (konservatif: lebih baik over-count weekly daripada under-count).

---

## 5. Subscription Lifecycle Integration

### 5.1. Subscribe / Activate

```
1. Set subscriptions.weekly_anchor = current_period_start.
2. Jika plan punya weekly limits: buat usage_counters (weekly) untuk window pertama.
3. Buat usage_counters (monthly) untuk semua enforced limit keys.
```

### 5.2. Upgrade (Carry Usage, Extend Limit)

```
1. Plan berubah: resolusi plan_limit_cycles berubah.
2. Monthly usage tetap (existing behavior via QuotaSyncer).
3. Jika plan baru punya weekly tapi plan lama tidak:
   a. Set weekly_anchor = now.
   b. Buat usage_counters(weekly) fresh (used=0).
4. Jika plan baru tidak punya weekly tapi plan lama punya:
   a. Weekly counters jadi orphan (tidak dihapus, tidak dipakai).
```

### 5.3. Downgrade (End of Cycle)

```
1. Limit monthly turun ke plan baru.
2. Jika usage sudah melebihi limit baru → status = blocked sampai reset.
3. Weekly config berubah sesuai plan baru (bisa hilang atau berubah).
```

### 5.4. Cycle Reset (Lazy)

```
Saat user pertama kali request di window baru:
1. usage_counters(monthly) baru dibuat (used=0).
2. usage_counters(weekly) baru dibuat (used=0).
3. token_periods baru dibuat (existing behavior).
4. Emit SSE event "quota_reset" dengan payload {cycle_type, limit_keys}.
```

---

## 6. Input Validation & SQL Injection Prevention

### 6.1. Parameterized Queries

**SEMUA** query menggunakan parameterized statements (`$1`, `$2`, ...). Tidak ada string concatenation untuk SQL:

```go
// BENAR:
row := tx.QueryRow(ctx, `
    SELECT used FROM usage_counters
    WHERE user_id = $1 AND limit_key = $2 AND cycle_type = $3
      AND cycle_start <= $4 AND cycle_end > $4
`, userID, limitKey, cycleType, now)

// SALAH (DILARANG):
// query := fmt.Sprintf("SELECT used FROM usage_counters WHERE limit_key = '%s'", limitKey)
```

### 6.2. Input Sanitization

- `limit_key`: divalidasi terhadap `metric_definitions.key` (whitelist dari DB).
- `cycle_type`: enum check `IN ('weekly', 'monthly')`.
- `degraded_mode`: enum check `IN ('hard_block', 'throttle', 'model_downgrade')`.
- `fallback_model`: varchar(128), trimmed, non-empty jika diperlukan.
- Semua integer: range check (`> 0`, `<= MAX`).

### 6.3. Authorization

- Cycle config endpoints: `RequireRole("super_admin")` middleware.
- User quota endpoints: auth middleware + `user_id` dari JWT, bukan dari request body.
- Super admin bypass entitlement checks (existing behavior).
