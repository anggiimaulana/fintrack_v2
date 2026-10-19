# 08 — Rencana Implementasi

Dokumen ini menetapkan urutan eksekusi Fase 1 dalam 7 fase, inventaris berkas yang dibuat/diubah, matriks dependensi, dan kriteria penerimaan per fase. Urutan dirancang agar **guard auth diperbaiki lebih dulu** (tanpa itu, halaman baru tidak dapat diakses) dan agar tiap fase meninggalkan repo dalam keadaan hijau.

> Rute memakai **URL berbahasa Inggris**; konten berbahasa Indonesia. Gaya visual mengikuti **design system neobrutalism** di dokumen 03.

---

## 1. Fase Implementasi

### Fase 0 — Fondasi Routing & Data (prasyarat)

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 0.1 | Perbaiki `decideRoute` → allowlist prefix publik | `frontend/lib/roles.ts` | — |
| 0.2 | Perbarui & tambah tes guard auth | `frontend/lib/roles.test.ts` (atau berkas tes yang ada) | 0.1 |
| 0.3 | Tambah rute publik katalog | `backend/internal/delivery/http/router.go` | — |
| 0.4 | Tambah rate-limit eksplisit untuk rute publik | `backend/internal/delivery/http/router.go` | 0.3 |
| 0.5 | Tes rute publik (anonim 200, `/plans` tetap 401) | `backend/tests/…` atau tes router | 0.3 |
| 0.6 | Buat klien server `fetchPublicPlans` + `enforcedLimits` | `frontend/lib/plans-server.ts` | 0.3 |
| 0.7 | Unit test `plans-server` | `frontend/lib/plans-server.test.ts` | 0.6 |
| 0.8 | Satukan tipe `PublicPlan` bersama | `frontend/lib/plans-types.ts`, `frontend/hooks/use-billing.ts` | 0.6 |

**Kriteria fase:** `go test ./...` hijau; `pnpm test` hijau; anonim mampu membuka `/pricing` (setelah halaman ada) dan tetap diblokir dari `/user`.

### Fase 1 — Struktur Marketing, Design System & Data Konten

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 1.1 | Route group + layout marketing | `frontend/app/(marketing)/layout.tsx` | 0.1 |
| 1.2 | Data situs (brand, kontak, sosial) | `frontend/lib/marketing/site.ts` | — |
| 1.3 | Data fitur (6 item + isi sub-halaman) | `frontend/lib/marketing/features.ts` | — |
| 1.4 | Data FAQ terkelompok | `frontend/lib/marketing/faq.ts` | — |
| 1.5 | Data legal (privasi & syarat) | `frontend/lib/marketing/legal.ts` | — |
| 1.6 | Naskah copy landing & CTA | `frontend/lib/marketing/copy.ts` | 1.2 |
| 1.7 | Helper SEO (metadata + JSON-LD builders) | `frontend/lib/seo.ts` | 1.2 |
| 1.8 | Komponen `JsonLd`, `Breadcrumbs`, `SectionHeading` | `frontend/components/marketing/*` | 1.7 |
| 1.9 | Komponen header + footer | `frontend/components/marketing/MarketingHeader.tsx`, `MarketingFooter.tsx` | 1.2 |
| 1.10 | Unit test data (kelengkapan fitur/FAQ/copy) | `frontend/lib/marketing/*.test.ts` | 1.3-1.6 |
| 1.11 | Design system: token aksen & motion + font display heading | `frontend/app/globals.css`, `frontend/app/layout.tsx` | 1.1 |

**Kriteria fase:** layout marketing merender header+footer; token aksen & font display terpasang; `pnpm test` hijau.

### Fase 2 — Landing Page Dirombak

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 2.1 | Pindahkan landing ke group marketing | `frontend/app/(marketing)/page.tsx` (hapus `frontend/app/page.tsx`) | 1.1 |
| 2.2 | `Hero` + `InteractiveDemo` (tab Chat/Suara/Struk, mock data) + copy dikoreksi | `frontend/components/marketing/Hero.tsx`, `InteractiveDemo.tsx` | 2.1, 1.6 |
| 2.3 | Elemen khas: `ReceiptTapeMarquee`, `TickerBand`, `StatCounters`, `Reveal` | `frontend/components/marketing/*` | 1.11, 2.1 |
| 2.4 | `TrustBar` (klaim faktual) | `frontend/components/marketing/TrustBar.tsx` | 2.1 |
| 2.5 | `ProblemSolution` + `FeatureGrid` | `frontend/components/marketing/ProblemSolution.tsx`, `FeatureGrid.tsx` | 1.3 |
| 2.6 | `HowItWorks` (hapus langkah API key) | `frontend/components/marketing/HowItWorks.tsx` | 2.1 |
| 2.7 | `SpkHighlight` | `frontend/components/marketing/SpkHighlight.tsx` | 2.1 |
| 2.8 | `PricingTeaser` (konsumsi `fetchPublicPlans`) | `frontend/components/marketing/PricingTeaser.tsx` | 0.6, 2.1 |
| 2.9 | `PersonaUseCases` (ganti testimoni) | `frontend/components/marketing/PersonaUseCases.tsx` | 2.1 |
| 2.10 | `FaqAccordion` + `CtaSection` | `frontend/components/marketing/*` | 1.4 |
| 2.11 | Pasang metadata landing + anchor `#pricing`/`#how-it-works` | `frontend/app/(marketing)/page.tsx` | 1.7 |
| 2.12 | DOM test + a11y test komponen landing | `frontend/components/marketing/*.dom.test.tsx`, `*.a11y.test.tsx` | 2.1-2.11 |

