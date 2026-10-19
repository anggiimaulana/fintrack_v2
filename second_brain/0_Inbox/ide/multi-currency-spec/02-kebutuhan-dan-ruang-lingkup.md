# 02 — Kebutuhan dan Ruang Lingkup

Dokumen ini menurunkan permintaan menjadi kebutuhan yang dapat diuji. Setiap kebutuhan punya ID agar bisa dipakai di implementation plan, test matrix, dan PR checklist.

---

## 1. Persona

| Persona | Kebutuhan utama |
|---|---|
| **Pengguna Indonesia** | Tetap memakai IDR; tidak ada perubahan perilaku |
| **Pengguna Jepang / pekerja di Jepang** | Mencatat pemasukan/pengeluaran dalam JPY, melihat dashboard dan laporan dalam JPY |
| **Pengguna multi-negara (masa depan)** | Menambah USD/MYR tanpa deploy; mengganti mata uang tampilan |
| **Super Admin** | Mengelola daftar mata uang, kurs, dan riwayat kurs; melihat audit |
| **Admin** | Melihat data pengguna; tidak boleh mengubah kurs |
| **Tim Finance/Support** | Memahami nominal asli vs nilai setara; bisa menjelaskan selisih kurs |

---

## 2. Kebutuhan Fungsional

### 2.1 Katalog mata uang

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-CUR-01 | Sistem menyimpan daftar mata uang di database: kode ISO 4217, nama, simbol, locale, jumlah desimal, status aktif, urutan, penanda base. | Must |
| FR-CUR-02 | Super Admin dapat menambah mata uang baru tanpa deploy. | Must |
| FR-CUR-03 | Super Admin dapat mengubah metadata mata uang (nama, simbol, locale, desimal, urutan, aktif). | Must |
| FR-CUR-04 | Kode mata uang unik, huruf besar, 3 karakter. | Must |
| FR-CUR-05 | Hanya satu base currency pada satu waktu; base currency tidak dapat dihapus. | Must |
| FR-CUR-06 | Mata uang nonaktif tidak muncul di pilihan pengguna, tetapi data historis tetap terbaca. | Must |
| FR-CUR-07 | Pengguna hanya melihat mata uang aktif. | Must |

### 2.2 Kurs

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-RATE-01 | Super Admin dapat menambahkan kurs baru untuk suatu mata uang. | Must |
| FR-RATE-02 | Setiap kurs memiliki `effective_at` dan nilai `rate_to_base`. | Must |
| FR-RATE-03 | Riwayat kurs bersifat append-only; tidak ada edit/delete. Koreksi = baris baru + catatan. | Must |
| FR-RATE-04 | Sistem mengambil kurs terbaru dengan `effective_at <= tanggal acuan`. | Must |
| FR-RATE-05 | Jika belum ada kurs yang berlaku, sistem menolak transaksi dengan pesan jelas, bukan memakai angka default. | Must |
| FR-RATE-06 | Super Admin dapat melihat riwayat kurs per mata uang. | Must |
| FR-RATE-07 | Perubahan kurs langsung berlaku untuk transaksi baru tanpa restart. | Must |
| FR-RATE-08 | Kurs tidak boleh 0, negatif, atau di luar batas kewajaran yang dikonfigurasi. | Must |
| FR-RATE-09 | Kurs memiliki `source` (`manual`, `migration`, `api` di masa depan) dan `note`. | Should |
| FR-RATE-10 | Base currency memiliki rate = 1 secara implisit; tidak perlu baris kurs. | Must |

### 2.3 Preferensi pengguna

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-PREF-01 | Pengguna dapat memilih mata uang pencatatan di **Profil → Tampilan & Bahasa**. | Must |
| FR-PREF-02 | Pilihan disimpan di database dan dikembalikan oleh `GET /me`. | Must |
| FR-PREF-03 | Default pengguna baru dan pengguna lama adalah IDR. | Must |
| FR-PREF-04 | Perubahan preferensi berlaku tanpa reload halaman. | Should |
| FR-PREF-05 | Preferensi ikut dalam ekspor data pengguna (UU PDP). | Must |
| FR-PREF-06 | Preferensi tidak mengubah role, akses, atau entitlement. | Must |
| FR-PREF-07 | Pengguna tidak dapat memilih mata uang nonaktif. | Must |
| FR-PREF-08 | Jika mata uang preferensi dinonaktifkan, pengguna tetap bisa melihat data lama; sistem memberi opsi pindah ke IDR. | Should |

