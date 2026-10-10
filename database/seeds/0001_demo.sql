-- ============================================================
-- seeds/0001_demo.sql  (SEED — not a migration)
-- Demo data for local development. Idempotent.
--
-- Derived from v1 migrations 010, 012, 013, 014, 016 — with:
--   * no `auth.users` block (users are created directly)
--   * bcrypt hashes written as literals (cost 12)
--   * no `ai_conversations` block (dead code)
--   * `refresh_tokens` left empty (everyone logs in fresh)
--
-- Credentials:
--   admin@fintrack.id / Admin@123456
--   budi|siti|andi@example.com / User@123456
-- ============================================================

-- ============================================================
-- A. USERS
-- ============================================================
INSERT INTO users (id, email, password_hash, full_name, role, is_active, created_at) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@fintrack.id',
     '$2a$12$jSYcrnfj6klIT1PoRftFVOAhuClK.47d2jD0RxXcmpTnUrsCG0mb6',
     'Admin FinTrack', 'admin', true, now()),
    ('00000000-0000-0000-0000-000000000002', 'budi@example.com',
     '$2a$12$qLsyCGrn7EJLBoyDcMQ1EutqnK7WZUELdj62e4d39m266MZzD1CyC',
     'Budi Santoso', 'user', true, now()),
    ('00000000-0000-0000-0000-000000000003', 'siti@example.com',
     '$2a$12$qLsyCGrn7EJLBoyDcMQ1EutqnK7WZUELdj62e4d39m266MZzD1CyC',
     'Siti Rahayu', 'user', true, now()),
    ('00000000-0000-0000-0000-000000000004', 'andi@example.com',
     '$2a$12$qLsyCGrn7EJLBoyDcMQ1EutqnK7WZUELdj62e4d39m266MZzD1CyC',
     'Andi Wijaya', 'user', true, now())
ON CONFLICT (email) DO NOTHING;

