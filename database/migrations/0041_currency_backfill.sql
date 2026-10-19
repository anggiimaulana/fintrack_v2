-- ============================================================
-- 0041_currency_backfill.sql   (MIGRATE)
--
-- Transaksi lama dipastikan berada pada basis IDR. Karena 0040 menambah
-- currency_code/exchange_rate dengan DEFAULT ('IDR', 1) dan amount_base adalah
-- kolom generated, baris lama SUDAH benar (amount_base = amount).
--
-- Migrasi ini HANYA memperbaiki baris IDR yang tidak konsisten. Baris non-IDR
-- TIDAK disentuh: setelah fitur dipakai, transaksi JPY/USD yang sah harus tetap
-- utuh, dan file ini dapat dijalankan ulang tanpa menghapus data pengguna.
--
-- down: tidak ada (data IDR sudah benar secara semantik).
-- ============================================================

-- Perbaiki IDR yang membawa sisa metadata kurs (exchange_rate <> 1 / rate_at).
UPDATE transactions
SET exchange_rate = 1,
    rate_at       = NULL
WHERE currency_code = 'IDR'
  AND (exchange_rate IS DISTINCT FROM 1 OR rate_at IS NOT NULL);

-- Sabuk pengaman untuk baris non-IDR yang kehilangan waktu kurs (dibuat oleh
-- versi lama). rate_at bersifat informatif; amount_base sudah tersimpan, jadi
-- menyamakannya dengan created_at aman dan memenuhi constraint di 0042.
UPDATE transactions
SET rate_at = created_at
WHERE currency_code IS DISTINCT FROM 'IDR'
  AND rate_at IS NULL;

-- Verifikasi (manual, harus 0):
--   SELECT COUNT(*) FROM transactions
--   WHERE is_deleted = false AND currency_code = 'IDR' AND amount_base <> amount;
