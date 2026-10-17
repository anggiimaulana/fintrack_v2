# 02 — Arsitektur Informasi & Routing

Dokumen ini menetapkan peta situs, hierarki URL, struktur berkas Next.js, perbaikan guard auth, dan strategi internal linking untuk situs marketing Fase 1.

**Kebijakan bahasa:** seluruh **URL & slug berbahasa Inggris** (profesional, konsisten, ramah SEO internasional), sedangkan **konten di dalam halaman berbahasa Indonesia** (`<html lang="id">`). Label navigasi juga Bahasa Indonesia karena bagian dari konten.

---

## 1. Peta Situs (Information Architecture)

```
/                         Landing (perombakan)
├── /pricing              Paket, komparasi fitur & limit, FAQ harga
├── /features             Hub fitur (ringkasan 6 fitur)
│   ├── /features/ai-chat           Asisten AI percakapan
│   ├── /features/voice-input       Catat via suara
│   ├── /features/receipt-scan      OCR foto struk
│   ├── /features/dashboard         Dashboard & grafik
│   ├── /features/financial-health  Skor & status kesehatan finansial (SPK)
│   └── /features/reports           Laporan & ekspor data
├── /about                Misi, cara kerja, perusahaan
├── /contact              Kontak (mailto/WhatsApp) + jam operasional
├── /faq                  Tanya jawab terkelompok
├── /privacy              Kebijakan Privasi (UU PDP)
├── /terms                Syarat & Ketentuan
│
├── /login                (auth, sudah ada)
├── /register             (auth, sudah ada)
├── /user/*               (dashboard user, sudah ada)
├── /admin/*              (dashboard admin, sudah ada)
└── /super-admin/*        (dashboard super admin, sudah ada)

# Fase 2 (belum dibuat di Fase 1):
#   /blog                 Hub artikel
#   /blog/[slug]          Artikel
```

Prinsip: **kedalaman maksimum 2 tingkat** dari root. Semua halaman dapat dijangkau ≤ 2 klik dari beranda (lihat §6).

## 2. Hierarki URL & Alasan

| Rute | Tipe | Alasan pemilihan URL |
|------|------|----------------------|
| `/` | Statis | Beranda. |
| `/pricing` | ISR | Istilah universal untuk halaman paket; jelas bagi pengunjung maupun mesin. |
| `/features` | Statis | Hub; standar produk SaaS. |
| `/features/{slug}` | SSG (`generateStaticParams`) | Slug Inggris deskriptif yang menargetkan long-tail intent (`/features/receipt-scan`). |
| `/about` | Statis | Halaman trust/company. |
| `/contact` | Statis | Halaman trust + kanal dukungan. |
| `/faq` | Statis | Akronim lazim; sama di kedua bahasa. |
| `/privacy` | Statis | Istilah legal standar. |
| `/terms` | Statis | Ringkas; hindari `/terms-and-conditions` yang panjang. |

Aturan teknis: huruf kecil, dipisah tanda hubung, tanpa akhiran `.html`, tanpa parameter kueri untuk konten utama, **tanpa trailing slash**.

### 2.1 Pemetaan slug 6 sub-halaman fitur

| Slug (URL) | Judul halaman (konten) | Fitur katalog terkait |
|------------|------------------------|-----------------------|
| `/features/ai-chat` | Catat & Tanya Lewat Chat | `ai_chat` |
| `/features/voice-input` | Catat dengan Suara | `voice_input` |
| `/features/receipt-scan` | Pindai Struk Otomatis | `ocr_receipt` |
| `/features/dashboard` | Dasbor Keuangan Interaktif | — |
| `/features/financial-health` | Analisis Kesehatan Finansial | — |
| `/features/reports` | Laporan & Ekspor Data | `reports_export` |

> Slug Inggris, judul & isi Indonesia. Tabel ini adalah satu-satunya sumber kebenaran slug; dipakai oleh `lib/marketing/features.ts`, sitemap, tautan internal, dan tes.

## 3. Struktur Berkas Next.js (Route Group)

