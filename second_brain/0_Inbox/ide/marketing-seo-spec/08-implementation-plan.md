# 08 — Rencana Implementasi

Dokumen ini menetapkan urutan eksekusi Fase 1 dalam 7 fase, inventaris berkas yang dibuat/diubah, matriks dependensi, dan kriteria penerimaan per fase. Urutan dirancang agar **guard auth diperbaiki lebih dulu** (tanpa itu, halaman baru tidak dapat diakses) dan agar tiap fase meninggalkan repo dalam keadaan hijau.

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

**Kriteria fase:** `go test ./...` hijau; `pnpm test` hijau; anonim mampu membuka `/harga` (setelah halaman ada) dan tetap diblokir dari `/user`.

### Fase 1 — Struktur Marketing & Data Konten

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 1.1 | Route group + layout marketing | `frontend/app/(marketing)/layout.tsx` | 0.1 |
| 1.2 | Data situs (brand, kontak, sosial) | `frontend/lib/marketing/site.ts` | — |
| 1.3 | Data fitur (6 item + isi sub-halaman) | `frontend/lib/marketing/features.ts` | — |
| 1.4 | Data FAQ terkelompok | `frontend/lib/marketing/faq.ts` | — |
| 1.5 | Data legal (privasi & syarat) | `frontend/lib/marketing/legal.ts` | — |
| 1.6 | Helper SEO (metadata + JSON-LD builders) | `frontend/lib/seo.ts` | 1.2 |
| 1.7 | Komponen `JsonLd`, `Breadcrumbs`, `SectionHeading` | `frontend/components/marketing/*` | 1.6 |
| 1.8 | Komponen header + footer | `frontend/components/marketing/MarketingHeader.tsx`, `MarketingFooter.tsx` | 1.2 |
| 1.9 | Unit test data (kelengkapan fitur/FAQ) | `frontend/lib/marketing/*.test.ts` | 1.3-1.5 |

**Kriteria fase:** layout marketing merender header+footer; `pnpm test` hijau.

### Fase 2 — Landing Page Dirombak

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 2.1 | Pindahkan landing ke group marketing | `frontend/app/(marketing)/page.tsx` (hapus `frontend/app/page.tsx`) | 1.1 |
| 2.2 | `Hero` + `ProductMockup` (copy dikoreksi) | `frontend/components/marketing/Hero.tsx`, `ProductMockup.tsx` | 2.1 |
| 2.3 | `TrustBar` (klaim faktual) | `frontend/components/marketing/TrustBar.tsx` | 2.1 |
| 2.4 | `FeatureGrid` + section Masalah→Solusi | `frontend/components/marketing/FeatureGrid.tsx` | 1.3 |
| 2.5 | `HowItWorks` (hapus langkah API key) | `frontend/components/marketing/HowItWorks.tsx` | 2.1 |
| 2.6 | `SpkHighlight` | `frontend/components/marketing/SpkHighlight.tsx` | 2.1 |
| 2.7 | `PricingTeaser` (konsumsi `fetchPublicPlans`) | `frontend/components/marketing/PricingTeaser.tsx` | 0.6, 2.1 |
| 2.8 | `PersonaUseCases` (ganti testimoni) | `frontend/components/marketing/PersonaUseCases.tsx` | 2.1 |
| 2.9 | `FaqAccordion` + `CtaSection` | `frontend/components/marketing/*` | 1.4 |
| 2.10 | Pasang metadata landing + `#harga`/`#cara-kerja` | `frontend/app/(marketing)/page.tsx` | 1.6 |
| 2.11 | DOM test + a11y test komponen landing | `frontend/components/marketing/*.dom.test.tsx`, `*.a11y.test.tsx` | 2.1-2.10 |

**Kriteria fase:** tidak ada anchor mati; BYOK & testimoni fiktif hilang; a11y hijau.

### Fase 3 — Halaman `/harga`

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 3.1 | `PricingCards` (client, toggle siklus) | `frontend/components/marketing/PricingCards.tsx` | 0.6 |
| 3.2 | `ComparisonTable` (fitur + limit enforced) | `frontend/components/marketing/ComparisonTable.tsx` | 0.6 |
| 3.3 | Halaman `/harga` + FAQ + schema `Product`/`Offer` | `frontend/app/(marketing)/harga/page.tsx` | 3.1, 3.2, 1.6 |
| 3.4 | Penanganan fallback katalog gagal | `frontend/app/(marketing)/harga/page.tsx` | 3.3 |
| 3.5 | DOM test `PricingCards` (toggle) + `ComparisonTable` | `frontend/components/marketing/*.dom.test.tsx` | 3.1, 3.2 |
| 3.6 | A11y test `/harga` | `frontend/components/marketing/*.a11y.test.tsx` | 3.1 |

**Kriteria fase:** harga tampil dari endpoint; "Tanpa batas" untuk `-1`; `bank_sync`/limit tak enforced tidak muncul.

