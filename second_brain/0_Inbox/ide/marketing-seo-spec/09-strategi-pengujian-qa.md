# 09 — Strategi Pengujian & QA

Dokumen ini menetapkan strategi pengujian berlapis (L1–L5), matriks kasus uji, perintah kanonik, dan Definition of Done untuk Fase 1.

---

## 1. Tingkat Pengujian

| Tingkat | Cakupan | Alat | Wajib? |
|---------|---------|------|--------|
| **L1 Unit (frontend)** | Fungsi murni: data konten, builder metadata/JSON-LD, `plans-server`, `roles` | Vitest (`node` project) | Ya |
| **L2 DOM (frontend)** | Komponen interaktif: toggle harga, akordeon FAQ, menu mobile | Vitest (`dom` project) + Testing Library | Ya |
| **L3 A11y (frontend)** | Komponen & halaman utama | `vitest-axe` / `axe-core` (`*.a11y.test.tsx`) | Ya |
| **L4 Backend (Go)** | Rute publik, regresi `/plans` terproteksi | `go test` | Ya |
| **L5 Manual/E2E** | Telusur seluruh rute tanpa login, metadata & schema di HTML sumber | Browser/`curl` | Ya (checklist) |

> Konfigurasi Vitest punya dua project (`unit` untuk `lib/**` & `hooks/**`, `dom` untuk `*.dom.test.tsx` / `*.a11y.test.tsx`) — lihat `CLAUDE.md`. Susun berkas tes mengikuti konvensi ini.

---

## 2. Matriks Kasus Uji

### 2.1 L1 — Unit Frontend

| Area | Kasus | File |
|------|-------|------|
| `roles.decideRoute` | anonim → `/` = `pass` | `lib/roles.test.ts` |
| `roles.decideRoute` | anonim → tiap prefix publik (`/harga`, `/fitur`, `/fitur/scan-struk`, `/tentang`, `/kontak`, `/faq`, `/privasi`, `/syarat`) = `pass` | `lib/roles.test.ts` |
| `roles.decideRoute` | anonim → `/hargax` / `/fiturx` = `redirect-login` (segmentasi) | `lib/roles.test.ts` |
| `roles.decideRoute` | anonim → `/user` = `redirect-login` | `lib/roles.test.ts` |
| `roles.decideRoute` | sesi valid → `/login` = redirect ke home per peran (regresi) | `lib/roles.test.ts` |
| `roles.decideRoute` | sesi valid → `/harga` = `pass` | `lib/roles.test.ts` |
| `fetchPublicPlans` | respons OK + `data` array | `lib/plans-server.test.ts` |
| `fetchPublicPlans` | respons non-OK → `[]` | `lib/plans-server.test.ts` |
| `fetchPublicPlans` | `data` bukan array → `[]` | `lib/plans-server.test.ts` |
| `enforcedLimits` | menyembunyikan `max_categories` & `max_wallets` | `lib/plans-server.test.ts` |
| `FEATURES` | 6 item, slug unik, semua slug valid kebab-case | `lib/marketing/features.test.ts` |
| `FEATURES` | setiap item punya problem, howItWorks ≥3, benefits ≥3, relatedSlugs valid | `lib/marketing/features.test.ts` |
| `FEATURES` | `enabledInPlans` konsisten (mis. `scan-struk` tanpa `free`) | `lib/marketing/features.test.ts` |
| `faq` | ≥ 4 kategori, tiap item punya `q` & `a` non-kosong | `lib/marketing/faq.test.ts` |
| `legal` | punya `effectiveDate` + seluruh bagian wajib | `lib/marketing/legal.test.ts` |
| `site` | `url` valid, `email`/`whatsapp` terisi | `lib/marketing/site.test.ts` |
| `lib/seo` | builder `Organization`/`WebSite`/`Product`/`BreadcrumbList` menghasilkan `@type` benar & field wajib | `lib/seo.test.ts` |
| `lib/seo` | `Product.offers` dibangkitkan dari data paket (jumlah sesuai) | `lib/seo.test.ts` |
| `sitemap()` | memuat seluruh rute statis + 6 rute fitur, URL absolut | `app/sitemap.test.ts` |
| `robots()` | `disallow` memuat `/user`, `/admin`, `/super-admin`; memuat `sitemap` | `app/robots.test.ts` |

### 2.2 L2 — DOM Frontend

| Komponen | Kasus |
|----------|-------|
| `MarketingHeader` | Menampilkan nav desktop (Fitur/Harga/FAQ) |
| `MarketingHeader` | Tombol hamburger punya `aria-expanded`; klik membuka panel; `Escape` menutup |
| `PricingCards` | Toggle Bulanan/Tahunan mengubah angka tampil |
| `PricingCards` | Nilai `-1` dirender "Tanpa batas" |
| `PricingCards` | Baris fitur hanya menampilkan fitur `true` |
| `ComparisonTable` | Baris `max_categories`/`max_wallets` tidak ada |
| `ComparisonTable` | Sel "tidak tersedia" memakai label aksesibel, bukan hanya warna |
| `FaqAccordion` | Klik `<summary>` membuka/menutup; konten ada di DOM meski tertutup |
| `PricingTeaser` | Data kosong → menampilkan tombol `Lihat Harga`, bukan grid kosong |

