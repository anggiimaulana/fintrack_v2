# Cache Berversi & Sinkronisasi Realtime (Entitlement Cache) — FinTrack v2

Dokumen ini mendokumentasikan arsitektur sistem penyimpanan sementara multi-level (*multi-level versioned caching*) pada paket `internal/entitlement/cache/`. Sistem ini dirancang untuk mencapai penegakan hak akses instan (*realtime RBAC*) dengan latensi sub-milidetik, tanpa mengharuskan pengguna login ulang saat terjadi perubahan status paket langganan (K-14, K-15, K-21).

---

## 1. Topologi Caching Multi-Level (L1 & L2)

```mermaid
graph TD
    Request[HTTP Request / API Call] --> L1Cache[L1: In-Memory LRU Cache\nTTL: 60 Detik\nLokal per-Node Go]
    
    L1Cache -- Hit (0.05ms) --> Return[Kembalikan UserEntitlement]
    L1Cache -- Miss --> CheckVer[Baca Versi User\nRedis: entitlement:ver:{uid}]
    
    CheckVer --> L2Cache[L2: Redis Distributed Cache\nKey: entitlement:u:{uid}:v:{ver}\nTTL: 24 Jam]
    
    L2Cache -- Hit (1.2ms) --> PopulateL1[Isi L1 Memory]
    PopulateL1 --> Return
    
    L2Cache -- Miss --> DB[(PostgreSQL 17\nResolver Query)]
    DB --> PopulateL2[Tulis ke Redis L2]
    PopulateL2 --> PopulateL1
    PopulateL1 --> Return

    subgraph Invalidation ["Alur Invalidasi Instan"]
        Mutation[Mutasi Plan / Upgrade / Downgrade] --> IncrVer[Redis: INCR entitlement:ver:{uid}]
        IncrVer --> PubSub[Redis: PUBLISH entitlement:invalidations {uid}]
        PubSub --> Node1[Node Go #1: Evict L1]
        PubSub --> Node2[Node Go #2: Evict L1]
        PubSub --> NodeN[Node Go #N: Evict L1]
    end
```

---

## 2. Skema Penamaan Kunci (*Cache Key Scheme*)

| Tipe Kunci | Format Kunci Redis | Tipe Data | TTL | Fungsi |
|---|---|:---:|:---:|---|
| **Version Counter** | `entitlement:ver:{user_id}` | `INTEGER` | Tanpa Kedaluwarsa | Menyimpan versi aktif konfigurasi hak akses user. |
| **Payload Entitlement**| `entitlement:u:{user_id}:v:{version}` | `STRING (JSON)` | 24 Jam | Menyimpan snapshot agregasi fitur boolean dan batasan kuota. |
| **Channel Pub/Sub** | `entitlement:invalidations` | Pub/Sub Topic | Transient | Kanal siaran pesan pembersihan L1 lintas instans backend. |

---

## 3. Alur Invalidasi Instan (Zero Relogin)

Ketika terjadi perubahan paket pengguna (misal: pembayaran langganan sukses, pembatalan oleh user, atau modifikasi paket oleh super admin):

1. **Peningkatan Versi Pengguna:**
   Backend mengeksekusi operasi atomik di Redis:
   ```redis
   INCR entitlement:ver:11111111-2222-3333-4444-555555555555
   ```
   *Misalkan versi bertambah dari 4 menjadi 5.*
2. **Siaran Pesan Pub/Sub:**
   Backend mempublikasikan ID pengguna ke kanal Redis:
   ```redis
   PUBLISH entitlement:invalidations "11111111-2222-3333-4444-555555555555"
   ```
3. **Pembersihan L1 pada Seluruh Node Backend:**
   Setiap instance Go yang sedang mendengarkan (*subscriber*) menerima pesan tersebut dan langsung menghapus entri pengguna dari memori lokal LRU masing-masing.
4. **Permintaan Pengguna Selanjutnya:**
   - Request HTTP berikutnya dari pengguna tidak menemukan entri di L1.
   - Sistem membaca versi terbaru dari Redis (`ver = 5`).
   - Sistem mencari kunci L2 `entitlement:u:{uid}:v:5`.
   - Karena kunci `v:5` belum ada, sistem membaca PostgreSQL, menyusun snapshot hak akses baru, dan menyimpannya di Redis dengan key `v:5` serta mengisi kembali L1 lokal.
   - Kunci lama `v:4` otomatis kedaluwarsa sesuai TTL 24 jam di Redis tanpa perlu dieksekusi pembersihan manual.

---

## 4. Implementasi Go: Cache Manager & Pub/Sub Subscriber

