# 03 — Model Domain & Skema Database

**Status:** DRAFT  
**Prasyarat:** keputusan D1–D9 di [`README.md`](./README.md) disetujui.

---

## 1. Prinsip Skema

1. **IDR = base accounting currency.** Semua nilai dasar disimpan dalam IDR.
2. **Nilai ganda.** Transaksi menyimpan nominal asli (mata uang pengguna) + snapshot kurs + nilai dasar IDR.
3. **Kurs append-only.** Riwayat kurs tidak diubah/dihapus; koreksi = baris baru.
4. **Katalog dari DB, bukan kode.** `currencies` menyimpan metadata format; `currency_rates` menyimpan kurs.
5. **Migrasi expand → backfill → contract.** Tidak mengedit migrasi lama. Nomor berikutnya: **`0039`** (migrasi terakhir saat ini `0038_dunning_subscription_period.sql`).
6. **Zona IDR tidak disentuh.** `budgets.amount_idr` (bigint), `invoices`, `payments`, `token_ledger`, harga paket, dan metrik admin tetap apa adanya.

---

## 2. ERD Ringkas

```
users (1) ─── (N) transactions
  │ display_currency ─┐
  │                   │ currency_code (FK, default 'IDR')
  │                   │ exchange_rate  (snapshot ke IDR)
  │                   │ amount_base    (IDR, = amount x exchange_rate)
  │
currencies (1) ─── (N) currency_rates        (append-only, effective_at)
      │
      └── katalog metadata: symbol, locale, decimal_digits,
          is_active, sort_order, is_base

users (1) ─── (N) transactions        (semua query agregat pakai amount_base)
users (1) ─── (N) budgets             (amount_idr — TIDAK berubah)
```

`currencies.code` adalah natural key (ISO 4217) yang dipakai sebagai FK di `transactions.currency_code` dan `users.display_currency`. FK natural key dipilih agar API dan ekspor tidak perlu join hanya untuk tahu kode.

---

## 3. Tabel Baru

### 3.1 `currencies` — katalog

```sql
CREATE TABLE IF NOT EXISTS currencies (
    code            varchar(3)   PRIMARY KEY,
    name            varchar(64)  NOT NULL,
    symbol          varchar(8)   NOT NULL,
    locale          varchar(16)  NOT NULL,
    decimal_digits  smallint     NOT NULL DEFAULT 0 CHECK (decimal_digits BETWEEN 0 AND 4),
    is_active       boolean      NOT NULL DEFAULT false,
    is_base         boolean      NOT NULL DEFAULT false,
    sort_order      integer      NOT NULL DEFAULT 0,
    -- Sumber kebijakan kurs: 'manual' sekarang; 'provider' disiapkan nanti.
    rate_source     varchar(16)  NOT NULL DEFAULT 'manual'
        CHECK (rate_source IN ('manual','provider')),
    created_at      timestamptz  NOT NULL DEFAULT now(),
    updated_at      timestamptz  NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER on_currency_updated
    BEFORE UPDATE ON currencies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Hanya satu base currency.
CREATE UNIQUE INDEX IF NOT EXISTS uq_currencies_base
    ON currencies ((is_base)) WHERE is_base;

CREATE INDEX IF NOT EXISTS idx_currencies_active
    ON currencies (is_active, sort_order);
```

Catatan:
- `decimal_digits` menggantikan asumsi "selalu 0 desimal" pada `formatRupiah`. JPY = 0, IDR = 0, USD = 2.
- `symbol` dan `locale` dipakai frontend lewat API, bukan konstanta kode.
- `is_active = false` berarti mata uang **tidak muncul** di picker pengguna dan tidak boleh dipakai transaksi baru. JPY di-seed nonaktif sampai Super Admin mengisi kurs.

### 3.2 `currency_rates` — riwayat kurs (append-only)

