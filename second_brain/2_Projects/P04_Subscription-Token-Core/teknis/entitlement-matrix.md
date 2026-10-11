# Matriks Entitlement — FinTrack v2

Dokumen ini mendefinisikan matriks hak akses (*entitlement matrix*) resmi untuk FinTrack v2 yang memisahkan kapabilitas sistem menjadi dua kategori utama:
1. **Fitur Boolean (`plan_entitlements`):** Saklar biner (`true` / `false`) yang menentukan apakah fitur tertentu diizinkan untuk digunakan.
2. **Limit Numerik (`plan_limits`):** Batasan kuota terukur (integer) untuk mencegah eksploitasi kapasitas sumber daya sistem. Nilai `-1` menandakan kuota tak terbatas (*unlimited*).

---

## 1. Matriks Perbandingan Paket (Free vs Pro vs Business)

| Kategori | Fitur / Kuota Key | Free | Pro | Business | Satuan / Keterangan |
|---|---|:---:|:---:|:---:|---|
| **Harga** | `price_monthly_idr` | Rp 0 | Rp 99.000 | Rp 299.000 | Per bulan (30 hari kalender) |
| | `price_yearly_idr` | Rp 0 | Rp 990.000 | Rp 2.990.000 | Per tahun (365 hari kalender) |
| **Kapasitas Dompet & Transaksi** | `max_wallets` | 2 | 10 | -1 (*unlimited*) | Jumlah dompet/rekening aktif |
| | `max_budgets` | 3 | 20 | -1 (*unlimited*) | Jumlah target anggaran aktif |
| | `max_custom_categories` | 5 | 50 | -1 (*unlimited*) | Kategori pengeluaran kustom |
| | `history_retention_days` | 90 | 730 | -1 (*unlimited*) | Retensi pencarian data historis |
| **Kecerdasan Buatan (AI)** | `ai_chat` | ✅ | ✅ | ✅ | Asisten chat finansial |
| | `ai_tokens_per_cycle` | 10.000 | 250.000 | 1.000.000 | Token AI per siklus (reset tiap awal siklus) |
| | `ocr_receipt` | ❌ | ✅ | ✅ | Pindai struk/nota otomatis |
| | `ocr_scans_per_cycle` | 0 | 25 | 150 | Jumlah scan nota per hari |
| | `voice_input` | ❌ | ✅ | ✅ | Input transaksi via rekaman suara |
| | `ai_financial_advisor` | ❌ | ✅ | ✅ | Analisis anomali & saran budget AI |
| **Integrasi & Sinkronisasi** | `bank_sync` | ❌ | ✅ | ✅ | Integrasi mutasi bank otomatis |
| | `max_synced_bank_accounts`| 0 | 3 | 10 | Akun bank terhubung |
| | `multi_currency` | ❌ | ✅ | ✅ | Transaksi valuta asing (valas) |
| **Laporan & Ekspor** | `export_csv` | ✅ | ✅ | ✅ | Unduh transaksi format CSV |
| | `export_pdf` | ❌ | ✅ | ✅ | Unduh laporan laba/rugi format PDF |
| | `advanced_analytics` | ❌ | ✅ | ✅ | Chart tren interaktif, peramalan |
| **Layanan & SLA** | `priority_support` | ❌ | ❌ | ✅ | Jalur dukungan pelanggan prioritas |
| | `api_access` | ❌ | ❌ | ✅ | Akses REST API personal untuk automasi |

---

## 2. Definisi Kunci Fitur Boolean (`plan_entitlements`)

Setiap fitur memiliki string konstan terdaftar yang digunakan dalam evaluasi middleware:

```go
package domain

const (
    // Fitur Dasar & Ekspor
    FeatureExportCSV         = "export_csv"
    FeatureExportPDF         = "export_pdf"
    FeatureAdvancedAnalytics = "advanced_analytics"
    FeatureMultiCurrency     = "multi_currency"

    // Fitur AI
    FeatureAIChat            = "ai_chat"
    FeatureOCRReceipt        = "ocr_receipt"
    FeatureVoiceInput        = "voice_input"
    FeatureAIFinancialAdvisor= "ai_financial_advisor"

    // Integrasi & Layanan
    FeatureBankSync          = "bank_sync"
    FeaturePrioritySupport   = "priority_support"
    FeatureAPIAccess         = "api_access"
)
```

