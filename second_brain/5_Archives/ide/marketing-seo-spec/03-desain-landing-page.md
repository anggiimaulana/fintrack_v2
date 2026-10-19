# 03 — Desain Visual (Neobrutalism) & Landing Page

Dokumen ini berisi dua hal: **(A) design system** — bahasa visual neobrutalism yang dipakai seluruh situs marketing, dan **(B) landing page** — susunan section, elemen khas, copy, dan aksesibilitas.

Sumber rekomendasi: `ui-ux-pro-max` (domain `style` → **neubrutalism**: border 3px+, hard shadow offset, tanpa gradien/blur, warna jenuh, tipografi bold, "ugly-cute"; domain `landing` → pola **product-demo-features**; domain `typography`/`google-fonts` → font display; domain `ux` → aturan motion).

> Ketegangan desain yang disengaja: **visual berani** (neobrutalism) + **pesan tenang** (copy kalem & profesional). Ini kombinasi yang lazim dipakai produk finansial untuk tampil berbeda tanpa kehilangan kredibilitas.

---

# Bagian A — Design System

## A.1 Prinsip

1. **Berani tapi terbaca.** Kontras tinggi, blok warna tegas — tapi hierarki teks selalu jelas.
2. **Sudut membulat, bukan lancip.** Radius tetap `rounded-md` (6px) / `rounded-lg` (8px) seperti sistem yang ada. Keputusan eksplisit: **bukan** 0px.
3. **Tanpa gradien, tanpa blur.** Warna solid + bayangan keras (hard offset shadow).
4. **Sedikit gerakan, bertujuan.** Maksimum 2 elemen beranimasi per viewport; tanpa parallax atau scroll-jacking.
5. **Satu aksen, satu peringatan.** Kuning asam untuk aksi/highlight, merah hanya untuk masalah/peringatan.
6. **Tanpa emoji.** 100% ikon `lucide-react`.

## A.2 Bentuk: border, bayangan, radius

| Elemen | Border | Hard shadow | Radius |
|--------|--------|-------------|--------|
| Kartu besar (fitur, harga, persona) | `border-[3px]` | `shadow-[6px_6px_0px_0px]` | `rounded-lg` |
| Kartu kecil / input / badge besar | `border-2` | `shadow-[4px_4px_0px_0px]` | `rounded-md` |
| Tombol | `border-2` | `shadow-[3px_3px_0px_0px]` | `rounded-md` |
| Blok gelap (hero/footer) | `border-b-2` / `border-t-2` | — | — |

Interaksi tombol & kartu mengikuti pola yang sudah ada: saat `:active` → `shadow-none` + `translate(3px, 3px)`; saat hover → bayangan naik satu tingkat (`4px→6px`). Transisi cepat (lihat A.5).

Kelas util yang dipakai: `.neo-card`, `.neo-card-sm`, `.neo-button`, `.neo-button-primary`, `.neo-badge`, `.neo-input`, `.neo-label`, `.neo-section-title` (sudah ada di `globals.css`). Tambahan baru untuk aksen: `.neo-highlight` (latar kuning asam + teks gelap) dan `.neo-accent-shadow` (shadow berwarna).

## A.3 Warna

### Token yang sudah ada (dipertahankan)

| Token | Nilai | Peran |
|-------|-------|-------|
| `--background` | `oklch(0.98 0 0)` | Latar halaman terang |
| `--foreground` | `oklch(0.15 0.02 260)` | Teks utama + garis border |
| `--primary` | `oklch(0.55 0.24 260)` | Biru aksi utama |
| `--primary-strong` | `oklch(0.5 0.24 260)` | Teks di atas permukaan `primary/10` (AA) |
| `--destructive` | `oklch(0.58 0.24 25)` | Merah peringatan |
| `--chart-3` | `oklch(0.65 0.2 145)` | Hijau (status baik) |
| `--surface-dark` / `-2` / `-3` | `oklch(0.22/0.26/0.31 0.03 265)` | Blok gelap hero & footer |