**Kriteria fase:** tidak ada anchor mati; BYOK & testimoni fiktif hilang; marquee/counter hormat `prefers-reduced-motion`; a11y hijau.

### Fase 3 — Halaman `/pricing`

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 3.1 | `PricingCards` (client, toggle siklus) | `frontend/components/marketing/PricingCards.tsx` | 0.6 |
| 3.2 | `ComparisonTable` (fitur + limit enforced) | `frontend/components/marketing/ComparisonTable.tsx` | 0.6 |
| 3.3 | Halaman `/pricing` + FAQ + schema `Product`/`Offer` | `frontend/app/(marketing)/pricing/page.tsx` | 3.1, 3.2, 1.7 |
| 3.4 | Penanganan fallback katalog gagal | `frontend/app/(marketing)/pricing/page.tsx` | 3.3 |
| 3.5 | DOM test `PricingCards` (toggle) + `ComparisonTable` | `frontend/components/marketing/*.dom.test.tsx` | 3.1, 3.2 |
| 3.6 | A11y test `/pricing` | `frontend/components/marketing/*.a11y.test.tsx` | 3.1 |

**Kriteria fase:** harga tampil dari endpoint; "Tanpa batas" untuk `-1`; `bank_sync`/limit tak enforced tidak muncul.

### Fase 4 — Halaman `/features` & Sub-halaman

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 4.1 | Hub `/features` | `frontend/app/(marketing)/features/page.tsx` | 1.3 |
| 4.2 | Sub-halaman dinamis + `generateStaticParams` | `frontend/app/(marketing)/features/[slug]/page.tsx` | 1.3, 1.7 |
| 4.3 | Isi 6 sub-halaman (≥300 kata unik masing-masing) | `frontend/lib/marketing/features.ts` | 4.2 |
| 4.4 | Blok "Fitur terkait" + badge paket | `frontend/app/(marketing)/features/[slug]/page.tsx` | 4.2 |
| 4.5 | DOM test hub & sub-halaman | `frontend/app/(marketing)/features/*.dom.test.tsx` | 4.1, 4.2 |

**Kriteria fase:** 6 slug valid, slug tak dikenal 404, tiap halaman punya konten unik.

### Fase 5 — Halaman Pendukung

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 5.1 | `/about` | `frontend/app/(marketing)/about/page.tsx` | 1.7 |
| 5.2 | `/contact` (mailto/wa.me) | `frontend/app/(marketing)/contact/page.tsx` | 1.2 |
| 5.3 | `/faq` | `frontend/app/(marketing)/faq/page.tsx` | 1.4 |
| 5.4 | Komponen `LegalPage` + `/privacy` + `/terms` | `frontend/app/(marketing)/privacy/page.tsx`, `terms/page.tsx` | 1.5 |
| 5.5 | A11y/DOM test halaman pendukung | `frontend/app/(marketing)/**/*.a11y.test.tsx` | 5.1-5.4 |

**Kriteria fase:** tidak ada `href="#"`; konten legal ada di HTML server.

### Fase 6 — Infrastruktur SEO

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 6.1 | `metadataBase` + template + OG/Twitter di root | `frontend/app/layout.tsx` | — |
| 6.2 | `robots.ts` | `frontend/app/robots.ts` | 1.2 |
| 6.3 | `sitemap.ts` | `frontend/app/sitemap.ts` | 1.3 |
| 6.4 | OG image dinamis | `frontend/app/opengraph-image.tsx` | 6.1 |
| 6.5 | JSON-LD `Organization`/`WebSite` di layout marketing | `frontend/app/(marketing)/layout.tsx` | 1.8 |
| 6.6 | Unit test `sitemap`/`robots`/JSON-LD builder | `frontend/lib/seo.test.ts`, `frontend/app/sitemap.test.ts` | 6.2-6.5 |
| 6.7 | Aset logo `Organization` (raster, mis. 512×512) | `frontend/public/logo.png` | 6.5 |

**Kriteria fase:** `/robots.txt` & `/sitemap.xml` benar; canonical/OG/JSON-LD ada di HTML sumber.

### Fase 7 — Verifikasi & Pembersihan

