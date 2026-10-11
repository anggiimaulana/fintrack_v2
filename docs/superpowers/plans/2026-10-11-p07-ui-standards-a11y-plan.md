# Implementation Plan — P07 Slice C: Standar UI, Aksesibilitas & Optimistic Update

| Field | Isi |
|---|---|
| **Fase** | P07 — Rollout RBAC, Realtime & Standar (**Slice C** dari 5) |
| **Versi** | 1.0 |
| **Tanggal** | 2026-10-11 |
| **Prasyarat** | Slice A (`9500e4b`) & B (`ea725e6`) selesai |
| **Spec desain** | `docs/superpowers/specs/2026-10-11-p07-ui-standards-a11y-design.md` |
| **Task terkait** | TASK-P07-011, 012, 014, 015 (013 → slice D) |
| **Status** | Siap dieksekusi |

---

## 1. Tujuan & Definition of Done

Menyelesaikan standar UI: tabel transaksi memakai `<DataTable>` dengan URL state + debounce,
audit cursor-pointer menyeluruh, a11y dengan tooling yang bisa diverifikasi, dan optimistic
update pada mutasi transaksi.

**Definition of Done:**
- [ ] `/user/transactions` memakai `<DataTable>` + URL state + debounce 300ms.
- [ ] Audit cursor-pointer selesai dengan konvensi eksplisit.
- [ ] `eslint-plugin-jsx-a11y` aktif di `pnpm lint` (0 error).
- [ ] `axe-core` di Vitest untuk komponen kunci: nol pelanggaran.
- [ ] Tiga kegagalan kontras diperbaiki & diverifikasi ulang dengan perhitungan.
- [ ] `aria-label` pada seluruh tombol ikon tanpa teks.
- [ ] Optimistic update update & delete dengan rollback 409 / 402-403 / generik.
- [ ] `pnpm lint` 0 error · `pnpm build` hijau · `pnpm test` hijau.
- [ ] Dokumentasi + tracking diperbarui; gap AC dicatat jujur.

---

## 2. Urutan Eksekusi

| Wave | Langkah | Deskripsi | Prasyarat |
|---|---|---|---|
| **W1** | C-1 | Aktifkan `eslint-plugin-jsx-a11y`, perbaiki pelanggaran | — |
| **W1** | C-2 | Perbaiki 3 kegagalan kontras (token + badge + tombol hapus) | — |
| **W1** | C-3 | `aria-label` untuk tombol ikon tanpa teks | C-1 |
| **W1** | C-4 | Audit cursor-pointer menyeluruh | — |
| **W2** | C-5 | Siapkan Vitest jsdom + `vitest-axe` | C-1 |
| **W2** | C-6 | Test axe untuk komponen kunci | C-5 |
| **W3** | C-7 | Refactor `/user/transactions` ke `<DataTable>` + URL state | C-4 |
| **W3** | C-8 | Debounce 300ms pada search | C-7 |
| **W4** | C-9 | Optimistic update update & delete + rollback 3 kelas error | C-7 |
| **W4** | C-10 | Test optimistic update | C-9 |
| **W5** | C-11 | Gerbang penuh: lint/build/test | C-1…C-10 |
| **W5** | C-12 | Koreksi dokumentasi + tracking + log | C-11 |
| **W5** | C-13 | Commit | C-12 |

**Jalur kritis:** C-5 → C-6 (tooling a11y harus hidup untuk membuktikan perbaikan) dan
C-7 → C-9 (optimistic update bergantung pada bentuk cache list yang baru).

---

## 3. Detail Langkah

### W1 — a11y dasar & kontras

#### C-1 — Aktifkan `eslint-plugin-jsx-a11y`
- **Target:** `frontend/eslint.config.mjs`
- **Langkah:** tambahkan plugin + rekomendasi ke flat config; jalankan `pnpm lint`; perbaiki
  pelanggaran nyata (bukan mematikan aturan). Aturan yang tidak relevan boleh di-*warn*, dengan
  alasan tertulis.
- **Verifikasi:** `pnpm lint` 0 error; daftar pelanggaran awal dicatat sebelum/sesudah.

#### C-2 — Perbaiki kontras
- **Target:** `frontend/app/globals.css`, `frontend/components/transactions/TransactionRow.tsx`,
  tombol hapus di halaman transaksi
- **Langkah (nilai sudah dihitung di spec §1.1):**
  1. `--muted-foreground` (light): `oklch(0.55 0 0)` → `oklch(0.53 0 0)` → 4.56 di atas `muted`.
  2. Badge "Scan": `amber-600` → `amber-700` di atas `amber-50` → 4.84.
  3. Tombol hapus: `red-600` → `red-700` di atas `red-100` → 5.30.
- **Verifikasi:** hitung ulang rasio dengan skrip yang sama; catat sebelum/sesudah.

#### C-3 — `aria-label` tombol ikon
- **Target:** seluruh `.tsx` dengan tombol ikon tanpa teks (21 kandidat)
- **Langkah:** tambahkan accessible name; verifikasi tidak ada duplikasi label menyesatkan.
- **Verifikasi:** `grep` memastikan tiap tombol ikon punya `aria-label`/teks; lint jsx-a11y.

#### C-4 — Audit cursor-pointer
- **Target:** seluruh `.tsx` di `components/` dan `app/`
- **Langkah:** terapkan konvensi (interaktif `cursor-pointer`; disabled `cursor-not-allowed`;
  statis default). Perbaiki hanya penyimpangan.
