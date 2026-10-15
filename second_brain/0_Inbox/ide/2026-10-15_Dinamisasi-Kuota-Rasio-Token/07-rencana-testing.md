# 07 — Rencana Testing

[← Index](./README.md) · [← Rencana Implementasi](./06-rencana-implementasi.md)

---

## 1. Strategi

| Lapisan | Tujuan | Lokasi |
|---|---|---|
| **Unit** | Fungsi murni: `RoundUnits`, rumus kalkulator, validasi | `billing/*_test.go`, `admin/*_test.go` |
| **Integrasi** | Ledger + DB nyata: konversi, invarian, idempotensi | `repository/postgres/*_test.go` |
| **E2E** | Alur lengkap lewat HTTP: gate → reserve → commit | `backend/tests/*_test.go` |
| **Concurrency** | Kuota tidak bocor/negatif saat paralel | `backend/tests/quota_concurrency_test.go` |
| **Regresi** | Rasio=1 ⇒ perilaku identik | Semua test lama |
| **Frontend** | Unit hook, komponen, a11y | `frontend/**/*.test.ts(x)` |

> **Aturan emas:** setiap invarian (I-1 … I-8) **wajib** punya test yang gagal bila
> invarian dilanggar.

---

## 2. Test unit — backend

### 2.1 `RoundUnits` (pembulatan dinamis)

```go
// billing/rounding_test.go
func TestRoundUnits_AllModes(t *testing.T) {
    cases := []struct{ raw, ratio int64; mode RoundingMode; want int64 }{
        // ceil (default, konservatif untuk bisnis)
        {0, 5, RoundCeil, 0},        // nol tetap nol
        {1, 5, RoundCeil, 1},        // minimum 1 unit
        {4, 5, RoundCeil, 1},
        {5, 5, RoundCeil, 1},
        {6, 5, RoundCeil, 2},
        {9945, 5, RoundCeil, 1989},  // kasus nyata
        {100, 1, RoundCeil, 100},    // rasio 1 = identitas
        {100, 0, RoundCeil, 100},    // ratio 0 dipaksa 1 (fail-safe)
        // floor (ramah user)
        {9945, 5, RoundFloor, 1989},
        {4, 5, RoundFloor, 0},       // ← bisa 0, butuh lantai
        // round (terdekat)
        {7, 5, RoundRound, 1},
        {8, 5, RoundRound, 2},
    }
    for _, c := range cases {
        if got := RoundUnits(c.raw, c.ratio, c.mode); got != c.want {
            t.Errorf("RoundUnits(%d,%d,%s) = %d, want %d", c.raw, c.ratio, c.mode, got, c.want)
        }
    }
}

// Lantai minimum menutup celah `floor` (round(4,5)=0).
func TestRoundUnits_MinChargeFloor(t *testing.T) {
    // dengan quota_min_charge_units = 1, hasil tidak boleh 0 bila raw > 0
    if got := applyMinCharge(RoundUnits(4, 5, RoundFloor), 1, 4); got != 1 {
        t.Fatalf("lantai minimum tidak diterapkan: got %d want 1", got)
    }
}
```

### 2.2 Rumus kalkulator

```go
// admin/quota_estimate_test.go
func TestEstimateQuota(t *testing.T) {
    // rasio 5, kuota 5000, 9945 token/chat
    got := computeQuota(5, 5000, 9945)
    if got.UnitPerChat != 1989 { t.Errorf("unit/chat = %d, want 1989", got.UnitPerChat) }
    if got.ChatsPerCycle != 2 { t.Errorf("chats = %d, want 2", got.ChatsPerCycle) }
    if got.Leftover != 3022 { t.Errorf("leftover = %d, want 3022", got.Leftover) }
}

func TestEstimate_RatioDoesNotAffectCost(t *testing.T) {
    // I-6: biaya harus SAMA untuk rasio berbeda
    r1 := computeCost(1, 9945)
    r5 := computeCost(5, 9945)
    if r1 != r5 {
        t.Fatalf("biaya berubah oleh rasio: %d != %d", r1, r5)
    }
}
```

### 2.3 Validasi rasio

