# 01 — Desain Teknis

[← Index](./README.md) · [← Ringkasan](./00-ringkasan-eksekutif.md)

---

## 1. Prinsip desain

| # | Prinsip | Alasan |
|---|---|---|
| D-1 | **Kuota plan = satuan abstrak ("unit")**, bukan token mentah | Memisahkan yang dilihat user dari biaya nyata |
| D-2 | **Biaya selalu dari token nyata** | Margin harus akurat & tidak bisa "disembunyikan" rasio |
| D-3 | **Rasio di-snapshot per siklus** (`token_periods.quota_ratio`) | Mengubah rasio mid-cycle tidak boleh mengubah akuntansi berjalan |
| D-4 | **Pembulatan `ceil`** (ke atas) | Jangan pernah menagih kurang dari biaya nyata |
| D-5 | **Rasio default `1`** | Backward compatible: perilaku lama identik |
| D-6 | **Ledger tetap append-only & idempoten** | ADR-002; tidak ada UPDATE/DELETE |
| D-7 | **Uang tetap integer Rupiah** | ADR-003; tidak ada float untuk uang |

---

## 2. Dua besaran yang dipisahkan

```
                    ┌─────────────────────────────────┐
                    │  Provider: total_tokens = 9.945 │
                    └────────────────┬────────────────┘
                                     │
              ┌──────────────────────┴──────────────────────┐
              │                                             │
   ┌──────────▼───────────┐                    ┌────────────▼─────────────┐
   │  RAW (token nyata)   │                    │  UNIT (kuota plan)       │
   │  raw_tokens = 9.945  │                    │  units = ceil(9945 / 5)  │
   │                      │                    │        = 1.989           │
   │  → PricingEngine     │                    │  → AdjustPeriod          │
   │  → cost_idr          │                    │  → used_tokens           │
   │  → margin            │                    │  → dibanding kuota       │
   └──────────────────────┘                    └──────────────────────────┘
```

**Invarian:** `raw_tokens` dan `amount` selalu tercatat berdampingan di setiap baris
`token_ledger` yang relevan, sehingga biaya dan kuota bisa direkonsiliasi kapan pun.

---

## 3. Alur metering (sesudah perubahan)

### 3.1 Ringkasan alur

```
1. GATE (middleware, pre-flight)
   ├─ ai_requests_per_cycle  → hitung baris reserve
   └─ ai_tokens_per_cycle    → hitung (used + reserved)  [SATUAN UNIT]
        ↓ lolos
2. RESERVE (handler → ledger, dalam tx + row lock)
   ├─ estimasi RAW = EstimateStore.ForChat(ctx, modality, len(message))
   ├─ estimasi UNIT = RoundUnits(estimasiRAW, period.QuotaRatio, mode)
   ├─ cek period.Available() >= estimasiUNIT
   └─ AdjustPeriod(used += 0, reserved += estimasiUNIT)
        ↓
3. PANGGIL AI (jaringan, TANPA lock)
   └─ provider lapor usage.TotalTokens (RAW)
        ↓
4. COMMIT (handler defer → ledger, tx + row lock)
   ├─ units = RoundUnits(rawTokens, period.QuotaRatio, mode)
   ├─ clamp: units = min(units, quota - used)
   ├─ AdjustPeriod(used += units, reserved -= reserve.Amount)
   └─ InsertLedger(amount=units, raw_tokens=raw, cost_idr=Price(raw))
```

### 3.2 Perubahan pada `Reserve`

```go
// billing/ledger.go — SESUDAH (konseptual)
func (s *LedgerService) Reserve(ctx, userID, estUnits int64, modality, idempotencyKey) (*Reservation, error) {
    // estUnits SUDAH dalam satuan unit plan (handler yang mengonversi,
    // karena handler yang tahu rasio & panjang pesan).
    ...
    if period.Available() < estUnits { return ErrInsufficientQuota }
    AdjustPeriod(0, +estUnits)
    ...
}
```

> **Perubahan kontrak:** parameter `estTokens` menjadi `estUnits`. Nama fungsi
> `Reserve` dipertahankan; makna satuannya berubah. Ini titik paling sensitif dan
> **wajib** disertai komentar + rename variabel agar tidak salah pakai.

### 3.3 Perubahan pada `Commit`

