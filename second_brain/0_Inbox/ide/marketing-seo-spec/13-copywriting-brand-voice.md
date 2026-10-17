# 13 — Copywriting & Brand Voice

Dokumen ini adalah **sumber kebenaran untuk kata-kata**: nada bicara, aturan penulisan, bank judul, CTA, microcopy, dan naskah lengkap per halaman. Dokumen 03 menetapkan **struktur & tempat** elemen; dokumen ini menetapkan **bunyinya**.

> Kunci kombinasi brand: **visual berani** (neobrutalism) + **pesan tenang** (kalem & profesional). Visual menarik perhatian; kata-kata membangun kepercayaan dan menutup keraguan.

---

## 1. Prinsip Nada Bicara

1. **Tenang dan percaya diri.** Menjelaskan, bukan berteriak. Tanpa hiperbola, tanpa tanda seru beruntun.
2. **Jelas lebih dulu.** Satu gagasan per kalimat. Hindari istilah teknis bila ada padanan sederhana.
3. **Jujur.** Tidak ada klaim yang tidak dapat dibuktikan. Tidak ada angka pengguna yang tidak terverifikasi.
4. **Manfaat sebelum fitur.** Sebut akibat bagi pembaca, lalu mekanismenya.
5. **Hormat.** Sapa dengan "Anda", bukan "kamu". Tanpa sindiran, tanpa merendahkan pilihan orang.
6. **Konsisten.** Istilah produk selalu memakai label resmi (lihat §3).

## 2. Aturan Penulisan

**Lakukan:**
- Kalimat aktif dan pendek (rata-rata ≤ 18 kata).
- Angka konkret daripada kata samar ("kuota 150.000 token", bukan "kuota besar").
- Suara aktif untuk manfaat: "FinTrack mencatatnya untuk Anda."
- Judul section berupa pernyataan bermanfaat, bukan label kosong ("Harga sederhana, tanpa kejutan" alih-alih "Harga").

**Hindari:**
- Superlatif tanpa dasar: "terbaik", "nomor 1", "tercepat di Indonesia".
- Klaim sosial tak terverifikasi: "ribuan pengguna", "dipercaya jutaan orang".
- Rasa takut berlebihan: "tanpa FinTrack keuangan Anda hancur".
- Bahasa gaul berlebihan dan emoji pada copy produk.
- Janji fitur yang tidak ada (mis. "sinkron bank otomatis", "AI key sendiri").
- Tanda seru lebih dari satu per halaman.

**Ganti ini:**

| Hindari | Gunakan |
|---------|---------|
| "Aplikasi keuangan terbaik!" | "Aplikasi pencatat keuangan berbasis AI berbahasa Indonesia." |
| "Cuma perlu 1 detik!" | "Cukup satu kalimat untuk mencatat." |
| "Dijamin hemat!" | "Membantu Anda melihat ke mana uang pergi." |
| "Sinkron bank otomatis" | *(tidak boleh — fitur belum tersedia)* |
| "Pakai API key Anda sendiri" | *(tidak boleh — BYOK sudah dihapus)* |

## 3. Terminologi Resmi (Wajib Konsisten)

Label diambil dari katalog backend; copy **tidak boleh** mengarang nama lain.

**Nama fitur:** Asisten AI (chat) · Pindai struk (OCR) · Input suara · Pencarian cerdas (RAG) · Anggaran lanjutan · Ekspor laporan · Akses API.

**Nama limit:** Token AI / siklus · Permintaan AI / siklus · Transaksi · Pindai struk / siklus · Permintaan suara / hari.

**Nama status kesehatan finansial:** Sangat Baik · Baik · Cukup · Boros · Kritis.

Aturan: nilai `-1` selalu ditulis **"Tanpa batas"**; pajak sebagai **baris terpisah**; `bank_sync` dan limit yang tidak di-enforce (`max_categories`, `max_wallets`) **tidak pernah disebut**.

## 4. Nada per Konteks

| Konteks | Nada | Contoh |
|---------|------|--------|
| Hero & judul section | Percaya diri, ringkas | "Catat keuangan otomatis. Kelola dengan tenang." |
| Penjelasan fitur | Tenang, konkret | "Foto struk belanja; item dan totalnya dibaca lalu disimpan." |
| CTA | Tegas, rendah tekanan | "Mulai Gratis" · "Lihat Detail Paket" |
| Harga | Transparan, tanpa jebakan | "Harga sederhana, tanpa kejutan." |
| Legal & privasi | Formal, tepat, mudah dipahami | "Anda dapat mengekspor atau menghapus data Anda kapan saja." |
| Error / fallback | Menenangkan, memberi jalan keluar | "Harga sedang tidak dapat dimuat. Coba lagi sebentar lagi." |

