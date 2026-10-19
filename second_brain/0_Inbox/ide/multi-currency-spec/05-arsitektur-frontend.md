# 05 — Arsitektur Frontend

**Status:** DRAFT  
**Stack:** Next.js 16 (App Router) + TanStack Query + Tailwind, bahasa UI Indonesia.

---

## 1. Prinsip

1. **Satu sumber format uang.** Semua nominal lewat helper bersama; tidak ada `Intl.NumberFormat(... currency: "IDR")` yang tersebar (89 kemunculan saat ini, dokumen 01).
2. Preferensi mata uang berasal dari server (`/me`), bukan `localStorage` — konsisten lintas perangkat.
3. Formatter lama **tetap ada** untuk zona IDR (billing/marketing/admin) supaya diff kecil dan tidak ada risiko regresi di permukaan non-keuangan.
4. Tidak ada dependensi baru; pakai `Intl.NumberFormat` yang sudah native.

---

## 2. Provider: `CurrencyProvider`

File baru: `frontend/providers/currency-provider.tsx` (pola mengikuti `theme-provider.tsx`).

```tsx
"use client";
// Membaca display_currency + katalog + rate as-of dari TanStack Query,
// lalu menyediakan objek format siap pakai ke seluruh aplikasi.
```

Nilai yang disediakan:

| Field | Asal |
|---|---|
| `code` | `me.display_currency` |
| `meta` | `GET /currencies` (symbol, locale, decimal_digits) |
| `rate` | `GET /currencies/rates?at=...` (kurs as-of) |
| `format(v, opts)` | `Intl.NumberFormat(meta.locale, {style:"currency",currency:code,...})` |
| `toBase(v)` / `fromBase(v)` | konversi untuk input & agregat |

Provider mount di root dashboard (di dalam `RealtimeProvider`, karena butuh sesi). Saat belum login/muat → fallback IDR agar tidak ada flash nilai kosong.

---

## 3. Helper: `frontend/lib/money.ts`

```ts
export type CurrencyMeta = {
  code: string; name: string; symbol: string; locale: string;
  decimalDigits: number; isBase: boolean;
};

export type CurrencyFormatCtx = { meta: CurrencyMeta; rate: number };

/** Format nilai dalam mata uang ctx, mis. 1000 + JPY -> "¥1,000". */
export function formatMoney(value: number, ctx: CurrencyFormatCtx): string;

/** Nominal asli transaksi (sudah dalam mata uangnya sendiri) -> teks. */
export function formatAmount(value: number, currencyCode: string, metas: CurrencyMeta[]): string;
```

`format.ts` yang ada:
- `formatRupiah` **dipertahankan** (zona IDR: billing, marketing, admin).
- Tambah re-export tipis atau deprecation comment yang menunjuk `money.ts` untuk permukaan keuangan pengguna.

Cache `Intl.NumberFormat` per (locale, code, digits) di `Map` module-level; membuat formatter baru setiap render mahal pada tabel besar.

---

## 4. Query Keys Baru

`frontend/lib/cache-keys.ts`:

```ts
currency: {
  catalog: ["currency", "catalog"] as const,
  rates: (at: string) => ["currency", "rates", at] as const,
},
profile: {
  me: ["profile", "me"] as const,       // sudah ada
},
```

Provider membaca `queryKeys.profile.me` (sudah dipakai) dan menambahkan katalog/rate. Realtime listener `config_changed` dengan `domain === "currency"` meng-invalidate `currency.catalog` + `currency.rates`.

---

## 5. Preferensi di Profil

`frontend/components/profile/AppearanceSettings.tsx` — bagian "Tampilan & Bahasa" saat ini hanya tema + bahasa (disabled).

Tambahan: picker mata uang.

- Sumber opsi: `GET /currencies` (`is_active=true`). IDR selalu tampil; JPY tampil setelah Super Admin mengaktifkan.
- Bila hanya 1 opsi aktif, tampilkan sebagai nilai terpilih + catatan "Mata uang lain dapat diaktifkan admin".
- Aksi: `PUT /me/preferences/currency` → invalidate `profile.me` + semua cache nominal (`transactions.all`, `dashboard.all`, `reports.all`, `budgets.all`).
- Optimistic update + rollback bila 422 (mis. mata uang dinonaktifkan di server).
- Microcopy: "Mata uang pencatatan. Langganan dan biaya AI tetap dalam Rupiah."

Analog: buat komponen terpisah `CurrencyPreference.tsx` dan render di grid `AppearanceSettings` (grid saat ini 2 kolom; tambah kolom ketiga atau baris baru). Menjaga `AppearanceSettings` tidak membengkak.

---

## 6. Permukaan yang Berubah (agregat → display currency)

Semua memakai `useCurrency()` + `formatMoney(amount_base → fromBase)`:

| Berkas | Perubahan |
|---|---|
| `app/(dashboard)/user/transactions/page.tsx` | Kartu statistik (income/expense/savings/net) pakai konversi; detail transaksi pakai nominal asli |
| `components/dashboard/StatCard.tsx` | Formatter ganti ke `formatMoney` |
| `components/transactions/TransactionRow.tsx` | Nominal baris = **nominal asli** + kode |
| `components/transactions/TransactionColumns.tsx` | idem (kolom amount, sorting tetap asli) |
| `components/charts/*` | `PieChartView`, `MultiSeriesChart`, `LineChartView`, `BarChartView`, `AreaChartView`, `CalendarHeatmap` → `formatMoney` |
| `components/reports/ReportsSummaryTab.tsx`, `TopMoversPanel.tsx`, `CategoryTable.tsx`, `BudgetPanel.tsx` | Ganti formatter; nilai dari server sudah dikonversi (dokumen 04 §11) **atau** FE konversi via `fromBase` — pilih satu, rekomendasi: server kirim nilai base + `rate_used`, FE hanya memformat |
| `app/(dashboard)/user/reports/page.tsx` | Ganti formatter baris 55 |
| `components/ai/*` (`ChatWindow`, `InlineAiAssistant`, `CrudConfirmCard`, `ReceiptScanner`) | Nominal konteks & konfirmasi ikut display currency |
| `lib/receipt.ts` `formatRupiahIDR` | Tetap IDR untuk struk? → struk asing bisa JPY. Ganti ke `formatAmount` + currency deteksi dari payload; fallback IDR |

