# Mitigasi Risiko & Security

Dokumen ini memetakan seluruh potensi risiko kegagalan, celah keamanan, dan strategi mitigasi teknis untuk sistem kuota dwi-siklus di FinTrack v2.

---

## 1. Matriks Analisis Risiko

| ID | Area | Deskripsi Risiko | Keparahan | Probabilitas | Mitigasi |
|----|------|-----------------|-----------|-------------|----------|
| **R-01** | Race Condition | Dua request konkuren membuat dua window `usage_counters` untuk user+key+cycle yang sama. | Tinggi | Sedang | `INSERT ... ON CONFLICT (user_id, limit_key, cycle_type, cycle_start) DO NOTHING` + `SELECT` setelahnya. UNIQUE constraint menjamin atomicity. |
| **R-02** | Race Condition | Dua request konkuren increment counter melampaui limit. | Tinggi | Tinggi | `UPDATE ... SET used = used + $delta WHERE id = $id AND used + $delta <= $limit RETURNING used`. Atomik di level DB — jika UPDATE mengembalikan 0 rows, request ditolak. |
| **R-03** | SQL Injection | Input `limit_key` atau `cycle_type` mengandung payload SQL. | Kritis | Rendah | Semua query parameterized (`$1, $2, ...`). `limit_key` divalidasi terhadap `metric_definitions.key`. `cycle_type` divalidasi terhadap enum `IN ('weekly', 'monthly')`. |
| **R-04** | Privilege Escalation | User biasa mengakses admin cycle config endpoint. | Kritis | Rendah | `RequireRole("super_admin")` middleware pada semua admin routes. User ID dari JWT, bukan request body. |
| **R-05** | Quota Bypass | User memanipulasi `weekly_anchor` untuk mendapatkan reset lebih awal. | Kritis | Rendah | `weekly_anchor` hanya dapat diubah oleh backend saat subscription create/upgrade. Tidak ada API endpoint untuk user mengubah anchor. Kolom tidak disertakan di response user. |
| **R-06** | Data Inconsistency | Token `used_tokens` di `token_periods` dan `used` di `usage_counters` (weekly) tidak sinkron. | Sedang | Sedang | Weekly counter di-increment di transaksi yang sama dengan commit `token_ledger`. Jika commit gagal, weekly counter tidak di-increment (rollback). |
| **R-07** | Clock Skew | Server time drift menyebabkan window calculation error. | Sedang | Rendah | Gunakan `now()` PostgreSQL (bukan Go `time.Now()`) untuk semua window calculation. DB clock adalah single source of truth. |
| **R-08** | Cascade Delete | Plan dihapus → `plan_limit_cycles` cascade delete → user kehilangan cycle config. | Sedang | Rendah | Plan tidak pernah di-DELETE (soft-delete via `is_active = false`). `ON DELETE CASCADE` hanya sebagai safety net. Admin UI tidak menyediakan hard delete. |
| **R-09** | Memory Leak | In-memory debounce untuk SSE emit (§ Realtime doc) menumpuk entries untuk user yang tidak pernah di-cleanup. | Rendah | Sedang | Gunakan TTL-based map: entry expire setelah 30 detik. Alternatively, gunakan LRU cache dengan max 10.000 entries. |
| **R-10** | Denial of Service | Attacker membuat banyak request cepat untuk memicu lazy reset berulang. | Sedang | Rendah | Lazy reset menggunakan `ON CONFLICT DO NOTHING` — request kedua dan seterusnya menggunakan row yang sudah ada. Tidak ada write amplification. Rate limiter global (existing) membatasi total requests. |
| **R-11** | Stale Cache | User melihat sisa kuota lama setelah cycle config berubah. | Rendah | Sedang | SSE event `cycle_config_changed` invalidate `quota.snapshot` dan `entitlements.me` pada semua user yang terpengaruh. Fallback: `staleTime = 30s` + polling 60s pada `useQuota`. |
| **R-12** | Timezone Edge Case | User di zona waktu berbeda melihat countdown yang tidak akurat. | Rendah | Rendah | Semua timestamp `timestamptz` (UTC). Countdown dihitung di frontend dari UTC timestamp. Display menggunakan `Intl.DateTimeFormat` dengan zona waktu user. |
| **R-13** | Infinite Loop | Lazy reset di endpoint A men-trigger SSE, yang men-trigger refetch di frontend, yang men-trigger endpoint B, dst. | Sedang | Rendah | SSE event tidak men-trigger backend request. Frontend hanya refetch `GET /me/quota` (read-only, no side effect). Lazy reset hanya terjadi di enforcement middleware, bukan di read endpoints. |
| **R-14** | Degraded Cap Overflow | `degraded_used` di-increment melampaui `max_degraded_usage` karena race condition. | Sedang | Sedang | Sama dengan R-02: `UPDATE ... SET degraded_used = degraded_used + $delta WHERE id = $id AND degraded_used + $delta <= $max RETURNING degraded_used`. Atomik di level DB. |

