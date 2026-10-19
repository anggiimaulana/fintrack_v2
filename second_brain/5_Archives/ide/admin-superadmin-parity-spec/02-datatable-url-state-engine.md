# Spesifikasi Teknis: Data Table & URL State Engine

Dokumen ini mendefinisikan arsitektur tabel data terstandarisasi, mekanisme sinkronisasi state URL dua arah, pagination bernomor server-side, pengurutan, pencarian ter-debounce, dan tata letak responsif untuk portal `admin` dan `super_admin`.

---

## 1. Arsitektur Komponen `DataTableLayout`

Komponen `DataTableLayout` merupakan komposisi standar yang membungkus seluruh tabel data di FinTrack v2:

```
+---------------------------------------------------------------------------------------------------------+
| [Kartu Statistik KPI 1]   | [Kartu Statistik KPI 2]   | [Kartu Statistik KPI 3]   | [Kartu Statistik KPI 4] |
+---------------------------------------------------------------------------------------------------------+
| [Search Input (Debounced 300ms)]       [Filter Dropdown 1]   [Filter Dropdown 2]        [Tombol Reset]  |
+---------------------------------------------------------------------------------------------------------+
| [No] | [Kolom A (Sortable)] | [Kolom B (Sortable)] | [Kolom C]             | [Status]        | [Aksi]   |
|------+----------------------+----------------------+-----------------------+-----------------+----------|
| 1    | Data 1               | Data 1B              | Data 1C               | [Badge Aktif]   | [Edit]   |
| 2    | Data 2               | Data 2B              | Data 2C               | [Badge Pending] | [Detail] |
| ...  | ...                  | ...                  | ...                   | ...             | ...      |
+---------------------------------------------------------------------------------------------------------+
| Menampilkan 1 s/d 20 dari 142 data       | [1] [2] [3] ... [8] |       Tampilkan: [20 / hal v]          |
+---------------------------------------------------------------------------------------------------------+
```

### 1.1. Props & Kontrak Antarmuka
```typescript
export interface DataTableLayoutProps<TData> {
  // Metrik & Statistik
  stats?: React.ReactNode;
  
  // Toolbar Filter
  searchPlaceholder?: string;
  filters?: React.ReactNode;
  
  // Data & Kolom Tabel
  columns: ColumnDef<TData, unknown>[];
  data: TData[];
  isLoading: boolean;
  error?: string | null;
  emptyMessage?: string;
  
  // URL State Binding
  tableState: UseDataTableUrlState;
  
  // Pagination Meta dari Server
  total: number;
  totalPages: number;
}
```

---

## 2. Sinkronisasi State URL Dua Arah (`useDataTableUrlState`)

### 2.1. Parameter URL Standar

Semua perubahan status pada tabel langsung tercermin pada URL query string tanpa reload halaman (menggunakan Next.js `useRouter` dengan opsi `{ scroll: false }` dan `window.history.replaceState`):

| Parameter URL | Tipe | Nilai Default | Deskripsi |
|---|---|---|---|
| `page` | integer | `1` | Nomor halaman aktif saat ini (1-based). |
| `per_page` | integer | `20` | Jumlah baris data per halaman (10, 20, 50, 100). |
| `sort` | string | `created_at` | Nama kolom yang sedang diurutkan di server. |
| `order` | string | `desc` | Arah pengurutan: `asc` atau `desc`. |
| `q` | string | `""` | Teks pencarian global (di-debounce 300ms). |
| `f_[key]` | string | `""` | Filter facet/kolom spesifik (contoh: `f_status=active`, `f_role=user`). |
| `tab` | string | - | Parameter penanda sub-modul atau tab aktif (jika ada). |

### 2.2. Implementasi Debounce pada Input Pencarian & Filter
Untuk mencegah flooding query ke server database:
1. State lokal input (`immediate state`) menangkap setiap ketikan karakter pengguna seketika (0ms lag).
2. Hook `useDebounce` menahan pembaruan ke URL search params selama **300ms**.
3. Setelah 300ms tanpa ketikan baru, URL query string diperbarui dan memicu request API TanStack Query dengan parameter baru.
4. Setiap pergantian kata kunci pencarian atau filter otomatis mereset `page` kembali ke `1`.

