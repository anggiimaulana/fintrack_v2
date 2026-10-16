# 09 — Lapisan Konfigurasi Dinamis

[← Index](./README.md) · [← Risiko & Rollback](./08-risiko-dan-rollback.md)

---

## 1. Tujuan

> **Permintaan user:** *"saya ingin semuanya itu dinamis tanpa ada yang hardcode atau
> ditulis pada kode. semua keputusan, logic, flow, dan kalkulasi itu dari super admin
> dan dinamis. metrix juga harus dinamis."*

Dokumen ini mendefinisikan **lapisan konfigurasi** yang membuat perilaku sistem
ditentukan **data di database**, bukan konstanta di kode.

---

## 2. Audit: hardcode yang ditemukan

| # | Lokasi | Isi hardcode | Dampak |
|---|---|---|---|
| **H-1** | `admin/settings.go:76` | `writableSettings` — allowlist key + validator (`kind`, `min`, `max`, `enum`, `maxLen`) | Setting baru **wajib** ubah kode + deploy |
| **H-2** | `usage_resolvers.go:35` | `UsageFor()` — `switch limitKey` → resolver | Limit key baru **wajib** ubah kode |
| **H-3** | `entitlement/catalog.go` | `FeatureCatalog`, `LimitCatalog` | Fitur/limit baru **wajib** ubah kode |
| **H-4** | `billing/ledger.go` | `ceilDiv` (mode pembulatan tetap) | Operator tidak bisa memilih floor/round |
| **H-5** | `billing/estimate.go` | `chatOverheadTokens = 3000`, `charsPerToken = 4` | Estimasi tidak bisa dituning tanpa deploy |
| **H-6** | `seeds/0002_*.sql` | Kuota/limit/harga awal | Sebenarnya SQL (bukan kode) — **sudah dinamis** ✅ |
| **H-7** | `cmd/server/main.go` | `quota < 0 → 1_000_000_000` (sentinel unlimited) | Konstanta sihir di kode |
| **H-8** | `router.go` | Daftar rute + gate yang dipasang | Struktur rute (wajar di kode) |

**Kesimpulan:** H-1, H-2, H-3, H-4, H-5, H-7 **harus** dipindahkan ke data.
H-8 (rute) wajar di kode — rute adalah struktur aplikasi, bukan konfigurasi bisnis.

---

## 3. Arsitektur lapisan konfigurasi

```
┌──────────────────────────────────────────────────────────────────────┐
│                        SUPER ADMIN UI                                │
│   Editor Plan · Editor Settings · Editor Metrics · Kalkulator        │
└───────────────────────────┬──────────────────────────────────────────┘
                            │ HTTP (GET/PUT /admin/...)
┌───────────────────────────▼──────────────────────────────────────────┐
│                       API ADMIN (validasi dari DB)                   │
└───────────────────────────┬──────────────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────────────┐
│                    TABEL KONFIGURASI (DB)                            │
│  ┌────────────────────┐ ┌────────────────────┐ ┌──────────────────┐  │
│  │ setting_definitions│ │ metric_definitions │ │feature_definitions│ │
│  │  (key, kind, min,  │ │ (key, label, unit, │ │ (key, label,     │  │
│  │   max, enum, desc) │ │  resolver, sort)   │ │  desc, sort)     │  │
│  └────────────────────┘ └────────────────────┘ └──────────────────┘  │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────────┐  │
│  │   settings   │ │ plan_limits  │ │ plan_entitlements / plans    │  │
│  │ (nilai aktual)│ │(kuota/plan) │ │ (fitur & rasio/plan)         │  │
│  └──────────────┘ └──────────────┘ └──────────────────────────────┘  │
└───────────────────────────┬──────────────────────────────────────────┘
                            │ dibaca runtime + cache BER-VERSI (invalidasi SSE)
┌───────────────────────────▼──────────────────────────────────────────┐
│                    RUNTIME RESOLVERS (kode = seam)                   │
│   SettingsResolver · MetricsResolver · EntitlementResolver           │
│   Ledger (rounding dari setting) · EstimateStore (dari setting)      │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.1 Batas yang jujur: "seam" yang tidak bisa dihilangkan

| Seam | Kenapa tetap di kode | Yang jadi data |
|---|---|---|
| **Cara menghitung usage** | Kode harus tahu cara `SELECT COUNT(*)` untuk `max_transactions` | **Pemetaan** limit_key → resolver (kolom `resolver_key`) |
| **Struktur rute** | Gin butuh definisi rute saat boot | Rute mana yang dipasang gate mana (sudah data via `limit_key`) |
| **Config bootstrap** | `DATABASE_URL`, `JWT_SECRET`, `ENCRYPTION_KEY`, `PORT` dibutuhkan **sebelum** DB bisa dibaca | Sisa konfigurasi lain → DB |
| **Katalog resolver** | Harus ada kode yang mengeksekusi | Daftar resolver yang tersedia → data, dipilih via dropdown |

> **Prinsip:** yang di-kode adalah **kemampuan** (capability). Yang di-data adalah
> **kebijakan** (policy). Operator memilih *kebijakan*; developer menyediakan
> *kemampuan*. Ini pola yang sama dengan `pricing_rules` yang sudah ada.

---

## 4. Tabel `setting_definitions` (menggantikan H-1)

### 4.1 Skema

```sql
-- ============================================================
-- setting_definitions: katalog setting yang dapat diubah operator.
-- Menggantikan allowlist hardcoded `writableSettings` di admin/settings.go.
-- ============================================================
CREATE TABLE IF NOT EXISTS setting_definitions (
    key           varchar(64) PRIMARY KEY,
    -- kind menentukan validasi & editor yang dipakai UI:
    --   string | int | bool | enum | secret
    kind          varchar(16) NOT NULL,
    -- Batas untuk kind = int (NULL = tak terbatas).
    min_value     bigint,
    max_value     bigint,
    -- Daftar nilai untuk kind = enum (JSONB array string).
    enum_values   jsonb,
    -- Batas panjang untuk kind = string (0 = tak terbatas).
    max_len       integer NOT NULL DEFAULT 0,
    -- Nilai default bila setting belum pernah diisi operator.
    default_value jsonb,
    -- Group untuk pengelompokan di UI (mis. "kuota", "pajak", "mail").
    group_key     varchar(32) NOT NULL DEFAULT 'umum',
    label         varchar(128) NOT NULL DEFAULT '',
    description   text NOT NULL DEFAULT '',
    -- Apakah operator boleh mengubah? (false = read-only/diurus endpoint lain)
    is_writable   boolean NOT NULL DEFAULT true,
    -- Apakah nilainya rahasia (tidak pernah dikirim ke klien)?
    is_secret     boolean NOT NULL DEFAULT false,
    sort_order    integer NOT NULL DEFAULT 0,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT setting_definitions_kind_check
        CHECK (kind IN ('string','int','bool','enum','secret'))
);

