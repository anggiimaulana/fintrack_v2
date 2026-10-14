# 00 — Audit Kuota, Biaya, & Dinamisasi Pricing

| Field | Isi |
|---|---|
| **Dokumen** | Audit Ledger, OCR, RBAC, Interaksi Kuota, & Dinamisasi Biaya |
| **Tanggal** | 2026-10-14 |
| **Status** | Temuan + perbaikan kritis + rencana |
| **Referensi** | Lanjutan `2026-10-13_Analisis-Entitlement-Kuota-Dinamis` |

---

## Ringkasan Eksekutif

Pertanyaan yang diajukan membuka **dua bug kritis** yang merugikan user:

| ID | Temuan | Severity | Status |
|---|---|---|---|
| **BUG-OCR-01** | `ocr_scans_per_cycle` menghitung **2× per scan** (reserve + commit) | 🔴 Kritis | ✅ **Diperbaiki** |
| **BUG-NEG-01** | OCR bisa membuat **kuota token NEGATIF** (tidak ada gate token + tanpa clamp commit) | 🔴 Kritis | ✅ **Diperbaiki** |
| **VER-01** | Scan di halaman Transaksi vs AI Chat → **endpoint sama**, konsisten | ✅ | Terverifikasi |
| **VER-02** | RBAC/entitlement plan di semua rute | ✅ | Terverifikasi |
| **GAP-PRC-01** | Biaya per **10.000** token belum didukung (hanya per 1.000) | 🟠 Tinggi | ⬜ Rencana |
| **GAP-PRC-02** | Markup hanya `×` + `%` (sudah ada), belum ada **per-modality price** terpisah | 🟡 Sedang | ⬜ Rencana |

---

## 1. Audit: Semua Query `token_ledger` (filter `entry_type`)

Anda meminta pengecekan menyeluruh terhadap pola bug double-count. Hasil audit **16 lokasi**:

| Lokasi | Filter | Verdict |
|---|---|---|
| `admin/metrics.go:101` (tokens 24h) | `entry_type='commit'` | ✅ |
| `admin/metrics.go:106` (tokens prev 24h) | `entry_type='commit'` | ✅ |
| `admin/metrics.go:224` (timeseries) | `entry_type='commit'` | ✅ |
| `admin/quota_monitor.go:57` (requests) | `entry_type='reserve'` | ✅ |
| `admin/tokens.go:124/132` (list ledger) | filter `$1::text='' OR entry_type=$1` | ✅ (by design) |
| `admin/tokens.go:254` (adjust replay) | by `idempotency_key` | ✅ |
| `billing_repo.go:288` (ai requests) | `entry_type='reserve'` | ✅ |
| `billing_repo.go:312` (**vision scans**) | ~~tanpa filter~~ → `entry_type='reserve'` | ✅ **FIXED** |
| `billing_repo.go:346/362` (by key/id) | by PK/unique | ✅ |
| `billing_repo.go:377` (`IsReservationResolved`) | `entry_type IN ('commit','release')` | ✅ |
| `billing_repo.go:387` (`ListExpiredReservations`) | `entry_type='reserve'` + NOT EXISTS resolved | ✅ |

**Kesimpulan:** hanya **satu** lokasi bermasalah (`CountVisionScansInPeriod`) — sudah diperbaiki. Tidak ada double-count lain.

---

## 2. BUG-OCR-01 — Double Count Kuota OCR (FIXED)

### Gejala
8 scan sebenarnya → tercatat **16** di `ocr_scans_per_cycle`.

### Akar masalah
Setiap OCR menulis **2 baris** ledger: `reserve` (amount=1) + `commit` (amount=token asli).
Query lama menghitung **semua** baris bermodality vision:

```sql
-- SEBELUM (salah): 2 baris per scan
WHERE l.modality = 'vision'
-- SESUDAH (benar): 1 baris per scan
WHERE l.modality = 'vision' AND l.entry_type = 'reserve'
```

### Dampak
Pengguna Pro (limit 100) hanya bisa scan **~50×** — separuh dari yang dijanjikan.

### Perbaikan
`CountVisionScansInPeriod` kini memfilter `entry_type='reserve'` (konsisten dengan
`CountAIRequestsInPeriod`). Test dijadikan regression guard (reserve-only + skenario 2 scan → 2).

---

## 3. BUG-NEG-01 — Kuota Token Bisa NEGATIF (FIXED)

