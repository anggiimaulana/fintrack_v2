# 02 — Skema Database & Migrasi

[← Index](./README.md) · [← Desain Teknis](./01-desain-teknis.md)

---

## 1. Ringkasan perubahan skema

### 1.1 Migrasi `0028` — rasio kuota

| Tabel | Perubahan | Tipe | Aman rolling deploy? |
|---|---|---|---|
| `plans` | `+ token_quota_ratio bigint NOT NULL DEFAULT 1` | **Expand** | ✅ ya |
| `token_periods` | `+ quota_ratio bigint NOT NULL DEFAULT 1` | **Expand** | ✅ ya |
| `token_ledger` | `+ raw_tokens bigint NOT NULL DEFAULT 0` | **Expand** | ✅ ya |

### 1.2 Migrasi `0029` — katalog konfigurasi dinamis

| Tabel | Isi | Tipe | Aman? |
|---|---|---|---|
| `setting_definitions` | Katalog setting (kind, min, max, enum, default, label) | **Tabel baru** | ✅ ya |
| `metric_definitions` | Katalog metrik/limit (label, unit, resolver, enforced) | **Tabel baru** | ✅ ya |
| `feature_definitions` | Katalog fitur (label, deskripsi) | **Tabel baru** | ✅ ya |

> **Tabel baru = paling aman.** Tidak ada konsumen lama, tidak ada locking pada tabel
> yang dipakai. Detail lengkap: [09 §4–5](./09-lapisan-konfigurasi-dinamis.md).

**Tidak ada** `DROP`, `RENAME`, `SET NOT NULL` tanpa default, atau perubahan
constraint yang membahayakan instance lama. Sesuai `database/migrations/README.md`.

Semua kolom memakai `NOT NULL DEFAULT <literal>` sehingga:
- instance **lama** yang belum tahu kolom ini tetap bisa INSERT (default mengisi),
- instance **baru** membaca nilai yang masuk akal,
- tidak ada locking panjang (PostgreSQL 11+ mengisi default tanpa rewrite tabel).

---

## 2. Migrasi `0028_plan_token_quota_ratio.sql`

> **Nama file:** `database/migrations/0028_plan_token_quota_ratio.sql`
> **Fase:** Expand (satu perubahan jelas: menambah rasio kuota)