| Input | Harapan |
|---|---|
| `token_quota_ratio: 0` | 400 `validation` |
| `token_quota_ratio: -1` | 400 `validation` |
| `token_quota_ratio: 1` | 200 OK |
| `token_quota_ratio: 1000` | 200 OK |
| `token_quota_ratio: 1001` | 400 `validation` |
| `token_quota_ratio` absen | default 1 |

---

## 3. Test integrasi — ledger (paling kritis)

### 3.1 Konversi dasar

```go
// repository/postgres/ledger_ratio_test.go
func TestCommit_ConvertsRawToUnits(t *testing.T) {
    // quota 5000 unit, rasio 5
    ledger := billing.NewLedgerService(repo, tx, fixedQuotaRatio{quota: 5000, ratio: 5, days: 30}, nil)
    userID := insertTestUser(t)

    res, err := ledger.Reserve(ctx, userID, 3809, billing.ModalityChat, "r-"+userID.String())
    // estimasi RAW 3809 → unit ceil(3809/5) = 762
    if res.Amount != 762 {
        t.Fatalf("reserve harus dalam UNIT: got %d want 762", res.Amount)
    }
    if err := ledger.Commit(ctx, userID, res.ID, 9945, "c-"+userID.String()); err != nil {
        t.Fatal(err)
    }
    period, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
    if period.UsedTokens != 1989 {
        t.Fatalf("used = %d, want 1989", period.UsedTokens)
    }
    if period.ReservedTokens != 0 {
        t.Fatalf("reserved = %d, want 0", period.ReservedTokens)
    }
}
```

### 3.2 Invarian I-6 — biaya tidak terpengaruh rasio

```go
func TestCommit_CostUnaffectedByRatio(t *testing.T) {
    // I-6: cost_idr HARUS sama untuk rasio 1 dan 5 pada token yang sama
    for _, ratio := range []int64{1, 5, 20} {
        ledger := newLedgerWithRatio(t, ratio)
        userID := insertTestUser(t)
        res, _ := ledger.Reserve(ctx, userID, 10000, billing.ModalityChat, key(userID))
        _ = ledger.Commit(ctx, userID, res.ID, 9945, ckey(userID))

        entry := lastCommitEntry(t, repo, userID)
        if entry.CostIDR != expectedCost(9945) {
            t.Fatalf("rasio %d: cost = %d, want %d", ratio, entry.CostIDR, expectedCost(9945))
        }
        if entry.RawTokens != 9945 {
            t.Fatalf("rasio %d: raw_tokens = %d, want 9945", ratio, entry.RawTokens)
        }
    }
}
```

### 3.3 Invarian I-1 — `used` tidak melebihi `quota` (clamp)

```go
func TestCommit_ClampsToQuota_WithRatio(t *testing.T) {
    // kuota 1000 unit, rasio 5, commit 100.000 token → 20.000 unit → clamp 1000
    ledger := billing.NewLedgerService(repo, tx, fixedQuotaRatio{quota: 1000, ratio: 5, days: 30}, nil)
    userID := insertTestUser(t)
    res, _ := ledger.Reserve(ctx, userID, 100, billing.ModalityChat, key(userID))
    _ = ledger.Commit(ctx, userID, res.ID, 100000, ckey(userID))

    period, _ := repo.GetPeriodForUpdate(ctx, userID, time.Now().UTC().Unix())
    if period.UsedTokens != 1000 { t.Fatalf("used = %d, want clamp 1000", period.UsedTokens) }
    if period.UsedTokens > period.QuotaTokens { t.Fatal("used melebihi quota") }
    if period.ReservedTokens != 0 { t.Fatal("reserved harus 0") }
}
```

### 3.4 Invarian I-4 — release melepas **seluruh** hold

```go
func TestRelease_ReturnsFullHold_WithRatio(t *testing.T) {
    ledger := newLedgerWithRatio(t, 5)
    userID := insertTestUser(t)
    res, _ := ledger.Reserve(ctx, userID, 9945, billing.ModalityChat, key(userID))
    // res.Amount = 1989 unit
    before, _ := repo.GetPeriodForUpdate(ctx, userID, nowUnix())
    _ = ledger.Release(ctx, userID, res.ID, rkey(userID))
    after, _ := repo.GetPeriodForUpdate(ctx, userID, nowUnix())

    if before.ReservedTokens-after.ReservedTokens != res.Amount {
        t.Fatal("release harus melepas SELURUH reserve.Amount")
    }
    if after.ReservedTokens != 0 { t.Fatal("reserved harus 0") }
    if after.UsedTokens != 0 { t.Fatal("release tidak boleh menagih") }
}
```

