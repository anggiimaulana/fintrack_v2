# 05 — Halaman Pendukung Publik

Dokumen ini merinci `/about`, `/contact`, `/faq`, `/privacy`, dan `/terms`: konten, struktur, metadata, dan schema. Kelima halaman ini adalah fondasi sinyal **Trustworthiness** dan **Authoritativeness** (E-E-A-T) sekaligus permukaan penangkapan intent informasional.

> **Gaya & nada (mengikuti dokumen 03):** visual neobrutalism (border 3px, hard shadow 6px, sudut membulat, aksen kuning asam/merah). Nada **kalem & profesional** dan sangat relevan di sini: kelima halaman ini adalah permukaan kepercayaan, sehingga setiap klaim harus faktual, tenang, dan tanpa hiperbola. `lang="id"` (konten Indonesia) meski URL berbahasa Inggris.

---

## 1. `/about` — Tentang FinTrack

### 1.1 Tujuan

Membangun kepercayaan: siapa yang membuat produk, apa prinsipnya, dan bagaimana data diperlakukan.

### 1.2 Struktur

| # | Blok | Isi |
|---|------|-----|
| 1 | Hero | H1 "Tentang FinTrack". Satu paragraf misi. |
| 2 | Misi | "Membuat pencatatan keuangan semudah mengobrol — tanpa mengorbankan privasi dan kejelasan." |
| 3 | Cara kerja teknis (ringkas, jujur) | Arsitektur tingkat tinggi: aplikasi web + AI Gateway terpusat + basis data PostgreSQL; tanpa klaim berlebihan. |
| 4 | Prinsip produk | 4 poin: Akurasi, Privasi (isolasi per akun), Transparansi kuota, Kendali data (ekspor/hapus). |
| 5 | Untuk siapa | Tautan ke section persona di landing (`/#who-its-for`). |
| 6 | Kontak | Ringkas + tautan ke `/contact`. |
| 7 | CTA | `Mulai Gratis`. |

> **Aturan kejujuran:** jangan mengarang sejarah perusahaan, jumlah tim, atau pendanaan. Bila informasi korporat belum final, gunakan pernyataan yang benar-benar berlaku (mis. "FinTrack dikembangkan oleh tim produk perangkat lunak keuangan.") tanpa detail fiktif. Bagian "Tim" hanya diisi bila ada data nyata.

### 1.3 Metadata & Schema

- `title`: "Tentang FinTrack — Misi & Prinsip"
- `description`: "Misi FinTrack: pencatatan keuangan semudah mengobrol, dengan privasi dan transparansi kuota."
- `canonical`: `/about`
- JSON-LD: `AboutPage` + `Organization` (nama, url, logo, kontak) + `BreadcrumbList`.

---

## 2. `/contact` — Kontak

### 2.1 Tujuan

