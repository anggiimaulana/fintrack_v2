# Middleware Penegakan Hak Akses (Enforcement Middleware) — FinTrack v2

Dokumen ini mendokumentasikan implementasi middleware Gin HTTP pada paket `internal/entitlement/middleware/` yang bertugas menegakkan hak akses fitur dan batasan kuota numerik secara otoritatif di sisi backend.

---

## 1. Prinsip Desain Middleware (K-13, K-14, K-P04-08)

1. **Backend Merupakan Titik Otoritas Mutlak:** Frontend hanya mencerminkan kondisi hak akses untuk kepentingan antarmuka. Middleware HTTP memvalidasi setiap pemanggilan rute privat.
2. **Bypass Role Super Admin:** Pengguna dengan peran `super_admin` secara otomatis melewati seluruh batasan fitur dan limit tanpa dipungut kuota atau diblokir.
3. **Eksekusi Ultra Cepat:** Evaluasi hak akses menggunakan cache terdistribusi dan in-memory berversi, menghasilkan latensi rata-rata $< 1\text{ ms}$ pada kondisi cache hit.
4. **Format Kesalahan Terstruktur:** Response penolakan memberikan kode kesalahan yang jelas, nama fitur/limit yang terkunci, nilai kuota, serta URL penawaran upgrade untuk memudahkan konsumsi UI di frontend.

---

## 2. Middleware `RequireFeature(featureKey string)`

Middleware ini memeriksa apakah pengguna berhak mengakses fitur boolean tertentu berdasarkan paket aktifnya.

### 2.1. Alur Logika Eksekusi
1. Ambil entitas `UserContext` dari konteks Gin (`c.MustGet("user")`).
2. Jika `user.Role == "super_admin"`, panggil `c.Next()` dan keluar.
3. Ambil `UserEntitlements` dari `VersionedCacheManager`.
4. Periksa apakah `entitlements.Features[featureKey] == true`.
5. Jika `true`, lanjutkan request (`c.Next()`).
6. Jika `false`, hentikan request dan kembalikan HTTP `403 Forbidden`.

### 2.2. Implementasi Go
```go
package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "fintrack_v2/internal/entitlement/service"
)

func RequireFeature(entitlementService service.EntitlementService, featureKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Dapatkan konteks pengguna terautentikasi
        userVal, exists := c.Get("currentUser")
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": gin.H{
                    "code":    "UNAUTHORIZED",
                    "message": "Autentikasi diperlukan.",
                },
            })
            return
        }
        user := userVal.(AuthenticatedUser)

        // 2. Bypass Super Admin
        if user.Role == "super_admin" {
            c.Next()
            return
        }

        // 3. Evaluasi hak akses dari cache berversi
        hasAccess, err := entitlementService.HasFeature(c.Request.Context(), user.ID, featureKey)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
                "error": gin.H{
                    "code":    "ENTITLEMENT_CHECK_FAILED",
                    "message": "Gagal memverifikasi hak akses pengguna.",
                },
            })
            return
        }

        if !hasAccess {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": gin.H{
                    "code":        "FEATURE_NOT_AVAILABLE",
                    "message":     "Fitur ini tidak tersedia pada paket langganan Anda saat ini.",
                    "feature":     featureKey,
                    "upgrade_url": "/pricing",
                },
            })
            return
        }

        c.Next()
    }
}
```

---

## 3. Middleware `WithinLimit(limitKey string, currentUsageResolver func(c *gin.Context) (int64, error))`

Middleware ini memvalidasi bahwa jumlah entitas yang dimiliki atau pemakaian pengguna masih berada di dalam kuota paket sebelum operasi pembuatan data baru diizinkan.

### 3.1. Alur Logika Eksekusi
1. Ambil entitas pengguna terautentikasi.
2. Jika peran `super_admin`, lewati pemeriksaan.
3. Ambil nilai limit yang diizinkan untuk pengguna dari cache (`limitValue`).
4. Jika `limitValue == -1` (unlimited), lanjutkan request (`c.Next()`).
5. Eksekusi `currentUsageResolver(c)` untuk menghitung pemakaian saat ini dari basis data (misal: hitung jumlah dompet aktif yang ada).
6. Jika `currentUsage >= limitValue`, hentikan request dengan HTTP `429 Too Many Requests`.