## 5. Naskah Landing (kanonik)

Mengikuti urutan section dokumen 03 §B.3.

### 5.1 Hero
- **Eyebrow:** `ASISTEN KEUANGAN BERBAHASA INDONESIA`
- **H1:** **Catat keuangan otomatis. Kelola dengan tenang.** *(kata "tenang" diberi highlight kuning asam)*
- **Subhead:** "Tulis, ucapkan, atau foto transaksinya. FinTrack mencatat dengan rapi dan membantu Anda memahami kondisi keuangan — tanpa spreadsheet."
- **CTA utama:** `Mulai Gratis` · **CTA sekunder:** `Lihat Cara Kerja`
- **Trust chips:** `Gratis untuk memulai` · `Tanpa kartu kredit` · `Data dapat diekspor`

### 5.2 Receipt Tape & Ticker Band
- Caption struk: "Ilustrasi tampilan transaksi yang tercatat."
- Ticker: `CATAT · PAHAMI · KELOLA · TENANG ·`

### 5.3 Trust Bar & Stat
- "Data tiap akun terpisah dan hanya Anda yang mengaksesnya."
- "Ekspor seluruh data Anda kapan saja."
- "Hapus akun beserta datanya dari pengaturan."
- "Kuota dan pemakaian AI terlihat jelas di dasbor."
- Stat: `3` cara mencatat · `5` indikator status kesehatan keuangan · `Rp 0` biaya untuk memulai

### 5.4 Masalah → Solusi
- **H2:** "Mencatat keuangan tidak seharusnya menyita waktu"
- Kiri (masalah): "Membuka spreadsheet dan mengisi kolom satu per satu." / "Lupa mencatat karena repot." / "Data menumpuk tanpa pernah ditinjau."
- Kanan (solusi): "Menulis 'makan siang 40 ribu' — langsung tercatat." / "Mencatat lewat chat, suara, atau foto struk." / "Mengajukan pertanyaan; mendapat skor dan status keuangan."

### 5.5 Fitur
- **H2:** "Satu aplikasi untuk mencatat, memahami, dan mengelola"
- Tagline kartu (kalem): "Tulis transaksi seperti mengirim pesan." · "Catat dengan suara saat tangan sibuk." · "Foto struk; sisanya dirapikan otomatis." · "Pantau arus kas dalam satu tampilan." · "Ketahui skor dan status keuangan." · "Hasilkan laporan dan unduh data Anda."

### 5.6 Cara Kerja
- **H2:** "Mulai dalam tiga langkah"
- 1. "Daftar gratis — buat akun dalam hitungan detik, tanpa kartu kredit."
- 2. "Catat transaksi — pilih cara paling nyaman: chat, suara, atau foto struk."
- 3. "Tinjau dan analisis — lihat ringkasan, lalu ajukan pertanyaan tentang keuangan Anda."

### 5.7 Kesehatan Finansial
- **H2:** "Ketahui kondisi keuangan Anda"
- "Penilaian meninjau pola pengeluaran, tabungan, dan kebiasaan Anda, lalu memberi skor beserta status yang jelas."
- Label status ditulis lengkap, tidak hanya lewat warna.

### 5.8 Teaser Harga
- **H2:** "Harga sederhana, tanpa kejutan"
- "Mulai gratis. Naikkan paket saat butuh kuota AI lebih besar."
- CTA: `Lihat Detail Paket`

### 5.9 Cocok untuk Siapa
- **H2:** "Cocok untuk berbagai kebutuhan"
- "Pendapatan tidak tetap? Lihat arus kas per bulan dan pastikan sisanya cukup." *(Freelancer)*
- "Catat gaji dan pengeluaran bulanan tanpa membuka spreadsheet." *(Pekerja kantoran)*
- "Foto struk belanja bulanan; catatannya dirapikan otomatis." *(Ibu rumah tangga)*
- "Pantau pemasukan dan pengeluaran usaha kecil dalam satu tempat." *(Pelaku UMKM)*