```sql
-- ============================================================
-- 0028_plan_token_quota_ratio.sql
--
-- Rasio konversi token provider → unit kuota plan.
--
-- Latar: kuota plan (plan_limits.ai_tokens_per_cycle) diukur dalam "unit",
-- sedangkan provider melaporkan token mentah. Satu percakapan nyata memakai
-- ~5.000–10.000 token, sehingga kuota Free 5.000 habis dalam satu chat.
--
-- Rasio memisahkan dua besaran:
--   * raw_tokens  → dasar BIAYA (Rupiah) & margin
--   * unit kuota  → raw_tokens / rasio, dibandingkan dengan kuota plan
--
-- Contoh (rasio Free = 5): chat 9.945 token menagih ceil(9945/5) = 1.989 unit,
-- sehingga kuota 5.000 masih menyisakan 3.011 — user bisa ~2–3 chat.
--
-- Default 1 = perilaku lama (1 token = 1 unit), jadi deploy ini TIDAK
-- mengubah akuntansi yang sedang berjalan.
--
-- Additive & idempotent (expand only; tidak ada DROP/RENAME/SET NOT NULL).
-- ============================================================

-- 1) Rasio per plan. Dipakai saat siklus kuota baru dibuat.
ALTER TABLE plans
    ADD COLUMN IF NOT EXISTS token_quota_ratio bigint NOT NULL DEFAULT 1;

-- Rasio harus >= 1 (1 = 1:1). Nilai < 1 berarti menagih LEBIH banyak unit
-- daripada token nyata, yang tidak masuk akal dan berisiko menyesatkan user.
ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_token_quota_ratio_check;
ALTER TABLE plans
    ADD CONSTRAINT plans_token_quota_ratio_check
    CHECK (token_quota_ratio >= 1);

-- 2) Snapshot rasio pada siklus kuota. Kolom ini IMMUTABLE: begitu siklus
--    berjalan, syaratnya (kuota + rasio) beku sampai siklus berakhir, sehingga
--    mengubah rasio plan tidak mengubah akuntansi siklus yang sedang berjalan.
ALTER TABLE token_periods
    ADD COLUMN IF NOT EXISTS quota_ratio bigint NOT NULL DEFAULT 1;

ALTER TABLE token_periods DROP CONSTRAINT IF EXISTS token_periods_quota_ratio_check;
ALTER TABLE token_periods
    ADD CONSTRAINT token_periods_quota_ratio_check
    CHECK (quota_ratio >= 1);

-- 3) Token mentah pada ledger. `amount` kini menyimpan UNIT kuota; `raw_tokens`
--    menyimpan token provider yang menjadi dasar biaya. Baris lama TIDAK
--    di-backfill (append-only) — dibaca via helper RawTokensOrAmount().
ALTER TABLE token_ledger
    ADD COLUMN IF NOT EXISTS raw_tokens bigint NOT NULL DEFAULT 0;

-- TIDAK ADA BACKFILL (keputusan P-6, Pilihan B).
-- token_ledger bersifat append-only (trigger token_ledger_no_update menolak
-- semua UPDATE). Baris lama dibiarkan raw_tokens = 0 dan dibaca lewat helper
-- RawTokensOrAmount() di kode — nilai unit-nya identik karena rasio saat itu 1.
-- Query agregat memakai GREATEST(raw_tokens, ABS(amount)).

-- Rasio tidak boleh 0 di ledger (0 = tidak ada konversi yang bermakna).
ALTER TABLE token_ledger DROP CONSTRAINT IF EXISTS token_ledger_raw_tokens_check;
ALTER TABLE token_ledger
    ADD CONSTRAINT token_ledger_raw_tokens_check
    CHECK (raw_tokens >= 0);

-- down:
--   ALTER TABLE token_ledger DROP CONSTRAINT IF EXISTS token_ledger_raw_tokens_check;
--   ALTER TABLE token_ledger DROP COLUMN IF EXISTS raw_tokens;
--   ALTER TABLE token_periods DROP CONSTRAINT IF EXISTS token_periods_quota_ratio_check;
--   ALTER TABLE token_periods DROP COLUMN IF EXISTS quota_ratio;
--   ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_token_quota_ratio_check;
--   ALTER TABLE plans DROP COLUMN IF EXISTS token_quota_ratio;
```

### 2.1 Catatan desain migrasi

| Pertanyaan | Jawaban |
|---|---|
| Kenapa `NOT NULL DEFAULT 1` bukan nullable? | Default literal aman untuk instance lama (PG 11+ tanpa rewrite) dan menghindari NULL di aritmetika |
| Kenapa TIDAK ada backfill `raw_tokens`? | `token_ledger` append-only; baris lama dibaca via `RawTokensOrAmount()` (keputusan P-6) |
| Kenapa ada `CHECK >= 1` di `plans` tapi `>= 0` di ledger? | `plans.token_quota_ratio` adalah pembagi (harus ≥1); `raw_tokens` adalah besaran (boleh 0 untuk release) |
| Perlu indeks baru? | **Tidak.** Tidak ada query baru yang memfilter kolom ini |
| Perlu `CONCURRENTLY`? | **Tidak.** Hanya `ADD COLUMN` + `CHECK` pada tabel kecil–sedang; `CHECK` divalidasi sekali. Jika `token_ledger` sudah besar (>10 juta baris), pertimbangkan `NOT VALID` lalu `VALIDATE CONSTRAINT` terpisah |

### 2.2 Jika `token_ledger` sudah besar