-- ============================================================
-- B. INCOME CATEGORIES (from v1 013)
-- ============================================================
INSERT INTO income_categories (id, name, icon, color, description, sort_order, created_by) VALUES
    ('c0010001-0000-0000-0000-000000000001', 'Gaji Tetap',       'briefcase',     '#3b5bdb', 'Penghasilan tetap dari pekerjaan utama',         1,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000002', 'Freelance/Proyek', 'laptop',        '#10b981', 'Penghasilan dari pekerjaan lepas atau proyek',    2,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000003', 'Investasi',        'trending-up',   '#f59e0b', 'Hasil investasi seperti saham, reksadana',        3,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000004', 'Bonus',            'gift',          '#f43f5e', 'Bonus tahunan, THR, atau insentif',               4,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000005', 'Bisnis',           'store',         '#06b6d4', 'Penghasilan dari usaha atau bisnis',              5,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000006', 'Hadiah/Hibah',     'heart',         '#ec4899', 'Hadiah, hibah, atau pemberian',                   6,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000007', 'Penjualan Aset',   'package',       '#8b5cf6', 'Hasil penjualan barang atau aset',                7,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000008', 'Uang Saku',        'wallet',        '#14b8a6', 'Uang saku atau kiriman keluarga',                 8,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-000000000009', 'Komisi',           'percent',       '#84cc16', 'Komisi penjualan atau afiliasi',                  9,  '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-00000000000a', 'Pendapatan Pasif', 'sleep',         '#6366f1', 'Royalti, afiliasi, atau pendapatan pasif lainnya', 10, '00000000-0000-0000-0000-000000000001'),
    ('c0010001-0000-0000-0000-00000000000b', 'Lainnya',          'ellipsis',      '#94a3b8', 'Pemasukan lain yang tidak terkategorikan',        11, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- C. EXPENSE CATEGORIES (from v1 013)
-- ============================================================
INSERT INTO expense_categories (id, name, icon, color, description, sort_order, created_by) VALUES
    ('c0020001-0000-0000-0000-000000000001', 'Makanan & Minuman',    'utensils-crossed', '#f43f5e', 'Makanan sehari-hari, jajan, minuman',          1,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000002', 'Transportasi',         'car',              '#f59e0b', 'Bensin, transportasi umum, ojek online',       2,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000003', 'Kesehatan',            'heart-pulse',      '#10b981', 'Biaya dokter, obat, rumah sakit',              3,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000004', 'Pendidikan',           'book-open',        '#3b5bdb', 'Kursus, buku, pelatihan, SPP',                 4,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000005', 'Hiburan & Rekreasi',   'popcorn',          '#06b6d4', 'Nonton, liburan, hobi',                        5,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000006', 'Utilitas & Tagihan',   'zap',              '#f59e0b', 'Listrik, air, internet, pulsa',                6,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000007', 'Belanja & Fashion',    'shopping-bag',     '#ec4899', 'Pakaian, aksesoris, belanja online',           7,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000008', 'Tempat Tinggal',       'home',             '#3b5bdb', 'Kontrakan, KPR, perawatan rumah',              8,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-000000000009', 'Elektronik & Gadget',  'monitor',          '#8b5cf6', 'HP, laptop, aksesoris elektronik',             9,  '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-00000000000a', 'Perawatan Diri',       'sparkles',         '#14b8a6', 'Skincare, salon, potong rambut',               10, '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-00000000000b', 'Sosial & Donasi',      'users',            '#84cc16', 'Donasi, amplop, arisan',                       11, '00000000-0000-0000-0000-000000000001'),
    ('c0020001-0000-0000-0000-00000000000c', 'Lain-lain',            'ellipsis',         '#94a3b8', 'Pengeluaran lain yang tidak terkategorikan',   12, '00000000-0000-0000-0000-000000000001')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- D. EXPENSE SUBCATEGORIES (from v1 013)
-- ============================================================
INSERT INTO expense_subcategories (category_id, name, icon, color, sort_order)
SELECT * FROM (VALUES
    ('c0020001-0000-0000-0000-000000000001'::uuid, 'Makan Siang',       'utensils-crossed',  '#f43f5e', 1),
    ('c0020001-0000-0000-0000-000000000001'::uuid, 'Sarapan',           'coffee',            '#f43f5e', 2),
    ('c0020001-0000-0000-0000-000000000001'::uuid, 'Jajan Ringan',      'cookie',            '#f43f5e', 3),
    ('c0020001-0000-0000-0000-000000000001'::uuid, 'Minuman',           'wine',              '#f43f5e', 4),
    ('c0020001-0000-0000-0000-000000000001'::uuid, 'Grosir Bulanan',    'shopping-cart',     '#f43f5e', 5),
    ('c0020001-0000-0000-0000-000000000002'::uuid, 'Bensin',            'fuel',              '#f59e0b', 1),
    ('c0020001-0000-0000-0000-000000000002'::uuid, 'Ojek Online',       'motorcycle',        '#f59e0b', 2),
    ('c0020001-0000-0000-0000-000000000002'::uuid, 'Transportasi Umum', 'bus',               '#f59e0b', 3),
    ('c0020001-0000-0000-0000-000000000002'::uuid, 'Parkir & Tol',      'circle-parking',    '#f59e0b', 4),
    ('c0020001-0000-0000-0000-000000000002'::uuid, 'Bensin Motor',      'fuel',              '#f59e0b', 5),
    ('c0020001-0000-0000-0000-000000000003'::uuid, 'Konsultasi Dokter', 'stethoscope',       '#10b981', 1),
    ('c0020001-0000-0000-0000-000000000003'::uuid, 'Obat-obatan',       'pill',              '#10b981', 2),
    ('c0020001-0000-0000-0000-000000000003'::uuid, 'Olahraga',          'dumbbell',          '#10b981', 3),
    ('c0020001-0000-0000-0000-000000000003'::uuid, 'Asuransi',          'shield',            '#10b981', 4),
    ('c0020001-0000-0000-0000-000000000004'::uuid, 'Kursus Online',     'monitor',           '#3b5bdb', 1),
    ('c0020001-0000-0000-0000-000000000004'::uuid, 'Buku & Alat Tulis', 'book',              '#3b5bdb', 2),
    ('c0020001-0000-0000-0000-000000000004'::uuid, 'SPP / Uang Sekolah','graduation-cap',    '#3b5bdb', 3),
    ('c0020001-0000-0000-0000-000000000004'::uuid, 'Pelatihan',         'briefcase',         '#3b5bdb', 4),
    ('c0020001-0000-0000-0000-000000000005'::uuid, 'Nonton Film',       'clapperboard',      '#06b6d4', 1),
    ('c0020001-0000-0000-0000-000000000005'::uuid, 'Liburan',           'plane',             '#06b6d4', 2),
    ('c0020001-0000-0000-0000-000000000005'::uuid, 'Hobi',              'palette',           '#06b6d4', 3),
    ('c0020001-0000-0000-0000-000000000005'::uuid, 'Streaming',         'tv',                '#06b6d4', 4),
    ('c0020001-0000-0000-0000-000000000005'::uuid, 'Game',              'gamepad-2',         '#06b6d4', 5),
    ('c0020001-0000-0000-0000-000000000006'::uuid, 'Listrik',           'zap',               '#f59e0b', 1),
    ('c0020001-0000-0000-0000-000000000006'::uuid, 'Air',               'droplets',          '#f59e0b', 2),
    ('c0020001-0000-0000-0000-000000000006'::uuid, 'Internet',          'wifi',              '#f59e0b', 3),
    ('c0020001-0000-0000-0000-000000000006'::uuid, 'Pulsa & Data',      'smartphone',        '#f59e0b', 4),
    ('c0020001-0000-0000-0000-000000000006'::uuid, 'Gas',               'flame',             '#f59e0b', 5),
    ('c0020001-0000-0000-0000-000000000007'::uuid, 'Pakaian',           'shirt',             '#ec4899', 1),
    ('c0020001-0000-0000-0000-000000000007'::uuid, 'Aksesoris',         'ring',              '#ec4899', 2),
    ('c0020001-0000-0000-0000-000000000007'::uuid, 'Belanja Online',    'package',           '#ec4899', 3),
    ('c0020001-0000-0000-0000-000000000007'::uuid, 'Sepatu',            'footprints',        '#ec4899', 4),
    ('c0020001-0000-0000-0000-000000000008'::uuid, 'Kontrakan',         'building',          '#3b5bdb', 1),
    ('c0020001-0000-0000-0000-000000000008'::uuid, 'KPR',               'landmark',          '#3b5bdb', 2),
    ('c0020001-0000-0000-0000-000000000008'::uuid, 'Perbaikan Rumah',   'wrench',            '#3b5bdb', 3),
    ('c0020001-0000-0000-0000-000000000008'::uuid, 'Perabotan',         'sofa',              '#3b5bdb', 4),
    ('c0020001-0000-0000-0000-000000000009'::uuid, 'Smartphone',        'smartphone',        '#8b5cf6', 1),
    ('c0020001-0000-0000-0000-000000000009'::uuid, 'Laptop/Komputer',   'monitor',           '#8b5cf6', 2),
    ('c0020001-0000-0000-0000-000000000009'::uuid, 'Aksesoris Gadget',  'headphones',        '#8b5cf6', 3),
    ('c0020001-0000-0000-0000-000000000009'::uuid, 'Komponen PC',       'cpu',               '#8b5cf6', 4),
    ('c0020001-0000-0000-0000-00000000000a'::uuid, 'Potong Rambut',     'scissors',          '#14b8a6', 1),
    ('c0020001-0000-0000-0000-00000000000a'::uuid, 'Skincare',          'sparkles',          '#14b8a6', 2),
    ('c0020001-0000-0000-0000-00000000000a'::uuid, 'Salon & Spa',       'flower-2',          '#14b8a6', 3),
    ('c0020001-0000-0000-0000-00000000000a'::uuid, 'Perawatan Gigi',    'tooth',             '#14b8a6', 4),
    ('c0020001-0000-0000-0000-00000000000b'::uuid, 'Donasi',            'hand-heart',        '#84cc16', 1),
    ('c0020001-0000-0000-0000-00000000000b'::uuid, 'Amplop Nikah',      'gift',              '#84cc16', 2),
    ('c0020001-0000-0000-0000-00000000000b'::uuid, 'Arisan',            'users',             '#84cc16', 3),
    ('c0020001-0000-0000-0000-00000000000b'::uuid, 'Nazar',             'church',            '#84cc16', 4),
    ('c0020001-0000-0000-0000-00000000000c'::uuid, 'Administrasi',      'file-text',         '#94a3b8', 1),
    ('c0020001-0000-0000-0000-00000000000c'::uuid, 'Denda',             'ban',               '#94a3b8', 2),
    ('c0020001-0000-0000-0000-00000000000c'::uuid, 'Lainnya',           'ellipsis',          '#94a3b8', 3),
    ('c0020001-0000-0000-0000-00000000000c'::uuid, 'Ongkir',            'truck',             '#94a3b8', 4)
) AS s(category_id, name, icon, color, sort_order)
WHERE NOT EXISTS (
    SELECT 1 FROM expense_subcategories sub
    WHERE sub.category_id = s.category_id AND sub.name = s.name
);

-- ============================================================
-- E. SAVINGS CATEGORIES (from v1 014)
-- ============================================================
INSERT INTO savings_categories (name, icon, color, description, sort_order) VALUES
    ('Tabungan Darurat', 'shield',        '#ef4444', 'Dana darurat untuk kebutuhan mendesak', 1),
    ('Investasi',        'trending-up',   '#10b981', 'Reksadana, saham, emas, dll',           2),
    ('Pensiun',          'clock',         '#3b5bdb', 'Dana persiapan pensiun',                3),
    ('Pendidikan',       'graduation-cap','#8b5cf6', 'Biaya pendidikan masa depan',           4),
    ('Liburan',          'plane',         '#f59e0b', 'Tabungan liburan',                      5),
    ('Rumah',            'home',          '#06b6d4', 'DP rumah, renovasi, dll',               6),
    ('Kendaraan',        'car',           '#ec4899', 'Tabungan kendaraan baru',               7),
    ('Lainnya',          'piggy-bank',    '#64748b', 'Kategori tabungan lainnya',             8)
ON CONFLICT DO NOTHING;

-- ============================================================
-- F. AI PROVIDERS (from v1 012/013)
-- ============================================================
INSERT INTO ai_providers (name, display_name, supports_fast, supports_reasoning) VALUES
    ('gemini',     'Google Gemini',      true,  true),
    ('openai',     'OpenAI',             true,  true),
    ('groq',       'Groq',               true,  false),
    ('openrouter', 'OpenRouter',         true,  true),
    ('claude',     'Claude (Anthropic)', false, true),
    ('qwen',       'Qwen',               true,  false),
    ('deepseek',   'DeepSeek',           true,  true),
    ('mistral',    'Mistral AI',         true,  false),
    ('cohere',     'Cohere',             true,  false),
    ('perplexity', 'Perplexity',         true,  false),
    ('other',      'Lainnya',            true,  false)
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- G. TRANSACTIONS (from v1 010 + 016), idempotent via NOT EXISTS
-- ============================================================
DO $$
DECLARE
    user_ids uuid[] := ARRAY[
        '00000000-0000-0000-0000-000000000002',
        '00000000-0000-0000-0000-000000000003',
        '00000000-0000-0000-0000-000000000004'
    ];
    uid uuid;
    base_date date;
    tx_date date;
    tx_type text;
    tx_amount numeric(15,2);
    inc_cat_id uuid;
    exp_cat_id uuid;
    sav_cat_id uuid;
    subcat_id uuid;
    tx_desc text;
    tx_source text;
    i int;
    j int;
    is_rec boolean;
    rec_int text;
    cat_idx int;
    subcat_ids uuid[];
    exp_cat_ids uuid[];
    inc_cat_ids uuid[];
    savings_cat_ids uuid[];
    descriptions_expense text[] := ARRAY[
        'Nasi Goreng', 'Mie Ayam', 'Bakso', 'Sate Ayam', 'Ayam Geprek',
        'Es Teh', 'Kopi Susu', 'Jus Buah', 'Air Mineral', 'Bubur Ayam',
        'Indomie Goreng', 'Seblak', 'Cilok', 'Siomay', 'Pempek',
        'Gojek ke kantor', 'Grab ke kampus', 'Bensin motor', 'Bensin mobil',
        'TransJakarta', 'MRT', 'KRL', 'Taksi online',
        'Obat flu', 'Vitamin C', 'Paracetamol', 'Dokter umum',
        'Kursus Inggris', 'Buku Python', 'Langganan Coursera', 'Alat tulis',
        'Netflix', 'Spotify', 'Bioskop XXI', 'Konser musik',
        'Listrik bulanan', 'Tagihan air', 'IndiHome', 'Pulsa Telkomsel',
        'Baju baru', 'Sepatu', 'Tas', 'Jam tangan',
        'Kontrakan bulanan', 'Perbaikan kran', 'Cat tembok', 'Sapu baru',
        'Case HP', 'Kabel data', 'Mouse wireless', 'Headset',
        'Potong rambut', 'Facial', 'Sabun muka', 'Shampoo',
        'Donasi bencana', 'Kotak amal', 'Amplop nikah teman', 'Arisan ibu-ibu',
        'Biaya admin bank', 'Parkir', 'Tol', 'Denda telat bayar'
    ];
    descriptions_income text[] := ARRAY[
        'Gaji bulanan', 'Gaji pokok', 'Tunjangan',
        'Project website', 'Design graphic', 'Writing article', 'Photography gig',
        'Dividen saham', 'Bunga deposito', 'Reksadana cair',
        'THR', 'Bonus tahunan', 'Insentif',
        'Jualan online', 'Hasil reseller', 'Katering',
        'Uang ultah', 'Hadiah lomba', 'THR Lebaran'
    ];
    savings_descs text[] := ARRAY[
        'Transfer tabungan darurat', 'Menabung untuk dana darurat',
        'Setoran investasi bulanan', 'Beli reksadana pasar uang',
        'Tabungan pendidikan anak', 'Dana pendidikan',
        'Tabungan liburan akhir tahun', 'Sisihkan untuk liburan',
        'DP rumah progres', 'Tabungan rumah', 'Cicil kendaraan baru',
        'Tabungan mobil', 'Dana pensiun', 'Iuran pensiun sukarela', 'Tabungan rutin'
    ];
    sources text[] := ARRAY['manual', 'manual', 'manual', 'ai', 'receipt', 'voice'];
BEGIN
    SELECT ARRAY_AGG(id) INTO inc_cat_ids FROM income_categories;
    SELECT ARRAY_AGG(id) INTO exp_cat_ids FROM expense_categories;
    SELECT ARRAY_AGG(id ORDER BY sort_order) INTO savings_cat_ids FROM savings_categories;

    FOREACH uid IN ARRAY user_ids
    LOOP
        -- Skip if this user already has seeded transactions (idempotent).
        IF EXISTS (SELECT 1 FROM transactions WHERE user_id = uid) THEN
            CONTINUE;
        END IF;

        -- 105 transactions per user: ~20 income + ~85 expense
        FOR i IN 1..105 LOOP
            base_date := '2026-03-16'::date;
            tx_date := base_date + (random() * 92)::int;

            IF i <= 20 THEN
                tx_type := 'income';
                cat_idx := 1 + (random() * (array_length(inc_cat_ids, 1) - 1))::int;
                inc_cat_id := inc_cat_ids[cat_idx];
                exp_cat_id := NULL;
                sav_cat_id := NULL;
                subcat_id := NULL;

                IF inc_cat_id = 'c0010001-0000-0000-0000-000000000001' THEN
                    tx_amount := (5 + random() * 15)::int * 1000000;
                    tx_desc := descriptions_income[1 + (random() * 1)::int];
                ELSIF inc_cat_id = 'c0010001-0000-0000-0000-000000000002' THEN
                    tx_amount := (500 + random() * 9500)::int * 1000;
                    tx_desc := descriptions_income[3 + (random() * 3)::int];
                ELSIF inc_cat_id = 'c0010001-0000-0000-0000-000000000003' THEN
                    tx_amount := (100 + random() * 2900)::int * 1000;
                    tx_desc := descriptions_income[6 + (random() * 2)::int];
                ELSIF inc_cat_id = 'c0010001-0000-0000-0000-000000000004' THEN
                    tx_amount := (1 + random() * 9)::int * 1000000;
                    tx_desc := descriptions_income[9 + (random() * 2)::int];
                ELSIF inc_cat_id = 'c0010001-0000-0000-0000-000000000005' THEN
                    tx_amount := (500 + random() * 4500)::int * 1000;
                    tx_desc := descriptions_income[12 + (random() * 2)::int];
                ELSE
                    tx_amount := (100 + random() * 900)::int * 1000;
                    tx_desc := descriptions_income[15 + (random() * 3)::int];
                END IF;

                is_rec := false;
                rec_int := NULL;
            ELSE
                tx_type := 'expense';
                inc_cat_id := NULL;
                sav_cat_id := NULL;
                cat_idx := 1 + (random() * (array_length(exp_cat_ids, 1) - 1))::int;
                exp_cat_id := exp_cat_ids[cat_idx];

                SELECT ARRAY_AGG(id) INTO subcat_ids
                FROM expense_subcategories
                WHERE category_id = exp_cat_id;
                IF subcat_ids IS NOT NULL AND array_length(subcat_ids, 1) > 0 THEN
                    subcat_id := subcat_ids[1 + (random() * (array_length(subcat_ids, 1) - 1))::int];
                ELSE
                    subcat_id := NULL;
                END IF;

                tx_amount := (5 + random() * 495)::int * 1000;
                tx_desc := descriptions_expense[1 + (random() * (array_length(descriptions_expense, 1) - 1))::int];

                is_rec := (random() < 0.05);
                IF is_rec THEN
                    rec_int := CASE (random() * 2)::int
                        WHEN 0 THEN 'daily'
                        WHEN 1 THEN 'weekly'
                        ELSE 'monthly'
                    END;
                ELSE
                    rec_int := NULL;
                END IF;
            END IF;

            tx_source := sources[1 + (random() * (array_length(sources, 1) - 1))::int];

            INSERT INTO transactions (
                user_id, type, amount, income_category_id, expense_category_id,
                savings_category_id, subcategory_id, description, tags, transaction_date,
                is_recurring, recurring_interval, source
            ) VALUES (
                uid, tx_type, tx_amount, inc_cat_id, exp_cat_id,
                sav_cat_id, subcat_id, tx_desc,
                CASE WHEN random() < 0.3 THEN ARRAY['penting', 'bulanan'] ELSE '{}' END,
                tx_date, is_rec, rec_int, tx_source
            );
        END LOOP;

        -- Explicit recurring monthly income (salary on 25th)
        FOR j IN 0..2 LOOP
            INSERT INTO transactions (
                user_id, type, amount, income_category_id, description,
                transaction_date, is_recurring, recurring_interval, source
            ) VALUES (
                uid, 'income', 7000000,
                'c0010001-0000-0000-0000-000000000001',
                'Gaji bulanan',
                ('2026-03-25'::date + (j * 30)::int),
                true, 'monthly', 'manual'
            );
        END LOOP;

        -- Explicit recurring monthly expense: rent on 1st
        FOR j IN 0..2 LOOP
            INSERT INTO transactions (
                user_id, type, amount, expense_category_id, subcategory_id,
                description, transaction_date, is_recurring, recurring_interval, source
            ) VALUES (
                uid, 'expense', 1500000,
                'c0020001-0000-0000-0000-000000000008',
                (SELECT id FROM expense_subcategories WHERE category_id = 'c0020001-0000-0000-0000-000000000008' AND name = 'Kontrakan' LIMIT 1),
                'Kontrakan bulanan',
                ('2026-04-01'::date + (j * 30)::int),
                true, 'monthly', 'manual'
            );
        END LOOP;

        -- Explicit recurring monthly expense: internet on 5th
        FOR j IN 0..2 LOOP
            INSERT INTO transactions (
                user_id, type, amount, expense_category_id, subcategory_id,
                description, transaction_date, is_recurring, recurring_interval, source
            ) VALUES (
                uid, 'expense', 350000,
                'c0020001-0000-0000-0000-000000000006',
                (SELECT id FROM expense_subcategories WHERE category_id = 'c0020001-0000-0000-0000-000000000006' AND name = 'Internet' LIMIT 1),
                'IndiHome bulanan',
                ('2026-04-05'::date + (j * 30)::int),
                true, 'monthly', 'manual'
            );
        END LOOP;

        -- Savings transactions (from v1 016)
        FOR i IN 1..10 LOOP
            base_date := '2026-03-16'::date;
            tx_date := base_date + (random() * 92)::int;
            tx_amount := (200 + random() * 4800)::int * 1000;

            sav_cat_id := savings_cat_ids[1 + (random() * (array_length(savings_cat_ids, 1) - 1))::int];
            tx_desc := savings_descs[1 + (random() * (array_length(savings_descs, 1) - 1))::int];

            INSERT INTO transactions (
                user_id, type, amount, savings_category_id,
                description, transaction_date, source
            ) VALUES (
                uid, 'savings', tx_amount, sav_cat_id,
                tx_desc, tx_date, 'manual'
            );
        END LOOP;

        -- Explicit monthly savings (from v1 016)
        FOR j IN 0..2 LOOP
            INSERT INTO transactions (
                user_id, type, amount, savings_category_id,
                description, transaction_date, is_recurring, recurring_interval, source
            ) VALUES (
                uid, 'savings', 500000,
                (SELECT id FROM savings_categories WHERE name = 'Tabungan Darurat' LIMIT 1),
                'Auto transfer gajian - dana darurat',
                ('2026-03-25'::date + (j * 30)::int),
                true, 'monthly', 'manual'
            );

            INSERT INTO transactions (
                user_id, type, amount, savings_category_id,
                description, transaction_date, is_recurring, recurring_interval, source
            ) VALUES (
                uid, 'savings', 300000,
                (SELECT id FROM savings_categories WHERE name = 'Investasi' LIMIT 1),
                'Auto investasi reksadana',
                ('2026-03-25'::date + (j * 30)::int),
                true, 'monthly', 'manual'
            );
        END LOOP;
    END LOOP;
END $$;

-- ============================================================
-- H. SPK DEMO RESULTS (from v1 010)
-- ============================================================
INSERT INTO spk_results (user_id, score, status, breakdown, period_start, period_end)
SELECT * FROM (VALUES
    ('00000000-0000-0000-0000-000000000002'::uuid, 72.50, 'Baik',
     '{"saving_ratio":0.25,"needs_vs_wants":0.70,"consistency":0.85,"spending_trend":0.65,"diversification":0.60}'::jsonb, '2026-04-01'::date, '2026-04-30'::date),
    ('00000000-0000-0000-0000-000000000002'::uuid, 75.00, 'Baik',
     '{"saving_ratio":0.28,"needs_vs_wants":0.72,"consistency":0.88,"spending_trend":0.62,"diversification":0.65}'::jsonb, '2026-05-01'::date, '2026-05-31'::date),
    ('00000000-0000-0000-0000-000000000002'::uuid, 78.50, 'Baik',
     '{"saving_ratio":0.30,"needs_vs_wants":0.75,"consistency":0.90,"spending_trend":0.60,"diversification":0.68}'::jsonb, '2026-06-01'::date, '2026-06-30'::date),
    ('00000000-0000-0000-0000-000000000003'::uuid, 85.00, 'Sangat Baik',
     '{"saving_ratio":0.35,"needs_vs_wants":0.80,"consistency":0.95,"spending_trend":0.70,"diversification":0.75}'::jsonb, '2026-04-01'::date, '2026-04-30'::date),
    ('00000000-0000-0000-0000-000000000003'::uuid, 86.50, 'Sangat Baik',
     '{"saving_ratio":0.36,"needs_vs_wants":0.82,"consistency":0.96,"spending_trend":0.68,"diversification":0.76}'::jsonb, '2026-05-01'::date, '2026-05-31'::date),
    ('00000000-0000-0000-0000-000000000003'::uuid, 88.00, 'Sangat Baik',
     '{"saving_ratio":0.38,"needs_vs_wants":0.84,"consistency":0.97,"spending_trend":0.65,"diversification":0.78}'::jsonb, '2026-06-01'::date, '2026-06-30'::date),
    ('00000000-0000-0000-0000-000000000004'::uuid, 45.00, 'Cukup',
     '{"saving_ratio":0.10,"needs_vs_wants":0.40,"consistency":0.60,"spending_trend":0.45,"diversification":0.35}'::jsonb, '2026-04-01'::date, '2026-04-30'::date),
    ('00000000-0000-0000-0000-000000000004'::uuid, 42.50, 'Cukup',
     '{"saving_ratio":0.08,"needs_vs_wants":0.38,"consistency":0.55,"spending_trend":0.50,"diversification":0.32}'::jsonb, '2026-05-01'::date, '2026-05-31'::date),
    ('00000000-0000-0000-0000-000000000004'::uuid, 38.00, 'Boros',
     '{"saving_ratio":0.05,"needs_vs_wants":0.35,"consistency":0.50,"spending_trend":0.55,"diversification":0.30}'::jsonb, '2026-06-01'::date, '2026-06-30'::date)
) AS s(user_id, score, status, breakdown, period_start, period_end)
WHERE NOT EXISTS (SELECT 1 FROM spk_results);

-- ============================================================
-- I. CHAT DEMO (chat_sessions + chat_messages)
-- Replaces the dead `ai_conversations` demo with the live schema.
-- ============================================================
DO $$
DECLARE
    sid uuid;
BEGIN
    IF EXISTS (SELECT 1 FROM chat_sessions) THEN
        RETURN;
    END IF;

    -- Budi
    INSERT INTO chat_sessions (user_id, title, source)
    VALUES ('00000000-0000-0000-0000-000000000002', 'Pengeluaran bulan ini', 'chatbot')
    RETURNING id INTO sid;
    INSERT INTO chat_messages (session_id, user_id, role, content, model_used, tokens_used) VALUES
        (sid, '00000000-0000-0000-0000-000000000002', 'user', 'Catat pengeluaran siomay 10.000 tadi pagi', 'gemini-2.0-flash', 45),
        (sid, '00000000-0000-0000-0000-000000000002', 'assistant', 'Sudah saya catat! Siomay Rp10.000 sebagai pengeluaran kategori Makanan & Minuman.', 'gemini-2.0-flash', 120),
        (sid, '00000000-0000-0000-0000-000000000002', 'user', 'Berapa total pengeluaran saya bulan ini?', 'gemini-2.0-flash', 30),
        (sid, '00000000-0000-0000-0000-000000000002', 'assistant', 'Total pengeluaran Anda bulan ini adalah Rp2.850.000. Kategori terbesar adalah Makanan & Minuman sebesar Rp1.200.000.', 'gemini-2.0-flash', 180);

    -- Siti
    INSERT INTO chat_sessions (user_id, title, source)
    VALUES ('00000000-0000-0000-0000-000000000003', 'Saran keuangan', 'chatbot')
    RETURNING id INTO sid;
    INSERT INTO chat_messages (session_id, user_id, role, content, model_used, tokens_used) VALUES
        (sid, '00000000-0000-0000-0000-000000000003', 'user', 'Buatkan laporan keuangan minggu ini', 'gemini-2.0-flash', 20),
        (sid, '00000000-0000-0000-0000-000000000003', 'assistant', 'Berikut laporan keuangan minggu ini. Total: Pengeluaran Rp540.000, Pemasukan Rp0.', 'gemini-2.0-flash', 250),
        (sid, '00000000-0000-0000-0000-000000000003', 'user', 'Saran keuangan untuk saya?', 'gemini-2.0-flash', 15),
        (sid, '00000000-0000-0000-0000-000000000003', 'assistant', 'Berdasarkan data keuangan Anda: alokasikan 20% untuk tabungan, batasi makanan 30%, dan siapkan dana darurat 3-6 bulan pengeluaran.', 'gemini-2.0-flash', 320);

    -- Andi
    INSERT INTO chat_sessions (user_id, title, source)
    VALUES ('00000000-0000-0000-0000-000000000004', 'Perkenalan', 'chatbot')
    RETURNING id INTO sid;
    INSERT INTO chat_messages (session_id, user_id, role, content, model_used, tokens_used) VALUES
        (sid, '00000000-0000-0000-0000-000000000004', 'user', 'Halo, apa kemampuanmu?', 'gemini-2.0-flash', 10),
        (sid, '00000000-0000-0000-0000-000000000004', 'assistant', 'Halo! Saya asisten keuangan FinTrack. Saya bisa membantu mencatat transaksi, menjawab pertanyaan keuangan, memberi saran, dan membuat laporan.', 'gemini-2.0-flash', 150);
END $$;
