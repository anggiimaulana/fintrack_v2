# Spesifikasi Teknis: Rollout RBAC & Penegakan Entitlement Universal

## 1. Arsitektur Otorisasi FinTrack v2

Otorisasi pada FinTrack v2 memisahkan secara tegas antara **Role Platform** dan **Entitlement Paket**.
- **Role**: Hanya ada dua role di sistem: `super_admin` dan `user` (K-02). Role menentukan kapabilitas administratif tingkat sistem.
- **Entitlement**: Menentukan fitur dan batas kuota yang dapat digunakan oleh akun `user` berdasarkan paket langganan aktif (`Free`, `Pro`, `Enterprise`) atau add-on yang dibeli (K-12, K-13).

```
                 [ Incoming HTTP Request ]
                            │
                            ▼
              [ Auth Middleware (JWT Verify) ]
                            │
               ┌────────────┴────────────┐
               ▼                         ▼
      [ RequireRole(...) ]      [ RequireEntitlement(...) / RequireLimit(...) ]
               │                         │
               │                   [ Check Redis ]
               │             ft:prod:entitlement:user:{uid}
               │                         │
               │         ┌───────────────┴───────────────┐
               │         │ Cache Hit                     │ Cache Miss
               │         ▼                               ▼
               │   [ Evaluate JSON ]            [ Query DB Postgre ]
               │         │                      (Plans & Subscriptions)
               │         │                               │
               │         │                      [ Populate Redis Cache ]
               │         ▼                               │
               └─────────►───────[ Allow / Deny (403/402) ]
```

---

## 2. Struktur Middleware Backend Go

Middleware dibangun pada paket `internal/middleware/`:

### 2.1. `RequireRole(allowedRoles ...string)`
Memastikan pengguna memiliki role yang diizinkan.
```go
package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "UNAUTHORIZED",
				"message": "Sesi tidak valid",
			})
			return
		}

		userRole := role.(string)
		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "FORBIDDEN",
			"message": "Anda tidak memiliki hak akses administratif",
		})
	}
}
```

### 2.2. `RequireEntitlement(featureKey string)`
Memvalidasi apakah fitur bernilai boolean `true` pada paket aktif user.
```go
func RequireEntitlement(entitlementService EntitlementService, featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		allowed, err := entitlementService.HasFeature(c.Request.Context(), userID, featureKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR"})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "ENTITLEMENT_REQUIRED",
				"feature": featureKey,
				"message": "Fitur ini membutuhkan peningkatan paket langganan",
				"upgrade_url": "/settings/subscription",
			})
			return
		}

		c.Next()
	}
}
```

### 2.3. `RequireLimit(limitKey string, countCost int)`
Memvalidasi konsumsi batas numerik (misal kuota transaksi per siklus, kuota ekspor data).
```go
func RequireLimit(entitlementService EntitlementService, limitKey string, countCost int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		permitted, remaining, err := entitlementService.CheckAndConsumeLimit(c.Request.Context(), userID, limitKey, countCost)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "INTERNAL_ERROR"})
			return
		}

		if !permitted {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "LIMIT_EXCEEDED",
				"limit_key": limitKey,
				"remaining": remaining,
				"message": "Batas kuota penggunaan Anda telah habis untuk periode ini",
				"upgrade_url": "/settings/subscription",
			})
			return
		}

		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Next()
	}
}
```

---

## 3. Matriks Audit Endpoint Backend FinTrack v2 (Slice A — kondisi nyata)

> **Dikoreksi 2026-10-11.** Versi sebelumnya mendaftar 28 endpoint yang sebagian **tidak ada**
> di kode (`/budgets`, `/goals`, `/analytics/cashflow`, `/analytics/predict`,
> `/transactions/export`, `/ai/receipt-scan`, `/ai/token-balance`, `/user/profile`,
> `/realtime/stream`) dan menyebut **429** untuk limit habis, padahal kode memakai **402**.
> Tabel di bawah mencerminkan rute yang benar-benar terdaftar di
> `backend/internal/delivery/http/router.go`.

### 3.1 Kontrak blok

| Kondisi | Status | `error` | Catatan |
|---|---|---|---|
| Fitur tidak ada di paket | **403** | `feature_not_in_plan` | payload memuat `feature_key`, `upgrade_hint`, `upgrade_url` |
| Kuota habis | **402** | `limit_exceeded` | payload memuat `limit_key`, `limit_value`, `current_usage`, `upgrade_url` |
| Kunci limit absen di paket | **402** | `limit_not_in_plan` | fail-closed |

**402 dipertahankan (bukan 429)** karena rate limiter sudah memakai 429
(`internal/delivery/http/middleware/rate_limit.go`); memakai 429 untuk kuota paket akan membuat
"terlalu cepat" dan "kuota habis" tidak terbedakan bagi klien.

### 3.2 Rute yang ditegakkan (Slice A)