### Fase 4 — Halaman `/fitur` & Sub-halaman

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 4.1 | Hub `/fitur` | `frontend/app/(marketing)/fitur/page.tsx` | 1.3 |
| 4.2 | Sub-halaman dinamis + `generateStaticParams` | `frontend/app/(marketing)/fitur/[slug]/page.tsx` | 1.3, 1.6 |
| 4.3 | Isi 6 sub-halaman (≥300 kata unik masing-masing) | `frontend/lib/marketing/features.ts` | 4.2 |
| 4.4 | Blok "Fitur terkait" + badge paket | `frontend/app/(marketing)/fitur/[slug]/page.tsx` | 4.2 |
| 4.5 | DOM test hub & sub-halaman | `frontend/app/(marketing)/fitur/*.dom.test.tsx` | 4.1, 4.2 |

**Kriteria fase:** 6 slug valid, slug tak dikenal 404, tiap halaman punya konten unik.

### Fase 5 — Halaman Pendukung

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 5.1 | `/tentang` | `frontend/app/(marketing)/tentang/page.tsx` | 1.6 |
| 5.2 | `/kontak` (mailto/wa.me) | `frontend/app/(marketing)/kontak/page.tsx` | 1.2 |
| 5.3 | `/faq` | `frontend/app/(marketing)/faq/page.tsx` | 1.4 |
| 5.4 | Komponen `LegalPage` + `/privasi` + `/syarat` | `frontend/app/(marketing)/privasi|syarat/page.tsx` | 1.5 |
| 5.5 | A11y/DOM test halaman pendukung | `frontend/app/(marketing)/**/*.a11y.test.tsx` | 5.1-5.4 |

**Kriteria fase:** tidak ada `href="#"`; konten legal ada di HTML server.

### Fase 6 — Infrastruktur SEO

| # | Task | Berkas | Dependensi |
|---|------|--------|-----------|
| 6.1 | `metadataBase` + template + OG/Twitter di root | `frontend/app/layout.tsx` | — |
| 6.2 | `robots.ts` | `frontend/app/robots.ts` | 1.2 |
| 6.3 | `sitemap.ts` | `frontend/app/sitemap.ts` | 1.3 |
| 6.4 | OG image dinamis | `frontend/app/opengraph-image.tsx` | 6.1 |
| 6.5 | JSON-LD `Organization`/`WebSite` di layout marketing | `frontend/app/(marketing)/layout.tsx` | 1.7 |
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
| 7.6 | Perbarui `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md` & HUD roadmap | 7.1-7.5 |

---

## 2. Inventaris Berkas

### 2.1 Dibuat (baru)

**Frontend**
- `app/(marketing)/layout.tsx`, `page.tsx`
- `app/(marketing)/harga/page.tsx`
- `app/(marketing)/fitur/page.tsx`, `app/(marketing)/fitur/[slug]/page.tsx`
- `app/(marketing)/tentang/page.tsx`, `kontak/page.tsx`, `faq/page.tsx`, `privasi/page.tsx`, `syarat/page.tsx`
- `app/robots.ts`, `app/sitemap.ts`, `app/opengraph-image.tsx`
- `components/marketing/*` (Header, Footer, Hero, ProductMockup, TrustBar, FeatureGrid, HowItWorks, SpkHighlight, PricingTeaser, PricingCards, ComparisonTable, PersonaUseCases, FaqAccordion, CtaSection, Breadcrumbs, JsonLd, SectionHeading, LegalPage)
- `lib/marketing/{site,features,faq,legal}.ts`
- `lib/plans-server.ts`, `lib/plans-types.ts`, `lib/seo.ts`
- Berkas tes pasangan (`*.test.ts`, `*.dom.test.tsx`, `*.a11y.test.tsx`)

### 2.2 Diubah

- `frontend/lib/roles.ts` (allowlist prefix publik)
- `frontend/app/layout.tsx` (metadata base/template/OG)
- `frontend/hooks/use-billing.ts` (impor tipe bersama — opsional, non-perilaku)
- `backend/internal/delivery/http/router.go` (rute publik + rate limit)
- Tes guard auth yang ada (kasus baru)
- `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`, HUD `roadmap.md`

### 2.3 Dihapus

- `frontend/app/page.tsx` (dipindahkan ke `app/(marketing)/page.tsx`)

### 2.4 Tidak disentuh

- `frontend/proxy.ts`, `app/(auth)/*`, `app/(dashboard)/*`, `next.config.ts`, `globals.css`, seluruh `backend/` selain `router.go`.

---

## 3. Urutan Pengerjaan yang Direkomendasikan

```text
Fase 0  (guard + endpoint + klien data)   ← wajib lebih dulu
  ↓
Fase 1  (struktur + data konten)
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
| 1 | Rendah | Rendah | Data + komponen dasar |
| 2 | Sedang | Rendah | Banyak komponen, tapi terisolasi |
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
- [ ] `robots.txt`, `sitemap.xml`, canonical, OG, dan JSON-LD ada.
- [ ] `go vet ./... && go test ./... && go build ./...` hijau.
- [ ] `pnpm lint` 0 error; `pnpm test` hijau; `pnpm build` hijau.
- [ ] Semua komponen baru lulus `axe-core`.
- [ ] `log_aktivitas.md` & HUD roadmap diperbarui.
