# 06 — Admin & UX

**Status:** DRAFT  
Dokumen ini merinci UX Super Admin (katalog & kurs) dan UX pengguna (profil & input), plus validasi dan microcopy.

---

## 1. Peta Peran

| Peran | Hak |
|---|---|
| **User** | Memilih mata uang pencatatan aktif, membuat transaksi, laporan, ekspor |
| **Admin** | Melihat katalog & riwayat kurs (read-only) |
| **Super Admin** | Tambah/ubah mata uang, aktif/nonaktif, tambah kurs, audit |

Pemisahan mengikuti pola router yang ada: `adminRoutes` (`RequireAdmin`) untuk GET, group `sa` (`RequireSuperAdmin`) untuk POST/PUT (dokumen 04 §6.2).

---

## 2. UX Pengguna — Profil

**Lokasi:** `/user/profile` → grup **"Tampilan & Bahasa"** (akordeon yang sudah ada).

Tata letak baru (grid tetap 2 kolom di desktop; tambah baris untuk mata uang):

```
┌─────────────────────────────┬─────────────────────────────┐
│ Tema                        │ Bahasa & Regional           │
│ [Terang][Gelap][Sistem]     │ [Indonesia (ID) ▾ disabled] │
│                             │ ID tetap: belum ada i18n    │
├─────────────────────────────┼─────────────────────────────┤
│ Mata Uang Pencatatan        │ Kurs yang dipakai           │
│ [Rupiah (IDR) ▾]            │ 1 JPY = 110,0000 IDR        │
│ Catatan biaya AI tetap IDR  │ diperbarui 19 Okt 2026      │
└─────────────────────────────┴─────────────────────────────┘
```

Detail:
- Dropdown menampilkan semua mata uang `is_active`. Hanya IDR bila JPY belum diaktifkan.
- Di sebelahnya tampilkan **kurs aktif** mata uang terpilih (transparansi).
- Perubahan langsung disimpan (optimistic), dengan spinner kecil di samping label.
- Toast sukses: "Mata uang pencatatan diubah ke Yen."
- Toast gagal: "Kurs JPY belum tersedia." / "Mata uang tidak tersedia."
- Catatan tetap ada: **"Biaya langganan, token, dan AI tetap dalam Rupiah."**

Microcopy kurs tidak boleh menampilkan angka fiktif. Bila JPY aktif tetapi rate lama, tampilkan tanggal rate, bukan "hari ini".

---

## 3. UX Pengguna — Input & Tampilan

| Layar | Perilaku |
|---|---|
| Form transaksi | Simbol mata uang aktif di prefix; boleh 0 desimal untuk JPY/IDR |
| Daftar transaksi | Tiap baris: nominal **asli** + kode (mis. `¥1.000`); transaksi IDR lama `Rp50.000` |
| Detail transaksi | Nominal asli + baris kecil "≈ Rp110.000 (kurs 110,00 pada 19 Okt 2026)" |
| Dashboard | Kartu total = agregat dikonversi ke mata uang aktif |
| Laporan | Total dikonversi dengan kurs **akhir periode**; legenda kurs di header |
| Ekspor | Judul memuat mata uang + kurs |
| Struk/OCR | Nominal yang diekstrak disimpan pada mata uang aktif |

**Campuran mata uang:** jika pengguna pernah mencatat JPY lalu beralih ke IDR, daftar akan menampilkan dua kode berbeda. Ini benar dan disengaja — nilai historis tidak berubah. Agregat tetap konsisten karena memakai `amount_base` (IDR).

---

## 4. UX Super Admin — Katalog Mata Uang

**Halaman baru:** `/super-admin/currencies` (mengikuti pola halaman `settings`/`plans`).

### 4.1 Tabel katalog

| Kolom | Isi |
|---|---|
| Kode | IDR, JPY, … |
| Nama | Rupiah Indonesia, Yen Jepang |
| Simbol | Rp, ¥ |
| Desimal | 0 / 2 |
| Status | Aktif / Nonaktif (badge) |
| Kurs terkini | nilai + tanggal, atau "belum ada" |
| Aksi | Aktifkan / Nonaktifkan / Ubah |

Kolom **Kurs terkini** memakai rate terbaru `effective_at <= now()`. Mata uang nonaktif menampilkan kurasi terakhir sebagai referensi.

### 4.2 Form tambah/ubah