| Metode | Path | Middleware | Kunci |
|---|---|---|---|
| `POST` | `/api/v1/transactions` | `WithinLimit` | `max_transactions` |
| `POST` | `/api/v1/ai/chat` | `RequireFeature` + `WithinLimit` | `ai_chat` / `ai_tokens_per_cycle` |
| `POST` | `/api/v1/ai/receipt` | `RequireFeature` + `WithinLimit` | `ocr_receipt` / `ocr_scans_per_cycle` |
| `POST` | `/api/v1/ai/receipt/transaction` | `RequireFeature` | `ocr_receipt` |
| `POST` | `/api/v1/ai/voice` | `RequireFeature` | `voice_input` |
| `GET` | `/api/v1/reports` | `RequireFeature` | `reports_export` |

### 3.3 Rute yang sengaja TIDAK ditegakkan

| Kelompok | Alasan |
|---|---|
| Semua rute baca (`GET /transactions`, `/dashboard/*`, `/transactions/stats`, `/categories/*`, `/spk/*`, `/ai/sessions*`, `/providers`) | bukan fitur termeter; cache 30s dipertahankan |
| `POST /ai/sessions` + update/delete sesi | pembukuan sesi, bukan fitur berbayar |
| `POST /spk/calculate` | tidak ada kunci seed yang cocok; sudah rate-limited 30/menit |
| `POST /gemini-keys*` (BYOK) | dihapus pada slice D |

### 3.4 Kunci seed yang BELUM ditegakkan (jangan dikira selesai)

| Kunci | Alasan |
|---|---|
| `ai_requests_per_cycle` | belum ada penghitung request; `token_periods` hanya punya `used_tokens`/`reserved_tokens`, ledger hanya `reserve/commit/release/adjust` |
| `max_categories` | kategori dikelola super admin; tidak ada rute tulis kategori untuk user |
| `max_wallets` | fitur dompet belum ada |
| `rag_search` | tidak ada endpoint RAG; RAG menyatu di `/ai/chat` (sudah digate `ai_chat`) |
| `advanced_budgeting`, `api_access`, `bank_sync` | belum ada rute/fitur terkait |

### 3.5 Role

| Guard | Menerima | Dipakai di |
|---|---|---|
| `AuthMiddleware` | token valid | grup `protected` |
| `RequireUser` | `user`, `admin`, `super_admin` | rute user |
| `RequireAdmin` | `admin`, `super_admin` | grup `/admin` |
| `RequireSuperAdmin` | `super_admin` (re-check peran hidup, K-14) | grup `/admin` P06 |

> **Perbaikan 2026-10-11:** `RequireUser` sebelumnya **menolak `super_admin`**, sehingga pemilik
> platform terkunci dari seluruh rute user (transaksi, dashboard, AI, laporan). Kini menerima
> ketiga role, sejalan dengan `RequireAdmin`.

### 3.6 Perilaku tanpa baris subscription

`GetActivePlanForUser` (`internal/repository/postgres/entitlement_source.go`) kini:

1. **Tidak ada baris `subscriptions`** → fallback ke plan `DEFAULT_PLAN_CODE` (default `free`)
   dengan status sentinel `billing.StatusNone`. Tanpa ini, 50 dari 52 user dev (yang dibuat sebelum
   subscription ada) akan langsung 403/402 begitu enforcement dinyalakan.
2. **Status tidak memberi hak** (`canceled`, `expired`) → juga fallback ke plan default. Sebelumnya
   baris subscription dibaca **tanpa memfilter status**, sehingga langganan kedaluwarsa tetap
   mendapat fitur berbayar selamanya.
3. **Plan default hilang/tak terbaca** → deny-all (fail-closed) + log. Akses tidak pernah
   dilonggarkan saat error.

Status yang tetap memberi hak: `active`, `trialing`, `past_due`, `grace` (masa tenggang dunning).

### 3.7 Valve rollout

`ENTITLEMENT_ENFORCEMENT` (default **on**). Bila `off`/`false`/`0`, kedua middleware menjadi no-op
dan menulis satu log peringatan. Ini **bypass operasional**: hanya untuk insiden, jangan
dinyalakan di produksi dalam kondisi normal.

## 4. Mekanisme Invalidation Instan Tanpa Logout (K-14)

Ketika Super Admin mengubah paket langganan pengguna dari `Free` ke `Pro`:
1. Handler Admin di backend mengupdate baris `subscriptions` di PostgreSQL.
2. Backend menaikkan counter versi integer di Redis:
   ```bash
   INCR ft:prod:entitlement:ver:{user_id}
   ```
3. Backend menghapus cache payload entitlement lama:
   ```bash
   DEL ft:prod:entitlement:payload:{user_id}
   ```
4. Backend memancarkan SSE event `plan_updated` ke channel pengguna melalui Redis Pub/Sub.
5. Saat pengguna mengirim request berikutnya (misal 50ms kemudian):
   - Middleware memeriksa Redis `ft:prod:entitlement:payload:{user_id}`.
   - Karena cache kosong (*cache miss*), middleware membaca data terbaru dari PostgreSQL.
   - Hasil baru disimpan ke Redis dengan tag versi baru.
   - **Hasil**: Akses langsung aktif seketika tanpa perlu logout, token JWT tidak perlu di-refresh.