```go
// billing/ledger.go — SESUDAH (konseptual)
func (s *LedgerService) Commit(ctx, userID, reservationID, rawTokens int64, idempotencyKey) error {
    ...
    return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
        period, _ := s.tokens.GetPeriodForUpdate(txCtx, userID, s.now().Unix())
        ratio := period.QuotaRatio            // snapshot siklus
        if ratio < 1 { ratio = 1 }            // fail-safe

        units := RoundUnits(rawTokens, ratio, mode)   // mode dari setting

        // clamp (tetap ada, sekarang dalam satuan unit)
        if maxCharge := period.QuotaTokens - period.UsedTokens; units > maxCharge {
            logger.Warn("token commit exceeds remaining quota; clamping", ...)
            if maxCharge < 0 { maxCharge = 0 }
            units = maxCharge
        }
        AdjustPeriod(units, -reserve.Amount)
        InsertLedger(TokenLedgerEntry{
            Amount:    units,
            RawTokens: rawTokens,             // ← KOLOM BARU
            CostIDR:   s.pricing.CostIDR(rawTokens, modality),  // ← dari RAW
            ...
        })
    })
}
```

**Perhatikan:** `CostIDR` memakai `rawTokens`, **bukan** `units`. Inilah inti K-1.

### 3.4 Perubahan pada `Release`

Tidak ada konversi — `Release` hanya mengembalikan `reserve.Amount` yang sudah dalam
unit:

```go
AdjustPeriod(0, -reserve.Amount)   // sudah unit, tidak berubah
InsertLedger(TokenLedgerEntry{EntryType: EntryRelease, Amount: -reserve.Amount, RawTokens: 0})
```

> `raw_tokens = 0` karena release tidak menagih biaya apa pun.

### 3.5 Perubahan pada `Refund`

```go
// SESUDAH: Refund menerima RAW, mengonversi internal (simetris dengan Commit)
func (s *LedgerService) Refund(ctx, userID, rawTokens int64, modality, idempotencyKey) error {
    ...
    units := RoundUnits(rawTokens, period.QuotaRatio, mode)
    if units > period.UsedTokens { units = period.UsedTokens }  // clamp (tetap)
    AdjustPeriod(-units, 0)
    InsertLedger(EntryRefund, Amount: -units, RawTokens: rawTokens)
}
```

> **Temuan audit:** `Refund` saat ini **belum dipanggil di jalur produksi** — hanya
> ada di test (`ledger_test.go`) dan endpoint refund pembayaran yang berbeda
> (`admin/payments.go:274`). Karena itu risiko regresinya rendah, tetapi tetap
> **wajib** dikonversi agar konsisten saat nanti dipakai (mis. OCR yang gagal setelah
> commit).

---

## 4. Titik konversi: di mana rasio dibaca?

### 4.1 Keputusan: snapshot di `token_periods.quota_ratio`

| Opsi | Kelebihan | Kekurangan | Dipilih? |
|---|---|---|---|
| A. Baca rasio dari `plans` setiap saat | Selalu terkini | Ubah rasio mid-cycle → akuntansi lama ikut berubah (retroaktif!) | ❌ |
| B. **Snapshot ke `token_periods.quota_ratio`** saat period dibuat | Immutable per siklus; audit bersih | Perlu kolom + backfill | ✅ |
| C. Simpan di reservation | Presisi | Kompleks; release/refund perlu ikut | ❌ |

**Alasan B:** siklus kuota adalah "kontrak" dengan user. Begitu siklus berjalan,
syaratnya (kuota + rasio) harus beku sampai siklus berakhir. Perubahan rasio berlaku
pada siklus **berikutnya**. Ini mencegah user "rugi" atau "untung" mendadak di tengah
siklus.

### 4.2 Bagaimana rasio masuk ke period?

`QuotaResolver` (di `cmd/server/main.go:623`) sudah mengambil kuota dari plan. Perlu
diperluas agar juga mengembalikan rasio:

```go
// billing/ledger.go — SESUDAH
type QuotaResolver interface {
    ResolveQuota(ctx context.Context, userID uuid.UUID, at time.Time) (
        periodStart, periodEnd time.Time, quota int64, ratio int64, err error)
}
```

```go
// cmd/server/main.go — SESUDAH (konseptual)
quotaResolver := billing.QuotaResolverFunc(func(ctx, userID, at) (start, end time.Time, quota, ratio int64, err error) {
    res, err := entResolver.Resolve(ctx, userID)
    ...
    quota = res.Limit("ai_tokens_per_cycle")
    if quota < 0 { quota = 1_000_000_000 }
    ratio = res.Plan.TokenQuotaRatio      // ← BARU
    if ratio < 1 { ratio = 1 }
    ...
})
```

`ensurePeriodLocked` meneruskan `ratio` ke `GetOrCreatePeriod`, yang menyimpannya ke
`token_periods.quota_ratio`.

