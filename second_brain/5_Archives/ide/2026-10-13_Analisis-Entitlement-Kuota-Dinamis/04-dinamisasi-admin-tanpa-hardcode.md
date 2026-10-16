# 04 — Dinamisasi Admin Tanpa Hardcode

| Field | Isi |
|---|---|
| **Dokumen** | Inventaris Hardcode & Desain "Semua Dinamis dari Super Admin" |
| **Tanggal** | 2026-10-13 |
| **Status** | Analisis final |

---

## 1. Prinsip (K-ENT-05)

> **Semua nilai yang memengaruhi hak akses, kuota, harga, dan markup WAJIB bersumber dari
> database dan dapat diubah dari portal Super Admin — tanpa deploy ulang, tanpa edit kode.**

Yang **boleh** tetap di kode (bukan "hardcode bisnis"):
- **Nama kunci** yang diinterpretasi sistem (`ai_tokens_per_cycle`, `max_transactions`, dst.)
  — karena kode harus tahu kunci mana yang dibaca untuk usage resolver.
- **Label terjemahan** di frontend (boleh, tetapi idealnya dari DB).
- **Default aman** saat DB kosong (fail-closed).

---

## 2. Inventaris: Sudah Dinamis vs Masih Hardcode

### 2.1 ✅ SUDAH dinamis dari Super Admin

| Entitas | Endpoint Admin | UI Admin |
|---|---|---|
| Plan (harga, periode, aktif, publik, gateway) | `GET/POST/PUT/DELETE /admin/plans` | `/super-admin/plans` |
| Fitur per plan (`plan_entitlements`) | via payload plan | `/super-admin/plans` (editor fitur) |
| Limit per plan (`plan_limits`) | via payload plan | `/super-admin/plans` (editor limit) |
| Subscription (extend/cancel/downgrade) | `/admin/subscriptions/*` | `/super-admin/subscriptions` |
| Token saldo & adjust | `/admin/tokens/*` | `/super-admin/tokens` |
| Gateway default | `PUT /admin/settings` | `/super-admin/settings` |
| Grace period, tax, invoice due | `PUT /admin/settings` | `/super-admin/settings` |
| Storage driver (local/R2) | `PUT /admin/settings` | `/super-admin/settings` |
| Feature flags | `/admin/feature-flags` | `/super-admin/settings` |
| Kill switch AI | `/admin/ai-gateway/kill-switch` | `/super-admin/ai-gateway` |
| AI endpoints & routes | `/admin/ai-gateway/*` | `/super-admin/ai-gateway` |

### 2.2 ❌ MASIH hardcode / tidak dinamis

| # | Item | Lokasi | Masalah |
|---|---|---|---|
| H-01 | **Pricing rules** (`base_per_1k_idr`, `markup_multiplier`, `markup_percent`, `modality_multipliers`) | `pricing_rules` table | Tidak ada endpoint admin; hanya bisa SQL |
| H-02 | **`ai_requests_per_cycle`** tidak ditegakkan | `usage_resolvers.go`, `router.go` | Nilai ada di DB tetapi tidak dipakai |
| H-03 | **`max_categories`/`max_wallets`** tanpa resolver | `usage_resolvers.go` | Nilai ada tetapi tidak dihitung |
| H-04 | **Label fitur/limit** di frontend | `PlanCard.tsx` `FEATURE_LABELS`/`LIMIT_LABELS` | Hardcode map; kunci baru tampil mentah |
| H-05 | **Katalog kunci** fitur/limit | tidak ada | Admin mengetik kunci manual (rawan typo) |
| H-06 | **"Unlimited" = 1e9** | `main.go` quotaResolver | Bukan `∞` sejati; angka sihir |
| H-07 | **Markup/kurs/rate limit AI** | env / kode | Tidak terlihat admin |
| H-08 | **Sisa kuota** tidak diekspos ke user | `/me/entitlements` | Hanya limit, tanpa usage |

---

## 3. Desain: Pricing Rules Dinamis (H-01)

### 3.1 Skema yang sudah ada

```sql
pricing_rules (
  id, rule_name, base_per_1k_idr, markup_multiplier, markup_percent,
  modality_multipliers JSONB,  -- {"chat":1.0,"voice":3.0,"vision":2.5,...}
  is_active, created_at, updated_at
)
```

Tabel **sudah ada**; yang kurang adalah **endpoint + UI + wiring**.

### 3.2 Endpoint baru yang diusulkan

| Method | Path | Fungsi |
|---|---|---|
| `GET` | `/api/v1/admin/pricing-rules` | Daftar rule |
| `GET` | `/api/v1/admin/pricing-rules/active` | Rule aktif |
| `POST` | `/api/v1/admin/pricing-rules` | Buat rule baru |
| `PUT` | `/api/v1/admin/pricing-rules/:id` | Ubah rule |
| `POST` | `/api/v1/admin/pricing-rules/:id/activate` | Jadikan aktif (deaktivasi yang lain) |
| `DELETE` | `/api/v1/admin/pricing-rules/:id` | Hapus (jika tidak aktif) |

**Aturan:**
- Semua mutasi → `audit_logs` (diff before/after).
- Aktivasi rule → invalidasi cache pricing (jika di-cache).
- Validasi: semua nilai ≥ 0; `modality_multipliers` hanya kunci modalitas yang dikenal.

### 3.3 Wiring pricing engine ke ledger

```go
// main.go — SEBELUM (pricing nil)
ledgerService := billing.NewLedgerService(billingRepo, txManager, quotaResolver, nil)

// SESUDAH — pricing aktif + refresh periodik
pricingRule, _ := billingRepo.GetActiveRule(bgCtx)
pricingEngine := billing.NewPricingEngine(pricingRule)
ledgerService := billing.NewLedgerService(billingRepo, txManager, quotaResolver, pricingEngine)
// + worker refresh rule tiap N menit, atau invalidate saat admin ubah rule
```

