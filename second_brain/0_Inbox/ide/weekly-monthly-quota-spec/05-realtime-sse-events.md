# Realtime Integration & SSE Events

Dokumen ini mendefinisikan integrasi sistem kuota dwi-siklus dengan bus realtime SSE yang sudah ada, termasuk event baru, payload, dispatching, dan cache invalidation.

---

## 1. Event SSE Baru & Modifikasi

### 1.1. Daftar Event

| Event | Trigger | Emitter | Target Channel |
|-------|---------|---------|----------------|
| `usage_changed` (EXISTING, DIPERLUAS) | Setiap usage increment (token commit, transaksi, OCR, voice, AI request) | Backend handler setelah operasi sukses | `user:{userId}` |
| `quota_reset` (EXISTING, DIPERLUAS) | Lazy reset membuat window baru (weekly atau monthly) | `DualCycleEnforcer.Evaluate` saat window baru dibuat | `user:{userId}` |
| `quota_degraded` (BARU) | User masuk mode degraded (weekly habis) | `DualCycleEnforcer.Evaluate` saat status berubah ke degraded | `user:{userId}` |
| `quota_blocked` (BARU) | User diblokir (monthly habis atau degraded cap) | `DualCycleEnforcer.Evaluate` saat status berubah ke blocked | `user:{userId}` |
| `cycle_config_changed` (BARU) | Super admin mengubah plan_limit_cycles | Admin handler PUT cycles | `admin:*` + semua `user:{userId}` yang terpengaruh |

### 1.2. Payload Detail

#### `usage_changed` (DIPERLUAS)

```json
{
  "event": "usage_changed",
  "data": {
    "limit_keys": ["ai_tokens_per_cycle"],
    "reason": "ai_chat_commit",
    "weekly": {
      "limit_key": "ai_tokens_per_cycle",
      "used": 12500,
      "limit": 50000,
      "remaining": 37500,
      "percent": 25
    },
    "monthly": {
      "limit_key": "ai_tokens_per_cycle",
      "used": 42500,
      "limit": 150000,
      "remaining": 107500,
      "percent": 28
    },
    "effective_remaining": 37500,
    "status": "normal",
    "warning_threshold": 80
  }
}
```

**Perubahan**: Penambahan objek `weekly`, `monthly`, `effective_remaining`, dan `status`. Field `limit_keys` tetap ada untuk backward compatibility.

#### `quota_reset` (DIPERLUAS)

```json
{
  "event": "quota_reset",
  "data": {
    "cycle_type": "weekly",
    "limit_keys": ["ai_tokens_per_cycle", "max_transactions", "ocr_scans_per_cycle"],
    "new_window": {
      "start": "2026-10-23T14:30:00+07:00",
      "end": "2026-10-30T14:30:00+07:00"
    }
  }
}
```

**Perubahan**: Penambahan `cycle_type` (`"weekly"` atau `"monthly"`) dan `new_window`. Saat monthly reset, `cycle_type = "monthly"` dan semua weekly counter juga tereset secara implisit (window baru).

#### `quota_degraded` (BARU)

```json
{
  "event": "quota_degraded",
  "data": {
    "limit_key": "ai_tokens_per_cycle",
    "degraded_mode": "model_downgrade",
    "fallback_model": "gpt-4o-mini",
    "weekly_reset_at": "2026-10-23T14:30:00+07:00",
    "message": "Kuota mingguan Kuota AI habis. AI akan menggunakan model gpt-4o-mini."
  }
}
```

#### `quota_blocked` (BARU)

```json
{
  "event": "quota_blocked",
  "data": {
    "limit_key": "ai_tokens_per_cycle",
    "block_reason": "monthly_exhausted",
    "monthly_reset_at": "2026-10-31T14:30:00+07:00",
    "message": "Kuota bulanan Kuota AI habis. Tingkatkan paket untuk melanjutkan."
  }
}
```

#### `cycle_config_changed` (BARU)

```json
{
  "event": "cycle_config_changed",
  "data": {
    "plan_id": "a0000000-0000-0000-0000-000000000002",
    "plan_code": "pro",
    "changed_keys": ["ai_tokens_per_cycle", "ocr_scans_per_cycle"]
  }
}
```

---

## 2. Backend Event Emitting