### 2.4 Pencatatan transaksi

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-TX-01 | Form transaksi menampilkan simbol dan desimal sesuai mata uang aktif. | Must |
| FR-TX-02 | Nominal yang dikirim ke backend adalah nominal asli dalam mata uang aktif. | Must |
| FR-TX-03 | Backend menyimpan `amount`, `currency_code`, `exchange_rate`, `amount_base`, `rate_at`. | Must |
| FR-TX-04 | `amount_base = round(amount × exchange_rate, 2)`. | Must |
| FR-TX-05 | Kurs yang dipakai adalah kurs pada `transaction_date` (bukan kurs hari ini jika tanggal di masa lalu). | Must |
| FR-TX-06 | Jika kurs untuk tanggal transaksi belum ada, transaksi ditolak dengan pesan yang menyebut tanggal dan mata uang. | Must |
| FR-TX-07 | Edit nominal memakai ulang kurs pada tanggal transaksi (bukan kurs terbaru), kecuali mata uang diubah. | Must |
| FR-TX-08 | Mengubah tanggal transaksi menghitung ulang `exchange_rate` dan `amount_base` berdasarkan tanggal baru. | Must |
| FR-TX-09 | Daftar transaksi menampilkan nominal asli dan kode mata uang; detail dapat menampilkan nilai setara IDR. | Must |
| FR-TX-10 | Transaksi lama tetap IDR dan tidak berubah setelah migrasi. | Must |
| FR-TX-11 | Nominal asli tidak boleh 0/negatif; `amount_base` tidak boleh 0/negatif. | Must |
| FR-TX-12 | Transaksi dengan mata uang nonaktif tetap dapat dibaca dan diedit tanpa mengganti mata uang. | Should |
| FR-TX-13 | Pengguna dapat melihat total dalam mata uang aktif dan nilai dasar IDR (opsional/tooltip). | Should |

### 2.5 Dashboard, laporan, dan analitik

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-AGG-01 | Semua agregasi keuangan dihitung dari `amount_base`. | Must |
| FR-AGG-02 | Dashboard menampilkan nilai dalam display currency. | Must |
| FR-AGG-03 | Statistik transaksi (pemasukan, pengeluaran, tabungan, saldo) dalam display currency. | Must |
| FR-AGG-04 | Laporan (`GET /reports`) mengembalikan nilai base dan/atau nilai display yang konsisten. | Must |
| FR-AGG-05 | Laporan historis memakai kurs pada akhir periode laporan. | Must |
| FR-AGG-06 | Analitik laporan (comparison, top movers, heatmap, anomaly, ratios, best/worst) konsisten dengan laporan utama. | Must |
| FR-AGG-07 | Chart menampilkan sumbu/tooltip dalam display currency. | Must |
| FR-AGG-08 | Perubahan kurs tidak boleh membuat total transaksi lama berubah pada periode yang sudah lewat (selama kurs historis tersedia). | Must |
| FR-AGG-09 | Rasio (savings rate, expense ratio, runway) tidak terpengaruh mata uang karena berbasis rasio. | Must |
| FR-AGG-10 | Label UI menjelaskan bahwa agregat adalah “nilai setara” hasil konversi. | Should |

### 2.6 Budget

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-BUD-01 | Pengguna dapat mengisi budget dalam display currency. | Must |
| FR-BUD-02 | Budget disimpan sebagai base IDR (`amount_idr`) dengan snapshot kurs. | Must |
| FR-BUD-03 | Status budget (actual, remaining, used_pct, over) dihitung dari `amount_base` transaksi. | Must |
| FR-BUD-04 | Tampilan budget memakai display currency. | Must |
| FR-BUD-05 | Perubahan kurs tidak boleh mengubah plafon base yang sudah disimpan. | Must |

### 2.7 AI, OCR, voice

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-AI-01 | Prompt AI menyebut mata uang aktif dan kurs yang berlaku. | Must |
| FR-AI-02 | Tool `create_transaction` menerima nominal dalam display currency. | Must |
| FR-AI-03 | Backend mengonversi nominal tool ke base dan menyimpan snapshot kurs. | Must |
| FR-AI-04 | Konteks statistik AI (bulan ini, tahunan, kategori, transaksi terbaru) dikonversi ke display currency. | Must |
| FR-AI-05 | Konfirmasi pencatatan AI menampilkan simbol/format display currency. | Must |
| FR-AI-06 | OCR struk mengembalikan nominal dan mata uang; minimal memakai display currency sebagai default. | Should |
| FR-AI-07 | Voice transcription yang menghasilkan transaksi mengikuti aturan yang sama. | Must |
| FR-AI-08 | Embedding transaksi memuat kode mata uang dan nominal asli (bukan selalu “Rp”). | Must |
| FR-AI-09 | Billing/kuota AI tetap IDR dan tidak terpengaruh display currency. | Must |

