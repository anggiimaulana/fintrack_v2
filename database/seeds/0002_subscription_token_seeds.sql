-- ============================================================
-- seeds/0002_subscription_token_seeds.sql  (SEED — not a migration)
-- Default plans, entitlements, limits, and pricing rules.
-- Idempotent: safe to re-run (ON CONFLICT ... DO UPDATE).
--
-- Source of truth for values: 0_Inbox/ide/.../07-konfigurasi-seed-awal.md
--   Free:  Rp0        / 5.000 tokens per cycle
--   Pro:   Rp39.000   / 150.000 tokens per cycle
--   Business: Rp129.000 / 1.000.000 tokens per cycle
-- Money = integer Rupiah; -1 = unlimited.
-- ============================================================

-- ------------------------------------------------------------
-- A. PLANS (deterministic ids so seeds are referable & idempotent)
-- ------------------------------------------------------------
INSERT INTO plans (id, code, name, description, price_monthly_idr, price_yearly_idr, billing_period_days, is_active, is_public, fallback_gateway, sort_order) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'free',     'Free',     'Paket gratis untuk mulai mencatat keuangan.', 0,      0,       30, true, true, 'manual',   1),
    ('a0000000-0000-0000-0000-000000000002', 'pro',      'Pro',      'Untuk pengguna aktif dengan kebutuhan AI.',  39000,  390000,  30, true, true, 'midtrans', 2),
    ('a0000000-0000-0000-0000-000000000003', 'business', 'Business', 'Untuk power user dan usaha kecil.',          129000, 1290000, 30, true, true, 'midtrans', 3)
