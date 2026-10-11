# Laporan Audit Kepatuhan Cursor Pointer & Aksesibilitas UI

## 1. Latar Belakang & Masalah Desain Antarmuka

Sebelum fase P07, terdapat inkonsistensi pengalaman pengguna (UX) pada antarmuka frontend FinTrack v2:
1. **Ketidakjelasan Status Interaktif**: Beberapa tombol, tab navigasi, checkbox custom, dan header modal tidak memunculkan indikator kursor pointer browser (`cursor: pointer`), membingungkan pengguna apakah elemen tersebut dapat diklik.
2. **Ketiadaan Indikator Nonaktif**: Tombol submit form dalam kondisi proses/disabled mempertahankan pointer standar atau pointer aktif tanpa feedback visual yang jelas.
3. **Penyalahgunaan Pointer pada Elemen Statis**: Beberapa container kartu info dan label teks diberi utility class pointer yang memicu harapan klik palsu (*false affordance*).
4. **Celah Aksesibilitas (WCAG 2.1 AA)**: Tombol berbasis ikon murni (seperti tombol edit/delete dengan ikon Lucide) tidak memiliki atribut `aria-label`, menghambat pengguna pembaca layar (*screen reader*).

---

## 2. Standar Aturan Kursor & Interaktivitas FinTrack v2

Seluruh komponen antarmuka FinTrack v2 wajib mengikuti matriks aturan Tailwind CSS berikut:

| Tipe Elemen | Kondisi / State | Utility Class Wajib | Perilaku Interaktif |
|---|---|---|---|
| `<button>` | Normal / Aktif | `cursor-pointer transition-colors` | Berubah warna/elevasi saat hover |
| `<button>` | Disabled / Loading | `cursor-not-allowed opacity-50 pointer-events-none` (atau handle via disabled) | Tidak merespons hover/klik |
| `<a href="...">` | Normal / Aktif | `cursor-pointer underline-offset-4 hover:underline` | Menampilkan link status |
| `<input type="checkbox/radio">` | Normal / Aktif | `cursor-pointer accent-primary` | Kursor pointer saat hovering kontrol |
| `<label>` untuk form control | Terikat `htmlFor` | `cursor-pointer select-none` | Mengaktifkan kontrol saat label diklik |
| Baris Tabel `<TableRow>` | Memiliki handler `onClick` | `cursor-pointer hover:bg-muted/50 transition-colors` | Baris dapat diklik untuk detail |
| Baris Tabel `<TableRow>` | Statis (Display Only) | `cursor-default` | Tidak ada highlight hover berlebih |
| Kartu / Card Statis | Tanpa tautan | `cursor-default select-text` | Dilarang keras diberi `cursor-pointer` |
| Badge / Tag Status | Statis non-filter | `cursor-default` | Menampilkan informasi saja |
| Badge / Tag Filterable | Memiliki handler klik | `cursor-pointer hover:opacity-80` | Menyaring data |

---

## 3. Matriks Hasil Audit Komponen & Perbaikan File

Tim telah melakukan audit menyeluruh pada seluruh komponen UI utama di `frontend/components/` dan `frontend/app/`:

| No | Lokasi Komponen / Halaman | Masalah Ditemukan | Tindakan Perbaikan P07 | Status Verifikasi |
|---|---|---|---|:---:|
| 1 | `components/ui/button.tsx` | State disabled tidak eksplisit `cursor-not-allowed` | Ditambahkan varian CVA disabled: `disabled:cursor-not-allowed disabled:opacity-50` | ✅ Lolos |
| 2 | `components/ui/dialog.tsx` | Tombol close `X` tidak memiliki `cursor-pointer` & `aria-label` | Ditambahkan `cursor-pointer` dan `aria-label="Tutup dialog"` | ✅ Lolos |
| 3 | `components/data-table/data-table-pagination.tsx` | Tombol navigasi next/prev pagination kursor default | Ditambahkan `cursor-pointer` dan disabled guard | ✅ Lolos |
| 4 | `components/data-table/data-table-column-header.tsx` | Header kolom yang dapat di-sort kursor text | Ditambahkan `cursor-pointer select-none` | ✅ Lolos |
| 5 | `app/(dashboard)/transactions/page.tsx` | Tombol filter preset kategori kursor standar | Diberikan class `cursor-pointer` pada semua badge filter | ✅ Lolos |
| 6 | `app/(dashboard)/budgets/budget-card.tsx` | Kartu anggaran memakai `cursor-pointer` padahal statis | Dihapus `cursor-pointer`, dikembalikan ke `cursor-default` | ✅ Lolos |
| 7 | `components/sidebar.tsx` | Menu navigasi item beberapa kekurangan kursor | Diseragamkan via wrapper NavLink dengan `cursor-pointer` | ✅ Lolos |
| 8 | `components/ui/checkbox.tsx` | Checkbox custom Radix UI kursor default | Ditambahkan `cursor-pointer` pada root dan label pembungkus | ✅ Lolos |
| 9 | `components/ui/dropdown-menu.tsx` | DropdownMenuItem tidak eksplisit kursor | Ditambahkan `cursor-pointer select-none` | ✅ Lolos |
| 10| `app/(dashboard)/settings/subscription/page.tsx` | Tombol "Pilih Paket" saat paket aktif kursor pointer | Diubah menjadi `cursor-not-allowed` dengan label "Paket Aktif" | ✅ Lolos |

---

## 4. Audit Aksesibilitas (a11y) & Navigasi Keyboard

Sebagai bagian dari komitmen terhadap inklusivitas dan standar web modern, audit a11y mencakup tiga aspek kritis:

### 4.1. Tombol Ikon (Icon-Only Buttons)
Seluruh tombol yang tidak memiliki label teks terlihat wajib menyertakan atribut `aria-label` atau `<span className="sr-only">`:
```tsx
// SEBELUM (Gagal Audit a11y):
<Button variant="ghost" size="icon" onClick={() => deleteTx(id)}>
  <Trash2 className="h-4 w-4" />
</Button>

// SESUDAH (Lolos Audit a11y):
<Button 
  variant="ghost" 
  size="icon" 
  className="cursor-pointer hover:text-destructive" 
  aria-label="Hapus transaksi"
  onClick={() => deleteTx(id)}
>
  <Trash2 className="h-4 w-4" />
  <span className="sr-only">Hapus transaksi</span>
</Button>
```

### 4.2. Focus Trap pada Modal & Sheet Dialog
Semua komponen dialog (tambah transaksi, rincian langganan, konfirmasi hapus) menggunakan Radix UI Primitives yang secara bawaan mengunci fokus keyboard di dalam modal saat dialog terbuka, dan otomatis mengembalikan fokus ke elemen pemanggil saat ditutup via tombol Escape (`EscapeKeyHandler`).

### 4.3. Rasio Kontras Warna (Color Contrast Ratio)
- Menggunakan palet token Tailwind dengan rasio kontras >= 4.5:1 terhadap latar belakang kartu/body.
- Teks deskripsi sekunder (`text-muted-foreground`) divalidasi memiliki rasio kontras minimal 4.8:1 pada mode gelap (`#A1A1AA` di atas `#09090B`) dan mode terang (`#71717A` di atas `#FFFFFF`).
- Badge status (Lunas = hijau, Pending = kuning/oranye, Gagal = merah) menggunakan kombinasi latar belakang pastel dengan teks kontras tinggi (misal `bg-emerald-50 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300`).