### Token aksen BARU (keputusan revisi)

```css
:root {
  /* Kuning asam: hanya untuk aksi utama & highlight. */
  --accent-acid: oklch(0.92 0.19 103);        /* ≈ #FFE23D */
  --accent-acid-strong: oklch(0.86 0.19 100); /* hover */
  --accent-acid-foreground: oklch(0.15 0.02 260); /* teks di atas kuning = gelap */
  /* Merah blok masalah memakai --destructive yang sudah ada. */
}
@theme inline {
  --color-accent-acid: var(--accent-acid);
  --color-accent-acid-strong: var(--accent-acid-strong);
  --color-accent-acid-foreground: var(--accent-acid-foreground);
}
```

### Aturan pemakaian warna

| Peran | Warna | Aturan |
|-------|-------|--------|
| Latar halaman | `background` | Terang. |
| Blok hero & footer | `surface-dark` | Gelap; teks minimal `text-neutral-300`, judul `text-white`. |
| Teks utama | `foreground` | Di atas latar terang. |
| CTA utama hero | `accent-acid` + `accent-acid-foreground` | Pop maksimum; **selalu** border gelap 2px + hard shadow. |
| CTA lain | `primary` + `primary-foreground` | Aksi sekunder/umum. |
| Highlight/label/sticker | `accent-acid` | Blok kecil, bukan latar section penuh. |
| Masalah / peringatan | `destructive` | Hanya untuk blok "tanpa FinTrack", galat, status kritis. |
| Status baik | `chart-3` (hijau) | Status SPK. |

**Aturan kontras (wajib):**
- **Jangan pernah** teks putih di atas kuning asam. Di atas `--accent-acid` selalu teks gelap (`#0F172A`).
- Merah hanya sebagai aksen blok; teks di atas merah memakai putih **atau** gelap sesuai hasil ukur ≥ 4.5:1.
- Teks kecil di atas `surface-dark` minimal `neutral-300`; `neutral-500` ke bawah dilarang untuk teks penting.
- Jangan menyampaikan makna **hanya** lewat warna: sertakan ikon/teks (mis. status SPK punya label, bukan cuma titik warna).

## A.4 Tipografi

### Font (via `next/font/google`, tanpa dependency npm baru)

| Peran | Font | Alasan |
|-------|------|--------|
| **Display (heading)** | **Funnel Display** (variable 300–800) — *rekomendasi* | Display geometric modern; tegas tapi tetap profesional (cocok untuk "menjual" tanpa norak). |
| Body & UI | **Inter** (sudah dipakai) | Netral, sangat terbaca, sudah terpasang. |
| Data & label angka | **JetBrains Mono** (sudah dipakai) | Angka Rupiah, label eyebrow uppercase, badge data. |

**Alternatif font display** (tukar satu baris di `layout.tsx`): **Lexend Mega** (paling brutalist, weight sampai 900 — pilihan "Neubrutalist Bold" dari database skill) atau **Baloo 2** (rounded, paling cocok dengan sudut membulat).

```ts
// app/layout.tsx
import { Inter, JetBrains_Mono, Funnel_Display } from "next/font/google";
const display = Funnel_Display({ variable: "--font-display", subsets: ["latin"] });
// <html className={`${inter.variable} ${jetbrainsMono.variable} ${display.variable} ...`}>
```
Lalu di `globals.css`: `--font-heading: var(--font-display);` (sebelumnya `--font-heading: var(--font-sans)`).

### Skala

| Peran | Ukuran | Gaya |
|-------|--------|------|
| H1 hero | `clamp(2.5rem, 6vw, 4.5rem)` | Display, weight 800, `leading-[1.05]`, `tracking-[-0.02em]` |
| H2 section | `clamp(1.875rem, 4vw, 3rem)` | Display, weight 800, uppercase |
| H3 kartu | `1.25rem` | Display/Inter, weight 800 |
| Body | `1rem`, `leading-relaxed` | Inter 400–500 |
| Teks sekunder | `0.875rem` | Inter 500, `muted-foreground` |
| Eyebrow/label | `0.75rem`, uppercase, `tracking-[0.08em]` | JetBrains Mono, weight 700 |
| Angka besar (stat) | `clamp(2rem, 5vw, 3.5rem)` | JetBrains Mono, weight 800 |

