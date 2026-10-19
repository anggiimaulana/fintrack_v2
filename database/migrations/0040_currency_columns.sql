-- ============================================================
-- 0040_currency_columns.sql   (EXPAND)
--
-- Preferensi mata uang pengguna + kolom mata uang pada transaksi.
--
-- Additive & idempoten. Kolom baru punya DEFAULT sehingga instance lama yang
-- tidak menulisnya tetap menghasilkan data yang benar.
--
-- amount_base adalah GENERATED STORED: DB yang menghitung, bukan aplikasi.
-- Ini (a) menjamin invariant amount_base = round(amount x exchange_rate, 2),
-- (b) membuat instance lama (yang hanya menulis amount) tetap menghasilkan
-- amount_base yang benar lewat default currency_code/IDR & exchange_rate/1.
--
-- Catatan operasional: menambah kolom GENERATED STORED menulis ulang tabel
-- (lock ACCESS EXCLUSIVE). Pada tabel sangat besar, jalankan di jendela sepi.
--
-- down:
--   ALTER TABLE transactions DROP COLUMN IF EXISTS amount_base;
--   ALTER TABLE transactions DROP COLUMN IF EXISTS rate_at;
--   ALTER TABLE transactions DROP COLUMN IF EXISTS exchange_rate;
--   ALTER TABLE transactions DROP COLUMN IF EXISTS currency_code;
--   ALTER TABLE users DROP COLUMN IF EXISTS display_currency;
-- ============================================================

-- ------------------------------------------------------------
-- users.display_currency - preferensi tampilan/pencatatan pengguna.
-- ------------------------------------------------------------
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS display_currency varchar(3)
        NOT NULL DEFAULT 'IDR' REFERENCES currencies(code) ON DELETE RESTRICT;

-- ------------------------------------------------------------
-- transactions - nominal asli + snapshot kurs + nilai dasar IDR.
-- ------------------------------------------------------------
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS currency_code varchar(3)
        NOT NULL DEFAULT 'IDR' REFERENCES currencies(code) ON DELETE RESTRICT;

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS exchange_rate numeric(20,10)
        NOT NULL DEFAULT 1 CHECK (exchange_rate > 0);

-- Nilai dasar IDR = amount x exchange_rate, dihitung oleh DB.
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS amount_base numeric(20,2)
        GENERATED ALWAYS AS (round(amount * exchange_rate, 2)) STORED;

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS rate_at timestamptz;

-- ------------------------------------------------------------
-- Index pendukung agregat berbasis amount_base (non-blocking).
-- ------------------------------------------------------------
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_user_base_date
    ON transactions (user_id, transaction_date)
    WHERE is_deleted = false;
