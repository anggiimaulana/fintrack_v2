# 07 — Kalkulasi & Alur Data

**Status:** DRAFT  
Semua rumus di sini adalah sumber kebenaran. Bila dokumen lain berselisih, dokumen ini yang menang.

---

## 1. Notasi

| Simbol | Arti |
|---|---|
| `A` | `amount` — nominal asli (mata uang transaksi) |
| `C` | `currency_code` |
| `R` | `exchange_rate` — 1 unit `C` = `R` IDR (snapshot) |
| `B` | `amount_base` — nilai IDR tersimpan |
| `base` | IDR (kode `IDR`, `R=1`) |
| `D` | `display_currency` pengguna |
| `RO(t)` | rate-as-of: `rate_to_base` terbaru dengan `effective_at <= t` |
| `digits(x)` | jumlah desimal mata uang `x` |

---

## 2. Rumus Inti

### 2.1 Saat menulis transaksi

```
R = 1                             jika C == IDR
R = RO(C, transaction_date)       selainnya     → error bila tidak ada
B = round(A × R, 2)               (IDR selalu 2 desimal)
rate_at = effective_at dari R
```

Invariant: `B = round(A × R, 2)`.

### 2.2 Tampilan nominal transaksi individual

```
teks = format(A, C)     → nominal asli, tidak berubah selamanya
```

Baris sekunder (opsional):
```
≈ format(B, IDR)  (kurs R pada rate_at)
```

### 2.3 Tampilan agregat

Agregasi dilakukan **di IDR** lebih dulu:
```
B_periode = SUM(B) untuk transaksi dalam periode
```
lalu dikonversi ke mata uang tampilan memakai kurs as-of tanggal acuan `t_ref`:
```
t_ref = now()                 untuk dashboard / "hari ini"
t_ref = period_end            untuk laporan / ekspor
A_display = round(B_periode / RO(D, t_ref), digits(D))
```

Bila `D == IDR`: `A_display = B_periode` (tanpa pembagian).

### 2.4 Budget

Storage tetap IDR (`budgets.amount_idr`):
```
budget_input_user (D)  →  amount_idr = round(input × RO(D, now), 2)
```
Perbandingan:
```
aktual = SUM(B) untuk kategori & periode
sisa   = amount_idr − aktual
persen = aktual / amount_idr × 100
```
Tampilan: `round(sisa / RO(D, now), digits(D))`.

### 2.5 Statistik transaksi (net balance)

```
total_income  = SUM(B WHERE type='income')
total_expense = SUM(B WHERE type='expense')
total_savings = SUM(B WHERE type='savings')
net           = total_income − total_expense     (IDR)
```
Konversi ke `D` hanya di akhir, sekali.

---

## 3. Alur: Membuat Transaksi

```
[FE] pengguna mengetik 1000, mata uang aktif JPY
  │  POST /transactions { amount: 1000, currency_code: "JPY", type, category, date }
  ▼
[BE] transaction_usecase.Create
  │  1. resolve currency  → meta + is_active?  (tolak bila nonaktif)
  │  2. R = RO(JPY, date)                       (tolak bila tidak ada)
  │  3. B = round(1000 × R, 2)
  │  4. simpan amount=1000, currency_code=JPY, exchange_rate=R,
  │            amount_base=B, rate_at=RO.effective_at
  │  5. enqueue SPK + embed (teks menyebut JPY)
  ▼
[FE] invalidate transactions.all, dashboard.all, reports.all, budgets.all
```

---

## 4. Alur: Mengubah Transaksi

Kasus yang harus benar:

| Perubahan | Aksi server |
|---|---|
| Hanya deskripsi | `amount`, `currency_code`, `exchange_rate`, `amount_base`, `rate_at` tidak berubah |
| `amount` berubah (mata uang sama) | `B = round(A_baru × R_lama, 2)`; **R dipertahankan** (kurs historis transaksi) |
| `currency_code` berubah | `R = RO(C_baru, transaction_date)`; `rate_at` diperbarui; `B` dihitung ulang |
| `amount` + `currency_code` | seperti di atas |

Rasional: nilai dasar harus mencerminkan mata uang yang benar-benar dicatat. Mengubah nominal pada mata uang yang sama **tidak** boleh menggeser kurs historis (kalau tidak, laporan lama berubah). Keputusan D11 (dokumen 09).

---

## 5. Alur: Ganti Preferensi Mata Uang

```
[FE] PUT /me/preferences/currency { display_currency: "JPY" }
  ▼
[BE] 1. validasi kode ada & is_active   (tolak 422)
     2. base currency selalu boleh
     3. UPDATE users.display_currency
     4. audit (opsional)
  ▼
[FE] invalidate profile.me + semua cache nominal
     → seluruh UI re-render dengan D=JPY (nilai dari server/konversi FE)
```

Data historis **tidak disentuh**. Hanya cara menampilkan agregat yang berubah.

---

## 6. Alur: Super Admin Menambah Kurs

