# FinTrack v2 — Analisis & Rancangan Multi-Kurs (IDR ↔ JPY)

**Status:** IMPLEMENTED (2026-10-19)  
**Tanggal:** 2026-10-19  
**Pemilik:** Product / Engineering  
**Cakupan:** Preferensi mata uang pencatatan pengguna, katalog mata uang dinamis, kurs dinamis dari Super Admin, dan adaptasi seluruh permukaan keuangan pengguna.

---

## 1. Ringkasan Eksekutif

FinTrack v2 saat ini adalah sistem keuangan berbasis **Rupiah (IDR)**. Nominal transaksi disimpan di satu kolom `transactions.amount`, seluruh agregasi (dashboard, statistik, laporan, ekspor, AI, embedding) menjumlahkan kolom tersebut, dan frontend memformat hampir semua angka dengan `Intl.NumberFormat` bermata uang `IDR`.

Permintaan baru: pengguna dapat memilih **Rupiah atau Yen** sebagai mata uang pencatatan di halaman profil, dengan daftar mata uang dan kurs yang **dikelola Super Admin secara dinamis** — bukan hardcode. Ke depan, mata uang lain (USD, MYR, dan sebagainya) harus dapat ditambahkan tanpa deploy ulang. Penggunaan AI, token, kuota, langganan, invoice, pembayaran, dan biaya provider **tetap Rupiah**.

Kesimpulan analisis:

1. **Mengganti formatter saja tidak cukup.** Jika hanya tampilan yang dikonversi, nilai transaksi historis akan berubah setiap kali kurs berubah, dan laporan multi-mata-uang menjadi tidak akuntabel.
2. **Diperlukan penyimpanan nilai ganda:** nominal asli dalam mata uang transaksi + nilai dasar (base) dalam IDR + snapshot kurs saat transaksi. IDR tetap menjadi **base accounting currency** karena seluruh billing dan AI sudah IDR.
3. **Kurs harus append-only dan memiliki tanggal efektif.** Laporan historis memakai kurs pada akhir periode, bukan kurs terbaru. Perubahan kurs tidak menimpa data lama.
4. **Preferensi mata uang disimpan di database pengguna**, bukan hanya `localStorage`, agar konsisten lintas perangkat, ikut ekspor data, dan dapat diubah tanpa redeploy.
5. **Billing/AI/quota adalah zona IDR yang dilindungi.** Tidak ada konversi mata uang pada invoice, pembayaran, harga paket, biaya token, kuota, atau metrik admin.
6. **Semua daftar mata uang, metadata format, dan kurs berasal dari database.** Kode hanya menyediakan kemampuan konversi; kebijakan ada di data. Pengecualian yang disengaja: IDR tetap base currency pada rilis ini.

---

## 2. Rekomendasi Arsitektur Inti

### 2.1 Model nilai

Setiap transaksi keuangan menyimpan:

| Field | Arti | Contoh |
|---|---|---|
| `amount` | Nominal asli yang diketik pengguna | `1000` |
| `currency_code` | Mata uang nominal asli | `JPY` |
| `exchange_rate` | 1 unit `currency_code` = berapa IDR, saat transaksi | `110.0000000000` |
| `amount_base` | Nilai dasar IDR = `amount × exchange_rate` | `110000.00` |
| `rate_at` | Waktu kurs yang dipakai | `2026-10-19T10:00:00Z` |

- **IDR tetap base currency.** `amount_base` selalu IDR.
- **Transaksi lama** di-backfill: `currency_code='IDR'`, `exchange_rate=1`, `amount_base=amount`.
- **Agregasi** (dashboard, laporan, statistik, budget aktual, AI context) menjumlahkan `amount_base`, bukan `amount`.
- **Tampilan transaksi individual** memakai nominal asli + kode mata uang, sehingga catatan historis tidak berubah saat kurs bergerak.
- **Tampilan agregat** dikonversi dari base IDR ke mata uang preferensi memakai kurs pada tanggal acuan (hari ini untuk dashboard; akhir periode untuk laporan).