### 3.5 Invarian I-7 — rasio siklus beku

```go
func TestPeriodRatio_IsSnapshot(t *testing.T) {
    // Buat period dengan rasio 5, lalu ubah plan jadi 2.
    // Period BERJALAN harus tetap rasio 5.
    userID := insertTestUser(t)
    ledger := newLedgerWithRatio(t, 5)
    res, _ := ledger.Reserve(ctx, userID, 100, billing.ModalityChat, key(userID))
    _ = ledger.Commit(ctx, userID, res.ID, 100, ckey(userID))

    setPlanRatio(t, userID, 2)  // ubah plan

    period, _ := repo.GetPeriodForUpdate(ctx, userID, nowUnix())
    if period.QuotaRatio != 5 {
        t.Fatalf("rasio siklus harus beku di 5, got %d", period.QuotaRatio)
    }
}
```

### 3.6 Idempotensi (tetap dipertahankan)

```go
func TestCommit_Idempotent_WithRatio(t *testing.T) {
    // commit dua kali dengan key sama → tidak menagih dua kali
    // dan tidak menambah raw_tokens
}
```

### 3.7 Refund simetris

```go
func TestRefund_SymmetricWithRatio(t *testing.T) {
    // commit 9945 RAW (1989 unit) → refund 9945 RAW → used kembali 0
    // refund 2× dengan key sama → hanya sekali
    // refund melebihi yang ditagih → di-clamp
}
```

### 3.8 Rekonsiliasi ledger (I-3)

```go
func TestLedger_Reconciliation(t *testing.T) {
    // sum(commit.amount) - sum(refund.amount) == period.used_tokens
}
```

---

## 4. Test concurrency

### 4.1 Kuota tidak bocor dengan rasio

```go
// backend/tests/quota_concurrency_ratio_test.go
// Kuota 25 unit, rasio 5, estimasi 10 RAW (= 2 unit).
// 20 request paralel → tepat 12 sukses (24 unit), 8 ditolak.
// Invariant: used <= quota, reserved akhir = 0.
```

### 4.2 Tidak ada double-charge saat paralel

```go
// Dua commit dengan idempotency key sama, dijalankan paralel
// → hanya satu yang menagih
```

---

## 5. Test E2E (HTTP)

### 5.1 Alur operator

```
1. POST /admin/plans/estimate {ratio:5, quota:5000, tokens:9945}
   → 200, chats_per_cycle == 2, margin > 0
2. PUT /admin/plans/:id {token_quota_ratio: 5}
   → 200
3. GET /admin/plans
   → plan.token_quota_ratio == 5
```

### 5.2 Alur user

```
1. User Free chat (siklus baru, rasio 5)
   → 200 SSE, commit unit ≈ 1989, raw_tokens = 9945
2. GET /me/entitlements
   → usage.ai_tokens_per_cycle.used == 1989, remaining == 3011, unit == "kuota"
3. Chat lagi → berhasil (sisa cukup)
4. Chat sampai habis → 402 limit_exceeded dengan limit_key ai_tokens_per_cycle
```

### 5.3 Backward compatibility

```
1. Plan dengan rasio 1 (default)
2. Chat → commit unit == raw_tokens (perilaku lama)
3. GET /me/entitlements → angka sama seperti sebelum migrasi
```

---

## 6. Test regresi (WAJIB)

| Test lama | Harapan |
|---|---|
| `ledger_test.go` | Semua hijau (rasio 1) |
| `ledger_concurrency_test.go` | Hijau (clamp tetap bekerja) |
| `quota_concurrency_test.go` | Hijau (gate tetap bekerja) |
| `entitlement_e2e_test.go` | Hijau |
| `billing_refund_test.go` | Hijau |
| Semua test `frontend` | Hijau |

