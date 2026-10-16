# Penampil Jejak Audit (Audit Log Viewer) — FinTrack v2

Dokumen ini mendefinisikan antarmuka penampil jejak audit (`/super-admin/audit`), visualisasi perbandingan perubahan data (diff viewer), serta mekanisme kepatuhan terhadap UU Perlindungan Data Pribadi (UU PDP No. 27/2022).

---

## 1. Arsitektur Audit Trail & Karakteristik Data

Sesuai prinsip **K-29**, seluruh tindakan administratif di FinTrack v2 dicatat secara *append-only* di tabel `audit_logs`. Karakteristik kunci:
1. **Immutable (Kekal)**: Tidak ada endpoint backend yang mendukung mutasi `UPDATE` atau `DELETE` pada tabel audit log.
2. **Konteks Lengkap**: Setiap rekaman memuat identitas aktor, tipe aksi (`CREATE`, `UPDATE`, `DELETE`, `EXECUTE`), jenis entitas target, snapshot status sebelum perubahan (`before_state`), snapshot status sesudah perubahan (`after_state`), alamat IP asli, serta string User-Agent browser.

---

## 2. Antarmuka Penampil & Tabel Audit Log

Halaman `/super-admin/audit` menggunakan komponen terstandarisasi `<DataTable>` dengan konfigurasi kolom:

| Kolom | Field Data | Representasi Visual |
|---|---|---|
| **Waktu** | `created_at` | Format lokal `DD MMM YYYY, HH:mm:ss` (WIB) |
| **Aktor** | `actor_email` | Avatar inisial + email admin + badge peran |
| **Aksi** | `action` | Badge berkode warna (`CREATE` Hijau, `UPDATE` Biru, `DELETE` Merah, `EXECUTE` Ungu) |
| **Entitas** | `entity_type` & `entity_id` | Teks monospace pendek (contoh: `plan:prod_premium_30d`) |
| **Alamat IP** | `ip_address` | Monospace IP dengan bendera negara/lokasi bila tersedia |
| **Inspeksi** | - | Tombol `Lihat Diff Perubahan` (Membuka dialog visual diff) |

### Facet Filters yang Tersedia:
- **Rentang Tanggal**: DateRangePicker (Hari ini, 7 hari, 30 hari, kustom).
- **Entitas Target**: Dropdown pilihan (`plan`, `subscription`, `payment`, `token`, `ai_route`, `settings`).
- **Tipe Aksi**: Multiselect checkbox (`CREATE`, `UPDATE`, `DELETE`, `EXECUTE`).
- **Pencarian Aktor**: Input teks debounce pencarian email admin.

---

## 3. Komponen Inspeksi Diff Visual JSON

Ketika tombol *Lihat Diff Perubahan* ditekan, sebuah dialog modal berukuran besar (`max-w-4xl`) ditampilkan:
1. **Tampilan Side-by-Side**: Kolom kiri menampilkan `Sebelum (Before)` dan kolom kanan menampilkan `Sesudah (After)`.
2. **Penyorotan Warna Baris**:
   - Baris properti yang ditambahkan disorot warna hijau muda (`bg-emerald-50 text-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300`).
   - Baris properti yang dihapus disorot warna merah muda (`bg-rose-50 text-rose-900 dark:bg-rose-950/40 dark:text-rose-300`).
   - Nilai yang dimodifikasi disorot warna kuning/oranye.
3. **Penyalinan JSON**: Tombol cepat untuk menyalin JSON mentah untuk keperluan audit eksternal.

---

## 4. Kepatuhan UU PDP No. 27/2022 & Sanitasi Data

Sesuai **K-30**, FinTrack v2 mematuhi regulasi perlindungan data pribadi Indonesia:
1. **Penyamaran Data Sensitif (Masking)**:
   - Properti yang memuat Data Pribadi Tertentu atau rahasia otentikasi disamarkan di layer backend sebelum dikirimkan ke frontend:
     - Nomor Kartu / Akun Bank: `4111-XXXX-XXXX-1234`
     - Nomor HP / Identitas: `+62 812-XXXX-890`
     - Secret Key / API Token: `sk-9r-********************`
     - Password Hash: Selalu dikecualikan secara absolut dari pencatatan log (`[REDACTED]`).
2. **Hak Koreksi & Audit Sengketa**:
   - Log perubahan menjadi bukti non-repudiasi resmi ketika terjadi sengketa tagihan antara pengguna dan pengelola platform.
3. **Kebijakan Retensi Data**:
   - Seluruh baris audit log disimpan aktif selama 2 tahun kalender, kemudian diarsipkan ke penyimpanan dingin terenkripsi (*cold storage*) sesuai aturan retensi data finansial.