Aturan: body minimal 16px; heading maksimal 1 baris untuk kata kunci utama; hindari heading > 3 baris di mobile.

## A.5 Motion

Berdasarkan panduan domain `ux` yang terverifikasi: hormati `prefers-reduced-motion`; **maksimum 1–2 elemen beranimasi per viewport**; tanpa parallax/scroll-jacking; durasi dari token bersama (bukan satu angka untuk semua).

```css
:root {
  --motion-fast: 100ms;   /* umpan balik interaksi (aktif/hover) */
  --motion-base: 200ms;   /* transisi umum */
  --motion-slow: 320ms;   /* reveal masuk */
}
```

| Animasi | Detail | Aturan |
|---------|--------|--------|
| Umpan balik tombol | `translate` + shadow (sudah ada) | `--motion-fast`, `ease-out` |
| Scroll reveal (`Reveal`) | opacity 0→1 + `translateY(10px)` → 0 | `--motion-slow`, sekali saja, via `IntersectionObserver` |
| Receipt tape marquee | `translateX` linear, loop | Pause saat hover/focus; **berhenti total** saat reduced-motion |
| Ticker band | idem | idem |
| Stat counter | angka naik sekali saat masuk viewport | Berhenti saat reduced-motion; nilai akhir tetap ada di DOM |

**Wajib:** hanya `transform` dan `opacity` yang dianimasikan — jangan `width`/`height`/`top`/`left` (menyebabkan layout thrashing).

## A.6 Komponen dasar

| Komponen | Deskripsi | Client? |
|----------|-----------|---------|
| `Button` | Sudah ada; tambah varian aksen (`accent-acid`) | tidak |
| `Card` (neo) | `border-[3px]` + `shadow-[6px_6px_0]` | tidak |
| `Badge`/`Sticker` | `neo-badge`, sebagian dirotasi `-2deg`/`2deg` | tidak |
| `SectionHeading` | Eyebrow + H2 + subjudul; konsisten antar section | tidak |
| `ReceiptTapeMarquee` | Strip struk berjalan (dekoratif, `aria-hidden`) | tidak |
| `TickerBand` | Pita teks berjalan pemisah section | tidak |
| `Reveal` | Pembungkus scroll-reveal tipis | **ya** |
| `StatCounters` | Angka statistik beranimasi | **ya** |
| `InteractiveDemo` | Tab Chat/Voice/Receipt di hero | **ya** |

## A.7 Do / Don't

| Do | Don't |
|----|-------|
| Border tebal **dan** konsisten | Mencampur border 1px dan 3px sembarangan |
| Hard shadow offset konsisten (3/4/6px) | `box-shadow` berblur (`shadow-lg`, `shadow-xl`) |
| Blok warna solid | Gradien, glow, atau efek neon |
| Kuning asam hanya untuk aksi/highlight | Kuning asam jadi latar section besar |
| Sudut membulat 6–8px | Sudut 0px (lancip) atau radius besar ala pil |
| 1–2 elemen beranimasi per viewport | Animasi di hampir semua elemen |
| Tipografi display tegas | Emoji sebagai ikon |

---

# Bagian B — Landing Page

## B.1 Tujuan

1. **Dipahami** < 10 detik: aplikasi pencatat keuangan berbasis AI berbahasa Indonesia.
2. **Dipercaya:** klaim akurat, bukti nyata (ekspor/hapus data), tanpa testimoni palsu.
3. **Bertindak:** **Mulai Gratis** (`/register`) sebagai aksi utama; `/pricing` sebagai jalur pertimbangan.

## B.2 Sasaran Kata Kunci