| # | Task | Dependensi |
|---|------|-----------|
| 7.1 | `cd backend && go vet ./... && go test ./... && go build ./...` | 0.3-0.5 |
| 7.2 | `cd frontend && pnpm lint` (0 error) | Semua |
| 7.3 | `cd frontend && pnpm test` (semua lulus) | Semua |
| 7.4 | `cd frontend && pnpm build` (hijau) | Semua |
| 7.5 | Smoke test manual: telusuri seluruh rute publik tanpa login | 7.4 |
| 7.6 | Perbarui `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md` | 7.1-7.5 |

---

## 2. Inventaris Berkas

### 2.1 Dibuat (baru)

**Frontend**
- `app/(marketing)/layout.tsx`, `page.tsx`
- `app/(marketing)/pricing/page.tsx`
- `app/(marketing)/features/page.tsx`, `app/(marketing)/features/[slug]/page.tsx`
- `app/(marketing)/about/page.tsx`, `contact/page.tsx`, `faq/page.tsx`, `privacy/page.tsx`, `terms/page.tsx`
- `app/robots.ts`, `app/sitemap.ts`, `app/opengraph-image.tsx`
- `components/marketing/*` (MarketingHeader, MarketingFooter, Hero, InteractiveDemo, ReceiptTapeMarquee, TickerBand, StatCounters, Reveal, TrustBar, ProblemSolution, FeatureGrid, HowItWorks, SpkHighlight, PricingTeaser, PricingCards, ComparisonTable, PersonaUseCases, FaqAccordion, CtaSection, Breadcrumbs, JsonLd, SectionHeading, LegalPage)
- `lib/marketing/{site,features,faq,legal,copy}.ts`
- `lib/plans-server.ts`, `lib/plans-types.ts`, `lib/seo.ts`
- Berkas tes pasangan (`*.test.ts`, `*.dom.test.tsx`, `*.a11y.test.tsx`)

### 2.2 Diubah

- `frontend/lib/roles.ts` (allowlist prefix publik)
- `frontend/app/layout.tsx` (metadata base/template/OG + font display heading)
- `frontend/app/globals.css` (token aksen kuning asam & merah, token motion, kelas `.neo-*` baru)
- `frontend/hooks/use-billing.ts` (impor tipe bersama — opsional, non-perilaku)
- `backend/internal/delivery/http/router.go` (rute publik + rate limit)
- Tes guard auth yang ada (kasus baru)
- `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`

### 2.3 Dihapus

- `frontend/app/page.tsx` (dipindahkan ke `app/(marketing)/page.tsx`)

### 2.4 Tidak disentuh

- `frontend/proxy.ts`, `app/(auth)/*`, `app/(dashboard)/*`, `next.config.ts`, seluruh `backend/` selain `router.go`.

---

## 3. Urutan Pengerjaan yang Direkomendasikan

```text
Fase 0  (guard + endpoint + klien data)   ← wajib lebih dulu
  ↓
Fase 1  (struktur + design system + data konten)
  ↓
Fase 2  (landing)        Fase 3 (harga)   Fase 4 (fitur)   Fase 5 (pendukung)
  ↓                        ↓                ↓                 ↓
Fase 6  (SEO infrastruktur — dapat sebagian dikerjakan paralel)
  ↓
Fase 7  (verifikasi & pembersihan)
```

Fase 3, 4, dan 5 independen satu sama lain setelah Fase 1 → boleh dikerjakan paralel.

---

## 4. Estimasi Kompleksitas

| Fase | Kompleksitas | Risiko | Catatan |
|------|--------------|--------|---------|
| 0 | Sedang | Sedang | Menyentuh guard auth + backend; tes wajib |
| 1 | Rendah–Sedang | Rendah | Data + komponen dasar + design system |
| 2 | Sedang | Rendah | Banyak komponen, tapi terisolasi; animasi terbatas |
| 3 | Sedang | Sedang | Bergantung backend; perlu penanganan fallback |
| 4 | Sedang | Rendah | Volume konten (6 sub-halaman) |
| 5 | Rendah | Rendah | Konten statis + legal |
| 6 | Rendah | Rendah | Konfigurasi SEO |
| 7 | Rendah | Rendah | Verifikasi |

---

## 5. Definition of Done (Ringkas)

- [ ] Semua rute publik dapat dibuka tanpa login (guard diperbaiki).
- [ ] Tidak ada anchor mati / tombol tanpa aksi.
- [ ] Klaim BYOK & testimoni fiktif hilang; copy akurat.
- [ ] Harga & fitur berasal dari endpoint publik; `-1` → "Tanpa batas"; limit tak enforced disembunyikan.
- [ ] Design system neobrutalism terpasang (border 3px, hard shadow 6px, sudut membulat, aksen kuning asam/merah); elemen khas tampil.
- [ ] Animasi hormat `prefers-reduced-motion`; maksimum 2 elemen beranimasi per viewport.
- [ ] `robots.txt`, `sitemap.xml`, canonical, OG, dan JSON-LD ada.
- [ ] `go vet ./... && go test ./... && go build ./...` hijau.
- [ ] `pnpm lint` 0 error; `pnpm test` hijau; `pnpm build` hijau.
- [ ] Semua komponen baru lulus `axe-core`.
- [ ] `log_aktivitas.md` diperbarui.
