# 09 — Pengujian, Risiko & Decision Log

**Status:** DRAFT  
Dokumen terakhir: matriks uji, risiko, log keputusan (D1–D12), dan checklist persetujuan.

---

## 1. Matriks Pengujian

### 1.1 Unit — backend

| ID | Skenario | Harapan |
|---|---|---|
| U-01 | `ConvertToBase(IDR, 50000)` | 50000, rate 1, tanpa query rate |
| U-02 | `ConvertToBase(JPY, 1000, rate 110)` | 110000 |
| U-03 | `ConvertToBase` JPY tanpa rate | `ErrNoRate` |
| U-04 | `ConvertToBase` mata uang nonaktif | `ErrInactiveCurrency` |
| U-05 | `ConvertFromBase(JPY, 110000, rate 115)` | round(956.52, 0) = 957 |
| U-06 | Rate-as-of memilih baris ≤ t terbaru | benar saat ada 3 riwayat |
| U-07 | Pembulatan 2 desimal IDR | tidak ada float error |
| U-08 | Update transaksi hanya deskripsi | rate & amount_base tidak berubah |
| U-09 | Update `amount` mata uang sama | `B = A_baru × R_lama` |
| U-10 | Update `currency_code` | rate dihitung ulang as-of tanggal |

### 1.2 Unit — frontend

| ID | Skenario | Harapan |
|---|---|---|
| F-01 | `formatMoney` IDR | identik `formatRupiah` |
| F-02 | `formatMoney` JPY | `¥1,000`, 0 desimal |
| F-03 | `formatMoney` USD | 2 desimal |
| F-04 | nilai null/NaN | `0` terformat |
| F-05 | Provider belum login | fallback IDR |
| F-06 | Realtime `currency` | invalidate catalog + rates |
| F-07 | Snapshot IDR sebelum/sesudah | sama persis |

### 1.3 Integrasi / API

| ID | Skenario | Harapan |
|---|---|---|
| I-01 | `POST /transactions` JPY | `amount_base` benar, `rate_at` terisi |
| I-02 | `POST` JPY tanpa rate | 422 `currency_no_rate` |
| I-03 | `PUT /me/preferences/currency` JPY aktif | tersimpan, `GET /me` mencerminkan |
| I-04 | `PUT` preferensi ke nonaktif | 422 |
| I-05 | `POST /admin/currencies` sebagai admin biasa | 403 |
| I-06 | Aktivasi JPY tanpa rate | 422 |
| I-07 | `POST /admin/currency-rates` duplikat (code, effective_at) | 409 |
| I-08 | Riwayat kurs tidak bisa diubah/dihapus | 405/404 |
| I-09 | GET agregat setelah campur IDR+JPY | konsisten dengan `amount_base` |
| I-10 | Ekspor JPY | header memuat kurs, angka 0 desimal |
| I-11 | Laporan periode lampau setelah kurs baru | memakai kurs akhir periode |

### 1.4 Regresi KRITIS — zona IDR beku

| ID | Skenario | Harapan |
|---|---|---|
| Z-01 | Invoice/pajak/pembayaran | tetap IDR, tidak berubah |
| Z-02 | Token ledger & biaya AI | tetap IDR |
| Z-03 | Kuota & estimasi | tetap IDR |
| Z-04 | Harga paket & halaman marketing | tetap IDR |
| Z-05 | Metrik admin (MRR, revenue) | tetap IDR |
| Z-06 | Tidak ada pemanggilan `currency.Service` dari paket billing/token/quota | grep + test |

### 1.5 Migrasi

| ID | Skenario | Harapan |
|---|---|---|
| M-01 | Backfill pada data lama | `amount_base = amount`, 0 selisih |
| M-02 | Migrasi idempoten (dijalankan dua kali) | tidak error, tidak menggandakan |
| M-03 | Constraint `amount_base` | menolak baris tidak konsisten |
| M-04 | Seed IDR/JPY | IDR aktif base, JPY nonaktif |
| M-05 | Rollback kolom | backend lama tetap jalan |