> Volume bukan estimasi pasti; ini heuristik intent. Validasi dengan Search Console setelah rilis. URL berbahasa Inggris **tidak** mengubah kata kunci — konten tetap Bahasa Indonesia.

| Peran | Kata kunci | Penempatan |
|-------|-----------|------------|
| Utama | aplikasi pencatat keuangan, catat keuangan otomatis | H1, title, 100 kata pertama |
| Sekunder | catat pengeluaran via chat, scan struk otomatis, input transaksi suara | H2 fitur |
| Sekunder | analisis keuangan pribadi, laporan keuangan otomatis | H2 SPK & laporan |
| Long-tail | cara mencatat pengeluaran harian, aplikasi keuangan tanpa ribet | FAQ, sub-halaman fitur |
| GEO/AI | "apa itu FinTrack", "apakah FinTrack gratis" | FAQ (answer-first) |
| Brand | FinTrack | seluruh halaman |

## B.3 Susunan Section

| # | Section | Komponen | Anchor |
|---|---------|----------|--------|
| 1 | Header | `MarketingHeader` | — |
| 2 | Hero + demo interaktif | `Hero` + `InteractiveDemo` | `#hero` |
| 3 | Receipt tape marquee | `ReceiptTapeMarquee` | — |
| 4 | Trust bar + stat counters | `TrustBar` + `StatCounters` | — |
| 5 | Masalah → Solusi | `ProblemSolution` | `#solution` |
| 6 | Fitur unggulan | `FeatureGrid` | `#features` |
| 7 | Cara kerja | `HowItWorks` | `#how-it-works` |
| 8 | Highlight kesehatan finansial | `SpkHighlight` | `#insights` |
| 9 | Ticker band | `TickerBand` | — |
| 10 | Teaser harga | `PricingTeaser` | `#pricing` |
| 11 | Cocok untuk siapa | `PersonaUseCases` | `#who-its-for` |
| 12 | FAQ | `FaqAccordion` | `#faq` |
| 13 | CTA penutup | `CtaSection` | — |
| 14 | Footer | `MarketingFooter` | — |

> Section `#pricing` **nyata** (menutup F-01). Anchor `#how-it-works` menjadi target tombol "Lihat Cara Kerja" (menutup F-03).
>
> **Disiplin motion:** hanya marquee (3 & 9), reveal, dan stat counter (4) yang beranimasi. Section lain statis.

## B.4 Detail Section

### B.4.1 Hero

- **Eyebrow (mono, uppercase):** `ASISTEN KEUANGAN BERBAHASA INDONESIA`
- **H1:** **Catat keuangan otomatis. Kelola dengan tenang.**
  - Kata kunci "catat keuangan otomatis" ada di H1 dan title.
  - Gaya: display 800, satu frasa per baris, baris kedua diberi highlight kuning asam (`.neo-highlight`) pada kata **"tenang"**.
- **Subhead:** "Tulis, ucapkan, atau foto transaksinya. FinTrack mencatat dengan rapi dan membantu Anda memahami kondisi keuangan — tanpa spreadsheet."
- **CTA utama:** `Mulai Gratis` → `/register` — varian **accent-acid** (pop maksimum).
- **CTA sekunder:** `Lihat Cara Kerja` → `#how-it-works` — varian outline di atas `surface-dark`.
- **Trust chips:** `Gratis untuk memulai` · `Tanpa kartu kredit` · `Data dapat diekspor`
- **Visual:** `InteractiveDemo` (lihat B.5).

### B.4.2 Receipt Tape Marquee (elemen khas #1)

Strip horizontal bergaya kertas struk yang bergerak perlahan, berisi baris transaksi **contoh**:

```
KOPI SUSU        Rp 25.000    MAKANAN
BELANJA BULANAN  Rp 312.500   RUMAH TANGGA
TOL CIKAMPEK     Rp 27.500    TRANSPORTASI
GAJI             +Rp 8.500.000  PEMASUKAN
```