ON CONFLICT (code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price_monthly_idr = EXCLUDED.price_monthly_idr,
    price_yearly_idr = EXCLUDED.price_yearly_idr,
    billing_period_days = EXCLUDED.billing_period_days,
    fallback_gateway = EXCLUDED.fallback_gateway,
    sort_order = EXCLUDED.sort_order;

-- ------------------------------------------------------------
-- B. PLAN ENTITLEMENTS (feature switches)
--    Free: chat only. Pro: +vision/voice/rag/reports/budgeting.
--    Business: +api_access.
-- ------------------------------------------------------------
INSERT INTO plan_entitlements (plan_id, feature_key, is_enabled) VALUES
    -- Free
    ('a0000000-0000-0000-0000-000000000001', 'ai_chat',              true),
    ('a0000000-0000-0000-0000-000000000001', 'ocr_receipt',          false),
    ('a0000000-0000-0000-0000-000000000001', 'voice_input',          false),
    ('a0000000-0000-0000-0000-000000000001', 'rag_search',           false),
    ('a0000000-0000-0000-0000-000000000001', 'reports_export',       false),
    ('a0000000-0000-0000-0000-000000000001', 'advanced_budgeting',   false),
    ('a0000000-0000-0000-0000-000000000001', 'api_access',           false),
    ('a0000000-0000-0000-0000-000000000001', 'bank_sync',            false),
    -- Pro
    ('a0000000-0000-0000-0000-000000000002', 'ai_chat',              true),
    ('a0000000-0000-0000-0000-000000000002', 'ocr_receipt',          true),
    ('a0000000-0000-0000-0000-000000000002', 'voice_input',          true),
    ('a0000000-0000-0000-0000-000000000002', 'rag_search',           true),
    ('a0000000-0000-0000-0000-000000000002', 'reports_export',       true),
    ('a0000000-0000-0000-0000-000000000002', 'advanced_budgeting',   true),
    ('a0000000-0000-0000-0000-000000000002', 'api_access',           false),
    ('a0000000-0000-0000-0000-000000000002', 'bank_sync',            false),
    -- Business
    ('a0000000-0000-0000-0000-000000000003', 'ai_chat',              true),
    ('a0000000-0000-0000-0000-000000000003', 'ocr_receipt',          true),
    ('a0000000-0000-0000-0000-000000000003', 'voice_input',          true),
    ('a0000000-0000-0000-0000-000000000003', 'rag_search',           true),
    ('a0000000-0000-0000-0000-000000000003', 'reports_export',       true),
    ('a0000000-0000-0000-0000-000000000003', 'advanced_budgeting',   true),
    ('a0000000-0000-0000-0000-000000000003', 'api_access',           true),
    ('a0000000-0000-0000-0000-000000000003', 'bank_sync',            false)
ON CONFLICT (plan_id, feature_key) DO UPDATE SET is_enabled = EXCLUDED.is_enabled;

-- ------------------------------------------------------------
-- C. PLAN LIMITS (numeric; -1 = unlimited)
-- ------------------------------------------------------------
INSERT INTO plan_limits (plan_id, limit_key, limit_value) VALUES
    -- Free
    ('a0000000-0000-0000-0000-000000000001', 'ai_tokens_per_cycle',   5000),
    ('a0000000-0000-0000-0000-000000000001', 'ai_requests_per_cycle', 50),
    ('a0000000-0000-0000-0000-000000000001', 'max_transactions',      100),
    ('a0000000-0000-0000-0000-000000000001', 'max_categories',        5),
    ('a0000000-0000-0000-0000-000000000001', 'max_wallets',           1),
    ('a0000000-0000-0000-0000-000000000001', 'ocr_scans_per_cycle',   0),
    ('a0000000-0000-0000-0000-000000000001', 'voice_requests_per_day', 1),
    -- Pro
    ('a0000000-0000-0000-0000-000000000002', 'ai_tokens_per_cycle',   150000),
    ('a0000000-0000-0000-0000-000000000002', 'ai_requests_per_cycle', 2000),
    ('a0000000-0000-0000-0000-000000000002', 'max_transactions',      5000),
    ('a0000000-0000-0000-0000-000000000002', 'max_categories',        30),
    ('a0000000-0000-0000-0000-000000000002', 'max_wallets',           5),
    ('a0000000-0000-0000-0000-000000000002', 'ocr_scans_per_cycle',   100),
    ('a0000000-0000-0000-0000-000000000002', 'voice_requests_per_day', 50),
    -- Business
    ('a0000000-0000-0000-0000-000000000003', 'ai_tokens_per_cycle',   1000000),
    ('a0000000-0000-0000-0000-000000000003', 'ai_requests_per_cycle', -1),
    ('a0000000-0000-0000-0000-000000000003', 'max_transactions',      -1),
    ('a0000000-0000-0000-0000-000000000003', 'max_categories',        -1),
    ('a0000000-0000-0000-0000-000000000003', 'max_wallets',           -1),
    ('a0000000-0000-0000-0000-000000000003', 'ocr_scans_per_cycle',   -1),
    ('a0000000-0000-0000-0000-000000000003', 'voice_requests_per_day', -1)
ON CONFLICT (plan_id, limit_key) DO UPDATE SET limit_value = EXCLUDED.limit_value;

-- ------------------------------------------------------------
-- D. PRICING RULES (default global rule)
-- ------------------------------------------------------------
INSERT INTO pricing_rules (rule_name, base_per_1k_idr, markup_multiplier, markup_percent, modality_multipliers, is_active) VALUES
    ('default', 300, 1.2500, 0,
     '{"chat":1.0,"vision":2.5,"voice":3.0,"embedding":0.5,"reasoning":2.0}'::jsonb, true)
ON CONFLICT (rule_name) DO UPDATE SET
    base_per_1k_idr = EXCLUDED.base_per_1k_idr,
    markup_multiplier = EXCLUDED.markup_multiplier,
    markup_percent = EXCLUDED.markup_percent,
    modality_multipliers = EXCLUDED.modality_multipliers;

-- ------------------------------------------------------------
-- E. AI RESERVE ESTIMATES (P3) — tokens held before an AI call
-- ------------------------------------------------------------
-- Editable from Super Admin > Pengaturan. These are HOLDS, not charges: the real
-- cost is settled at commit with the provider-reported token count. Tune them to
-- match typical usage so a commit rarely exceeds the hold (which would clamp).
INSERT INTO settings (key, value, description) VALUES
    ('ai_estimate_chat',      '2000'::jsonb, 'Token di-reserve sebelum panggilan AI chat'),
    ('ai_estimate_reasoning', '4000'::jsonb, 'Token di-reserve sebelum panggilan AI reasoning'),
    ('ai_estimate_vision',    '5000'::jsonb, 'Token di-reserve sebelum panggilan AI vision (OCR)'),
    ('ai_estimate_voice',     '2000'::jsonb, 'Token di-reserve sebelum transkripsi suara'),
    ('ai_estimate_embedding', '1000'::jsonb, 'Token di-reserve sebelum pembuatan embedding')
ON CONFLICT (key) DO NOTHING;

-- ------------------------------------------------------------
-- F. DUNNING REMINDER OFFSETS (hours before an invoice is due)
-- ------------------------------------------------------------
-- Editable from Super Admin > Pengaturan. A renewal reminder is emitted once per
-- bucket: urgent at <= dunning_hours_urgent, soon at <= dunning_hours_soon, and
-- a due reminder at H-0. Kept monotonic (urgent <= soon) by the worker.
INSERT INTO settings (key, value, description) VALUES
    ('dunning_hours_urgent', '24'::jsonb, 'Ingatkan (mendesak) saat sisa waktu invoice <= N jam'),
    ('dunning_hours_soon',   '72'::jsonb, 'Ingatkan (segera) saat sisa waktu invoice <= N jam')
ON CONFLICT (key) DO NOTHING;

-- ------------------------------------------------------------
-- G. AI GATEWAY CIRCUIT BREAKER TUNING
-- ------------------------------------------------------------
-- Editable from Super Admin > Pengaturan. Read at boot and re-applied on change.
-- The endpoint breaker guards a whole upstream; the route breaker a single
-- (endpoint, model) pair so a dead model is skipped while its siblings serve.
INSERT INTO settings (key, value, description) VALUES
    ('breaker_failure_threshold',   '5'::jsonb,  'Kegagalan beruntun sebelum circuit breaker endpoint terbuka'),
    ('breaker_cooldown_seconds',    '30'::jsonb, 'Durasi (detik) breaker endpoint terbuka sebelum uji coba'),
    ('breaker_half_open_probes',    '1'::jsonb,  'Jumlah probe paralel yang diizinkan saat half-open'),
    ('route_breaker_threshold',     '2'::jsonb,  'Kegagalan beruntun sebelum rute (endpoint+model) dikarantina'),
    ('route_breaker_cooldown_secs', '60'::jsonb, 'Durasi (detik) karantina rute sebelum dicoba lagi')
ON CONFLICT (key) DO NOTHING;

-- ------------------------------------------------------------
-- H. AI TOKEN METERING HEURISTICS ESTIMATION
-- ------------------------------------------------------------
-- Editable from Super Admin > Pengaturan. Used when an upstream provider omits
-- token usage. Read at boot and re-applied immediately upon change.
INSERT INTO settings (key, value, description) VALUES
    ('metering_chars_per_token',      '4'::jsonb,  'Rasio rata-rata karakter per token untuk teks latin/Indonesia'),
    ('metering_safety_buffer_pct',    '10'::jsonb, 'Persentase buffer pengaman overhead estimasi teks'),
    ('metering_image_tokens',         '85'::jsonb, 'Estimasi kuota token per gambar/struk (OCR low-res patch)'),
    ('metering_audio_tokens_per_sec', '25'::jsonb, 'Estimasi kuota token per detik audio transkripsi suara')
ON CONFLICT (key) DO NOTHING;