### Rincian Perilaku Fitur:
1. `export_csv`: Memungkinkan pengguna mengunduh riwayat transaksi dalam format CSV/Excel sederhana. Tersedia untuk seluruh paket.
2. `export_pdf`: Memungkinkan pembuatan dokumen cetak PDF laporan arus kas formal dengan grafik ringkasan. Terkunci untuk paket Free.
3. `ocr_receipt`: Memungkinkan pengunggahan gambar struk belanja belanjaan untuk diekstraksi menjadi transaksi otomatis melalui modul multimodal AI.
4. `voice_input`: Memungkinkan perekaman pesan suara (*voice note*) untuk pencatatan pengeluaran instan menggunakan model Speech-to-Text.
5. `bank_sync`: Menghubungkan akun bank (BCA, Mandiri, BRI, dll) melalui aggregator Open Finance untuk penarikan mutasi otomatis.
6. `advanced_analytics`: Membuka akses visualisasi tren arus kas multi-kuartal, heatmap pengeluaran, dan prediksi arus kas 30 hari ke depan.
7. `api_access`: Memberikan personal API token untuk mengintegrasikan FinTrack dengan bot Telegram pribadi, script automasi n8n, atau webhook kustom.

---

## 3. Definisi Kunci Batasan Limit Numerik (`plan_limits`)

Limit kuota dinyatakan dalam bilangan bulat (`int64`):

```go
package domain

const (
    // Batasan Dompet & Anggaran
    LimitMaxWallets             = "max_wallets"
    LimitMaxBudgets             = "max_budgets"
    LimitMaxCustomCategories    = "max_custom_categories"
    LimitHistoryRetentionDays   = "history_retention_days"

    // Batasan AI & Pemrosesan
    LimitAITokensPerCycle          = "ai_tokens_per_cycle"
    LimitDailyOCRScans          = "ocr_scans_per_cycle"
    LimitMaxSyncedBankAccounts  = "max_synced_bank_accounts"
)

const LimitUnlimited = int64(-1)
```

### Aturan Evaluasi Nilai Limit:
- Jika `limit_value == -1`: Pemeriksaan batas selalu lolos (*allow any amount*).
- Jika `limit_value >= 0`: Jumlah pemakaian aktual (*current usage*) tidak boleh melampaui `limit_value`.
- Upaya penambahan data baru saat `current_usage >= limit_value` menghasilkan error HTTP `429 Too Many Requests` dengan kode `LIMIT_EXCEEDED`.

---

## 4. Skrip Seeder SQL (`database/seeds/0007_default_plans.sql`)

Cuplikan SQL seeder untuk mengisi matriks default:

```sql
-- 1. Insert Plans
INSERT INTO plans (id, code, name, description, price_monthly_idr, price_yearly_idr, is_active, is_public, sort_order)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 'free', 'Free Starter', 'Paket gratis untuk pencatatan keuangan harian mandiri', 0, 0, TRUE, TRUE, 1),
    ('22222222-2222-2222-2222-222222222222', 'pro', 'Pro Personal', 'Paket lengkap dengan asisten AI dan analitik mendalam', 99000, 990000, TRUE, TRUE, 2),
    ('33333333-3333-3333-3333-333333333333', 'business', 'Business Pro', 'Paket tanpa batas untuk profesional dan usaha mikro', 299000, 2990000, TRUE, TRUE, 3)
ON CONFLICT (code) DO NOTHING;

-- 2. Insert Boolean Entitlements (Contoh Free)
INSERT INTO plan_entitlements (plan_id, feature_key, is_enabled) VALUES
    ('11111111-1111-1111-1111-111111111111', 'export_csv', TRUE),
    ('11111111-1111-1111-1111-111111111111', 'export_pdf', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'advanced_analytics', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'multi_currency', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'ai_chat', TRUE),
    ('11111111-1111-1111-1111-111111111111', 'ocr_receipt', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'voice_input', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'ai_financial_advisor', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'bank_sync', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'priority_support', FALSE),
    ('11111111-1111-1111-1111-111111111111', 'api_access', FALSE)
ON CONFLICT (plan_id, feature_key) DO UPDATE SET is_enabled = EXCLUDED.is_enabled;

-- 3. Insert Boolean Entitlements (Pro)
INSERT INTO plan_entitlements (plan_id, feature_key, is_enabled) VALUES
    ('22222222-2222-2222-2222-222222222222', 'export_csv', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'export_pdf', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'advanced_analytics', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'multi_currency', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'ai_chat', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'ocr_receipt', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'voice_input', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'ai_financial_advisor', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'bank_sync', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'priority_support', FALSE),
    ('22222222-2222-2222-2222-222222222222', 'api_access', FALSE)
ON CONFLICT (plan_id, feature_key) DO UPDATE SET is_enabled = EXCLUDED.is_enabled;

-- 4. Insert Limits (Free vs Pro vs Business)
INSERT INTO plan_limits (plan_id, limit_key, limit_value) VALUES
    ('11111111-1111-1111-1111-111111111111', 'max_wallets', 2),
    ('11111111-1111-1111-1111-111111111111', 'max_budgets', 3),
    ('11111111-1111-1111-1111-111111111111', 'max_custom_categories', 5),
    ('11111111-1111-1111-1111-111111111111', 'ai_tokens_per_cycle', 10000),
    ('11111111-1111-1111-1111-111111111111', 'ocr_scans_per_cycle', 0),
    ('11111111-1111-1111-1111-111111111111', 'history_retention_days', 90),
    
    ('22222222-2222-2222-2222-222222222222', 'max_wallets', 10),
    ('22222222-2222-2222-2222-222222222222', 'max_budgets', 20),
    ('22222222-2222-2222-2222-222222222222', 'max_custom_categories', 50),
    ('22222222-2222-2222-2222-222222222222', 'ai_tokens_per_cycle', 250000),
    ('22222222-2222-2222-2222-222222222222', 'ocr_scans_per_cycle', 25),
    ('22222222-2222-2222-2222-222222222222', 'history_retention_days', 730),

    ('33333333-3333-3333-3333-333333333333', 'max_wallets', -1),
    ('33333333-3333-3333-3333-333333333333', 'max_budgets', -1),
    ('33333333-3333-3333-3333-333333333333', 'max_custom_categories', -1),
    ('33333333-3333-3333-3333-333333333333', 'ai_tokens_per_cycle', 1000000),
    ('33333333-3333-3333-3333-333333333333', 'ocr_scans_per_cycle', 150),
    ('33333333-3333-3333-3333-333333333333', 'history_retention_days', -1)
ON CONFLICT (plan_id, limit_key) DO UPDATE SET limit_value = EXCLUDED.limit_value;
```

---

## 5. Algoritma Resolusi Entitlement Pengguna

Ketika sebuah request memerlukan pemeriksaan hak akses:
1. **Pemeriksaan Role:** Jika `user.role == "super_admin"`, seluruh fitur dianggap `true` dan semua limit diset `-1` (*unlimited*).
2. **Pencarian Subscription Aktif:** Query subscription aktif pengguna di basis data:
   - Jika status langganan adalah `active` atau `trialing`: Ambil `plan_id` terkait.
   - Jika status adalah `past_due` atau `grace`: Tetap berikan hak akses paket berbayar, namun tandai metadata peringatan.
   - Jika status adalah `canceled`: Selama `now() <= current_period_end`, berikan hak akses paket lama.
   - Jika `expired` atau tidak ditemukan: Mundur ke paket `free` (*fallback to free plan*).
3. **Agregasi Entitlements & Limits:** Baca `plan_entitlements` dan `plan_limits` untuk paket tersebut.
4. **Penyimpanan ke Cache:** Susun objek `UserEntitlement` dan simpan ke L1 (Memory) dan L2 (Redis) terasosiasi dengan versi pengguna saat ini.
