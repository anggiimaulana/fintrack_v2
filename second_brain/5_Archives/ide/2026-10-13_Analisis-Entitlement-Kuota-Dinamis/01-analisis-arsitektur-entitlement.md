# 01 — Analisis Arsitektur Entitlement vs RBAC

| Field | Isi |
|---|---|
| **Dokumen** | Arsitektur Entitlement, Resolusi, Cache, & Penegakan |
| **Tanggal** | 2026-10-13 |
| **Status** | Analisis final (kode + verifikasi live) |

---

## 1. Entitlement vs RBAC — Bukan Hal yang Sama

Pertanyaan Anda: *"apakah levelnya mirip seperti RBAC?"*

**Jawaban: mirip secara konsep (keduanya kontrol akses), tetapi berbeda mendasar pada
sumber kebenaran dan dinamika.**

| Aspek | **RBAC** (Role-Based) | **Entitlement** (Plan-Based) |
|---|---|---|
| Sumber | Kolom `users.role` | `subscriptions` → `plans` → `plan_entitlements` + `plan_limits` |
| Nilai | Tetap: `user`, `admin`, `super_admin` | Dinamis per-user, berubah saat plan berubah |
| Granularitas | Kasar (peran global) | Halus (per fitur & per limit numerik) |
| Dikontrol oleh | Admin/Super Admin (perubahan peran) | Langganan user + katalog plan Super Admin |
| Berubah tanpa re-login? | Ya (di-resolve ulang per request) | Ya (cache berversi + pub/sub) |
| Lokasi cek | `middleware/auth.go` + `rbac.go` | `middleware/entitlement.go` |
| Contoh | "boleh akses `/admin/*`?" | "boleh OCR? berapa kuota token?" |

**Kesimpulan:** keduanya **berlapis, bukan saling menggantikan**:

```
Request
  └─ AuthMiddleware        (siapa kamu? JWT valid?)        → 401 jika gagal
       └─ RequireUser/Admin (peran kamu apa? RBAC)          → 403 jika gagal
            └─ RequireFeature / WithinLimit (plan kamu apa? Entitlement) → 403/402 jika gagal
                 └─ Handler
```

> **RBAC = "siapa kamu", Entitlement = "kamu bayar apa".** Seorang `super_admin` tetap
> `super_admin` (RBAC), tetapi entitlement-nya di-bypass (K-ENT-03) sehingga bisa menguji
> semua fitur.

---

## 2. Model Data (Sumber Kebenaran)

### 2.1 Tabel kunci

```
plans                 (code, name, price_monthly_idr, billing_period_days, is_active, is_public, ...)
  ├─ plan_entitlements (plan_id, feature_key, is_enabled)      → fitur boolean
  └─ plan_limits       (plan_id, limit_key, limit_value)       → limit numerik (-1 = unlimited)

subscriptions         (user_id, plan_id, status, current_period_start/end, cancel_at_period_end, metadata)
  └─ token_periods    (user_id, period_start, period_end, quota_tokens, used_tokens, reserved_tokens)
       └─ token_ledger (period_id, user_id, entry_type, amount, modality, cost_idr, idempotency_key)

pricing_rules         (base_per_1k_idr, markup_multiplier, markup_percent, modality_multipliers)
settings              (payment_gateway_default, tax_rate_percent, grace_period_hours, storage_driver, ...)
```

### 2.2 Data seed aktual (dari DB live, 2026-10-13)

**`plans`:**

| code | price_monthly_idr | billing_period_days |
|---|---|---|
| free | 0 | 30 |
| pro | 39000 | 30 |
| business | 129000 | 30 |

**`plan_entitlements` (fitur):**

| feature_key | free | pro | business |
|---|:---:|:---:|:---:|
| `ai_chat` | ✅ | ✅ | ✅ |
| `advanced_budgeting` | ❌ | ✅ | ✅ |
| `api_access` | ❌ | ❌ | ✅ |
| `bank_sync` | ❌ | ❌ | ❌ |
| `ocr_receipt` | ❌ | ✅ | ✅ |
| `rag_search` | ❌ | ✅ | ✅ |
| `reports_export` | ❌ | ✅ | ✅ |
| `voice_input` | ❌ | ✅ | ✅ |

**`plan_limits` (kuota):**