Untuk aman pada tabel besar, pisahkan validasi constraint:

```sql
-- Alternatif bila token_ledger > ~10 juta baris:
ALTER TABLE token_ledger
    ADD CONSTRAINT token_ledger_raw_tokens_check CHECK (raw_tokens >= 0) NOT VALID;
-- lalu di migrasi berikutnya (setelah traffic rendah):
ALTER TABLE token_ledger VALIDATE CONSTRAINT token_ledger_raw_tokens_check;
```

---

## 3. Data awal rasio — DITIADAKAN (keputusan P-8)

> **Keputusan user:** *"serahkan ke operator"*. **Tidak ada** migrasi seed rasio.

Alasan: nilai rasio adalah **kebijakan bisnis**, bukan skema. Menyuntiknya lewat
migrasi berarti:

- nilai ikut ter-reset bila migrasi dijalankan ulang di lingkungan lain,
- keputusan operator bisa tertimpa,
- riwayat perubahan tidak tercatat di audit log.

**Sebagai gantinya:**

| Mekanisme | Perilaku |
|---|---|
| `plans.token_quota_ratio DEFAULT 1` | Plan lama & baru mulai dari 1 (perilaku identik) |
| Setting `quota_default_ratio` | Rasio awal untuk **plan baru** yang dibuat operator |
| Editor plan | Operator menetapkan rasio per plan kapan pun |

Sehingga setelah migrasi 0028 dijalankan:

1. Semua plan punya `token_quota_ratio = 1` → **perilaku tidak berubah**.
2. Operator membuka `/super-admin/plans` dan menetapkan rasio (mis. Free 5).
3. Perubahan berlaku pada **siklus berikutnya** (snapshot di `token_periods`).

> **Konsekuensi yang harus dikomunikasikan:** segera setelah deploy, sistem masih
> memakai rasio 1. Manfaat baru terasa setelah operator mengatur rasio. Ini
> disengaja — operator yang memutuskan, bukan kode.

---

## 4. Perubahan repository (SQL)

### 4.1 `GetOrCreatePeriod` — tulis rasio

```go
// repository/postgres/billing_repo.go — SESUDAH
func (r *BillingRepo) GetOrCreatePeriod(ctx context.Context, userID uuid.UUID,
    periodStart, periodEnd int64, quota int64, ratio int64) (*billing.TokenPeriod, error) {
    ps := time.Unix(periodStart, 0).UTC()
    pe := time.Unix(periodEnd, 0).UTC()
    _, err := q(ctx, r.pool).Exec(ctx,
        `INSERT INTO token_periods (user_id, period_start, period_end, quota_tokens, quota_ratio)
         VALUES ($1,$2,$3,$4,$5)
         ON CONFLICT (user_id, period_start) DO NOTHING`,
        userID, ps, pe, quota, ratio)
    if err != nil { return nil, err }
    return r.getPeriodByStart(ctx, userID, ps)
}
```

### 4.2 `getPeriodByStart` / `currentPeriod` — baca rasio

```sql
SELECT id, user_id, period_start, period_end, quota_tokens, quota_ratio,
       used_tokens, reserved_tokens, created_at, updated_at
  FROM token_periods
 WHERE user_id = $1 AND period_start = $2
```

### 4.3 `InsertLedger` — tulis `raw_tokens`

```go
`INSERT INTO token_ledger (id, period_id, user_id, reservation_id, entry_type,
                           amount, raw_tokens, modality, cost_idr, idempotency_key,
                           reference_id, notes)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING created_at`
```

### 4.4 `GetLedgerByID` / `GetLedgerByIdempotencyKey` / list admin — baca `raw_tokens`

Tambahkan `raw_tokens` ke semua SELECT yang mengembalikan `TokenLedgerEntry`.

### 4.5 Query agregasi admin (audit)

Tambahkan kolom token nyata pada daftar ledger admin:

```sql
-- admin/tokens.go — daftar ledger: tampilkan unit DAN token nyata
SELECT id, entry_type, amount AS units, raw_tokens, modality, cost_idr, created_at
  FROM token_ledger
 WHERE user_id = $1
 ORDER BY created_at DESC
 LIMIT $2
```

---

## 5. Verifikasi migrasi

### 5.1 Prosedur uji (wajib sesuai README migrasi)

```bash
# 1. Migrasi maju pada DB uji
psql "$TEST_DATABASE_URL" -f database/migrations/0028_plan_token_quota_ratio.sql

# 2. Jalankan ULANG untuk membuktikan idempoten (tidak boleh error)
psql "$TEST_DATABASE_URL" -f database/migrations/0028_plan_token_quota_ratio.sql

# 3. Verifikasi kolom ada dengan default benar
psql "$TEST_DATABASE_URL" -c "\d+ plans" | grep token_quota_ratio
psql "$TEST_DATABASE_URL" -c "\d+ token_periods" | grep quota_ratio
psql "$TEST_DATABASE_URL" -c "\d+ token_ledger" | grep raw_tokens

# 4. Verifikasi kolom baru (tanpa backfill)
psql "$TEST_DATABASE_URL" -c "
  SELECT count(*) AS kolom_ada
    FROM information_schema.columns
   WHERE table_name='token_ledger' AND column_name='raw_tokens'"
# harus 1
```

### 5.2 Checklist sebelum commit

- [ ] Idempoten (sudah diuji dengan menjalankan dua kali)
- [ ] Tidak ada `DROP COLUMN` / `RENAME` / `SET NOT NULL`
- [ ] `NOT NULL DEFAULT` literal (bukan ekspresi non-immutable)
- [ ] Backfill terverifikasi (0 baris tersisa)
- [ ] `database/dump/fintrack_v2.sql` di-regenerate
- [ ] Tidak mengedit migrasi yang sudah di-commit
- [ ] Sudah diuji di `fintrack_test`

### 5.3 Regenerasi dump

```bash
# Sesuai prosedur di database/README.md
pg_dump ... > database/dump/fintrack_v2.sql
```

---

## 6. Pertimbangan khusus

### 6.1 Mengapa snapshot di `token_periods`, bukan di ledger?

Rasio perlu dibaca **saat commit**, dan commit memakai `period` yang sudah di-lock.
Menyimpannya di `token_periods` berarti:
- satu pembacaan saja (sudah ada `GetPeriodForUpdate`),
- immutable selama siklus,
- release/refund otomatis memakai rasio yang sama (period yang sama).

Menyimpannya di `token_ledger` (per baris) akan lebih granular tetapi:
- perlu membaca baris reserve untuk tahu rasio saat commit,
- lebih banyak kolom redundan,
- tidak memberi manfaat karena rasio tidak berubah mid-cycle.

### 6.2 Bagaimana dengan period lama (sebelum migrasi)?

`quota_ratio` default `1` → perilaku lama. Tidak ada perubahan retroaktif. ✅

### 6.3 Apakah `used_tokens` perlu dikonversi ulang untuk period lama?

**Tidak.** Period lama sudah selesai (atau sedang berjalan dengan rasio 1). Mengubah
`used_tokens` akan merusak audit trail dan melanggar sifat append-only. Biarkan apa
adanya; rasio baru berlaku untuk period berikutnya.

### 6.4 Trigger `updated_at`

`token_periods` **tidak** memiliki trigger `on_*_updated` (hanya `plans` dan
`subscriptions` yang punya). `AdjustPeriod` menulis `updated_at = now()` secara
eksplisit — pola ini dipertahankan.

### 6.5 Append-only `token_ledger`

Migrasi **tidak** melakukan UPDATE sama sekali pada `token_ledger`. Ini **disengaja**
(keputusan P-6): trigger `token_ledger_no_update` memblokir semua UPDATE, dan
catatan keuangan harus tetap append-only.