```sql
CREATE TABLE IF NOT EXISTS currency_rates (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    currency_code  varchar(3)   NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    -- 1 unit currency_code = rate_to_base unit base (IDR).
    rate_to_base   numeric(20,10) NOT NULL CHECK (rate_to_base > 0),
    effective_at   timestamptz  NOT NULL DEFAULT now(),
    source         varchar(16)  NOT NULL DEFAULT 'manual'
        CHECK (source IN ('manual','provider')),
    note           text,
    created_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz  NOT NULL DEFAULT now(),
    -- Idempoten & anti-duplikat: satu baris per (mata uang, waktu efektif).
    UNIQUE (currency_code, effective_at)
);

-- Query utama: kurs terbaru yang effective_at <= T, per mata uang.
CREATE INDEX IF NOT EXISTS idx_currency_rates_lookup
    ON currency_rates (currency_code, effective_at DESC);
```

Invariant:
- **Tidak ada UPDATE/DELETE.** Koreksi = INSERT baris baru. (Ditegakkan di layer usecase + review kode; opsional trigger penolak.)
- `rate_to_base` untuk base currency selalu `1`. Baris rate untuk IDR tidak wajib, tapi boleh ada sebagai jejak audit.
- `effective_at` boleh di masa depan (kurs terjadwal). Konversi hanya memakai baris `effective_at <= acuan`.

### 3.3 Kolom baru pada `users`

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS display_currency varchar(3)
        NOT NULL DEFAULT 'IDR' REFERENCES currencies(code) ON DELETE RESTRICT;
```

- Default `IDR` menjaga kompatibilitas dan backfill aman.
- FK ke `currencies` mencegah kode mata uang tak dikenal.
- Divalidasi agar hanya mata uang `is_active = true` (validasi di usecase saat update preferensi).

### 3.4 Kolom baru pada `transactions`

```sql
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS currency_code varchar(3)
        NOT NULL DEFAULT 'IDR' REFERENCES currencies(code) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS exchange_rate numeric(20,10)
        NOT NULL DEFAULT 1 CHECK (exchange_rate > 0),
    ADD COLUMN IF NOT EXISTS amount_base   numeric(20,2)
        NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rate_at       timestamptz;
```

- `amount` **tetap** menyimpan nominal asli (semantik berubah, tipe tidak berubah: `numeric(15,2)`). Nominal JPY 0 desimal tetap muat.
- `amount_base` = `amount × exchange_rate`, dibulatkan 2 desimal (IDR).
- `rate_at` = `effective_at` kurs yang dipakai saat transaksi dibuat.
- Untuk transaksi IDR: `currency_code='IDR'`, `exchange_rate=1`, `amount_base=amount`, `rate_at=NULL` (opsional).

**Catatan presisi:** `amount_base numeric(20,2)` mengakomodasi akumulasi besar. `exchange_rate numeric(20,10)` cukup untuk JPY→IDR (mis. `110.0000000000`) maupun IDR→USD terbalik.

### 3.5 Invariant & constraint tambahan

```sql
-- amount_base harus konsisten dengan amount x exchange_rate (toleransi 1 sen).
ALTER TABLE transactions
    ADD CONSTRAINT chk_transactions_amount_base
    CHECK (amount_base = round(amount * exchange_rate, 2));

-- rate_at wajib terisi bila bukan base currency.
ALTER TABLE transactions
    ADD CONSTRAINT chk_transactions_rate_at
    CHECK (currency_code = 'IDR' OR rate_at IS NOT NULL);
```

> **Peringatan implementasi:** constraint `chk_transactions_amount_base` berpotensi melambat pada tabel besar dan menolak pembulatan yang sudah dinormalisasi Go. Alternatif: tanpa CHECK, andalkan usecase + test. Rekomendasi rilis pertama: pasang CHECK (kebijakan `NOT VALID` agar tidak memblokir migrasi, lalu `VALIDATE` setelah backfill).

Indeks pendukung agregat:

```sql
-- Semua agregat baru memfilter/ menjumlah amount_base.
CREATE INDEX IF NOT EXISTS idx_transactions_user_base_date
    ON transactions (user_id, transaction_date)
    WHERE is_deleted = false;
