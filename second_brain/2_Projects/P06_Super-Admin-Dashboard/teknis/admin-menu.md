# Struktur Menu & Navigasi Super Admin — FinTrack v2

Dokumen ini mendokumentasikan taksonomi menu navigasi, hierarki rute Next.js App Router, breadcrumbs, dan aturan antarmuka portal Super Admin FinTrack v2.

---

## 1. Hierarki Rute Next.js App Router

Semua rute portal administratif ditempatkan di bawah folder rute grup `app/(dashboard)/super-admin/`:

```
frontend/app/(dashboard)/super-admin/
├── layout.tsx                     # Layout induk: Auth SSR Guard, Admin Sidebar, Header, Breadcrumbs
├── page.tsx                       # Dashboard Utama (Ringkasan Metrik & KPI Dashboard)
├── plans/
│   ├── page.tsx                   # Manajemen Plan (Katalog Paket, Entitlement, Dynamic Pricing)
├── subscriptions/
│   ├── page.tsx                   # Manajemen Subscription (Monitoring, Extend, Force Downgrade, Cancel)
├── payments/
│   ├── page.tsx                   # Riwayat Pembayaran (Semua Transaksi, Invoices, Refund Log)
│   └── review/
│       └── page.tsx               # Antrean Review Manual Pembayaran (Bukti Transfer, Approve, Reject)
├── tokens/
│   ├── page.tsx                   # Manajemen Token (Saldo User & Riwayat Ledger Transaksi)
├── ai-gateway/
│   ├── page.tsx                   # AI Gateway Control Center (Pool Endpoints, Routing Matrix, Kill Switch)
├── audit/
│   ├── page.tsx                   # Audit Log Viewer (Jejak Audit Append-Only, JSON Diff Inspector)
└── settings/
    └── page.tsx                   # Pengaturan Platform Global (Switch Gateway, Grace Period, Flags)
```

---

## 2. Struktur Menu Navigasi Sidebar

Sidebar Super Admin diposisikan di sisi kiri layar dengan navigasi hierarkis, ikon representatif (Lucide React), dan badge notifikasi dinamis:

| Posisi | Label Menu | Rute URL | Ikon Lucide | Badge Notifikasi Dinamis | Keterangan |
|:---:|---|---|---|:---:|---|
| 1 | **Dashboard** | `/super-admin` | `LayoutDashboard` | - | Ikhtisar KPI, metrik keuangan, dan status sistem |
| 2 | **Manajemen Plan** | `/super-admin/plans` | `PackageCheck` | - | Katalog paket, centang entitlement, harga dinamis |
| 3 | **Subscription** | `/super-admin/subscriptions` | `UsersRound` | - | Monitoring pelanggan aktif, aksi perpanjangan/downgrade |
| 4 | **Pembayaran** | `/super-admin/payments` | `CreditCard` | `N Pending` (Warna Kuning) | Sub-menu: Semua Transaksi & Antrean Review Manual |
| 5 | **Token & Kuota** | `/super-admin/tokens` | `Coins` | - | Saldo token per user, ledger append-only, penyesuaian manual |
| 6 | **AI Gateway** | `/super-admin/ai-gateway` | `Cpu` | `Alert` (Bila Circuit Open) | Status pool, rute 5 use-case, emergency kill switch |
| 7 | **Jejak Audit** | `/super-admin/audit` | `ScrollText` | - | Log aktivitas administratif, diff before/after state |
| 8 | **Pengaturan** | `/super-admin/settings` | `Settings2` | - | Default payment gateway, grace period, feature flags |

---

## 3. Sistem Breadcrumbs

Setiap halaman di portal Super Admin menyediakan breadcrumbs navigasi di bagian header atas untuk memudahkan pelacakan konteks hierarki:

- Dashboard: `Super Admin > Dashboard`
- Plans: `Super Admin > Plans > Katalog Paket`
- Subscriptions: `Super Admin > Subscriptions > Monitoring Pelanggan`
- Payments Review: `Super Admin > Pembayaran > Antrean Review Manual`
- Tokens: `Super Admin > Token & Kuota > Saldo Pengguna`
- AI Gateway: `Super Admin > AI Gateway > Status & Routing`
- Audit: `Super Admin > Jejak Audit > Log Perubahan`
- Settings: `Super Admin > Pengaturan > Konfigurasi Global`

---

## 4. Indikator Status & Visual Feedback

1. **Badge Notifikasi Sidebar**:
   - Menu **Pembayaran** menampilkan badge angka kuning bila ada bukti transfer yang memerlukan verifikasi manual (`status = pending_review`).
   - Menu **AI Gateway** menampilkan badge peringatan merah berdenyut (*pulse animation*) apabila terdapat endpoint dalam keadaan `DEAD` atau circuit breaker dalam keadaan `OPEN`.
2. **Kemudahan Akses Kembali (User Switcher)**:
   - Pada bagian bawah sidebar admin terdapat tombol aksi cepat `Kembali ke Dashboard FinTrack User` (`/dashboard`), memungkinkan Super Admin melihat tampilan aplikasi sebagai pengguna reguler tanpa perlu re-login.
