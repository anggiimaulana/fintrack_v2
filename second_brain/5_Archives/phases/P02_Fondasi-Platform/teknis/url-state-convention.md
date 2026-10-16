# Dokumen Teknis — Konvensi State URL (URL-First State)

## 1. Prinsip Utama: URL sebagai Single Source of Truth
Pada FinTrack v2, setiap interaksi pada tabel data (halaman, jumlah per halaman, kolom urutan, arah urutan, kata kunci pencarian, filter status) disimpan secara eksplisit pada query parameter URL browser.

### Manfaat Arsitektur:
1. **Shareable & Bookmarkable**: Pengguna dan tim support dapat menyalin URL dan melihat data persis sama.
2. **Histori Navigasi Native**: Tombol *Back* dan *Forward* browser berfungsi alami tanpa merusak filter.
3. **Persistensi Reload**: Melakukan refresh halaman (F5) tidak me-reset state tabel ke default.

---

## 2. Standar Penamaan Parameter Query String

| Parameter | Tipe Data | Nilai Default | Contoh Nilai | Keterangan |
|---|---|---|---|---|
| `page` | Integer (>= 1) | `1` | `page=2` | Nomor halaman aktif (1-indexed). |
| `per_page` | Integer | `20` | `per_page=50` | Batas item per halaman (10, 20, 50, 100). |
| `sort` | String (snake_case) | `created_at` | `sort=amount` | Kolom target pengurutan. |
| `order` | Enum (`asc` \| `desc`) | `desc` | `order=asc` | Arah pengurutan data. |
| `q` | String | `""` (kosong) | `q=budi%20santoso` | Kata kunci pencarian global (ter-debounce). |
| `filter_<key>` | String / List | `""` | `filter_status=active` | Filter dinamis per kolom tertentu. |

Contoh URL Lengkap:
```
https://app.fintrack.id/admin/users?page=2&per_page=20&sort=email&order=asc&q=developer&filter_role=super_admin
```

---

## 3. Implementasi Hook `useDataTableUrlState`

Hook kustom Next.js untuk integrasi mulus dengan `next/navigation` (`useSearchParams`, `useRouter`, `usePathname`):

```tsx
"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useMemo, useTransition } from "react";

export interface DataTableUrlState {
  page: number;
  perPage: number;
  sortBy: string;
  sortOrder: "asc" | "desc";
  search: string;
  filters: Record<string, string>;
}

export function useDataTableUrlState(defaultSortBy = "created_at") {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [isPending, startTransition] = useTransition();

  // Ekstraksi State dari URL
  const state: DataTableUrlState = useMemo(() => {
    const page = Math.max(1, parseInt(searchParams.get("page") || "1", 10));
    const rawPerPage = parseInt(searchParams.get("per_page") || "20", 10);
    const perPage = [10, 20, 50, 100].includes(rawPerPage) ? rawPerPage : 20;
    const sortBy = searchParams.get("sort") || defaultSortBy;
    const sortOrder = searchParams.get("order") === "asc" ? "asc" : "desc";
    const search = searchParams.get("q") || "";

    const filters: Record<string, string> = {};
    searchParams.forEach((val, key) => {
      if (key.startsWith("filter_")) {
        filters[key.replace("filter_", "")] = val;
      }
    });

    return { page, perPage, sortBy, sortOrder, search, filters };
  }, [searchParams, defaultSortBy]);

  // Update URL Query Params secara atomic
  const updateUrl = useCallback(
    (updates: Partial<DataTableUrlState>) => {
      const params = new URLSearchParams(searchParams.toString());

      if (updates.page !== undefined) {
        if (updates.page <= 1) params.delete("page");
        else params.set("page", updates.page.toString());
      }

      if (updates.perPage !== undefined) {
        if (updates.perPage === 20) params.delete("per_page");
        else params.set("per_page", updates.perPage.toString());
      }

      if (updates.sortBy !== undefined) {
        if (updates.sortBy === defaultSortBy) params.delete("sort");
        else params.set("sort", updates.sortBy);
      }

      if (updates.sortOrder !== undefined) {
        if (updates.sortOrder === "desc") params.delete("order");
        else params.set("order", updates.sortOrder);
      }

      if (updates.search !== undefined) {
        if (!updates.search) params.delete("q");
        else params.set("q", updates.search);
        // Reset page ke 1 setiap pencarian berubah
        params.delete("page");
      }

      if (updates.filters !== undefined) {
        // Hapus filter lama
        Array.from(params.keys()).forEach((k) => {
          if (k.startsWith("filter_")) params.delete(k);
        });
        // Tambahkan filter baru
        Object.entries(updates.filters).forEach(([k, v]) => {
          if (v) params.set(`filter_${k}`, v);
        });
        params.delete("page");
      }

      startTransition(() => {
        router.replace(`${pathname}?${params.toString()}`, { scroll: false });
      });
    },
    [router, pathname, searchParams, defaultSortBy]
  );

  return {
    state,
    updateUrl,
    isPending,
  };
}
```

---

## 4. Debounce Handling pada Pencarian
Pencarian teks menggunakan jeda waktu 300ms untuk mencegah eksekusi mutasi URL berulang-ulang pada tiap ketikan keyboard:

```tsx
import { useEffect, useState } from "react";

export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}
```

---

## 5. Sinkronisasi dengan TanStack Query v5
Hook `useDataTableUrlState` langsung mengalirkan parameternya ke query key TanStack Query:

```tsx
const { state, updateUrl } = useDataTableUrlState();

const { data, isLoading } = useQuery({
  queryKey: ["admin", "users", state],
  queryFn: () => fetchUsers(state),
  staleTime: 30 * 1000, // 30 detik
});
```
Dengan pola ini, pergantian parameter URL otomatis memicu fetch TanStack Query dengan caching terkelola secara otomatis.