Landing yang sekarang berada di `frontend/app/page.tsx` **dipindahkan** ke `app/(marketing)/page.tsx`. Route group `(...)` **tidak mengubah URL**, sehingga `/` tetap `/`. Root `app/layout.tsx` tetap sebagai layout akar.

```
frontend/app/
├── layout.tsx                     # root (diubah: metadataBase, template, OG default, font display)
├── globals.css                    # (diubah: tambah token aksen kuning/merah + kelas neo)
├── providers.tsx                  # (tidak diubah)
├── sitemap.ts                     # BARU
├── robots.ts                      # BARU
├── opengraph-image.tsx            # BARU (OG image dinamis)
├── (marketing)/
│   ├── layout.tsx                 # BARU — MarketingHeader + MarketingFooter
│   ├── page.tsx                   # DIPINDAH dari app/page.tsx + dirombak
│   ├── pricing/page.tsx           # BARU
│   ├── features/
│   │   ├── page.tsx               # BARU
│   │   └── [slug]/page.tsx        # BARU
│   ├── about/page.tsx             # BARU
│   ├── contact/page.tsx           # BARU
│   ├── faq/page.tsx               # BARU
│   ├── privacy/page.tsx           # BARU
│   └── terms/page.tsx             # BARU
├── (auth)/                        # (tidak diubah)
└── (dashboard)/                   # (tidak diubah)

frontend/components/marketing/     # BARU
├── MarketingHeader.tsx            # client (menu mobile)
├── MarketingFooter.tsx            # server
├── SectionHeading.tsx             # server
├── Hero.tsx                       # server
├── InteractiveDemo.tsx            # client (3 tab: Chat / Voice / Receipt)
├── ReceiptTapeMarquee.tsx         # server + CSS marquee (elemen khas)
├── TickerBand.tsx                 # server + CSS marquee (elemen khas)
├── StatCounters.tsx               # client (IntersectionObserver, hormati reduced-motion)
├── TrustBar.tsx                   # server
├── ProblemSolution.tsx            # server
├── FeatureGrid.tsx                # server
├── HowItWorks.tsx                 # server
├── PersonaUseCases.tsx            # server (pengganti testimoni)
├── SpkHighlight.tsx               # server
├── PricingTeaser.tsx              # server (menerima data paket via props)
├── PricingCards.tsx               # client (toggle bulanan/tahunan)
├── ComparisonTable.tsx            # server
├── FaqAccordion.tsx               # client (`<details>`/`<summary>`)
├── CtaSection.tsx                 # server
├── Breadcrumbs.tsx                # server
├── LegalPage.tsx                  # server (render privacy & terms)
├── Reveal.tsx                     # client tipis (IntersectionObserver, no-dependency)
└── JsonLd.tsx                     # server

frontend/lib/marketing/            # BARU
├── site.ts                        # brand, URL dasar, kontak, kanal
├── features.ts                    # data 6 fitur + isi sub-halaman
├── faq.ts                         # Q&A per kategori
├── legal.ts                       # isi privacy & terms
└── copy.ts                        # naskah landing & CTA (lihat dokumen 13)
frontend/lib/
├── plans-server.ts                # BARU — fetch katalog publik (server-only)
├── plans-types.ts                 # BARU — tipe PublicPlan bersama
└── seo.ts                         # BARU — helper metadata & JSON-LD
```

### Aturan batas modul

- `lib/plans-server.ts` **wajib server-only** (memakai `process.env.BACKEND_ORIGIN`). Tidak boleh diimpor oleh komponen client. Beri komentar tegas di kepala berkas.
- Komponen ber-`"use client"` wajib sekecil mungkin: `MarketingHeader`, `PricingCards`, `FaqAccordion`, `InteractiveDemo`, `StatCounters`, `Reveal`. Sisanya Server Component.
- `lib/marketing/*` adalah data murni bertipe (tanpa JSX), sehingga dapat diuji unit tanpa DOM.
- Tidak ada dependency runtime baru: marquee & reveal memakai CSS/`tw-animate-css` + `IntersectionObserver`.

---

## 4. Perbaikan Guard Auth (`lib/roles.ts`) — KRITIS

