-- ============================================================
-- 0037_ai_reasoning_controls.sql
--
-- Pengaturan dinamis kontrol mode penalaran (thinking mode) AI:
-- 1. Batas token reasoning (ai_reasoning_max_tokens) untuk mencegah
--    lonjakan kuota drastis dari Chain-of-Thought (CoT).
-- 2. Sakelar aktivasi global mode penalaran (ai_reasoning_enabled).
-- 3. Menata ulang group_key setting AI ke 'ai_estimate' agar muncul
--    secara dinamis di Tab Pengaturan Kuota & AI Super Admin.
-- ============================================================

DO $$
BEGIN
    IF to_regclass('public.setting_definitions') IS NOT NULL THEN
        -- Satukan group_key agar semua pengaturan kuota & estimasi AI muncul di Super Admin
        UPDATE setting_definitions
        SET group_key = 'ai_estimate'
        WHERE key IN (
            'ai_estimate_chat',
            'ai_estimate_reasoning',
            'ai_estimate_vision',
            'ai_estimate_voice',
            'ai_estimate_embedding',
            'ai_estimate_overhead_tokens',
            'ai_estimate_output_headroom',
            'ai_estimate_chars_per_token',
            'ai_reasoning_effort'
        );

        -- Tambah setting batas token maksimal untuk reasoning
        INSERT INTO setting_definitions
            (key, kind, min_value, max_value, default_value, group_key, label, description, sort_order)
        VALUES
            ('ai_reasoning_max_tokens', 'int', 0, 100000, '4000'::jsonb, 'ai_estimate',
             'Reasoning: Batas Max Token',
             'Batas maksimal token output + penalaran (Chain-of-Thought) per request mode penalaran. 0 = tanpa batas.', 25),
            ('ai_reasoning_enabled', 'bool', NULL, NULL, 'true'::jsonb, 'ai_estimate',
             'Reasoning: Aktifkan Mode Penalaran',
             'Sakelar global ketersediaan mode penalaran (thinking). Bila nonaktif, request diarahkan ke mode cepat.', 26)
        ON CONFLICT (key) DO UPDATE
            SET min_value = EXCLUDED.min_value,
                max_value = EXCLUDED.max_value,
                default_value = EXCLUDED.default_value,
                group_key = EXCLUDED.group_key,
                label = EXCLUDED.label,
                description = EXCLUDED.description,
                sort_order = EXCLUDED.sort_order;
    END IF;

    -- Nilai default di tabel settings (jsonb)
    INSERT INTO settings (key, value)
    VALUES
        ('ai_reasoning_effort', '"low"'::jsonb),
        ('ai_reasoning_max_tokens', '4000'::jsonb),
        ('ai_reasoning_enabled', 'true'::jsonb)
    ON CONFLICT (key) DO NOTHING;
END $$;