---

## 5. Sumber rasio: katalog plan

| Lapisan | Perubahan |
|---|---|
| `plans` (tabel) | +`token_quota_ratio bigint NOT NULL DEFAULT 1` |
| `billing.Plan` (struct) | +`TokenQuotaRatio int64` |
| `entitlement.Resolved` | +`Plan` (sudah ada?) atau +`QuotaRatio` |
| `PlanDTO` (admin) | +`token_quota_ratio` |
| `PlanPayload` (admin) | +`token_quota_ratio` |
| `PlanFormDialog` | +input rasio + pratinjau |
| `entitlement/catalog.go` | +entri katalog `token_quota_ratio` (label, deskripsi) |

---

## 6. Invarian yang harus dipertahankan

| # | Invarian | Cara verifikasi |
|---|---|---|
| I-1 | `used_tokens` **tidak pernah** > `quota_tokens` | Clamp di `Commit` (sudah ada, dipertahankan) |
| I-2 | `used_tokens` & `reserved_tokens` **tidak pernah** negatif | `chk_token_balance` (DB) + clamp |
| I-3 | `sum(commit.amount) - sum(refund.amount)` untuk periode = `used_tokens` | Test rekonsiliasi |
| I-4 | `reserve.Amount` = jumlah unit yang di-hold; `commit` melepas **seluruhnya** | Kode `AdjustPeriod(units, -reserve.Amount)` |
| I-5 | Total unit kembali (release+refund) **tidak melebihi** yang ditahan/ditagih | Clamp di `Refund` |
| I-6 | `cost_idr` selalu dari `raw_tokens`, tidak terpengaruh rasio | Test: ubah rasio, biaya tetap |
| I-7 | Rasio sebuah siklus tidak berubah setelah period dibuat | Kolom immutable (tidak ada UPDATE) |
| I-8 | `RoundUnits(0, r, m) = 0`; `RoundUnits(n, 1, m) = n` untuk semua mode; hasil `floor` tetap ≥ `quota_min_charge_units` bila `n > 0` | Unit test (semua mode) |

### 6.1 Fungsi pembulatan (dinamis)

> **PENTING (revisi):** mode pembulatan **tidak** dikonstanta di kode. Operator memilih
> via setting `quota_rounding_mode` (default `ceil`). Lihat
> [09-lapisan-konfigurasi-dinamis.md §6](./09-lapisan-konfigurasi-dinamis.md).

```go
// billing/rounding.go
type RoundingMode string

const (
    RoundCeil  RoundingMode = "ceil"
    RoundFloor RoundingMode = "floor"
    RoundRound RoundingMode = "round"
)

// RoundUnits membulatkan raw/ratio sesuai mode yang dikonfigurasi operator.
// Mode berasal dari setting `quota_rounding_mode` — bukan konstanta kode.
func RoundUnits(raw, ratio int64, mode RoundingMode) int64 {
    if ratio < 1 { ratio = 1 }
    if raw <= 0 { return 0 }
    switch mode {
    case RoundFloor: return raw / ratio
    case RoundRound: return (raw + ratio/2) / ratio
    default:         return (raw + ratio - 1) / ratio  // RoundCeil
    }
}
```

Contoh dengan `ceil`: `RoundUnits(9945, 5, ceil) = 1989`, `RoundUnits(1, 5, ceil) = 1`,
`RoundUnits(0, 5, ceil) = 0`.

> **Catatan lantai minimum:** dengan mode `floor`, `RoundUnits(1, 5) = 0` → request
> kecil **tidak menagih**. Karena itu ada setting `quota_min_charge_units` (default
> `1`) sebagai lantai, agar gate kuota tetap bermakna dan tidak bisa di-*spam* gratis.
> Ini **juga dinamis**.

---

## 7. Bagaimana dengan estimasi reserve?

Estimasi `EstimateStore.ForChat` tetap menghasilkan **token RAW** (karena menghitung
overhead prompt + panjang pesan). Konversi ke unit terjadi di handler:

```go
// ai_handler.go — SESUDAH (konseptual)
func (h *AIHandler) chatEstimateUnits(ctx context.Context, modality billing.Modality,
    message string, ratio int64) int64 {
    raw := h.chatEstimate(ctx, modality, message)   // tetap RAW
    return billing.CeilDiv(raw, ratio)              // → UNIT
}
```

**Masalah:** handler perlu tahu `ratio` **sebelum** period dibuat (karena reserve
adalah langkah pertama). Padahal `ratio` di-snapshot saat period dibuat.