### Kondisi sekarang (F-07)

```ts
const publicPath = pathname === '/login' || pathname === '/' || pathname === '/register';
```

### Perubahan yang diminta

Ganti dengan **allowlist prefix** yang mencakup permukaan publik (kini berbahasa Inggris). Pertahankan perilaku rute dashboard persis seperti sekarang.

```ts
/**
 * Prefix yang boleh diakses tanpa sesi. Dipakai untuk memutuskan apakah
 * pengunjung anonim diarahkan ke /login atau dibiarkan lewat.
 * Daftar ini mencerminkan permukaan marketing publik (lihat spec Fase 1).
 */
const PUBLIC_PREFIXES = [
  '/pricing',
  '/features',
  '/about',
  '/contact',
  '/faq',
  '/privacy',
  '/terms',
] as const;

function isPublicPath(pathname: string): boolean {
  // Halaman akar & auth persis.
  if (pathname === '/' || pathname === '/login' || pathname === '/register') return true;
  // Prefix marketing (cocok untuk '/pricing' maupun '/features/receipt-scan').
  return PUBLIC_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(`${p}/`),
  );
}
```

Lalu `const publicPath = isPublicPath(pathname);`.

### Matriks perilaku yang diharapkan

| pathname | ada sesi? | token valid? | Hasil |
|----------|-----------|--------------|-------|
| `/` | tidak | — | `pass` |
| `/pricing` | tidak | — | `pass` |
| `/features/receipt-scan` | tidak | — | `pass` |
| `/about` | tidak | — | `pass` |
| `/privacy` | tidak | — | `pass` |
| `/login` | tidak | — | `pass` |
| `/user` | tidak | — | `redirect-login` |
| `/admin` | tidak | — | `redirect-login` |
| `/pricing` | ya | ya | `pass` |
| `/login` | ya | ya | `{ redirect: homePathForRole(role) }` |
| `/`,`/pricing` | ya | tidak | `pass` (biar klien refresh) |

> Catatan penting: prefix harus dibandingkan sebagai **segmen penuh** (`=== p` atau `startsWith(p + '/')`), bukan `startsWith(p)`, agar `/pricingx` atau `/featuresx` tidak ikut lolos.

### Dampak pada tes

`frontend/lib/roles.test.ts` (berkas tes yang sudah ada, pola `cases` array) harus ditambah:
- anonim → tiap path publik = `pass`;
- anonim → `/pricingx` dan `/featuresx` = `redirect-login` (regresi segmentasi);
- sesi valid di `/login` tetap bounce ke home per peran;
- seluruh perilaku `/user`, `/admin`, `/super-admin` tidak berubah.

### Pertimbangan keamanan

Membuka rute marketing ke anonim **tidak** memaparkan data sensitif: halaman ini hanya konten statis + harga publik dari endpoint publik baru (dokumen 06). Tidak ada query ber-scope pengguna. Endpoint `/api/v1/plans` yang lama **tetap** di balik autentikasi (tidak diubah).

---

## 5. Navigasi & Footer

### MarketingHeader (komponen client)

- Logo `FinTrack` (ikon `Wallet`, tanpa emoji) → `/`.
- Nav desktop (label Indonesia, tujuan Inggris): **Fitur** (`/features`), **Cara Kerja** (`/#how-it-works`), **Harga** (`/pricing`), **FAQ** (`/faq`).
- Aksi: **Masuk** (`/login`) — `variant="outline"`; **Mulai Gratis** (`/register`) — `variant="default"`.
- Mobile (< `md`): tombol hamburger dengan `aria-expanded`, `aria-controls`, dan `aria-label="Buka menu"`. Panel menu menampilkan tautan + aksi; menutup saat tautan dipilih, saat `Escape`, dan saat fokus keluar. Mencegah scroll latar saat terbuka.
- Sticky `top-0`, `border-b-2 border-foreground`, latar `bg-card`.

### MarketingFooter (komponen server)

Empat kolom tautan nyata (tidak ada `#` hampa kecuali anchor yang benar-benar ada):