> **Catatan:** test lama memanggil `Reserve`/`Commit` dengan token. Karena rasio
> default 1 pada `fixedQuota` fixture, nilai unit == token, sehingga test **tidak
> perlu diubah**. Bila fixture perlu `ratio`, tambahkan field dengan default 1 agar
> test lama tetap kompilasi.

### 6.1 Penyesuaian fixture

```go
// SEBELUM
type fixedQuota struct{ quota int64; days int }

// SESUDAH (backward compatible: ratio default 0 → diperlakukan 1)
type fixedQuota struct{ quota int64; days int; ratio int64 }

func (f fixedQuota) ResolveQuota(...) (time.Time, time.Time, int64, int64, error) {
    r := f.ratio
    if r < 1 { r = 1 }   // ← test lama tanpa ratio tetap bekerja
    return start, end, f.quota, r, nil
}
```

---

## 7. Test frontend

### 7.1 Unit — hook & util

```ts
// hooks/useQuotaEstimate.test.ts
describe('useQuotaEstimate', () => {
  it('mengembalikan kuota & biaya dari API', async () => { /* ... */ })
  it('tidak memanggil API bila input tidak valid', async () => { /* ... */ })
})
```

### 7.2 Unit — kalkulasi tampilan

```ts
// lib/quota-format.test.ts
describe('formatQuota', () => {
  it('menampilkan satuan kuota, bukan token', () => {
    expect(formatQuota({ used: 1989, limit: 5000, unit: 'kuota' })).toContain('kuota')
    expect(formatQuota({ used: 1989, limit: 5000, unit: 'kuota' })).not.toContain('token')
  })
})
```

### 7.3 Komponen — editor plan

```tsx
// components/super-admin/plans/PlanFormDialog.dom.test.tsx
it('menampilkan dropdown katalog, bukan input teks bebas', () => {
  render(<PlanFormDialog open ... />)
  expect(screen.getByRole('combobox', { name: /limit/i })).toBeInTheDocument()
})
it('memperingatkan limit yang belum ditegakkan', () => {
  // pilih max_categories → peringatan muncul
})
```

### 7.4 A11y

```tsx
it('tidak melanggar a11y', async () => {
  const { container } = render(<QuotaCalculator />)
  expect(await axe(container)).toHaveNoViolations()
})
```

### 7.5 Regresi label

```tsx
it('QuotaUsageCard memakai label "Kuota AI", bukan "Token AI"', () => {
  render(<QuotaUsageCard usage={{ ai_tokens_per_cycle: { used: 1, limit: 5, remaining: 4, unlimited: false, unit: 'kuota' } }} />)
  expect(screen.getByText(/kuota ai/i)).toBeInTheDocument()
  expect(screen.queryByText(/token ai/i)).not.toBeInTheDocument()
})
```

---

## 8. Matriks cakupan invarian

| Invarian | Deskripsi | Test |
|---|---|---|
| I-1 | `used` ≤ `quota` | §3.3, §4.1 |
| I-2 | `used`/`reserved` ≥ 0 | §3.3, §4.1 (DB check) |
| I-3 | Rekonsiliasi ledger | §3.8 |
| I-4 | Release melepas seluruh hold | §3.4 |
| I-5 | Refund tidak melebihi tagihan | §3.7 |
| I-6 | Biaya tidak terpengaruh rasio | §2.2, §3.2 |
| I-7 | Rasio siklus beku | §3.5 |
| I-8 | `RoundUnits` benar di semua mode + lantai minimum | §2.1 |

---

## 8b. Test konfigurasi dinamis (BARU)

> Verifikasi bahwa kebijakan benar-benar berasal dari DB, bukan konstanta kode.
> Detail: [09-lapisan-konfigurasi-dinamis.md](./09-lapisan-konfigurasi-dinamis.md).

### 8b.1 Katalog setting