**Solusi:** handler meminta rasio dari resolver yang sama. Karena `quotaResolver`
sudah tersedia, tambahkan metode publik di `LedgerService`:

```go
// CurrentRatio mengembalikan rasio siklus aktif, atau rasio plan bila belum ada
// period. Dipakai handler untuk mengonversi estimasi sebelum Reserve.
func (s *LedgerService) CurrentRatio(ctx context.Context, userID uuid.UUID) (int64, error)
```

Implementasi: coba `GetCurrentPeriod` → pakai `period.QuotaRatio`; jika nil, panggil
`quota.ResolveQuota` → pakai `ratio` (tanpa membuat period).

> **Alternatif yang lebih sederhana (dan direkomendasikan untuk implementasi awal):**
> pindahkan konversi **ke dalam** `Reserve`. Handler tetap mengirim RAW; `Reserve`
> yang mengonversi setelah `ensurePeriodLocked` (saat rasio sudah diketahui):

```go
// OPSI LEBIH SEDERHANA — Reserve menerima RAW, konversi internal
func (s *LedgerService) Reserve(ctx, userID, estRawTokens int64, modality, idempotencyKey) (*Reservation, error) {
    return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
        period, _ := s.ensurePeriodLocked(txCtx, userID)   // rasio diketahui di sini
        ratio := maxInt64(period.QuotaRatio, 1)
        estUnits := RoundUnits(estRawTokens, ratio, mode)
        if period.Available() < estUnits { return ErrInsufficientQuota }
        AdjustPeriod(0, +estUnits)
        // Reservation.Amount = estUnits (unit!)
        ...
    })
}
```

**Rekomendasi: pakai Opsi Lebih Sederhana.** Alasannya:
- Handler tidak perlu tahu rasio (tidak ada permintaan rasio tambahan ke DB).
- Konversi terpusat di ledger — satu tempat, satu sumber kebenaran.
- `Reservation.Amount` otomatis konsisten dalam unit, sehingga `Commit`/`Release`
  tidak perlu konversi tambahan untuk melepas hold.
- Symmetric: `Reserve(raw)`, `Commit(raw)`, `Refund(raw)` — semuanya menerima RAW,
  konversi internal. Kontrak jadi seragam dan sulit disalahgunakan.

Konsekuensinya: **`Reserve` sekarang butuh `ensurePeriodLocked` sebelum menghitung
unit** — sudah begitu (baris pertama di tx). Tidak ada biaya tambahan.

---

## 8. Dampak ke lapisan lain

### 8.1 Middleware gate (tidak berubah kodenya)

`WithinLimit("ai_tokens_per_cycle")` membandingkan `used + reserved` (unit) dengan
`limit` (unit). Karena keduanya unit, **tidak ada perubahan**. ✅

### 8.2 `usage_resolvers.go` (tidak berubah)

`TokenPeriodUsage` mengembalikan `used + reserved` — sudah unit. ✅

### 8.3 `GET /me/entitlements` (berubah label saja)

Nilai `used`, `limit`, `remaining` kini unit. **Field `unit`** dari katalog dipakai UI
untuk label ("kuota"). Tidak ada perubahan struktur.

### 8.4 Admin token ledger (berubah tampilan)

Kolom `amount` = unit, `raw_tokens` = token nyata, `cost_idr` = Rp. Tabel admin
sebaiknya menampilkan ketiganya agar operator bisa audit.

### 8.5 Pricing engine (tidak berubah)

`CostIDR(rawTokens, modality)` — dipanggil dengan RAW. Formula tidak disentuh. ✅

### 8.6 Kuota monitor Super Admin (berubah)

`admin/quota_monitor.go` menghitung pemakaian; kini unit. Perlu menampilkan token
nyata sebagai kolom terpisah untuk visibilitas biaya.

---

## 9. Contoh perhitungan end-to-end

**Skenario:** user Free (kuota 5.000 unit, rasio 5) chat "Buat grafik pengeluaran per kategori".

| Langkah | Nilai | Keterangan |
|---|---|---|
| Pesan | 36 karakter | |
| Estimasi RAW | `3000 + 36/4 + 800 = 3.809` | `ForChat` |
| Estimasi UNIT | `RoundUnits(3809, 5, ceil) = 762` | di dalam `Reserve` |
| Cek available | `5000 >= 762` ✅ | |
| Hold | `reserved = 762` | |
| Provider lapor | `total_tokens = 9.945` | RAW |
| Commit UNIT | `RoundUnits(9945, 5, ceil) = 1.989` | |
| Clamp | `maxCharge = 5000 - 0 = 5000`; `1989 < 5000` → tidak di-clamp ✅ | |
| Settle | `used = 1.989`, `reserved = 0` | |
| Biaya | `CostIDR(9945, chat)` = mis. Rp 3.729 | dari RAW ✅ |
| **Sisa kuota** | **3.011 unit** | user masih bisa chat ✅ |

