# Rencana Implementasi: Paritas Admin & Super Admin FinTrack v2

Rencana implementasi ini dirancang dengan pendekatan bertahap (*phased rollout*), memisahkan fondasi bersama (*shared foundation*) dari integrasi modul spesifik guna menjamin stabilitas sistem, kemudahan review kode, dan *zero downtime*.

---

## 1. Rangkaian Fase Eksekusi

```
+---------------------------------------------------------------------------------------------------------+
| FASE 1: Shared Shell & Header Foundation                                                                 |
| (DashboardShell, DashboardHeader, SearchCommand RBAC, Sidebar Collapse, ThemeToggle, ProfileMenu)       |
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
| FASE 2: Form Controls, Validasi Reaktif & Modal Konfirmasi                                              |
| (RupiahInput parser & min=1, Reactive Submit Button disabled guard, ConfirmActionModal)                 |
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
| FASE 3: Data Table Layout & URL State Engine                                                            |
| (Backend Pagination 20 & Search, DataTableLayout, Numbered Pagination, Multi-param URL State & Debounce)|
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
| FASE 4: Analitik, Grafik Interaktif & Smart Insight                                                     |
| (Chart URL State binding, SmartInsightCard, Implementasi grafik di Admin & Super Admin)                |
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
| FASE 5: Realtime Caching (SSE) & Dynamic RBAC Bus                                                       |
| (Backend Realtime Emitters, In-Memory Patch Dispatcher, Surgical Invalidation, Dynamic RBAC Guard)     |
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
| FASE 6: Migrasi Halaman Menyeluruh & QA Paritas Penuh                                                   |
| (Pembaruan seluruh halaman /admin/* dan /super-admin/*, Regression test suite, Final Smoke Run)         |
+---------------------------------------------------------------------------------------------------------+
```

---

## 2. Rincian Teknis per Fase

### Fase 1: Shared Shell & Header Foundation
- **Tujuan**: Membangun kerangka navigasi terpadu untuk ketiga peran pengguna.
- **Berkas Dibuat / Diubah**:
  - `frontend/components/layout/DashboardShell.tsx` (Baru: pembungkus utama).
  - `frontend/components/layout/DashboardHeader.tsx` (Baru: header terpadu dengan role awareness).
  - `frontend/components/layout/SearchCommand.tsx` (Diperbarui: dukungan RBAC & shortcut aksi).
  - `frontend/lib/search-catalog.ts` (Diperbarui: indeks menu & fitur admin/super admin).
  - `frontend/app/(dashboard)/admin/layout.tsx` (Diperbarui: adopsi DashboardShell & DashboardHeader).
  - `frontend/app/(dashboard)/super-admin/layout.tsx` (Diperbarui: adopsi DashboardShell & DashboardHeader).
- **Kriteria Keberhasilan**:
  - Header muncul konsisten di semua rute `/admin/*` dan `/super-admin/*`.
  - Shortcut `Ctrl+K` menampilkan menu yang sesuai peran aktif.
  - Sidebar desktop dapat diciutkan dan dimekarkan; status tersimpan di `localStorage`.
  - Tema 3-mode (light/dark/system) berganti mulus tanpa kedip (*flicker*).

### Fase 2: Form Controls, Validasi Reaktif & Modal Konfirmasi
- **Tujuan**: Menjamin integritas data numerik, format Rupiah otomatis, dan perlindungan aksi kritis.
- **Berkas Dibuat / Diubah**:
  - `frontend/components/shared/RupiahInput.tsx` (Diperbarui: penegakan batas `min=1` dan visual prefix `Rp`).
  - `frontend/components/shared/ConfirmActionModal.tsx` (Baru: dialog konfirmasi standar delete/suspend/deactivate).
  - `frontend/components/admin/CategoryForm.tsx` (Diperbarui: integrasi RupiahInput & validasi reaktif).
  - `frontend/components/super-admin/plans/PlanFormDialog.tsx` (Diperbarui: integrasi RupiahInput & validasi reaktif).
  - `frontend/components/super-admin/tokens/AdjustTokensDialog.tsx` (Diperbarui: validasi min=1 & guard tombol).
