# 01 — Implementation & Testing Plan

| Field | Isi |
|---|---|
| **Dokumen** | Rencana Implementasi & Pengujian lanjutan audit kuota/biaya |
| **Tanggal** | 2026-10-14 |
| **Status** | Siap eksekusi (P0 selesai) |
| **Referensi** | `00-audit-kuota-biaya.md` |

> **Cara pakai:** tandai `[x]` saat selesai. Jangan tandai tanpa bukti nyata.

---

## P0 — Perbaikan Kritis (SELESAI ✅)

### TASK-P0-01 — Double-count kuota OCR

- [x] `CountVisionScansInPeriod` memfilter `entry_type='reserve'`
- [x] Test regression: reserve-only + 2 scan → count 2
- [x] Live: 8 scan → tercatat 8 (sebelumnya 16)

### TASK-P0-02 — Kuota token negatif

- [x] `Commit` meng-clamp `actualTokens` ke `quota − used` + log peringatan
- [x] `/ai/receipt` di-gate `ai_tokens_per_cycle`
- [x] Test regression: commit 5.000 pada kuota 100 → used=100 (bukan negatif)
- [x] Live: token habis + OCR → **402** (bukan 200); `used` tetap 150000

**Gate kualitas P0:**
- [x] `go vet ./...` + `go build ./...` hijau
- [x] `go test ./internal/... ./pkg/...` hijau
- [x] Test integrasi baru lulus
- [x] Tidak ada regresi

---

## P1 — Konsistensi Gate AI

### TASK-P1-01 — Gate token untuk voice

**Target:** `internal/delivery/http/router.go`

**Langkah:** tambah `limitGate(config, "ai_tokens_per_cycle")` pada `POST /ai/voice`
(voice mengonsumsi token audio, harus dibebankan seperti chat & OCR).

**DoD:**
- [ ] `/ai/voice` ter-gate `ai_tokens_per_cycle`
- [ ] Uji: token habis → voice 402
- [ ] Uji: voice sukses → ledger `voice commit > 0`

### TASK-P1-02 — Audit ulang seluruh gate

- [ ] Semua rute yang mengonsumsi AI punya `ai_tokens_per_cycle`
- [ ] Semua rute yang memakai fitur punya `featureGate` yang sesuai
- [ ] Dokumentasikan matriks di dokumen ini

---

## P2 — Dinamisasi Biaya (Permintaan Utama)

### TASK-P2-01 — Satuan harga per 10k + harga beli provider

**Target:** `database/migrations/0017_pricing_cost_basis.sql`, `internal/billing/domain.go`,
`internal/billing/pricing.go`

**Langkah:**
1. Migrasi **aditif** (nullable/DEFAULT):
   ```sql
   ALTER TABLE pricing_rules
     ADD COLUMN IF NOT EXISTS provider_cost_per_1k_idr BIGINT NOT NULL DEFAULT 0,
     ADD COLUMN IF NOT EXISTS price_unit VARCHAR(8) NOT NULL DEFAULT 'per_1k';  -- 'per_1k' | 'per_10k'
   ```
2. `PricingRule` tambah field `ProviderCostPer1kIDR`, `PriceUnit`.
3. `CostIDR` menormalkan satuan: `per_10k` → `base / 10` per 1k, atau kalikan `tokens/10000`.
4. Tambah helper `MarginIDR(tokens, modality)` = harga jual − harga beli.

**DoD:**
- [ ] Migrasi replayable (maju & mundur)
- [ ] `base_per_1k_idr=300` ≡ `per_10k=3000` (uji kesetaraan)
- [ ] `MarginIDR` benar (uji: jual 3000, beli 2000 → margin 1000 per 10k)
- [ ] Tidak ada regresi biaya

### TASK-P2-02 — UI admin: satuan, harga beli, margin

**Target:** `app/(dashboard)/super-admin/pricing/page.tsx`, `hooks/useSuperAdminPricing.ts`

