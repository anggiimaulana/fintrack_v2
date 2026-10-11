-- ============================================================
-- 0008_subscription_token_core.sql
-- Subscription & Token Core: plans, entitlements, limits,
-- subscriptions, subscription events, token periods/ledger, pricing.
--
-- Guarantees:
--   * Additive & idempotent (CREATE TABLE IF NOT EXISTS).
--   * Money = BIGINT (integer Rupiah); tokens = BIGINT. No floats.
--   * token_ledger is append-only (enforced by trigger, like audit_logs).
--   * Quota cycle = per subscription period (NOT daily); reset each cycle.
--
-- Idempotent: safe to re-run.
-- ============================================================

-- ------------------------------------------------------------
-- plans
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS plans (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code               varchar(32) NOT NULL UNIQUE,
    name               varchar(64) NOT NULL,
    description        text,
    price_monthly_idr  bigint NOT NULL DEFAULT 0,
    price_yearly_idr   bigint NOT NULL DEFAULT 0,
    billing_period_days integer NOT NULL DEFAULT 30,
    is_active          boolean NOT NULL DEFAULT true,
    is_public          boolean NOT NULL DEFAULT true,
    fallback_gateway   varchar(32) NOT NULL DEFAULT 'midtrans',
    sort_order         integer NOT NULL DEFAULT 0,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_plan_updated
    BEFORE UPDATE ON plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ------------------------------------------------------------
-- plan_entitlements (feature switches)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS plan_entitlements (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id     uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    feature_key varchar(64) NOT NULL,
    is_enabled  boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_plan_feature UNIQUE (plan_id, feature_key)
);

-- ------------------------------------------------------------
-- plan_limits (numeric limits; -1 = unlimited)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS plan_limits (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id     uuid NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    limit_key   varchar(64) NOT NULL,
    limit_value bigint NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_plan_limit UNIQUE (plan_id, limit_key)
);

-- ------------------------------------------------------------
-- subscriptions (one per user)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS subscriptions (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id                uuid NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    status                 varchar(32) NOT NULL DEFAULT 'active'
                           CHECK (status IN ('trialing','active','past_due','grace','canceled','expired')),
    current_period_start   timestamptz NOT NULL,
    current_period_end     timestamptz,
    cancel_at_period_end   boolean NOT NULL DEFAULT false,
    canceled_at            timestamptz,
    gateway                varchar(32) NOT NULL DEFAULT 'manual',
    gateway_subscription_id varchar(128),
    metadata               jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_subscription UNIQUE (user_id)
);

CREATE OR REPLACE TRIGGER on_subscription_updated
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_subscriptions_status_period ON subscriptions (status, current_period_end);

-- ------------------------------------------------------------
-- subscription_events (append-only audit)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS subscription_events (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id uuid NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type      varchar(64) NOT NULL,
    previous_state  jsonb NOT NULL DEFAULT '{}'::jsonb,
    new_state       jsonb NOT NULL DEFAULT '{}'::jsonb,
    reason          text,
    actor_id        uuid REFERENCES users(id) ON DELETE SET NULL,
    ip_address      varchar(45),
    user_agent      text,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sub_events_user ON subscription_events (user_id, created_at DESC);

-- ------------------------------------------------------------
-- token_periods (one row per user per subscription cycle)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS token_periods (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_start    timestamptz NOT NULL,
    period_end      timestamptz NOT NULL,
    quota_tokens    bigint NOT NULL,
    used_tokens     bigint NOT NULL DEFAULT 0,
    reserved_tokens bigint NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_token_period UNIQUE (user_id, period_start),
    CONSTRAINT chk_token_balance CHECK (used_tokens >= 0 AND reserved_tokens >= 0)
);

CREATE INDEX IF NOT EXISTS idx_token_periods_user_window ON token_periods (user_id, period_start, period_end);

-- ------------------------------------------------------------
-- token_ledger (append-only)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS token_ledger (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id       uuid NOT NULL REFERENCES token_periods(id) ON DELETE RESTRICT,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reservation_id  uuid,
    entry_type      varchar(32) NOT NULL
                    CHECK (entry_type IN ('reserve','commit','release','grant','adjust')),
    amount          bigint NOT NULL,
    modality        varchar(32) NOT NULL DEFAULT 'chat',
    cost_idr        bigint NOT NULL DEFAULT 0,
    idempotency_key varchar(64) NOT NULL UNIQUE,
    reference_id    varchar(128),
    notes           text,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_token_ledger_user ON token_ledger (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_token_ledger_reservation ON token_ledger (reservation_id) WHERE reservation_id IS NOT NULL;

-- Append-only enforcement (no UPDATE/DELETE).
CREATE OR REPLACE FUNCTION token_ledger_block_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'table token_ledger is append-only (no % allowed)', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;

DROP TRIGGER IF EXISTS token_ledger_no_update ON token_ledger;
CREATE TRIGGER token_ledger_no_update
    BEFORE UPDATE ON token_ledger
    FOR EACH ROW EXECUTE FUNCTION token_ledger_block_mutation();

DROP TRIGGER IF EXISTS token_ledger_no_delete ON token_ledger;
CREATE TRIGGER token_ledger_no_delete
    BEFORE DELETE ON token_ledger
    FOR EACH ROW EXECUTE FUNCTION token_ledger_block_mutation();

-- ------------------------------------------------------------
-- pricing_rules (dynamic pricing; global rules)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pricing_rules (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name            varchar(64) NOT NULL UNIQUE,
    base_per_1k_idr      bigint NOT NULL DEFAULT 300,
    markup_multiplier    numeric(6,4) NOT NULL DEFAULT 1.2500,
    markup_percent       integer NOT NULL DEFAULT 0,
    modality_multipliers jsonb NOT NULL DEFAULT '{"chat":1.0,"vision":2.5,"voice":3.0,"embedding":0.5,"reasoning":2.0}'::jsonb,
    is_active            boolean NOT NULL DEFAULT true,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_pricing_rule_updated
    BEFORE UPDATE ON pricing_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