**Aturan penting:** konversi agregat **sekali di server**. FE tidak menjumlahkan sendiri dari baris-baris berbeda mata uang. FE hanya memformat dan (bila perlu) `fromBase` untuk satu nilai agregat.

---

## 7. Permukaan yang Tetap IDR (tidak disentuh)

| Berkas | Alasan |
|---|---|
| `components/marketing/*` (`PricingCards`, `PricingTeaser`, `copy.ts`, `InteractiveDemo`) | Harga publik IDR |
| `components/billing/*`, `app/(dashboard)/billing/*` | Invoice/pembayaran IDR |
| `app/(dashboard)/super-admin/*` (`tokens`, `payments`, `kuota`) | Metrik & biaya admin IDR |
| `components/usage/*` | Riwayat pemakaian token (IDR/unit) |
| `components/shared/Money.tsx` | Komponen IDR generik; tetap ada untuk zona IDR |
| `SmartInsightCard` | Konteks biaya AI/admin |

Ini menjaga diff kecil dan memastikan klaim "billing tetap IDR" dapat diuji.

---

## 8. Input Nominal

Komponen baru `components/ui/CurrencyInput.tsx`:

- Prefix simbol dari `meta.symbol` (mis. `¥`).
- `inputMode="decimal"`, pemisah ribuan sesuai locale meta.
- Batas desimal = `meta.decimalDigits` (JPY 0 → tidak boleh ada `,`).
- Nilai yang dikirim ke API adalah **angka polos** dalam mata uang aktif + `currency_code`.
- Tidak ada konversi di client saat submit; server yang menghitung `amount_base`.

Dipakai di: form transaksi, form budget, scan struk, AI confirm.

---

## 9. Sesi & Tipe

`frontend/lib/auth-client.ts`:

```ts
export type AuthUser = {
  // ...
  display_currency?: string; // default "IDR"
};
```

`lib/session-context.tsx` meneruskan nilai ini; provider membacanya. Bila `display_currency` menunjuk mata uang yang dinonaktifkan, server mengembalikan nilai efektif (IDR) di `/me`; FE tidak perlu logika khusus.

---

## 10. Realtime

`frontend/hooks/use-realtime-stream.ts` / dispatcher yang ada sudah menangani `config_changed`. Tambah cabang `domain === "currency"`:

```
config_changed { domain: "currency" }
  → qc.invalidateQueries(currency.catalog)
  → qc.invalidateQueries(currency.rates)   // prefix
  → (opsional) qc.invalidateQueries(profile.me)
```

Pastikan key persis sama dengan `cache-keys.ts` (jebakan yang sudah dicatat pada `gatewayEndpoints`).

---

## 11. Strategi Migrasi Formatter

Agar tidak "big bang" (berisiko 89 titik):

1. **Fase 1:** tambah provider + `money.ts` tanpa mengubah tampilan (default IDR → output identik `formatRupiah`).
2. **Fase 2:** konversi permukaan keuangan pengguna satu-satu (dashboard → transactions → reports → budgets → charts → AI).
3. **Fase 3:** uji tombol mata uang IDR/JPY, bandingkan tampilan IDR sebelum/sesudah (harus identik).
4. **Fase 4:** (opsional, terpisah) audit sisa `Intl.NumberFormat` IDR untuk memastikan hanya zona IDR tersisa.

Uji regresi IDR: snapshot tampilan dengan `display_currency=IDR` harus sama persis dengan sebelum perubahan.

---

## 12. Test Frontend

| Test | Isi |
|---|---|
| `lib/money.test.ts` | IDR 0 desimal, JPY 0 desimal, USD 2 desimal, simbol, nilai null/NaN |
| `lib/format.test.ts` (ada) | tetap hijau → bukti zona IDR tidak berubah |
| Provider test | fallback IDR saat belum login |
| Realtime dispatcher | event `currency` meng-invalidate key yang benar |
| E2E/manual | ganti IDR→JPY, cek dashboard/laporan/ekspor konsisten |

---

## 13. Ringkasan Keputusan Frontend

| # | Keputusan |
|---|---|
| F-1 | `CurrencyProvider` membaca `/me` + katalog + rate |
| F-2 | `lib/money.ts` sebagai satu-satunya formatter keuangan pengguna |
| F-3 | `formatRupiah` tetap hidup untuk zona IDR |
| F-4 | Preferensi di `AppearanceSettings` via komponen `CurrencyPreference` |
| F-5 | Konversi agregat di server; FE memformat |
| F-6 | Baris transaksi menampilkan nominal asli; agregat dikonversi |
| F-7 | `CurrencyInput` menghormati `decimal_digits` |
| F-8 | Realtime `config_changed(domain=currency)` meng-invalidate katalog/rate |
| F-9 | Migrasi formatter bertahap, verifikasi snapshot IDR |
