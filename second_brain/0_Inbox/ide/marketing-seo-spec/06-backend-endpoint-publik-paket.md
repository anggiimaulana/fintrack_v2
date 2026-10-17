# 06 — Backend: Endpoint Publik Katalog Paket

Dokumen ini merinci satu-satunya perubahan backend di Fase 1: membuka katalog paket untuk pengunjung anonim, agar halaman `/harga` dan teaser harga di landing dapat menampilkan harga nyata dari database.

---

## 1. Latar & Masalah

- Rute katalog saat ini: `userRoutes.GET("/plans", paymentH.ListPublicPlans)` (`backend/internal/delivery/http/router.go:369`).
- `userRoutes` berada di bawah `protected` (middleware `AuthMiddleware`) dan `RequireUser()`, sehingga **butuh JWT**.
- Pengunjung anonim tidak memiliki JWT → tidak bisa memuat harga → halaman publik tidak mungkin menampilkan paket nyata.

## 2. Keputusan

Menambah **satu rute GET publik** yang **memakai ulang** handler dan service yang sudah ada. Tidak ada logika katalog baru, tidak ada migrasi, tidak ada perubahan skema.

### 2.1 Mengapa aman

- Data yang dikembalikan adalah katalog paket yang **memang publik** (`is_public = true`, `is_active = true`) — sudah dirancang untuk ditampilkan (`catalog.go:105-108`).
- Tidak ada data pengguna yang disertakan.
- Middleware `OriginCheck` hanya menjaga metode mutasi `POST/PUT/PATCH/DELETE` (`origin_check.go:31-37`), sehingga GET publik ini tidak terpengaruh CSRF.
- Rute `/api/v1/plans` yang lama **tetap** di balik autentikasi (tidak diubah, tidak dihapus) — aplikasi dashboard yang sudah ada tidak terpengaruh.

## 3. Perubahan Kode

### 3.1 `backend/internal/delivery/http/router.go`

Di dalam grup publik `api` (blok `api := r.Group("/api/v1")`), daftarkan rute baru. Tempatkan berdekatan dengan webhook publik yang sudah ada:

```go
api := r.Group("/api/v1")
api.Use(rateLimitMw)
api.Use(middleware.OriginCheck(config.CORSOrigin))
{
    // Payment webhook is public (no JWT); signature is verified by the gateway.
    if paymentH != nil {
        api.POST("/payments/webhook/midtrans", paymentH.MidtransWebhook)

        // Public marketing catalog: the pricing page renders before login, so it
        // must be able to read the public plan list. Same handler/service as the
        // authenticated /plans route; only publicly-flagged plans are returned.
        api.GET("/public/plans", paymentH.ListPublicPlans)
    }

    auth := api.Group("/auth")
    // ... tidak berubah
}
```

Tidak ada perubahan pada handler (`payment_user_handler.go`) maupun service (`catalog.go`).

### 3.2 Rate limiting

Rute baru tidak terdaftar di `rateLimitMap`, sehingga memakai batas default middleware (pola yang sama dengan webhook publik). Jika lalu lintas beranda tinggi dan ingin batas lebih ketat, tambahkan entri eksplisit:

```go
"/api/v1/public/plans": {Period: 1 * time.Minute, Limit: 60},
```

> Keputusan: tambahkan pemetaan ini agar endpoint publik (tanpa autentikasi) punya anggaran eksplisit dan tidak bergantung default.

### 3.3 Cache backend

**Tidak** menambah cache backend di Fase 1. Cache berada di frontend lewat ISR (lihat §5). Alasan: `CacheMiddleware(30s)` adalah cache berbasis versi di grup terproteksi; memindahkannya ke rute publik menambah kompleksitas tanpa manfaat berarti karena halaman sudah di-cache statis oleh Next.

## 4. Kontrak Respons

Envelope standar (`backend/pkg/response/response.go:61-67`):

```json
{
  "success": true,
  "message": "Katalog paket berhasil dimuat",
  "data": [
    {
      "code": "free",
      "name": "Free",
      "description": "Paket gratis untuk mulai mencatat keuangan.",
      "price_monthly_idr": 0,
      "price_yearly_idr": 0,
      "billing_period_days": 30,
      "is_public": true,
      "sort_order": 1,
      "fallback_gateway": "manual",
      "features": { "ai_chat": true, "ocr_receipt": false, "...": false },
      "limits": { "ai_tokens_per_cycle": 5000, "...": 0 },
      "limit_meta": {
        "ai_tokens_per_cycle": { "key": "ai_tokens_per_cycle", "label": "Token AI / siklus", "unit": "token", "enforced": true }
      },
      "cycles": { "...": { } },
      "tax_enabled": false,
      "tax_label": "",
      "tax_rate_percent": 0,
      "tax_idr": 0,
      "total_idr": 0
    }
  ]
}
```

Catatan:

- `data` selalu berupa array (bila kosong: `[]`, bukan `null` — komunikasi-nya dijamin di handler `payment_user_handler.go:60-62`).
- Urutan mengikuti `sort_order`.
- `limit_meta` hanya memuat key yang didefinisikan paket tersebut (`catalog.go:131-136`).
- Nilai `-1` pada `limits` berarti **tanpa batas**; klien menerjemahkannya menjadi label "Tanpa batas".

## 5. Klien Server Frontend — `lib/plans-server.ts`

Berkas ini **server-only**. Memberikan data paket ke Server Component dengan ISR.

