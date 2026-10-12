# Temuan Alur Bisnis: Kesenjangan Alur Pembelian Paket Langganan Mandiri (Checkout Flow Gap)

**Kode Dokumen**: AUDIT-FLOW-01  
**Klasifikasi**: Alur Fungsional Bisnis & Monetisasi (Monetization & Checkout Integration)  

---

## 1. Lokasi & Fakta Kesenjangan

### 1.1 Sisi Backend
- **Berkas**: `backend/internal/payment/checkout.go`
- **Kondisi**:
  Struktur `CheckoutService` telah selesai dibangun dan teruji unit di `checkout_test.go`. Service ini mampu menentukan gateway aktif, menghitung pajak (PPN), membuat invoice, memanggil Snap API Midtrans, dan mencatat transaksi pembayaran dengan status `pending`.
- **Kesenjangan**:
  `CheckoutService` **belum didaftarkan ke handler HTTP** di `backend/internal/delivery/http/handler/payment_handler.go` dan **belum diekspos sebagai rute** di `backend/internal/delivery/http/router.go`.
  Di `cmd/server/main.go` baris 315, bahkan terdapat catatan kode:
  ```go
  _ = payment.NewInvoiceService(paymentRepo, settingsRepo, payTax) // available for P06 checkout endpoints
  ```
  Tetapi endpoint `POST /api/v1/payments/checkout` belum pernah dihubungkan.

### 1.2 Sisi Frontend
- **Berkas**: `frontend/app/(dashboard)/user/settings/subscription/page.tsx`
- **Kondisi**:
  Baris 11–18 secara eksplisit mendokumentasikan:
  ```typescript
  /**
   * Read-only view of the user's plan, features and limits.
   *
   * This page is the destination behind the backend's `upgrade_url`, so every
   * 403/402 the API returns points somewhere real. It deliberately has no purchase
   * flow — billing UX is owned by another phase; this only reports what the
   * entitlement resolver currently grants.
   */
  ```
- **Kesenjangan**:
  Halaman langganan pengguna murni bersifat tampilan statis (*read-only*). Ketika pengguna paket Free mencapai batas token AI atau transaksi dan diarahkan ke `/user/settings/subscription`, pengguna **tidak menemukan tombol untuk membeli atau meng-upgrade paket** ke Pro atau Business.

---

## 2. Dampak terhadap Operasional Produksi
1. **Model Bisnis Terhambat**: Pengguna yang ingin membayar tidak memiliki mekanisme swalayan untuk membeli langganan berbayar.
2. **Ketergantungan Intervensi Manual**: Untuk menaikkan paket pengguna, saat ini operator harus masuk ke portal Super Admin (`/super-admin/subscriptions`) dan mengubah paket pengguna secara manual.
3. **Dead End UX**: Notifikasi batasan kuota (HTTP 402/403) mengarahkan pengguna ke halaman langganan yang tidak memiliki tombol aksi pembelian (*call-to-action*).

---

## 3. Desain Solusi & Rencana Implementasi

### 3.1 Langkah 1: Registrasi Endpoint `POST /api/v1/payments/checkout` di Backend
1. Tambahkan method `Checkout` pada `PaymentHandler`:
```go
// backend/internal/delivery/http/handler/payment_handler.go
func (h *PaymentHandler) Checkout(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    var req struct {
        PlanCode string `json:"plan_code" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "Kode paket wajib diisi")
        return
    }

    // Ambil profil user & harga paket
    plan, err := h.planRepo.GetPlanByCode(c.Request.Context(), req.PlanCode)
    if err != nil || plan == nil || !plan.IsActive {
        response.NotFound(c, "Paket tidak ditemukan atau tidak aktif")
        return
    }

    userProfile, err := h.userRepo.GetByID(c.Request.Context(), userID)
    if err != nil || userProfile == nil {
        response.InternalError(c, "Gagal memuat profil pengguna")
        return
    }

    res, err := h.checkoutSvc.Checkout(c.Request.Context(), payment.CheckoutRequest{
        UserID:        userID,
        PlanCode:      plan.Code,
        PlanName:      plan.Name,
        AmountIDR:     plan.PriceMonthlyIDR,
        CustomerName:  userProfile.FullName,
        CustomerEmail: userProfile.Email,
    })
    if err != nil {
        response.InternalError(c, "Gagal membuat sesi pembayaran: " + err.Error())
        return
    }

    response.Created(c, "Sesi pembayaran berhasil dibuat", res)
}
```

2. Daftarkan rute pada `userRoutes` di `backend/internal/delivery/http/router.go`:
```go
userRoutes.POST("/payments/checkout", paymentH.Checkout)
```

### 3.2 Langkah 2: Integrasi Midtrans Snap SDK di Frontend
1. Buat hook mutasi `useCheckout`:
```typescript
// frontend/hooks/useCheckout.ts
export function useCheckout() {
  return useMutation({
    mutationFn: async (planCode: string) => {
      const res = await api.post('/payments/checkout', { plan_code: planCode });
      return res.data?.data;
    },
  });
}
```

2. Tambahkan tombol interaktif **"Tingkatkan ke Pro (Rp 39.000/bln)"** dan **"Tingkatkan ke Business (Rp 99.000/bln)"** pada `frontend/app/(dashboard)/user/settings/subscription/page.tsx`:
   - Jika response mengembalikan `snap_token`: Buka modal popup Midtrans Snap via `window.snap.pay(token, { onSuccess, onPending, onError })`.
   - Jika response mengembalikan `redirect_url`: Buka tab baru menuju URL pembayaran gateway.
   - Sediakan instruksi transfer manual jika gateway yang aktif adalah `manual`.

3. Integrasikan auto-refresh entitlement via Realtime SSE: begitu webhook `payment_received` atau `plan_updated` tiba melalui SSE, antarmuka langganan otomatis memutakhirkan badge paket menjadi `Pro (Aktif)` secara instan tanpa perlu refresh manual.
