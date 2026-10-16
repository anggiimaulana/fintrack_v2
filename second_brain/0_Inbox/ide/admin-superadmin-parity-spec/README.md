# Master Spec: Paritas Tampilan & Performa Admin & Super Admin FinTrack v2

Paket spesifikasi komprehensif untuk standarisasi portal **Admin** dan **Super Admin** agar memiliki kesetaraan fungsional, performa tinggi, dan pengalaman pengguna (UX) yang identik dengan portal **User** di FinTrack v2.

---

## 1. Daftar Dokumen Spesifikasi Modular

Folder ini memuat 8 dokumen spesifikasi teknis mendalam yang siap dieksekusi:

| Dokumen | Topik Pembahasan | Deskripsi Singkat |
|---|---|---|
| [`01-arsitektur-shell-header.md`](./01-arsitektur-shell-header.md) | **Shell & Header Terpadu** | `DashboardShell` terpadu, `DashboardHeader` sticky, toggle sidebar desktop (`w-64` / `w-16`), pencarian modul `Ctrl+K`, switch tema 3-mode (dark/light/system), profil menu, dan notifikasi realtime. |
| [`02-datatable-url-state-engine.md`](./02-datatable-url-state-engine.md) | **Data Table & URL State Engine** | Standarisasi tabel ber-KPI statistik, sorting, server pagination default 20 data bernomor, debounce pencarian & filter 300ms, rumus penomoran urut absolut, dan sinkronisasi URL dua arah. |
| [`03-analytics-charts-insights.md`](./03-analytics-charts-insights.md) | **Analitik & Grafik Interaktif** | Integrasi Recharts, pemetaan grafik pada halaman target Admin & Super Admin, sinkronisasi state chart ke URL params, dan kartu *Smart Insight* bermakna operasional. |
| [`04-form-controls-validasi-guards.md`](./04-form-controls-validasi-guards.md) | **Kontrol Form & Modal Konfirmasi** | Parser otomatis Rupiah (`50000` -> `50.000`), batas minimum input $\ge 1$, validasi reaktif tombol simpan/aksi dinonaktifkan (`disabled`), dan modal konfirmasi hapus/suspend/nonaktif (`ConfirmActionModal`). |
| [`05-realtime-caching-rbac-bus.md`](./05-realtime-caching-rbac-bus.md) | **Realtime Caching & RBAC Bus** | Arsitektur SSE tunggal bebas polling, strategi caching hybrid (in-memory patch 0ms + surgical query invalidation), dynamic RBAC update tanpa refresh browser, dan mitigasi kebocoran memori. |
| [`06-implementation-plan.md`](./06-implementation-plan.md) | **Rencana Implementasi Bertahap** | Rencana eksekusi 6 fase sekuensial, inventaris berkas yang dibuat/diubah, matriks dependensi, dan kriteria penerimaan tiap fase. |
| [`07-testing-matrix-qa.md`](./07-testing-matrix-qa.md) | **Matriks Pengujian & QA Berkala** | Kebijakan verifikasi ketat, pengujian bertingkat (L1 s/d L5), pengujian interaksi DOM, skenario uji formulir & SSE, serta checklist smoke test sebelum rilis. |
| [`08-mitigasi-resiko.md`](./08-mitigasi-resiko.md) | **Mitigasi Risiko & Zero Downtime** | Matriks risiko teknis (R-01 s/d R-08), strategi deployment tanpa downtime, kompatibilitas mundur API, penanganan failover, dan rencana rollback kontainer. |

---

## 2. Prinsip Rekayasa Utama (Non-Negotiable Constraints)

1. **Efisiensi Kueri & Database**:
   - Tidak ada pemborosan kueri ke database. Seluruh pengambilan data tabular memanfaatkan limit-offset server-side (`LIMIT 20 OFFSET X`) terindeks.
2. **Koneksi Realtime Murni (Zero Polling)**:
   - Polling berulang (`setInterval` / `refetchInterval`) dilarang 100%. Sinkronisasi data mutasi mengandalkan aliran SSE `/api/v1/realtime/stream` yang terdistribusi ke bus internal klien.
3. **Penyajian Rupiah & Input Numerik**:
   - Seluruh nominal uang diformat dengan kaidah Indonesia (titik pemisah ribuan).
   - Seluruh input angka wajib bernilai minimal 1 (`min=1`).
4. **Validasi Form Reaktif**:
   - Tombol submit/aksi otomatis terkunci (`disabled`) jika input wajib belum lengkap atau belum valid, serta mencegah eksekusi ganda (*anti-double-click*).
5. **Aman dari Aksi Destruktif**:
   - Seluruh aksi hapus, penangguhan (suspend), atau penonaktifan data wajib menampilkan dialog modal konfirmasi resmi dengan keterangan konsekuensi yang jelas.
6. **Bebas Emoticon**:
   - Nol karakter emotikon pada seluruh kode antarmuka. 100% representasi visual menggunakan ikon resmi SVG dari pustaka `lucide-react`.
7. **Verifikasi Jujur & Berkala**:
   - Menjalankan perintah uji kanonik `cd backend && go test ./...` dan `cd frontend && pnpm test && pnpm build` secara berkala tanpa klaim palsu.