Bandingkan **sebelum** perubahan: `used = 5.000` (clamp), sisa **0**, biaya tetap
Rp 3.729. Rasio menyelamatkan pengalaman user **tanpa** mengubah biaya.

---

## 10. Alternatif yang ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Naikkan kuota saja (tanpa rasio) | Angka kuota membengkak; eksposur biaya naik tanpa kendali; user tetap bingung satuan token |
| Kuota chat terpisah (batas permintaan) | Tidak mengontrol biaya model mahal; user bisa spam model reasoning |
| Rasio global (bukan per-plan) | Tidak fleksibel; tidak bisa membedakan Free vs Business |
| Rasio per-modalitas | Kompleksitas UI tinggi; ditunda sebagai opsi lanjutan |
| Konversi di UI saja | Tidak aman: kuota adalah gate otorisasi, harus di server |
| Simpan rasio di reservation | Perlu ikut di release/refund; lebih rawan tidak konsisten |

---

## 11. Ringkasan perubahan kode

| File | Perubahan | Risiko |
|---|---|---|
| `database/migrations/0025_*.sql` | +2 kolom + backfill | Rendah (expand-only) |
| `database/migrations/0026_*.sql` | +3 tabel katalog (`setting_definitions`, `metric_definitions`, `feature_definitions`) | Rendah (tabel baru) |
| `billing/domain.go` | `Plan.TokenQuotaRatio`, `TokenPeriod.QuotaRatio`, `TokenLedgerEntry.RawTokens`, `QuotaResolver` signature | Sedang (signature) |
| `billing/ledger.go` | `Reserve`/`Commit`/`Refund` konversi; `RoundUnits` (mode dinamis) | **Tinggi** (inti) |
| `billing/rounding.go` | **BARU** — `RoundingMode` + `RoundUnits` | Sedang |
| `billing/estimate.go` | Baca overhead/chars dari settings (fallback konstanta) | Rendah |
| `repository/postgres/billing_repo.go` | SELECT/INSERT kolom baru | Sedang |
| `admin/plans.go` | DTO/payload + validasi rasio | Rendah |
| `admin/settings.go` | `writableSettings` → `setting_definitions` | Sedang |
| `admin/catalog.go` | Katalog dari DB, bukan kode | Sedang |
| `middleware/usage_resolvers.go` | `UsageFor` → registry + `metric_definitions` | Sedang |
| `cmd/server/main.go` | `quotaResolver` mengembalikan rasio; sentinel dari setting | Sedang |
| Frontend | editor plan/settings/metrik, kalkulator, label kuota | Rendah |

---

## 12. Lapisan konfigurasi dinamis (ringkas)

> **Detail lengkap:** [09-lapisan-konfigurasi-dinamis.md](./09-lapisan-konfigurasi-dinamis.md)

Sesuai permintaan user ("semuanya dinamis tanpa hardcode"), tiga tabel katalog
menggantikan konstanta di kode:

| Hardcode lama | Menjadi | Efek |
|---|---|---|
| `writableSettings` (`admin/settings.go`) | `setting_definitions` | Setting baru tanpa deploy |
| `UsageFor()` switch (`usage_resolvers.go`) | `metric_definitions.resolver_key` | Metrik baru tanpa deploy |
| `FeatureCatalog`/`LimitCatalog` (`entitlement/catalog.go`) | `feature_definitions`/`metric_definitions` | Fitur/limit baru tanpa deploy |
| `ceilDiv` | `quota_rounding_mode` | Mode pembulatan bisa diubah |
| `chatOverheadTokens`/`charsPerToken` | `ai_estimate_*` settings | Estimasi bisa dituning |
| Sentinel `1e9` | `quota_unlimited_sentinel` | Bisa diubah |

**Seam yang diakui:** kode tetap menyediakan **kemampuan** (resolver usage, struktur
rute, config bootstrap); database menyimpan **kebijakan**. Operator memilih kebijakan
lewat UI; developer menyediakan kemampuan.

---

**Navigasi:** [← Ringkasan](./00-ringkasan-eksekutif.md) · [Skema & Migrasi →](./02-skema-database-dan-migrasi.md)