### 1.6 Manual / E2E

| ID | Skenario |
|---|---|
| E-01 | Ganti IDR→JPY di profil, reload, lintas perangkat |
| E-02 | Buat transaksi JPY, lihat baris & agregat |
| E-03 | Super Admin tambah kurs, user lihat update tanpa reload manual |
| E-04 | Ekspor laporan JPY, buka di Excel |
| E-05 | AI buat transaksi saat D=JPY |
| E-06 | Budget JPY vs aktual |
| E-07 | Kembalikan ke IDR, snapshot sama seperti awal |

---

## 2. Risiko & Mitigasi

| # | Risiko | Dampak | Mitigasi |
|---|---|---|---|
| R-01 | Laporan historis berubah saat kurs naik | Akuntabilitas hilang | Snapshot `exchange_rate` + `rate_at` per transaksi; agregat pakai base IDR |
| R-02 | Agregat campur mata uang | Angka salah | Semua agregat `SUM(amount_base)`; uji I-09 |
| R-03 | Billing ikut terkonversi | Keuangan bisnis kacau | Zona beku + test Z-01..Z-06 |
| R-04 | Float error nominal besar | Selisih sen | `numeric`/decimal di server; bulat 2 desimal IDR |
| R-05 | Predator rate hilang (JPY nonaktif tanpa rate) | Transaksi gagal membingungkan | Validasi aktivasi + error 422 jelas |
| R-06 | Backfill memblokir tabel besar | Downtime | Constraint `NOT VALID`, batch, index concurrent |
| R-07 | Formatter IDR tersisa di permukaan pengguna | Inkonsisten | Audit bertahap; snapshot IDR + uji grep |
| R-08 | Rate-as-of query lambat | Latensi dashboard | Index `(currency_code, effective_at DESC)`; cache per hari |
| R-09 | Admin salah input kurs | Nilai tampilan melenceng | Append-only + catatan + audit; koreksi baris baru |
| R-10 | Desimal JPY salah (mis. ¥1,000.00) | Tampilan buruk | `decimal_digits` per mata uang; `CurrencyInput` membatasi |
| R-11 | Dua implementasi pembulatan (BE & FE) | Selisih hasil | Server kirim nilai display; FE hanya format (dokumen 07 §9) |
| R-12 | Nonaktifkan mata uang yang sedang dipakai | Preferensi invalid | Fallback ke IDR saat load; validasi server |
| R-13 | Ekspor XLSX menyimpan teks bukan angka | Tidak bisa dihitung | Tulis numerik + format sel, bukan string |
| R-14 | Embedding menyebut Rp padahal JPY | Pencarian semantik salah | `buildEmbeddingText` sertakan kode mata uang |
| R-15 | Timezone `effective_at` salah hari | Kurs beda sehari | Simpan `timestamptz`; konversi sesuai zona aplikasi |

---

## 3. Non-Goals (ditegaskan)

- Base currency tidak berubah dari IDR.
- Tidak ada pembayaran/invoice non-IDR.
- Tidak ada auto-fetch kurs dari provider (kolom `source` disiapkan).
- Tidak ada i18n bahasa UI.
- Tidak ada override kurs per transaksi oleh user.
- Tidak ada multi-base ledger per pengguna.

---

## 4. Decision Log

### 4.1 Dari README (D1–D9)

| # | Pertanyaan | Keputusan |
|---|---|---|
| D1 | Simpan nominal asli atau konversi tampilan saja? | Simpan asli + snapshot + `amount_base` |
| D2 | Kurs historis wajib? | Ya, append-only |
| D3 | Pengguna boleh ganti kapan saja? | Ya |
| D4 | Ekspor ikut mata uang tampilan? | Ya + legenda kurs |
| D5 | Manual atau auto-fetch kurs? | Manual dulu |
| D6 | Mata uang di-seed? | IDR + JPY (JPY nonaktif) |
| D7 | Admin boleh lihat katalog? | Read-only |
| D8 | Batas desimal? | Ikuti `decimal_digits` |
| D9 | Laporan nilai asli per mata uang? | Fase berikutnya |