```go
func TestSettingsCatalog_UsesDBDefinitions(t *testing.T) {
    // 1. Tambah definisi baru ke DB
    // 2. Pastikan key baru DAPAT diubah (sebelumnya ditolak)
    // 3. Hapus definisi → key kembali ditolak (fail-closed)
}

func TestSettingsCatalog_FallbackWhenTableEmpty(t *testing.T) {
    // Tabel kosong → sistem tetap berjalan dengan konstanta fallback
    // (bukan error 500, bukan penolakan semua setting)
}

func TestSettingsCatalog_ValidationFromDB(t *testing.T) {
    // min/max/enum diambil dari DB, bukan kode:
    // ubah max_value di DB → nilai di luar batas baru ditolak
}
```

### 8b.2 Katalog metrik & resolver

```go
func TestMetricsCatalog_NewMetricWithoutDeploy(t *testing.T) {
    // 1. INSERT metric_definitions (resolver_key = resolver yang sudah ada)
    // 2. Pastikan UsageFor(key) mengembalikan resolver yang benar
    // 3. Pastikan gate menolak saat usage mencapai limit
}

func TestMetricsCatalog_EnforcedRequiresResolver(t *testing.T) {
    // is_enforced = true tanpa resolver_key → validasi menolak
}

func TestUsageFor_UnknownResolverLogsAndReturnsNil(t *testing.T) {
    // resolver_key tidak ada di registry → usage = 0 + log error (tidak crash)
}

func TestUsageFor_NonEnforcedMetricHasNoResolver(t *testing.T) {
    // is_enforced = false → UsageFor mengembalikan nil
}
```

### 8b.3 Pembulatan dinamis end-to-end

```go
func TestLedger_RoundingModeFromSetting(t *testing.T) {
    // Set quota_rounding_mode = "floor" → commit 4 raw (rasio 5)
    //   dengan lantai 1 → menagih 1 unit (bukan 0)
    // Set quota_rounding_mode = "ceil" → commit 4 raw → menagih 1 unit
    // Set "round" → commit 8 raw → menagih 2 unit
    // Perubahan setting berlaku tanpa restart (dalam TTL cache)
}
```

### 8b.4 Default dinamis

```go
func TestPlanDefaults_FromSettings(t *testing.T) {
    // Set quota_default_ratio = 3, quota_default_tokens_per_cycle = 20000
    // Buat plan baru tanpa menentukan nilai
    // → plan memakai default dari setting (bukan konstanta kode)
}

func TestUnlimitedSentinel_FromSetting(t *testing.T) {
    // Set quota_unlimited_sentinel = 500000
    // Plan dengan limit -1 → quota period = 500000 (bukan 1e9 hardcoded)
}
```

### 8b.5 Invalidasi event-driven (P-13)

```go
func TestConfigCache_InvalidatedOnVersionBump(t *testing.T) {
    // 1. Muat katalog (versi 0)
    // 2. Ubah setting di DB + BumpVersion("cfg")
    // 3. Pembacaan berikutnya HARUS memuat ulang (bukan nilai lama)
}

func TestConfigCache_NoReloadWhenVersionUnchanged(t *testing.T) {
    // Versi sama → tidak query DB lagi (bukti cache bekerja)
    // Hitung pemanggilan loader: harus 1× untuk N pembacaan
}

func TestConfigChanged_PublishedAfterWrite(t *testing.T) {
    // PUT /admin/settings → capture emitter
    // → event config_changed terkirim dengan scope + keys benar
}

func TestConfigChanged_ScopesAllWriters(t *testing.T) {
    // settings / metrics / features / plans / pricing
    // masing-masing memicu event dengan scope yang tepat
}

func TestConfigReload_ForcesInvalidation(t *testing.T) {
    // POST /admin/config/reload → versi naik + event terkirim
}

func TestConfigCache_FallsBackToMemoryWhenRedisDown(t *testing.T) {
    // Redis mati → CacheEngine pakai map lokal → sistem tetap jalan
}
```

```ts
// frontend/lib/realtime-dispatcher.test.ts
it('config_changed menginvalidasi katalog terkait', () => {
  const keys = keysForEvent('config_changed')
  expect(keys).toContainEqual(queryKeys.admin.metricDefinitions)
  expect(keys).toContainEqual(queryKeys.entitlements.me)
})
it('config_changed TIDAK memicu cache storm', () => {
  // harus daftar key spesifik, bukan invalidateQueries() tanpa filter
  expect(keysForEvent('config_changed').length).toBeLessThan(8)
})
```

