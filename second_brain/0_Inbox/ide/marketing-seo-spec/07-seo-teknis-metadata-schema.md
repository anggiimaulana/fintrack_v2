# 07 — SEO Teknis, Metadata & Structured Data

Dokumen ini merinci lapisan SEO teknis: metadata, canonical, Open Graph/Twitter, OG image dinamis, `robots.ts`, `sitemap.ts`, JSON-LD, Core Web Vitals, dan kesiapan GEO (generative engine optimization).

> Catatan kejujuran: metrik di dokumen ini adalah **heuristik internal dan praktik baik**, bukan sinyal internal Google. Google tidak mempublikasikan bobot peringkat. Validasi nyata memakai Search Console dan pengukuran pihak pertama.

---

## 1. Metadata Global (`app/layout.tsx`)

Ubah metadata akar agar menyediakan basis URL dan default konsisten.

```ts
import type { Metadata } from "next";

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3001";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "FinTrack — Catat Keuangan Otomatis dengan AI",
    template: "%s · FinTrack",
  },
  description:
    "FinTrack mencatat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia yang menyimpan rapi dan menganalisis kondisi keuangan Anda.",
  applicationName: "FinTrack",
  keywords: [
    "aplikasi pencatat keuangan",
    "catat keuangan otomatis",
    "catat pengeluaran",
    "scan struk otomatis",
    "analisis keuangan pribadi",
    "aplikasi keuangan AI",
  ],
  openGraph: {
    type: "website",
    locale: "id_ID",
    siteName: "FinTrack",
    url: siteUrl,
    title: "FinTrack — Catat Keuangan Otomatis dengan AI",
    description:
      "Catat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia.",
  },
  twitter: {
    card: "summary_large_image",
    title: "FinTrack — Catat Keuangan Otomatis dengan AI",
    description:
      "Catat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia.",
  },
  alternates: { canonical: "/" },
  robots: { index: true, follow: true },
};
```

**Perubahan penting dari baseline:** `metadataBase` (wajib agar canonical/OG absolut benar), `title.template`, OG/Twitter default.

## 2. Metadata Per Halaman

Setiap halaman mendefinisikan `title`, `description`, dan `alternates.canonical` unik.

| Rute | title | canonical |
|------|-------|-----------|
| `/` | (default) | `/` |
| `/harga` | Harga FinTrack — Paket Gratis, Pro, dan Business | `/harga` |
| `/fitur` | Fitur FinTrack — AI, OCR Struk, Suara, Laporan | `/fitur` |
| `/fitur/{slug}` | {Judul} — FinTrack | `/fitur/{slug}` |
| `/tentang` | Tentang FinTrack — Misi & Prinsip | `/tentang` |
| `/kontak` | Kontak FinTrack — Bantuan & Pertanyaan | `/kontak` |
| `/faq` | Pertanyaan Umum (FAQ) — FinTrack | `/faq` |
| `/privasi` | Kebijakan Privasi — FinTrack | `/privasi` |
| `/syarat` | Syarat & Ketentuan — FinTrack | `/syarat` |

Aturan:

- Panjang `title` ≤ ~60 karakter; `description` ~120–160 karakter.
- Satu `H1` per halaman, selaras dengan `title`.
- `canonical` selalu absolut setelah `metadataBase` diset (Next menyelesaikan relatif terhadap `metadataBase`).

## 3. Open Graph Image Dinamis

Buat `app/opengraph-image.tsx` (dan `twitter-image.tsx` bila perlu) memakai `ImageResponse` dari `next/og`, sehingga tidak perlu aset desain statis.

```tsx
import { ImageResponse } from "next/og";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "FinTrack — Catat Keuangan Otomatis dengan AI";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div style={{ /* neo-brutalist: latar gelap, border tebal, aksen primer */ }}>
        <div style={{ fontSize: 64, fontWeight: 900, color: "#fff" }}>
          Catat Keuangan Otomatis dengan AI
        </div>
        <div style={{ fontSize: 28, color: "#a3a3a3" }}>
          Chat, suara, atau foto struk — FinTrack
        </div>
      </div>
    ),
    { ...size },
  );
}
```

- Warna & gaya mengikuti token neo-brutalist (`globals.css`).
- Teks harus memuat kata kunci utama.
- Halaman `/harga` boleh punya OG image khusus ("Paket mulai gratis").

## 4. `robots.ts`

```ts
import type { MetadataRoute } from "next";

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3001";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: ["/api/", "/user", "/admin", "/super-admin", "/login"],
      },
    ],
    sitemap: `${siteUrl}/sitemap.xml`,
    host: siteUrl,
  };
}
```

