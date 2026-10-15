-- ============================================================
-- 0029_config_catalog.sql
--
-- Katalog konfigurasi dinamis: memindahkan kebijakan dari KODE ke DATA.
--
-- Sebelumnya tiga hal di-hardcode di Go:
--   1. `writableSettings` (admin/settings.go)  → allowlist + validator setting
--   2. `UsageFor()` switch (middleware/usage_resolvers.go) → limit_key → resolver
--   3. `FeatureCatalog` / `LimitCatalog` (entitlement/catalog.go)
--
-- Dengan tabel-tabel ini, operator dapat menambah/mengubah setting, metrik, dan
-- fitur dari Super Admin TANPA deploy. Kode tetap menyediakan KEMAMPUAN
-- (resolver); database menyimpan KEBIJAKAN.
--
-- Additive & idempotent: tabel baru + seed dengan ON CONFLICT DO NOTHING/UPDATE
-- terbatas, sehingga tidak menimpa penyesuaian operator.
--
-- down:
--   DROP TABLE IF EXISTS setting_definitions;
--   DROP TABLE IF EXISTS metric_definitions;
--   DROP TABLE IF EXISTS feature_definitions;
-- ============================================================

-- ------------------------------------------------------------
-- setting_definitions — katalog setting yang dapat diubah operator.
-- Menggantikan allowlist hardcoded `writableSettings`.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS setting_definitions (
    key           varchar(64) PRIMARY KEY,
    -- kind menentukan validasi & editor yang dipakai UI:
    --   string | int | bool | enum | secret
    kind          varchar(16) NOT NULL,
    min_value     bigint,
    max_value     bigint,
    enum_values   jsonb,
    max_len       integer NOT NULL DEFAULT 0,
    default_value jsonb,
    group_key     varchar(32) NOT NULL DEFAULT 'umum',
    label         varchar(128) NOT NULL DEFAULT '',
    description   text NOT NULL DEFAULT '',
    is_writable   boolean NOT NULL DEFAULT true,
    is_secret     boolean NOT NULL DEFAULT false,
    sort_order    integer NOT NULL DEFAULT 0,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT setting_definitions_kind_check
        CHECK (kind IN ('string','int','bool','enum','secret'))
);

CREATE OR REPLACE TRIGGER on_setting_definition_updated
    BEFORE UPDATE ON setting_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_setting_definitions_group
    ON setting_definitions (group_key, sort_order);

-- ------------------------------------------------------------
-- metric_definitions — katalog limit (metrik) yang dapat ditegakkan.
-- Menggantikan LimitCatalog + switch UsageFor().
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS metric_definitions (
    key            varchar(64) PRIMARY KEY,
    label          varchar(128) NOT NULL,
    unit           varchar(32) NOT NULL DEFAULT '',
    description    text NOT NULL DEFAULT '',
    -- Apakah backend menghitung usage untuk metrik ini?
    is_enforced    boolean NOT NULL DEFAULT false,
    -- Resolver mana yang dipakai (harus salah satu yang tersedia di kode).
    resolver_key   varchar(64),
    -- Apakah nilainya dipengaruhi rasio token (catatan di UI)?
    ratio_adjusted boolean NOT NULL DEFAULT false,
    per_cycle      boolean NOT NULL DEFAULT true,
    group_key      varchar(32) NOT NULL DEFAULT 'kuota',
    sort_order     integer NOT NULL DEFAULT 0,
    is_active      boolean NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    -- Metrik enforced WAJIB punya resolver; non-enforced boleh tanpa.
    CONSTRAINT metric_definitions_enforced_needs_resolver
        CHECK (NOT is_enforced OR resolver_key IS NOT NULL)
);

CREATE OR REPLACE TRIGGER on_metric_definition_updated
    BEFORE UPDATE ON metric_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_metric_definitions_group
    ON metric_definitions (group_key, sort_order);