- **Kriteria Keberhasilan**:
  - Input `50000` otomatis terformat menjadi `50.000` dan mengembalikan angka integer `50000`.
  - Angka $< 1$ menampilkan helper error dan menonaktifkan tombol simpan.
  - Seluruh aksi hapus, suspend, dan nonaktif memunculkan `ConfirmActionModal`.

### Fase 3: Data Table Layout & URL State Engine
- **Tujuan**: Menghadirkan tabel data dengan statistik, pagination kelipatan 20, penomoran urut, dan sinkronisasi URL.
- **Berkas Dibuat / Diubah**:
  - `backend/internal/delivery/http/handler/admin_handler.go` (Diperbarui: penambahan parameter kueri pagination, search, status).
  - `backend/internal/usecase/user_usecase.go` (Diperbarui: kueri ber-limit dan total counter).
  - `frontend/components/data-table/DataTableLayout.tsx` (Baru: wrapper tabel standar ber-KPI).
  - `frontend/components/data-table/data-table-pagination.tsx` (Diperbarui: selector 20 data/halaman & windowing angka).
  - `frontend/components/admin/UserTable.tsx` (Diperbarui: adopsi DataTableLayout).
  - `frontend/app/(dashboard)/admin/users/page.tsx` (Diperbarui: integrasi useDataTableUrlState).
- **Kriteria Keberhasilan**:
  - URL otomatis memuat `?page=1&per_page=20&sort=created_at&order=desc`.
  - Input search di-debounce 300ms sebelum request API dipicu.
  - Penomoran baris absolut akurat pada setiap halaman.

### Fase 4: Analitik, Grafik Interaktif & Smart Insight
- **Tujuan**: Menyediakan visualisasi metrik yang dapat di-share via URL beserta kartu insight bermakna.
- **Berkas Dibuat / Diubah**:
  - `frontend/components/shared/SmartInsightCard.tsx` (Baru: komponen kartu wawasan data).
  - `frontend/lib/chart-params.ts` (Diperbarui: perluasan grup & periode).
  - `frontend/app/(dashboard)/admin/page.tsx` (Diperbarui: penambahan grafik transaksi & kategori).
  - `frontend/app/(dashboard)/admin/users/page.tsx` (Diperbarui: grafik pertumbuhan pengguna).
  - `frontend/app/(dashboard)/admin/providers/page.tsx` (Diperbarui: grafik latensi & success rate).
  - `frontend/app/(dashboard)/super-admin/page.tsx` & sub-halamannya (Diperbarui: adopsi SmartInsightCard & URL state).
- **Kriteria Keberhasilan**:
  - Mengubah periode grafik (misal 3 bulan) memperbarui URL `?period=3months`.
  - Kartu insight menampilkan ringkasan performa dan rekomendasi aksi operasional.
  - Tidak ada lag atau layout break saat kontainer di-resize.

### Fase 5: Realtime Caching (SSE) & Dynamic RBAC Bus
- **Tujuan**: Sinkronisasi instan tanpa polling dan update hak akses zero-downtime.
- **Berkas Dibuat / Diubah**:
  - `backend/pkg/contracts/events.go` (Diperbarui: penambahan event SSE domain admin).
  - `frontend/lib/realtime-dispatcher.ts` (Diperbarui: dispatcher in-memory patch & surgical invalidation).
  - `frontend/lib/session-context.tsx` (Diperbarui: dynamic role/entitlement listener).
- **Kriteria Keberhasilan**:
  - Perubahan status user atau kuota langsung ter-update di layar dalam 0ms tanpa request HTTP GET tambahan.
  - Demosi peran atau suspend user memicu penyesuaian menu atau logout aman secara instan.
  - Tidak ada polling interval yang berjalan di tab browser.

### Fase 6: Migrasi Menyeluruh & QA Paritas Penuh
- **Tujuan**: Audit menyeluruh seluruh halaman admin & super admin untuk memastikan 100% kepatuhan spesifikasi.
- **Kriteria Keberhasilan**:
  - 100% halaman bebas dari emoticon, 100% SVG resmi.
  - Seluruh test suite (`go test ./...` dan `pnpm test`) lulus 100%.
  - Build frontend lulus bersih (`pnpm build`).