- Rute dashboard & auth **tidak** diindeks (halaman privat).
- `/register` boleh diindeks (titik masuk konversi) — jangan masukkan ke `disallow`.
- Tidak memblokir crawler AI (Googlebot, Bingbot, GPTBot, PerplexityBot, dsb.) pada halaman publik; pemblokiran akan mematikan peluang sitasi AI.

## 5. `sitemap.ts`

```ts
import type { MetadataRoute } from "next";
import { FEATURES } from "@/lib/marketing/features";

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3001";

export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date();
  const staticRoutes: MetadataRoute.Sitemap = [
    { url: `${siteUrl}/`, changeFrequency: "weekly", priority: 1.0, lastModified: now },
    { url: `${siteUrl}/harga`, changeFrequency: "weekly", priority: 0.9, lastModified: now },
    { url: `${siteUrl}/fitur`, changeFrequency: "monthly", priority: 0.8, lastModified: now },
    { url: `${siteUrl}/faq`, changeFrequency: "monthly", priority: 0.7, lastModified: now },
    { url: `${siteUrl}/tentang`, changeFrequency: "yearly", priority: 0.5, lastModified: now },
    { url: `${siteUrl}/kontak`, changeFrequency: "yearly", priority: 0.5, lastModified: now },
    { url: `${siteUrl}/privasi`, changeFrequency: "yearly", priority: 0.3, lastModified: now },
    { url: `${siteUrl}/syarat`, changeFrequency: "yearly", priority: 0.3, lastModified: now },
  ];
  const featureRoutes: MetadataRoute.Sitemap = FEATURES.map((f) => ({
    url: `${siteUrl}/fitur/${f.slug}`,
    changeFrequency: "monthly",
    priority: 0.7,
    lastModified: now,
  }));
  return [...staticRoutes, ...featureRoutes];
}
```

- Fase 2 menambahkan rute `/blog` dan `/blog/{slug}` ke sitemap ini.
- Tidak memasukkan rute privat.

## 6. Structured Data (JSON-LD)

Komponen `JsonLd` (server) merender `<script type="application/ld+json">` dengan `JSON.stringify` (aman, tanpa `dangerouslySetInnerHTML` mentah — gunakan `JSON.stringify` yang sudah meloloskan karakter berbahaya).

### 6.1 `Organization` + `WebSite` (root layout, sekali)

```json
{
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "Organization",
      "@id": "https://<domain>/#organization",
      "name": "FinTrack",
      "url": "https://<domain>/",
      "logo": "https://<domain>/logo.png",
      "contactPoint": [{
        "@type": "ContactPoint",
        "contactType": "customer support",
        "email": "halo@fintrack.id",
        "availableLanguage": ["id"]
      }]
    },
    {
      "@type": "WebSite",
      "@id": "https://<domain>/#website",
      "url": "https://<domain>/",
      "name": "FinTrack",
      "inLanguage": "id-ID",
      "publisher": { "@id": "https://<domain>/#organization" }
    }
  ]
}
```

> Jangan menambah `SearchAction` bila situs tidak punya pencarian internal.
>
> **Aset `logo` wajib ada.** Google mengharapkan logo raster yang valid. Tambahkan `public/logo.png` (mis. 512×512) sebagai bagian dari Fase 6; sampai aset tersedia, `Organization.logo` harus menunjuk ke berkas yang benar-benar ada agar tidak menjadi URL 404. Alternatif tanpa aset statis: buat `app/icon.tsx` (ImageResponse) untuk ikon situs.

### 6.2 `Product` + `Offer` (`/harga`)

```json
{
  "@context": "https://schema.org",
  "@type": "Product",
  "name": "FinTrack",
  "description": "Aplikasi pencatat keuangan otomatis dengan AI berbahasa Indonesia.",
  "brand": { "@type": "Brand", "name": "FinTrack" },
  "offers": [
    { "@type": "Offer", "name": "Free", "price": "0", "priceCurrency": "IDR", "url": "https://<domain>/harga", "availability": "https://schema.org/InStock" },
    { "@type": "Offer", "name": "Pro", "price": "39000", "priceCurrency": "IDR", "url": "https://<domain>/harga", "availability": "https://schema.org/InStock" },
    { "@type": "Offer", "name": "Business", "price": "129000", "priceCurrency": "IDR", "url": "https://<domain>/harga", "availability": "https://schema.org/InStock" }
  ]
}
```

- Harga **dibangkitkan dari data endpoint** (bukan angka hardcode) agar selalu konsisten.
- **Kebijakan pajak (eksplisit & final):** `price` pada `Offer` = **subtotal, belum termasuk pajak** — selaras dengan tampilan kartu di `/harga` (dokumen 04 §A.2). Bila `tax_enabled`, sebutkan di `description` bahwa pajak ditambahkan saat checkout; jangan memasukkan pajak ke `price`.

