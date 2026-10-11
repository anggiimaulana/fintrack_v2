# P07 Slice C — Standar UI, Aksesibilitas & Optimistic Update: Design

**Date:** 2026-10-11
**Status:** Approved (pending written-spec review)
**Phase:** P07 (Rollout RBAC, Realtime & Standar) — slice 3 of 5
**Related tasks:** TASK-P07-011 … TASK-P07-015 (minus 013 → slice D)

---

## 1. Problem

Standar UI belum menyeluruh dan belum pernah diaudit terhadap kenyataan.

| Area | Kondisi terverifikasi |
|---|---|
| `<DataTable>` | ada (`components/data-table/`, 5 berkas) tetapi hanya dipakai **halaman super-admin** (audit, payments, subscriptions, tokens) |
| `/user/transactions` | memakai `transactions.map(...)` custom; state list di `useState` lokal |
| Debounce search | **tidak ada** — `onChange` memanggil `setSearch` setiap ketikan (AC-011 meminta 300ms) |
| `cursor-pointer` | 36 dari 127 `.tsx`; belum pernah diaudit |
| Tooling a11y | **tidak ada** — tanpa axe, tanpa `eslint-plugin-jsx-a11y`, tanpa Lighthouse |
| `aria-label` | 19 berkas; ada 21 tombol `size="icon"` yang perlu diverifikasi accessible name |
| Dialog | berbasis Radix → focus trap + Escape **sudah** ditangani |
| Optimistic update | **belum ada** di `useTransactions.ts` |

### 1.1 Audit kontras (dihitung manual dari token `app/globals.css`)

Mode **gelap**: seluruh pasangan lolos (terendah 5.53).

Mode **terang** — tiga temuan nyata:

| Pasangan | Rasio | Ambang | Verdict |
|---|---|---|---|
| `muted-foreground` di atas `muted` | **4.19** | 4.5 | **GAGAL** |
| `destructive` di atas `background` | **4.50** | 4.5 | Lolos, tanpa margin |
| `amber-600` di atas `amber-50` (badge "Scan" di `TransactionRow`) | **3.07** | 4.5 | **GAGAL** |
| `red-600` di atas `red-100` (tombol hapus `hover:bg-red-100`) | **3.95** | 4.5 | **GAGAL** |

Nilai perbaikan yang sudah diverifikasi:

| Perubahan | Hasil |
|---|---|
| `--muted-foreground` terang: `oklch(0.55 0 0)` → `oklch(0.53 0 0)` | 4.56 di atas `muted`, 4.98 di atas `background` |
| badge Scan: `amber-600` → `amber-700` di atas `amber-50` | 4.84 |
| tombol hapus: `red-600` → `red-700` di atas `red-100` | 5.30 |

### 1.2 Task yang tidak dapat dikerjakan

- **TASK-P07-012** menyebut `/budgets` dan `/categories` (user). **Keduanya tidak ada**: kategori dikelola admin (`/admin/categories/*`), fitur budget tidak ada. Yang tersisa hanya `/user/transactions`.
- **TASK-P07-015** menyebut "Lighthouse Accessibility ≥ 95". Tidak ada Lighthouse di repo; menjalankannya butuh server + browser. **AC ini tidak akan terpenuhi** dan ditandai sebagai gap, bukan diklaim.

---

## 2. Goals / Non-goals

**Goals**

1. `/user/transactions` memakai `<DataTable>` dengan state list di URL (bookmarkable) + debounce 300ms.
2. Audit `cursor-pointer` menyeluruh dengan konvensi eksplisit.
3. a11y: tooling statis (`eslint-plugin-jsx-a11y`) + axe untuk komponen kunci + perbaikan kontras yang terukur.
4. Optimistic update pada mutasi transaksi dengan rollback untuk tiga kelas error.

**Non-goals**

- Tidak mengubah `<DataTable>` (sudah memenuhi AC).
- Tidak membuat halaman `/budgets` atau `/categories` untuk user.
- Tidak menambah Lighthouse/CI browser.
- Tidak menghapus BYOK (slice D).

---

## 3. Design

### 3.1 Refactor `/user/transactions`

- Ganti list custom dengan `<DataTable>` + `<DataTablePagination>` + `<DataTableToolbar>`.
- Pindahkan state list (`page`, `search`, `filterType`) ke `useDataTableUrlState` — URL menjadi sumber kebenaran, sesuai AC-011 ("URL otomatis merefleksikan filter dan halaman").
  - Filter tanggal yang sudah ada di URL (`preset`/`start`/`end`) **dipertahankan**; tidak dipindah ke `useDataTableUrlState` agar tidak bentrok namespace.