**DoD:**
- [ ] Pilihan satuan (per 1k / per 10k)
- [ ] Field harga beli provider
- [ ] Preview margin (Rp & %) per modalitas
- [ ] Preview "10.000 token chat = Rp X (margin Rp Y)"

### TASK-P2-03 — Harga dasar per-modalitas (opsional)

**DoD:**
- [ ] `modality_base_prices` JSONB (opsional override `base_per_1k`)
- [ ] Fallback ke `base_per_1k` bila kunci modalitas absen
- [ ] Uji: override vision ≠ chat

---

## P3 — Peningkatan UX & Akurasi

### TASK-P3-01 — Estimasi reserve lebih akurat

**Target:** `internal/delivery/http/handler/ai_handler.go`

**Langkah:** naikkan `estTokens` reserve (mis. 2.000 untuk vision, 1.000 untuk chat)
atau hitung dari ukuran input, agar over-run (clamp) jarang terjadi.

**DoD:**
- [ ] Over-run (log `token commit exceeds remaining quota`) jarang/tiada
- [ ] Kuota tetap tidak pernah negatif
- [ ] Uji: banyak scan berturut → tidak ada clamp

### TASK-P3-02 — Estimasi biaya sebelum aksi (opsional)

**DoD:**
- [ ] Endpoint `GET /me/ai-cost-estimate?modality=&tokens=` (atau di payload entitlements)
- [ ] UI menampilkan perkiraan biaya sebelum scan/chat
- [ ] Aditif (tidak breaking)

### TASK-P3-03 — Panel margin di Super Admin

**DoD:**
- [ ] Ringkasan: total token, total biaya beli, total biaya jual, margin
- [ ] Grafik margin per modalitas
- [ ] Filter periode

---

## Matriks Testing

| Test ID | Deskripsi | Fase | Status |
|---|---|---|---|
| T-P0-01 | Vision count = reserve-only (1 scan → 1) | P0 | ✅ |
| T-P0-02 | 2 scan lengkap → count 2 (bukan 4) | P0 | ✅ |
| T-P0-03 | Commit 5.000 pada kuota 100 → used=100, bukan negatif | P0 | ✅ |
| T-P0-04 | OCR saat token habis → 402 | P0 | ✅ |
| T-P1-01 | Voice saat token habis → 402 | P1 | ⬜ |
| T-P1-02 | Voice sukses → ledger voice commit > 0 | P1 | ⬜ |
| T-P2-01 | `per_10k` setara `per_1k` (3000 ≡ 300) | P2 | ⬜ |
| T-P2-02 | Margin = jual − beli | P2 | ⬜ |
| T-P2-03 | Migrasi replayable maju & mundur | P2 | ⬜ |
| T-P2-04 | Ubah satuan di UI → biaya ikut berubah live | P2 | ⬜ |
| T-P3-01 | Reserve akurat → tanpa clamp pada N scan | P3 | ⬜ |
| T-P3-02 | Estimasi biaya akurat vs `cost_idr` aktual | P3 | ⬜ |

---

## Gate Kualitas (setiap fase)

- [ ] `cd backend && go vet ./... && go build ./... && go test ./...` hijau
- [ ] `cd frontend && pnpm lint && pnpm test && pnpm build` hijau
- [ ] Tidak ada regresi P04–P08
- [ ] Mutasi pricing teraudit (`audit_logs`)
- [ ] Dokumentasi diperbarui

---

## Catatan Desain Penting

1. **Clamp vs tagih penuh**: saat over-run, sistem meng-clamp (tidak menagih melebihi kuota)
   dan mencatat log. Ini melindungi user; trade-off-nya sebagian biaya tidak tertagih.
   Naikkan estimasi reserve (P3-01) untuk meminimalkan.
2. **Uang = integer Rupiah** (K-27). Semua perhitungan margin integer, floor.
3. **Ledger append-only**: koreksi via `adjust`, bukan update.
4. **Pricing live**: perubahan berlaku tanpa restart (hook admin + refresh 60s).