Menyediakan kanal dukungan yang nyata dan mudah dipakai. **Tanpa backend** (keputusan #4).

### 2.2 Struktur

| # | Blok | Isi |
|---|------|-----|
| 1 | Hero | H1 "Hubungi Kami". Satu paragraf: kapan sebaiknya menghubungi. |
| 2 | Kartu kanal | Email (`mailto:`) dan WhatsApp (`wa.me`) dengan tombol jelas. |
| 3 | Jam operasional | Hari & jam respons (pernyataan realistis; sesuaikan dengan praktik nyata). |
| 4 | Sebelum menghubungi | Tautan cepat: FAQ, Harga, Fitur (mengurangi pertanyaan berulang). |
| 5 | Untuk urusan akun | "Soal langganan/pembayaran/penghapusan data, sebutkan email akun Anda saat menghubungi." |
| 6 | CTA | `Mulai Gratis`. |

### 2.3 Alamat kanal

Diambil dari `lib/marketing/site.ts` (satu sumber, agar tidak ada email berbeda antar halaman):

```ts
export const site = {
  name: "FinTrack",
  url: process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3001",
  email: "halo@fintrack.id",      // placeholder — konfirmasi saat implementasi
  whatsapp: "+62...",              // placeholder — konfirmasi saat implementasi
  social: { /* diisi bila ada */ },
};
```

> **Catatan implementasi:** nilai email/WhatsApp di atas adalah placeholder dan **harus** dikonfirmasi pemilik produk sebelum rilis. Jangan menggunakan alamat yang tidak aktif.

### 2.4 Aksesibilitas kanal

- Tautan `mailto:` diberi `aria-label` deskriptif ("Kirim email ke tim FinTrack").
- Tautan WhatsApp `aria-label` ("Chat WhatsApp dengan tim FinTrack") dan `rel="noopener noreferrer"`.

### 2.5 Metadata & Schema

- `title`: "Kontak FinTrack — Bantuan & Pertanyaan"
- `description`: "Hubungi tim FinTrack lewat email atau WhatsApp. Lihat juga FAQ, harga, dan panduan fitur."
- `canonical`: `/contact`
- JSON-LD: `ContactPage` + `Organization.contactPoint` (email, dan jenis kontak) + `BreadcrumbList`.

---

## 3. `/faq` — Tanya Jawab

### 3.1 Tujuan

Menjawab keberatan dan pertanyaan berulang secara tuntas; sekaligus permukaan utama untuk **GEO/AI citation** (jawaban ringkas dan langsung).

### 3.2 Struktur & Kelompok

Empat kelompok, tiap jawaban diawali **jawaban langsung** (answer-first) baru detail:

| Kelompok | Contoh pertanyaan |
|----------|-------------------|
| Umum | Apakah FinTrack gratis? Siapa yang cocok memakainya? Apakah berbahasa Indonesia? |
| Akun & Langganan | Bagaimana cara upgrade/downgrade? Bisakah berhenti langganan? Apa yang terjadi pada akhir periode? |
| AI & Privasi | Apakah perlu API key? Bagaimana data saya diperlakukan? Bisakah saya ekspor/hapus data? Siapa yang bisa melihat data saya? |
| Pembayaran | Metode apa saja? Apakah ada pajak? Bagaimana verifikasi transfer manual? |

### 3.3 Prinsip penulisan jawaban

- **Answer-first:** kalimat pertama menjawab langsung; 1–2 baris. Contoh: "Ya. Paket Free bisa dipakai tanpa batas waktu untuk mulai mencatat." 
- Akurasi: rujuk perilaku nyata (downgrade berlaku di akhir periode, verifikasi manual oleh admin).
- Tautan kontekstual ke `/pricing`, `/features/*`, `/privacy`, `/contact`.
- Panjang jawaban 40–120 kata.

### 3.4 Implementasi

- `FaqAccordion` memakai elemen native `<details>`/`<summary>` (dapat diakses, berfungsi tanpa JS) — boleh ada versi client untuk animasi, tetapi konten harus ada di HTML server.
- Data dari `lib/marketing/faq.ts` (`{ category, items: [{q, a, links?}] }`).

### 3.5 Metadata & Schema

- `title`: "Pertanyaan Umum (FAQ) — FinTrack"
- `description`: "Jawaban atas pertanyaan umum soal FinTrack: harga, langganan, AI, privasi data, dan pembayaran."
- `canonical`: `/faq`
- JSON-LD: `BreadcrumbList` + `WebPage`.
- **Tidak memakai `FAQPage`.** Google telah menghentikan tampilan FAQ rich result; menandai `FAQPage` tidak lagi memberi manfaat tampilan dan berisiko menjadi markup yang tidak sesuai. Konten Q&A tetap dirender sebagai teks untuk dikutip mesin AI.

---

## 4. `/privacy` — Kebijakan Privasi

### 4.1 Tujuan

Kepatuhan UU PDP dan bukti Trustworthiness. Harus **spesifik pada produk nyata**, bukan template generik.

### 4.2 Bagian wajib

| # | Bagian | Isi pokok |
|---|--------|-----------|
| 1 | Ringkasan | Poin utama dalam bahasa sederhana (data apa, untuk apa, kendali apa). |
| 2 | Data yang kami kumpulkan | Akun (nama, email, telepon opsional, avatar opsional); data keuangan yang Anda masukkan; data teknis (log, perangkat); data langganan/pembayaran. |
| 3 | Tujuan pemrosesan | Menyediakan layanan, memproses AI, penagihan, keamanan, dukungan. |
| 4 | Pemrosesan AI | Transaksi diproses lewat gateway AI terpusat; sebutkan bahwa tidak ada API key pribadi yang disimpan (konsisten dengan koreksi F-04). |
| 5 | Berbagi data | Hanya ke penyedia yang diperlukan (pemroses AI, gateway pembayaran) sesuai kontrak; tidak menjual data. |
| 6 | Penyimpanan & retensi | Lama penyimpanan + penghapusan/anonimisasi (rujuk anonimisasi PII saat penghapusan akun — `CLAUDE.md`). |
| 7 | Hak Anda | Akses, ekspor, koreksi, hapus akun, keberatan. Sebutkan cara nyata: ekspor (`/user/profile`) dan hapus (`/user/profile`). |
| 8 | Keamanan | Kontrol akses berbasis peran, isolasi data per akun, enkripsi kredensial gateway. |
| 9 | Anak-anak | Pernyataan kelayakan usia. |
| 10 | Perubahan kebijakan | Cara memberitahukan perubahan + tanggal berlaku. |
| 11 | Kontak privasi | Tautan ke `/contact`. |

### 4.3 Aturan kejujuran

- Jangan mengklaim kepatuhan/sertifikasi yang tidak dimiliki.
- Rujuk fitur nyata (ekspor data, hapus akun) yang memang ada.
- Beri **tanggal berlaku**. Bila isi belum ditinjau ahli hukum, tandai sebagai draf internal di berkas sumber (`legal.ts`) agar tidak terbit tanpa tinjauan.

### 4.4 Metadata

- `title`: "Kebijakan Privasi — FinTrack"
- `description`: "Bagaimana FinTrack mengumpulkan, memakai, dan melindungi data Anda; serta hak akses, ekspor, dan penghapusan."
- `canonical`: `/privacy`; halaman **diizinkan diindeks** (penting untuk trust).

---

## 5. `/terms` — Syarat & Ketentuan

### 5.1 Bagian wajib

| # | Bagian | Isi pokok |
|---|--------|-----------|
| 1 | Penerimaan | Dengan memakai FinTrack Anda menyetujui syarat ini. |
| 2 | Akun | Kelayakan usia, keamanan kata sandi, tanggung jawab aktivitas akun. |
| 3 | Penggunaan yang diizinkan | Larangan penyalahgunaan, otomatisasi berlebihan, upaya bypass kuota. |
| 4 | Langganan & pembayaran | Siklus penagihan, perpanjangan, pajak, pembatalan (perilaku nyata: akses sampai akhir periode). |
| 5 | Kuota AI | Kuota per siklus/hari (rujuk label resmi), penanganan saat kuota habis. |
| 6 | Hak kekayaan intelektual | Kepemilikan perangkat lunak dan milik data Anda. |
| 7 | Batasan tanggung jawab | Sesuai praktik umum; tanpa jaminan berlebihan. |
| 8 | Penghentian | Kapan akun dapat ditangguhkan/ditutup. |
| 9 | Perubahan syarat | Cara pemberitahuan. |
| 10 | Hukum yang berlaku & kontak | Yurisdiksi Indonesia + tautan `/contact`. |

### 5.2 Metadata

- `title`: "Syarat & Ketentuan — FinTrack"
- `description`: "Ketentuan penggunaan layanan FinTrack: akun, langganan, pembayaran, kuota AI, dan tanggung jawab."
- `canonical`: `/terms`; diizinkan diindeks.

---

## 6. Penanganan Konten Legal (Implementasi)

- Konten disimpan sebagai struktur data di `lib/marketing/legal.ts` (array bagian `{ id, heading, paragraphs[], list? }`), dirender oleh komponen Server yang sama (`LegalPage`).
- Memberi **daftar isi otomatis** (anchor per bagian) di layar besar untuk keterbacaan.
- Setiap bagian memakai heading `H2` (dan `H3` bila bersarang) untuk struktur yang benar.
- Menyertakan tanggal "Berlaku sejak: {tanggal}" dari field `effectiveDate`.
- Karena konten legal panjang dan stabil, halaman dirender statis (`force-static`) dan tidak melakukan fetch apa pun.

## 7. Ringkasan Metadata Kelima Halaman

| Rute | title | Schema |
|------|-------|--------|
| `/about` | Tentang FinTrack — Misi & Prinsip | `AboutPage`, `Organization`, `BreadcrumbList` |
| `/contact` | Kontak FinTrack — Bantuan & Pertanyaan | `ContactPage`, `BreadcrumbList` |
| `/faq` | Pertanyaan Umum (FAQ) — FinTrack | `WebPage`, `BreadcrumbList` |
| `/privacy` | Kebijakan Privasi — FinTrack | `WebPage` |
| `/terms` | Syarat & Ketentuan — FinTrack | `WebPage` |

## 8. Kriteria Selesai Halaman Pendukung

- Tidak ada `href="#"` atau anchor mati.
- Tidak ada klaim yang tidak dapat dibuktikan.
- Semua tautan internal menunjuk rute yang benar-benar ada.
- Semua halaman dirender tanpa JS (konten ada di HTML) dan lulus `axe-core`.
- Contact & legal memakai satu sumber data (`site.ts`, `legal.ts`) tanpa duplikasi teks.
