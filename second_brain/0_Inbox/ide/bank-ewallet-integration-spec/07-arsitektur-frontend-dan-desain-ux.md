# 07 — Arsitektur Frontend & Desain UX

**Topik:** Rancangan Antarmuka Pengguna Next.js, Komponen shadcn/Tailwind, Alur Impor Berkas, & Visualisasi Data  
**Tanggal:** 2026-10-20  
**Status:** DRAFTED & REVIEWED  

---

## 1. Arsitektur Halaman & Routing Frontend

Fitur integrasi perbankan dan e-wallet diimplementasikan di folder `frontend/src/app/(dashboard)/accounts/` dengan struktur modular:

```
frontend/src/app/(dashboard)/accounts/
├── page.tsx                       # Halaman utama: Ringkasan Net Worth, Grid Kartu Akun, Cash Flow
├── layout.tsx                     # Entitlement guard layout (redirect ke paywall jika non-Business)
├── batches/
│   └── page.tsx                   # Halaman audit riwayat batch impor & tombol rollback
└── components/
    ├── net-worth-card.tsx         # Hero card total kekayaan bersih + donut chart aset
    ├── account-grid.tsx           # Grid kartu rekening bank & e-wallet
    ├── account-card.tsx           # Satu kartu rekening (logo, saldo, masked number, action menu)
    ├── add-account-modal.tsx      # Modal tambah rekening / e-wallet baru
    ├── import-statement-modal.tsx # Wizard modal impor berkas mutasi (upload, parse, pre-commit)
    ├── sankey-flow-chart.tsx      # Visualisasi diagram arus kas (income -> transfer -> expense)
    ├── top-merchants-chart.tsx    # Bar chart horizontal pengeluaran vendor terbesar
    └── batch-rollback-dialog.tsx  # Dialog konfirmasi pembatalan batch impor
```

---

## 2. Alur Pengalaman Pengguna (User Flow)

### 2.1 Alur Pembuatan Akun Baru (`add-account-modal.tsx`)
```
[Klik "+ Tambah Akun"] 
         │
         ▼
[Pilih Institusi: Grid Logo Bank/E-Wallet] (BCA, Mandiri, Jago, SeaBank, GoPay, DANA, ShopeePay)
         │
         ▼
[Input Metadata Akun]
- Nama Rekening (misal: "BCA Tabungan Operasional")
- Nomor Rekening Mask (4 digit terakhir, misal: "****8821")
- Saldo Awal (misal: Rp 5.000.000)
- Warna Kartu (color picker palette)
         │
         ▼
[Simpan Akun] ──► Saldo langsung tercatat di Net Worth Hero Card
```

### 2.2 Alur Impor Berkas Mutasi 3 Langkah (`import-statement-modal.tsx`)
```
Step 1: Upload
┌─────────────────────────────────────────────────────────────┐
│  PILIH REKENING TUJUAN: [ Bank Central Asia (BCA)   ▼ ]     │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │   [Drag & Drop E-Statement PDF / CSV di sini]         │  │
│  │   atau klik untuk memilih berkas dari komputer        │  │
│  │   Format didukung: PDF e-statement myBCA, CSV KlikBCA │  │
│  └───────────────────────────────────────────────────────┘  │
│  [i] Cara unduh e-Statement resmi BCA: Panduan 1-Klik       │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼ (Sistem memproses pre-flight)
Step 2: Pre-Commit Preview & Review
┌─────────────────────────────────────────────────────────────┐
│ HASIL PARSING: Rekening BCA (1 Okt 2026 - 15 Okt 2026)      │
├─────────────────────────────────────────────────────────────┤
│ Ringkasan:                                                  │
│ - Total Transaksi Ditemukan : 42 transaksi                  │
│ - Transaksi Baru            : 38 transaksi (akan diimpor)   │
│ - Duplikat Dilewati         : 4 transaksi (sudah ada di DB) │
│ - Terdeteksi Transfer       : 2 pasang (ke GoPay & Jago)    │
│                                                             │
│ [ Tabel Preview 5 Transaksi Pertama dengan Kategori AI ]    │
│                                                             │
│ [ Batalkan ]                    [ Konfirmasi & Impor (38) ] │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
Step 3: Sukses & Update Realtime
Saldo Akun BCA bertambah/berkurang sesuai mutasi, Net Worth ter-update, dan RAG siap ditanya.
```

---

## 3. Komponen Visualisasi Data Interaktif

### 3.1 Kartu Hero Net Worth (`net-worth-card.tsx`)
- Menampilkan total gabungan saldo seluruh akun aktif dalam mata uang preferensi pengguna (default `IDR`).
- Disertai metrik perbandingan bulanan: *"▲ +12.4% dibanding bulan lalu"*.
- Tab filter cepat: *"Semua Akun"*, *"Hanya Bank"*, *"Hanya E-Wallet"*.

### 3.2 Sankey Cash Flow Chart (`sankey-flow-chart.tsx`)
Menggunakan pustaka visualisasi yang sudah ada di frontend (Recharts / SVG):
- **Sisi Kiri (Inflow):** Sumber pendapatan (Gaji, Proyek, Dividen).
- **Sisi Tengah (Nodes):** Rekening penampung (BCA, Mandiri, Bank Jago).
- **Cabang Tengah (Transfers):** Garis penghubung transfer internal (BCA $\rightarrow$ GoPay, BCA $\rightarrow$ SeaBank).
- **Sisi Kanan (Outflow):** Pos kategori pengeluaran (Makanan, Tagihan Listrik, Hiburan, Belanja).

### 3.3 Top Merchants Breakdown (`top-merchants-chart.tsx`)
- Menampilkan 5 merchant dengan total transaksi terbesar.
- Interaktif: Klik pada salah satu merchant (contoh: *"Tokopedia"*) akan membuka drawer transaksi terfilter yang menampilkan seluruh pembelian di merchant tersebut beserta akun pembayarannya.

---

## 4. Penyesuaian Antarmuka Transaksi yang Sudah Ada (`/transactions`)

Tabel transaksi utama diperkaya tanpa mengubah konsistensi layout:
1. **Dropdown Filter Akun:** Di samping filter Kategori dan Tanggal, ditambahkan filter *"Semua Rekening"* dengan opsi memilih rekening tertentu.
2. **Badge Akun:** Setiap baris transaksi menampilkan badge kecil logo institusi (misal: icon BCA biru atau icon GoPay hijau) di samping tanggal.
3. **Indikator Transfer Internal:** Transaksi bertipe `transfer` memiliki badge dua panah (`↔`) dan tidak memiliki label pengeluaran/pendapatan merah/hijau biasa, melainkan abu-abu netral untuk menegaskan transaksi non-beban.

---

## 5. Paywall Modal & Upsell Gate (Non-Business Users)

Ketika pengguna dengan paket Free atau Pro mengakses `/accounts`:
- Halaman tidak menampilkan error 404/500, melainkan render komponen `BusinessPaywallView.tsx`:
  - Visual ilustrasi interaktif dashboard multi-akun.
  - Kartu perbandingan fitur: Free vs Pro vs Business.
  - Testimonial pengguna bisnis.
  - Tombol CTA: **"Tingkatkan ke Paket Business"** yang membuka modal checkout Stripe/Midtrans/Manual Transfer untuk paket Business.