### 2.2 Katalog dinamis

Tabel `currencies` menyimpan metadata yang selama ini hardcode:

- kode ISO 4217 (`IDR`, `JPY`, `USD`, …)
- nama tampilan
- simbol (`Rp`, `¥`, `$`)
- locale format (`id-ID`, `ja-JP`, `en-US`)
- jumlah desimal (IDR 0, JPY 0, USD 2)
- aktif/nonaktif
- urutan tampilan
- penanda base currency

Tabel `currency_rates` menyimpan riwayat kurs **append-only** dengan `effective_at`. Tidak ada edit/delete kurs; koreksi dilakukan dengan menambah baris baru dan catatan.

### 2.3 Preferensi pengguna

`users.display_currency` menyimpan pilihan pengguna (default `IDR`). Preferensi ini:

- dibaca oleh `GET /api/v1/me`;
- diubah lewat endpoint khusus preferensi, bukan lewat `PUT /me` yang bersifat full update;
- memengaruhi transaksi baru, dashboard, laporan, budget, AI, dan ekspor;
- **tidak** memengaruhi billing, invoice, pembayaran, kuota, token, atau metrik admin.

### 2.4 Zona yang tetap IDR

| Zona | Alasan |
|---|---|
| Harga paket, invoice, pajak, pembayaran, refund | Nilai kontrak dan gateway pembayaran sudah IDR |
| Token ledger, biaya AI, harga per 1k token, kuota | Satuan biaya internal dan pricing rule sudah integer Rupiah |
| Metrik admin, MRR, revenue, payment history | Konsolidasi bisnis dalam IDR |
| Halaman marketing/publik | Harga publik dalam IDR |
| Notifikasi pembayaran | Nominal transaksi pembayaran, bukan pencatatan pribadi |

---

## 3. Ruang Lingkup

### 3.1 Termasuk

- Katalog mata uang + kurs dinamis di portal Super Admin.
- Riwayat kurs dan audit perubahan.
- Preferensi mata uang pengguna di **Profil → Tampilan & Bahasa**.
- Input nominal transaksi sesuai mata uang aktif (termasuk desimal bila mata uang mendukung).
- Dashboard, statistik transaksi, daftar transaksi, detail transaksi.
- Laporan, analitik laporan, ekspor CSV/XLSX/PDF.
- Budget dan status budget.
- AI chat: konteks nominal, pembuatan transaksi, konfirmasi, receipt/OCR, voice.
- Embedding teks transaksi (agar pencarian semantik tetap menyebut mata uang).
- Fallback, validasi, error handling, audit, cache invalidation, realtime refresh.

### 3.2 Tidak termasuk (rilis ini)

- Mengubah base accounting currency dari IDR ke mata uang lain.
- Pembayaran/invoice dalam JPY atau mata uang asing.
- Konversi harga paket, pajak, refund, atau token.
- Auto-fetch kurs dari provider eksternal (disiapkan kolom `source`, implementasi menyusul).
- Multi-base ledger per pengguna (mis. pengguna menyimpan kekayaan dalam beberapa base currency).
- Kripto, emas, atau unit non-fiat.
- i18n bahasa aplikasi (JPY hanya format angka; bahasa UI tetap Indonesia).
- Override kurs manual per transaksi oleh pengguna (hanya Super Admin yang mengatur kurs).
- Sinkronisasi kurs real-time antar-negara.

---

## 4. Keputusan yang Diminta Sebelum Implementasi