### 2.3 L3 — A11y

| Target | Kasus |
|--------|-------|
| `MarketingHeader` | Lulus `axe`; toggle punya nama aksesibel |
| `MarketingFooter` | Semua tautan punya teks bermakna; lulus `axe` |
| `PricingCards` | `aria-pressed` pada toggle; `aria-live` pada harga |
| `ComparisonTable` | Tabel punya `<caption>`/`<th scope>`; lulus `axe` |
| `FaqAccordion` | `<summary>` dapat difokuskan; lulus `axe` |
| Halaman `/`, `/harga`, `/faq` | Tidak ada pelanggaran `axe` serius |

### 2.4 L4 — Backend (Go)

| Kasus | Harapan |
|-------|---------|
| `GET /api/v1/public/plans` tanpa cookie | `200`, `success: true`, `data` array |
| `GET /api/v1/plans` tanpa cookie | `401` (regresi proteksi) |
| Paket `is_public = false` | tidak muncul |
| Paket `is_active = false` | tidak muncul |
| Katalog `nil` | `200`, `data: []` |
| Urutan hasil | sesuai `sort_order` |

### 2.5 L5 — Manual / E2E

Checklist smoke (tanpa login):

- [ ] `/` merender; tombol `Lihat Cara Kerja` menggulir ke `#cara-kerja`; `Mulai Gratis` → `/register`.
- [ ] Nav: Fitur → `/fitur`; Harga → `/harga`; FAQ → `/faq`; menu mobile berfungsi.
- [ ] `/harga`: kartu & tabel terisi; toggle siklus bekerja; baris tak enforced absen.
- [ ] `/fitur` → 6 kartu; tiap sub-halaman terbuka; slug ngawur → 404.
- [ ] `/tentang`, `/kontak`, `/faq`, `/privasi`, `/syarat` terbuka; tautan `mailto:`/`wa.me` benar.
- [ ] Tidak ada tautan `#` hampa di HTML.
- [ ] View Source: `<link rel="canonical">`, `og:*`, `application/ld+json` ada.
- [ ] `/robots.txt` & `/sitemap.xml` benar.
- [ ] `/user` tanpa login → diarahkan ke `/login` (regresi).
- [ ] Tidak ada emoji di UI.
- [ ] Kontras teks di atas permukaan gelap memadai.

---

## 3. Perintah Uji Kanonik

```bash
# Backend (Go)
cd backend && go vet ./... && go test ./... && go build ./...

# Frontend — tes
cd frontend && pnpm test

# Frontend — satu berkas / satu tes
cd frontend && pnpm test -- lib/plans-server.test.ts
cd frontend && pnpm test -- -t "nama test"

# Frontend — lint & build (wajib hijau)
cd frontend && pnpm lint
cd frontend && pnpm build
```

---

## 4. Kriteria Penerimaan (Definition of Done)

### 4.1 Backend

- [ ] Rute `GET /api/v1/public/plans` melayani anonim (200) dan memakai ulang catalog service.
- [ ] `/api/v1/plans` **tetap** 401 untuk anonim.
- [ ] Rate limit eksplisit terpasang untuk rute publik.
- [ ] `go vet ./... && go test ./... && go build ./...` hijau.

### 4.2 Frontend — Fungsional

- [ ] Guard auth membuka seluruh prefix marketing untuk anonim; dashboard tetap terlindungi.
- [ ] Landing dirender ulang; tidak ada anchor mati; `Lihat Cara Kerja` berfungsi.
- [ ] `/harga` menampilkan paket dari endpoint; `-1` → "Tanpa batas"; pajak tampil bila `tax_enabled`.
- [ ] `/fitur` + 6 sub-halaman punya konten unik; slug invalid → 404.
- [ ] Halaman pendukung selesai; tautan kontak berfungsi.
- [ ] Footer tidak punya tautan mati.

### 4.3 Frontend — SEO

- [ ] `metadataBase`, template judul, OG/Twitter konsisten.
- [ ] `canonical` unik per halaman.
- [ ] `sitemap.ts` & `robots.ts` valid.
- [ ] JSON-LD (`Organization`, `WebSite`, `Product`+`Offer`, `BreadcrumbList`, `AboutPage`, `ContactPage`) terpasang.
- [ ] Tidak memakai schema `Review`/`AggregateRating`/`FAQPage`.

### 4.4 Kualitas

- [ ] `pnpm lint` 0 error.
- [ ] `pnpm test` hijau (unit + DOM + a11y).
- [ ] `pnpm build` hijau.
- [ ] Tidak ada emoji di UI.
- [ ] Semua ikon dari `lucide-react`.
- [ ] `log_aktivitas.md` & HUD roadmap diperbarui.

---

## 5. Aturan Verifikasi (Anti-Klaim Palsu)

- Tidak menyatakan "selesai" sebelum menjalankan perintah kanonik dan melihat hasilnya hijau.
- Bila DB/Redis tidak tersedia, tes integrasi backend akan **skip** — catat itu sebagai "skip", bukan "lulus".
- Jangan mengklaim nilai SEO (peringkat/volume) tanpa data pihak pertama.
