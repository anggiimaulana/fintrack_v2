# Panduan Teknis: Frontend Entitlement Mirror & UX Guards

## 1. Filosofi Otorisasi Frontend (Client Mirroring)

Sesuai dengan keputusan terkunci **K-13** (*Penegakan hak akses di backend otoritatif; FE hanya cermin UX*), kode yang berjalan di browser klien Next.js **TIDAK PERNAH** dianggap sebagai mekanisme keamanan.

Tujuan utama sistem otorisasi frontend adalah:
1. **User Experience (UX) yang Transparan**: Menyembunyikan atau menonaktifkan tombol fitur berbayar sebelum pengguna mengkliknya, menghindari kekecewaan akibat error HTTP 403.
2. **Edukasi & Dorongan Upgrade (Upsell)**: Menampilkan banner penjelasan yang informatif (misal: "Tingkatkan ke paket Pro untuk mengekspor ke Excel") beserta tombol langsung menuju alur pembayaran.
3. **Penyelarasan Menu Dinamis**: Menyaring item sidebar atau tab navigasi berdasarkan role dan paket akun saat ini.

---

## 2. Hook Terpusat: `useEntitlements()`

Frontend mengonsumsi data hak akses pengguna melalui hook React terpadu `frontend/hooks/use-entitlements.ts`:

```typescript
import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-keys';
import { api } from '@/lib/api-client';

export interface UserEntitlements {
  role: 'super_admin' | 'user';
  planId: string;
  planName: string;
  features: Record<string, boolean>; // Contoh: { export_csv: true, ai_chat: true, receipt_ocr: false }
  limits: Record<string, { current: number; max: number }>; // Contoh: { monthly_transactions: { current: 42, max: 100 } }
}

export function useEntitlements() {
  const { data, isLoading, error } = useQuery<UserEntitlements>({
    queryKey: queryKeys.entitlements.all,
    queryFn: async () => {
      const res = await api.get('/user/entitlements');
      return res.data;
    },
    staleTime: 5 * 60 * 1000, // 5 menit cache lokal, di-invalidate instan via SSE plan_updated
  });

  const hasFeature = (featureKey: string): boolean => {
    if (data?.role === 'super_admin') return true;
    return !!data?.features[featureKey];
  };

  const isLimitReached = (limitKey: string): boolean => {
    if (data?.role === 'super_admin') return false;
    const limit = data?.limits[limitKey];
    if (!limit) return false;
    return limit.current >= limit.max;
  };

  const getRemainingQuota = (limitKey: string): number => {
    const limit = data?.limits[limitKey];
    if (!limit) return 0;
    return Math.max(0, limit.max - limit.current);
  };

  return {
    entitlements: data,
    isLoading,
    error,
    role: data?.role ?? 'user',
    isSuperAdmin: data?.role === 'super_admin',
    hasFeature,
    isLimitReached,
    getRemainingQuota,
  };
}
```

---

## 3. Komponen Pelindung Visual: `<EntitlementGuard>`

Komponen pembungkus serbaguna `components/entitlement-guard.tsx` digunakan untuk mengontrol rendering bagian antarmuka:

```tsx
import React from 'react';
import { useEntitlements } from '@/hooks/use-entitlements';
import { Button } from '@/components/ui/button';
import { Lock } from 'lucide-react';
import Link from 'next/link';

interface EntitlementGuardProps {
  feature?: string;
  limitKey?: string;
  fallback?: React.ReactNode;
  children: React.ReactNode;
  mode?: 'hide' | 'disable' | 'banner';
}

export function EntitlementGuard({
  feature,
  limitKey,
  fallback,
  children,
  mode = 'hide',
}: EntitlementGuardProps) {
  const { hasFeature, isLimitReached, isLoading } = useEntitlements();

  if (isLoading) {
    return <div className="animate-pulse h-10 bg-muted rounded-md" />;
  }

  const featureAllowed = feature ? hasFeature(feature) : true;
  const limitAvailable = limitKey ? !isLimitReached(limitKey) : true;
  const isAllowed = featureAllowed && limitAvailable;

  if (isAllowed) {
    return <>{children}</>;
  }

  // Jika hak akses tidak terpenuhi:
  if (fallback) {
    return <>{fallback}</>;
  }

  if (mode === 'hide') {
    return null;
  }

  if (mode === 'banner') {
    return (
      <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-950 dark:bg-amber-950/40">
        <div className="flex items-center gap-3">
          <Lock className="h-5 w-5 text-amber-600 dark:text-amber-400" />
          <div className="flex-1 text-sm text-amber-800 dark:text-amber-300">
            Fitur ini memerlukan paket langganan lebih tinggi untuk dapat digunakan.
          </div>
          <Button asChild size="sm" variant="default" className="cursor-pointer">
            <Link href="/settings/subscription">Tingkatkan Paket</Link>
          </Button>
        </div>
      </div>
    );
  }

  // mode === 'disable'
  return (
    <div className="relative inline-block cursor-not-allowed opacity-60">
      <div className="pointer-events-none">{children}</div>
    </div>
  );
}
```

---

## 4. Contoh Pola Penggunaan pada Halaman Aplikasi

### 4.1. Proteksi Tombol Ekspor CSV/Excel
Pada halaman daftar transaksi:
```tsx
import { EntitlementGuard } from '@/components/entitlement-guard';
import { Button } from '@/components/ui/button';
import { Download } from 'lucide-react';

export function ExportTransactionsButton() {
  return (
    <EntitlementGuard 
      feature="export_csv_excel" 
      mode="banner"
      fallback={
        <Button variant="outline" size="sm" disabled className="cursor-not-allowed opacity-50">
          <Download className="mr-2 h-4 w-4" />
          Ekspor CSV (Khusus Pro)
        </Button>
      }
    >
      <Button 
        variant="outline" 
        size="sm" 
        className="cursor-pointer" 
        onClick={handleDownloadCsv}
      >
        <Download className="mr-2 h-4 w-4" />
        Ekspor CSV
      </Button>
    </EntitlementGuard>
  );
}
```

### 4.2. Indikator Batas Kuota Input Transaksi per Siklus
Pada formulir pencatatan transaksi:
```tsx
export function CreateTransactionForm() {
  const { isLimitReached, getRemainingQuota } = useEntitlements();
  const limitReached = isLimitReached('monthly_transactions');
  const remaining = getRemainingQuota('monthly_transactions');

  return (
    <form onSubmit={handleSubmit}>
      {limitReached ? (
        <div className="mb-4 p-3 bg-destructive/10 text-destructive text-sm rounded-md">
          Kuota transaksi per siklus Anda telah mencapai batas maksimal (0 transaksi tersisa). Silakan tingkatkan paket Anda untuk melanjutkan pencatatan tanpa batas.
        </div>
      ) : (
        <div className="mb-2 text-xs text-muted-foreground">
          Sisa kuota transaksi bulan ini: <span className="font-semibold">{remaining}</span> transaksi.
        </div>
      )}

      {/* Input form fields... */}

      <Button 
        type="submit" 
        disabled={limitReached} 
        className={limitReached ? "cursor-not-allowed opacity-50" : "cursor-pointer"}
      >
        Simpan Transaksi
      </Button>
    </form>
  );
}
```

---

## 5. Ringkasan Sinkronisasi Realtime dengan SSE

Ketika status langganan berubah melalui SSE event `plan_updated`:
1. `realtime-dispatcher.ts` menerima pesan server.
2. Memanggil `queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.all })`.
3. Hook `useEntitlements()` otomatis melakukan refetch data hak akses terbaru di latar belakang (< 150ms).
4. Komponen `<EntitlementGuard>` me-re-render tampilan secara instan: tombol yang tadinya terkunci atau nonaktif seketika berubah menjadi aktif tanpa membutuhkan refresh halaman (*zero full-page reload*).
