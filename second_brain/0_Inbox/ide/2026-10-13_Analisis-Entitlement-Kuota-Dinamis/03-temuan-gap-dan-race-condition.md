# 03 — Temuan Gap & Analisis Race Condition

| Field | Isi |
|---|---|
| **Dokumen** | Daftar Temuan (Bug/Gap) dengan Bukti Kode & Uji Live |
| **Tanggal** | 2026-10-13 |
| **Status** | Terverifikasi terhadap instance live |

---

## Ringkasan Temuan

| ID | Temuan | Severity | Status |
|---|---|---|---|
| **BUG-01** | Sidebar: "Langganan" & "Pengaturan" aktif bersamaan | 🟢 Rendah | ✅ Diperbaiki |
| **GAP-01** | OCR/vision tidak menulis ledger → kuota OCR tak terbatas | 🔴 Kritis | ✅ F1 |
| **GAP-02** | Chat menagih 1 token flat, bukan token asli | 🔴 Kritis | ✅ F1 |
| **GAP-03** | `ai_requests_per_cycle` tidak ditegakkan | 🟠 Tinggi | ✅ F1 |
| **GAP-04** | Pricing engine tidak di-wire → `cost_idr = 0` | 🟠 Tinggi | ⬜ |
| **GAP-05** | `max_categories`/`max_wallets` tanpa usage resolver | 🟡 Sedang | ⬜ |
| **GAP-06** | `pricing_rules` tanpa endpoint admin (hardcode efektif) | 🟠 Tinggi | ⬜ |
| **GAP-07** | User tak melihat sisa kuota real-time | 🟡 Sedang | ⬜ |
| **GAP-08** | Error kuota (429) tak dipetakan ke toast upsell | 🟡 Sedang | ⬜ |
| **GAP-09** | `ApplyScheduledDowngrade` tidak pernah dipanggil | 🟠 Tinggi | ⬜ |
| **GAP-10** | Token cleaner tidak di-start → reservasi menggantung | 🟠 Tinggi | ⬜ |
| **GAP-11** | Grace/dunning worker di-wire dengan `nil` status setter | 🟠 Tinggi | ⬜ |

---

## BUG-01 — Sidebar Dua Menu Aktif Bersamaan

**Severity:** 🟢 Rendah (kosmetik, sudah diperbaiki)

### Fakta

`frontend/components/layout/Sidebar.tsx` (dan `Header.tsx`) memakai:

```tsx
const isActive = pathname === item.href || (item.href !== "/user" && pathname.startsWith(item.href));
```

Pada `/user/settings/subscription`, ekspresi ini benar untuk **dua** item:
- `/user/settings/subscription` (Langganan) → `startsWith` cocok
- `/user/settings` (Pengaturan) → `startsWith` cocok