### 5.10 FAQ (ringkas, 6 item)
Jawaban selalu **answer-first**:
1. "Ya. Ada paket Free yang bisa dipakai tanpa batas waktu untuk mulai mencatat."
2. "AI memproses teks, suara, dan gambar, lalu menyusunnya menjadi transaksi terstruktur."
3. "Data tiap akun terpisah. Anda dapat mengekspor atau menghapusnya kapan saja."
4. "Tidak. FinTrack memakai gateway AI terpusat — Anda tidak perlu API key sendiri."
5. "Ya. Anda dapat menghasilkan laporan dan mengekspor data keuangan Anda."
6. "Anda dapat naik atau turun paket kapan saja; downgrade berlaku di akhir periode yang sudah dibayar."

### 5.11 CTA Penutup
- **H2:** "Mulai rapikan keuangan Anda hari ini"
- **Subhead:** "Gratis untuk memulai. Tanpa kartu kredit. Data Anda dapat diekspor atau dihapus kapan saja."
- **CTA:** `Daftar Gratis` → `/register`

## 6. Naskah Halaman `/pricing`

- **H1:** "Harga FinTrack"
- **Subhead:** "Mulai gratis, naikkan paket saat butuh lebih banyak kuota AI."
- **Label toggle:** `Bulanan` / `Tahunan` — bila tahunan, tampilkan juga harga efektif per bulan.
- **Badge:** `Paling Populer` pada Pro (disengaja secara produk, bukan klaim statistik).
- **Baris pajak:** "PPN {rate}%: Rp {tax} · Total: Rp {total}" — hanya bila `tax_enabled`.
- **Catatan transparansi:** "Harga dapat berubah; biaya akhir selalu ditampilkan sebelum pembayaran."
- **FAQ harga:** lihat dokumen 04 §A.7 (jawaban answer-first, tanpa janji berlebih).
- **CTA:** `Mulai Gratis` (Free) / `Pilih {Nama}`.

## 7. Naskah Halaman Fitur

### 7.1 Hub `/features`
- **H1:** "Fitur FinTrack"
- **Subhead:** "Semua yang Anda butuhkan untuk mencatat, memahami, dan mengelola keuangan pribadi."

### 7.2 Enam sub-halaman (tagline, keyword, masalah)

| Rute | Tagline | Keyword utama | Masalah |
|------|---------|---------------|---------|
| `/features/ai-chat` | "Catat dan bertanya seperti mengobrol." | catat pengeluaran via chat, asisten keuangan AI | "Menulis catatan keuangan terasa seperti mengisi formulir." |
| `/features/voice-input` | "Catat dengan suara, tanpa menghentikan aktivitas." | catat transaksi suara, input suara Indonesia | "Tangan sibuk saat pengeluaran terjadi." |
| `/features/receipt-scan` | "Foto struk, catatannya dirapikan otomatis." | scan struk otomatis, OCR struk belanja | "Mengetik ulang deretan belanjaan itu melelahkan dan rawan salah." |
| `/features/dashboard` | "Lihat arus kas Anda dalam satu tampilan." | dashboard keuangan, grafik pengeluaran | "Angka berserakan, sulit melihat gambaran besarnya." |
| `/features/financial-health` | "Ketahui skor dan status kesehatan finansial." | analisis keuangan pribadi, skor keuangan | "Sulit menilai apakah kebiasaan keuangan Anda membaik." |
| `/features/reports` | "Hasilkan laporan, unduh data Anda." | laporan keuangan otomatis, ekspor data keuangan | "Data terkunci di aplikasi, sulit dipakai di tempat lain." |

Aturan per sub-halaman: **selalu** sertakan masalah, langkah, manfaat, satu contoh nyata (bukan testimoni), badge ketersediaan paket, dan 2 tautan "fitur terkait". Panjang ≥ 300 kata unik.

## 8. Naskah Halaman Pendukung

- **`/about`** — H1 "Tentang FinTrack" · Misi: "Membuat pencatatan keuangan semudah mengobrol — tanpa mengorbankan privasi dan kejelasan."
- **`/contact`** — H1 "Hubungi Kami" · "Sampaikan pertanyaan atau kendala Anda; kami membalas pada hari kerja." · Untuk urusan akun: "Sebutkan email akun Anda saat menghubungi."
- **`/faq`** — H1 "Pertanyaan Umum" · empat kelompok (Umum, Akun & Langganan, AI & Privasi, Pembayaran).
- **`/privacy`** — H1 "Kebijakan Privasi" · ringkasan sederhana di awal, lalu rincian.
- **`/terms`** — H1 "Syarat & Ketentuan" · bahasa formal namun terbaca.

## 9. Bank Judul (untuk uji A/B)