- Dibungkus `border-y-2 border-foreground` + latar `bg-muted`; font mono.
- **Dekoratif** → `aria-hidden="true"`, duplikasi track untuk loop mulus.
- Caption kecil (bukan aria-hidden): "Ilustrasi tampilan transaksi yang tercatat."
- Pause saat hover/focus; mati saat `prefers-reduced-motion` (track menjadi baris statis yang membungkus).
- Data ini **contoh**, bukan data pengguna.

### B.4.3 Trust Bar + Stat Counters

**Trust bar** — 4 poin **faktual**:

| Poin | Kalimat | Basis kebenaran |
|------|---------|-----------------|
| Isolasi data | "Data tiap akun terpisah dan hanya Anda yang mengaksesnya." | Isolasi per-akun |
| Kendali data | "Ekspor seluruh data Anda kapan saja." | `GET /api/v1/user/export-data` |
| Hapus kapan saja | "Hapus akun beserta datanya dari pengaturan." | `DELETE /api/v1/me` |
| Transparan | "Kuota dan pemakaian AI terlihat jelas di dasbor." | `/user/usage`, `/user/subscription` |

**Stat counters** (elemen khas #5) — hanya angka yang benar:

| Angka | Label |
|-------|-------|
| `3` | cara mencatat: chat, suara, foto |
| `5` | indikator status kesehatan keuangan |
| `Rp 0` | biaya untuk memulai |

### B.4.4 Masalah → Solusi (`#solution`)

- **H2:** "Mencatat keuangan tidak seharusnya menyita waktu"
- Dua kolom dengan warna tegas: kolom kiri berlatar **merah redup** (masalah), kanan berlatar **terang** dengan aksen kuning (solusi).

| Tanpa FinTrack | Dengan FinTrack |
|----------------|-----------------|
| Membuka spreadsheet dan mengisi kolom satu per satu | Menulis "makan siang 40 ribu" — langsung tercatat |
| Lupa mencatat karena repot | Mencatat lewat chat, suara, atau foto struk |
| Data menumpuk tanpa pernah ditinjau | Mengajukan pertanyaan; mendapat skor dan status keuangan |

### B.4.5 Fitur Unggulan (`#features`) — 6 kartu

| Ikon | Judul | Deskripsi (nada kalem) | Tautan |
|------|-------|------------------------|--------|
| `MessageSquare` | Catat & Tanya Lewat Chat | "Tulis transaksi seperti mengirim pesan. Jumlah, kategori, dan tanggalnya disusun otomatis." | `/features/ai-chat` |
| `Mic` | Catat dengan Suara | "Sampaikan pengeluaran dengan suara saat tangan Anda sedang sibuk." | `/features/voice-input` |
| `Camera` | Pindai Struk Otomatis | "Foto struk belanja; item dan totalnya dibaca lalu disimpan." | `/features/receipt-scan` |
| `LayoutDashboard` | Dasbor Keuangan Interaktif | "Pantau pemasukan, pengeluaran, dan tren dalam tampilan yang mudah dibaca." | `/features/dashboard` |
| `Target` | Analisis Kesehatan Finansial | "Penilaian kondisi keuangan dengan skor dan status yang jelas." | `/features/financial-health` |
| `Download` | Laporan & Ekspor Data | "Hasilkan laporan dan unduh data Anda kapan saja." | `/features/reports` |

> **F-04 ditutup:** tidak ada klaim "AI key milik Anda sendiri". Nilai privasi disampaikan lewat fakta yang benar.

### B.4.6 Cara Kerja (`#how-it-works`) — 3 langkah

| Langkah | Judul | Deskripsi |
|---------|-------|-----------|
| 1 | Daftar gratis | "Buat akun dalam hitungan detik. Tidak perlu kartu kredit." |
| 2 | Catat transaksi | "Pilih cara paling nyaman: chat, suara, atau foto struk." |
| 3 | Tinjau & analisis | "Lihat ringkasan, lalu ajukan pertanyaan tentang keuangan Anda." |

Nomor langkah memakai blok kuning asam dengan teks gelap + hard shadow.

### B.4.7 Highlight Kesehatan Finansial (`#insights`)

- **H2:** "Ketahui kondisi keuangan Anda"
- Deskripsi: "Penilaian meninjau pola pengeluaran, tabungan, dan kebiasaan Anda, lalu memberi skor beserta status yang jelas."
- Skala status (dari konfigurasi metrik SPK backend, bukan karangan): **Sangat Baik · Baik · Cukup · Boros · Kritis**.
- Kartu contoh: skor `85` dengan cincin progres + label "Sangat Baik" + penjelasan singkat.
- Setiap status punya **label teks** (bukan hanya warna).

### B.4.8 Ticker Band (elemen khas #3)

Pita teks besar berjalan sebagai pemisah: `CATAT · PAHAMI · KELOLA · TENANG ·`.

- Latar `accent-acid` + teks gelap, `border-y-2`.
- `aria-hidden="true"` + satu versi statis untuk pembaca layar.
- Aturan motion sama seperti marquee.

### B.4.9 Teaser Harga (`#pricing`)

- **H2:** "Harga sederhana, tanpa kejutan"
- Menerima data dari `lib/plans-server.ts` (dokumen 06). Tampilkan 3 kartu ringkas: nama, harga bulanan, 3 poin fitur utama, tombol `Lihat Detail Paket` → `/pricing`.
- Menandai paket **Pro** sebagai "Paling Populer" (disengaja secara produk, bukan klaim statistik).
- **Fallback:** bila data gagal dimuat, sembunyikan kartu dan tampilkan satu tombol `Lihat Harga` → `/pricing`.

### B.4.10 Cocok untuk Siapa (`#who-its-for`) — pengganti testimoni (F-05 ditutup)

Empat kartu persona, **tanpa nama fiktif dan tanpa kutipan palsu**:

| Persona | Kalimat |
|---------|---------|
| Freelancer | "Pendapatan tidak tetap? Lihat arus kas per bulan dan pastikan sisanya cukup." |
| Pekerja kantoran | "Catat gaji dan pengeluaran bulanan tanpa membuka spreadsheet." |
| Ibu rumah tangga | "Foto struk belanja bulanan; catatannya dirapikan otomatis." |
| Pelaku UMKM | "Pantau pemasukan dan pengeluaran usaha kecil dalam satu tempat." |

### B.4.11 FAQ (`#faq`) — 6 item

Ringkas di landing; versi lengkap di `/faq`. Jawaban **answer-first**.

1. Apakah FinTrack gratis? — "Ada paket Free untuk mulai. Paket berbayar menambah kuota AI dan fitur." → `/pricing`
2. Bagaimana AI memahami perintah saya? — "AI memproses teks, suara, dan gambar, lalu menyusun transaksi terstruktur." → `/features/ai-chat`
3. Apakah data saya aman? — isolasi per akun, ekspor, hapus. → `/privacy`
4. Apakah perlu API key sendiri? — **"Tidak."** → `/features/ai-chat`
5. Apakah bisa ekspor data? — ya; sebutkan fiturnya. → `/features/reports`
6. Bagaimana cara berlangganan atau membatalkan? — jelaskan singkat. → `/pricing`

### B.4.12 CTA Penutup

- **H2:** "Mulai rapikan keuangan Anda hari ini"
- **Subhead:** "Gratis untuk memulai. Tanpa kartu kredit. Data Anda dapat diekspor atau dihapus kapan saja." (menggantikan klaim "ribuan pengguna")
- **CTA:** `Daftar Gratis` → `/register` — varian accent-acid.
- Latar `surface-dark` agar kontras dengan section sebelumnya; satu blok kuning asam di dalamnya sebagai aksen.

## B.5 Interactive Demo (elemen khas #2)

Demo produk hidup di hero — **mock data statis, tanpa panggilan backend**. Ini menjelaskan nilai lebih baik daripada gambar statis.

**Struktur:** `role="tablist"` dengan tiga tab: **Chat**, **Suara**, **Struk**. Panel menampilkan contoh interaksi.

| Tab | Contoh tampilan |
|-----|-----------------|
| Chat | Input: "beli kopi 25 ribu tadi siang" → hasil: "Pengeluaran Rp 25.000 — Kopi. Tercatat." |
| Suara | Bentuk gelombang suara + transkrip "makan siang empat puluh ribu" → "Pengeluaran Rp 40.000 — Makan. Tercatat." |
| Struk | Kartu struk + daftar item hasil baca + total `Rp 312.500` |

**Aksesibilitas:**
- `role="tablist"`, tiap tab `role="tab"` dengan `aria-selected` + `aria-controls`; panel `role="tabpanel"` + `aria-labelledby`.
- Navigasi keyboard panah kiri/kanan + `Home`/`End`; `tabIndex` roving.
- **Semua panel ada di DOM** (disembunyikan dengan `hidden`), sehingga konten tersedia tanpa interaksi.
- Perpindahan tab boleh beranimasi halus (`--motion-base`); tanpa autoplay.
- Label kecil "Contoh tampilan" agar tidak dikira data nyata.

## B.6 Alur Konversi

```text
Masuk (organik/ads)
  → Hero: paham dalam 10 detik → [Mulai Gratis] atau [Lihat Cara Kerja]
  → Demo interaktif: nilai produk terlihat langsung
  → Trust bar + stat: bukti yang jujur
  → Masalah → Solusi: relevansi
  → Fitur: cocokkan kebutuhan → klik sub-fitur (bila ragu)
  → Cara kerja: turunkan keraguan "sulit dipakai"
  → Kesehatan finansial + teaser harga: tunjukkan nilai → [Lihat Detail Paket] ke /pricing
  → Persona: "ini untuk saya"
  → FAQ: jawab keberatan (harga, privasi, API key)
  → CTA penutup → /register
```

Prinsip: **satu aksi utama** (daftar) dengan banyak pintu masuk menuju aksi itu, tanpa memaksa berpindah halaman di awal.

## B.7 Hirarki Heading

- Satu `H1` (hero). `H2` per section. `H3` untuk kartu/judul item.
- Tidak melompat level (`H1` → `H3`).
- Heading tidak boleh menyampaikan makna hanya lewat warna/latar.

## B.8 Aksesibilitas (F-08 ditutup)

| Area | Persyaratan |
|------|-------------|
| Menu mobile | `<button>` ber-`aria-expanded`, `aria-controls`, `aria-label`; `Escape` menutup; fokus kembali ke tombol; `aria-hidden` saat tertutup |
| Tabs demo | `role="tablist"/"tab"/"tabpanel"`, panah keyboard, `aria-selected` |
| Marquee/ticker | `aria-hidden="true"` untuk track bergerak; versi statis tersedia; pause saat hover/focus; mati saat reduced-motion |
| Counter | Nilai akhir ada di DOM; animasi dekoratif; statis saat reduced-motion |
| Ikon | Dekoratif → `aria-hidden="true"`; bermakna → punya teks/`aria-label` |
| Kontras | ≥ 4.5:1 untuk teks; kuning asam selalu berpasangan teks gelap |
| Fokus | Cincin fokus terlihat di semua tautan/tombol |
| Akordeon FAQ | `<details>`/`<summary>` native → berfungsi tanpa JS |
| Bahasa | `lang="id"` (root layout) meski URL berbahasa Inggris |

## B.9 Performa

- Landing seluruhnya Server Component kecuali `MarketingHeader`, `InteractiveDemo`, `StatCounters`, `Reveal`.
- Tanpa gambar raster besar; mockup & struk dibuat CSS/HTML.
- Tanpa library animasi (tanpa GSAP); memakai CSS + `tw-animate-css` + `IntersectionObserver`.
- Font display di-subset `latin` lewat `next/font` (self-hosted, tanpa shift).
- Animasi hanya `transform`/`opacity`; maksimum 2 elemen beranimasi per viewport.