```

---

## 4. Backfill

### 4.1 Transaksi lama

```sql
UPDATE transactions
SET currency_code = 'IDR',
    exchange_rate = 1,
    amount_base   = amount,
    rate_at       = NULL
WHERE amount_base = 0 OR currency_code IS NULL;
```

Sifat: idempoten. Setelah dijalankan, kunci invariant `amount_base = amount`.

Verifikasi (harus 0 baris):

```sql
SELECT COUNT(*) FROM transactions
WHERE is_deleted = false AND amount_base <> amount AND currency_code = 'IDR';
```

### 4.2 Pengguna lama

`display_currency` sudah default `IDR`; tidak ada backfill khusus.

---

## 5. Seed Katalog

Seed **IDR** (base, aktif) dan **JPY** (nonaktif sampai Super Admin mengisi kurs). Jangan mengarang kurs produksi.

```sql
INSERT INTO currencies
    (code, name, symbol, locale, decimal_digits, is_active, is_base, sort_order, rate_source)
VALUES
    ('IDR', 'Rupiah Indonesia', 'Rp', 'id-ID', 0, true,  true,  10, 'manual'),
    ('JPY', 'Yen Jepang',       '¥',  'ja-JP', 0, false, false, 20, 'manual')
ON CONFLICT (code) DO NOTHING;

-- Kurs base sendiri = 1 (jejak eksplisit; opsional).
INSERT INTO currency_rates (currency_code, rate_to_base, effective_at, source, note)
VALUES ('IDR', 1, now(), 'manual', 'Base currency')
ON CONFLICT (currency_code, effective_at) DO NOTHING;
```

JPY baru dapat diaktifkan setelah ada baris `currency_rates` untuk JPY. Aturan aktivasi ada di usecase Admin (dokumen 06), bukan hanya DB.

---

## 6. Urutan Migrasi

| File | Tujuan | Sifat |
|---|---|---|
| `0039_currency_catalog.sql` | Buat `currencies`, `currency_rates`, index, seed IDR/JPY | Additive |
| `0040_currency_columns.sql` | Tambah `users.display_currency`, kolom `transactions.*` | Expand (kolom NOT NULL + DEFAULT, aman) |
| `0041_currency_backfill.sql` | Backfill transaksi lama ke IDR | Data |
| `0042_currency_constraints.sql` | `CHECK` amount_base & rate_at (NOT VALID → VALIDATE), index agregat | Contract (lunak) |

Semua idempoten (`IF NOT EXISTS`, `ON CONFLICT DO NOTHING`), mengikuti pola `0029_config_catalog.sql`.

**Aturan zona IDR:** bila `0042` dianggap terlalu berisiko pada data produksi besar, blue-green / batch backfill dengan `WHERE amount_base = 0` per 10k baris. Constraint tetap `NOT VALID` sampai backfill tuntas.

---

## 7. Query Agregat yang Berubah

Semua `SUM(amount)` pada permukaan pengguna → `SUM(amount_base)`. Titik di `backend/internal/repository/postgres/transaction_repo.go`:

| Method | Perubahan |
|---|---|
| `GetStats` | `SUM(CASE WHEN type=... THEN amount_base ELSE 0 END)` |
| `GetMonthlyExpense` | `SUM(amount_base)` |
| `GetCategoryExpensesByPeriod` | `SUM(t.amount_base)` |
| `GetCategoryIncomesByPeriod` | `SUM(t.amount_base)` |
| `GetCategorySavingsByPeriod` | `SUM(t.amount_base)` |
| `GetTimeSeries` | `SUM(... amount_base ...)` |
| `GetCategoryTimeSeries` | `SUM(t.amount_base)` |

`List` dan `GetByID` mengembalikan `amount` (asli) + kolom baru; penyajian mata uang ditangani frontend (dokumen 05). Sorting `amount` tetap nominal asli — namun bila ada transaksi campuran mata uang, urutan nominal asli tidak sebanding. Untuk rilis ini sorting tetap `amount`; opsi `amount_base` dicatat sebagai K-4 di dokumen 09.

**Query yang TIDAK berubah:** `CountActiveByUser`, `GetDailyTransactionCount`, `HasTransactions`, `budgets.amount_idr`, seluruh billing/invoice/token.

---

## 8. Dampak ke Tabel Lain

| Tabel | Dampak | Alasan |
|---|---|---|
| `budgets` | Tidak ada perubahan kolom | `amount_idr` adalah nilai dasar; input JPY dikonversi ke IDR di usecase |
| `invoices`, `payments`, `token_ledger` | Tidak ada | Zona IDR |
| `settings` / `setting_definitions` | Tidak ada baris baru wajib | Katalog mata uang punya tabel sendiri; bukan setting skalar |
| `transaction_embeddings` | Metadata bertambah `currency_code` (opsional) | Teks embedding menyebut mata uang (dokumen 04) |
| `audit_logs` | Tidak ada perubahan skema | Perubahan kurs dicatat lewat pola audit yang ada |

**Kenapa bukan `settings` JSON?** Mata uang adalah entitas berulang dengan riwayat (N baris per mata uang). Menaruh di `settings` JSON akan menghalangi FK, index, constraint, dan query "kurs terbaru ≤ T". Tabel relasional adalah bentuk paling sederhana yang benar di sini.

---

## 9. Diagram Alur Data Nilai

```
Input pengguna (JPY)                     Penyimpanan                    Tampilan
────────────────────                     ───────────                    ────────
amount  = 1000
currency_code = JPY
                                         transactions:
                                           amount        = 1000
                                           currency_code = JPY
                                           exchange_rate = 110.0 (snapshot)
                                           amount_base   = 110000
                                           rate_at       = 2026-10-19T00:00Z

