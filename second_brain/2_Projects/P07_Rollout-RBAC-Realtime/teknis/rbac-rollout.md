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
               └─────────►───────[ Allow / Deny (403/429) ]
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

## 3. Matriks Audit 28 Endpoint Backend FinTrack v2

Tabel berikut mendokumentasikan hasil audit kepatuhan seluruh rute API backend, status lama versus perlindungan baru fase P07:

| No | Metode | Path Endpoint | Proteksi Lama (P01) | Proteksi Baru (P07) | Entitlement / Role Requirement |
|---|---|---|---|---|---|
| 1 | `POST` | `/api/v1/auth/register` | Publik | Publik | Publik (Rate limit IP) |
| 2 | `POST` | `/api/v1/auth/login` | Publik | Publik | Publik (Rate limit IP) |
| 3 | `POST` | `/api/v1/auth/refresh` | Cookie Auth | Cookie Auth | Auth Only |
| 4 | `POST` | `/api/v1/auth/logout` | Cookie Auth | Cookie Auth | Auth Only |
| 5 | `GET` | `/api/v1/user/profile` | Auth | Auth | Any Role |
| 6 | `PUT` | `/api/v1/user/profile` | Auth | Auth | Any Role |
| 7 | `GET` | `/api/v1/transactions` | Auth | Auth + Entitlement | Limit: `monthly_transactions` read |
| 8 | `POST` | `/api/v1/transactions` | Auth | Auth + RequireLimit | Limit: `monthly_transactions` (Free: 100/bln) |
| 9 | `GET` | `/api/v1/transactions/:id` | Auth | Auth (Ownership) | Any Role + Owner Check |
| 10 | `PUT` | `/api/v1/transactions/:id` | Auth | Auth (Ownership) | Any Role + Owner Check + Lock 409 |
| 11 | `DELETE`| `/api/v1/transactions/:id` | Auth | Auth (Ownership) | Any Role + Owner Check |
| 12 | `POST` | `/api/v1/transactions/export` | Auth | Auth + RequireEntitlement | Feature: `export_csv_excel` (Pro/Enterprise) |
| 13 | `GET` | `/api/v1/budgets` | Auth | Auth | Any Role |
| 14 | `POST` | `/api/v1/budgets` | Auth | Auth + RequireLimit | Limit: `max_budgets` (Free: 3, Pro: Unlimited) |
| 15 | `PUT` | `/api/v1/budgets/:id` | Auth | Auth (Ownership) | Any Role + Owner Check |
| 16 | `DELETE`| `/api/v1/budgets/:id` | Auth | Auth (Ownership) | Any Role + Owner Check |
| 17 | `GET` | `/api/v1/goals` | Auth | Auth | Any Role |
| 18 | `POST` | `/api/v1/goals` | Auth | Auth + RequireLimit | Limit: `max_financial_goals` (Free: 2, Pro: Unl) |
| 19 | `GET` | `/api/v1/analytics/cashflow`| Auth | Auth | Any Role (Basic Chart) |
| 20 | `GET` | `/api/v1/analytics/predict` | Auth | Auth + RequireEntitlement | Feature: `advanced_analytics_ml` (Pro) |
| 21 | `POST` | `/api/v1/ai/chat` | Auth + BYOK Key | Auth + Token Gate | Feature: `ai_advisor` + Token Balance Check |
| 22 | `POST` | `/api/v1/ai/receipt-scan`| Auth + BYOK Key | Auth + Token Gate | Feature: `ai_receipt_ocr` + Token Balance |
| 23 | `GET` | `/api/v1/ai/token-balance`| N/A (Belum Ada) | Auth | Any Role |
| 24 | `GET` | `/api/v1/realtime/stream` | N/A (Belum Ada) | Auth (Cookie) | Any Role |
| 25 | `GET` | `/api/v1/admin/users` | Auth (Role Admin) | Auth + RequireRole | Role: `super_admin` Only |
| 26 | `POST` | `/api/v1/admin/plans` | N/A (Belum Ada) | Auth + RequireRole | Role: `super_admin` Only |
| 27 | `POST` | `/api/v1/admin/ai/endpoints`| N/A (Belum Ada) | Auth + RequireRole | Role: `super_admin` Only |
| 28 | `POST` | `/api/v1/payments/webhook` | Publik | Header Signature Auth | Validasi HMAC signature Midtrans/Xendit |

---

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