### 2.8 Ekspor dan data pengguna

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-EXP-01 | Ekspor CSV/XLSX/PDF mengikuti display currency pengguna. | Must |
| FR-EXP-02 | Ekspor menyertakan kode mata uang, kurs yang dipakai, dan tanggal kurs. | Must |
| FR-EXP-03 | Ekspor data pribadi (`/user/export-data`) menyertakan preferensi mata uang. | Must |
| FR-EXP-04 | Nama file ekspor tidak berubah formatnya. | Should |
| FR-EXP-05 | PDF/CSV tetap bisa dibuka di locale Indonesia. | Must |

### 2.9 Admin dan audit

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-ADM-01 | Hanya Super Admin yang dapat menulis katalog/kurs. | Must |
| FR-ADM-02 | Admin biasa dapat membaca katalog/kurs (read-only) jika dibutuhkan untuk support. | Should |
| FR-ADM-03 | Setiap create/update/deactivate mata uang dan create rate masuk audit log. | Must |
| FR-ADM-04 | Audit mencatat before/after, aktor, waktu, dan alasan/keterangan. | Must |
| FR-ADM-05 | Perubahan katalog/kurs memicu `config_changed` dan invalidasi cache. | Must |
| FR-ADM-06 | Super Admin dapat melihat preview konversi sebelum menyimpan kurs. | Should |
| FR-ADM-07 | Super Admin mendapat peringatan jika menonaktifkan mata uang yang sedang dipakai pengguna/transaksi. | Should |

### 2.10 Kompatibilitas dan fallback

| ID | Kebutuhan | Prioritas |
|---|---|---|
| FR-COMP-01 | Kode lama tetap berjalan selama rolling deploy (expand/contract). | Must |
| FR-COMP-02 | Transaksi lama dianggap IDR dengan rate 1. | Must |
| FR-COMP-03 | Jika service kurs gagal, halaman keuangan menampilkan fallback IDR dan pesan, bukan angka kosong. | Must |
| FR-COMP-04 | Jika kurs hilang untuk transaksi baru, form menampilkan error yang dapat ditindaklanjuti. | Must |
| FR-COMP-05 | Feature flag `multi_currency_enabled` dapat mematikan fitur tanpa rollback schema. | Must |

---

## 3. Kebutuhan Non-Fungsional

| ID | Kebutuhan | Target |
|---|---|---|
| NFR-01 | Akurasi konversi | Base dibulatkan 2 desimal; selisih agregat 0 terhadap penjumlahan base |
| NFR-02 | Latensi | Tambahan query kurs ≤ 5 ms p95 pada create transaction |
| NFR-03 | Konsistensi | Tidak ada double conversion antara backend dan frontend |
| NFR-04 | Ketersediaan | Kegagalan rate service tidak mematikan dashboard; fallback IDR |
| NFR-05 | Keamanan | Hanya Super Admin dapat mengubah kurs; audit wajib |
| NFR-06 | Auditabilitas | Riwayat kurs tidak dapat dihapus |
| NFR-07 | Aksesibilitas | Select/input/dialog lolos axe; label dan error terbaca screen reader |
| NFR-08 | Kompatibilitas data | Backfill idempoten; tidak ada kehilangan nominal |
| NFR-09 | Rollback | Schema additive; feature flag bisa OFF tanpa deploy |
| NFR-10 | Observability | Metrik konversi, error rate, rate age, cache hit |
| NFR-11 | Internasionalisasi angka | Format mengikuti locale mata uang, bukan locale UI |
| NFR-12 | Privasi | Tidak ada data pribadi baru selain preferensi mata uang |
| NFR-13 | Performa frontend | Tidak ada request tambahan per komponen; satu query katalog |
| NFR-14 | Konsistensi cache | Perubahan kurs terlihat ≤ 5 detik di semua pod |

---

## 4. Acceptance Criteria Utama

### AC-01 — Pilih JPY

