-- ============================================================
-- 0031_voice_vision_dynamic_estimation.sql
--
-- Dynamic Quota Estimation for Voice & Vision:
-- Configurable parameters for audio duration, tokens-per-second,
-- image base token cost, file size thresholds, and output headrooms.
-- Editable from Super Admin settings and respected by both the
-- reservation engine and local metering fallback.
-- ============================================================

BEGIN;

DO $$
BEGIN
  IF to_regclass('public.setting_definitions') IS NOT NULL THEN
    INSERT INTO setting_definitions
      (key, kind, min_value, max_value, default_value, group_key, label, description, sort_order) VALUES
      ('ai_estimate_voice_bytes_per_sec',  'int', 1000, 64000, '4000'::jsonb, 'ai_estimate', 'Voice: Byte per detik', 'Perkiraan ukuran byte audio per detik (WebM Opus ~4000 byte/detik, MP3 ~16000 byte/detik).', 30),
      ('ai_estimate_voice_tokens_per_sec', 'int', 1, 200,     '25'::jsonb,   'ai_estimate', 'Voice: Token per detik', 'Rasio token provider yang dikonsumsi per detik audio.', 31),
      ('ai_estimate_voice_overhead',       'int', 0, 5000,    '50'::jsonb,   'ai_estimate', 'Voice: Overhead (token)', 'Overhead token tetap untuk pemrosesan transkripsi suara.', 32),
      ('ai_estimate_voice_output_headroom','int', 0, 5000,    '150'::jsonb,  'ai_estimate', 'Voice: Headroom output (token)', 'Perkiraan jumlah token hasil transkripsi teks audio.', 33),
      ('ai_estimate_vision_image_tokens',  'int', 50, 10000,  '800'::jsonb,  'ai_estimate', 'Vision: Token per gambar', 'Estimasi token provider untuk satu gambar struk.', 40),
      ('ai_estimate_vision_output_headroom','int', 50, 5000,   '400'::jsonb,  'ai_estimate', 'Vision: Headroom output (token)', 'Perkiraan jumlah token JSON hasil ekstraksi struk.', 41),
      ('ai_estimate_vision_large_file_bytes','int', 100000, 5000000, '500000'::jsonb, 'ai_estimate', 'Vision: Ambang file besar (byte)', 'Batas ukuran file gambar di mana token tambahan dialokasikan.', 42),
      ('ai_estimate_vision_large_file_extra','int', 0, 5000,  '400'::jsonb,  'ai_estimate', 'Vision: Token ekstra file besar', 'Token tambahan untuk gambar berukuran di atas ambang batas.', 43)
    ON CONFLICT (key) DO NOTHING;
  END IF;
END $$;

-- Seed default values into settings table
INSERT INTO settings (key, value) VALUES
    ('ai_estimate_voice_bytes_per_sec', '4000'::jsonb),
    ('ai_estimate_voice_tokens_per_sec', '25'::jsonb),
    ('ai_estimate_voice_overhead', '50'::jsonb),
    ('ai_estimate_voice_output_headroom', '150'::jsonb),
    ('ai_estimate_vision_image_tokens', '800'::jsonb),
    ('ai_estimate_vision_output_headroom', '400'::jsonb),
    ('ai_estimate_vision_large_file_bytes', '500000'::jsonb),
    ('ai_estimate_vision_large_file_extra', '400'::jsonb)
ON CONFLICT (key) DO NOTHING;

-- Lower the flat "lantai" (floor) values for voice/vision so the DYNAMIC
-- calculation actually dominates. The old seeds (voice=2000, vision=5000) were
-- sized for a flat reserve; with a dynamic estimate they would act as a hard
-- floor and silently defeat the feature (every call would reserve the flat
-- value again). They remain as a small safety minimum an operator can raise.
UPDATE settings SET value = '200'::jsonb WHERE key = 'ai_estimate_voice' AND value = '2000'::jsonb;
UPDATE settings SET value = '500'::jsonb WHERE key = 'ai_estimate_vision' AND value = '5000'::jsonb;

COMMIT;
