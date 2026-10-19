# Spesifikasi Teknis: Realtime Caching (SSE), Bus Event & RBAC Zero-Downtime

Dokumen ini mendefinisikan arsitektur *event-driven* realtime berbasis Server-Sent Events (SSE), strategi caching hybrid (in-memory patch + surgical invalidation), penegakan otorisasi dinamis (*Dynamic RBAC*), dan mitigasi kebocoran memori/lag di FinTrack v2.

---

## 1. Arsitektur Koneksi Tunggal SSE (Zero Polling)

FinTrack v2 secara mutlak **melarang polling berulang** (`setInterval` / `refetchInterval`) pada seluruh modul admin dan super admin. Seluruh sinkronisasi data berpijak pada koneksi SSE tunggal per tab browser.

### 1.1. Alur Koneksi SSE
```
+---------------------------------------------------------------------------------------------------------+
| Browser Tab (React)                                                               Backend (Go / Gin)   |
|                                                                                                         |
| 1. GET /api/v1/realtime/stream  --------------------------------------------->  Autentikasi Cookie ft_at|
|                                                                                 Register client ke Hub  |
| 2. HTTP 200 text/event-stream   <---------------------------------------------  Header No-Buffer, Flush |
|                                                                                                         |
| 3. Heartbeat (tiap 15s)         <---------------------------------------------  Tulis ping comment      |
|                                                                                                         |
| 4. Domain Event Dispatched      <---------------------------------------------  Broadcast dari Handler  |
|    (id, event, data payload)                                                                            |
|                                                                                                         |
| 5. Terputus (Jaringan Drop)     - - - - - - - - - - - - - - - - - - - - - - ->  Unregister client       |
| 6. Reconnect (Backoff 1s..16s)  --- last_event_id=xyz ------------------------>  Kirim buffered events   |
+---------------------------------------------------------------------------------------------------------+
```

### 1.2. Bus Realtime Lokal (`realtime-bus.ts`)
Komponen antarmuka (Header Bell, Data Table, Admin Shell, Charts) **tidak boleh membuka koneksi EventSource mandiri**. Seluruh komponen berlangganan ke bus event internal di memory klien:

```typescript
// frontend/lib/realtime-bus.ts
type EventCallback = (event: DomainEvent) => void;

const listeners = new Set<EventCallback>();

export function subscribeDomainEvents(cb: EventCallback): () => void {
  listeners.add(cb);
  return () => listeners.delete(cb); // Pembersihan otomatis saat unmount
}

export function publishDomainEvent(event: DomainEvent): void {
  listeners.forEach((cb) => {
    try {
      cb(event);
    } catch (err) {
      console.error("[RealtimeBus] Listener callback error:", err);
    }
  });
}
```

---

## 2. Strategi Caching Hybrid (Zero Waste HTTP)

Untuk menjamin performa tanpa jeda dan mencegah pemborosan kueri database (*cache storm*), FinTrack v2 menerapkan pembagian strategi pembaruan cache:

### 2.1. Tipe A: Direct In-Memory Cache Patch (Update Baris & Status)
Digunakan untuk: Perubahan status user (`is_active`), approval pembayaran manual, pembaruan kuota parsial, toggle feature flag, dan karantina route AI.

- **Mekanisme**:
  - Payload SSE membawa data baris yang diperbarui secara utuh atau parsial (`{ id: string, patch: Partial<T> }`).
  - Dispatcher langsung memutasi objek di TanStack Query Cache melalui `queryClient.setQueryData`.
  - **Hasil**: Antarmuka memperbarui status baris dalam **0 milidetik** tanpa mengirim request HTTP GET baru ke server.

```typescript
// Cuplikan Dispatcher In-Memory Patch
export function patchUserStatusInCache(queryClient: QueryClient, userId: string, isActive: boolean) {
  queryClient.setQueriesData<PaginatedResponse<UserRow>>({ queryKey: ["admin", "users"] }, (old) => {
    if (!old) return old;
    return {
      ...old,
      data: old.data.map((user) => (user.id === userId ? { ...user, is_active: isActive } : user)),
    };
  });
}
```

