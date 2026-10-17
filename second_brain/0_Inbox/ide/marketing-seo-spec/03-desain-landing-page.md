# 03 — Desain Landing Page

Dokumen ini merinci susunan ulang landing page (`/`), copy final, strategi kata kunci, alur konversi, koreksi klaim, dan penanganan aksesibilitas.

---

## 1. Tujuan Halaman

Landing harus memenuhi tiga tujuan sekaligus, berurutan:

1. **Dipahami** — pengunjung mengerti dalam < 10 detik: ini aplikasi pencatat keuangan berbasis AI berbahasa Indonesia.
2. **Dipercaya** — klaim akurat, bukti nyata (fitur ekspor/hapus data), tanpa testimoni palsu.
3. **Bertindak** — mendorong **Mulai Gratis** (`/register`) sebagai aksi utama, dengan `/harga` sebagai jalur pertimbangan.

## 2. Sasaran Kata Kunci

> Angka volume bukan estimasi pasti; ini heuristik intent. Validasi dengan Search Console setelah rilis.

| Peran | Kata kunci | Penempatan |
|-------|-----------|------------|
| Utama | aplikasi pencatat keuangan, catat keuangan otomatis | H1, title, first 100 kata |
| Sekunder | catat pengeluaran via chat, scan struk otomatis, input transaksi suara | H2 fitur, deskripsi section |
| Sekunder | analisis keuangan pribadi, laporan keuangan otomatis | H2 SPK, H2 laporan |
| Long-tail | cara mencatat pengeluaran harian, aplikasi keuangan tanpa ribet | FAQ, sub-halaman fitur |
| GEO/AI | "apa itu FinTrack", "apakah FinTrack gratis", "bagaimana cara mencatat struk" | FAQ, definisi ringkas |
| Brand | FinTrack | seluruh halaman |

Aturan: keyword utama muncul di `title`, `H1`, dan paragraf pertama. Kepadatan alami (bukan stuffing); variasi semantik disarankan.

## 3. Susunan Section (Final)

| # | Section | Komponen | Anchor |
|---|---------|----------|--------|
| 1 | Header | `MarketingHeader` | — |
| 2 | Hero | `Hero` | — |
| 3 | Trust bar | `TrustBar` | — |
| 4 | Masalah → Solusi | `SectionHeading` + grid | `#solusi` |
| 5 | Fitur unggulan | `FeatureGrid` | `#fitur` |
| 6 | Cara kerja | `HowItWorks` | `#cara-kerja` |
| 7 | Highlight SPK | `SpkHighlight` | `#analisis` |
| 8 | Teaser harga | `PricingTeaser` | `#harga` |
| 9 | Cocok untuk siapa | `PersonaUseCases` | `#untuk-siapa` |
| 10 | FAQ | `FaqAccordion` | `#faq` |
| 11 | CTA penutup | `CtaSection` | — |
| 12 | Footer | `MarketingFooter` | — |

> Section `#harga` sekarang **nyata** (menutup F-01). Anchor `#cara-kerja` menjadi target tombol "Lihat Cara Kerja" (menutup F-03).

### 3.1. Hero

- **Eyebrow/badge:** "Asisten Keuangan Berbahasa Indonesia"
- **H1:** `Catat Keuangan Otomatis dengan AI — Cukup Diketik, Diucapkan, atau Difoto`
- **Subhead:** "FinTrack memahami perintah bahasa Indonesia. Tulis "beli kopi 25 ribu", ucapkan, atau foto struk belanja — transaksi langsung tercatat rapi dan bisa Anda tanyakan kapan saja."
- **CTA primer:** `Mulai Gratis` → `/register`
- **CTA sekunder:** `Lihat Cara Kerja` → `#cara-kerja`
- **Trust chips:** `Gratis untuk memulai` · `Tanpa kartu kredit` · `Bahasa Indonesia`
- **Visual:** `ProductMockup` — kartu ringkasan pemasukan/pengeluaran + gelembung chat (dipertahankan dari lama, copy percakapan diperbarui; harga "25rb" dan konfirmasi "Rp 25.000" akurat).

### 3.2. Trust Bar (menggantikan klaim "ribuan pengguna")

Deretan 4 poin **faktual** yang semuanya benar-benar ada di produk:

| Poin | Kalimat | Basis kebenaran |
|------|---------|-----------------|
| Isolasi data | "Data tiap akun terpisah dan hanya Anda yang mengaksesnya." | RBAC per-user, tanpa RLS lintas-akun |
| Kendali data | "Ekspor seluruh data Anda kapan saja." | `GET /api/v1/user/export-data` |
| Hapus kapan saja | "Hapus akun beserta datanya dari pengaturan." | `DELETE /api/v1/me` |
| Transparan | "Kuota dan pemakaian AI terlihat jelas di dasbor." | Halaman `/user/usage` & `/user/subscription` |

### 3.3. Masalah → Solusi

- **H2:** "Mencatat keuangan tidak seharusnya merepotkan"
- Dua kolom:

| Tanpa FinTrack | Dengan FinTrack |
|----------------|-----------------|
| Buka spreadsheet, isi manual kolom demi kolom | Tulis "makan siang 40rb" → langsung tercatat |
| Lupa mencatat karena ribet | Catat lewat chat, suara, atau foto struk |
| Data menumpuk tapi tak pernah dianalisis | Tanyakan ke AI; dapat skor & status keuangan |

### 3.4. Fitur Unggulan (6 kartu) — copy dikoreksi

| Ikon | Judul | Deskripsi (final) | Tautan |
|------|-------|-------------------|--------|
| `MessageSquare` | Catat Lewat Chat | "Ketik transaksi seperti mengobrol. AI mengekstrak jumlah, kategori, dan tanggal otomatis." | `/fitur/chat-ai` |
| `Mic` | Catat Lewat Suara | "Sedang berkendara atau sibuk? Ucapkan pengeluaran Anda dan biarkan AI mencatatnya." | `/fitur/input-suara` |
| `Camera` | Foto Struk | "Unggah foto bon belanja; AI membaca item dan totalnya, lalu menyimpannya untuk Anda." | `/fitur/scan-struk` |
| `LayoutDashboard` | Dasbor Interaktif | "Lihat pemasukan, pengeluaran, dan tren dalam grafik yang mudah dibaca." | `/fitur/dashboard` |
| `Target` | Analisis Kesehatan Finansial | "Sistem pendukung keputusan menilai kondisi keuangan Anda dari Sangat Baik hingga Kritis." | `/fitur/analisis-spk` |
| `Download` | Laporan & Ekspor | "Hasilkan laporan dan unduh data Anda kapan pun — data Anda milik Anda." | `/fitur/laporan` |

> **F-04 ditutup:** tidak ada lagi klaim "AI key milik Anda sendiri". Nilai privasi disampaikan lewat fakta yang benar (isolasi data, ekspor, hapus).

### 3.5. Cara Kerja (3 langkah) — F-04 ditutup

| Langkah | Judul | Deskripsi |
|---------|-------|-----------|
| 1 | Daftar Gratis | "Buat akun dalam hitungan detik. Tidak perlu kartu kredit untuk memulai." |
| 2 | Catat Transaksi | "Lewat chat, suara, atau foto struk — pilih cara yang paling nyaman." |
| 3 | Tanya & Analisis | "Ajukan pertanyaan soal keuangan Anda; AI menjawab berdasarkan data Anda sendiri." |

### 3.6. Highlight SPK

Dipertahankan dari versi lama (akurat). Salinan:

- **H2:** "Ketahui Kondisi Keuangan Anda"
- Deskripsi: "Sistem Pendukung Keputusan menilai pola pengeluaran, tabungan, dan kebiasaan Anda, lalu memberi skor dan status."
- Skala status: Sangat Baik · Baik · Cukup · Boros · Kritis (dengan indikator warna).
- Kartu contoh skor 85 "Sangat Baik" dengan penjelasan rasio tabungan.

> Semua label status diambil dari konfigurasi metrik SPK (`backend/internal/domain/spk_metrics.go`), bukan dikarang.

### 3.7. Teaser Harga

- **H2:** "Harga Sederhana, Tanpa Kejutan"
- Menerima data paket dari `lib/plans-server.ts` (dokumen 06) dan menampilkan 3 kartu ringkas: nama, harga bulanan, 3 poin fitur utama, tombol `Lihat Detail` → `/harga`.
- Menandai paket **Pro** sebagai "Paling Populer".
- **Fallback:** bila data paket gagal dimuat, section menyembunyikan kartu dan menampilkan satu tombol `Lihat Harga` → `/harga` (halaman `/harga` punya penanganan fallback sendiri).

### 3.8. Cocok untuk Siapa (pengganti testimoni — F-05 ditutup)

Empat kartu persona **tanpa nama fiktif dan tanpa kutipan palsu**:

| Persona | Kalimat |
|---------|---------|
| Freelancer | "Pendapatan tidak tetap? Lihat arus kas per proyek dan pastikan sisanya cukup." |
| Pekerja kantoran | "Catat gaji dan pengeluaran bulanan tanpa membuka spreadsheet." |
| Ibu rumah tangga | "Foto struk belanja bulanan; AI yang merapikan catatannya." |
| Pelaku UMKM | "Pantau pemasukan dan pengeluaran usaha kecil dalam satu tempat." |

### 3.9. FAQ (6 item)

Ringkas di landing, versi lengkap di `/faq`:

1. Apakah FinTrack gratis? — "Ada paket Free untuk mulai. Paket berbayar menambah kuota AI dan fitur." → `/harga`
2. Bagaimana AI memahami perintah saya? — "AI memproses teks, suara, dan gambar, lalu menyusun transaksi terstruktur." → `/fitur/chat-ai`
3. Apakah data saya aman? — isolasi per akun, ekspor, hapus. → `/privasi`
4. Apakah butuh API key sendiri? — **"Tidak."** (menegaskan koreksi F-04)
5. Apakah bisa ekspor data? — ya, sebutkan endpoint/fitur. → `/fitur/laporan`
6. Bagaimana cara berlangganan/membatalkan? — jelaskan singkat → `/harga`

### 3.10. CTA Penutup (F-05 ditutup)

- **H2:** "Mulai Rapikan Keuangan Anda Hari Ini"
- **Subhead:** "Gratis untuk memulai. Tanpa kartu kredit. Data Anda dapat diekspor atau dihapus kapan saja." (menggantikan "ribuan pengguna lain")
- **CTA:** `Daftar Gratis` → `/register`

---

## 4. Alur Konversi

```text
Masuk (organik/ads)
  → Hero: paham dalam 10 detik → [Mulai Gratis] atau [Lihat Cara Kerja]
  → Fitur unggulan: cocokkan kebutuhan → klik sub-fitur (bila ragu)
  → Cara kerja: turunkan keraguan "sulit dipakai"
  → SPK + teaser harga: tunjukkan nilai → [Lihat Detail] ke /harga
  → Persona: "ini untuk saya"
  → FAQ: jawab keberatan (harga, privasi, API key)
  → CTA penutup → /register
```

Prinsip: **satu aksi utama** (daftar) dengan banyak pintu masuk menuju aksi itu, tanpa memaksa berpindah halaman di awal.

## 5. Hirarki Heading

- Satu `H1` saja (hero).
- `H2` untuk tiap section.
- `H3` untuk kartu/judul item dalam section.
- Tidak melompat level (tidak `H1` → `H3`).

## 6. Aksesibilitas (F-08 ditutup)

| Area | Persyaratan |
|------|-------------|
| Menu mobile | Tombol `<button>` ber-`aria-expanded`, `aria-controls`, `aria-label`; panel ber-`id`; `Escape` menutup; fokus kembali ke tombol; `aria-hidden` saat tertutup. |
| Ikon | Semua ikon dekoratif `aria-hidden="true"`; ikon bermakna punya teks atau `aria-label`. |
| Kontras | Teks di atas `surface-dark` memakai `text-neutral-300`/`white` ≥ 4.5:1; jangan pakai `text-neutral-500` untuk teks penting. |
| Fokus | Semua tautan/tombol punya cincin fokus terlihat (jangan `outline: none` tanpa pengganti). |
| Akordeon FAQ | Memakai `<details>`/`<summary>` native sehingga tanpa JS tetap berfungsi dan dapat diakses keyboard. |
| Tautan lompat | Sediakan "Lewati ke konten" (`#main`) sesuai praktik baik; opsional bila navigasi ringkas. |
| Bahasa | `lang="id"` sudah diset di root layout. |

## 7. Performa

- Landing seluruhnya Server Component kecuali `MarketingHeader` dan `FaqAccordion` (fungsional, kecil).
- Tanpa gambar raster besar; mockup dibuat CSS/HTML.
- Tanpa library animasi; gunakan transisi CSS singkat (`transition-all duration-100`) sesuai gaya yang ada.
- Target: rasio JS rendah dan LCP memadai (lihat dokumen 07 §Core Web Vitals).

## 8. Konsistensi Desain

Ikuti bahasa visual yang ada:

- Kartu: `border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground`.
- Tombol: pakai `components/ui/button.tsx` (varian `default`, `outline`).
- Warna dari token `globals.css` (`primary`, `muted`, `surface-dark`, `primary-strong` untuk teks di atas `primary/10`).
- Sudut `rounded-md`/`rounded-lg`, bukan pil.