### Gejala (dibuktikan live)
```
Kuota: 150.000 | Terpakai: 149.900 (sisa 100)
→ 1× OCR (tanpa gate token) → used_tokens = 154.474  ← NEGATIF (-4.474)
```

### Akar masalah — **dua lapis**

**Lapis 1: OCR tidak ter-gate `ai_tokens_per_cycle`.**
```
POST /ai/chat    → featureGate(ai_chat) + limitGate(ai_requests) + limitGate(ai_tokens)  ✅
POST /ai/receipt → featureGate(ocr_receipt) + limitGate(ai_requests) + limitGate(ocr_scans)  ❌ tanpa ai_tokens
```
OCR mengonsumsi token vision, tetapi tidak diperiksa terhadap anggaran token.

**Lapis 2: `Commit` tidak meng-clamp terhadap sisa kuota.**
Reserve = **estimasi 1 token** (murah). Commit = **token asli** (mis. 4.600). Tidak ada
penjagaan, sehingga `used_tokens` melewati `quota_tokens`.

```
reserve 1  → reserved=1, available-=1
commit 4600 → used += 4600, reserved -= 1     ← melewati kuota!
```

### Dampak
- **User dirugikan**: kuota tampil negatif (sisa -4.474), membingungkan & tidak adil.
- Pelanggaran invarian: "saldo tidak boleh negatif".

### Perbaikan (dua lapis)
1. **Gate token untuk OCR**: tambah `limitGate(config, "ai_tokens_per_cycle")` pada `/ai/receipt`.
2. **Clamp commit**: `Commit` membatasi `actualTokens` ke `quota − used` yang tersisa, dan
   **mencatat peringatan** bila over-run (agar estimasi reserve bisa disetel).

```go
// billing/ledger.go — Commit
if maxCharge := period.QuotaTokens - period.UsedTokens; actualTokens > maxCharge {
    logger.Warn("token commit exceeds remaining quota; clamping", ...)
    actualTokens = max(0, maxCharge)
}
```

### Bukti perbaikan (live)
```
token HABIS (150000/150000), OCR sisa banyak
POST /ai/receipt → 402 limit_exceeded (limit_key=ai_tokens_per_cycle)  ✅
used_tokens tetap 150000 (TIDAK negatif)  ✅
```

> **Catatan desain:** clamping berarti sebagian biaya token "hilang" saat over-run. Ini
> disengaja (tidak boleh menagih melebihi kuota) dan dicatat di log untuk penyesuaian
> estimasi reserve. Alternatif (menagih penuh + kuota negatif) ditolak karena merugikan user.

---

## 4. VER-01 — Scan Halaman Transaksi vs AI Chat

### Temuan: **konsisten** ✅

Kedua jalur memakai **satu endpoint** yang sama, sehingga ledger & RBAC identik:

| Jalur UI | Komponen | Endpoint |
|---|---|---|
| Halaman **Transaksi** | `ReceiptScanner` (dialog "Pindai Struk") | `POST /api/v1/ai/receipt` |
| **AI Chat** | `ReceiptUpload` (via `ChatInput`) | `POST /api/v1/ai/receipt` |
| Transaksi (inline) | `InlineAiAssistant` | `POST /api/v1/ai/receipt` |

Tidak ada endpoint OCR alternatif di `router.go` (hanya `/ai/receipt` + `/ai/receipt/transaction`).

### Bukti live
2 scan dari 2 jalur berbeda → `ocr_scans_per_cycle` **9→11** (tepat +2), token +9.272,
requests +2. Keduanya dibebankan **sama**.

### Catatan
- Jalur "buat transaksi dari struk" (`/ai/receipt/transaction`) **tidak** memanggil AI lagi —
  hanya menyimpan transaksi dari hasil yang sudah dianalisis. Gate-nya hanya `ocr_receipt`
  (benar, karena tidak ada konsumsi AI tambahan).

---

## 5. VER-02 — RBAC & Entitlement Plan per Rute

### Matriks penegakan (live-verified)

| Rute | RBAC | Feature gate | Limit gate |
|---|---|---|---|
| `POST /ai/chat` | RequireUser | `ai_chat` | `ai_requests_per_cycle`, `ai_tokens_per_cycle` |
| `POST /ai/receipt` | RequireUser | `ocr_receipt` | `ai_requests_per_cycle`, `ocr_scans_per_cycle`, **`ai_tokens_per_cycle`** (baru) |
| `POST /ai/receipt/transaction` | RequireUser | `ocr_receipt` | — |
| `POST /ai/voice` | RequireUser | `voice_input` | `ai_requests_per_cycle` |
| `GET /reports` | RequireUser | `reports_export` | — |
| `POST /transactions` | RequireUser | — | `max_transactions` |
| `/admin/*` | RequireAdmin / RequireSuperAdmin | — | — |