### 3.2. Implementasi Go
```go
package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "fintrack_v2/internal/entitlement/service"
)

type UsageResolver func(c *gin.Context) (int64, error)

func WithinLimit(entitlementService service.EntitlementService, limitKey string, resolver UsageResolver) gin.HandlerFunc {
    return func(c *gin.Context) {
        userVal, exists := c.Get("currentUser")
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }
        user := userVal.(AuthenticatedUser)

        if user.Role == "super_admin" {
            c.Next()
            return
        }

        // 1. Ambil batasan limit dari cache berversi
        limitValue, err := entitlementService.GetLimit(c.Request.Context(), user.ID, limitKey)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa kuota pengguna"})
            return
        }

        // -1 menandakan kuota tak terbatas
        if limitValue == -1 {
            c.Next()
            return
        }

        // 2. Hitung penggunaan saat ini
        currentUsage, err := resolver(c)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung penggunaan sumber daya"})
            return
        }

        // 3. Evaluasi soft-limit
        if currentUsage >= limitValue {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": gin.H{
                    "code":        "LIMIT_EXCEEDED",
                    "message":     "Anda telah mencapai batas maksimum kuota untuk paket Anda.",
                    "limit_key":   limitKey,
                    "current":     currentUsage,
                    "max_allowed": limitValue,
                    "upgrade_url": "/pricing",
                },
            })
            return
        }

        c.Next()
    }
}
```

---

## 4. Contoh Penggunaan pada Pendaftaran Rute Gin

Berikut contoh integrasi middleware pada rute aplikasi di `internal/api/routes.go`:

```go
func SetupRoutes(r *gin.Engine, entSvc service.EntitlementService, walletRepo wallet.Repository) {
    v1 := r.Group("/api/v1")
    v1.Use(authMiddleware.RequireAuth())
    {
        // 1. Proteksi Fitur Boolean
        v1.POST("/receipts/ocr", 
            middleware.RequireFeature(entSvc, "ocr_receipt"), 
            ocrHandler.ProcessReceipt,
        )

        v1.GET("/reports/export-pdf", 
            middleware.RequireFeature(entSvc, "export_pdf"), 
            reportHandler.ExportPDF,
        )

        v1.POST("/sync/bank", 
            middleware.RequireFeature(entSvc, "bank_sync"), 
            bankHandler.InitiateSync,
        )

        // 2. Proteksi Batasan Kuota Numerik
        v1.POST("/wallets", 
            middleware.WithinLimit(entSvc, "max_wallets", func(c *gin.Context) (int64, error) {
                user := c.MustGet("currentUser").(AuthenticatedUser)
                return walletRepo.CountActiveWallets(c.Request.Context(), user.ID)
            }), 
            walletHandler.CreateWallet,
        )

        v1.POST("/budgets", 
            middleware.WithinLimit(entSvc, "max_budgets", func(c *gin.Context) (int64, error) {
                user := c.MustGet("currentUser").(AuthenticatedUser)
                return budgetRepo.CountActiveBudgets(c.Request.Context(), user.ID)
            }), 
            budgetHandler.CreateBudget,
        )
    }
}
```

---

## 5. Karakteristik Performa

| Kondisi | Jalur Resolusi | Latensi Pengecekan |
|---|---|:---:|
| **L1 Cache Hit** | Memori lokal instance Go (in-process hash map) | $\le 0.05\text{ ms}$ |
| **L2 Cache Hit** | Memori Redis via koneksi TCP lokal | $\le 1.2\text{ ms}$ |
| **Cache Miss** | Query PostgreSQL (agregasi `plans` + `plan_limits`) | $\le 8.5\text{ ms}$ |
| **Bypass Super Admin** | Pengecekan string `user.Role` di konteks memori | $\le 0.01\text{ ms}$ |