Judul utama (pilih satu, sisanya untuk pengujian):
1. **Catat keuangan otomatis. Kelola dengan tenang.** *(kanonik)*
2. "Keuangan yang tertata, tanpa repot mencatat."
3. "Satu kalimat. Transaksi tercatat."
4. "Pencatatan keuangan yang berjalan sendiri."

Judul section alternatif: "Kenapa mencatat harus merepotkan?" · "Bukan sekadar mencatat" · "Uang rapi, keputusan lebih tenang".

> Aturan: alternatif tetap harus jujur dan tenang. Hindari varian yang menjanjikan hal yang tidak dimiliki produk.

## 10. CTA & Microcopy

| Elemen | Teks |
|--------|------|
| CTA utama | `Mulai Gratis` |
| CTA sekunder | `Lihat Cara Kerja` |
| CTA harga | `Lihat Detail Paket` / `Pilih {Nama}` |
| CTA penutup | `Daftar Gratis` |
| Menu mobile | `Buka menu` / `Tutup menu` |
| Label demo | `Contoh tampilan` |
| Katalog gagal | "Harga sedang tidak dapat dimuat. Coba lagi sebentar lagi." + tombol `Coba lagi` |
| Data paket kosong | "Detail paket belum tersedia. Lihat halaman harga." + tombol `Lihat Harga` |
| Blog (Fase 2) | `Segera hadir` |

## 11. Judul & Deskripsi SEO (kanonik)

| Rute | title | description |
|------|-------|-------------|
| `/` | FinTrack — Catat Keuangan Otomatis dengan AI | "Catat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia yang menyimpan rapi dan menganalisis kondisi keuangan Anda." |
| `/pricing` | Harga FinTrack — Paket Gratis, Pro, dan Business | "Bandingkan paket FinTrack: kuota AI, pindai struk, input suara, dan laporan. Mulai gratis, upgrade kapan saja." |
| `/features` | Fitur FinTrack — AI, OCR Struk, Suara, Laporan | "Enam fitur utama FinTrack: catat lewat chat, suara, dan foto struk; dasbor, analisis kesehatan finansial, serta laporan dan ekspor data." |
| `/about` | Tentang FinTrack — Misi & Prinsip | "Misi FinTrack: pencatatan keuangan semudah mengobrol, dengan privasi dan transparansi kuota." |
| `/contact` | Kontak FinTrack — Bantuan & Pertanyaan | "Hubungi tim FinTrack lewat email atau WhatsApp. Lihat juga FAQ, harga, dan panduan fitur." |
| `/faq` | Pertanyaan Umum (FAQ) — FinTrack | "Jawaban atas pertanyaan umum soal FinTrack: harga, langganan, AI, privasi data, dan pembayaran." |
| `/privacy` | Kebijakan Privasi — FinTrack | "Bagaimana FinTrack mengumpulkan, memakai, dan melindungi data Anda; serta hak akses, ekspor, dan penghapusan." |
| `/terms` | Syarat & Ketentuan — FinTrack | "Ketentuan penggunaan layanan FinTrack: akun, langganan, pembayaran, kuota AI, dan tanggung jawab." |

Panjang `title` ≤ ~60 karakter, `description` ~120–160 karakter.

## 12. Anchor Text Internal

Gunakan teks bermakna yang menjelaskan tujuan tautan — jangan "klik di sini".

| Tujuan | Contoh anchor |
|--------|---------------|
| `/pricing` | "lihat paket dan harga" |
| `/features/receipt-scan` | "cara memindai struk" |
| `/features/voice-input` | "mencatat dengan suara" |
| `/privacy` | "kebijakan privasi" |
| `/contact` | "hubungi tim kami" |

## 13. Checklist Nada Sebelum Rilis

- [ ] Tidak ada superlatif tanpa dasar ("terbaik", "nomor 1", "tercepat").
- [ ] Tidak ada klaim sosial tak terverifikasi ("ribuan pengguna", testimoni).
- [ ] Tidak ada janji fitur yang tidak ada (bank sync, BYOK/API key, kategori/wallet tak terbatas).
- [ ] Semua label fitur & limit memakai terminologi resmi §3.
- [ ] Nilai `-1` tertulis "Tanpa batas"; pajak sebagai baris terpisah.
- [ ] Sapaan konsisten "Anda"; tidak ada "kamu".
- [ ] Tidak ada emoji; tanpa tanda seru beruntun.
- [ ] Setiap FAQ diawali jawaban langsung (answer-first).
- [ ] `title`/`description` sesuai bank §11 dan memuat kata kunci.
