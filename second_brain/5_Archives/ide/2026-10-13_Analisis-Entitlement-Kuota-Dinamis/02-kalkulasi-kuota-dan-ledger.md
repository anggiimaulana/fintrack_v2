# 02 — Kalkulasi Kuota & Token Ledger

| Field | Isi |
|---|---|
| **Dokumen** | Siklus Kuota, Rumus, Ledger 2-Fase, Skenario Kuota Menipis |
| **Tanggal** | 2026-10-13 |
| **Status** | Analisis final (kode + verifikasi live) |

---

## 1. Apa Itu "Kuota" di FinTrack

Ada **tiga jenis batas** yang sering disebut "kuota" — dan penting dibedakan:

| Jenis | Contoh kunci | Mekanisme | Di mana |
|---|---|---|---|
| **Feature (biner)** | `ocr_receipt` | Boolean: boleh/tidak | `plan_entitlements` |
| **Limit siklus (numerik)** | `max_transactions`, `ocr_scans_per_cycle`, `ai_requests_per_cycle` | Dihitung per periode | `plan_limits` + usage resolver |
| **Kuota token (anggaran)** | `ai_tokens_per_cycle` | Saldo token dikonsumsi via ledger | `token_periods` + `token_ledger` |

Yang tampil di halaman Langganan (Gambar #2) adalah **limit siklus** dan **kuota token**.

---

## 2. Siklus Kuota (Token Period)

### 2.1 Definisi siklus

```
token_periods:
  period_start | period_end | quota_tokens | used_tokens | reserved_tokens
```

- **Available** = `quota_tokens − used_tokens − reserved_tokens`
- Siklus default = `billing_period_days` plan (30 hari), **tidak ada rollover**.
- Siklus dibuat otomatis saat AI call pertama pada periode baru.

**Bukti:** `billing/domain.go` (`TokenPeriod.Available()`),
`billing/ledger.go:224-250` (`ensurePeriodLocked`).

### 2.2 Resolusi kuota per siklus

```go
// cmd/server/main.go — quotaResolver
quota := res.Limit("ai_tokens_per_cycle")
if quota < 0 { quota = 1_000_000_000 }   // unlimited → angka besar
sub, _ := billingRepo.GetByUserID(ctx, userID)
if sub != nil && sub.CurrentPeriodEnd != nil {
    return sub.CurrentPeriodStart, *sub.CurrentPeriodEnd, quota, nil
}
// tanpa subscription: siklus 30 hari dari sekarang
start := at.Truncate(24 * time.Hour)
return start, start.AddDate(0, 0, 30), quota, nil
```

> ⚠️ **Catatan:** "unlimited" diterjemahkan menjadi `1e9` token (bukan sentinel `-1`),
> karena ledger butuh angka nyata untuk pengurangan. Ini berarti Business secara teknis
> dibatasi 1 miliar token/siklus — praktis tak terbatas, tetapi bukan `∞` sejati.

---

## 3. Rumus Kuota — Angka di Halaman Langganan

| Kunci | Free | Pro | Business | Cara dihitung |
|---|---:|---:|---:|---|
| `ai_requests_per_cycle` | 50 | 2.000 | ∞ | Jumlah **request AI** per siklus — 🔴 **belum ditegakkan** |
| `ai_tokens_per_cycle` | 5.000 | 150.000 | 1.000.000 | Saldo token; dikurangi per AI call — 🔴 **flat 1/chat** |
| `max_categories` | 5 | 30 | ∞ | Jumlah kategori — 🔴 **tanpa usage resolver** |
| `max_transactions` | 100 | 5.000 | ∞ | `COUNT(transactions)` — ✅ **berjalan** |
| `max_wallets` | 1 | 5 | ∞ | Jumlah dompet — 🔴 **tanpa usage resolver** |
| `ocr_scans_per_cycle` | 0 | 100 | ∞ | `COUNT(ledger vision)` — 🔴 **ledger tak ditulis** |

### 3.1 Yang BENAR-benar berjalan

**`max_transactions`** (satu-satunya limit numerik yang lengkap):

```go
// usage_resolvers.go:34
case "max_transactions": return p.countTransactions
// main.go
CountTransactions: txRepo.CountActiveByUser,
```
→ Setiap `POST /transactions` memeriksa `COUNT(*) >= limit` → 402 jika penuh. ✅

### 3.2 Yang TIDAK berjalan

- **`ai_requests_per_cycle`**: tidak ada `case` di `UsageFor`, tidak ada gate di router.
  Angka 50/2.000/∞ **murni kosmetik**.
- **`max_categories`, `max_wallets`**: tidak ada usage resolver → `WithinLimit` membaca
  usage `0` (karena resolver nil) → **selalu lolos**.
- **`ocr_scans_per_cycle`**: resolver ada (`CountVisionScansInPeriod`), **tetapi** ledger
  vision tidak pernah ditulis → count selalu 0 → **selalu lolos**.
- **`ai_tokens_per_cycle`**: dihitung, tetapi chat hanya membebankan **1 token** per pesan.

---

## 4. Token Ledger 2-Fase (Reserve → Commit/Release)

### 4.1 Mengapa 2-fase?

AI call bisa berlangsung lama & bisa gagal. Kita tidak tahu token asli sebelum respons.
Solusinya: **kunci dulu (reserve)**, lalu **tetapkan biaya asli (commit)** atau
**kembalikan (release)**.

```
┌─────────────┐   Reserve(est)    ┌──────────────────┐
│ Available   │ ───────────────▶  │ reserved += est  │
└─────────────┘                   └──────────────────┘
                                        │
                          ┌─────────────┴─────────────┐
                          ▼                           ▼
                   Commit(actual)               Release()
                   used += actual               reserved -= est
                   reserved -= est              (tidak dibebankan)
```

### 4.2 Implementasi

```go
// billing/ledger.go:80-118 — Reserve
Reserve(userID, estTokens, modality, idempotencyKey):
  if existing := GetLedgerByIdempotencyKey(key); existing != nil { return existing }  // idempoten
  RunInTx:
    period := ensurePeriodLocked()               // SELECT ... FOR UPDATE
    if period.Available() < estTokens { return ErrInsufficientQuota }
    AdjustPeriod(usedDelta=0, reservedDelta=+est)
    InsertLedger(reserve)

// billing/ledger.go:126-177 — Commit
Commit(userID, reservationID, actualTokens, key):
  if resolved(reservationID) { return ErrReservationAlreadyResolved }
  RunInTx:
    period := GetPeriodForUpdate()
    AdjustPeriod(usedDelta=+actual, reservedDelta=-est)   // lepas reserve, bebankan actual
    InsertLedger(commit, cost = pricing.CostIDR(actual, modality))

// billing/ledger.go:180-222 — Release
Release(userID, reservationID, key):
  RunInTx:
    AdjustPeriod(usedDelta=0, reservedDelta=-est)         // kembalikan penuh
    InsertLedger(release)
```

### 4.3 Rumus biaya (pricing engine)

```go
// billing/pricing.go:28-51
cost = floor( base_per_1k × (tokens/1000) × markup_multiplier
              × (1 + markup_percent/100) × modality_multiplier )
```

**Nilai rule aktif (DB live):**

| Field | Nilai |
|---|---|
| `base_per_1k_idr` | 300 |
| `markup_multiplier` | 1.25 |
| `markup_percent` | 0 |
| `modality_multipliers` | `{chat:1.0, voice:3.0, vision:2.5, embedding:0.5, reasoning:2.0}` |

Contoh: 1.000 token chat → `floor(300 × 1 × 1.25 × 1.0 × 1.0)` = **375 rupiah**.

> 🔴 **GAP:** engine ini **tidak pernah di-wire** (`NewPricingEngine` tidak dipanggil di
> `main.go`) → `s.pricing == nil` → `cost_idr` selalu **0**. Bukti DB: semua baris
> `token_ledger.cost_idr = 0` kecuali 5 baris dari test.

---

## 5. Skenario Kuota Menipis — Analisis Menyeluruh

### 5.1 Skenario A: Chat saja, kuota tipis

Sisa 3 token, 5 chat bersamaan:

```
Chat 1 → Reserve(1) OK  → reserved=1, available=2
Chat 2 → Reserve(1) OK  → reserved=2, available=1
Chat 3 → Reserve(1) OK  → reserved=3, available=0
Chat 4 → Reserve(1) → Available()=0 < 1 → 429 insufficient_quota ❌
Chat 5 → 429 ❌
```

**Hasil:** 3 sukses, 2 ditolak. **Aman** (row lock mencegah over-spend).

### 5.2 Skenario B: Chat + OCR + Voice bersamaan, kuota tipis

Sisa 1 token:

```
Chat  → Reserve(1) OK → available=0
OCR   → tidak menyentuh ledger → LOLOS (bug!)
Voice → tidak menyentuh ledger → LOLOS (bug!)
```

**Hasil:** kuota "habis" tetapi OCR & voice tetap berjalan. **Tidak aman secara bisnis.**

### 5.3 Skenario C: Chat dengan token asli besar

Pesan chat mengonsumsi ~1.200 token asli, tetapi:

```
Reserve(1)  → reserved=1
(ai call, token asli = 1200)
Commit(1)   → used += 1   ← SALAH, seharusnya += 1200
```

**Hasil:** kuota bocor 1.199 token per pesan. Dengan limit 5.000 (Free), user bisa
melakukan ~5.000 pesan alih-alih ~4 pesan. **Kebocoran besar.**

### 5.4 Skenario D: AI call gagal / timeout

```
Reserve(1) → reserved=1
(ai call timeout)
Release()  → reserved=0   ✅ token kembali
```

**Hasil:** benar — token dikembalikan. (Di `ai_handler.go:107`, `gotContent=false` →
`Release`.)

### 5.5 Skenario E: Proses crash setelah Reserve

```
Reserve(1) → reserved=1
[proses mati sebelum Commit/Release]
→ reserved=1 selamanya (token terkunci)
```

**Mitigasi yang SEHARUSNYA:** cleaner auto-release.
🔴 **GAP:** cleaner **tidak di-start** → token terkunci permanen.

### 5.6 Ringkasan skenario

| Skenario | Mekanisme lock | Hasil bisnis |
|---|---|---|
| A. Chat konkuren | ✅ Benar | ✅ Aman |
| B. Chat+OCR+Voice | ⚠️ Parsial | ❌ OCR/Voice tak dibatasi |
| C. Token asli besar | ✅ Benar | ❌ Kuota bocor (flat 1) |
| D. AI gagal | ✅ Benar | ✅ Token kembali |
| E. Crash | ✅ Ada mekanisme | ❌ Cleaner mati → terkunci |

---

## 6. Yang Terjadi Jika Kuota Menipis (Perilaku Nyata)

| Kondisi | Respons saat ini |
|---|---|
| Token habis (ledger) | **429** `insufficient_quota` — "Sisa kuota token Anda tidak mencukupi..." |
| Limit transaksi penuh | **402** `limit_exceeded` |
| Fitur tak ada di plan | **403** `feature_not_in_plan` |
| Limit OCR "penuh" | 🔴 **tidak pernah terjadi** (tak dihitung) |
| Request AI "penuh" | 🔴 **tidak pernah terjadi** (tak ditegakkan) |

> ⚠️ **UX gap:** frontend `useAiChat.ts` **tidak** memetakan `429 insufficient_quota`
> ke toast upsell (`notifyEntitlementError`). User hanya melihat error generik.

---

## 7. Implementasi di Sisi User & Super Admin

### 7.1 Sisi user — saat ini

- Halaman Langganan menampilkan **daftar limit statis** (dari plan), bukan **sisa real-time**.
- 🔴 **Tidak ada** indikator "sisa kuota Anda" (token terpakai vs kuota).
- 🔴 **Tidak ada** notifikasi saat kuota menipis (mis. 80%).

### 7.2 Sisi super admin — saat ini

| Fitur admin | Status |
|---|---|
| `GET /admin/tokens/users` (saldo per user) | ✅ Ada (quota/used/reserved/available) |
| `GET /admin/tokens/ledger` (mutasi) | ✅ Ada |
| `POST /admin/tokens/adjust` (koreksi manual) | ✅ Ada (idempoten) |
| `GET /admin/metrics/overview` (token 24h) | ✅ Ada |
| Monitoring kuota mendekati habis | ❌ Tidak ada |
| Endpoint ubah `pricing_rules` | ❌ Tidak ada |
| Breakdown konsumsi per modalitas | ⚠️ Parsial (ledger punya `modality`) |

**Bukti live:**
```json
GET /admin/tokens/users →
{"user_id":"...","quota_tokens":10000,"used_tokens":1000,"reserved_tokens":0,"available_tokens":9000}

GET /admin/metrics/overview →
{"mrr_idr":117000,"tokens_consumed_24h":14,...}
```

---

## 8. Kesimpulan Bagian Ini

| Pertanyaan | Jawaban |
|---|---|
| Bagaimana kuota dihitung? | Per siklus (`token_periods`), available = quota − used − reserved |
| Bagaimana implementasinya? | Ledger 2-fase (reserve/commit/release) dengan row lock + idempotency |
| Apa yang terjadi jika kuota menipis? | 429 untuk token, 402 untuk limit transaksi |
| Bagaimana chat+voice+struk bersamaan? | Lock benar, tetapi OCR/voice tak dibebankan (cakupan tidak lengkap) |
| Apakah semua logic sudah ada? | **Tidak** — 4 gap kritis (OCR, token asli, request count, pricing) |

Lanjut ke **[03 — Temuan Gap & Race Condition](03-temuan-gap-dan-race-condition.md)**.
