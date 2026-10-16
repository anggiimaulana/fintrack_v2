# Spesifikasi Teknis: Arsitektur Shell & Header Terpadu

Dokumen ini mendefinisikan arsitektur antarmuka tingkat atas (*shell*) dan navigasi untuk portal `admin` dan `super_admin`, menyelaraskannya dengan standar visual dan fungsional modul `user` pada FinTrack v2.

---

## 1. Tujuan & Prinsip Desain
1. **Unifikasi Shell**: Mengeliminasi duplikasi layout dengan mengadopsi satu kerangka `DashboardShell` yang melayani ketiga peran (`user`, `admin`, `super_admin`).
2. **Konsistensi Navigasi**: Menyediakan `DashboardHeader` sticky di setiap halaman dengan indikator breadcrumb, pencarian perintah terpadu, notifikasi realtime, pengganti tema, dan menu profil.
3. **Kontrol Sidebar Desktop**: Menyediakan mekanisme buka/tutup (collapse/expand) sidebar desktop dengan penyimpanan status lokal (`localStorage`), tanpa mode drawer mobile (skip per kebutuhan).
4. **Bebas Emoticon**: Seluruh antarmuka menggunakan ikon SVG resmi dari pustaka `lucide-react`.

---

## 2. Struktur Komponen Shell

### 2.1. Komponen `DashboardShell`
- **Lokasi Berkas**: `frontend/components/layout/DashboardShell.tsx`
- **Peran**: Pembungkus tingkat tinggi di dalam `app/(dashboard)/*/layout.tsx`.
- **Status State**:
  - `collapsed` (boolean): Menentukan apakah sidebar dalam status diciutkan (`w-16`) atau dibuka (`w-64`).
  - Nilai awal dibaca dari `localStorage.getItem("fintrack_sidebar_collapsed")` dengan nilai default `false`.
  - Transisi CSS: `transition-all duration-200 ease-in-out`.

```tsx
export interface DashboardShellProps {
  children: React.ReactNode;
  sidebar: React.ReactNode;
  role: "user" | "admin" | "super_admin";
}
```

### 2.2. Hirarki Layout App Router
```
app/(dashboard)/
├── layout.tsx                      # Root dashboard provider (Theme, Realtime, Session)
├── user/
│   └── layout.tsx                  # DashboardShell dengan <UserSidebar />
├── admin/
│   └── layout.tsx                  # DashboardShell dengan <AdminSidebar />
└── super-admin/
    └── layout.tsx                  # DashboardShell dengan <SuperAdminSidebar /> + <GatewayAlertListener />
```

---

## 3. Komponen `DashboardHeader`

### 3.1. Penataan Layout & Posisi
- Dimensi: Tinggi tetap `h-14` (56px).
- Penempatan: `sticky top-0 z-40 w-full border-b-2 border-foreground bg-card px-4 lg:px-6`.
- Tata Letak: Flexbox horizontal `flex items-center justify-between`.

### 3.2. Komposisi Elemen Header

```
+---------------------------------------------------------------------------------------------------------+
| [Toggle Sidebar] | [Breadcrumb / Judul Halaman]    ...    [Ctrl+K Cari] [Notifikasi] [Tema] [Avatar Profil] |
+---------------------------------------------------------------------------------------------------------+
```

1. **Sisi Kiri**:
   - **Tombol Toggle Sidebar**:
     - Ikon saat terbuka: `PanelLeftClose` (lucide-react).
     - Ikon saat diciutkan: `PanelLeftOpen` (lucide-react).
     - Atribut aksesibilitas: `aria-label="Toggle Sidebar"`, `aria-expanded={!collapsed}`.
   - **Breadcrumb & Judul Halaman**:
     - Membaca `pathname` via hook `usePathname()`.
     - Melakukan translasi rute ke nama label bahasa Indonesia yang ramah pengguna.
     - Contoh: `/super-admin/tokens` -> `Super Admin / Token & Kuota AI`.

2. **Sisi Kanan**:
   - **Pencarian Modul/Fitur (`Ctrl+K`)**:
     - Tombol visual bergaya input mini: `border-2 border-foreground bg-background px-3 py-1.5 text-sm`.
     - Teks: "Cari modul atau aksi..." dengan badge shortcut `Ctrl K` / `⌘ K`.
     - Menekan tombol atau shortcut keyboard membuka dialog `SearchCommand`.
   - **Notifikasi Realtime (`NotificationBell`)**:
     - Ikon: `Bell` (lucide-react).
     - Badge merah: Jumlah notifikasi belum dibaca (`unread_count`).
     - Pembaruan otomatis via event SSE tanpa polling.
     - Mengklik tombol mengarahkan ke halaman notifikasi atau membuka drawer ringkasan notifikasi.
   - **Switch Mode Tema (`ThemeToggle`)**:
     - Mendukung 3 pilihan: `light`, `dark`, `system`.
     - Dropdown menu atau toggle berurutan dengan ikon dinamis: `Sun` (light), `Moon` (dark), `Laptop` (system).
     - Mencegah hydration mismatch via status `mounted` React.
   - **Menu Profil Pengguna (`ProfileMenu`)**:
     - Komponen Avatar menampilkan inisial pengguna atau foto resmi jika tersedia.
     - Dropdown memuat:
       - Nama lengkap pengguna & alamat email.
       - Badge penanda peran: `ADMIN` atau `SUPER ADMIN`.
       - Menu aksi: `Pengaturan Profil`, `Sesi Aktif`, `Log Audit Pribadi`.
       - Separator visual `border-t border-border`.
       - Tombol `Keluar` (Logout) yang memicu dialog konfirmasi keluar.