```typescript
// Cuplikan Logika Debounce di Hook useDataTableUrlState
const [searchDraft, setSearchDraft] = useState(initialSearch);
const debouncedSearch = useDebounce(searchDraft, 300);

useEffect(() => {
  if (debouncedSearch !== currentUrlSearch) {
    updateUrl({ q: debouncedSearch || undefined, page: 1 });
  }
}, [debouncedSearch]);
```

---

## 3. Pagination Bernomor Server-Side (`DataTablePagination`)

### 3.1. Ukuran Halaman Standar
- Standar default: **20 data per halaman**.
- Pilihan selector per halaman: `10`, `20`, `50`, `100`.
- Memilih ukuran halaman baru otomatis mereset posisi navigasi ke `page=1`.

### 3.2. Penomoran Halaman (Windowing Pagination)
Maksimum 5 tombol angka halaman yang ditampilkan di antara tombol panah sebelumnya (`ChevronLeft`) dan berikutnya (`ChevronRight`). Posisi angka menggunakan algoritma windowing dengan pemisah elipsis (`...`):
- Kasus A: Total halaman $\le 5$ -> Tampilkan seluruhnya `[1] [2] [3] [4] [5]`.
- Kasus B: Posisi di awal (misal page 2 dari 20) -> `[1] [2] [3] [4] ... [20]`.
- Kasus C: Posisi di tengah (misal page 10 dari 20) -> `[1] ... [9] [10] [11] ... [20]`.
- Kasus D: Posisi di akhir (misal page 19 dari 20) -> `[1] ... [17] [18] [19] [20]`.

### 3.3. Penomoran Baris Absolut
Setiap baris data memiliki nomor urut berkelanjutan lintas halaman:
$$\text{Nomor Baris} = (\text{page} - 1) \times \text{per\_page} + \text{index} + 1$$
- Pada halaman 1 dengan per-page 20: Baris bernomor `1` s/d `20`.
- Pada halaman 2 dengan per-page 20: Baris bernomor `21` s/d `40`.
- Kolom penomoran berdimensi tetap `w-14 text-center font-mono text-muted-foreground`.

---

## 4. Efisiensi Kueri & Kontrak API Backend

### 4.1. Parameter Request HTTP
Endpoint backend tidak pernah mengirim seluruh tabel ke klien. Kueri HTTP wajib menyertakan filter pagination:
```http
GET /api/v1/admin/users?page=1&per_page=20&sort=created_at&order=desc&q=john&status=active HTTP/1.1
Authorization: Bearer <token>
```

### 4.2. Standar Response Payload Backend
```json
{
  "code": 200,
  "message": "Data berhasil dimuat",
  "data": [
    {
      "id": "e0e29b12-984e-4f6c-8519-c70e0fbb1a90",
      "full_name": "John Doe",
      "email": "john@example.com",
      "role": "user",
      "is_active": true,
      "created_at": "2026-10-15T08:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 142,
    "total_pages": 8
  }
}
```

### 4.3. Implementasi Kueri SQL Terindeks (PostgreSQL)
Query backend wajib menerapkan limit offset dan count cepat:
```sql
-- Hitung total data sesuai filter
SELECT COUNT(id) FROM users 
WHERE deleted_at IS NULL 
  AND ($1 = '' OR full_name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
  AND ($2 = '' OR is_active = ($2 = 'true'));

-- Ambil subset data spesifik
SELECT id, full_name, email, role, is_active, created_at 
FROM users
WHERE deleted_at IS NULL 
  AND ($1 = '' OR full_name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
  AND ($2 = '' OR is_active = ($2 = 'true'))
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;
```

---

## 5. Tata Letak Responsif & Pencegahan Overflow

1. **Kontainer Tabel**:
   - Dibungkus elemen `<div className="w-full overflow-x-auto rounded-lg border border-border bg-card">`.
   - `min-w-full` memastikan tabel dapat digeser horizontal secara mulus pada layar sempit tanpa merusak layout shell.
2. **Penanganan Teks Panjang**:
   - Kolom email, nama transaksi, deskripsi, atau payload UUID menerapkan utilitas Tailwind `truncate max-w-[200px] inline-block`.
   - Hover pada sel yang terpotong menampilkan `<Tooltip>` berisi teks lengkap tanpa terpotong.
3. **Skeleton Loading**:
   - Saat `isLoading === true`, komponen merender `DataTableSkeleton` sebanyak baris `per_page` yang diminta dengan lebar kolom yang konsisten, mencegah pergeseran layout mendadak (*Cumulative Layout Shift / CLS = 0*).