```go
package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
    "fintrack_v2/internal/entitlement/domain"
)

type VersionedCacheManager struct {
    rdb          *redis.Client
    localCache   sync.Map // In-Memory L1 (Alternatif: Ristretto / Hashicorp LRU)
    resolverRepo domain.EntitlementResolverRepository
}

func NewVersionedCacheManager(rdb *redis.Client, repo domain.EntitlementResolverRepository) *VersionedCacheManager {
    mgr := &VersionedCacheManager{
        rdb:          rdb,
        resolverRepo: repo,
    }
    go mgr.subscribeInvalidations(context.Background())
    return mgr
}

// GetUserEntitlements mengambil data hak akses dengan strategi L1 -> L2 -> DB Fallback
func (m *VersionedCacheManager) GetUserEntitlements(ctx context.Context, userID string) (*domain.UserEntitlement, error) {
    // 1. Cek L1 Memory
    if val, found := m.localCache.Load(userID); found {
        return val.(*domain.UserEntitlement), nil
    }

    // 2. Baca Versi User dari Redis
    verKey := fmt.Sprintf("entitlement:ver:%s", userID)
    version, err := m.rdb.Get(ctx, verKey).Int64()
    if err != nil {
        if err == redis.Nil {
            version = 1
            _ = m.rdb.Set(ctx, verKey, version, 0).Err()
        } else {
            // Redis error fallback: Baca langsung dari DB
            return m.resolverRepo.ResolveUserEntitlements(ctx, userID)
        }
    }

    // 3. Baca L2 Redis Cache
    dataKey := fmt.Sprintf("entitlement:u:%s:v:%d", userID, version)
    cachedBytes, err := m.rdb.Get(ctx, dataKey).Bytes()
    if err == nil {
        var ent domain.UserEntitlement
        if err := json.Unmarshal(cachedBytes, &ent); err == nil {
            m.localCache.Store(userID, &ent)
            return &ent, nil
        }
    }

    // 4. Cache Miss: Ambil dari DB PostgreSQL
    freshEnt, err := m.resolverRepo.ResolveUserEntitlements(ctx, userID)
    if err != nil {
        return nil, err
    }
    freshEnt.Version = version

    // 5. Simpan ke L2 (Redis) & L1 (Memory)
    if bytes, err := json.Marshal(freshEnt); err == nil {
        _ = m.rdb.Set(ctx, dataKey, bytes, 24*time.Hour).Err()
    }
    m.localCache.Store(userID, freshEnt)

    return freshEnt, nil
}

// InvalidateUser menaikkan versi dan menyiarkan sinyal invalidasi
func (m *VersionedCacheManager) InvalidateUser(ctx context.Context, userID string) error {
    verKey := fmt.Sprintf("entitlement:ver:%s", userID)
    if err := m.rdb.Incr(ctx, verKey).Err(); err != nil {
        return err
    }

    m.localCache.Delete(userID)
    return m.rdb.Publish(ctx, "entitlement:invalidations", userID).Err()
}

// subscribeInvalidations mendengarkan siaran pembersihan cache lintas node
func (m *VersionedCacheManager) subscribeInvalidations(ctx context.Context) {
    pubsub := m.rdb.Subscribe(ctx, "entitlement:invalidations")
    defer pubsub.Close()

    ch := pubsub.Channel()
    for msg := range ch {
        m.localCache.Delete(msg.Payload)
    }
}
```

---

## 5. Mekanisme Fallback Resilient (Graceful Degradation)

Jika koneksi Redis terputus total atau mengalami latensi tinggi ($> 50\text{ ms}$):
1. **Circuit Breaker Fallback:** Fungsi pembacaan mendeteksi kegagalan koneksi Redis dan langsung mengalihkan rute pembacaan ke PostgreSQL via `resolverRepo.ResolveUserEntitlements(ctx, userID)`.
2. **Local L1 Retain:** Selama insiden jaringan Redis, L1 in-memory tetap melayani request yang sudah tersimpan untuk meminimalkan lonjakan query (*query stampede*) ke database PostgreSQL utama.
3. **Pencatatan Log:** Seluruh kegagalan Redis dicatat dengan tingkat keparahan `WARN` atau `ERROR` untuk pemantauan SRE tanpa melempar HTTP 500 ke pengguna akhir.

---

## 6. Sinkronisasi Frontend (Endpoint `/api/v1/me/entitlements`)

Frontend Next.js menggunakan TanStack Query v5 untuk menyinkronkan hak akses pengguna:

```typescript
// hooks/use-entitlements.ts
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';

export interface UserEntitlementsResponse {
  plan: {
    code: string;
    name: string;
    status: string;
    current_period_end: string | null;
    cancel_at_period_end: boolean;
  };
  features: Record<string, boolean>;
  limits: Record<string, { limit: number; used: number; remaining: number }>;
  version: number;
}

export function useEntitlements() {
  return useQuery<UserEntitlementsResponse>({
    queryKey: ['me', 'entitlements'],
    queryFn: async () => {
      const { data } = await apiClient.get('/api/v1/me/entitlements');
      return data;
    },
    staleTime: 5 * 60 * 1000, // 5 menit cache browser
    refetchOnWindowFocus: true,
  });
}
```

Ketika event SSE `entitlement_updated` diterima oleh browser dari backend, hook memicu `queryClient.invalidateQueries({ queryKey: ['me', 'entitlements'] })`, menghasilkan pembaruan visual komponen antarmuka seketika tanpa perlu *refresh* halaman.