### 6.3 `BreadcrumbList` (halaman anak)

Dibangkitkan `Breadcrumbs` dari `siteUrl` + label, mis. `/fitur/scan-struk`:
Beranda › Fitur › Pindai Struk Otomatis.

### 6.4 `AboutPage`, `ContactPage`, `WebPage`

- `/tentang`: `AboutPage` dengan `mainEntity` → `Organization`.
- `/kontak`: `ContactPage` + `ContactPoint`.
- `/faq`, `/privasi`, `/syarat`: `WebPage` sederhana dengan `inLanguage: "id-ID"`.

### 6.5 Yang **tidak** dipakai

- **`FAQPage`** — Google telah menghentikan FAQ rich result (lihat skill seo-content). Konten Q&A tetap dirender sebagai teks agar dapat dikutip mesin AI.
- `AggregateRating`/`Review` — **dilarang** sampai ada ulasan asli (testimoni fiktif sudah dihapus; menandai rating palsu melanggar pedoman).

## 7. Core Web Vitals

| Metrik | Target | Strategi |
|--------|--------|----------|
| LCP | < 2.5s | Hero tanpa gambar raster besar; teks H1 sebagai elemen LCP; font sudah via `next/font` (self-hosted, tanpa layout shift). |
| CLS | < 0.1 | Ruang kartu harga dipesan (skeleton/min-height) saat data ISR termuat; tanpa iklan/embed yang menyisipkan konten. |
| INP | < 200ms | JS klien minimal (header, akordeon, toggle harga); tanpa library animasi. |
| TTFB | rendah | Halaman statis/ISR disajikan dari cache Next. |

Praktik:

- Server Component sebagai default; `"use client"` hanya di daun interaktif.
- Tanpa dependensi runtime baru (tidak ada `@next/mdx`, tidak ada library UI tambahan).
- Gambar (bila ada di masa depan) memakai `next/image` dengan `width`/`height` eksplisit.

## 8. Rendering & Caching

| Rute | Mode | Alasan |
|------|------|--------|
| `/`, `/fitur`, `/fitur/{slug}`, `/tentang`, `/kontak`, `/faq`, `/privasi`, `/syarat` | Statis (SSG) | Konten stabil, tanpa data pengguna. Sub-fitur memakai `generateStaticParams`. |
| `/harga` | ISR (`revalidate: 3600`) | Bergantung katalog paket dari backend. |
| Landing (teaser harga) | ISR (`revalidate: 3600`) | Sama seperti `/harga`. |

`dynamicParams = false` pada `/fitur/[slug]` agar slug tak dikenal menghasilkan 404, bukan halaman kosong.

## 9. Kesiapan GEO / AI Search

Berpedoman pada panduan Google: mengoptimalkan untuk fitur generatif sama dengan mengoptimalkan SEO; tidak diperlukan berkas/markup khusus AI.

Yang diterapkan:

- **Answer-first** — jawaban ringkas dan langsung pada FAQ dan deskripsi fitur.
- **Heading bersih** — H1→H2→H3 tanpa lompatan, agar bagian mudah diekstrak.
- **Pernyataan dapat dikutip** — kalimat definisi jelas ("FinTrack adalah aplikasi pencatat keuangan …").
- **Entitas jelas** — `Organization`, nama produk, dan istilah fitur didefinisikan dengan label resmi.
- **Data pihak pertama** — harga dan fitur dari katalog nyata, bukan klaim.
- **Tabel & daftar** untuk data komparatif (tabel harga).
- **Crawler AI tidak diblokir** pada halaman publik.

## 10. Hal yang Dihindari (SEO Black-hat / Risk)

| Praktik | Status |
|---------|--------|
| Keyword stuffing | Dilarang |
| Cloaking (konten berbeda untuk bot) | Dilarang |
| Testimoni/rating palsu + `Review` schema | Dilarang (sudah dihapus) |
| Link farm / spam komentar | Dilarang |
| Duplikasi konten antar sub-halaman fitur | Dilarang (tiap halaman wajib unik) |
| Halaman tipis demi volume | Dilarang (sub-halaman ≥ 300 kata bermakna) |
| `noindex` pada halaman legal | Dilarang (legal harus terindeks demi trust) |

## 11. Verifikasi SEO

- `curl -s http://localhost:3001/robots.txt` → memuat aturan + `Sitemap:`.
- `curl -s http://localhost:3001/sitemap.xml` → seluruh rute publik ada.
- Periksa `<link rel="canonical">`, `<meta property="og:*">`, dan `<script type="application/ld+json">` pada HTML sumber (View Source) tiap halaman — harus ada sebelum JS berjalan.
- Validasi JSON-LD dengan Rich Results Test / Schema Markup Validator (opsional, manual).