### 2.1. Emitter Points

#### Saat Usage Increment

```go
// Di setiap handler yang increment usage:
func (h *AIHandler) commitTokens(ctx context.Context, ...) {
    // ... existing commit logic ...

    // BARU: Increment weekly counter.
    if weeklyCounter != nil {
        h.usageRepo.IncrementUsage(ctx, weeklyCounter.ID, committedTokens, weeklyLimit)
    }

    // BARU: Emit usage_changed dengan dual-cycle info.
    emitUsageChanged(ctx, h.emitter, userID, "ai_chat_commit",
        WithWeeklyInfo(weeklyUsed, weeklyLimit),
        WithMonthlyInfo(monthlyUsed, monthlyLimit),
        WithStatus(computeStatus(weeklyUsed, weeklyLimit, monthlyUsed, monthlyLimit, threshold)),
    )
}
```

#### Saat Lazy Reset

```go
func (e *DualCycleEnforcer) Evaluate(ctx context.Context, ...) (*EvalResult, error) {
    // ... cari/buat counter ...

    if counterCreated {
        // Window baru → emit quota_reset.
        e.emitter.Emit(ctx, userID, "quota_reset", map[string]any{
            "cycle_type": cycleType,
            "limit_keys": []string{limitKey},
            "new_window": map[string]string{
                "start": counter.CycleStart.Format(time.RFC3339),
                "end":   counter.CycleEnd.Format(time.RFC3339),
            },
        })
    }

    // ... evaluate ...
}
```

#### Saat Status Berubah

```go
func (e *DualCycleEnforcer) Evaluate(ctx context.Context, ...) (*EvalResult, error) {
    // ... evaluate ...

    // Cek apakah status berubah dari normal → degraded.
    if result.Degraded && !previouslyDegraded {
        e.emitter.Emit(ctx, userID, "quota_degraded", map[string]any{
            "limit_key":       limitKey,
            "degraded_mode":   result.DegradedMode,
            "fallback_model":  result.FallbackModel,
            "weekly_reset_at": result.ResetAt.Format(time.RFC3339),
        })
    }

    // Cek apakah status berubah ke blocked.
    if !result.Allowed && !result.Degraded {
        e.emitter.Emit(ctx, userID, "quota_blocked", map[string]any{
            "limit_key":        limitKey,
            "block_reason":     result.BlockReason,
            "monthly_reset_at": monthlyEnd.Format(time.RFC3339),
        })
    }
}
```

### 2.2. Optimasi: Throttle Emitting

Untuk menghindari banjir event saat user membuat banyak transaksi cepat:

- `usage_changed`: emit maksimal 1× per 5 detik per user per limit_key. Gunakan in-memory debounce di publisher.
- `quota_reset`, `quota_degraded`, `quota_blocked`: emit segera (state transition, bukan counter update).

---

## 3. Frontend Event Handling

### 3.1. Modifikasi `realtime-dispatcher.ts`

```typescript
// Tambah event types baru:
export type RealtimeEvent =
  | "payment_received"
  | "plan_updated"
  | "quota_reset"
  | "usage_changed"
  | "quota_degraded"    // BARU
  | "quota_blocked"     // BARU
  | "cycle_config_changed" // BARU
  | ... ;

// Di dalam dispatchEvent:
case "quota_degraded":
  return [
    queryKeys.entitlements.me,
    queryKeys.quota.snapshot,
    queryKeys.usage.all,
  ];

case "quota_blocked":
  return [
    queryKeys.entitlements.me,
    queryKeys.quota.snapshot,
    queryKeys.usage.all,
  ];

case "cycle_config_changed":
  return [
    queryKeys.entitlements.me,
    queryKeys.quota.snapshot,
    queryKeys.superAdmin.planCycles(payload.plan_id),
    queryKeys.usage.all,
  ];
```

### 3.2. Toast Handler (BARU)