CREATE OR REPLACE TRIGGER on_setting_definition_updated
    BEFORE UPDATE ON setting_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_setting_definitions_group
    ON setting_definitions (group_key, sort_order);
```

### 4.2 Perubahan `admin/settings.go`

```go
// SEBELUM: peta hardcoded
var writableSettings = map[string]settingSpec{ "tax_rate_percent": {...}, ... }

// SESUDAH: dibaca dari DB, cache ber-versi (invalidasi event-driven, P-13)
type SettingsCatalog struct {
    defs map[string]SettingDefinition
    ttl  time.Duration
}

// Validate memakai definisi dari DB, bukan konstanta kode.
func (c *SettingsCatalog) Validate(key string, raw json.RawMessage) error {
    def, ok := c.defs[key]
    if !ok { return ErrSettingNotWritable }
    if !def.IsWritable { return ErrSettingNotWritable }
    return validateByKind(def, raw)   // kind/min/max/enum/maxLen dari DB
}
```

**Efek:** menambah setting baru = `INSERT INTO setting_definitions`. **Tanpa deploy.**

### 4.3 Seed definisi (memindahkan `writableSettings` ke data)

```sql
INSERT INTO setting_definitions
    (key, kind, min_value, max_value, enum_values, max_len, default_value,
     group_key, label, description, is_secret, sort_order)
VALUES
    -- Pajak
    ('tax_enabled', 'bool', NULL, NULL, NULL, 0, 'false', 'pajak',
     'Aktifkan pajak', 'Apakah pajak diterapkan pada invoice.', false, 10),
    ('tax_rate_percent', 'int', 0, 100, NULL, 0, '0', 'pajak',
     'Persentase pajak', 'Persentase pajak (0-100).', false, 11),
    ('tax_label', 'string', NULL, NULL, NULL, 32, '"PPN"', 'pajak',
     'Label pajak', 'Nama pajak pada invoice (mis. PPN, PB1).', false, 12),
    -- Kuota (BARU — lihat §5)
    ('quota_rounding_mode', 'enum', NULL, NULL,
     '["ceil","floor","round"]'::jsonb, 0, '"ceil"', 'kuota',
     'Mode pembulatan kuota',
     'Cara membulatkan unit kuota: ceil (ke atas), floor (ke bawah), round (terdekat).',
     false, 20),
    ('quota_default_ratio', 'int', 1, 1000, NULL, 0, '1', 'kuota',
     'Rasio kuota default', 'Rasio awal untuk plan baru.', false, 21),
    ('quota_default_tokens_per_cycle', 'int', 1, 1000000000, NULL, 0, '10000', 'kuota',
     'Kuota default / siklus',
     'Kuota awal untuk plan baru (dalam unit kuota).', false, 22),
    ('quota_unit_label', 'string', NULL, NULL, NULL, 24, '"kuota"', 'kuota',
     'Label satuan kuota', 'Satuan yang ditampilkan ke user.', false, 23),
    ('quota_unlimited_sentinel', 'int', 1, 1000000000000, NULL, 0, '1000000000', 'kuota',
     'Nilai pengganti unlimited',
     'Angka yang dipakai saat limit plan = -1 (unlimited).', false, 24),
    -- Estimasi AI (H-5)
    ('ai_estimate_overhead_tokens', 'int', 0, 1000000, NULL, 0, '3000', 'ai',
     'Overhead estimasi', 'Token tetap per chat (system prompt + tools + RAG + riwayat).',
     false, 30),
    ('ai_estimate_output_headroom', 'int', 0, 1000000, NULL, 0, '800', 'ai',
     'Headroom keluaran', 'Perkiraan token keluaran per chat.', false, 31),
    ('ai_estimate_chars_per_token', 'int', 1, 100, NULL, 0, '4', 'ai',
     'Karakter per token', 'Perkiraan rasio karakter→token untuk estimasi.', false, 32),
    ('ai_estimate_chat', 'int', 1, 1000000, NULL, 0, '2000', 'ai',
     'Estimasi chat (lantai)', 'Batas bawah reserve untuk modalitas chat.', false, 33),
    -- ... (key lain dari writableSettings dipindahkan serupa)
