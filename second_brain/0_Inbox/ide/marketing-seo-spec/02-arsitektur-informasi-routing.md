# 02 — Arsitektur Informasi & Routing

Dokumen ini menetapkan peta situs, hierarki URL, struktur berkas Next.js, perbaikan guard auth, dan strategi internal linking untuk situs marketing Fase 1.

---

## 1. Peta Situs (Information Architecture)

```
/                         Landing (perombakan)
├── /harga                Paket, komparasi fitur & limit, FAQ harga
├── /fitur                Hub fitur (ringkasan 6 fitur)
│   ├── /fitur/chat-ai            Asisten AI percakapan
│   ├── /fitur/input-suara        Catat via suara
│   ├── /fitur/scan-struk         OCR foto struk
│   ├── /fitur/dashboard          Dashboard & grafik
│   ├── /fitur/analisis-spk       Skor & status kesehatan finansial
│   └── /fitur/laporan            Laporan & ekspor data
├── /tentang              Misi, cara kerja, perusahaan
├── /kontak               Kontak (mailto/WhatsApp) + jam operasional
├── /faq                  Tanya jawab terkelompok
├── /privasi              Kebijakan Privasi (UU PDP)
├── /syarat               Syarat & Ketentuan
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

Prinsip: **kedalaman maksimum 2 tingkat** dari root. Semua halaman dapat dijangkau ≤ 2 klik dari beranda (silakan lihat §5).

---

## 2. Hierarki URL & Alasan

| Rute | Tipe | Alasan pemilihan URL |
|------|------|----------------------|
| `/` | Statis | Beranda. |
| `/harga` | ISR | Kata "harga" lebih alami dicari pengguna Indonesia daripada "pricing". |
| `/fitur` | Statis | Hub; satu kata, mudah diingat. |
| `/fitur/{slug}` | SSG (`generateStaticParams`) | Slug deskriptif berbahasa Indonesia (`/fitur/scan-struk`) menargetkan long-tail intent. |
| `/tentang` | Statis | Halaman trust/company. |
| `/kontak` | Statis | Halaman trust + kanal dukungan. |
| `/faq` | Statis | Akronim yang lazim dicari di Indonesia. |
| `/privasi` | Statis | Ejaan Indonesia yang umum. |
| `/syarat` | Statis | Ringkas; hindari `/terms-and-conditions` yang Inggris. |

Semua slug huruf kecil, dipisah tanda hubung, tanpa akhiran `.html`, tanpa parameter kueri untuk konten utama.

---

## 3. Struktur Berkas Next.js (Route Group)

Landing yang sekarang berada di `frontend/app/page.tsx` **dipindahkan** ke `app/(marketing)/page.tsx`. Route group `(...)` **tidak mengubah URL**, sehingga `/` tetap `/`. Root `app/layout.tsx` tetap sebagai layout akar.

```
frontend/app/
├── layout.tsx                     # root (diubah: metadataBase, template, OG default)
├── globals.css                    # (tidak diubah)
├── providers.tsx                  # (tidak diubah)
├── sitemap.ts                     # BARU
├── robots.ts                      # BARU
├── opengraph-image.tsx            # BARU (OG image dinamis)
├── (marketing)/
│   ├── layout.tsx                 # BARU — MarketingHeader + MarketingFooter
│   ├── page.tsx                   # DIPINDAH dari app/page.tsx + dirombak
│   ├── harga/page.tsx             # BARU
│   ├── fitur/
│   │   ├── page.tsx               # BARU
│   │   └── [slug]/page.tsx        # BARU
│   ├── tentang/page.tsx           # BARU
│   ├── kontak/page.tsx            # BARU
│   ├── faq/page.tsx               # BARU
│   ├── privasi/page.tsx           # BARU
│   └── syarat/page.tsx            # BARU
├── (auth)/                        # (tidak diubah)
└── (dashboard)/                   # (tidak diubah)