### 2.2. Tipe B: Targeted Surgical Invalidation (Create, Delete & Refilter)
Digunakan untuk: Penambahan pengguna baru, penghapusan data secara permanen, atau perubahan yang mengubah total agregasi/pagination.

- **Mekanisme**:
  - Penambahan atau penghapusan data memengaruhi nomor baris, total baris, dan jumlah halaman.
  - Dispatcher mengeksekusi invalidasi terarah:
    `queryClient.invalidateQueries({ queryKey: ["admin", "users", { page: activePage }], exact: false })`.
  - **Hasil**: Server hanya mengeksekusi 1 kueri `LIMIT 20 OFFSET X` untuk halaman yang sedang aktif dilihat oleh admin saat itu. Halaman atau modul lain yang sedang berada di background tidak akan di-fetch ulang.

---

## 3. Realtime RBAC & Dynamic Access Guard (Zero Downtime)

Sistem FinTrack v2 mendukung mutasi hak akses pengguna secara dinamis tanpa mengharuskan pengguna melakukan reload browser (F5) atau re-login manual:

### 3.1. Daftar Event Kontrak RBAC & Otorisasi

| Nama Event SSE | Produser (Backend) | Dampak pada Klien (Frontend) |
|---|---|---|
| `role_updated` | `SetUserRoleHandler` | Memperbarui state sesi di `session-context.tsx`. Memperbarui menu sidebar dan command catalog seketika. |
| `user_suspended` | `SuspendUserHandler` | Jika `userId` sama dengan sesi aktif, force logout seketika dengan pesan: *"Sesi Anda telah ditangguhkan oleh Administrator"*. |
| `entitlement_changed` | `SubscriptionWorker` | Memperbarui daftar hak akses fitur (AI scan, voice, limit transaksi). Menampilkan/menyembunyikan menu terkait. |
| `config_changed` | `SettingsHandler` | Memperbarui konfigurasi global (gateway aktif, timeout AI, maintenance mode flag). |

### 3.2. Penanganan Demosi Peran Dinamis (Role Demotion Guard)
Jika seorang pengguna diturunkan hak aksesnya dari `super_admin` menjadi `user` saat sedang membuka halaman `/super-admin/audit`:
1. Event `role_updated` diterima oleh tab browser pengguna tersebut.
2. Hook `useSession` memperbarui nilai `isSuperAdmin = false`.
3. Komponen guard layout mendeteksi bahwa rute `/super-admin/*` tidak lagi berhak diakses.
4. Router Next.js melakukan transisi navigasi halus (`router.replace('/user')`) disertai toast peringatan ramah.
5. Jika pengguna mencoba memintas via manipulasi DOM, middleware otorisasi backend di Go tetap memblokir seluruh request dengan response **HTTP 403 Forbidden**.

---

## 4. Pencegahan Memory Leak, Lag & Crash

Untuk menjamin komitmen performa tinggi tanpa lag:

1. **Unsubscribe Bersih pada Siklus Hidup Komponen**:
   - Setiap hook React yang memasang listener ke `subscribeDomainEvents` wajib mengembalikan fungsi pembersih pada blok `return` di `useEffect`.
2. **Deduplikasi Pesan Berdasarkan `event_id`**:
   - Klien menyimpan buffer ringkas 50 ID event terakhir dalam `Set<string>`.
   - Jika terjadi network glitch dan server mengirimkan ulang event yang sudah diproses, klien mengabaikan duplikasi tersebut secara instan.
3. **Pemberhentian Aliran Saat Tab Background Tidak Aktif (Opsional)**:
   - Jantung koneksi SSE tetap dipertahankan dengan beban memori minimal (< 2MB footprint).
   - Rendering UI chart dan tabel dihentikan sementara saat `document.visibilityState === 'hidden'` dan di-catch up saat tab kembali aktif.
4. **Isolasi Error Listener**:
   - Pemanggilan callback listener dibungkus blok `try-catch` terisolasi. Kesalahan fatal pada satu komponen tidak akan pernah menghentikan jalannya bus realtime atau menyebabkan tab browser crash.