| limit_key | free | pro | business |
|---|---:|---:|---:|
| `ai_requests_per_cycle` | 50 | 2000 | -1 (∞) |
| `ai_tokens_per_cycle` | 5000 | 150000 | 1000000 |
| `max_categories` | 5 | 30 | -1 (∞) |
| `max_transactions` | 100 | 5000 | -1 (∞) |
| `max_wallets` | 1 | 5 | -1 (∞) |
| `ocr_scans_per_cycle` | 0 | 100 | -1 (∞) |

> Inilah yang tampil di halaman Langganan Anda (Gambar #2). Semuanya **dinamis** dari DB.

---

## 3. Alur Resolusi Entitlement

### 3.1 Diagram

```
Resolve(userID)
  │
  ├─ 1. Baca versi user dari Redis: cache.ver:ent:{userID}
  │
  ├─ 2. Cek L1 (in-memory, TTL 60s) → HIT? & versi cocok? → return
  │
  ├─ 3. Cek L2 (Redis): entitlement:u:{userID}:v{version}
  │      └─ HIT? cek planVersion masih segar? → return
  │
  └─ 4. MISS → resolveFromSource()
         ├─ EntitlementSource.GetActivePlanForUser(userID)
         │    ├─ Ada subscription & status grants access? → pakai plan-nya
         │    ├─ Tidak ada subscription → fallback plan DEFAULT_PLAN_CODE (free)
         │    └─ Status lapsed (canceled/expired) → fallback free
         ├─ GetEntitlements(planID)  → map[feature]bool
         └─ GetLimits(planID)        → map[limit]int64
         → simpan ke L2 + L1 (dengan planVersion)
```

**Bukti kode:** `internal/entitlement/resolver.go:52-100` (`Resolve`),
`internal/repository/postgres/entitlement_source.go:49-90` (`GetActivePlanForUser`).

### 3.2 Fail-closed (K-ENT-02)

Dua jalur fail-closed yang benar:

```go
// resolver.go:88-95 — tanpa plan → deny-all
if plan == nil {
    return res, nil   // Features/Limits kosong → semua fitur false, limit 0
}
```
```go
// entitlement_source.go:79-83 — default plan hilang → deny-all
if errors.Is(err, billing.ErrPlanNotFound) {
    logger.Warn("default plan not found; denying entitlements", ...)
    return nil, "", nil
}
```

> **Penting:** jika `DEFAULT_PLAN_CODE` dikosongkan, user tanpa subscription akan
> **deny-all** (bukan free). Ini adalah valve operasional.

---

## 4. Cache Berversi (Propagasi Tanpa Re-login)

### 4.1 Mekanisme

Dua komponen versi independen (O(1) invalidation):

| Versi | Scope key | Di-bump saat |
|---|---|---|
| **User version** | `cache.ver:ent:{userID}` | Langganan user berubah (upgrade/downgrade/admin) |
| **Plan version** | `cache.ver:ent:plan:{planID}` | Entitlement/limit plan diedit admin |

**Bukti:** `resolver.go:141-160` (`Invalidate`, `InvalidatePlan`),
`resolver.go:110-116` (`planVersionCurrent` — mendeteksi edit plan pada read).

### 4.2 Jalur invalidasi

```
Admin edit plan
  └─ invalidatePlan(planID)  → cache.BumpVersion("ent:plan:{id}")
       └─ planVersionCurrent() = false pada semua subscriber → re-resolve ✅

Langganan berubah (webhook/approve/admin)
  └─ entBroadcaster.PublishInvalidate(userID)
       └─ BumpVersion("ent:{userID}") + Redis Pub/Sub lintas instance ✅
```

**Bukti:** `internal/entitlement/broadcaster.go:38-66`, `internal/admin/plans.go:332`.

### 4.3 Verifikasi live

Uji aktual: user diubah ke Pro via SQL → cache **tidak** otomatis (karena bypass app),
setelah cache di-flush/restart → entitlement langsung Pro. Ini mengonfirmasi:

- ✅ Cache berversi bekerja (invalidasi instan saat perubahan lewat jalur app).
- ⚠️ Perubahan **langsung ke DB** (bypass app) tidak memicu invalidasi — ini perilaku
  yang benar (semua mutasi harus lewat handler ber-audit), tetapi perlu dicatat sebagai
  batasan operasional.

---

## 5. Penegakan (Enforcement) & Isolasi Plan

### 5.1 Middleware

```go
// internal/delivery/http/middleware/entitlement.go
func (m *EntitlementMiddleware) RequireFeature(featureKey string) gin.HandlerFunc
func (m *EntitlementMiddleware) WithinLimit(limitKey string, usage UsageResolver) gin.HandlerFunc
```

- `RequireFeature`: `!res.Feature(key)` → **403** `feature_not_in_plan`.
- `WithinLimit`: `current >= limit` → **402** `limit_exceeded`; `limit == 0` (absen) →
  **402** `limit_not_in_plan` (fail-closed); `limit < 0` → lolos (unlimited).
- Bypass: `super_admin` + valve `ENTITLEMENT_ENFORCEMENT` (default on).

**Bukti:** `middleware/entitlement.go:89-172`.

### 5.2 Penerapan per-rute

| Rute | Feature gate | Limit gate |
|---|---|---|
| `POST /ai/chat` | `ai_chat` | `ai_tokens_per_cycle` |
| `POST /ai/receipt` | `ocr_receipt` | `ocr_scans_per_cycle` |
| `POST /ai/receipt/transaction` | `ocr_receipt` | — |
| `POST /ai/voice` | `voice_input` | — |
| `GET /reports` | `reports_export` | — |
| `POST /transactions` | — | `max_transactions` |

**Bukti:** `internal/delivery/http/router.go:204,213,236-253`.

### 5.3 Jawaban: bisakah plan rendah "menabrak" fitur plan tinggi?

**Tidak — dan inilah alasannya (berlapis):**

1. **Backend otoritatif (K-ENT-01).** Frontend hanya menyembunyikan tombol; API tetap
   memeriksa. User yang memanipulasi UI tetap ditolak.
2. **Fail-closed.** Fitur yang tidak ada di plan → `false` (bukan default allow).
3. **Resolusi berbasis plan aktif saja.** `GetActivePlanForUser` hanya mengembalikan plan
   dari subscription yang **status-nya grants access** (`active`/`trialing`/`grace`/`past_due`).
   Status `canceled`/`expired` → fallback free.
4. **Tidak ada "union" hak akses.** Entitlement adalah hasil plan **tunggal** (bukan
   gabungan plan lama + baru). Saat downgrade, hak plan lama langsung tidak berlaku.

**Bukti uji live:**

```
GET /api/v1/reports  (user Free)
→ HTTP 403 {"error":"feature_not_in_plan","feature_key":"reports_export",
            "upgrade_hint":true,"upgrade_url":"/user/settings/subscription"}
```

### 5.4 Skenario downgrade (soft-limit)

Saat user downgrade Pro → Free:
- Plan berubah pada `current_period_end` (terjadwal), **bukan** langsung.
- Sampai periode berakhir, hak Pro **tetap** (soft-limit, tidak ada data hilang).
- Setelah periode berakhir, resolver mengembalikan plan Free → fitur Pro otomatis hilang.

**Bukti:** `billing/subscription_management.go:124-190` (`ScheduleDowngrade`,
`ApplyScheduledDowngrade`).

> 🔴 **GAP TERKONFIRMASI:** `ApplyScheduledDowngrade` **tidak pernah dipanggil** di mana pun
> (grep seluruh kode non-test = 0 pemanggil). Akibatnya, downgrade yang dijadwalkan
> **tidak akan pernah benar-benar diterapkan** saat periode berakhir — user tetap di plan
> lama tanpa batas. Lihat GAP-09 di dokumen 03.

---

## 6. Race Condition — Analisis

### 6.1 Invarian yang dijaga

| Invarian | Mekanisme | Bukti |
|---|---|---|
| Tidak ada double-spend token | Row lock `SELECT ... FOR UPDATE` + cek `Available()` dalam transaksi | `ledger.go:96-118` |
| Tidak ada kuota negatif | `if period.Available() < estTokens { return ErrInsufficientQuota }` | `ledger.go:102-104` |
| Mutasi idempoten | `idempotency_key` UNIQUE + `ON CONFLICT DO NOTHING` | `ledger.go:82-91`, `billing_repo.go` |
| Commit/Release ganda | `IsReservationResolved` + `ErrReservationAlreadyResolved` | `ledger.go:140-148` |
| Cache tidak stale | Versi user + versi plan + pub/sub | `resolver.go` |

### 6.2 Analisis: 20 request paralel, sisa kuota 5

Skenario klasik (P08-G5). Alur per request:

```
Reserve(1 token):
  BEGIN
    SELECT ... FOR UPDATE  ← serialisasi antar request
    if Available() < 1 → ROLLBACK, ErrInsufficientQuota   (429)
    UPDATE reserved += 1
    INSERT ledger (reserve)
  COMMIT
```

Karena `FOR UPDATE` **menserialisasi** akses ke baris `token_periods`, request ke-6 dst
melihat `Available() == 0` → ditolak. Hasil: tepat 5 lolos, 15 ditolak.

**Bukti test:** `TestP08_001_TokenQuotaConcurrency` (20 request → 5 sukses, 15 ditolak
429, saldo akhir 0, tidak negatif).

### 6.3 Analisis: chat + voice + struk bersamaan pada kuota tipis

Ini pertanyaan spesifik Anda. Kondisi **saat ini**:

| Operasi | Menyentuh ledger? | Efek pada kuota |
|---|---|---|
| Chat | ✅ Reserve/Commit **1 token** | Kuota turun 1 (bukan token asli) |
| OCR/struk | ❌ **Tidak** | Kuota **tidak** turun (GAP-01) |
| Voice | ❌ **Tidak** | Kuota **tidak** turun |

**Konsekuensi pada kuota tipis:**

1. Chat A dan Chat B bersamaan → keduanya `Reserve(1)`. Row lock membuat salah satu
   menunggu; keduanya berhasil jika sisa ≥ 2. **Aman** (tidak over-spend).
2. Chat + OCR bersamaan pada sisa 1 token → chat ambil 1, OCR **tidak** menyentuh kuota →
   OCR lolos meski kuota habis. **Tidak aman secara bisnis** (OCR tak dibatasi).
3. Tiga chat bersamaan pada sisa 2 → 2 sukses, 1 ditolak 429. **Aman.**

**Kesimpulan:** mekanisme lock-nya **benar** (tidak ada double-spend), tetapi **cakupan
akuntansinya tidak lengkap** (OCR & voice tidak dibebankan), sehingga "kuota habis"
tidak benar-benar menghentikan semua konsumsi AI.

### 6.4 Race pada reserve→call→commit (in-flight)

```
T1: Reserve(1) → reserved=1
T2: (AI call berjalan 30 detik)
T3: Commit(actual) → used += actual, reserved -= 1
```

Selama T2, `Available()` = `quota - used - reserved` sudah dikurangi 1. Jadi request
lain **tidak** bisa memakai token yang sedang di-reserve. Ini benar.

**Tapi** karena `actual` selalu di-commit sebagai `1` (GAP-02), token nyata yang
dikonsumsi (bisa 500-2000 per pesan) **tidak pernah** dibebankan → kuota bocor besar.

### 6.5 Reservasi menggantung (crash saat AI call)

Jika proses mati setelah `Reserve` tanpa `Commit`/`Release`, token tetap "terkunci" di
`reserved`. Solusi yang ada: `billing/cleaner.go` (auto-release reservasi basi).

**Bukti:** `internal/billing/cleaner.go` — sweep `ListExpiredReservations` → `Release`.

> 🔴 **GAP TERKONFIRMASI:** cleaner **tidak di-start** di `cmd/server/main.go` (grep
> `NewCleaner` = 0 hasil). Reservasi menggantung akibat crash **tidak akan** pernah
> di-release otomatis → kuota user bisa "terkunci" permanen. Lihat GAP-10 di dokumen 03.

---

## 7. Kesimpulan Bagian Ini

| Pertanyaan | Jawaban |
|---|---|
| Apakah entitlement mirip RBAC? | Konsep serupa, tetapi independen & berlapis; RBAC = peran, entitlement = plan |
| Bisakah plan rendah menabrak fitur plan tinggi? | **Tidak** (backend otoritatif, fail-closed, plan tunggal) |
| Bagaimana race condition kuota? | Lock benar (no double-spend), tetapi **cakupan akuntansi tidak lengkap** (OCR/voice tak dibebankan) |
| Bagaimana penanganannya? | Perlu: bebankan OCR/voice ke ledger, commit token asli, tegakkan `ai_requests_per_cycle` |

Lanjut ke **[02 — Kalkulasi Kuota & Ledger](02-kalkulasi-kuota-dan-ledger.md)** untuk
detail rumus dan skenario kuota menipis.