**Given** pengguna login dengan preferensi IDR  
**When** pengguna membuka Profil → Tampilan & Bahasa dan memilih JPY  
**Then**

- pilihan tersimpan di `users.display_currency`;
- setelah reload, pilihan tetap JPY;
- dashboard dan transaksi menampilkan `¥`;
- billing/kuota tetap `Rp`.

### AC-02 — Catat transaksi JPY

**Given** preferensi JPY dan kurs 1 JPY = 110 IDR  
**When** pengguna mencatat pengeluaran `1000`  
**Then**

- `amount=1000`, `currency_code='JPY'`, `exchange_rate=110`, `amount_base=110000`;
- daftar transaksi menampilkan `¥1.000`;
- total pengeluaran dashboard bertambah setara `¥1.000` (pada kurs yang sama).

### AC-03 — Kurs berubah

**Given** transaksi JPY lama tercatat pada kurs 110  
**When** Super Admin menambahkan kurs baru 1 JPY = 115 IDR  
**Then**

- transaksi lama tetap `amount=1000`, `exchange_rate=110`, `amount_base=110000`;
- transaksi baru memakai 115;
- laporan periode lama tetap memakai 110 (jika rate as of akhir periode = 110).

### AC-04 — Laporan historis

**Given** laporan bulan September 2026  
**When** kurs berubah pada Oktober 2026  
**Then** laporan September tetap memakai kurs yang berlaku pada 30 September 2026.

### AC-05 — Billing aman

**Given** preferensi JPY  
**When** pengguna membuka halaman langganan/pembayaran  
**Then** semua harga, invoice, dan nominal pembayaran tetap `Rp`.

### AC-06 — Super Admin menambah mata uang

**Given** Super Admin  
**When** menambahkan `USD` dengan simbol `$`, locale `en-US`, desimal 2, dan kurs  
**Then** USD muncul di pilihan pengguna tanpa deploy, dan dapat dipakai mencatat transaksi.

### AC-07 — Rate hilang

**Given** mata uang aktif tetapi belum punya kurs untuk tanggal transaksi  
**When** pengguna menyimpan transaksi  
**Then** backend menolak dengan pesan “Kurs USD untuk tanggal … belum tersedia” dan tidak menyimpan data setengah jadi.

### AC-08 — Audit

**Given** Super Admin mengubah kurs  
**When** perubahan disimpan  
**Then** audit log mencatat aktor, waktu, nilai lama/baru, dan catatan.

---

## 5. Asumsi

1. IDR tetap base currency dan tidak berubah pada rilis ini.
2. Pengguna memiliki satu preferensi display currency, bukan per transaksi.
3. Kurs ditentukan Super Admin; pengguna tidak dapat mengoverride.
4. Transaksi lama seluruhnya IDR.
5. Nilai `amount` pada transaksi lama tidak diubah.
6. Billing/AI/quota tidak memerlukan konversi.
7. Mata uang fiat ISO 4217; tidak ada kripto.
8. Locale UI tetap Indonesia; yang berubah hanya format angka.
9. Tidak ada kebutuhan multi-tenant per organisasi pada rilis ini.
10. Rate provider otomatis bukan bagian rilis ini.

---

## 6. Out of Scope (Eksplisit)

- Revaluasi aset/kewajiban.
- Akun bank multi-mata-uang.
- Transfer antar-mata-uang.
- Pajak multi-mata-uang.
- Invoice pelanggan dalam JPY.
- Pembayaran Midtrans dalam JPY.
- Kurs real-time dari BI/JIS/API eksternal.
- Historis kurs per menit/detik.
- Per-user exchange rate.
- Multi-base currency per pengguna.
- Kripto/emas/saham.
- Perubahan bahasa UI.
- Konversi metrik admin ke JPY.

---

## 7. Pertanyaan Terbuka

Pertanyaan yang belum dijawab di dokumen ini diringkas di `09-pengujian-risiko-dan-keputusan.md`. Yang paling menentukan:

1. Apakah pengguna boleh mengganti mata uang berkali-kali, atau dikunci sekali?
2. Apakah laporan harus bisa menampilkan total native per mata uang, bukan hanya nilai setara?
3. Apakah Super Admin perlu menjadwalkan kurs berlaku di masa depan?
4. Apakah JPY langsung aktif setelah migrasi, atau menunggu kurs manual?
5. Apakah admin non-super perlu akses baca ke halaman kurs?