---

## 4. Modul Pencarian & Shortcut Keyboard (`SearchCommand`)

### 4.1. Katalog Pencarian Berbasis Peran (`lib/search-catalog.ts`)
Setiap entri pencarian memiliki struktur data kontrak:

```typescript
export interface SearchEntry {
  id: string;
  label: string;
  group: "Modul" | "Fitur" | "Aksi";
  href: string;
  iconName: string; // Nama ikon lucide-react resmi
  keywords: string[];
  roles: ("user" | "admin" | "super_admin")[];
  description?: string;
  actionId?: string; // Jika merupakan aksi instan (misal: "open_adjust_token_modal")
}
```

### 4.2. Matriks Indeks Menu Admin & Super Admin

| ID | Label | Grup | Rute / Target | Peran Diizinkan | Kata Kunci |
|---|---|---|---|---|---|
| `adm-dash` | Dashboard Admin | Modul | `/admin` | `admin`, `super_admin` | statistik, ringkasan, home, transaksi |
| `adm-users` | Manajemen Pengguna | Modul | `/admin/users` | `admin`, `super_admin` | user, akun, suspend, aktivasi, daftar pengguna |
| `adm-cat-exp` | Kategori Pengeluaran | Fitur | `/admin/categories/expense` | `admin`, `super_admin` | kategori, belanja, pos biaya |
| `adm-cat-inc` | Kategori Pemasukan | Fitur | `/admin/categories/income` | `admin`, `super_admin` | kategori, gaji, dividen, omset |
| `adm-cat-sav` | Kategori Tabungan | Fitur | `/admin/categories/savings` | `admin`, `super_admin` | pos tabungan, investasi, alokasi |
| `adm-prov` | Monitoring Provider | Modul | `/admin/providers` | `admin`, `super_admin` | midtrans, xendit, openai, anthropic |
| `sa-dash` | Dashboard Super Admin | Modul | `/super-admin` | `super_admin` | mrr, kpi, overview, finansial |
| `sa-tokens` | Token & Kuota AI | Modul | `/super-admin/tokens` | `super_admin` | saldo token, ledger, kredit, penyesuaian |
| `sa-sub` | Manajemen Langganan | Modul | `/super-admin/subscriptions` | `super_admin` | paket, langganan aktif, perpanjang, downgrade |
| `sa-pay` | Pembayaran & Review | Modul | `/super-admin/payments` | `super_admin` | mutasi, transfer manual, bukti bayar, verifikasi |
| `sa-plans` | Manajemen Paket (Plans) | Modul | `/super-admin/plans` | `super_admin` | katalog paket, entitlement, batas kuota |
| `sa-audit` | Jejak Audit (Audit Logs) | Modul | `/super-admin/audit` | `super_admin` | log aktivitas, diff json, uu pdp, inspeksi |
| `sa-ai-gw` | Operasional AI Gateway | Modul | `/super-admin/ai-gateway` | `super_admin` | circuit breaker, kill switch, endpoint ai, latensi |
| `sa-settings` | Pengaturan Sistem | Modul | `/super-admin/settings` | `super_admin` | konfigurasi global, payment gateway, feature flag |

### 4.3. Mekanisme Dialog & Interaksi Keyboard
1. **Shortcut Listener Global**:
   - `useEffect` pada tingkat header mendaftarkan event listener `keydown` ke objek `window`.
   - Deteksi tombol: `(event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k'`.
   - Mencegah default event browser (`event.preventDefault()`) dan memicu `setSearchOpen(true)`.
2. **Navigasi Keyboard dalam Dialog**:
   - `ArrowDown` & `ArrowUp`: Mengubah indeks baris aktif (`activeIndex`).
   - `Enter`: Menjalankan navigasi router ke `href` entri yang dipilih, atau memancarkan event custom untuk `actionId`.
   - `Escape`: Menutup dialog seketika dan mengembalikan fokus keyboard ke elemen sebelumnya.
3. **Pencarian Reaktif Cepat**:
   - Normalisasi teks: lowercase, pembersihan tanda baca, pencocokan substring terhadap `label` dan `keywords`.
   - Hasil dikelompokkan berdasarkan grup: `Modul`, `Fitur`, `Aksi`.

---

## 5. Sinkronisasi Status Sidebar Desktop

1. **Penyimpanan Status**:
   - Kunci penyimpanan: `fintrack_sidebar_collapsed`.
   - Saat state berubah, tulis langsung ke `localStorage.setItem("fintrack_sidebar_collapsed", String(nextState))`.
2. **Layout Adaptif**:
   - Saat `collapsed === false`:
     - Lebar sidebar: `w-64` (256px).
     - Label teks menu ditampilkan lengkap di samping ikon.
   - Saat `collapsed === true`:
     - Lebar sidebar: `w-16` (64px).
     - Label teks disembunyikan; hanya ikon SVG yang tampak di tengah.
     - Setiap ikon dilengkapi `<Tooltip>` dengan delay 100ms untuk memunculkan nama modul saat pointer hover.
3. **Pemberitahuan Komponen**:
   - Komponen chart atau tabel memantau perubahan ukuran kontainer via `ResizeObserver` alami dari CSS Flexbox/Grid tanpa re-render manual yang berat.