```sql
-- Trigger yang ada (0008):
CREATE TRIGGER token_ledger_no_update
    BEFORE UPDATE ON token_ledger
    FOR EACH ROW EXECUTE FUNCTION token_ledger_block_mutation();
```

Karena trigger menolak semua UPDATE, migrasi **tidak** boleh menulis ulang baris lama.
Dua opsi yang dipertimbangkan:

```sql
-- PILIHAN A (DITOLAK oleh user): nonaktifkan trigger saat backfill, lalu aktifkan.
ALTER TABLE token_ledger DISABLE TRIGGER token_ledger_no_update;
UPDATE token_ledger SET raw_tokens = ABS(amount) WHERE raw_tokens = 0 AND amount <> 0;
ALTER TABLE token_ledger ENABLE TRIGGER token_ledger_no_update;
```

```sql
-- PILIHAN B (DIPILIH — keputusan P-6): biarkan baris lama raw_tokens = 0 dan
-- tangani di kode — pembacaan memperlakukan 0 sebagai "sama dengan amount".
-- Tidak ada UPDATE sama sekali. Trigger append-only tetap utuh.
```

> **KEPUTUSAN USER (P-6): PILIHAN B.** Alasan: tabel `token_ledger` adalah catatan
> keuangan append-only; menonaktifkan trigger — meski sementara — membuka celah dan
> menimbulkan preseden buruk. Sebagai gantinya, tambahkan helper pembacaan:

```go
// RawTokensOrAmount mengembalikan token nyata sebuah baris ledger. Baris lama
// (sebelum migrasi 0028) memiliki raw_tokens = 0 karena rasio saat itu 1, jadi
// nilai unit-nya identik dengan token nyata.
func (e *TokenLedgerEntry) RawTokensOrAmount() int64 {
    if e.RawTokens > 0 { return e.RawTokens }
    if e.Amount < 0 { return -e.Amount }
    return e.Amount
}
```

Dengan Pilihan B, migrasi menjadi lebih sederhana:

```sql
-- Ganti blok UPDATE pada 0025 dengan:
-- (tidak ada UPDATE — baris lama dibaca via RawTokensOrAmount())
```

**Konsekuensi:** `raw_tokens` baris lama tetap 0. Query agregasi admin yang
menjumlahkan token nyata **harus** memakai `GREATEST(raw_tokens, ABS(amount))`:

```sql
SELECT COALESCE(SUM(GREATEST(raw_tokens, ABS(amount))), 0) AS raw_total
  FROM token_ledger
 WHERE entry_type = 'commit' AND period_id = $1
```

> **Pilihan A ditolak** (keputusan user): menonaktifkan trigger append-only dianggap
> terlalu berisiko untuk catatan keuangan.

---

## 7. Keputusan yang sudah dikunci

| # | Pertanyaan | **Keputusan user** |
|---|---|---|
| **P-6** | Backfill `raw_tokens` baris lama | **Pilihan B** — biarkan `0` + helper `RawTokensOrAmount()`; trigger append-only utuh |
| **P-7** | Constraint `CHECK` pada tabel besar | **`NOT VALID` bila > 10 juta baris** (lihat §2.2) |
| **P-8** | Migrasi seed rasio | **Serahkan ke operator** — tidak ada migrasi seed (§3) |

---

## 8. Perubahan pada file dump

Setelah migrasi, `database/dump/fintrack_v2.sql` **wajib** di-regenerate agar:
- struktur `plans`/`token_periods`/`token_ledger` terbaru tercatat,
- data seed (`0002_subscription_token_seeds.sql`) sinkron,
- developer baru mendapat skema yang benar.

Pastikan **tidak** ada secret yang ikut (sesuai `.cursorrules` & `.windsurfrules`).

---

**Navigasi:** [← Desain Teknis](./01-desain-teknis.md) · [Spesifikasi API →](./03-spesifikasi-api.md)