---

## 2. Security Hardening

### 2.1. Input Validation Layers

```
Layer 1: HTTP Handler → JSON binding + field validation (gin binding tags).
Layer 2: Service Layer → Business rule validation (weekly ≤ monthly, dll).
Layer 3: Database → CHECK constraints (enum, range, NOT NULL).
```

Tiga layer redundan: jika satu layer gagal, layer berikutnya menangkap.

### 2.2. Parameterized Queries — Enforcement

Semua repository methods WAJIB menggunakan parameterized queries. Contoh pattern yang ditegakkan:

```go
// DIIZINKAN:
_, err := tx.Exec(ctx, `
    INSERT INTO usage_counters (user_id, limit_key, cycle_type, cycle_start, cycle_end)
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (user_id, limit_key, cycle_type, cycle_start) DO NOTHING
`, userID, limitKey, cycleType, cycleStart, cycleEnd)

// DILARANG (compile-time review):
// query := "INSERT INTO usage_counters ... WHERE limit_key = '" + limitKey + "'"
```

Scan CI/linter: `gosec` rule `G201/G202` mendeteksi string concatenation di SQL.

### 2.3. Authorization Matrix

| Endpoint | Role | Validasi |
|----------|------|----------|
| `GET /api/v1/me/quota` | `user`, `admin`, `super_admin` | user_id dari JWT |
| `GET /api/v1/me/entitlements` | `user`, `admin`, `super_admin` | user_id dari JWT |
| `GET /api/v1/admin/plans/:planId/cycles` | `super_admin` | RequireRole middleware |
| `PUT /api/v1/admin/plans/:planId/cycles` | `super_admin` | RequireRole middleware |
| `GET /api/v1/admin/quota-notification-config` | `super_admin` | RequireRole middleware |
| `PUT /api/v1/admin/quota-notification-config` | `super_admin` | RequireRole middleware |

### 2.4. Rate Limiting

- Global rate limit (existing): 100 req/s per user.
- Admin endpoints: 10 req/s per user (existing pattern).
- Throttle mode rate limit: per user per limit_key, configured di `plan_limit_cycles`.

### 2.5. Audit Trail

Semua mutasi admin (cycle config changes) dicatat di `audit_logs` (existing table):

```go
h.audit.Log(ctx, audit.Entry{
    Actor:      adminUserID,
    Action:     "update_plan_cycles",
    Resource:   "plan_limit_cycles",
    ResourceID: planID.String(),
    Details: map[string]any{
        "changed_keys": changedKeys,
        "before":       beforeState,
        "after":        afterState,
    },
})
```

---

## 3. Zero Downtime Deployment Strategy

### 3.1. Database Migration

1. **Additive only**: `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`.
2. **No locks**: Tidak ada `ALTER TABLE ... RENAME`, `DROP COLUMN`, atau `LOCK TABLE`.
3. **No data migration**: Tabel baru kosong. Existing data tidak terpengaruh.
4. **Rollback**: `DROP TABLE IF EXISTS usage_counters; DROP TABLE IF EXISTS plan_limit_cycles; DROP TABLE IF EXISTS quota_notification_config; ALTER TABLE subscriptions DROP COLUMN IF EXISTS weekly_anchor;`

### 3.2. Backend Deployment

1. **Feature flag**: `DualCycleEnforcer` diaktifkan oleh setting `quota_dual_cycle_enabled` (default: `false`).
2. **Gradual rollout**: Aktifkan per plan (via `plan_limit_cycles` existence) — bukan global toggle.
3. **Fallback**: Jika `DualCycleEnforcer` tidak terpasang (nil), middleware jatuh ke existing single-cycle enforcement.
4. **Rolling restart**: Stateless Go binary. Restart satu instance, yang lain tetap serving.

### 3.3. Frontend Deployment

1. **Backward compatible**: Jika `usage[key].weekly` null/undefined, komponen render monthly-only (existing behavior).
2. **Progressive enhancement**: `DualQuotaCard` gracefully degrades ke `QuotaUsageCard` behavior saat weekly data tidak tersedia.
3. **CDN cache**: Next.js static assets di-version oleh build hash. Tidak ada stale JS.

### 3.4. Deployment Sequence