### 4.2 Keputusan tambahan (D10–D12)

| # | Pertanyaan | Keputusan | Alasan |
|---|---|---|---|
| D10 | Library desimal backend? | `shopspring/decimal` bila belum ada; jika menghindari dependensi, `big.Rat`/integer minor units | Presisi nominal & rate |
| D11 | Semantik update saat `amount` berubah? | Pertahankan rate historis bila mata uang sama; hitung ulang bila mata uang berubah | Cegah laporan lama berubah |
| D12 | Di mana konversi agregat dilakukan? | Server (kirim nilai display + base) | Satu implementasi pembulatan |

### 4.3 Keputusan skema/arsitektur (ringkas)

| # | Keputusan |
|---|---|
| S-1 | Tabel `currencies` + `currency_rates`; bukan JSON di `settings` |
| S-2 | `users.display_currency` di DB; bukan localStorage |
| S-3 | Endpoint preferensi terpisah dari `PUT /me` |
| S-4 | Base code IDR satu konstanta di `internal/currency` |
| S-5 | Realtime memakai event `config_changed` yang ada |
| S-6 | Flag `multi_currency` menggerbang UI |
| S-7 | Migrasi 0039–0042 |
| S-8 | `budgets.amount_idr` tidak berubah |

---

## 5. Pertanyaan Terbuka (perlu jawaban sebelum implementasi)

| # | Pertanyaan | Dampak bila salah |
|---|---|---|
| Q-1 | Apakah pengguna perlu melihat kolom "nilai asli" di laporan (D9)? | Desain laporan |
| Q-2 | Apakah sortir daftar transaksi harus berdasar nilai dasar agar sebanding? | UX daftar campuran mata uang |
| Q-3 | Apakah kurs boleh dijadwalkan (effective_at masa depan)? | Validasi admin |
| Q-4 | Apakah pembulatan laporan perlu mode (ceil/floor) seperti kuota? | Akurasi pajak/legal |
| Q-5 | Perlu batas jumlah mata uang aktif? | Performa picker |

Semua dapat dijawab sambil berjalan untuk rilis ini; Q-1 dan Q-2 paling berdampak.

---

## 6. Checklist Persetujuan

- [ ] D1–D9 disetujui (atau direvisi).
- [ ] D10–D12 disetujui.
- [ ] S-1–S-8 disetujui.
- [ ] Non-goals diterima.
- [ ] Zona IDR beku disepakati.
- [ ] Rencana 8 PR (dokumen 08 §10) disepakati.
- [ ] Owner implementasi & QA ditunjuk.

Setelah checklist ini terisi, buat **implementation plan** terpisah dan mulai dari Fase 1 (migrasi).

---

## 7. Definisi Selesai (DoD)

1. Semua test U/F/I/Z/M/E hijau.
2. Snapshot tampilan IDR identik dengan sebelum perubahan.
3. Backfill 0 selisih.
4. `go test ./...`, `pnpm test`, `pnpm lint`, `pnpm build` hijau.
5. Kurs JPY dapat ditambah Super Admin tanpa deploy.
6. Billing/AI-quota terbukti tidak terkonversi.
7. Audit perubahan kurs tercatat.
8. Dokumentasi 01–09 diperbarui sesuai hasil.

---

## 8. Penutup

Rancangan ini memenuhi permintaan: pemilihan mata uang di profil, katalog & kurs dinamis dari Super Admin tanpa hardcode, seluruh permukaan keuangan pengguna mengikuti pilihan, dan billing/AI tetap IDR. Inti keamanannya adalah **model nilai ganda + riwayat kurs append-only + base IDR**, yang membuat perubahan kurs tidak pernah merusak data lama.

Langkah berikutnya: konfirmasi keputusan dan pertanyaan terbuka, lalu implementasi bertahap sesuai dokumen 08.