### Bukti live (user Free)
```
GET /reports          → 403 feature_not_in_plan   ✅
POST /ai/receipt      → 403 feature_not_in_plan   ✅
POST /admin/plans     → 403 (user) / 401 (anon)   ✅
```

### Catatan
- `/ai/voice` **tidak** ter-gate `ai_tokens_per_cycle` (voice membebankan token audio).
  **Rekomendasi:** tambahkan, agar konsisten dengan chat & OCR (lihat rencana).
- `/ai/receipt/transaction` tidak butuh limit AI (tidak memanggil AI).

---

## 6. GAP-PRC-01/02 — Dinamisasi Biaya (Permintaan Anda)

### Yang Anda minta
> "biaya per 10k token (dari provider) = 3k penggunaan di langganan, ada markup/margin
> berdasarkan kali dan persenan"

### Yang SUDAH ada

Rumus saat ini (`billing/pricing.go`):
```
cost = floor( base_per_1k_idr × (tokens/1000) × markup_multiplier × (1 + markup_percent/100) × modality_multiplier )
```

| Kebutuhan Anda | Status |
|---|---|
| Harga dasar per token | ✅ Ada — tetapi **per 1.000**, bukan per 10.000 |
| Markup **kali** (×) | ✅ `markup_multiplier` |
| Markup **persen** (%) | ✅ `markup_percent` |
| Pengali per modalitas | ✅ `modality_multipliers` (chat/vision/voice/reasoning/embedding) |
| Diatur dari Super Admin | ✅ `/super-admin/pricing` (CRUD + preview biaya) |
| Berlaku live tanpa restart | ✅ hook admin + refresh 60s |

### Contoh perhitungan (per 1.000 saat ini)
`base=300, markup=1.25, modality=1.0` → 1.000 token chat = `floor(300×1×1.25×1.0)` = **375**

### Yang BELUM (gap)

| # | Gap | Usulan |
|---|---|---|
| **GAP-PRC-01** | Satuan hanya **per 1.000** token | Tambah pilihan satuan (`per_1k`/`per_10k`) atau kolom `price_per_10k_idr` |
| **GAP-PRC-02** | Harga **per-modalitas** hanya via pengali, bukan harga dasar terpisah | Opsional: `modality_base_prices` JSONB |
| **GAP-PRC-03** | Tidak ada "harga jual per 10k" vs "harga beli provider" (margin eksplisit) | Tambah `provider_cost_per_1k_idr` + turunkan margin otomatis |

**Interpretasi permintaan Anda:** "biaya per 10k token dari provider = 3k penggunaan di
langganan" berarti:
- **Harga beli provider**: Rp X per 10.000 token
- **Harga jual ke user (langganan)**: Rp 3.000 per 10.000 token
- **Margin** = harga jual − harga beli, atau markup `×`/`%` di atas harga beli

Dengan rumus sekarang, ini **sudah bisa** diekspresikan sebagai:
```
base_per_1k_idr = 300   → setara Rp 3.000 / 10.000 token  ✅
markup_multiplier = 1.25 (margin 25%)
```
Namun **satuan 10k tidak eksplisit**, dan **harga beli provider tidak tercatat** — sehingga
margin tidak bisa dihitung/ditampilkan.

---

## 7. Kesimpulan & Prioritas

### Sudah diperbaiki (kritis)
1. ✅ Double-count kuota OCR (`entry_type='reserve'`)
2. ✅ Kuota token negatif (gate `ai_tokens_per_cycle` untuk OCR + clamp commit)
3. ✅ Regression test untuk keduanya

### Rekomendasi lanjutan
| # | Item | Prioritas |
|---|---|---|
| R1 | Gate `ai_tokens_per_cycle` untuk `/ai/voice` | 🟠 Tinggi |
| R2 | Satuan harga **per 10k** + harga beli provider + tampilan margin | 🟠 Tinggi |
| R3 | Tampilkan **estimasi biaya** sebelum aksi AI ke user (opsional) | 🟡 Sedang |
| R4 | Naikkan `estTokens` reserve agar lebih dekat ke token asli (kurangi over-run) | 🟡 Sedang |

Lanjut ke **[01 — Implementation & Testing Plan](01-implementation-testing-plan.md)**.