| # | Pertanyaan | Rekomendasi |
|---|---|---|
| D1 | Apakah nominal transaksi disimpan dalam mata uang asli, atau hanya tampilan konversi dari IDR? | **Simpan nominal asli + snapshot kurs + amount_base** (akuntabel) |
| D2 | Apakah kurs historis per tanggal transaksi wajib? | **Ya**, append-only `currency_rates` |
| D3 | Apakah pengguna boleh mengganti mata uang kapan saja? | **Ya** untuk rilis ini; opsi kunci setelah pilihan pertama bisa ditambah nanti |
| D4 | Ekspor laporan mengikuti mata uang tampilan atau IDR? | **Mengikuti mata uang tampilan**, dengan kolom/legenda kurs |
| D5 | Sumber kurs: manual dulu atau auto-fetch? | **Manual dulu**; siapkan `source` untuk provider nanti |
| D6 | Mata uang yang di-seed? | **IDR + JPY**; JPY nonaktif sampai Super Admin mengisi kurs |
| D7 | Admin (non-super) boleh melihat katalog/kurs? | **Read-only**; perubahan hanya Super Admin |
| D8 | Batas desimal? | Ikuti metadata `decimal_digits`; JPY 0, IDR 0, USD 2 |
| D9 | Apakah laporan harus bisa melihat “nilai asli” per mata uang? | **Fase berikutnya**; rilis ini fokus nilai setara base |

Jika ada keputusan yang berbeda, dokumen ini perlu direvisi sebelum implementasi.

---

## 5. Peta Dokumen

| Dokumen | Isi |
|---|---|
| [`01-audit-kondisi-saat-ini.md`](./01-audit-kondisi-saat-ini.md) | Audit kode, titik hardcode, skema saat ini, dan gap |
| [`02-kebutuhan-dan-ruang-lingkup.md`](./02-kebutuhan-dan-ruang-lingkup.md) | Kebutuhan fungsional/non-fungsional, acceptance criteria, asumsi |
| [`03-model-domain-dan-skema-database.md`](./03-model-domain-dan-skema-database.md) | ERD, DDL, migrasi, backfill, invariant data |
| [`04-arsitektur-backend.md`](./04-arsitektur-backend.md) | Package baru, service konversi, API, cache, audit, wiring |
| [`05-arsitektur-frontend.md`](./05-arsitektur-frontend.md) | Provider, formatter, input, perubahan per halaman |
| [`06-admin-dan-ux.md`](./06-admin-dan-ux.md) | UI Super Admin, UX profil, validasi, microcopy |
| [`07-kalkulasi-dan-alur-data.md`](./07-kalkulasi-dan-alur-data.md) | Rumus, urutan alur, contoh angka, edge case |
| [`08-rencana-implementasi-migrasi-rollout.md`](./08-rencana-implementasi-migrasi-rollout.md) | Fase, urutan deploy, feature flag, rollback, observability |
| [`09-pengujian-risiko-dan-keputusan.md`](./09-pengujian-risiko-dan-keputusan.md) | Matriks uji, risiko, decision log, checklist persetujuan |

---

## 6. Definisi Istilah

| Istilah | Definisi |
|---|---|
| **Base currency** | Mata uang acuan pembukuan. Pada rilis ini IDR. |
| **Display currency** | Mata uang yang dipilih pengguna untuk input dan tampilan. |
| **Original amount** | Nominal yang diketik pengguna dalam display currency. |
| **Base amount** | Nilai setara dalam base currency (IDR), dibekukan saat transaksi. |
| **Rate to base** | Jumlah unit base untuk 1 unit mata uang. Contoh: 1 JPY = 110 IDR → `110`. |
| **Rate as of** | Kurs terbaru yang `effective_at <= tanggal acuan`. |
| **Append-only rate** | Riwayat kurs tidak diubah/dihapus; koreksi = baris baru. |
| **Agregat** | Total/statistik yang dihitung dari banyak transaksi. |
| **Snapshot** | Nilai kurs yang disimpan bersama transaksi agar historis tidak berubah. |

---

## 7. Kriteria Keberhasilan

1. Pengguna dapat memilih IDR atau JPY di **Profil → Tampilan & Bahasa**, tersimpan setelah reload dan lintas perangkat.
2. Transaksi baru menampilkan nominal sesuai mata uang aktif; nominal asli dan nilai IDR tersimpan benar.
3. Dashboard, daftar transaksi, statistik, laporan, budget, dan ekspor menampilkan mata uang aktif secara konsisten.
4. Perubahan kurs oleh Super Admin langsung terlihat tanpa redeploy dan tanpa restart.
5. Laporan historis tetap memakai kurs pada tanggal acuan, bukan kurs terbaru.
6. Billing, invoice, pembayaran, token, kuota, dan metrik admin **tidak berubah** dan tetap IDR.
7. Seluruh perubahan kurs tercatat di audit log.
8. Tidak ada lagi formatter IDR hardcode pada permukaan keuangan pengguna.
9. `go test ./...`, `pnpm test`, `pnpm lint`, dan `pnpm build` hijau.
10. Backfill transaksi lama menghasilkan `amount_base = amount` dan tidak ada selisih agregat.