ON CONFLICT (key) DO UPDATE SET
    kind = EXCLUDED.kind, min_value = EXCLUDED.min_value, max_value = EXCLUDED.max_value,
    enum_values = EXCLUDED.enum_values, max_len = EXCLUDED.max_len,
    group_key = EXCLUDED.group_key, label = EXCLUDED.label,
    description = EXCLUDED.description, sort_order = EXCLUDED.sort_order
WHERE setting_definitions.label = '';  -- jangan menimpa penyesuaian operator
```

> **Catatan:** `default_value` **tidak** ditimpa oleh `ON CONFLICT` (agar operator bisa
> mengubah default). Kolom `default_value` juga bisa diedit operator via UI.

---

## 5. Tabel `metric_definitions` (menggantikan H-2 & H-3)

"Metrix harus dinamis" → katalog limit/fitur menjadi data.

### 5.1 Skema

```sql
-- ============================================================
-- metric_definitions: katalog limit (metrik) yang dapat ditegakkan.
-- Menggantikan LimitCatalog hardcoded + switch UsageFor().
-- ============================================================
CREATE TABLE IF NOT EXISTS metric_definitions (
    key            varchar(64) PRIMARY KEY,
    label          varchar(128) NOT NULL,
    -- Satuan tampilan (mis. "kuota", "permintaan", "transaksi").
    unit           varchar(32) NOT NULL DEFAULT '',
    description    text NOT NULL DEFAULT '',
    -- Apakah backend menghitung usage untuk metrik ini? Metrik non-enforced
    -- hanya informasional (ditampilkan di plan tanpa gate).
    is_enforced    boolean NOT NULL DEFAULT false,
    -- Resolver mana yang dipakai untuk menghitung usage. Nilai harus salah satu
    -- resolver yang tersedia di kode (lihat §5.3). NULL = tidak dihitung.
    resolver_key   varchar(64),
    -- Apakah nilainya dipengaruhi rasio token (untuk catatan di UI)?
    ratio_adjusted boolean NOT NULL DEFAULT false,
    -- Apakah metrik ini diukur per siklus (true) atau kumulatif (false)?
    per_cycle      boolean NOT NULL DEFAULT true,
    group_key      varchar(32) NOT NULL DEFAULT 'kuota',
    sort_order     integer NOT NULL DEFAULT 0,
    is_active      boolean NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_metric_definition_updated
    BEFORE UPDATE ON metric_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE INDEX IF NOT EXISTS idx_metric_definitions_group
    ON metric_definitions (group_key, sort_order);

-- ============================================================
-- feature_definitions: katalog fitur boolean.
-- Menggantikan FeatureCatalog hardcoded.
-- ============================================================
CREATE TABLE IF NOT EXISTS feature_definitions (
    key         varchar(64) PRIMARY KEY,
    label       varchar(128) NOT NULL,
    description text NOT NULL DEFAULT '',
    group_key   varchar(32) NOT NULL DEFAULT 'fitur',
    sort_order  integer NOT NULL DEFAULT 0,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_feature_definition_updated
    BEFORE UPDATE ON feature_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

### 5.2 Seed (memindahkan katalog ke data)

```sql
INSERT INTO metric_definitions
    (key, label, unit, description, is_enforced, resolver_key, ratio_adjusted, sort_order)
VALUES
    ('ai_tokens_per_cycle', 'Kuota AI / siklus', 'kuota',
     'Kuota AI per siklus, diukur dalam unit kuota (bukan token mentah).',
     true, 'token_period_usage', true, 10),
    ('ai_requests_per_cycle', 'Permintaan AI / siklus', 'permintaan',
     'Jumlah panggilan AI (chat, OCR, suara) per siklus.',
     true, 'ai_requests_in_period', false, 20),
    ('max_transactions', 'Transaksi', 'transaksi',
     'Transaksi aktif yang dapat disimpan.', true, 'active_transactions', false, 30),
    ('ocr_scans_per_cycle', 'Pindai struk / siklus', 'scan',
     'Pemindaian struk (OCR) per siklus.', true, 'vision_scans_in_period', false, 40),
    ('max_categories', 'Kategori', 'kategori',
     'Jumlah kategori maksimum (belum ditegakkan).', false, NULL, false, 50),
    ('max_wallets', 'Dompet', 'dompet',
     'Jumlah dompet maksimum (belum ditegakkan).', false, NULL, false, 60)
ON CONFLICT (key) DO UPDATE SET
    label = EXCLUDED.label, unit = EXCLUDED.unit,
    description = EXCLUDED.description, resolver_key = EXCLUDED.resolver_key,
    ratio_adjusted = EXCLUDED.ratio_adjusted
WHERE metric_definitions.label = '';

INSERT INTO feature_definitions (key, label, description, sort_order) VALUES
    ('ai_chat', 'Asisten AI (chat)', 'Chat keuangan dengan AI.', 10),
    ('ocr_receipt', 'Pindai struk (OCR)', 'Ekstraksi data dari foto struk.', 20),
    ('voice_input', 'Input suara', 'Transkripsi perintah suara.', 30),
    ('rag_search', 'Pencarian cerdas (RAG)', 'Pencarian semantik atas transaksi.', 40),
    ('reports_export', 'Ekspor laporan', 'Unduh laporan keuangan.', 50),
    ('advanced_budgeting', 'Anggaran lanjutan', 'Anggaran & analisis lanjutan.', 60),
    ('api_access', 'Akses API', 'Akses API publik.', 70),
    ('bank_sync', 'Sinkronisasi bank', 'Sinkronisasi rekening bank (belum tersedia).', 80)
ON CONFLICT (key) DO UPDATE SET
    label = EXCLUDED.label, description = EXCLUDED.description
WHERE feature_definitions.label = '';
```

### 5.3 Registry resolver (seam yang diakui)

Kode menyediakan **daftar kemampuan**; data memilih mana yang dipakai.

```go
// middleware/usage_resolvers.go — SESUDAH
//
// resolverRegistry memetakan resolver_key (dari metric_definitions) ke fungsi
// yang menghitung usage. Kode menyediakan KEMAMPUAN; database memilih KEBIJAKAN.
func (p *UsageProviders) resolverRegistry() map[string]UsageResolver {
    return map[string]UsageResolver{
        "token_period_usage":     p.currentTokens,
        "ai_requests_in_period":  p.countAIRequests,
        "active_transactions":    p.countTransactions,
        "vision_scans_in_period": p.countVisionScans,
        // Tambah kemampuan baru di sini (deploy) — lalu pilih via DB (tanpa deploy).
    }
}

// UsageFor kini memakai definisi dari DB, bukan switch hardcoded.
func (p *UsageProviders) UsageFor(ctx context.Context, limitKey string) UsageResolver {
    def, err := p.metrics.Get(ctx, limitKey)   // dari metric_definitions
    if err != nil || def == nil || !def.IsEnforced || def.ResolverKey == nil {
        return nil
    }
    return p.resolverRegistry()[*def.ResolverKey]
}
```

> **Perhatikan:** `UsageFor` kini butuh `context` (query DB). Pemanggil
> (`limitGate`) harus meneruskan `c.Request.Context()`. Untuk menghindari query
> tiap request, gunakan **cache ber-versi** — dimuat sekali, di-drop hanya saat
> operator mengubah katalog (lihat §12, keputusan P-13).

**Rekomendasi:** gunakan `MetricsCatalog` ber-versi (invalidasi event-driven):

```go
type MetricsCatalog struct {
    reader MetricsReader
    mu     sync.RWMutex
    cache  map[string]MetricDefinition
    ver    int64          // versi terakhir yang dimuat (P-13)
}
```

### 5.4 Menambah metrik baru (tanpa deploy)

```
1. Developer: tambah resolver di registry (mis. "budgets_created")   ← deploy
2. Operator:  INSERT INTO metric_definitions (key, resolver_key='budgets_created', ...)
3. Operator:  set plan_limits untuk key baru
4. Selesai — gate otomatis bekerja, UI otomatis menampilkan
```

> Langkah 1 tetap perlu deploy **hanya bila** butuh **cara hitung baru**. Bila
> resolvernya sudah ada (mis. ingin `ai_requests_per_cycle` untuk modalitas lain),
> **tidak perlu deploy** sama sekali.

---

## 6. Pembulatan dinamis (menggantikan H-4)

### 6.1 Setting `quota_rounding_mode`

| Nilai | Perilaku | Kapan dipakai |
|---|---|---|
| `ceil` | Ke atas (konservatif untuk bisnis) | **Default** — jangan menagih kurang |
| `floor` | Ke bawah (ramah user) | Bila ingin user lebih hemat |
| `round` | Terdekat | Kompromi |

```go
// billing/rounding.go — BARU
type RoundingMode string

const (
    RoundCeil  RoundingMode = "ceil"
    RoundFloor RoundingMode = "floor"
    RoundRound RoundingMode = "round"
)

// RoundUnits membulatkan `raw/ratio` sesuai mode yang dikonfigurasi operator.
// Mode berasal dari setting `quota_rounding_mode` (dinamis, bukan konstanta kode).
func RoundUnits(raw, ratio int64, mode RoundingMode) int64 {
    if ratio < 1 { ratio = 1 }
    if raw <= 0 { return 0 }
    switch mode {
    case RoundFloor:
        return raw / ratio
    case RoundRound:
        return (raw + ratio/2) / ratio
    default: // RoundCeil — paling aman
        return (raw + ratio - 1) / ratio
    }
}
```

`LedgerService` menerima `RoundingMode` dari resolver (bukan konstanta):

```go
// LedgerService menyimpan RoundingReader (dari settings), bukan nilai tetap.
func (s *LedgerService) WithRounding(r RoundingReader) *LedgerService { ... }

// di Commit/Reserve/Refund:
mode := s.rounding.Mode(ctx)   // baca setting (cache ber-versi)
units := RoundUnits(rawTokens, ratio, mode)
```

**Konsekuensi penting:** dengan `floor`, `RoundUnits(1, 5) = 0` → request sangat kecil
**tidak menagih kuota**. Ini bisa membuka celah spam. **Mitigasi:** operator dapat
menetapkan `quota_min_charge_units` (setting baru, default `1`) yang menjadi lantai:

```go
if units < minCharge && raw > 0 { units = minCharge }
```

---

## 7. Estimasi dinamis (menggantikan H-5)

`EstimateStore` membaca overhead & rasio dari settings (bukan konstanta):

```go
// billing/estimate.go — SESUDAH
func (s *EstimateStore) ForChat(ctx context.Context, modality Modality, promptChars int64) int64 {
    overhead := s.cfg.Int(ctx, "ai_estimate_overhead_tokens", 3000)
    headroom := s.cfg.Int(ctx, "ai_estimate_output_headroom", 800)
    charsPer := s.cfg.Int(ctx, "ai_estimate_chars_per_token", 4)
    if charsPer < 1 { charsPer = 1 }

    base := s.For(ctx, modality)          // lantai per modalitas
    estimated := overhead + (promptChars/charsPer) + headroom
    if estimated < base { return base }
    return estimated
}
```

> Nilai `3000`/`800`/`4` hanya **fallback** bila setting belum diisi. Setelah seed
> dijalankan, nilainya berasal dari DB dan dapat diubah operator.

---

## 8. Sentinel unlimited dinamis (menggantikan H-7)

```go
// SEBELUM (cmd/server/main.go)
if quota < 0 { quota = 1000000000 }   // konstanta sihir

// SESUDAH
if quota < 0 {
    quota = h.settings.Int(ctx, "quota_unlimited_sentinel", 1_000_000_000)
}
```

---

## 9. API katalog konfigurasi

| Method | Path | Fungsi |
|---|---|---|
| `GET` | `/admin/setting-definitions` | Daftar definisi setting (untuk render UI) |
| `PUT` | `/admin/setting-definitions/:key` | Ubah definisi (min/max/enum/default/label) |
| `GET` | `/admin/metric-definitions` | Daftar metrik |
| `POST` | `/admin/metric-definitions` | Tambah metrik baru |
| `PUT` | `/admin/metric-definitions/:key` | Ubah metrik |
| `DELETE` | `/admin/metric-definitions/:key` | Nonaktifkan metrik |
| `GET` | `/admin/feature-definitions` | Daftar fitur |
| `POST` | `/admin/feature-definitions` | Tambah fitur |
| `GET` | `/admin/resolvers` | Daftar `resolver_key` yang tersedia di kode |

### 9.1 `GET /admin/resolvers`

```json
{
  "success": true,
  "data": {
    "resolvers": [
      { "key": "token_period_usage", "label": "Pemakaian kuota token per siklus",
        "unit": "kuota", "requires_period": true },
      { "key": "ai_requests_in_period", "label": "Jumlah permintaan AI per siklus",
        "unit": "permintaan", "requires_period": true },
      { "key": "active_transactions", "label": "Transaksi aktif",
        "unit": "transaksi", "requires_period": false },
      { "key": "vision_scans_in_period", "label": "Pemindaian struk per siklus",
        "unit": "scan", "requires_period": true }
    ]
  }
}
```

Operator memilih `resolver_key` dari dropdown ini saat membuat metrik baru.

### 9.2 Validasi silang

Saat operator menyimpan metrik:

| Aturan | Pesan |
|---|---|
| `resolver_key` harus ada di registry | "Resolver tidak dikenal" |
| `is_enforced = true` wajib punya `resolver_key` | "Metrik enforced harus punya resolver" |
| `is_enforced = false` boleh tanpa resolver | — |
| Key unik | "Key metrik sudah dipakai" |

---

## 10. Alur lengkap: menambah metrik baru tanpa deploy

**Skenario:** operator ingin membatasi jumlah anggaran (`budgets_created`).

```
1. Operator buka /super-admin/metrics → "Tambah Metrik"
2. Isi:
     key          : budgets_created
     label        : "Anggaran / siklus"
     unit         : "anggaran"
     resolver_key : budgets_created     ← dari dropdown (harus sudah ada di kode)
     enforced     : true
3. Simpan → baris masuk metric_definitions
4. Operator buka editor plan → dropdown limit kini memuat "Anggaran / siklus"
5. Set nilai untuk Pro: 10
6. Gate otomatis aktif pada rute yang memakai limit itu (bila rute memanggilnya)
7. User melihat "Anggaran: 3 / 10" di halaman langganan
```

**Yang perlu deploy:** hanya bila `budgets_created` resolver **belum ada** di registry.

---

## 11. Skenario: mengubah semua tanpa deploy

| Perubahan | Cara | Deploy? |
|---|---|---|
| Kuota plan | Editor plan | ❌ |
| Matriks fitur | Editor plan | ❌ |
| Rasio token | Editor plan | ❌ |
| Harga jual token | Editor pricing | ❌ |
| Biaya provider | Editor pricing | ❌ |
| Default kuota plan baru | Editor settings | ❌ |
| Mode pembulatan | Editor settings | ❌ |
| Label satuan kuota | Editor settings | ❌ |
| Overhead estimasi | Editor settings | ❌ |
| Sentinel unlimited | Editor settings | ❌ |
| Metrik baru (resolver ada) | Editor metrics | ❌ |
| Metrik baru (resolver baru) | Registry + metrics | ✅ |
| Fitur baru | Editor features | ❌ |
| Validator setting baru | setting_definitions | ❌ |

---

## 12. Invalidasi cache: event-driven (keputusan P-13)

> **Keputusan user:** *"daripada TTL, mending gunakan SSE biar lebih terjamin, murah,
> dan tidak memberatkan server/resources."* ✅ Setuju — dan infrastrukturnya **sudah
> ada**.

### 12.1 Masalah dengan TTL

| Masalah | Dampak |
|---|---|
| Perubahan tidak langsung berlaku | Operator ubah rasio → menunggu s/d 30s baru terasa |
| Query berulang saat cache kedaluwarsa | Beban DB naik (apalagi saat traffic tinggi) |
| TTL pendek = beban; TTL panjang = tidak segar | Trade-off yang tidak perlu |
| Tiap instance punya TTL sendiri | Instance bisa berbeda pandangan sementara |

### 12.2 Solusi: invalidasi via SSE + Redis pub/sub

Infrastruktur **sudah tersedia** (dibangun untuk P07):

| Komponen | Lokasi | Fungsi |
|---|---|---|
| SSE Hub | `realtime/hub.go` | Stream per-user & admin, heartbeat, resume `Last-Event-ID` |
| Redis Pub/Sub | `realtime/publisher.go` | Fan-out lintas-instance |
| Buffer replay | `realtime/buffer.go` | Event yang terlewat (offline) tetap terkirim saat resume |
| Cache ber-versi | `infrastructure/redis/cache.go` | `BumpVersion(scope)` → invalidasi O(1) lintas-instance |
| Dispatcher klien | `frontend/lib/realtime-dispatcher.ts` | Event → invalidasi query TanStack |

**Pola yang dipakai** (sudah terbukti di `plan_updated`):

```
1. Operator menyimpan perubahan (setting/metrik/fitur/plan)
        ↓
2. Handler admin, dalam TRANSAKSI yang sama:
   ├─ tulis perubahan ke DB
   ├─ CacheEngine.BumpVersion(ctx, "cfg")        ← invalidasi cache server
   └─ Publisher.PublishUser/Publish(...)         ← SSE ke klien
        ↓
3. Redis mem-fan-out ke SEMUA instance
        ↓
4. Tiap instance:
   ├─ penerima pub/sub men-drop cache lokal
   └─ klien menerima event → dispatcher invalidasi query
        ↓
5. Perubahan berlaku SEGERA (≈ milidetik), bukan menunggu TTL
```

### 12.3 Implementasi konkret

**Server — muat katalog ber-versi (bukan TTL):**

```go
// admin/config_cache.go — BARU
//
// ConfigCache menyimpan katalog (setting/metric/feature definitions) dengan
// validasi versi. Tidak ada TTL: cache hanya di-drop saat versi berubah, dan
// versi berubah HANYA saat operator menyimpan. Ini yang membuat perubahan
// langsung berlaku tanpa polling atau kedaluwarsa waktu.
type ConfigCache struct {
    cache *infraRedis.CacheEngine
    load  func(ctx context.Context) (map[string]SettingDefinition, error)
    mu    sync.RWMutex
    defs  map[string]SettingDefinition
    ver   int64
}

// Get mengembalikan katalog, memuat ulang hanya bila versi berubah.
func (c *ConfigCache) Get(ctx context.Context) (map[string]SettingDefinition, error) {
    ver := c.cache.GetVersion(ctx, "cfg")   // 0 bila Redis mati
    c.mu.RLock()
    if c.defs != nil && c.ver == ver {
        d := c.defs
        c.mu.RUnlock()
        return d, nil
    }
    c.mu.RUnlock()
    return c.reload(ctx, ver)
}

// Invalidate dipanggil SETELAH perubahan tersimpan. BumpVersion membuat semua
// instance melihat versi baru pada pembacaan berikutnya (O(1), tanpa scan key).
func (c *ConfigCache) Invalidate(ctx context.Context) {
    c.cache.BumpVersion(ctx, "cfg")
}
```

**Server — publish event setelah simpan:**

```go
// admin/settings.go / plans.go / catalog.go — SESUDAH
func (h *Handler) UpdateSettings(c *gin.Context) {
    // ... validasi + tulis DB (transaksi) ...

    // 1. Invalidasi cache server (lintas-instance, O(1))
    h.configCache.Invalidate(ctx)

    // 2. Beri tahu klien (SSE) agar UI segera memuat ulang
    h.publisher.Publish(ctx, contracts.NewEvent(
        contracts.EventConfigChanged, time.Now().Unix(),
        contracts.ConfigChangedPayload{Scope: "settings", Keys: changedKeys},
    ))
}
```

**Kontrak event baru:**

```go
// pkg/contracts/events.go
const EventConfigChanged = "config_changed"

// ConfigChangedPayload memberi tahu klien APA yang berubah sehingga UI dapat
// memuat ulang hanya bagian terkait (bukan seluruh cache — hindari cache storm).
type ConfigChangedPayload struct {
    // Scope: "settings" | "metrics" | "features" | "plans" | "pricing".
    Scope string `json:"scope"`
    // Keys: entri spesifik yang berubah (boleh kosong = seluruh scope).
    Keys []string `json:"keys,omitempty"`
}
```

**Klien — dispatcher:**

```ts
// frontend/lib/realtime-dispatcher.ts
case "config_changed":
  // Katalog berubah (setting/metrik/fitur). Muat ulang HANYA katalog terkait.
  // Entitlements ikut karena kuota/rasio bisa berubah dari sisi plan.
  return [
    queryKeys.admin.entitlementCatalog,
    queryKeys.admin.settingDefinitions,
    queryKeys.admin.metricDefinitions,
    queryKeys.admin.featureDefinitions,
    queryKeys.entitlements.me,
  ];
```

### 12.4 Mengapa ini lebih baik daripada TTL

| Aspek | TTL 30s | **SSE + Redis (P-13)** |
|---|---|---|
| Kesegaran | s/d 30 detik | **segera** (≈ milidetik) |
| Beban DB | query berulang tiap TTL habis | **1× saat berubah** |
| Konsistensi antar-instance | bisa berbeda sementara | **seragam** (versi bersama) |
| Skalabilitas | TTL pendek = beban naik | **tidak bergantung traffic** |
| Biaya resource | polling implisit | **push** — hemat |
| Offline client | menunggu TTL | **buffer replay** saat resume |

### 12.5 Jaring pengaman (WAJIB — tidak boleh hanya SSE)

SSE adalah **akselerator**, bukan sumber kebenaran. Bila SSE/Redis mati, sistem harus
tetap benar. Karena itu **tetap** diperlukan:

| Pengaman | Fungsi | Kenapa tetap ada |
|---|---|---|
| **Versi cache (`BumpVersion`)** | Invalidasi lintas-instance **tanpa Redis pub/sub** | Pub/sub bisa gagal; versi tetap terbaca saat request berikutnya |
| **Fallback in-memory** | `CacheEngine` jatuh ke map lokal bila Redis mati | Deployment single-instance tetap jalan |
| **Muat saat versi berubah** | Deteksi perubahan **tanpa event** | Bila event hilang, pembacaan berikutnya tetap melihat versi baru |
| **TTL panjang sebagai jaring** | Batas atas kesegaran | Jaring terakhir bila versi tak pernah ter-bump (mis. perubahan manual via SQL) |

> **Rekomendasi TTL jaring:** bukan 30 detik, melainkan **5 menit** — cukup sebagai
> jaring terakhir untuk perubahan di luar jalur API (mis. operator mengubah DB
> langsung), tanpa menjadi mekanisme utama. Beban query turun ~10× dibanding TTL 30s.

**Prinsip:** *event-driven untuk kesegaran, versi untuk konsistensi, TTL panjang
untuk jaring terakhir.* Tidak ada satu titik kegagalan.

### 12.6 Kasus khusus: perubahan di luar jalur API

Bila operator mengubah DB langsung (mis. `UPDATE settings SET ...` via psql):

- Tidak ada event SSE → klien tidak tahu segera.
- Versi tidak ter-bump → instance lain tidak tahu.
- **Jaring terakhir:** TTL 5 menit → perubahan terlihat dalam ≤ 5 menit.

Untuk kasus ini, sediakan juga endpoint admin:

```
POST /admin/config/reload   → BumpVersion + publish config_changed
```

Sehingga operator dapat memicu invalidasi manual setelah perubahan DB langsung.

---

## 13. Konsekuensi & risiko

| # | Konsekuensi | Mitigasi |
|---|---|---|
| K-1 | Konfigurasi bisa **rusak** oleh operator | Validasi dari `setting_definitions` + batas min/max |
| K-2 | Query DB tiap request | **Cache ber-versi**: 1× muat saat berubah (P-13), bukan TTL |
| K-3 | Perubahan tidak langsung berlaku | **Invalidasi event-driven** (SSE + Redis `BumpVersion`) |
| K-4 | Metrik bisa disalahkonfigurasi (resolver salah) | Validasi silang §9.2 + peringatan |
| K-5 | Menghapus definisi memutus data | Gunakan `is_active = false`, jangan DELETE |
| K-6 | `floor` membuka celah spam | `quota_min_charge_units` |
| K-7 | Audit sulit bila terlalu dinamis | Audit log pada setiap perubahan definisi |

### 13.1 Fail-safe wajib

| Skenario | Perilaku |
|---|---|
| `metric_definitions` kosong | Gate **tidak** dipasang (fail-open untuk gate, tapi plan tanpa limit tetap ditolak middleware `limit_not_in_plan`) |
| `setting_definitions` kosong | Setting **tidak bisa** diubah (fail-closed) |
| Setting hilang saat runtime | Pakai `default_value` dari definisi, lalu fallback konstanta kode |
| Resolver tidak ditemukan | Usage = 0 + **log error** (jangan crash) |
| DB error saat baca katalog | Pakai cache terakhir; bila tidak ada, fallback default |

> **Prinsip fail-safe:** untuk **keamanan** → fail-closed; untuk **ketersediaan** →
> fallback ke default & cache. Kuota adalah kontrol biaya, jadi **fail-closed** saat
> ragu (lebih baik menolak daripada membocorkan biaya).

---

## 14. Migrasi dari hardcode ke data

### 14.1 Urutan aman

```
Fase A: Buat tabel + seed dari konstanta kode saat ini (nilai SAMA)
        → perilaku tidak berubah
Fase B: Ubah kode membaca DB, dengan fallback ke konstanta lama
        → aman bila tabel kosong
Fase C: Verifikasi nilai dari DB == nilai lama
Fase D: (Opsional) Hapus konstanta lama setelah yakin
```

### 14.2 Prinsip penting

> **Jangan hapus konstanta kode sekaligus.** Simpan sebagai **fallback** permanen.
> Alasan: bila tabel konfigurasi kosong/rusak, sistem tetap berjalan dengan default
> yang aman. Ini yang membedakan "dinamis" dengan "rapuh".

---

## 15. Keputusan yang sudah dikunci

| # | Pertanyaan | **KEPUTUSAN USER** |
|---|---|---|
| **P-12** | Konstanta lama | ✅ **Simpan sebagai fallback** — tabel kosong tidak mematikan sistem |
| **P-13** | Kesegaran konfigurasi | ✅ **Invalidasi event-driven via SSE + Redis pub/sub** (§12) — bukan TTL |
| **P-14** | Resolver baru | ✅ **Ya** — seam diakui (kemampuan di kode, kebijakan di DB) |
| **P-15** | `quota_min_charge_units` | ✅ **Ada** (default 1) — cegah celah `floor` |
| **P-16** | Definisi dihapus | ✅ **Soft delete** (`is_active = false`) |

---

## 16. Ringkasan perubahan untuk "semuanya dinamis"

| Hardcode | Menjadi | Tabel/Setting |
|---|---|---|
| H-1 `writableSettings` | `setting_definitions` | tabel baru |
| H-2 `UsageFor()` switch | `metric_definitions.resolver_key` | tabel baru |
| H-3 `FeatureCatalog`/`LimitCatalog` | `feature_definitions`/`metric_definitions` | tabel baru |
| H-4 `ceilDiv` tetap | `quota_rounding_mode` | setting |
| H-5 `chatOverheadTokens`/`charsPerToken` | `ai_estimate_*` | setting |
| H-7 sentinel `1e9` | `quota_unlimited_sentinel` | setting |
| — | `quota_default_ratio` | setting (P-2) |
| — | `quota_default_tokens_per_cycle` | setting (P-9) |
| — | `quota_unit_label` | setting (P-3) |
| — | `pricing_default_provider_cost_per_1k_idr` | setting (P-11) |
| — | **Invalidasi cache** | **SSE `config_changed` + `BumpVersion`** (P-13) |
| — | `quota_min_charge_units` | setting (P-15) |

**Hasil:** seluruh keputusan, logic, flow, kalkulasi, dan metrik dapat diubah dari
Super Admin tanpa deploy.

---

**Navigasi:** [← Risiko & Rollback](./08-risiko-dan-rollback.md) · [Index](./README.md)