### 3.4 UI admin

Tab baru di `/super-admin/settings` (atau halaman `/super-admin/pricing`):
- Editor `base_per_1k_idr`, `markup_multiplier`, `markup_percent`.
- Editor matriks `modality_multipliers` (chat/vision/voice/reasoning/embedding).
- Preview biaya: "1.000 token chat = Rp X".

---

## 4. Desain: Katalog Kunci Fitur/Limit (H-05)

Masalah: admin mengetik `ai_chat` manual → rawan typo (`ai_chatt`) yang membuat fitur
tidak berfungsi.

### 4.1 Solusi: registry kunci (code-defined, admin-consumed)

Definisikan katalog kunci **di kode** (karena kode yang menginterpretasi), lalu ekspos
ke admin sebagai pilihan (bukan input bebas):

```go
// internal/entitlement/catalog.go
type FeatureKey struct { Key, Label, Description string }
type LimitKey struct { Key, Label, Unit string; Default int64 }

var FeatureCatalog = []FeatureKey{
  {"ai_chat", "Asisten AI (chat)", "..."},
  {"ocr_receipt", "Pindai struk (OCR)", "..."},
  // ...
}
var LimitCatalog = []LimitKey{
  {"ai_tokens_per_cycle", "Token AI / siklus", "token", 0},
  {"max_transactions", "Transaksi", "transaksi", 0},
  // ...
}
```

Endpoint: `GET /api/v1/admin/entitlement-catalog` → daftar kunci + label.
UI admin: dropdown dari katalog (bukan input bebas), sehingga typo tidak mungkin.

### 4.2 Label frontend (H-04)

Ganti `FEATURE_LABELS`/`LIMIT_LABELS` hardcode dengan data dari katalog
(`GET /plans` sudah bisa menyertakan `feature_labels`/`limit_labels`), atau endpoint
katalog publik.

---

## 5. Desain: Sisa Kuota ke User (H-08)

### 5.1 Perluas payload `/me/entitlements`

```json
{
  "plan_code": "pro",
  "features": { ... },
  "limits": { "ai_tokens_per_cycle": 150000, ... },
  "usage": {
    "ai_tokens_per_cycle": { "used": 1200, "limit": 150000, "remaining": 148800 },
    "max_transactions":    { "used": 42,   "limit": 5000,   "remaining": 4958 },
    "ocr_scans_per_cycle": { "used": 3,    "limit": 100,    "remaining": 97 }
  },
  "period": { "start": "...", "end": "..." }
}
```

> Perubahan ini **aditif** (tidak breaking) — klien lama tetap bekerja.

### 5.2 UI user

- Kartu "Pemakaian Kuota" di halaman Langganan (bar progres per limit).
- Badge peringatan saat > 80%.
- Toast upsell saat 429 (GAP-08).

---

## 6. Desain: "Unlimited" Sejati (H-06)

Saat ini `-1` → `1e9` di ledger. Alternatif lebih bersih:

- Ledger tetap butuh angka; gunakan konstanta jelas `UnlimitedQuota = 1<<62` atau
  pertahankan `1e9` tetapi **dokumentasikan** sebagai "praktis tak terbatas".
- Yang penting: **UI menampilkan `∞`** (sudah: `PlanCard` menampilkan "Tak terbatas").

---

## 7. Matriks Target Akhir

| Nilai | Sumber kebenaran | Dapat diubah admin? |
|---|---|---|
| Harga plan | `plans.price_monthly_idr` | ✅ |
| Fitur per plan | `plan_entitlements` | ✅ |
| Limit per plan | `plan_limits` | ✅ |
| **Base harga token** | `pricing_rules.base_per_1k_idr` | ⬜ (perlu endpoint) |
| **Markup** | `pricing_rules.markup_*` | ⬜ (perlu endpoint) |
| **Multiplier modalitas** | `pricing_rules.modality_multipliers` | ⬜ (perlu endpoint) |
| Gateway default | `settings` | ✅ |
| Grace period | `settings` | ✅ |
| Tax rate | `settings` | ✅ |
| Storage driver | `settings` | ✅ |
| Feature flags | `feature_flags` | ✅ |
| AI endpoints/routes | `ai_endpoints`/`ai_routes` | ✅ |
| **Katalog kunci** | kode (registry) → diekspos | ⬜ (perlu endpoint) |
| **Sisa kuota user** | `token_periods` | ⬜ (perlu ekspos) |

---

## 8. Checklist Dinamisasi

- [ ] **DYN-01** Endpoint CRUD `pricing_rules` + audit
- [ ] **DYN-02** Wiring `PricingEngine` ke `LedgerService`
- [ ] **DYN-03** UI admin editor pricing + preview biaya
- [ ] **DYN-04** Endpoint `GET /admin/entitlement-catalog` (katalog kunci)
- [ ] **DYN-05** UI admin: dropdown kunci dari katalog
- [ ] **DYN-06** Ekspos `usage` di `/me/entitlements`
- [ ] **DYN-07** UI user: kartu pemakaian kuota + peringatan 80%
- [ ] **DYN-08** Label fitur/limit dari katalog (hapus hardcode frontend)
- [ ] **DYN-09** Dokumentasikan konvensi "unlimited"
- [ ] **DYN-10** Refresh/invalidasi pricing rule saat admin ubah

Lanjut ke **[05 — Implementation Plan](05-implementation-plan.md)**.
