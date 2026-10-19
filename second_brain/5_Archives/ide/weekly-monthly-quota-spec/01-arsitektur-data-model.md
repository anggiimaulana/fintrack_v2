# Arsitektur & Data Model — Sistem Kuota Mingguan + Bulanan

Dokumen ini mendefinisikan skema basis data, strategi migrasi, dan relasi antar-tabel untuk sistem kuota dwi-siklus (mingguan + bulanan) di FinTrack v2.

---

## 1. Prinsip Desain

| # | Prinsip | Penjelasan |
|---|---------|------------|
| P-01 | **Zero Hardcode** | Seluruh konfigurasi limit, cycle type, degraded mode, threshold — disimpan di DB. Kode hanya menyediakan *kemampuan* (resolver); database menyimpan *kebijakan*. |
| P-02 | **Rolling Anchor** | Minggu dan bulan dihitung dari timestamp subscription user, bukan kalender. Presisi menit (`timestamptz`). |
| P-03 | **Lazy Reset** | Tidak ada cron/scheduler. Counter di-reset secara atomik saat user pertama kali mengakses fitur setelah window berakhir. |
| P-04 | **Monthly Caps Weekly** | `remaining = min(weekly_limit - weekly_used, monthly_limit - monthly_used)`. Weekly tidak pernah melebihi monthly. |
| P-05 | **Carry on Upgrade** | Usage yang sudah terpakai tetap dihitung saat upgrade/downgrade. Weekly counter mulai fresh jika sebelumnya tidak ada weekly. |
| P-06 | **Dinamis per Fitur** | Setiap limit type bisa memiliki cycle type berbeda per plan (weekly+monthly atau monthly_only). |

---

## 2. Skema Tabel Baru & Perubahan

### 2.1. Tabel `plan_limit_cycles` (BARU)

Menentukan cycle type dan degraded behavior untuk setiap limit key per plan. Satu plan bisa punya konfigurasi berbeda per limit key.

```sql
CREATE TABLE IF NOT EXISTS plan_limit_cycles (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id               uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    limit_key             varchar(64) NOT NULL,

    -- Tipe siklus: 'weekly_monthly' = ada limit mingguan + bulanan;
    -- 'monthly_only' = hanya bulanan (default, backward-compatible).
    cycle_type            varchar(32) NOT NULL DEFAULT 'monthly_only'
                          CHECK (cycle_type IN ('weekly_monthly', 'monthly_only')),

    -- Limit mingguan (hanya berlaku jika cycle_type = 'weekly_monthly').
    -- NULL = tidak ada limit mingguan. -1 = unlimited mingguan.
    weekly_limit          bigint,

    -- Degraded mode saat weekly habis:
    --   'hard_block' = blokir total sampai reset.
    --   'throttle'   = rate-limit sesuai parameter.
    --   'model_downgrade' = turunkan ke model AI fallback.
    -- NULL jika cycle_type = 'monthly_only'.
    degraded_mode         varchar(32)
                          CHECK (degraded_mode IS NULL OR degraded_mode IN (
                              'hard_block', 'throttle', 'model_downgrade'
                          )),

    -- Parameter throttle (berlaku jika degraded_mode = 'throttle').
    throttle_rate_amount       integer,           -- jumlah request per window
    throttle_rate_window_secs  integer,           -- window dalam detik

    -- Parameter model downgrade (berlaku jika degraded_mode = 'model_downgrade').
    fallback_model             varchar(128),      -- model AI fallback (e.g. 'gpt-4o-mini')

    -- Batas penggunaan dalam mode degraded (opsional).
    -- Jika tercapai, hard block sampai reset berikutnya.
    -- NULL = unlimited dalam mode degraded (sampai monthly habis).
    max_degraded_usage         bigint,

    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT uq_plan_limit_cycle UNIQUE (plan_id, limit_key),

    -- Validasi: weekly_monthly WAJIB punya weekly_limit dan degraded_mode.
    CONSTRAINT chk_weekly_requires_config
        CHECK (
            cycle_type = 'monthly_only'
            OR (weekly_limit IS NOT NULL AND degraded_mode IS NOT NULL)
        ),

    -- Validasi: throttle mode WAJIB punya parameter rate.
    CONSTRAINT chk_throttle_params
        CHECK (
            degraded_mode IS DISTINCT FROM 'throttle'
            OR (throttle_rate_amount IS NOT NULL AND throttle_rate_amount > 0
                AND throttle_rate_window_secs IS NOT NULL AND throttle_rate_window_secs > 0)
        ),

    -- Validasi: model_downgrade WAJIB punya fallback_model.
    CONSTRAINT chk_downgrade_model
        CHECK (
            degraded_mode IS DISTINCT FROM 'model_downgrade'
            OR (fallback_model IS NOT NULL AND fallback_model <> '')
        )
);

CREATE OR REPLACE TRIGGER on_plan_limit_cycle_updated
    BEFORE UPDATE ON plan_limit_cycles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_plan_limit_cycles_plan
    ON plan_limit_cycles (plan_id);
```