Detail transaksi  ──────────────────────────────────────────► ¥1.000  (nominal asli)
Agregat dashboard  = SUM(amount_base) = 110000
                                          ↓ konversi kurs hari ini
                                       ¥1.000 (atau Rp110.000 jika pengguna IDR)
```

---

## 10. Rollback Skema

| Migrasi | Rollback |
|---|---|
| `0039` | `DROP TABLE currency_rates; DROP TABLE currencies;` (hanya jika kolom di 0040 belum ada) |
| `0040` | `ALTER TABLE ... DROP COLUMN ...` (aman; kolom default tidak dipakai kode lama) |
| `0041` | Tanpa rollback; data IDR sudah benar secara semantik |
| `0042` | `ALTER TABLE ... DROP CONSTRAINT ...` |

Karena 0040 bersifat additive dengan DEFAULT, deploy backend lama tetap jalan (kolom diabaikan). Ini kunci zero-downtime: **skema lebih dulu, kode menyusul**.

---

## 11. Ringkasan Keputusan Skema

| # | Keputusan |
|---|---|
| K-1 | Base currency = IDR, tidak dapat diubah lewat UI |
| K-2 | Natural key `currencies.code` + FK dari transaksi dan users |
| K-3 | `currency_rates` append-only dengan `effective_at` |
| K-4 | `transactions` menyimpan `amount`, `currency_code`, `exchange_rate`, `amount_base`, `rate_at` |
| K-5 | Agregat memakai `amount_base`; nominal asli hanya untuk tampilan individual |
| K-6 | Backfill `amount_base = amount` untuk data lama, idempoten |
| K-7 | `budgets.amount_idr` tidak berubah |
| K-8 | JPY di-seed nonaktif; aktivasi butuh kurs |
| K-9 | Migrasi 0039–0042, expand → backfill → constraint |
