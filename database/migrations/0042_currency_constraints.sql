-- ============================================================
-- 0042_currency_constraints.sql   (CONTRACT lunak)
--
-- Aturan integritas mata uang yang tidak dapat dijamin kolom generated:
-- transaksi non-base WAJIB menyimpan waktu kurs yang dipakai.
--
-- Idempoten: constraint dilepas lebih dulu bila sudah ada, sehingga file ini
-- dapat dijalankan ulang (migrate.sh memutar ulang seluruh file tanpa ledger).
--
-- amount_base TIDAK perlu CHECK terpisah: ia kolom GENERATED, sehingga
-- invariant amount_base = round(amount x exchange_rate, 2) selalu benar.
--
-- down:
--   ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_rate_at;
-- ============================================================

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS chk_transactions_rate_at;

ALTER TABLE transactions
    ADD CONSTRAINT chk_transactions_rate_at
    CHECK (currency_code = 'IDR' OR rate_at IS NOT NULL) NOT VALID;

ALTER TABLE transactions
    VALIDATE CONSTRAINT chk_transactions_rate_at;
