# 06 — Admin Entitlement & Monetisasi Paket

**Topik:** Konfigurasi Dinamis Paket Super Admin, Feature Switches, Limit Kuota, & Model Shared Token Pool  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Arsitektur Entitlement & Monetisasi Dinamis

FinTrack v2 mengadopsi model **Data-Driven Feature Flags** di mana izin akses fitur dan kuota penggunaan disimpan di database dan dikendalikan oleh Super Admin secara dinamis (*zero downtime*).

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       DYNAMIC ENTITLEMENT ENFORCEMENT                       │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
               [Pengguna Mengakses /accounts atau Upload Mutasi]
                                    │
                                    ▼
                     ┌─────────────────────────────┐
                     │ 1. Entitlement Middleware   │
                     │    Periksa subscription user│
                     └──────────────┬──────────────┘
                                    │
                  ┌─────────────────┴─────────────────┐
       [bank_integration = false]          [bank_integration = true]
                  │                                   │
                  ▼                                   ▼
    ┌───────────────────────────┐       ┌───────────────────────────┐
    │ 403 Forbidden             │       │ 2. Limit Check            │
    │ ERR_PLAN_UPGRADE_REQUIRED │       │    - max_financial_accs   │
    │ Kirim Payload Upsell      │       │    - monthly_imports      │
    └───────────────────────────┘       └─────────────┬─────────────┘
                                                      │
                                        ┌─────────────┴─────────────┐
                                  [Limit Tercapai]           [Limit Aman]
                                        │                         │
                                        ▼                         ▼
                          ┌───────────────────────────┐  ┌──────────────────┐
                          │ 429 Too Many Requests     │  │ Lolos Eksekusi   │
                          │ ERR_PLAN_LIMIT_REACHED    │  │ Pipeline Ingest  │
                          └───────────────────────────┘  └──────────────────┘
```

---

## 2. Definisi Kunci Fitur & Batasan Numerik

### 2.1 Feature Switches (`plan_entitlements`)

| Feature Key | Tipe | Deskripsi | Nilai Default Awal |
|---|---|---|---|
| `bank_integration` | Boolean | Mengizinkan pembuatan akun bank/e-wallet dan impor berkas mutasi. | **Free:** `false`<br>**Pro:** `false`<br>**Business:** `true` |
| `bank_auto_sync` | Boolean | Mengizinkan sinkronisasi otomatis via agregator Open Finance (Fase 2). | **Free:** `false`<br>**Pro:** `false`<br>**Business:** `false` (diaktifkan saat Fase 2 rilis) |

### 2.2 Batasan Numerik (`plan_limits`)

| Limit Key | Tipe | Arti Nilai | Batas Paket Awal |
|---|---|---|---|
| `max_financial_accounts` | Int64 | Jumlah maksimal rekening bank dan e-wallet aktif yang dapat dibuat pengguna (`-1` = tanpa batas / unlimited). | **Free:** `0`<br>**Pro:** `0`<br>**Business:** `-1` (unlimited) |
| `monthly_statement_imports` | Int64 | Jumlah maksimal file mutasi yang dapat diimpor per bulan kalender (`-1` = tanpa batas). | **Free:** `0`<br>**Pro:** `0`<br>**Business:** `100` |

---

## 3. Manajemen Super Admin (Zero Downtime)

Super Admin mengelola fitur ini secara visual melalui menu **Paket Langganan** (`/admin/plans`):

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ EDIT PAKET: BUSINESS (ID: 8fa7b2... )                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ [V] Fitur Integrasi Bank & E-Wallet (bank_integration)                      │
│ [ ] Auto-Sync Agregator Realtime (bank_auto_sync)                           │
│                                                                             │
│ Kuota Maksimal Akun Finansial : [-1       ] (Isi -1 untuk Unlimited)        │
│ Kuota Impor Mutasi Bulanan    : [100      ] berkas/bulan                    │
│                                                                             │
│ [ Simpan Perubahan Paket ]                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Mekanisme Zero Downtime:
1. Super Admin menekan tombol simpan di frontend admin.
2. Frontend memanggil endpoint yang sudah ada: `PATCH /admin/plans/:id`.
3. Handler admin mengeksekusi `UPSERT INTO plan_entitlements` dan `UPSERT INTO plan_limits` di PostgreSQL.
4. Karena middleware membaca entitlement langsung dari tabel atau cache ber-TTL pendek (5 detik) dengan invalidasi event, seluruh perubahan berlaku **seketika itu juga** bagi seluruh pengguna tanpa perlu restart service backend atau deploy ulang kode.

---

## 4. Penegakan Middleware (Backend Enforcement)

Implementasi middleware Go pada rute perbankan (`backend/internal/delivery/http/middleware/entitlement.go`):

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/pkg/response"
)

// RequireFeature memeriksa feature switch boolean pada paket aktif pengguna.
func RequireFeature(billingRepo billing.Repository, featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		sub, err := billingRepo.GetByUserID(c.Request.Context(), userID)
		if err != nil || sub == nil || !sub.IsActive() {
			response.Forbidden(c, "Fitur ini memerlukan paket langganan aktif", gin.H{
				"code": "ERR_SUBSCRIPTION_REQUIRED",
				"required_feature": featureKey,
			})
			c.Abort()
			return
		}

		entitlements, err := billingRepo.GetEntitlements(c.Request.Context(), sub.PlanID)
		if err != nil || !entitlements[featureKey] {
			response.Forbidden(c, "Fitur integrasi bank & e-wallet hanya tersedia pada paket Business", gin.H{
				"code": "ERR_PLAN_UPGRADE_REQUIRED",
				"required_feature": featureKey,
				"current_plan": sub.PlanID,
				"upgrade_url": "/billing/upgrade?plan=business",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
```

---

## 5. Model Biaya: Shared Token Pool Invariant

Untuk normalisasi mutasi Stage 2 (pemanggilan AI Gateway ketika regex tidak mengenali merchant):
- **Tidak Ada Tagihan Tersembunyi:** Sistem **tidak** mengenakan biaya langganan tambahan per baris impor.
- **Konsumsi Kuota Token Pengguna:** Setiap pemanggilan ekstraksi kategori/merchant oleh AI memotong unit token dari `token_quota` akun pengguna melalui `TokenMeter` yang sudah beroperasi.
- **Proteksi Saat Kuota Token Habis:**
  - Jika saldo token pengguna habis saat memproses file mutasi berisi 50 transaksi, ingestion **tidak dibatalkan**.
  - Baris transaksi tetap dimasukkan ke database dengan kategori default *"Uncategorized"* dan flag `needs_review = true`.
  - Sistem menampilkan banner informatif di UI: *"Impor selesai. Beberapa transaksi belum dikategorikan otomatis karena kuota token AI Anda habis. Silakan top up token atau atur kategori manual."*

---

## 6. Desain Upsell & Paywall Experience (Frontend)

Ketika pengguna paket **Free** atau **Pro** mencoba mengakses fitur ini:
1. Tab menu *"Akun & Rekening"* tetap terlihat di sidebar navigasi dengan penanda badge *"Business"*.
2. Saat diklik, pengguna disambut dengan **Interactive Value Showcase**:
   - Preview visual kartu Net Worth & Sankey Diagram.
   - Daftar logo bank didukung (BCA, Mandiri, Jago, SeaBank, GoPay, DANA, ShopeePay).
   - Penjelasan waktu yang dihemat (rata-rata 4 jam pencatatan manual per bulan).
   - Tombol Call-to-Action (CTA): **"Upgrade ke Business — Coba 14 Hari"** yang langsung mengarahkan ke alur checkout paket Business.