- **Verifikasi:** hitung cakupan sebelum/sesudah; catat di `teknis/cursor-pointer-audit.md`.

### W2 — Verifikasi a11y

#### C-5 — Vitest jsdom + axe
- **Target:** `frontend/vitest.config.ts`, `frontend/test/setup.ts`
- **Langkah:** environment jsdom untuk tes komponen; daftarkan `vitest-axe` matcher di setup.
  Pertahankan environment `node` untuk tes murni (dispatcher/backoff) agar tidak melambat.
- **Verifikasi:** satu tes contoh render + axe lolos.

#### C-6 — Test axe komponen kunci
- **Target:** `frontend/components/**/*.a11y.test.tsx`
- **Komponen:** `ConfirmDialog`, `DataTable` (dengan data & kosong), `DataTableToolbar`,
  `TransactionRow`, form transaksi.
- **Verifikasi:** nol pelanggaran axe; kalau ada temuan, perbaiki (bukan di-skip).

### W3 — Refactor tabel transaksi

#### C-7 — `<DataTable>` + URL state
- **Target:** `frontend/app/(dashboard)/user/transactions/page.tsx`,
  `frontend/components/transactions/TransactionColumns.tsx` (baru)
- **Langkah:**
  1. Definisikan kolom dengan `DataTableColumnHeader` (sorting server-side).
  2. Ganti state lokal (`page`, `search`, `filterType`) dengan `useDataTableUrlState`.
  3. **Pertahankan** param tanggal (`preset`/`start`/`end`) di namespace-nya sendiri.
  4. Pasang `DataTablePagination` + `DataTableToolbar`.
- **Verifikasi:** URL memuat `?page=&sort=&order=&q=`; refresh mempertahankan state; tes.

#### C-8 — Debounce search
- **Target:** halaman transaksi
- **Langkah:** debounce 300ms sebelum menulis ke URL (bukan per ketikan).
- **Verifikasi:** tes — N ketikan cepat menghasilkan 1 pembaruan.

### W4 — Optimistic update

#### C-9 — Optimistic update update & delete
- **Target:** `frontend/hooks/useTransactions.ts`
- **Langkah:**
  1. `onMutate`: snapshot cache list (bentuk `{data,total,page,perPage}`), patch optimistik,
     kembalikan konteks.
  2. `onError`: rollback dari snapshot + toast sesuai kelas error (409 / 402-403 via
     `notifyEntitlementError` / generik).
  3. `onSettled`: invalidate `transactions.all` + `dashboard.all` + `spk.all`.
  4. **Create tetap non-optimistik** (keputusan opsi C, spec §3.4).
- **Verifikasi:** tes optimistik + rollback tiga kelas error.

#### C-10 — Test optimistic update
- **Target:** `frontend/hooks/useTransactions.test.tsx`
- **Kasus:** cache berubah sebelum resolve; rollback tepat pada 409, 402/403, dan error generik;
  create tidak mengubah cache sebelum resolve.

### W5 — Finalisasi

#### C-11 — Gerbang penuh
`pnpm lint && pnpm build && pnpm test`.

#### C-12 — Dokumentasi
`teknis/cursor-pointer-audit.md` (temuan + konvensi), `tracking/tracking.md` (status + **gap AC**),
`log_aktivitas.md`.

#### C-13 — Commit

---

## 4. Matriks Bukti

| Langkah | Artefak | Bukti |
|---|---|---|
| C-1 | `eslint.config.mjs` | `pnpm lint` 0 error |
| C-2 | `globals.css`, `TransactionRow.tsx` | rasio kontras sebelum/sesudah |
| C-3 | berkas `.tsx` terkait | grep + lint |
| C-4 | berkas `.tsx` terkait | laporan cakupan di `cursor-pointer-audit.md` |
| C-6 | `*.a11y.test.tsx` | `pnpm test` (nol pelanggaran axe) |
| C-7/C-8 | halaman transaksi + kolom | URL state + tes debounce |
| C-9/C-10 | `useTransactions.ts` + tes | tes rollback tiga kelas error |

---

## 5. Perintah Verifikasi

```bash
cd frontend && pnpm lint && pnpm build && pnpm test
```

---

## 6. Rollback

Revert commit slice C. **Tanpa migrasi.** Perubahan kontras adalah perubahan token/warna —
mengembalikannya mengembalikan kegagalan a11y, jadi jangan di-rollback sendirian tanpa alasan.

---

## 7. Gap yang dicatat (bukan diklaim)

- **AC-015 "Lighthouse Accessibility ≥ 95" TIDAK terpenuhi.** Tidak ada Lighthouse di repo;
  butuh server + browser. Digantikan oleh: jsx-a11y + axe + perhitungan kontras manual.
- **AC-010 "0ms saat menambah" tidak sepenuhnya terpenuhi.** Create sengaja non-optimistik
  (list dipaginasi + di-sort server-side).
- **TASK-P07-012 hanya sebagian:** `/budgets` & `/categories` (user) tidak ada.

---

## 8. Handoff

Slice berikutnya: **D** — health gates, graceful shutdown 30s, expand/contract, dan BYOK.
**Untuk D, keputusan sudah diambil:** naikkan `feature_ai_gateway_centralized` ke 100%
(atau perbaiki pemanggilan agar memakai userID) dan verifikasi seluruh user terlayani gateway
**sebelum** menghapus BYOK — karena saat ini BYOK adalah satu-satunya jalur AI bagi ~75% user
(flag rollout 25%, dan `gatewayEnabled` dipanggil tanpa userID sehingga selalu false untuk
rollout parsial).
