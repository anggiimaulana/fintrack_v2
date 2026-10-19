-- ============================================================
-- 0039_currency_catalog.sql
--
-- Katalog mata uang + riwayat kurs dinamis (menggantikan hardcode IDR).
--
-- Kebijakan (mata uang mana, kurs berapa) ada di DATA; kode hanya menyediakan
-- KEMAMPUAN konversi. IDR tetap base accounting currency pada rilis ini.
--
-- Additive & idempotent.
--
-- down:
--   DROP TABLE IF EXISTS currency_rates;
--   DROP TABLE IF EXISTS currencies;
-- ============================================================

-- ------------------------------------------------------------
-- currencies - metadata format & kebijakan per mata uang.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS currencies (
    code            varchar(3)   PRIMARY KEY,
    name            varchar(64)  NOT NULL,
    symbol          varchar(8)   NOT NULL,
    locale          varchar(16)  NOT NULL,
    decimal_digits  smallint     NOT NULL DEFAULT 0 CHECK (decimal_digits BETWEEN 0 AND 4),
    is_active       boolean      NOT NULL DEFAULT false,
    is_base         boolean      NOT NULL DEFAULT false,
    sort_order      integer      NOT NULL DEFAULT 0,
    -- Sumber kurs: 'manual' sekarang; 'provider' disiapkan untuk auto-fetch nanti.
    rate_source     varchar(16)  NOT NULL DEFAULT 'manual'
        CHECK (rate_source IN ('manual','provider')),
    created_at      timestamptz  NOT NULL DEFAULT now(),
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    -- Kode ISO 4217 huruf besar 3 karakter.
    CONSTRAINT currencies_code_format CHECK (code ~ '^[A-Z]{3}$')
);

CREATE OR REPLACE TRIGGER on_currency_updated
    BEFORE UPDATE ON currencies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Hanya satu base currency.
CREATE UNIQUE INDEX IF NOT EXISTS uq_currencies_base
    ON currencies ((is_base)) WHERE is_base;

CREATE INDEX IF NOT EXISTS idx_currencies_active
    ON currencies (is_active, sort_order);

-- ------------------------------------------------------------
-- currency_rates - riwayat kurs APPEND-ONLY.
-- 1 unit currency_code = rate_to_base unit base (IDR).
-- Koreksi dilakukan dengan menambah baris baru, bukan mengubah yang lama.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS currency_rates (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    currency_code  varchar(3)    NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    rate_to_base   numeric(20,10) NOT NULL CHECK (rate_to_base > 0),
    effective_at   timestamptz   NOT NULL DEFAULT now(),
    source         varchar(16)   NOT NULL DEFAULT 'manual'
        CHECK (source IN ('manual','provider')),
    note           text,
    created_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz   NOT NULL DEFAULT now(),
    UNIQUE (currency_code, effective_at)
);

-- Query utama: kurs terbaru yang effective_at <= T, per mata uang.
CREATE INDEX IF NOT EXISTS idx_currency_rates_lookup
    ON currency_rates (currency_code, effective_at DESC);

-- ------------------------------------------------------------
-- SEED - IDR base aktif; JPY nonaktif sampai Super Admin mengisi kurs.
-- Tidak mengarang kurs produksi. `ON CONFLICT DO NOTHING` tidak menimpa
-- penyesuaian operator.
-- ------------------------------------------------------------
INSERT INTO currencies
    (code, name, symbol, locale, decimal_digits, is_active, is_base, sort_order, rate_source)
VALUES
    ('IDR', 'Rupiah Indonesia', 'Rp', 'id-ID', 0, true,  true,  10, 'manual'),
    ('JPY', 'Yen Jepang',       '¥',  'ja-JP', 0, false, false, 20, 'manual')
ON CONFLICT (code) DO NOTHING;

-- Kurs base sendiri = 1 (jejak eksplisit).
INSERT INTO currency_rates (currency_code, rate_to_base, effective_at, source, note)
VALUES ('IDR', 1, '1970-01-01T00:00:00Z', 'manual', 'Base currency')
ON CONFLICT (currency_code, effective_at) DO NOTHING;
