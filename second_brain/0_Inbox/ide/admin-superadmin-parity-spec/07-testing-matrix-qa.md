# Matriks Pengujian, Quality Assurance & Verifikasi Berkala

Dokumen ini mendefinisikan strategi pengujian ketat (*test-driven & verification*), standar assertions, pengujian interaksi DOM, dan prosedur audit integritas FinTrack v2.

---

## 1. Kebijakan Integritas & Perintah Kanonik

### 1.1. Prinsip Kejujuran Hasil Uji (Strict Verification)
- **Larangan Klaim Palsu**: Dilarang menyatakan suatu fitur selesai atau berfungsi sebelum pengujian konkret dijalankan dan hasil keluaran (*exit code 0*) terbukti di terminal.
- **Koreksi Langsung**: Jika ditemukan kegagalan (*test failure*), kegagalan tersebut harus dicatat, dianalisis akar masalahnya (*root-cause*), diperbaiki kodenya, dan diuji ulang.

### 1.2. Perintah Uji Kanonik Repositori
```bash
# Pengujian Backend (Go)
cd backend && go test -v -race ./...

# Pengujian Unit & DOM Frontend (Vitest)
cd frontend && pnpm test

# Verifikasi Kompilasi & Type Check Frontend (Next.js)
cd frontend && pnpm build
```

---

## 2. Matriks Pengujian Bertingkat

| Level | Ruang Lingkup | Sasaran Modul | Perangkat Uji | Kriteria Lulus |
|---|---|---|---|---|
| **L1** | Unit & Parser Murni | `RupiahInput` formatter, `table-params.ts`, `chart-params.ts`, `search-catalog.ts` | Vitest / Go Testing | 100% fungsi murni deterministik lulus tanpa efek samping. |
| **L2** | Interaksi Komponen & DOM | `DashboardHeader`, `SearchCommand`, `ConfirmActionModal`, `DataTablePagination` | `@testing-library/react` | Event klik, navigasi keyboard, status tombol `disabled`, dan render modal terverifikasi. |
| **L3** | Realtime Bus & SSE Stream | `realtime-bus.ts`, `realtime-dispatcher.ts`, `use-realtime-stream.ts` | Vitest (Fake EventSource) | Invalidation key akurat, in-memory patch berjalan, tidak ada memory leak saat unmount. |
| **L4** | Integrasi API & RBAC Backend | `admin_handler.go`, kueri pagination terindeks, middleware otorisasi role | Go `httptest` & Test Database | Respon JSON sesuai kontrak `meta`, HTTP 403 untuk unauthorized role, kueri ber-limit. |
| **L5** | End-to-End Smoke Run | Alur paritas lengkap pada browser | Manual & Throwaway Script | URL state bookmarkable, zero layout shift, zero console error. |

---

## 3. Rincian Skenario Pengujian Kritis

### 3.1. Pengujian `RupiahInput` & Validasi Form (`RupiahInput.dom.test.tsx`)
1. **Pengetikan Normal**:
   - Ketik `"50000"` -> Nilai tampilan input menjadi `"50.000"`.
   - Handler `onChange` menerima angka integer murni `50000`.
2. **Karakter Non-Digit**:
   - Ketik `"abc50xyz000"` -> Otomatis disanitasi menjadi `"50.000"`.
3. **Penegakan Batas Minimum**:
   - Masukkan angka `0` atau `-500` -> Input menandai error, pesan *"Nilai minimal adalah 1"* muncul.
   - Tombol submit form dalam status `disabled === true`.
4. **Reaktivitas Tombol Submit**:
   - Seluruh input valid -> Tombol submit berubah menjadi `disabled === false`.
   - Klik tombol submit -> Status mutasi memicu spinner SVG dan tombol terkunci kembali (`isSubmitting`).

### 3.2. Pengujian `useDataTableUrlState` & Pagination (`use-data-table-url-state.dom.test.tsx`)
1. **Sinkronisasi Parameter URL**:
   - Panggil `setPage(3)` -> URL query string memuat `page=3`.
   - Panggil `setPerPage(50)` -> URL query string memuat `per_page=50&page=1` (otomatis reset ke halaman 1).
   - Panggil `setSort("full_name", "asc")` -> URL memuat `sort=full_name&order=asc`.
2. **Debounce 300ms**:
   - Ketik pencarian cepat `"john"` -> URL tidak langsung berubah pada 100ms pertama; tepat setelah 300ms tanpa ketikan baru, URL memuat `q=john&page=1`.
3. **Penomoran Absolut**:
   - Halaman 2 dengan per-page 20 -> Baris data pertama berindeks `21`, baris terakhir berindeks `40`.

### 3.3. Pengujian Dialog Konfirmasi (`ConfirmActionModal.dom.test.tsx`)
1. **Pemicu Dialog**:
   - Klik tombol hapus baris -> Modal konfirmasi terbuka dengan ikon `Trash2` dan tombol merah `Hapus Data`.
2. **Aksi Batal**:
   - Klik tombol Batal atau tekan `Escape` -> Modal tertutup seketika, fungsi mutasi backend **tidak dipanggil**.
3. **Aksi Konfirmasi**:
   - Klik tombol Konfirmasi -> Fungsi mutasi dipanggil sekali; tombol konfirmasi menampilkan state loading spinner SVG dan tombol terkunci (mencegah klik ganda).

### 3.4. Pengujian Realtime SSE & In-Memory Patch (`realtime-dispatcher.test.ts`)
1. **Direct Cache Patch**:
   - Kirim event SSE `user_status_updated` dengan payload `{ id: "u-1", is_active: false }`.
   - Data user `u-1` pada cache TanStack Query berubah menjadi `is_active: false` dalam 0ms.
   - Tidak ada HTTP GET request yang dipicu.
2. **Surgical Invalidation**:
   - Kirim event SSE `user_created`.
   - Hanya kueri queryKey `["admin", "users", { page: 1 }]` yang di-invalidasi untuk mengambil 20 data terbaru.
   - Query key modul lain (misal `["admin", "categories"]`) tetap utuh di cache.

---

## 4. Checklist Verifikasi Sebelum Rilis (Smoke Test Checklist)

- [ ] Seluruh halaman `/admin/*` dan `/super-admin/*` memiliki `DashboardHeader` sticky di atas konten.
- [ ] Menekan `Ctrl+K` atau `⌘K` membuka modal pencarian; navigasi panah atas/bawah dan `Enter` berfungsi.
- [ ] Toggle sidebar desktop memperkecil lebar sidebar menjadi ikon saja; reload browser mempertahankan status collapse.
- [ ] Switch mode tema (Dark, Light, System) berganti secara instan tanpa glitch latar belakang.
- [ ] Tabel menampilkan 20 data per halaman secara default; selector per-page dapat diganti ke 10, 50, 100.
- [ ] URL dapat disalin-tempel ke tab baru dengan mempertahankan halaman, sorting, filter, dan tab aktif yang sama.
- [ ] Tidak ada polling di inspect network tab browser (hanya 1 koneksi `/api/v1/realtime/stream` yang terbuka).
- [ ] Nol karakter emoticon di seluruh halaman; seluruh indikator visual menggunakan SVG resmi `lucide-react`.
- [ ] Eksekusi `cd backend && go test ./...` -> Seluruh test backend lulus (PASS).
- [ ] Eksekusi `cd frontend && pnpm test` -> Seluruh test frontend lulus (PASS).
- [ ] Eksekusi `cd frontend && pnpm build` -> Build kompilasi Next.js sukses tanpa error type check.
