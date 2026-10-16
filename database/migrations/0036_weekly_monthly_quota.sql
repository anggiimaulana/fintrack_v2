-- ============================================================
-- 0036_weekly_monthly_quota.sql
--
-- Skema kuota dwi-siklus (mingguan + bulanan), degraded mode,
-- usage counters rolling, dan konfigurasi notifikasi threshold.
-- ============================================================

-- ------------------------------------------------------------
-- 1. Alter subscriptions: tambah weekly_anchor
-- ------------------------------------------------------------
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS weekly_anchor timestamptz;

COMMENT ON COLUMN subscriptions.weekly_anchor IS
    'Timestamp awal siklus mingguan rolling (7x24 jam dari anchor ini).';

-- ------------------------------------------------------------
-- 2. Alter metric_definitions: tambah icon dan display_order
-- ------------------------------------------------------------
ALTER TABLE metric_definitions
    ADD COLUMN IF NOT EXISTS icon varchar(64) NOT NULL DEFAULT 'gauge',
    ADD COLUMN IF NOT EXISTS display_order integer NOT NULL DEFAULT 0;

-- ------------------------------------------------------------
-- 3. plan_limit_cycles: konfigurasi cycle dan degraded mode per plan & limit_key
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS plan_limit_cycles (
    id                         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id                    uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    limit_key                  varchar(64) NOT NULL,
    cycle_type                 varchar(32) NOT NULL DEFAULT 'monthly_only'
                               CHECK (cycle_type IN ('weekly_monthly', 'monthly_only')),
    weekly_limit               bigint,
    degraded_mode              varchar(32)
                               CHECK (degraded_mode IS NULL OR degraded_mode IN ('hard_block', 'throttle', 'model_downgrade')),
    throttle_rate_amount       integer,
    throttle_rate_window_secs  integer,
    fallback_model             varchar(128),
    max_degraded_usage         bigint,
    created_at                 timestamptz NOT NULL DEFAULT now(),
    updated_at                 timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT uq_plan_limit_cycle UNIQUE (plan_id, limit_key),
    CONSTRAINT chk_weekly_requires_config
        CHECK (
            cycle_type = 'monthly_only'
            OR (weekly_limit IS NOT NULL AND degraded_mode IS NOT NULL)
        ),
    CONSTRAINT chk_throttle_params
        CHECK (
            degraded_mode IS DISTINCT FROM 'throttle'
            OR (throttle_rate_amount IS NOT NULL AND throttle_rate_amount > 0
                AND throttle_rate_window_secs IS NOT NULL AND throttle_rate_window_secs > 0)
        ),
    CONSTRAINT chk_downgrade_model
        CHECK (
            degraded_mode IS DISTINCT FROM 'model_downgrade'
            OR (fallback_model IS NOT NULL AND fallback_model <> '')
        )
);

CREATE INDEX IF NOT EXISTS idx_plan_limit_cycles_plan ON plan_limit_cycles (plan_id);

DROP TRIGGER IF EXISTS on_plan_limit_cycle_updated ON plan_limit_cycles;
CREATE TRIGGER on_plan_limit_cycle_updated
    BEFORE UPDATE ON plan_limit_cycles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- 4. usage_counters: pencatatan penggunaan per window (weekly/monthly)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS usage_counters (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    limit_key         varchar(64) NOT NULL,
    cycle_type        varchar(16) NOT NULL CHECK (cycle_type IN ('weekly', 'monthly')),
    cycle_start       timestamptz NOT NULL,
    cycle_end         timestamptz NOT NULL,
    used              bigint NOT NULL DEFAULT 0 CHECK (used >= 0),
    degraded_used     bigint NOT NULL DEFAULT 0 CHECK (degraded_used >= 0),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT uq_usage_counter UNIQUE (user_id, limit_key, cycle_type, cycle_start)
);

CREATE INDEX IF NOT EXISTS idx_usage_counters_user_active
    ON usage_counters (user_id, limit_key, cycle_type, cycle_end DESC);

DROP TRIGGER IF EXISTS on_usage_counter_updated ON usage_counters;
CREATE TRIGGER on_usage_counter_updated
    BEFORE UPDATE ON usage_counters
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- 5. quota_notification_config: threshold peringatan kuota
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS quota_notification_config (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id           uuid REFERENCES plans(id) ON DELETE CASCADE,
    warning_threshold integer NOT NULL DEFAULT 80
                      CHECK (warning_threshold > 0 AND warning_threshold <= 100),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT uq_quota_notification_plan UNIQUE NULLS NOT DISTINCT (plan_id)
);

DROP TRIGGER IF EXISTS on_quota_notification_config_updated ON quota_notification_config;
CREATE TRIGGER on_quota_notification_config_updated
    BEFORE UPDATE ON quota_notification_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Seed default global (plan_id IS NULL)
INSERT INTO quota_notification_config (plan_id, warning_threshold)
VALUES (NULL, 80)
ON CONFLICT (plan_id) DO NOTHING;

-- ------------------------------------------------------------
-- 6. Permissions to fintrack_app role
-- ------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fintrack_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE plan_limit_cycles TO fintrack_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE usage_counters TO fintrack_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE quota_notification_config TO fintrack_app;
    END IF;
END $$;