frontend/components/marketing/     # BARU
├── MarketingHeader.tsx            # client (menu mobile)
├── MarketingFooter.tsx            # server
├── SectionHeading.tsx             # server
├── Hero.tsx                       # server
├── ProductMockup.tsx              # server (dipindah dari landing lama)
├── TrustBar.tsx                   # server
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
├── LegalPage.tsx                  # server (render privasi & syarat)
└── JsonLd.tsx                     # server

frontend/lib/marketing/            # BARU
├── site.ts                        # brand, URL dasar, kontak, kanal
├── features.ts                    # data 6 fitur + isi sub-halaman
├── faq.ts                         # Q&A per kategori
└── legal.ts                       # isi privasi & syarat
frontend/lib/
├── plans-server.ts                # BARU — fetch katalog publik (server-only)
└── seo.ts                         # BARU — helper metadata & JSON-LD
frontend/app/(marketing)/opengraph-image.tsx  # opsional per-halaman (bila perlu)
```

### Aturan batas modul

- `lib/plans-server.ts` **wajib server-only** (memakai `process.env.BACKEND_ORIGIN`). Tidak boleh diimpor oleh komponen client. Beri komentar tegas di kepala berkas.
- `components/marketing/PricingCards.tsx` dan `FaqAccordion.tsx` satu-satunya komponen marketing ber-`"use client"`. Perilaku interaktif harus diisolasi ke daun terkecil.
- `lib/marketing/*` adalah data murni bertipe (tanpa JSX), sehingga dapat diuji unit tanpa DOM.

---

## 4. Perbaikan Guard Auth (`lib/roles.ts`) — KRITIS

### Kondisi sekarang (F-07)

```ts
const publicPath = pathname === '/login' || pathname === '/' || pathname === '/register';
```

### Perubahan yang diminta

Ganti dengan **allowlist prefix** yang mencakup permukaan publik. Pertahankan perilaku rute dashboard persis seperti sekarang.

```ts
/**
 * Prefix yang boleh diakses tanpa sesi. Dipakai untuk memutuskan apakah
 * pengunjung anonim diarahkan ke /login atau dibiarkan lewat.
 * Daftar ini mencerminkan permukaan marketing publik (lihat spec Fase 1).
 */
const PUBLIC_PREFIXES = [
  '/harga',
  '/fitur',
  '/tentang',
  '/kontak',
  '/faq',
  '/privasi',
  '/syarat',
] as const;

