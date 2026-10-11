# Strategi Cache Invalidation: Redis Pub/Sub & Surgical Updates TanStack Query

## 1. Masalah Cache Invalidation di Aplikasi Keuangan

Pada aplikasi keuangan pribadi seperti FinTrack v2, inkonsistensi cache menimbulkan dampak buruk:
- Pengguna melihat sisa saldo atau transaksi yang belum terupdate setelah checkout langganan.
- Super admin melihat status pembayaran pending padahal webhook gateway sudah berstatus lunas.
- Refetch global tanpa filter memicu *cache storm* (ratusan request bersamaan ke backend) yang membebani database PostgreSQL.

FinTrack v2 menyelesaikan masalah ini melalui dua prinsip:
1. **Direct User Action**: Menggunakan **Optimistic Updates** & `setQueryData` instan pada klien yang melakukan aksi.
2. **External / System Event**: Menggunakan **Surgical Cache Invalidation** TanStack Query yang dipicu oleh stream SSE dan Redis Pub/Sub.

---

## 2. Struktur Namespace Redis & Penggantian `globalCache`

Seluruh variabel `globalCache` memori lokal Go dihapus total. Sistem beralih ke Redis dengan standardisasi namespace berversi:

```
ft:{env}:{modul}:{entitas}:{id}
```

### Tabel Namespace Kunci Redis:
| Pola Key Redis | Tipe Data | TTL | Kegunaan |
|---|---|---|---|
| `ft:prod:entitlement:ver:{uid}` | String (Integer) | Tanpa batas | Pointer versi entitlement pengguna (di-INCR saat plan berubah) |
| `ft:prod:entitlement:payload:{uid}` | String (JSON) | 1 Jam | Cache payload hasil evaluasi fitur dan limit user |
| `ft:prod:realtime:buffer:{uid}` | Sorted Set (ZSET) | 10 Menit | Ring buffer penyimpanan event SSE untuk resume `Last-Event-ID` |
| `ft:prod:plan:cache:{plan_id}` | String (JSON) | 24 Jam | Cache detail paket dan fitur plan |
| `ft:prod:ai:route:config` | String (JSON) | 6 Jam | Konfigurasi routing AI gateway (chat, vision, embedding 768d) |

---

## 3. Registri Query Key Terpusat Frontend (TanStack Query v5)

Untuk mencegah typo dan mempermudah pelacakan invalidasi, seluruh query key frontend dideklarasikan di `frontend/src/lib/query-keys.ts`:

```typescript
export const queryKeys = {
  auth: {
    me: ['auth', 'me'] as const,
    session: ['auth', 'session'] as const,
  },
  entitlements: {
    all: ['entitlements'] as const,
    user: (userId: string) => ['entitlements', 'user', userId] as const,
    features: ['entitlements', 'features'] as const,
  },
  transactions: {
    all: ['transactions'] as const,
    list: (filters: Record<string, unknown>) => ['transactions', 'list', filters] as const,
    detail: (id: string) => ['transactions', 'detail', id] as const,
    summary: (month: string) => ['transactions', 'summary', month] as const,
  },
  budgets: {
    all: ['budgets'] as const,
    list: ['budgets', 'list'] as const,
    progress: (id: string) => ['budgets', 'progress', id] as const,
  },
  billing: {
    plans: ['billing', 'plans'] as const,
    subscription: ['billing', 'subscription'] as const,
    invoices: ['billing', 'invoices'] as const,
    tokens: ['billing', 'tokens'] as const,
  },
  superAdmin: {
    users: (params: Record<string, unknown>) => ['admin', 'users', params] as const,
    plans: ['admin', 'plans'] as const,
    gateways: ['admin', 'gateways'] as const,
    auditLogs: (params: Record<string, unknown>) => ['admin', 'audit-logs', params] as const,
  },
};
```

---

## 4. Pola Surgical Cache Invalidation pada SSE Event Dispatcher

Ketika peramban menerima event SSE melalui koneksi stream, file `frontend/src/lib/realtime-dispatcher.ts` melakukan invalidasi bedah:

```typescript
import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from './query-keys';

export function handleDomainSSEEvent(queryClient: QueryClient, eventData: SSEMessage) {
  switch (eventData.event) {
    case 'payment_received':
      // HANYA invalidasi query langganan, invoice, dan profil user
      queryClient.invalidateQueries({ queryKey: queryKeys.billing.subscription });
      queryClient.invalidateQueries({ queryKey: queryKeys.billing.invoices });
      queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.all });
      break;

    case 'plan_updated':
      // Invalidasi entitlement dan data batas kuota transaksi
      queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.billing.plans });
      queryClient.invalidateQueries({ queryKey: queryKeys.billing.subscription });
      break;

    case 'quota_reset':
      // Segarkan status limit kuota transaksi dan saldo token
      queryClient.invalidateQueries({ queryKey: queryKeys.billing.tokens });
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.summary });
      break;

    case 'ai_endpoint_down':
      // Khusus Super Admin: segarkan status health dashboard AI gateway
      queryClient.invalidateQueries({ queryKey: queryKeys.superAdmin.gateways });
      break;

    case 'manual_approval_needed':
      // Khusus Super Admin: segarkan tabel invoice pending transfer
      queryClient.invalidateQueries({ queryKey: queryKeys.billing.invoices });
      break;

    default:
      console.warn(`[SSE Dispatcher] Unhandled event type: ${eventData.event}`);
  }
}
```

---

## 5. Implementasi Optimistic Updates pada Aksi Langsung Pengguna

Untuk aksi mutasi yang dimulai oleh pengguna sendiri (tambah/sunting transaksi), sistem tidak menunggu SSE maupun respon server, melainkan langsung melakukan injeksi optimistic ke cache:

```typescript
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-keys';

export function useCreateTransaction() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (newTx: CreateTransactionDTO) => api.post('/transactions', newTx),
    
    // 1. Sebelum request jaringan dikirim:
    onMutate: async (newTx) => {
      // Batalkan refetch yang sedang berjalan untuk menghindari penimpaan cache
      await queryClient.cancelQueries({ queryKey: queryKeys.transactions.all });

      // Ambil snapshot cache sebelumnya untuk rollback bila gagal
      const previousTransactions = queryClient.getQueryData(queryKeys.transactions.all);

      // Mutasi optimistik ke cache lokal
      queryClient.setQueryData(queryKeys.transactions.all, (oldData: any) => {
        if (!oldData) return [newTx];
        return [
          { ...newTx, id: `temp-${Date.now()}`, created_at: new Date().toISOString() },
          ...oldData,
        ];
      });

      return { previousTransactions };
    },

    // 2. Bila request jaringan gagal (misal 409 Conflict atau 429 Limit Exceeded):
    onError: (err, newTx, context) => {
      if (context?.previousTransactions) {
        // Pulihkan cache ke kondisi awal (Rollback)
        queryClient.setQueryData(queryKeys.transactions.all, context.previousTransactions);
      }
    },

    // 3. Setelah sukses atau gagal:
    onSettled: () => {
      // Validasi ulang dengan data sebenarnya dari server
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all });
    },
  });
}
```

---

## 6. Diagram Alir Sinkronisasi Cache

```
[ User Klik "Beli Pro" ]
          │
          ▼
 [ Midtrans Checkout ] ──(Sukses Bayar)──► [ Webhook Midtrans ]
                                                    │
                                                    ▼
                                            [ Backend API ]
                                             (Update DB Postgre)
                                                    │
                                                    ├─► [ Redis INCR ver ]
                                                    │
                                                    ├─► [ Redis PUBLISH "realtime:events:usr_123" ]
                                                    │
                                                    ▼
                                            [ Hub SSE Broker ]
                                                    │
                                                    ▼ (event: payment_received)
                                            [ Browser Klien ]
                                                    │
                                                    ▼
                                            [ surgical invalidateQueries ]
                                            (['billing', 'subscription'])
                                                    │
                                                    ▼
                                            [ UI Menampilkan Badge Pro ]
                                            (Selesai dalam < 400ms)
```