- **Debounce 300ms** pada search (saat ini per-ketikan).
- Kolom: tanggal, deskripsi, kategori, sumber, jumlah — dengan `DataTableColumnHeader` untuk sorting server-side.

**Konsekuensi yang perlu diketahui:** refresh tidak lagi mereset filter (itulah tujuan AC). Perilaku berubah, disengaja.

### 3.2 Cursor pointer

Konvensi yang diterapkan dan diaudit:

| Elemen | Kelas |
|---|---|
| Interaktif (`button`, `a`, `[role=button]`, baris tabel ber-`onRowClick`, checkbox, switch, tab) | `cursor-pointer` |
| Disabled | `cursor-not-allowed` (plus `disabled:opacity-50`) |
| Statis (teks, kartu non-klik, header) | `cursor-default` (default; hanya ditambah bila ada kelas kursor lain) |

Audit menyeluruh 127 berkas `.tsx`, perbaikan hanya di tempat yang menyimpang.

### 3.3 Aksesibilitas

| Verifikasi | Cara | Otomatis? |
|---|---|---|
| Aturan JSX a11y | `eslint-plugin-jsx-a11y` masuk `pnpm lint` | ya |
| Struktur rendered | `axe-core` via Vitest pada komponen kunci (dialog, DataTable, toolbar, form) | ya |
| Kontras | perhitungan manual dari token (sudah dijalankan, §1.1) | **tidak** |
| Lighthouse ≥ 95 | tidak dijalankan | **tidak** |

Perbaikan:
- `aria-label` untuk seluruh tombol ikon tanpa teks (21 kandidat).
- `--muted-foreground` terang → `oklch(0.53 0 0)`.
- Badge Scan `amber-600` → `amber-700`; tombol hapus `red-600` → `red-700`.
- Focus trap + Escape sudah dari Radix — diverifikasi lewat test axe, bukan diasumsikan.

### 3.4 Optimistic update

Pola `onMutate` / `onError` / `onSettled` untuk update & delete transaksi.

**Keputusan: optimistic insert pada *create* TIDAK dilakukan** (opsi C). Alasan: list dipaginasi server-side dengan sorting server-side; baris baru tidak punya posisi yang diketahui, sehingga prepend optimistik akan menampilkan baris di tempat yang salah lalu melompat. Menampilkan posisi salah lebih buruk daripada spinner singkat.

**Konsekuensi terhadap AC-010** ("latensi persepsi 0ms saat menambah atau menyunting"): huruf *menambah* **tidak sepenuhnya terpenuhi**. Dicatat eksplisit; update & delete (yang paling sering dipakai) tetap instan.

**Rollback tiga kelas error:**
| Error | Perilaku |
|---|---|
| 409 (optimistic lock) | rollback + toast "data sudah diubah di tempat lain" |
| 402/403 (entitlement, slice A) | rollback + `notifyEntitlementError` (helper sudah ada) |
| lainnya | rollback + toast generik |

Bentuk cache list adalah `{ data, total, page, perPage }` (bukan array telanjang) — snapshot/rollback harus menangani bentuk ini.

---

## 4. Testing

### 4.1 Frontend (Vitest)
- `axe-core` pada render komponen kunci: nol pelanggaran.
- Debounce search: satu request per jeda, bukan per ketikan.
- Optimistic update: cache berubah sebelum resolve; rollback tepat untuk 409, 402/403, dan error generik.
- `table-params`/URL state: putar-balik state ke URL (sudah ada tesnya, diperluas).

### 4.2 Gerbang
`pnpm lint` (kini termasuk jsx-a11y) 0 error · `pnpm build` hijau · `pnpm test` hijau.

### 4.3 Yang TIDAK diverifikasi otomatis
Kontras (manual, §1.1) dan Lighthouse. Dicatat apa adanya di tracking.

---

## 5. Dokumentasi yang dikoreksi
- `teknis/cursor-pointer-audit.md`: temuan nyata + konvensi.
- `tracking/tracking.md`: status 011/012/014/015 + gap AC yang tidak terpenuhi.
- `log_aktivitas.md`.

---

## 6. Risks

| Risiko | Mitigasi |
|---|---|
| Refactor URL-state mengubah perilaku refresh | Disengaja (tujuan AC); dicatat di log |
| Optimistic rollback salah bentuk cache | Tes eksplisit untuk bentuk `{data,total,page,perPage}` + tiga kelas error |
| Regresi visual saat memperbaiki kontras | Perubahan hanya pada token/warna teks; diverifikasi build + tes render |
| axe tidak menangkap masalah kontras token | Kontras dihitung manual, bukan diserahkan ke axe |
| AC-010 & AC-015 tidak terpenuhi penuh | Dicatat eksplisit di tracking, tidak diklaim |
