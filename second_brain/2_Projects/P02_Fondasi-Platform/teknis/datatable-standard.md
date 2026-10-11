# Dokumen Teknis — Standar Komponen DataTable

## 1. Ikhtisar Arsitektur
Komponen `<DataTable>` adalah komponen presentasi data tabular generik berbasis **TanStack Table v8** (React Table) yang digunakan secara seragam di seluruh area aplikasi (User Dashboard, Super Admin, Log Audit, Billing).

Komponen ini dirancang headless, accessible (WAI-ARIA compliant), dan terikat secara server-side dengan pagination, pengurutan, pencarian ter-debounce, dan penyaringan faset.

---

## 2. Kontrak API Server-Side (Envelope Response)
Seluruh endpoint backend yang menyajikan data tabular wajib mengembalikan format JSON standar:

```json
{
  "success": true,
  "data": [
    {
      "id": "usr_9981",
      "name": "Budi Santoso",
      "email": "budi@example.com",
      "role": "user",
      "created_at": "2026-10-11T09:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total_items": 142,
    "total_pages": 8,
    "sort_by": "created_at",
    "sort_order": "desc"
  }
}
```

---

## 3. Komposisi Komponen UI Next.js

Struktur folder komponen di `frontend/src/components/data-table/`:
```
components/data-table/
├── data-table.tsx                # Komponen induk tabel
├── data-table-toolbar.tsx        # Search bar & filter controls
├── data-table-pagination.tsx     # Kontrol halaman & limit dropdown
├── data-table-column-header.tsx  # Header dengan sorting toggle
└── data-table-skeleton.tsx       # Placeholder saat proses fetch
```

### 3.1. Implementasi Komponen Induk `<DataTable>`
```tsx
"use client";

import * as React from "react";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { DataTableToolbar } from "./data-table-toolbar";
import { DataTablePagination } from "./data-table-pagination";
import { DataTableSkeleton } from "./data-table-skeleton";

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  isLoading?: boolean;
  totalItems: number;
  totalPages: number;
  currentPage: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  onSearchChange: (query: string) => void;
  searchValue?: string;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
  onSortChange: (column: string, order: "asc" | "desc") => void;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  isLoading = false,
  totalItems,
  totalPages,
  currentPage,
  pageSize,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  searchValue = "",
  sortBy,
  sortOrder,
  onSortChange,
}: DataTableProps<TData, TValue>) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
    pageCount: totalPages,
  });

  return (
    <div className="space-y-4">
      <DataTableToolbar
        searchValue={searchValue}
        onSearchChange={onSearchChange}
      />
      <div className="rounded-md border border-neutral-200 dark:border-neutral-800">
        <table className="w-full text-sm text-left">
          <thead className="bg-neutral-50 dark:bg-neutral-900 border-b">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th key={header.id} className="h-10 px-4 font-medium">
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {isLoading ? (
              <DataTableSkeleton columnCount={columns.length} rowCount={pageSize} />
            ) : table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <tr
                  key={row.id}
                  className="border-b transition-colors hover:bg-neutral-50/50"
                >
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="p-4 align-middle">
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </td>
                  ))}
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={columns.length}
                  className="h-24 text-center text-neutral-500"
                >
                  Tidak ada data ditemukan.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <DataTablePagination
        totalItems={totalItems}
        currentPage={currentPage}
        totalPages={totalPages}
        pageSize={pageSize}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
      />
    </div>
  );
}
```

---

## 4. Komponen Pagination Terstandarisasi
Ukuran halaman dikunci secara ketat pada opsi: `[10, 20, 50, 100]` dengan default `20`.
Tampilan memuat:
- Total data keseluruhan.
- Posisi baris aktif (misal: "Menampilkan 1-20 dari 142 data").
- Tombol navigasi: Pertama, Sebelumnya, Berikutnya, Terakhir.
- Dropdown pengubah ukuran halaman.

---

## 5. Aksesibilitas (A11y) & Performa
- Seluruh tag interaktif pada header tabel memiliki `aria-sort="ascending"`, `aria-sort="descending"`, atau `aria-sort="none"`.
- Setiap tombol pagination memiliki `aria-label` eksplisit (misal: "Ke halaman berikutnya").
- Memori browser dijaga dengan menghindari DOM virtualisasi berlebihan untuk subset data 20 item server-side.
