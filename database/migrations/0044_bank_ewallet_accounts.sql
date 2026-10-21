-- ============================================================
-- 0044_bank_ewallet_accounts.sql
--
-- Fondasi Akun Finansial (Bank & E-Wallet), Ingestion Batch,
-- Transaksi Transfer, Fingerprint Deduplikasi, & Entitlement Paket.
--
-- Idempoten & additive.
-- ============================================================

-- ------------------------------------------------------------
-- 1. Perluas tipe transaksi dengan 'transfer'
-- ------------------------------------------------------------
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_type_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('income', 'expense', 'savings', 'transfer'));

-- ------------------------------------------------------------
-- 2. Tabel financial_accounts (Rekening Bank & E-Wallet Pengguna)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS financial_accounts (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                varchar(100) NOT NULL,
    account_type        varchar(32) NOT NULL CHECK (account_type IN ('bank', 'ewallet', 'cash', 'investment')),
    institution_code    varchar(50) NOT NULL, -- 'bca', 'mandiri', 'jago', 'seabank', 'gopay', 'dana', 'shopeepay', 'custom'
    account_number_mask varchar(30),          -- Format masking: '****1234' (UU PDP: tidak boleh simpan nomor rekening utuh)
    currency_code       varchar(3) NOT NULL DEFAULT 'IDR' REFERENCES currencies(code) ON DELETE RESTRICT,
    current_balance     numeric(18, 2) NOT NULL DEFAULT 0.00,
    initial_balance     numeric(18, 2) NOT NULL DEFAULT 0.00,
    color_hex           varchar(7) NOT NULL DEFAULT '#2563EB',
    icon_key            varchar(50) NOT NULL DEFAULT 'wallet',
    is_active           boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_financial_account_name UNIQUE (user_id, name)
);

CREATE OR REPLACE TRIGGER on_financial_accounts_updated
    BEFORE UPDATE ON financial_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_fin_accounts_user_active
    ON financial_accounts(user_id)
    WHERE is_active = true;

-- ------------------------------------------------------------
-- 3. Tabel import_batches (Audit Impor Berkas Mutasi)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS import_batches (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id          uuid NOT NULL REFERENCES financial_accounts(id) ON DELETE CASCADE,
    file_name           varchar(255) NOT NULL,
    file_size_bytes     bigint NOT NULL DEFAULT 0,
    file_hash_sha256    char(64) NOT NULL,
    total_rows          int NOT NULL DEFAULT 0,
    imported_rows       int NOT NULL DEFAULT 0,
    skipped_duplicates  int NOT NULL DEFAULT 0,
    status              varchar(32) NOT NULL DEFAULT 'completed'
                        CHECK (status IN ('processing', 'completed', 'failed', 'rolled_back')),
    error_message       text,
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_import_batches_user_account
    ON import_batches(user_id, account_id, created_at DESC);

-- ------------------------------------------------------------
-- 4. Kolom baru pada tabel transactions
-- ------------------------------------------------------------
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS account_id uuid REFERENCES financial_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS import_batch_id uuid REFERENCES import_batches(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS fingerprint_hash char(64),
    ADD COLUMN IF NOT EXISTS clean_merchant varchar(150),
    ADD COLUMN IF NOT EXISTS transfer_pair_id uuid REFERENCES transactions(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_tx_fingerprint
    ON transactions(user_id, fingerprint_hash)
    WHERE is_deleted = false AND fingerprint_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transactions_account_date
    ON transactions(account_id, transaction_date DESC)
    WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_transactions_transfer_pair
    ON transactions(transfer_pair_id)
    WHERE transfer_pair_id IS NOT NULL;

-- ------------------------------------------------------------
-- 5. Katalog Konfigurasi Dinamis (Feature & Metric Definitions)
-- ------------------------------------------------------------
INSERT INTO feature_definitions (key, label, description, group_key, sort_order) VALUES
    ('bank_integration', 'Integrasi Bank & E-Wallet', 'Pencatatan otomatis mutasi bank digital & dompet digital.', 'fitur', 85)
ON CONFLICT (key) DO NOTHING;

INSERT INTO metric_definitions (key, label, unit, description, is_enforced, resolver_key, ratio_adjusted, sort_order) VALUES
    ('max_financial_accounts', 'Akun Bank & E-Wallet', 'akun', 'Jumlah maksimal rekening/dompet aktif yang dapat dikelola.', false, NULL, false, 70),
    ('monthly_statement_imports', 'Impor Mutasi / Bulan', 'impor', 'Jumlah maksimal berkas mutasi yang dapat diimpor per bulan.', false, NULL, false, 80)
ON CONFLICT (key) DO NOTHING;

-- ------------------------------------------------------------
-- 6. Seed Entitlement & Limit per Paket
--    (Awal: Hanya paket Business yang aktif)
-- ------------------------------------------------------------
INSERT INTO plan_entitlements (plan_id, feature_key, is_enabled) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'bank_integration', false),
    ('a0000000-0000-0000-0000-000000000002', 'bank_integration', false),
    ('a0000000-0000-0000-0000-000000000003', 'bank_integration', true)
ON CONFLICT (plan_id, feature_key) DO UPDATE SET is_enabled = EXCLUDED.is_enabled;

INSERT INTO plan_limits (plan_id, limit_key, limit_value) VALUES
    -- Free: 0 akun, 0 import
    ('a0000000-0000-0000-0000-000000000001', 'max_financial_accounts', 0),
    ('a0000000-0000-0000-0000-000000000001', 'monthly_statement_imports', 0),
    -- Pro: 0 akun, 0 import
    ('a0000000-0000-0000-0000-000000000002', 'max_financial_accounts', 0),
    ('a0000000-0000-0000-0000-000000000002', 'monthly_statement_imports', 0),
    -- Business: Unlimited akun (-1), 100 imports/month
    ('a0000000-0000-0000-0000-000000000003', 'max_financial_accounts', -1),
    ('a0000000-0000-0000-0000-000000000003', 'monthly_statement_imports', 100)
ON CONFLICT (plan_id, limit_key) DO UPDATE SET limit_value = EXCLUDED.limit_value;