### 2.2. Tabel `usage_counters` (BARU)

Menyimpan counter penggunaan per user per limit key per cycle window. Menggantikan kebutuhan `token_periods` untuk fitur non-token (transaksi, OCR, voice, AI request). Token tetap menggunakan `token_periods` + `token_ledger` yang sudah ada.

```sql
CREATE TABLE IF NOT EXISTS usage_counters (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    limit_key         varchar(64) NOT NULL,

    -- Cycle type: 'weekly' atau 'monthly'.
    cycle_type        varchar(16) NOT NULL CHECK (cycle_type IN ('weekly', 'monthly')),

    -- Anchor: timestamp awal window ini.
    cycle_start       timestamptz NOT NULL,
    -- Anchor + durasi: timestamp akhir window ini.
    cycle_end         timestamptz NOT NULL,

    -- Counter utama.
    used              bigint NOT NULL DEFAULT 0 CHECK (used >= 0),

    -- Counter degraded: berapa banyak request yang dilayani dalam mode
    -- degraded (throttle/downgrade) selama window ini.
    degraded_used     bigint NOT NULL DEFAULT 0 CHECK (degraded_used >= 0),

    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),

    -- Satu row per user per limit_key per cycle_type per window.
    CONSTRAINT uq_usage_counter UNIQUE (user_id, limit_key, cycle_type, cycle_start)
);

CREATE INDEX IF NOT EXISTS idx_usage_counters_user_active
    ON usage_counters (user_id, limit_key, cycle_type, cycle_end DESC);

CREATE OR REPLACE TRIGGER on_usage_counter_updated
    BEFORE UPDATE ON usage_counters
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

### 2.3. Perubahan Tabel `subscriptions`

Tambah kolom anchor untuk weekly cycle:

```sql
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS weekly_anchor timestamptz;

COMMENT ON COLUMN subscriptions.weekly_anchor IS
    'Timestamp pertama kali weekly cycle dimulai. '
    'Diisi saat subscription dibuat atau saat plan di-upgrade ke plan yang '
    'memiliki weekly limits. Rolling: setiap 7×24 jam dari anchor ini.';
```

### 2.4. Perubahan Tabel `metric_definitions`

Tambah kolom metadata display untuk mendukung UI dinamis:

```sql
ALTER TABLE metric_definitions
    ADD COLUMN IF NOT EXISTS icon varchar(64) NOT NULL DEFAULT 'gauge',
    ADD COLUMN IF NOT EXISTS display_order integer NOT NULL DEFAULT 0;

COMMENT ON COLUMN metric_definitions.icon IS
    'Nama ikon lucide-react untuk tampilan di UI quota card.';
```

### 2.5. Tabel `quota_notification_config` (BARU)

Konfigurasi threshold notifikasi per plan (default global + override per plan).

```sql
CREATE TABLE IF NOT EXISTS quota_notification_config (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- plan_id NULL = konfigurasi global default.
    plan_id           uuid REFERENCES plans(id) ON DELETE CASCADE,
    -- Threshold persen saat notifikasi dikirim (e.g. 80).
    warning_threshold integer NOT NULL DEFAULT 80
                      CHECK (warning_threshold > 0 AND warning_threshold <= 100),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),

    -- Hanya satu config global (plan_id IS NULL) dan satu per plan.
    CONSTRAINT uq_quota_notification_plan UNIQUE (plan_id)
);

-- Seed: default global 80%.
INSERT INTO quota_notification_config (plan_id, warning_threshold)
VALUES (NULL, 80)
ON CONFLICT (plan_id) DO NOTHING;

CREATE OR REPLACE TRIGGER on_quota_notification_config_updated
    BEFORE UPDATE ON quota_notification_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

---

## 3. Relasi Antar-Tabel (ERD)

```
plans
  ├── plan_limits          (1:N) — limit_key → limit_value (BULANAN, existing)
  ├── plan_limit_cycles    (1:N) — limit_key → cycle config + degraded behavior
  ├── plan_entitlements    (1:N) — feature switches (existing)
  └── quota_notification_config (1:1) — warning threshold per plan

subscriptions
  ├── plan_id → plans      (N:1)
  ├── weekly_anchor        — titik awal rolling weekly
  ├── current_period_start — titik awal rolling monthly
  └── current_period_end   — akhir rolling monthly

usage_counters
  ├── user_id → users      (N:1)
  └── (user_id, limit_key, cycle_type, cycle_start) — UNIQUE

metric_definitions         — katalog dinamis limit types (existing, extended)
  ├── icon                 — ikon UI (BARU)
  └── display_order        — urutan tampil (BARU)

token_periods              — khusus token (existing, TIDAK DIUBAH)
token_ledger               — append-only log token (existing, TIDAK DIUBAH)
```

---

## 4. Strategi Backward Compatibility

### 4.1. Plan Tanpa Weekly (`monthly_only`)

Jika `plan_limit_cycles` tidak memiliki row untuk suatu `(plan_id, limit_key)`, sistem memperlakukannya sebagai `monthly_only` — perilaku identik dengan sistem saat ini. Ini menjamin:

- Plan `Free` dan `Pro` yang sudah ada tetap berfungsi tanpa modifikasi.
- Migrasi tidak memerlukan seeding wajib untuk setiap plan.

### 4.2. `plan_limits.limit_value` = Monthly Limit

Tabel `plan_limits` yang sudah ada menyimpan `limit_value` sebagai **monthly limit**. Ini tidak berubah. Kolom `plan_limit_cycles.weekly_limit` menambahkan dimensi baru di atasnya.

### 4.3. Token-Specific: `token_periods` + `token_ledger`

Untuk `limit_key = 'ai_tokens_per_cycle'`, mekanisme `token_periods` + `token_ledger` (reserve → commit/release) tetap menjadi sumber kebenaran untuk **monthly**. Weekly token counter menggunakan `usage_counters` sebagai soft-limit overlay:

1. `usage_counters(weekly)` melacak token yang **committed** (bukan reserved) per minggu.
2. Sebelum reserve, enforcement cek: `min(token_period_remaining, weekly_remaining)`.
3. Setelah commit, increment `usage_counters.used` untuk weekly window aktif.

Ini menghindari perubahan pada append-only `token_ledger`.

---

## 5. Lazy Reset: Mekanisme Detail

### 5.1. Alur Lazy Reset

```
User Request → Middleware Cek Kuota
  ↓
  Cari usage_counters WHERE user_id AND limit_key AND cycle_type
    AND cycle_start <= now < cycle_end
  ↓
  Row ada?
  ├── YA → Gunakan row, cek limit.
  └── TIDAK → Hitung window baru:
       ├── weekly:  anchor = subscriptions.weekly_anchor
       │            cycle_start = anchor + N×7d (dimana N×7d <= now < (N+1)×7d)
       │            cycle_end   = cycle_start + 7d
       └── monthly: cycle_start = subscriptions.current_period_start
                    cycle_end   = subscriptions.current_period_end
       ↓
       INSERT usage_counters (used=0, degraded_used=0)
       ON CONFLICT DO NOTHING  ← race-safe
       ↓
       SELECT kembali → row pasti ada.
```

### 5.2. Window Calculation (Weekly)

```
anchor   = subscriptions.weekly_anchor  (e.g. 2026-10-16T14:30:00+07:00)
now      = current timestamp
elapsed  = now - anchor
weeks    = floor(elapsed / 7 days)
cycle_start = anchor + (weeks × 7 days)
cycle_end   = cycle_start + 7 days
```

### 5.3. Atomicity & Race Safety

- `INSERT ... ON CONFLICT (user_id, limit_key, cycle_type, cycle_start) DO NOTHING` mencegah duplicate row saat dua request konkuren menciptakan window baru.
- Increment `used` menggunakan `UPDATE ... SET used = used + $delta WHERE id = $id AND used + $delta <= $limit` — atomik, fail-closed.
- Seluruh operasi dalam satu transaksi database.

---

## 6. Migrasi Database

### 6.1. File Migrasi

```
database/migrations/0036_weekly_monthly_quota.sql
```

### 6.2. Strategi Migrasi

1. **Additive only**: Semua perubahan adalah `CREATE TABLE IF NOT EXISTS` dan `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`. Tidak ada drop/rename.
2. **Idempotent**: Safe untuk di-run ulang.
3. **Zero downtime**: Tidak ada exclusive lock pada tabel yang sudah ada. `ALTER TABLE subscriptions ADD COLUMN` pada tabel kecil (<100k rows) selesai instan.
4. **Backward compatible**: Aplikasi v-lama yang belum paham weekly masih berjalan karena `plan_limit_cycles` kosong = `monthly_only`.

### 6.3. Seed Data

Seed default untuk plan yang sudah ada (`Free`, `Pro`, `Business`) **tidak** menambahkan weekly limits. Super admin mengaktifkan weekly limits dari UI setelah deploy. Hanya `quota_notification_config` global (80%) yang di-seed.

---

## 7. Retensi Data & Cleanup

### 7.1. `usage_counters` Historis

Row `usage_counters` yang sudah lewat (`cycle_end < now`) tidak dihapus otomatis — berguna untuk:
- Riwayat usage di halaman `/user/usage`.
- Audit trail di panel admin.
- Analitik trend penggunaan.

### 7.2. Cleanup Opsional

Super admin dapat mengonfigurasi retensi (misalnya 6 bulan) melalui setting `usage_counter_retention_days`. Cleanup dilakukan oleh admin script, bukan cron otomatis.

```sql
INSERT INTO setting_definitions
    (key, kind, min_value, max_value, default_value, group_key, label, description, sort_order)
VALUES
    ('usage_counter_retention_days', 'int', 30, 3650, '180'::jsonb,
     'kuota', 'Retensi data penggunaan (hari)',
     'Berapa hari data usage_counters historis disimpan. Cleanup manual via admin.', 70)
ON CONFLICT (key) DO NOTHING;
```