```
1. Deploy database migration (0036)         — additive, no downtime.
2. Deploy backend with feature flag OFF      — code ada, enforcer nil.
3. Deploy frontend with backward compat     — dual card, fallback monthly.
4. Super admin: configure cycles per plan   — data masuk DB.
5. Enable feature flag per plan             — enforcer aktif untuk plan itu.
6. Monitor: check SSE events, usage counters, degraded states.
7. Full rollout: enable for all plans.
```

---

## 4. Fallback & Degradation Strategies

### 4.1. Fallback: Enforcer Failure

Jika `DualCycleEnforcer.Evaluate` gagal (DB error, timeout):

```go
result, err := m.dualCycle.Evaluate(ctx, ...)
if err != nil {
    // Log error.
    logger.Error("dual cycle evaluation failed", "err", err, "user", userID, "key", limitKey)
    // Fallback: gunakan existing single-cycle enforcement (fail-open terhadap weekly,
    // fail-closed terhadap monthly yang sudah ada).
    // JANGAN block user karena error infrastruktur.
    c.Next()
    return
}
```

**Prinsip**: Lebih baik user mendapat lebih dari haknya selama beberapa detik daripada user yang sah terblokir karena transient error.

### 4.2. Fallback: SSE Connection Lost

- Existing reconnect mechanism (exponential backoff 1s–16s + jitter).
- Saat reconnect: refetch `GET /api/v1/me/quota` untuk sync state.
- `last_event_id` untuk replay missed events.

### 4.3. Fallback: Frontend Quota Endpoint Unavailable

- `useQuota` refetch setiap 60s (polling fallback).
- `staleTime = 30s` — user melihat data stale max 30 detik.
- UI menampilkan skeleton/loading state, bukan error page.

---

## 5. Performance Considerations

### 5.1. Database Query Complexity

Enforcement per request = **2 queries** (worst case: lazy reset):

```sql
-- 1. Get or create counter (INSERT + SELECT).
INSERT INTO usage_counters (...) VALUES (...) ON CONFLICT DO NOTHING;
SELECT * FROM usage_counters WHERE user_id=$1 AND limit_key=$2 AND cycle_type=$3 AND cycle_end > $4;

-- 2. Increment counter (atomik UPDATE).
UPDATE usage_counters SET used = used + $1 WHERE id = $2 AND used + $1 <= $3 RETURNING used;
```

**Best case** (counter sudah ada): 1 SELECT + 1 UPDATE = ~2ms.
**Worst case** (lazy reset): 1 INSERT + 1 SELECT + 1 UPDATE = ~4ms.

### 5.2. Index Strategy

```sql
-- Primary lookup: user + key + cycle + active window.
CREATE INDEX idx_usage_counters_user_active
    ON usage_counters (user_id, limit_key, cycle_type, cycle_end DESC);

-- Plan lookup: plan + key.
CREATE INDEX idx_plan_limit_cycles_plan
    ON plan_limit_cycles (plan_id);
```

### 5.3. Caching

- `CycleConfig` per plan: cached in-memory di `DualCycleEnforcer` dengan TTL 30s (sama pattern dengan `SettingsQuotaConfig`). Plan config jarang berubah.
- `usage_counters`: TIDAK di-cache. Selalu read dari DB untuk consistency.
- `quota_notification_config`: cached in-memory, TTL 60s.

### 5.4. Connection Pool

Tidak ada perubahan pada pool size. Enforcement queries ringan (2-4ms) dan menggunakan existing pool. Tidak ada long-running transactions.

---

## 6. Observability & Alerting

### 6.1. Structured Logging

```go
logger.Info("quota_evaluated",
    "user_id", userID,
    "limit_key", limitKey,
    "weekly_used", result.WeeklyUsed,
    "weekly_limit", result.WeeklyLimit,
    "monthly_used", result.MonthlyUsed,
    "monthly_limit", result.MonthlyLimit,
    "status", result.Status,
    "degraded", result.Degraded,
    "latency_us", elapsed.Microseconds(),
)
```

### 6.2. Metric Counters (untuk integrasi Prometheus/Grafana di masa depan)

| Metric | Type | Labels |
|--------|------|--------|
| `quota_evaluations_total` | Counter | `limit_key`, `status` (allow/degraded/block) |
| `quota_lazy_resets_total` | Counter | `limit_key`, `cycle_type` |
| `quota_evaluation_duration_ms` | Histogram | `limit_key` |
| `quota_degraded_users_current` | Gauge | `limit_key`, `degraded_mode` |

### 6.3. Alerting Conditions

| Kondisi | Severity | Aksi |
|---------|----------|------|
| `quota_evaluation_duration_ms > 100ms` (p99) | Warning | Investigasi DB performance |
| `quota_evaluations_total{status=block} / total > 30%` | Warning | Review plan limits |
| Enforcer error rate > 1% | Critical | Cek koneksi DB, consider fallback |