-- ------------------------------------------------------------
-- feature_definitions — katalog fitur boolean.
-- Menggantikan FeatureCatalog.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS feature_definitions (
    key         varchar(64) PRIMARY KEY,
    label       varchar(128) NOT NULL,
    description text NOT NULL DEFAULT '',
    group_key   varchar(32) NOT NULL DEFAULT 'fitur',
    sort_order  integer NOT NULL DEFAULT 0,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_feature_definition_updated
    BEFORE UPDATE ON feature_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ============================================================
-- SEED — memindahkan konstanta kode saat ini ke data.
-- Nilai SAMA persis, sehingga perilaku tidak berubah.
-- `WHERE label = ''` mencegah menimpa penyesuaian operator.
-- ============================================================

-- ---- Setting: kuota (BARU, dipakai fitur rasio) ----
INSERT INTO setting_definitions
    (key, kind, min_value, max_value, enum_values, max_len, default_value,
     group_key, label, description, sort_order) VALUES
    ('quota_rounding_mode', 'enum', NULL, NULL, '["ceil","floor","round"]'::jsonb, 0, '"ceil"'::jsonb,
     'kuota', 'Mode pembulatan kuota',
     'Cara membulatkan unit kuota: ceil (ke atas), floor (ke bawah), round (terdekat).', 10),
    ('quota_default_ratio', 'int', 1, 1000, NULL, 0, '1'::jsonb,
     'kuota', 'Rasio kuota default',
     'Rasio awal untuk plan baru yang dibuat operator.', 20),
    ('quota_default_tokens_per_cycle', 'int', 1, 1000000000, NULL, 0, '10000'::jsonb,
     'kuota', 'Kuota default / siklus',
     'Kuota awal untuk plan baru (dalam unit kuota).', 30),
    ('quota_unit_label', 'string', NULL, NULL, NULL, 24, '"kuota"'::jsonb,
     'kuota', 'Label satuan kuota',
     'Satuan yang ditampilkan ke user (mis. kuota).', 40),
    ('quota_unlimited_sentinel', 'int', 1, 1000000000000, NULL, 0, '1000000000'::jsonb,
     'kuota', 'Nilai pengganti unlimited',
     'Angka yang dipakai saat limit plan = -1 (unlimited).', 50),
    ('quota_min_charge_units', 'int', 0, 1000000, NULL, 0, '1'::jsonb,
     'kuota', 'Lantai tagih minimum',
     'Minimum unit yang ditagih per permintaan (menutup celah mode floor).', 60)
ON CONFLICT (key) DO NOTHING;

-- ---- Setting: estimasi AI (sebelumnya konstanta di billing/estimate.go) ----
INSERT INTO setting_definitions
    (key, kind, min_value, max_value, default_value, group_key, label, description, sort_order) VALUES
    ('ai_estimate_overhead_tokens', 'int', 0, 1000000, '3000'::jsonb,
     'ai', 'Overhead estimasi (token)',
     'Token tetap per chat: system prompt + tools + RAG + riwayat.', 10),
    ('ai_estimate_output_headroom', 'int', 0, 1000000, '800'::jsonb,
     'ai', 'Headroom keluaran (token)',
     'Perkiraan token keluaran per chat.', 20),
    ('ai_estimate_chars_per_token', 'int', 1, 100, '4'::jsonb,
     'ai', 'Karakter per token',
     'Perkiraan rasio karakter→token untuk estimasi panjang pesan.', 30)
ON CONFLICT (key) DO NOTHING;

-- ---- Setting: harga (default biaya provider, P-11) ----
INSERT INTO setting_definitions
    (key, kind, min_value, max_value, default_value, group_key, label, description, sort_order) VALUES
    ('pricing_default_provider_cost_per_1k_idr', 'int', 0, 100000000, '150'::jsonb,
     'pricing', 'Biaya provider default / 1k token',
     'Dipakai saat membuat aturan harga baru, agar margin langsung terukur.', 10)
ON CONFLICT (key) DO NOTHING;

-- ---- Setting: pindahkan allowlist lama dari kode ----
INSERT INTO setting_definitions
    (key, kind, min_value, max_value, enum_values, max_len, group_key, label, sort_order) VALUES
    ('payment_gateway_default', 'enum', NULL, NULL, '["manual","midtrans"]'::jsonb, 0, 'pembayaran', 'Gateway pembayaran default', 10),
    ('grace_period_hours', 'int', 0, 720, NULL, 0, 'pembayaran', 'Masa tenggang (jam)', 20),
    ('tax_rate_percent', 'int', 0, 100, NULL, 0, 'pajak', 'Persentase pajak', 10),
    ('tax_enabled', 'bool', NULL, NULL, NULL, 0, 'pajak', 'Aktifkan pajak', 20),
    ('tax_label', 'string', NULL, NULL, NULL, 32, 'pajak', 'Label pajak', 30),
    ('invoice_due_hours', 'int', 1, 720, NULL, 0, 'invoice', 'Jatuh tempo invoice (jam)', 10),
    ('invoice_expiry_hours', 'int', 0, 8760, NULL, 0, 'invoice', 'Kedaluwarsa invoice (jam)', 20),
    ('invoice_expiry_minutes', 'int', 0, 59, NULL, 0, 'invoice', 'Kedaluwarsa invoice (menit)', 30),
    ('storage_driver', 'enum', NULL, NULL, '["local","r2"]'::jsonb, 0, 'penyimpanan', 'Driver penyimpanan', 10),
    ('ai_estimate_chat', 'int', 1, 1000000, NULL, 0, 'ai', 'Estimasi chat (lantai)', 40),
    ('ai_estimate_reasoning', 'int', 1, 1000000, NULL, 0, 'ai', 'Estimasi reasoning (lantai)', 50),
    ('ai_estimate_vision', 'int', 1, 1000000, NULL, 0, 'ai', 'Estimasi vision (lantai)', 60),
    ('ai_estimate_voice', 'int', 1, 1000000, NULL, 0, 'ai', 'Estimasi voice (lantai)', 70),
    ('ai_estimate_embedding', 'int', 1, 1000000, NULL, 0, 'ai', 'Estimasi embedding (lantai)', 80),
    ('ai_reasoning_effort', 'enum', NULL, NULL, '["","low","medium","high"]'::jsonb, 0, 'ai', 'Reasoning effort', 90),
    ('dunning_hours_urgent', 'int', 1, 720, NULL, 0, 'dunning', 'Pengingat mendesak (jam)', 10),
    ('dunning_hours_soon', 'int', 1, 720, NULL, 0, 'dunning', 'Pengingat segera (jam)', 20),
    ('breaker_failure_threshold', 'int', 1, 100, NULL, 0, 'gateway', 'Ambang kegagalan breaker', 10),
    ('breaker_cooldown_seconds', 'int', 1, 3600, NULL, 0, 'gateway', 'Cooldown breaker (detik)', 20),
    ('breaker_half_open_probes', 'int', 1, 100, NULL, 0, 'gateway', 'Probe half-open', 30),
    ('route_breaker_threshold', 'int', 1, 100, NULL, 0, 'gateway', 'Ambang breaker rute', 40),
    ('route_breaker_cooldown_secs', 'int', 1, 3600, NULL, 0, 'gateway', 'Cooldown breaker rute (detik)', 50),
    ('metering_chars_per_token', 'int', 1, 100, NULL, 0, 'metering', 'Karakter per token (metering)', 10),
    ('metering_safety_buffer_pct', 'int', 0, 100, NULL, 0, 'metering', 'Buffer keamanan (%)', 20),
    ('metering_image_tokens', 'int', 1, 10000, NULL, 0, 'metering', 'Token per gambar', 30),
    ('metering_audio_tokens_per_sec', 'int', 1, 1000, NULL, 0, 'metering', 'Token audio per detik', 40),
    ('mail_enabled', 'bool', NULL, NULL, NULL, 0, 'mail', 'Aktifkan email', 10),
    ('mail_host', 'string', NULL, NULL, NULL, 0, 'mail', 'SMTP host', 20),
    ('mail_port', 'int', 1, 65535, NULL, 0, 'mail', 'SMTP port', 30),
    ('mail_username', 'string', NULL, NULL, NULL, 0, 'mail', 'SMTP username', 40),
    ('mail_from', 'string', NULL, NULL, NULL, 0, 'mail', 'Alamat pengirim', 50),
    ('mail_from_name', 'string', NULL, NULL, NULL, 0, 'mail', 'Nama pengirim', 60),
    ('mail_use_tls', 'bool', NULL, NULL, NULL, 0, 'mail', 'Gunakan TLS', 70),
    ('midtrans_enabled', 'bool', NULL, NULL, NULL, 0, 'midtrans', 'Aktifkan Midtrans', 10),
    ('midtrans_client_key', 'string', NULL, NULL, NULL, 0, 'midtrans', 'Midtrans client key', 20),
    ('midtrans_is_production', 'bool', NULL, NULL, NULL, 0, 'midtrans', 'Midtrans produksi', 30),
    ('manual_bank_name', 'string', NULL, NULL, NULL, 0, 'manual', 'Nama bank', 10),
    ('manual_account_number', 'string', NULL, NULL, NULL, 0, 'manual', 'Nomor rekening', 20),
    ('manual_account_holder', 'string', NULL, NULL, NULL, 0, 'manual', 'Nama pemilik rekening', 30),
    ('r2_account_id', 'string', NULL, NULL, NULL, 0, 'r2', 'R2 account id', 10),
    ('r2_access_key_id', 'string', NULL, NULL, NULL, 0, 'r2', 'R2 access key id', 20),
    ('r2_bucket', 'string', NULL, NULL, NULL, 0, 'r2', 'R2 bucket', 30),
    ('r2_region', 'string', NULL, NULL, NULL, 0, 'r2', 'R2 region', 40),
    ('r2_endpoint', 'string', NULL, NULL, NULL, 0, 'r2', 'R2 endpoint', 50),
    ('r2_prefix', 'string', NULL, NULL, NULL, 0, 'r2', 'R2 prefix', 60),
    ('r2_public_base_url', 'string', NULL, NULL, NULL, 0, 'r2', 'R2 public base url', 70)
ON CONFLICT (key) DO NOTHING;

-- ---- Setting: rahasia (kind=secret, tidak pernah dikirim ke klien) ----
INSERT INTO setting_definitions
    (key, kind, is_secret, group_key, label, sort_order) VALUES
    ('mail_password', 'secret', true, 'mail', 'SMTP password', 45),
    ('midtrans_server_key', 'secret', true, 'midtrans', 'Midtrans server key', 25),
    ('r2_secret_access_key', 'secret', true, 'r2', 'R2 secret access key', 25),
    ('gemini_embedding_api_key', 'secret', true, 'ai', 'Gemini embedding API key', 100)
ON CONFLICT (key) DO NOTHING;

-- ---- Metrik (menggantikan LimitCatalog) ----
INSERT INTO metric_definitions
    (key, label, unit, description, is_enforced, resolver_key, ratio_adjusted, sort_order) VALUES
    ('ai_tokens_per_cycle', 'Kuota AI / siklus', 'kuota',
     'Kuota AI per siklus, diukur dalam unit kuota (bukan token mentah).',
     true, 'token_period_usage', true, 10),
    ('ai_requests_per_cycle', 'Permintaan AI / siklus', 'permintaan',
     'Jumlah panggilan AI (chat, OCR, suara) per siklus.',
     true, 'ai_requests_in_period', false, 20),
    ('max_transactions', 'Transaksi', 'transaksi',
     'Transaksi aktif yang dapat disimpan.', true, 'active_transactions', false, 30),
    ('ocr_scans_per_cycle', 'Pindai struk / siklus', 'scan',
     'Pemindaian struk (OCR) per siklus.', true, 'vision_scans_in_period', false, 40),
    ('voice_requests_per_day', 'Input suara / hari', 'permintaan',
     'Permintaan suara per hari.', true, 'voice_requests_per_day', false, 45),
    ('max_categories', 'Kategori', 'kategori',
     'Jumlah kategori maksimum (belum ditegakkan).', false, NULL, false, 50),
    ('max_wallets', 'Dompet', 'dompet',
     'Jumlah dompet maksimum (belum ditegakkan).', false, NULL, false, 60)
ON CONFLICT (key) DO NOTHING;

-- ---- Fitur (menggantikan FeatureCatalog) ----
INSERT INTO feature_definitions (key, label, description, sort_order) VALUES
    ('ai_chat', 'Asisten AI (chat)', 'Chat keuangan dengan AI.', 10),
    ('ocr_receipt', 'Pindai struk (OCR)', 'Ekstraksi data dari foto struk.', 20),
    ('voice_input', 'Input suara', 'Transkripsi perintah suara.', 30),
    ('rag_search', 'Pencarian cerdas (RAG)', 'Pencarian semantik atas transaksi.', 40),
    ('reports_export', 'Ekspor laporan', 'Unduh laporan keuangan.', 50),
    ('advanced_budgeting', 'Anggaran lanjutan', 'Anggaran & analisis lanjutan.', 60),
    ('api_access', 'Akses API', 'Akses API publik.', 70),
    ('bank_sync', 'Sinkronisasi bank', 'Sinkronisasi rekening bank (belum tersedia).', 80)
ON CONFLICT (key) DO NOTHING;