---

## 8. Status Implementasi

Dokumen 01–09 **lengkap** dan implementasi rilis ini **sudah berjalan**. Keputusan D1–D12 dipakai apa adanya (D1 original + snapshot + base, D2 append-only, D4 ekspor mengikuti mata uang tampilan + legenda kurs, D6 seed IDR + JPY nonaktif, D7 katalog read-only untuk admin non-super, D8 `decimal_digits`).

### 8.1 Yang sudah terpasang

| Lapisan | Artefak |
|---|---|
| Skema | `database/migrations/0039_currency_catalog.sql` … `0042_currency_constraints.sql` (`currencies`, `currency_rates`, `users.display_currency`, kolom transaksi, `amount_base` GENERATED STORED) |
| Domain | `backend/internal/currency/currency.go`, `domain/user.go`, `domain/transaction.go`, `domain/report.go` |
| Repo | `repository/postgres/currency_repo.go`, `transaction_repo.go` (agregasi `SUM(amount_base)`), `user_repo.go` |
| Usecase | `currency_usecase.go` (validasi + merge metadata presence-aware), `transaction_usecase.go` (`resolveCurrency`/`applyRate` via `RateForWrite`), `report_usecase.go` (`applyDisplayCurrency` — konversi memakai kurs akhir periode), `auth_usecase.go`, `ai_usecase.go` + `agent_tools.go` |
| HTTP | `handler/currency_handler.go`, `handler/auth_handler.go`, `router.go` (`RouterConfig.Currency`) |
| Ekspor | `export/export.go` + `export/pdf.go` (memformat nilai yang sudah dikonversi; legenda Currency/Rate) |
| Frontend | `lib/currency.ts`, `providers/currency-provider.tsx`, `hooks/useCurrencies.ts`, `hooks/useSuperAdminCurrency.ts`, `components/profile/AppearanceSettings.tsx`, `app/(dashboard)/super-admin/currency/page.tsx` |
| Endpoint | `GET /api/v1/currencies`, `PUT /api/v1/me/preferences/currency`, `GET/POST /api/v1/admin/currencies`, `GET/POST /api/v1/admin/currency-rates` |

### 8.2 Penyimpangan kecil yang disengaja

1. **Konversi laporan di server.** Backend mengonversi agregat laporan memakai kurs pada **akhir periode** dan mengisi `CurrencyCode`/`RateUsed`/`RateAt`; bila tidak ada kurs untuk tanggal itu, nilai dibiarkan IDR dan `CurrencyCode` dipaku `IDR`. Frontend (`formatDisplay`) dan ekspor **tidak** mengonversi lagi — hanya memformat. Grafik laporan memakai mode `raw` (nilai sudah dikonversi) sehingga tidak ada konversi ganda.
2. **Grafik default IDR.** `PieChartView`/`BarChartView`/`LineChartView`/`AreaChartView`/`MultiSeriesChart` memakai `formatRupiah` kecuali prop `display` (base → display) atau `raw` (sudah dikonversi) diberikan. Halaman pengguna: dashboard/AI/analitik memberi `display`; laporan memberi `raw`; metrik admin/super-admin tetap IDR.
3. **Budget.** Tabel budget tetap `amount_idr`; form mengonversi input tampilan → base (`toBase`, dibulatkan) dan agregat dikonversi saat render.
4. **Struk/OCR.** Nominal OCR adalah IDR, jadi payload transaksi dari struk memaku `currency_code: "IDR"` agar tidak salah dibaca sebagai mata uang tampilan.
5. **AI.** Tool `create_transaction` menerima nominal dalam mata uang tampilan dan menyimpan snapshot kurs; konteks/konfirmasi AI diformat dalam mata uang tampilan. Billing/token/kuota AI tetap IDR.