**Dampak:** dua menu tersorot (terlihat di Gambar #1).

### Perbaikan (SUDAH DILAKUKAN)

Dibuat `frontend/lib/nav.ts` dengan **longest-prefix matching**:

```ts
export function activeNavHref(pathname, items) {
  let best = null;
  for (const item of items) {
    const matches = pathname === item.href || pathname.startsWith(`${item.href}/`);
    if (matches && (best === null || item.href.length > best.length)) best = item.href;
  }
  return best;
}
```

- Cocok hanya jika sama persis **atau** anak sebenarnya (`href + "/"`) — `/user/settingsX`
  tidak lagi cocok dengan `/user/settings`.
- Saat beberapa cocok, href **terpanjang** menang.

`Sidebar.tsx` & `Header.tsx` kini memakai `isNavItemActive()`. Diverifikasi oleh
`lib/nav.test.ts` (7 test, termasuk regresi "hanya Langganan yang aktif").

---

## GAP-01 — 🔴 OCR/Vision Tidak Menulis Ledger

**Severity:** Kritis

### Bukti kode

`internal/delivery/http/handler/ai_handler.go` — `AnalyzeReceipt` (baris 238-289)
**tidak** memanggil `h.ledger.Reserve`/`Commit`. Bandingkan dengan `Chat` (baris 66, 107)
yang memanggil keduanya.

```go
// ai_handler.go — Chat MEMAKAI ledger:
res, err := h.ledger.Reserve(c.Request.Context(), userID, 1, modality, idempotencyKey)

// ai_handler.go — AnalyzeReceipt TIDAK memakai ledger:
result, err := h.aiUC.AnalyzeReceipt(c.Request.Context(), userID, imageData, contentType)
```

`CountVisionScansInPeriod` (usage resolver untuk `ocr_scans_per_cycle`) menghitung
`token_ledger WHERE modality='vision'` — yang **tidak pernah ditulis**.

### Bukti uji live

```
User Pro (ocr_scans_per_cycle = 100)
POST /api/v1/ai/receipt × 4 → HTTP 200, 200, 200, 200
Query: SELECT modality, count(*) FROM token_ledger WHERE user_id=... 
→ (0 rows)   ← TIDAK ADA entri vision
```

Ledger `vision` yang ada di DB hanya berasal dari **test** (`vision | reserve | 200 | 5`),
bukan produksi.

### Dampak

- Kuota OCR **tidak pernah** berkurang → OCR Pro/Business **tak terbatas**.
- Batas Free (0 scan) ditegakkan **hanya** oleh feature gate `ocr_receipt=false`, bukan limit.
- Biaya AI vision tidak terlacak.

### ✅ Perbaikan (F1, 2026-10-13)

`AnalyzeReceipt` kini membungkus panggilan dengan ledger: `Reserve(1, ModalityVision)` →
`AnalyzeReceipt` → `Commit(usage.TotalTokens)` / `Release`. **Bukti live:** 3 OCR → ledger
`vision reserve=1 ×3`, `vision commit ≈3750 ×3`; `used_tokens` naik 11.387; limit OCR
diturunkan ke 5 → request berikutnya `402 limit_exceeded`.

---

## GAP-02 — 🔴 Chat Menagih 1 Token Flat

**Severity:** Kritis

### Bukti kode

```go
// ai_handler.go:66 — reserve SELALU 1
res, err := h.ledger.Reserve(c.Request.Context(), userID, 1, modality, idempotencyKey)

// ai_handler.go:107 — commit SELALU 1 (bukan token asli dari respons)
_ = h.ledger.Commit(context.Background(), userID, reservation.ID, 1, "commit:"+...)
```

Token asli dari provider (biasanya 200-2.000 per pesan) **tidak pernah** dibaca atau
di-commit.

### Bukti uji live

```sql
SELECT modality, entry_type, amount, count(*) FROM token_ledger WHERE modality='chat' GROUP BY 1,2,3;
-- chat | commit | 1 | 15     ← 15 commit bernilai 1
```

### Dampak

- Kuota token **bocor besar**: 1 pesan ≈ 1 token, padahal nyata ≈ 1.200 token.
- Free (5.000 token) bisa ~5.000 pesan, bukan ~4 pesan.
- `cost_idr` per chat juga salah (hanya 1 token).

### ✅ Perbaikan (F1, 2026-10-13)

`gatewayStream` kini menangkap `usage` dari chunk terminal (provider memakai
`stream_options.include_usage`), mengakumulasinya lintas panggilan agent-loop, dan
meneruskannya pada event `done` (`ChatStreamEvent.Usage`). Handler mem-`Commit` token asli,
dengan fallback estimasi `metering.FromProvider` bila provider tidak melaporkan usage.
**Bukti live:** 1 chat → `usage.total_tokens=9945` (provider_reported), ledger
`chat commit=9945` (sebelumnya selalu 1).

---

## GAP-03 — 🟠 `ai_requests_per_cycle` Tidak Ditegakkan

**Severity:** Tinggi

### Bukti

```bash
grep -rn "ai_requests_per_cycle" --include=*.go . | grep -v _test
# (kosong — tidak ada hasil)
```

- Tidak ada `case "ai_requests_per_cycle"` di `usage_resolvers.go` (hanya 3 kasus).
- Tidak ada `limitGate(config, "ai_requests_per_cycle")` di `router.go`.

### Dampak

Angka **50 / 2.000 / ∞** di halaman Langganan **murni kosmetik** — tidak ada penegakan.

### ✅ Perbaikan (F1, 2026-10-13)

Ditambah `CountAIRequestsInPeriod` (menghitung entri `reserve` per siklus), resolver
`ai_requests_per_cycle` di `UsageFor`, dan `limitGate(config, "ai_requests_per_cycle")` pada
rute `/ai/chat`, `/ai/receipt`, `/ai/voice`. **Bukti live:** usage 9 request, limit 8 →
`402 limit_exceeded` (`current_usage:9, limit_value:8`).

---

## GAP-04 — 🟠 Pricing Engine Tidak Di-Wire

**Severity:** Tinggi

### Bukti

```bash
grep -rn "NewPricingEngine" --include=*.go . | grep -v _test
# hanya definisi di pricing.go — tidak ada pemanggil

grep -n "NewLedgerService" cmd/server/main.go
# NewLedgerService(billingRepo, txManager, quotaResolver, nil)
#                                                     ↑ pricing = nil
```

`GetActiveRule` (repo pembaca `pricing_rules`) juga **tidak pernah dipanggil**.

### Bukti uji live

```sql
SELECT entry_type, cost_idr, count(*) FROM token_ledger GROUP BY 1,2;
-- commit | 0 | 50    ← semua cost 0
-- commit | 45 | 5    ← hanya dari test
```

### Dampak

- `cost_idr` selalu **0** → biaya AI tidak terlacak.
- `base_per_1k_idr` & `markup` **tidak berpengaruh** sama sekali.
- Admin mengubah pricing rule → tidak ada efek.

---

## GAP-05 — 🟡 `max_categories`/`max_wallets` Tanpa Usage Resolver

**Severity:** Sedang

### Bukti

```go
// usage_resolvers.go:33-41
switch limitKey {
case "max_transactions":    return p.countTransactions
case "ai_tokens_per_cycle": return p.currentTokens
case "ocr_scans_per_cycle": return p.countVisionScans
default:                    return nil       // ← max_categories, max_wallets
}
```

Resolver nil → middleware membaca usage `0` → `0 >= limit` selalu false → **lolos**.

### Dampak

- Batas kategori (5/30/∞) & dompet (1/5/∞) **tidak ditegakkan**.
- User Free bisa membuat kategori/dompet tak terbatas.

---

## GAP-06 — 🟠 `pricing_rules` Tanpa Endpoint Admin

**Severity:** Tinggi

### Bukti

```bash
grep -rn "pricing_rules\|pricing" internal/admin/*.go | grep -v _test
# (kosong)
```

Tidak ada handler admin yang menyentuh `pricing_rules`. Satu-satunya cara mengubah
`base_per_1k_idr`/`markup` adalah SQL langsung (bypass audit).

### Dampak

- Harga/markup AI **tidak dinamis dari Super Admin** — melanggar permintaan Anda.
- Perubahan tidak tercatat di `audit_logs`.

---

## GAP-07 — 🟡 User Tidak Melihat Sisa Kuota

**Severity:** Sedang

### Bukti

```bash
grep -rn "kuota\|sisa.*token\|remaining" frontend/app/(dashboard)/user ...
# (kosong)
```

Halaman Langganan menampilkan **daftar limit** (dari plan), bukan **pemakaian real-time**.

### Dampak

- User tidak tahu seberapa dekat ke batas.
- Tidak ada peringatan dini saat kuota menipis.

---

## GAP-08 — 🟡 Error Kuota Tidak Dipetakan ke Toast Upsell

**Severity:** Sedang

### Bukti

```bash
grep -n "insufficient_quota\|notifyEntitlementError" frontend/hooks/useAiChat.ts
# (kosong)
```

`notifyEntitlementError` (`lib/entitlement-errors.ts`) hanya menangani
`feature_not_in_plan`/`limit_exceeded`/`limit_not_in_plan`. Error ledger
`429 insufficient_quota` tidak dipetakan → user melihat error generik tanpa CTA upgrade.

### Dampak

- Peluang upsell hilang saat user benar-benar kehabisan kuota.
- UX inkonsisten (403/402 punya toast, 429 tidak).

---

## GAP-09 — 🟠 `ApplyScheduledDowngrade` Tidak Pernah Dipanggil

**Severity:** Tinggi

### Bukti

```bash
grep -rn "ApplyScheduledDowngrade" --include=*.go . | grep -v _test
# hanya definisi di subscription_management.go — 0 pemanggil
```

Grace worker (`grace_worker.go`) & dunning worker **tidak** memanggilnya; tidak ada
scheduler lain.

### Dampak

- Downgrade yang dijadwalkan (`cancel_at_period_end` + `metadata.scheduled_plan_code`)
  **tidak akan pernah diterapkan** saat periode berakhir.
- User tetap di plan mahal tanpa batas → pendapatan hilang & hak tidak konsisten.

> Catatan: P1 (2026-10-13) sudah memperbaiki **persistensi** `metadata` di repo `Update`,
> sehingga `scheduled_plan_code` kini tersimpan. Tetapi **penerapannya** masih belum ada.

---

## GAP-10 — 🟠 Token Cleaner Tidak Di-Start

**Severity:** Tinggi

### Bukti

```bash
grep -rn "NewCleaner\|Cleaner" cmd/server/main.go
# (kosong)
```

`billing/cleaner.go` ada (auto-release reservasi basi), tetapi **tidak pernah
di-konstruksi/di-start**.

### Dampak

- Reservasi menggantung (proses crash setelah `Reserve`) **tidak** pernah di-release.
- `reserved_tokens` bisa "terkunci" permanen → kuota user berkurang selamanya.
- Perlu koreksi manual via `POST /admin/tokens/adjust`.

---

## GAP-11 — 🟠 Grace/Dunning Worker Di-Wire dengan `nil`

**Severity:** Tinggi

### Bukti

```go
// cmd/server/main.go:410-411
dunningWorker := payment.NewDunningWorker(paymentRepo, nil, time.Hour)
graceWorker := payment.NewGraceWorker(paymentRepo, settingsRepo, nil, 30*time.Minute)
//                                                                 ↑ setStatus = nil
```

`GraceWorker.RunOnce` memanggil `w.setSubscriptionStatus` hanya jika tidak nil — jadi
transisi status langganan (`grace` → `past_due`) **tidak terjadi**.

### Dampak

- Dunning H-3/H-1/H-0 (dunning worker `nil` = tidak ada notifikasi?) — perlu ditelusuri.
- Grace period tidak mengubah status langganan → user lapsed tetap punya akses penuh.
- `past_due` (soft-limit) tidak pernah tercapai.

---

## Analisis Race Condition (Lengkap)

### RC-1: Reserve konkuren (SAFE)

Mekanisme `SELECT ... FOR UPDATE` + cek `Available()` di dalam transaksi **benar**.
Dibuktikan `TestP08_001_TokenQuotaConcurrency`.

### RC-2: Commit/Release ganda (SAFE)

`IsReservationResolved` mencegah commit/release ganda. `ErrReservationAlreadyResolved`.

### RC-3: Idempotency key (SAFE)

`idempotency_key` UNIQUE + replay mengembalikan entri lama.

### RC-4: Chat + OCR konkuren (UNSAFE secara bisnis)

Karena OCR tidak menyentuh ledger, ia **melewati** semua penguncian. Pada sisa 1 token,
chat ambil 1 (habis) tetapi OCR tetap jalan.

### RC-5: Perubahan plan saat request berjalan (RENDAH RISIKO)

Entitlement di-resolve di awal request. Jika plan berubah di tengah request, request
berjalan memakai entitlement lama (satu request). Dapat diterima.

### RC-6: Cache stale lintas instance (SAFE)

Versi user/plan disimpan di Redis → semua instance melihat invalidasi. Dibuktikan
`resolver_test.go`.

### RC-7: Reservasi menggantung (UNSAFE — GAP-10)

Cleaner mati → tidak ada pemulihan otomatis.

---

## Verifikasi Ulang (Reproduksi Cepat)

```bash
# GAP-01: OCR tidak menulis ledger
# (dengan user Pro) POST /api/v1/ai/receipt × 4
docker exec fintrack_v2_db psql -U postgres -d fintrack \
  -c "SELECT modality,count(*) FROM token_ledger WHERE modality='vision' GROUP BY 1;"
# → (0 rows)

# GAP-02: chat flat 1 token
docker exec fintrack_v2_db psql -U postgres -d fintrack \
  -c "SELECT amount,count(*) FROM token_ledger WHERE modality='chat' AND entry_type='commit' GROUP BY 1;"
# → commit 1 (banyak)

# GAP-03/04/09/10: tidak ada pemanggil
cd backend
grep -rn "ai_requests_per_cycle\|NewPricingEngine\|ApplyScheduledDowngrade\|NewCleaner" --include=*.go . | grep -v _test
# → hanya definisi
```

---

## Kesimpulan

Mekanisme **inti** (lock, idempotency, cache berversi, fail-closed) **solid**. Yang
bermasalah adalah **kelengkapan cakupan**: beberapa operasi tidak dibebankan, beberapa
limit tidak ditegakkan, dan beberapa worker tidak di-start. Perbaikan bersifat
**aditif & terisolasi** — tidak perlu mengubah arsitektur.

Lanjut ke **[04 — Dinamisasi Admin](04-dinamisasi-admin-tanpa-hardcode.md)**.