function isPublicPath(pathname: string): boolean {
  // Halaman akar & auth persis.
  if (pathname === '/' || pathname === '/login' || pathname === '/register') return true;
  // Prefix marketing (cocok untuk '/harga' maupun '/fitur/scan-struk').
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
| `/harga` | tidak | — | `pass` |
| `/fitur/scan-struk` | tidak | — | `pass` |
| `/login` | tidak | — | `pass` |
| `/privasi` | tidak | — | `pass` |
| `/user` | tidak | — | `redirect-login` |
| `/admin` | tidak | — | `redirect-login` |
| `/harga` | ya | ya | `pass` |
| `/login` | ya | ya | `{ redirect: homePathForRole(role) }` |
| `/`,`/harga` | ya | tidak | `pass` (biar klien refresh) |

> Catatan penting: prefix harus dibandingkan sebagai **segmen penuh** (`=== p` atau `startsWith(p + '/')`), bukan `startsWith(p)`, agar `/hargax` atau `/fiturx` tidak ikut lolos.

### Dampak pada tes

`frontend/lib/roles.ts` punya tes (dipakai bersama proxy). Tes harus ditambah:
- anonim → tiap path publik = `pass`;
- anonim → `/hargax` dan `/fiturx` = `redirect-login` (regresi segmentasi);
- sesi valid di `/login` tetap bounce ke home per peran;
- seluruh perilaku `/user`, `/admin`, `/super-admin` tidak berubah.

### Pertimbangan keamanan

Membuka rute marketing ke anonim **tidak** memaparkan data sensitif: halaman ini hanya konten statis + harga publik dari endpoint publik baru (dokumen 06). Tidak ada query ber-scope pengguna. Endpoint `/api/v1/plans` yang lama **tetap** di balik autentikasi (tidak diubah).

---

## 5. Navigasi & Footer

### MarketingHeader (komponen client)

- Logo `FinTrack` (ikon `Wallet`, tanpa emoji) → `/`.
- Nav desktop: **Fitur** (`/fitur`), **Cara Kerja** (`/#cara-kerja`), **Harga** (`/harga`), **FAQ** (`/faq`).
- Aksi: **Masuk** (`/login`) — `variant="outline"`; **Mulai Gratis** (`/register`) — `variant="default"`.
- Mobile (< `md`): tombol hamburger dengan `aria-expanded`, `aria-controls`, dan `aria-label="Buka menu"`. Panel menu menampilkan tautan + aksi; menutup saat tautan dipilih, saat `Escape`, dan saat fokus keluar. Mencegah scroll latar saat terbuka.
- Sticky `top-0`, `border-b-2 border-foreground`, latar `bg-card`.

### MarketingFooter (komponen server)

Empat kolom tautan nyata (tidak ada `#` hampa kecuali anchor yang benar-benar ada):

| Kolom | Tautan |
|-------|--------|
| Produk | Fitur, Harga, Cara Kerja (`/#cara-kerja`), FAQ |
| Perusahaan | Tentang, Kontak |
| Sumber Daya | Blog (Fase 2 — tampil disabled/"Segera hadir" sampai ada), Panduan (→ `/fitur`), Status (opsional) |
| Legal | Kebijakan Privasi, Syarat & Ketentuan |

Baris bawah: hak cipta + tagline. Footer juga memuat JSON-LD `Organization` (dokumen 07).

### Alur navigasi (klik dari beranda)

| Tujuan | Klik dari `/` |
|--------|---------------|
| `/harga` | 1 (nav atau teaser harga) |
| `/fitur` | 1 |
| `/fitur/scan-struk` | 2 |
| `/faq` | 1 |
| `/tentang` | 2 (footer) |
| `/kontak` | 2 (footer) |
| `/privasi` | 2 (footer) |
| `/syarat` | 2 (footer) |

---

## 6. Strategi Internal Linking

Tujuan: menyebarkan otoritas, memudahkan penemuan halaman, dan memberi konteks ke mesin pencari.

1. **Dari landing:**
   - Hero → `/register`; "Lihat Cara Kerja" → `#cara-kerja`.
   - Kartu fitur unggulan → masing-masing menautkan ke sub-halaman `/fitur/{slug}` yang relevan.
   - Teaser harga → `/harga`.
   - FAQ landing (6 item) + tautan "Lihat semua FAQ" → `/faq`.
   - Footer → seluruh halaman.
2. **Dari `/harga`:**
   - Setiap kartu paket → `/register`.
   - Tabel komparasi → tautan kontekstual ke sub-halaman fitur (mis. baris "Pindai struk" → `/fitur/scan-struk`).
   - Blok FAQ harga → `/faq`.
3. **Dari tiap `/fitur/{slug}`:**
   - Breadcrumb → `/` dan `/fitur`.
   - Bagian "Fitur terkait" → 2 sub-halaman lain (mis. scan-struk ↔ input-suara ↔ chat-ai).
   - CTA → `/harga` dan `/register`.
4. **Dari `/tentang` & `/faq`:** tautan ke `/fitur/*`, `/harga`, `/kontak` sesuai konteks.
5. **Anchor teks deskriptif** — hindari "klik di sini". Contoh: "lihat cara memindai struk".

Aturan kuantitatif: 3–5 tautan internal relevan per ±1000 kata pada halaman panjang (`/harga`, sub-halaman fitur).

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

Setiap halaman mendefinisikan `title`, `description`, dan `alternates.canonical` sendiri (dokumen 07).

---

## 8. Non-Goals Arsitektur

- Tidak mengubah `app/(dashboard)` atau `app/(auth)` selain metadata `(auth)` bila perlu.
- Tidak mengubah `proxy.ts` (logika tetap di `lib/roles.ts`, satu sumber).
- Tidak memindahkan root layout; tetap `app/layout.tsx`.
- Tidak menambah state global baru (Redux/Zustand) untuk marketing.
