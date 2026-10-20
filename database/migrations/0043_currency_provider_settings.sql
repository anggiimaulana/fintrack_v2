-- ============================================================
-- 0043_currency_provider_settings.sql
--
-- Pengaturan dinamis untuk provider kurs pasar dan background sync worker.
-- Semua nilai dapat diubah langsung oleh Super Admin tanpa deploy ulang.
-- ============================================================

INSERT INTO settings (key, value, description) VALUES
    ('currency_auto_sync_enabled',    'true'::jsonb,          'Aktifkan pembaruan kurs otomatis di latar belakang (true/false)'),
    ('currency_sync_interval_hours',  '6'::jsonb,             'Frekuensi pembaruan kurs otomatis dalam jam (1-168)'),
    ('currency_rate_provider',        '"open_er_api"'::jsonb, 'Driver provider kurs aktif (open_er_api|floatrates|frankfurter|custom)'),
    ('currency_provider_base_url',    '""'::jsonb,            'URL kustom untuk provider kurs atau proxy internal'),
    ('currency_provider_api_key',     '""'::jsonb,            'Kunci API untuk provider kurs berbayar/privat'),
    ('currency_sync_timeout_seconds', '10'::jsonb,            'Batas waktu request HTTP ke provider kurs (detik)'),
    ('currency_dedup_hours',          '24'::jsonb,            'Jendela waktu deduplikasi nilai kurs identik (jam)')
ON CONFLICT (key) DO UPDATE
    SET description = EXCLUDED.description
    WHERE settings.description = '' OR settings.description IS NULL;