### 8b.6 Fail-safe

| Skenario | Harapan |
|---|---|
| `setting_definitions` kosong | Setting tidak dapat diubah (fail-closed); sistem tetap jalan |
| `metric_definitions` kosong | Gate tidak dipasang; `limit_not_in_plan` tetap menolak |
| Setting hilang saat runtime | Pakai `default_value` definisi → fallback konstanta |
| DB error saat baca katalog | Pakai cache terakhir; bila tidak ada → fallback default |
| `resolver_key` tidak dikenal | Usage = 0 + log error (tidak crash) |
| SSE/Redis pub/sub mati | `BumpVersion` tetap terbaca saat request → perubahan tetap berlaku |
| Event hilang (klien offline) | Buffer replay saat resume + TTL 5 menit jaring terakhir |
| Perubahan DB langsung (di luar API) | `POST /admin/config/reload` atau tunggu TTL 5 menit |

---

## 9. Perintah verifikasi

```bash
# Backend
cd backend
go vet ./...
go build ./...
go test ./...                              # semua
go test ./internal/billing/... -run Ratio  # fokus
go test ./internal/repository/postgres/... # integrasi (butuh DB)
go test ./tests/... -run Quota             # e2e

# Frontend
cd frontend
pnpm lint
pnpm test
pnpm build

# Database
psql "$TEST_DATABASE_URL" -f database/migrations/0028_plan_token_quota_ratio.sql
psql "$TEST_DATABASE_URL" -f database/migrations/0028_plan_token_quota_ratio.sql  # ulang = idempoten
```

---

## 10. Kriteria kelulusan

| # | Kriteria | Cara verifikasi |
|---|---|---|
| 1 | Semua test lama hijau | `go test ./...`, `pnpm test` |
| 2 | Test invarian baru hijau | Test §2–§5 |
| 3 | Rasio 1 identik perilaku lama | Test regresi §6 |
| 4 | Biaya tidak berubah oleh rasio | Test I-6 §3.2 |
| 4b | Kebijakan dari DB, bukan kode | Test §8b |
| 5 | Concurrency aman | Test §4 |
| 6 | Migrasi idempoten | Jalankan 2× |
| 7 | UI aksesibel | Test axe |
| 8 | Label user benar | Test §7.5 |

---

## 11. Test manual (checklist operator)

**A. Konfigurasi dinamis (tanpa deploy)**

- [ ] Buka `/super-admin/settings` → semua setting dirender dari katalog DB
- [ ] Ubah `quota_rounding_mode` ke `floor` → simpan → chat menagih pembulatan bawah
- [ ] Ubah `quota_unit_label` → label berubah di halaman user
- [ ] Ubah `quota_default_tokens_per_cycle` → buat plan baru → kuota mengikuti default
- [ ] Buka `/super-admin/metrics` → tambah metrik baru (resolver dari dropdown)
- [ ] Metrik baru muncul di dropdown limit editor plan **tanpa restart**
- [ ] Buka `/super-admin/features` → tambah fitur → muncul di editor plan

**B. Plan & kalkulator**

- [ ] Buka editor plan → dropdown key muncul, tidak bisa ketik bebas
- [ ] Ubah rasio → simulasi otomatis tampil
- [ ] Buka kalkulator → geser rasio, angka berubah < 1 detik
- [ ] Isi `provider_cost_per_1k_idr` → margin muncul (bukan 100%)
- [ ] Simpan plan → muat ulang, nilai tersimpan

**C. Pengalaman user**

- [ ] Login sebagai user Free → badge kuota menunjukkan satuan "kuota"
- [ ] Chat → kuota berkurang sesuai (bukan langsung habis)
- [ ] Habiskan kuota → pesan jelas + tautan upgrade

**D. Fail-safe**

- [ ] Kosongkan `setting_definitions` → sistem tetap berjalan, setting tidak bisa diubah
- [ ] Set `resolver_key` ke nilai tak dikenal → tidak crash, ada log error

---

**Navigasi:** [← Rencana Implementasi](./06-rencana-implementasi.md) · [Risiko & Rollback →](./08-risiko-dan-rollback.md)