```typescript
// frontend/lib/quota-toast-handler.ts

import { toast } from "sonner";

const shown = new Set<string>();

export function handleQuotaEvent(event: string, payload: any) {
  const dedupKey = `${event}_${payload.limit_key}_${payload.cycle_type || ""}`;
  if (shown.has(dedupKey)) return;
  shown.add(dedupKey);
  // Clear after window reset.
  setTimeout(() => shown.delete(dedupKey), 60_000);

  switch (event) {
    case "quota_degraded":
      toast.warning(payload.message, {
        description: `Reset: ${formatCountdown(payload.weekly_reset_at)}`,
        duration: 8000,
      });
      break;

    case "quota_blocked":
      toast.error(payload.message, {
        description: `Reset: ${formatCountdown(payload.monthly_reset_at)}`,
        duration: 10000,
        action: {
          label: "Tingkatkan",
          onClick: () => window.location.href = "/user/subscription",
        },
      });
      break;

    case "quota_reset":
      if (payload.cycle_type === "weekly") {
        toast.success("Kuota mingguan telah direset.", { duration: 5000 });
      } else {
        toast.success("Kuota bulanan telah direset. Selamat menikmati siklus baru!", { duration: 5000 });
      }
      // Clear all dedup keys on reset.
      shown.clear();
      break;
  }
}
```

### 3.3. Integrasi dengan SSE Bus

```typescript
// Di dalam useRealtimeStream (existing):
onEvent(event, payload) {
  // Existing: invalidate queries.
  const keys = dispatchEvent(event, payload);
  keys.forEach(k => queryClient.invalidateQueries({ queryKey: k }));

  // BARU: Handle quota toasts.
  if (["quota_degraded", "quota_blocked", "quota_reset"].includes(event)) {
    handleQuotaEvent(event, payload);
  }

  // BARU: In-memory cache patch untuk usage_changed (optimistic).
  if (event === "usage_changed" && payload.weekly) {
    queryClient.setQueryData(queryKeys.quota.snapshot, (old: any) => {
      if (!old) return old;
      const quotas = { ...old.quotas };
      const key = payload.limit_keys?.[0];
      if (key && quotas[key]) {
        quotas[key] = {
          ...quotas[key],
          monthly: payload.monthly,
          weekly: payload.weekly,
          effective_remaining: payload.effective_remaining,
          status: payload.status,
        };
      }
      return { ...old, quotas };
    });
  }
}
```

---

## 4. Cache Invalidation Strategy

### 4.1. Surgical Invalidation

Tidak ada blanket refetch. Setiap event hanya invalidate query keys yang terpengaruh:

| Event | Query Keys Invalidated |
|-------|----------------------|
| `usage_changed` | `entitlements.me`, `quota.snapshot`, `usage.all` |
| `quota_reset` | `entitlements.me`, `quota.snapshot`, `usage.all`, `transactions.all`, `dashboard.all` |
| `quota_degraded` | `entitlements.me`, `quota.snapshot`, `usage.all` |
| `quota_blocked` | `entitlements.me`, `quota.snapshot`, `usage.all` |
| `cycle_config_changed` | `entitlements.me`, `quota.snapshot`, `superAdmin.planCycles`, `usage.all` |

### 4.2. In-Memory Cache Patch

Untuk `usage_changed`: direct `setQueryData` update pada `quota.snapshot` — zero HTTP hit untuk update counter. Surgical invalidation hanya untuk queries lain.

### 4.3. SSE Reconnect

Saat SSE reconnect (existing mechanism):
- Refetch `quota.snapshot` untuk menyinkronkan state yang mungkin missed.
- `last_event_id` menjamin event replay untuk state transitions (degraded, blocked, reset).

---

## 5. Event Contracts (`backend/pkg/contracts/events.go`)

Tambahkan kontrak event baru:

```go
// Quota dual-cycle events.
const (
    EventQuotaDegraded       = "quota_degraded"
    EventQuotaBlocked        = "quota_blocked"
    EventCycleConfigChanged  = "cycle_config_changed"
    // EventQuotaReset dan EventUsageChanged sudah ada.
)
```

---

## 6. Monitoring & Observability

### 6.1. SSE Event Metrics

Log setiap event emit dengan:
- `event_type`
- `user_id`
- `limit_key`
- `cycle_type`
- `latency_ms` (waktu dari trigger ke emit)

### 6.2. Dashboard Admin: Event Stream

Di halaman `/super-admin/metrics` (existing), tambahkan counter:
- "Quota events / menit" — per event type.
- "Degraded users saat ini" — count user dalam mode degraded.
- "Blocked users saat ini" — count user yang terblokir.