```ts
import "server-only"; // bila paket tersedia; jika tidak, jaga dengan komentar + hanya diimpor dari Server Component

const BACKEND_ORIGIN = process.env.BACKEND_ORIGIN ?? "http://localhost:8081";

export interface PublicPlanLimitMeta { key: string; label: string; unit: string; enforced: boolean }
export interface PublicPlan {
  code: string; name: string; description?: string;
  price_monthly_idr: number; price_yearly_idr: number;
  billing_period_days: number; is_public: boolean; sort_order: number;
  fallback_gateway: string;
  features: Record<string, boolean>;
  limits: Record<string, number>;
  limit_meta?: Record<string, PublicPlanLimitMeta>;
  tax_enabled?: boolean; tax_label?: string; tax_rate_percent?: number;
  tax_idr?: number; total_idr?: number;
}

/** Fetch katalog paket publik. Mengembalikan [] bila gagal (dipanggil dari Server Component). */
export async function fetchPublicPlans(): Promise<PublicPlan[]> {
  const res = await fetch(`${BACKEND_ORIGIN}/api/v1/public/plans`, {
    next: { revalidate: 3600 },
  });
  if (!res.ok) return [];
  const body = (await res.json()) as { data?: PublicPlan[] };
  return Array.isArray(body.data) ? body.data : [];
}

/** Filter limit yang di-enforce backend (mengikuti plan-limits.ts). */
export function enforcedLimits(plan: PublicPlan): Array<[string, number]> {
  return Object.entries(plan.limits).filter(
    ([key]) => plan.limit_meta?.[key]?.enforced !== false,
  );
}
```

### 5.1 Mengapa memakai `BACKEND_ORIGIN` langsung, bukan `/api/v1`

- Server Component berjalan di server Next; memanggil `/api/v1/...` relatif akan memerlukan URL absolut dan berpotensi memicu permintaan ke diri sendiri (self-request).
- `next.config.ts` sudah memakai `BACKEND_ORIGIN` untuk rewrite (`next.config.ts:3,45-52`), jadi variabelnya sudah menjadi konvensi deployment yang ada.
- Fallback `http://localhost:8081` konsisten dengan default `next.config.ts`.

### 5.2 Kompatibilitas tipe

`PublicPlan` di `hooks/use-billing.ts:28-50` sudah mendefinisikan bentuk serupa. Untuk menghindari duplikasi, pertimbangkan memindahkan tipe bersama ke `lib/plans-types.ts` dan mengimpornya dari kedua tempat. Bila dilakukan, ubah `use-billing.ts` agar mengimpor tipe tersebut (perubahan tipe murni, tanpa perubahan perilaku).

## 6. Penanganan Kegagalan

| Skenario | Perilaku |
|----------|----------|
| Backend mati saat build/revalidate pertama | `fetchPublicPlans` mengembalikan `[]`. `/harga` menyembunyikan grid harga tetapi tetap menampilkan FAQ, cara pembayaran, dan CTA. Landing menyembunyikan kartu teaser, menampilkan tombol `Lihat Harga`. |
| Backend mati setelah pernah sukses | Next ISR menyajikan halaman statis terakhir yang berhasil (hingga revalidate berikutnya). |
| Respons 200 tetapi `data` bukan array | Dikembalikan `[]` (defensif). |
| Timeout jaringan | `fetch` gagal → `[]` (tambahkan `AbortSignal.timeout(5000)` bila diinginkan; opsional). |

> **Trade-off ISR 3600s:** perubahan harga oleh Super Admin dapat memakan waktu hingga 1 jam untuk muncul di halaman publik. Checkout selalu memakai harga dari database (otoritatif), sehingga risiko finansial nol; hanya ada risiko kebingungan tampilan. Mitigasi opsional (di luar Fase 1): endpoint revalidasi bertoken dari backend ke Next (`revalidatePath('/harga')`). Bila diinginkan lebih responsif, turunkan ke `revalidate: 600`.

## 7. Pengujian

### 7.1 Unit/route (Go)

| Kasus | Harapan |
|-------|---------|
| `GET /api/v1/public/plans` tanpa cookie JWT | `200`, `success: true`, `data` berisi paket publik |
| `GET /api/v1/plans` tanpa cookie JWT | `401` (tetap terproteksi — regresi) |
| Katalog `nil` (service belum dikonfigurasi) | `200` dengan `data: []` |
| Paket dengan `is_public = false` | tidak muncul di respons |
| Paket dengan `is_active = false` | tidak muncul di respons |

Cara: perluas tes router/handler yang ada. Bila tes memerlukan DB, ikuti pola `backend/tests/` (skip bila env tidak diset, sesuai `CLAUDE.md`).

### 7.2 Frontend

| Kasus | Harapan |
|-------|---------|
| `fetchPublicPlans` respons OK dengan data | mengembalikan array terurai |
| `fetchPublicPlans` respons non-OK | mengembalikan `[]` |
| `fetchPublicPlans` `data` bukan array | mengembalikan `[]` |
| `enforcedLimits` menyembunyikan `max_categories`, `max_wallets` | hanya limit `enforced` tersisa |

## 8. Perintah Verifikasi

```bash
cd backend && go vet ./... && go test ./... && go build ./...
cd frontend && pnpm test && pnpm build
```

## 9. Kepatuhan Zona Repo

- Hanya menyentuh `backend/internal/delivery/http/router.go` (kode produksi) dan `frontend/lib/plans-server.ts` (kode produksi).
- Tidak menambah skrip debug ke area produksi (aturan zona `CLAUDE.md`).