### 8.3 Perbaikan audit (hardening, 2026-10-19)

Audit mendalam pasca-implementasi menemukan dan memperbaiki:

1. **Migrasi 0041 destruktif & 0042 tidak idempoten.** `0041` sebelumnya me-reset semua baris non-IDR (akan menghapus data JPY yang sah saat dijalankan ulang); kini hanya memperbaiki baris IDR yang tidak konsisten. `0042` kini `DROP CONSTRAINT IF EXISTS` sebelum `ADD` + `VALIDATE`.
2. **Metadata katalog terhapus tak sengaja.** Upsert kini presence-aware (`CurrencyInput` dengan pointer): field yang tidak dikirim dipertahankan, `is_active=false` tetap dihormati, dan validasi baru menolak kode bukan 3 huruf, `decimal_digits` di luar 0–4, locale tidak valid, simbol/nama kosong.
3. **Aktivasi tanpa kurs berlaku.** Aktivasi sekarang menuntut kurs yang efektif **saat ini** (bukan sekadar ada baris), mencegah mata uang tak terpakai.
4. **Crash locale di frontend.** `Intl.NumberFormat` dibungkus fallback `id-ID`; locale divalidasi server-side.
5. **Kurs hilang membuat tulis gagal senyap.** `RateForWrite` memvalidasi mata uang aktif dan jatuh ke kurs terbaru untuk transaksi bertanggal sebelum riwayat kurs.
6. **`rate_at` tidak terhapus saat pindah ke IDR.** Update menulis `rate_at` langsung (bukan `COALESCE`).
7. **Kurs ganda / non-finite.** `currency_rates` unik per `(currency_code, effective_at)`; duplikat → `409`; rate divalidasi positif & finite dengan batas atas.
8. **Preferensi mata uang nonaktif.** `displayCurrency` memvalidasi dan jatuh ke IDR bila pilihan pengguna sudah tidak selectable (tidak mengunci tulis).
9. **Sortir nominal campuran.** Sortir `amount` memakai `amount_base`, bukan nominal asli.
10. **Embedding.** Backfill embedding kini menyertakan `currency_code`/`amount_base` agar teks tidak menyebut IDR untuk transaksi JPY.
11. **Input desimal.** `RupiahInput` menerima prop `decimals` (memakai `decimal_digits` mata uang) untuk USD/MYR nanti; default tetap integer.
12. **Detail transaksi.** Daftar/detail/detail-dialog menampilkan **nominal asli + kode mata uang** (tidak di-re-quote oleh kurs baru); agregat tetap dalam mata uang tampilan.
13. **Parsing nominal AI.** `create_transaction` menerima nominal sebagai string **atau** angka JSON.

### 8.4 Verifikasi

- `go build ./...`, `go vet ./...`, `go test ./... -run zzzNoMatch` hijau.
- `npx tsc --noEmit` bersih untuk seluruh file multi-currency (7 error lama di test tak terkait).
- `vitest run lib/currency.test.ts lib/receipt.test.ts components/charts components/reports hooks/useReports.test.ts components/shared/RupiahInput.dom.test.tsx` hijau (33 test).
- Satu test lama `components/layout/SearchCommand.dom.test.tsx` flaky pada run paralel penuh; lulus saat dijalankan sendiri (tidak terkait perubahan ini).

### 8.5 Langkah berikutnya

1. Jalankan migrasi `0039`–`0042` di staging, lalu isi kurs JPY lewat Super Admin → aktifkan JPY.
2. Uji manual: ganti mata uang di profil, buat transaksi JPY, cek dashboard/laporan/ekspor, kembalikan ke IDR.
3. Backfill diverifikasi: `SELECT count(*) FROM transactions WHERE currency_code='IDR' AND (exchange_rate<>1 OR amount_base<>amount)` harus `0`.