| Field | Validasi |
|---|---|
| Kode ISO 4217 | 3 huruf, unik, tidak dapat diubah setelah dibuat |
| Nama | 1–64 karakter |
| Simbol | 1–8 karakter |
| Locale | pola `xx-XX` (mis. `ja-JP`) |
| Desimal | 0–4 |
| Urutan | integer ≥ 0 |
| Sumber kurs | `manual` (rilis ini) |

### 4.3 Aturan aktivasi (ditegakkan server)

- **Aktifkan** hanya bila ada ≥ 1 baris rate valid → else error "Isi kurs JPY terlebih dahulu."
- **Base currency (IDR)** tidak dapat dinonaktifkan atau diubah kodenya.
- Nonaktifkan JPY yang sedang dipakai user: user tersebut di-fallback ke IDR saat load berikutnya.

### 4.4 Panel riwayat kurs (append-only)

- Daftar baris: tanggal efektif, `1 CODE = X IDR`, sumber, catatan, pembuat.
- **Tidak ada tombol edit/hapus.** Hanya "Tambah kurs".
- Koreksi = tambah baris baru dengan catatan, mis. "Koreksi input 19 Okt".
- Tampilkan grafik kecil kurs 30/90 hari (opsional, memakai komponen chart yang ada).

---

## 5. UX Super Admin — Tambah Kurs

Form minimal:

```
Mata Uang     [JPY ▾]
1 JPY =       [110,0000] IDR
Berlaku dari  [2026-10-19 00:00]
Catatan       [Kurs BI 19 Okt]
[Simpan]
```

Validasi:
- `rate_to_base > 0`.
- `effective_at` boleh masa depan (kurs terjadwal) dengan catatan eksplisit di UI.
- Duplikat `(code, effective_at)` → 409 "Kurs untuk waktu itu sudah ada."
- Konfirmasi bila `effective_at` di masa lalu **dan** mengubah nilai agregat laporan publik (opsional; hanya peringatan).

Microcopy: "Riwayat kurs tidak dapat dihapus. Untuk mengoreksi, tambahkan baris baru."

---

## 6. UX Admin (read-only)

- Halaman yang sama, tombol mutasi disembunyikan untuk `admin` (bukan super admin).
- Bila admin mencoba POST/PUT secara langsung → 403 dari server (bukan hanya UI).

---

## 7. Audit UX

Halaman `/super-admin/audit` yang ada menampilkan entri:
- `currency_created`, `currency_updated`, `currency_rate_added`.
- Filter berdasar domain "currency" (mengikuti pola filter audit yang ada).

---

## 8. Microcopy Lengkap

| Konteks | Teks |
|---|---|
| Label profil | Mata Uang Pencatatan |
| Catatan profil | Biaya langganan, token, dan AI tetap dalam Rupiah. |
| Sukses ganti | Mata uang pencatatan diubah ke {nama}. |
| Gagal no rate | Kurs {kode} belum tersedia. Hubungi admin. |
| Gagal nonaktif | Mata uang tidak tersedia. |
| Admin aktifkan tanpa rate | Isi kurs {kode} terlebih dahulu. |
| Riwayat kurs | Riwayat kurs tidak dapat dihapus. Tambahkan baris baru untuk mengoreksi. |
| Detail transaksi | ≈ {base} (kurs {rate} pada {tanggal}) |
| Ekspor header | Mata uang: {kode} (1 {kode} = {rate} IDR, kurs {tanggal}) |

---

## 9. Aksesibilitas & State

- Dropdown mata uang: `<select>` native (bukan custom div) agar keyboard/screen-reader benar.
- Perubahan preferensi punya `aria-live="polite"` untuk status simpan.
- Skeleton saat katalog memuat; fallback IDR bila gagal.
- Angka besar tetap format locale; tidak memotong desimal yang bermakna (JPY 0, USD 2).

---

## 10. Ringkasan Keputusan UX

| # | Keputusan |
|---|---|
| U-1 | Picker mata uang di grup "Tampilan & Bahasa" profil |
| U-2 | Tampilkan kurs aktif di samping picker (transparansi) |
| U-3 | Halaman `/super-admin/currencies` + panel riwayat kurs |
| U-4 | Admin read-only; mutasi hanya Super Admin |
| U-5 | Aktivasi menuntut rate valid |
| U-6 | Riwayat kurs tidak dapat dihapus dari UI |
| U-7 | Baris transaksi = nominal asli; agregat dikonversi |
| U-8 | Microcopy menegaskan zona IDR untuk biaya |