```
[SA] POST /admin/currency-rates { code:"JPY", rate_to_base:110, effective_at:"2026-10-19T00:00Z" }
  ▼
[BE] INSERT (append-only) + audit currency_rate_added
     invalidate cache rate + masuk Redis version bump
     publish config_changed { domain:"currency" }
  ▼
[FE] SSE → invalidate currency.catalog + currency.rates
     → dashboard & laporan (yang memakai t_ref >= effective_at) ikut refresh
```

Transaksi yang sudah tersimpan memakai `R` snapshot-nya sendiri → **tidak berubah**.

---

## 7. Contoh Angka

### 7.1 Transaksi IDR (hari ini)

```
A = 50.000, C = IDR, R = 1
B = 50.000
Tampil (D=IDR): Rp50.000
```

### 7.2 Transaksi JPY, D=JPY

```
A = 1.000, C = JPY, R = RO(JPY, 2026-10-19) = 110
B = round(1000 × 110, 2) = 110.000
Tampil baris: ¥1.000
Agregat bulan ini (mis. hanya transaksi ini): B=110.000 → /110 = ¥1.000 ✓
```

### 7.3 Transaksi JPY, D=IDR (pengguna lihat IDR)

```
Agregat = 110.000 (IDR) → tampil Rp110.000
Transaksi individual tetap ¥1.000
```

### 7.4 Kurs berubah setelah transaksi

```
Transaksi: R=110, B=110.000
Kurs baru 1 JPY = 115 IDR
- Baris transaksi: tetap ¥1.000
- Laporan periode `t_ref` sebelum efektif kurs baru: pakai 110
- Dashboard hari ini (setelah kurs baru): 110.000/115 ≈ ¥957
```

Nilai dasar IDR tidak berubah; hanya representasi tampilan harian.

### 7.5 USD di masa depan (2 desimal)

```
A = 12,50 USD, R = 16.000 IDR/USD
B = round(12.5 × 16000, 2) = 200.000
```

### 7.6 Budget

```
D=JPY, budget input ¥10.000, RO(JPY, now)=110
amount_idr = 1.100.000
aktual SUM(B) = 550.000 → sisa 550.000 → tampil ¥5.000
persen = 50%
```

---

## 8. Edge Case & Aturan

| Kasus | Perilaku |
|---|---|
| Rate tidak ada untuk mata uang nonaktif | tolak, 422 `currency_no_rate` |
| Mata uang dinonaktifkan setelah dipakai | transaksi lama tetap valid; pilihan user di-fallback ke IDR |
| `amount` = 0 | tidak boleh (`check amount > 0`) |
| JPY dengan desimal | ditolak UI; bila lolos, dibulatkan 0 saat tampilan (storage 2 desimal aman) |
| Rate sangat kecil (IDR→USD ≈ 0,0000625) | `numeric(20,10)` cukup; ini arah `ConvertFromBase`, bukan disimpan |
| Dua rate `effective_at` sama | `UNIQUE` menolak; koreksi = waktu berbeda |
| Periode laporan lintas pergantian kurs | satu `t_ref` (akhir periode); tidak mencampur rate per hari (rilis ini) |
| Backfill data lama | `B = A`, `C=IDR`, `R=1`; agregat identik dengan sebelum migrasi |
| Pembulatan agregat | bulatkan **setelah** pembagian, sekali; jangan bulatkan per baris |

**Anti-pattern yang dilarang:**
- Menjumlahkan `amount` lintas mata uang berbeda.
- Menyimpan `amount_base` dari kurs terbaru (harus kurs transaksi).
- Mengonversi `amount_base` kembali ke IDR (identitas, tidak perlu).
- Mengubah rate lama (harus append).
- Memakai `amount` untuk agregat apa pun.

---

## 9. Hook FE untuk Konversi Agregat

Bila server mengirim base IDR + `rate_used`, FE tidak menghitung konversi:

```
{ total_income: 110000, currency: "JPY", rate_used: 110, rate_at: "..." }
  → formatMoney(110000 / 110, ctx)  // atau server kirim total_income_display
```

Rekomendasi: **server mengirim nilai display** (`*_display`) untuk laporan/dashboard, dan raw base untuk audit/legenda. Ini menghilangkan dua implementasi pembulatan.

---

## 10. Urutan Determinisme

Untuk memastikan hasil sama di server dan uji:

1. Tentukan `C`, `R`, `t_ref` secara eksplisit.
2. Hitung di ruang IDR (2 desimal).
3. Bulatkan ke `digits(D)` hanya di langkah terakhir.
4. Jangan gunakan float mentah untuk nominal > 2^53; gunakan decimal pada server.

---

## 11. Ringkasan Rumus

```
Tulis:    R = RO(C, date)          B = round(A·R, 2)      rate_at = R.effective_at
Agregat:  B_per   = Σ B
Tampil:   A_disp  = round(B_per / RO(D, t_ref), digits(D))
Budget:   idr     = round(input·RO(D, now), 2)
Net:      net     = ΣB_income − ΣB_expense
```
