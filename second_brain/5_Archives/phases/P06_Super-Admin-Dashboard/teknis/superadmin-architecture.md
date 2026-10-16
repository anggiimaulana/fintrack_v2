# Arsitektur Super Admin Dashboard — FinTrack v2

Dokumen ini menguraikan arsitektur sistem, aliran autentikasi dan otorisasi, keamanan sesi, serta batas antarmuka modul Super Admin Dashboard pada FinTrack v2.

---

## 1. Ikhtisar Arsitektur Sistem

Modul Super Admin Dashboard dibangun mengikuti prinsip **Backend Otoritatif** (K-13) dan **Hibrida A+B / Strangler Fig** (K-24). Modul ini mengonsolidasikan kontrol operasional atas billing, AI gateway, manajemen token, dan pengaturan platform tanpa mengganggu alur kerja pengguna aplikasi reguler.

```mermaid
graph TD
    subgraph Browser Client (Next.js 16)
        AdminUI["Portal Super Admin (app/(dashboard)/super-admin/*)"]
        SharedDT["<DataTable> TanStack Table v8"]
        Charts["Interaktif Visualisasi Recharts"]
        URLState["URL Search Params (?page=&sort=...)"]
        AdminUI <--> SharedDT
        AdminUI <--> Charts
        SharedDT <--> URLState
    end

    subgraph Reverse Proxy / CDN
        Caddy["Caddy / Nginx Reverse Proxy"]
    end

    subgraph Backend Go/Gin Engine
        AuthMid["Auth & Session Middleware"]
        SuperAdminGuard["RequireSuperAdmin() Guard (K-02, K-14)"]
        
        AdminRoutes["Admin Handlers (/api/v1/admin/*)"]
        MetricsH["Metrics Handler"]
        PlanH["Plan & Pricing Handler"]
        SubH["Subscription Handler"]
        PayH["Payment & Review Handler"]
        TokenH["Token & Ledger Handler"]
        AIGateH["AI Ops & Kill Switch Handler"]
        AuditH["Audit Query Handler"]
        SetH["System Settings Handler"]
        
        AuthMid --> SuperAdminGuard --> AdminRoutes
        AdminRoutes --> MetricsH
        AdminRoutes --> PlanH
        AdminRoutes --> SubH
        AdminRoutes --> PayH
        AdminRoutes --> TokenH
        AdminRoutes --> AIGateH
        AdminRoutes --> AuditH
        AdminRoutes --> SetH
    end

    subgraph Data & Cache Tier
        PG[(PostgreSQL 17 + pgvector)]
        Redis[(Redis 7 - Session & Cache & Pub/Sub)]
        AuditDB[(audit_logs Append-Only)]
    end

    AdminUI -->|HTTPS / HttpOnly Cookie| Caddy
    Caddy --> AuthMid
    MetricsH <--> Redis
    PlanH --> PG
    SubH --> PG
    PayH --> PG
    TokenH --> PG
    AIGateH <--> Redis
    AIGateH --> PG
    AuditH --> AuditDB
    AdminRoutes -.->|Every Mutation Write| AuditDB
```

---

## 2. Aliran Autentikasi dan Otorisasi (RBAC Flow)

1. **Autentikasi Pengguna**:
   - Super admin melakukan otentikasi melalui flow login terpadu. Token sesi disimpan dalam `HttpOnly`, `Secure`, `SameSite=Lax` Cookie untuk mitigasi serangan XSS.
2. **Pemeriksaan Otorisasi Sisi Klien (SSR Guard)**:
   - File `app/(dashboard)/super-admin/layout.tsx` adalah Next.js Server Component.
   - Sesi diperiksa sebelum halaman dirender:
     ```typescript
     // Pseudocode server-side guard
     const session = await getServerSession();
     if (!session || session.user.role !== 'super_admin') {
       redirect('/dashboard'); // Lempar user reguler ke dashboard mereka
     }
     ```
3. **Pemeriksaan Otorisasi Sisi Server (Backend Enforcement)**:
   - Setiap endpoint `/api/v1/admin/*` diverifikasi oleh middleware `RequireSuperAdmin()` di Go:
     ```go
     func RequireSuperAdmin(cache *redis.Client, db *sql.DB) gin.HandlerFunc {
         return func(c *gin.Context) {
             userID := c.GetString("user_id")
             // Verifikasi role dari cache Redis berversi (K-14)
             role, err := getUserRoleVersioned(c.Request.Context(), cache, db, userID)
             if err != nil || role != "super_admin" {
                 c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                     "error": "Akses ditolak: Memerlukan hak akses super_admin",
                     "code":  "FORBIDDEN_SUPER_ADMIN_REQUIRED",
                 })
                 return
             }
             c.Next()
         }
     }
     ```
   - Sesuai K-14, perubahan role pengguna dari `super_admin` menjadi role lain di database langsung menginvalasi cache Redis, mencabut akses admin secara instan tanpa perlu menunggu token kedaluwarsa.

---

## 3. Desain Komponen UI dan Komunikasi Data

1. **State Management Berbasis URL**:
   - Komponen `<DataTable>` tidak menyimpan state pagination di memory terisolasi, melainkan di URL via Next.js `useSearchParams()` dan `router.replace()`.
   - Menggunakan query string: `?page=1&per_page=20&sort=created_at&order=desc&q=john&status=active`.
2. **Data Fetching dengan TanStack Query v5**:
   - Data diambil via REST API dengan query key yang memuat seluruh parameter URL:
     ```typescript
     const { data, isLoading } = useQuery({
       queryKey: ['admin', 'subscriptions', searchParams],
       queryFn: () => fetchAdminSubscriptions(searchParams),
       staleTime: 10_000, // 10 detik
     });
     ```
   - Aksi mutasi (seperti Approve Payment atau Extend Subscription) menggunakan optimis update atau pemanggilan `queryClient.invalidateQueries({ queryKey: ['admin', ...] })`.

---

## 4. Keamanan, Anti-CSRF, dan Audit Logging

1. **Proteksi CSRF**:
   - Seluruh mutasi HTTP (`POST`, `PUT`, `DELETE`) mewajibkan header kustom `X-Requested-With: XMLHttpRequest` atau header token anti-CSRF ganda untuk mencegah eksploitasi cross-site request forgery.
2. **Audit Logging Otomatis (K-29)**:
   - Setiap kali handler admin berhasil mengeksekusi operasi mutasi, sebuah helper `auditService.LogAction(...)` dipanggil dalam transaksi database yang sama:
     ```go
     auditService.Log(ctx, &AuditEntry{
         ActorID:     adminID,
         ActorEmail:  adminEmail,
         Action:      "UPDATE",
         EntityType:  "plan",
         EntityID:    planID,
         BeforeState: beforeJSON,
         AfterState:  afterJSON,
         IPAddress:   c.ClientIP(),
         UserAgent:   c.Request.UserAgent(),
     })
     ```
3. **Penyembunyian Data Sensitif (Masking UU PDP No. 27/2022)**:
   - Serializer audit log secara otomatis menyaring field sensitif (`api_key`, `secret`, `password`, `card_number`) menjadi teks berbintang (`***`) sebelum disimpan dan dikirimkan ke UI viewer.