| Kolom | Tautan |
|-------|--------|
| Produk | Fitur (`/features`), Harga (`/pricing`), Cara Kerja (`/#how-it-works`), FAQ (`/faq`) |
| Perusahaan | Tentang (`/about`), Kontak (`/contact`) |
| Sumber Daya | Blog (Fase 2 — tampil disabled/"Segera hadir" sampai ada), Panduan (→ `/features`) |
| Legal | Kebijakan Privasi (`/privacy`), Syarat & Ketentuan (`/terms`) |

Baris bawah: hak cipta + tagline. Footer juga memuat JSON-LD `Organization` (dokumen 07).

### Alur navigasi (klik dari beranda)

| Tujuan | Klik dari `/` |
|--------|---------------|
| `/pricing` | 1 (nav atau teaser harga) |
| `/features` | 1 |
| `/features/receipt-scan` | 2 |
| `/faq` | 1 |
| `/about` | 2 (footer) |
| `/contact` | 2 (footer) |
| `/privacy` | 2 (footer) |
| `/terms` | 2 (footer) |

---

## 6. Strategi Internal Linking

Tujuan: menyebarkan otoritas, memudahkan penemuan halaman, dan memberi konteks ke mesin pencari.

1. **Dari landing:**
   - Hero → `/register`; "Lihat Cara Kerja" → `#how-it-works`.
   - Kartu fitur unggulan → masing-masing menautkan ke sub-halaman `/features/{slug}` yang relevan.
   - Teaser harga (`#pricing`) → `/pricing`.
   - FAQ landing (6 item) + tautan "Lihat semua FAQ" → `/faq`.
   - Footer → seluruh halaman.
2. **Dari `/pricing`:**
   - Setiap kartu paket → `/register`.
   - Tabel komparasi → tautan kontekstual ke sub-halaman fitur (mis. baris "Pindai struk" → `/features/receipt-scan`).
   - Blok FAQ harga → `/faq`.
3. **Dari tiap `/features/{slug}`:**
   - Breadcrumb → `/` dan `/features`.
   - Bagian "Fitur terkait" → 2 sub-halaman lain (mis. `/features/receipt-scan` ↔ `/features/voice-input` ↔ `/features/ai-chat`).
   - CTA → `/pricing` dan `/register`.
4. **Dari `/about` & `/faq`:** tautan ke `/features/*`, `/pricing`, `/contact` sesuai konteks.
5. **Anchor teks deskriptif** — hindari "klik di sini". Contoh: "lihat cara memindai struk".

Aturan kuantitatif: 3–5 tautan internal relevan per ±1000 kata pada halaman panjang (`/pricing`, sub-halaman fitur).

---

## 7. Metadata Layout Marketing

`app/(marketing)/layout.tsx` menetapkan default yang diwariskan halaman anak:

```ts
export const metadata: Metadata = {
  title: { default: "FinTrack — Catat Keuangan Otomatis dengan AI", template: "%s · FinTrack" },
  description:
    "Catat transaksi lewat chat, suara, atau foto struk. FinTrack memahami bahasa Indonesia, menyimpan rapi, dan menganalisis kondisi keuangan Anda.",
  openGraph: { type: "website", siteName: "FinTrack", locale: "id_ID" },
  twitter: { card: "summary_large_image" },
};
```

Setiap halaman mendefinisikan `title`, `description`, dan `alternates.canonical` sendiri (dokumen 07). `canonical` memakai **jalur Inggris** (mis. `/pricing`), sejalan dengan URL publik.

---

## 8. Non-Goals Arsitektur

- Tidak mengubah `app/(dashboard)` atau `app/(auth)` selain metadata `(auth)` bila perlu.
- Tidak mengubah `proxy.ts` (logika tetap di `lib/roles.ts`, satu sumber).
- Tidak memindahkan root layout; tetap `app/layout.tsx`.
- Tidak menambah state global baru (Redux/Zustand) untuk marketing.
- **Tidak** membuat redirect dari URL Indonesia (`/harga`, `/fitur`, …) karena URL tersebut belum pernah tayang di produksi — tidak ada trafik atau peringkat lama yang perlu dipertahankan.
