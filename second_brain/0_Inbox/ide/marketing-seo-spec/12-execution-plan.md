# 12 — Rencana Eksekusi Granular

Dokumen ini adalah versi **siap-eksekusi** dari [`08-implementation-plan.md`](./08-implementation-plan.md): setiap tugas diberi berkas persis, perubahan yang diharapkan, perintah verifikasi, dan bukti selesai. Berkas ini menjadi acuan kerja implementasi Fase 1.

> Berkas ini tidak mengubah keputusan desain. Bila terjadi pertentangan, dokumen 02–07 dan **13** yang berlaku.

---

## 1. Aturan Eksekusi

1. **Urutan wajib:** Fase 0 → Fase 1 → (Fase 2/3/4/5 boleh paralel) → Fase 6 → Fase 7.
2. **TDD:** untuk logika murni (`roles`, `plans-server`, `seo`, data konten), tulis tes lebih dulu lalu implementasi.
3. **Verifikasi per tugas:** jalankan perintah verifikasi yang tercantum; jangan menandai selesai tanpa bukti.
4. **Hijau antar-fase:** setiap fase harus meninggalkan `pnpm build`, `pnpm lint`, `pnpm test` (dan `go test` bila menyentuh backend) dalam keadaan hijau.
5. **Zona repo:** hanya `frontend/`, `backend/`, `database/` untuk kode produksi. Tidak menambah skrip debug/scratch ke sana (`CLAUDE.md`).
6. **Tanpa emoji:** seluruh UI memakai ikon `lucide-react`.
7. **Komit kecil & bertema:** satu komit per fase (lihat §7).

---

## 2. Fase 0 — Fondasi Routing & Data (KRITIS, kerjakan lebih dulu)

### 0.1 — Perbaiki `decideRoute` (allowlist prefix publik)

**Berkas:** `frontend/lib/roles.ts`

**Perubahan:**
- Ganti baris `const publicPath = pathname === '/login' || pathname === '/' || pathname === '/register';` dengan pemanggilan helper.
- Tambah (di atas `decideRoute`):

```ts
const PUBLIC_PREFIXES = [
  '/pricing', '/features', '/about', '/contact', '/faq', '/privacy', '/terms',
] as const;

/** True bila pathname boleh diakses tanpa sesi (permukaan marketing publik). */
export function isPublicPath(pathname: string): boolean {
  if (pathname === '/' || pathname === '/login' || pathname === '/register') return true;
  return PUBLIC_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}
```

- Di `decideRoute`: `const publicPath = isPublicPath(pathname);`
- Pertahankan seluruh perilaku lain (`/login` bounce, `/admin`, `/user`, `/super-admin`).

**Verifikasi:**

```bash
cd frontend && pnpm test -- lib/roles.test.ts
```

**Bukti:** seluruh kasus lama tetap lulus + kasus baru di 0.2 lulus.

---

### 0.2 — Tes guard auth

**Berkas:** `frontend/lib/roles.test.ts`

**Perubahan:** tambah entri pada array `cases` di `describe("decideRoute")` (pola yang sudah ada, baris 73–199):

| name | pathname | hasSession | hasRefresh | expected |
|------|----------|-----------|-----------|----------|
| anon on /pricing -> pass | `/pricing` | false | — | `"pass"` |
| anon on /features -> pass | `/features` | false | — | `"pass"` |
| anon on /features/receipt-scan -> pass | `/features/receipt-scan` | false | — | `"pass"` |
| anon on /about -> pass | `/about` | false | — | `"pass"` |
| anon on /contact -> pass | `/contact` | false | — | `"pass"` |
| anon on /faq -> pass | `/faq` | false | — | `"pass"` |
| anon on /privacy -> pass | `/privacy` | false | — | `"pass"` |
| anon on /terms -> pass | `/terms` | false | — | `"pass"` |
| anon on /pricingx -> redirect-login (segmentasi) | `/pricingx` | false | — | `"redirect-login"` |
| anon on /featuresx -> redirect-login | `/featuresx` | false | — | `"redirect-login"` |
| authenticated user on /pricing -> pass | `/pricing` | true, valid | — | `"pass"` |

**Verifikasi:** `cd frontend && pnpm test -- lib/roles.test.ts`

**Bukti:** semua kasus lulus; tidak ada kasus lama yang berubah hasil.

---

### 0.3 — Rute publik katalog (backend)

**Berkas:** `backend/internal/delivery/http/router.go`

**Perubahan:** di dalam blok grup publik `api` (sekitar baris 256–271), tambahkan `api.GET("/public/plans", paymentH.ListPublicPlans)` di dalam `if paymentH != nil` yang sama dengan webhook.

**Verifikasi:**

```bash
cd backend && go vet ./... && go build ./...
```

**Bukti:** build hijau; rute terdaftar (uji di 0.5).

---

### 0.4 — Rate limit eksplisit rute publik

**Berkas:** `backend/internal/delivery/http/router.go`

**Perubahan:** tambah entri pada `rateLimitMap`:

```go
"/api/v1/public/plans": {Period: 1 * time.Minute, Limit: 60},
```

**Verifikasi:** `cd backend && go vet ./... && go build ./...`

---

### 0.5 — Tes rute publik (backend)

**Berkas:** berkas tes router/handler yang ada (cari pola di `backend/`), atau tes baru mengikuti konvensi `backend/tests/`.

**Kasus:** lihat dokumen 09 §2.4 (anonim 200 & data array; `/plans` tetap 401; `is_public=false` tak muncul).

**Verifikasi:** `cd backend && go test ./...`

**Bukti:** tes lulus atau tercatat **skip** bila butuh DB yang tidak tersedia (jangan klaim lulus bila skip).

---

### 0.6 — Klien server `plans-server.ts`

**Berkas (baru):** `frontend/lib/plans-server.ts`

**Perubahan:** implementasi `fetchPublicPlans()` dan `enforcedLimits()` seperti dokumen 06 §5, dengan `revalidate: 3600` dan penanganan kegagalan → `[]`.

**Verifikasi:** `cd frontend && pnpm build`

**Bukti:** berkas terkompilasi; tidak diimpor dari komponen client.

---

### 0.7 — Tes `plans-server`

**Berkas (baru):** `frontend/lib/plans-server.test.ts` (project `unit`)

**Kasus:** OK+array; non-OK → `[]`; `data` bukan array → `[]`; `enforcedLimits` menyembunyikan `max_categories`/`max_wallets`.

**Verifikasi:** `cd frontend && pnpm test -- lib/plans-server.test.ts`

---

### 0.8 — Satukan tipe `PublicPlan`

**Berkas:** `frontend/lib/plans-types.ts` (baru), `frontend/hooks/use-billing.ts` (ubah impor)

**Perubahan:** pindahkan `PublicPlan`/`PlanLimitMeta` ke `plans-types.ts`; impor dari sana di `use-billing.ts` dan `plans-server.ts`. Perubahan tipe murni, tanpa ubah perilaku.

**Verifikasi:** `cd frontend && pnpm build && pnpm test`

**Bukti:** `use-billing.test.ts` tetap lulus.

**Kriteria Fase 0:** anonim dapat "melewati" guard untuk seluruh prefix marketing; endpoint publik melayani; klien data siap; semua tes hijau.

---

## 3. Fase 1 — Struktur Marketing & Data Konten

| ID | Aksi | Berkas | Verifikasi |
|----|------|--------|-----------|
| 1.1 | Layout marketing (header+footer, metadata default) | `app/(marketing)/layout.tsx` | `pnpm build` |
| 1.2 | Data situs (brand, url, email, whatsapp) | `lib/marketing/site.ts` | `pnpm test -- lib/marketing/site.test.ts` |
| 1.3 | Data 6 fitur + isi sub-halaman | `lib/marketing/features.ts` | `pnpm test -- lib/marketing/features.test.ts` |
| 1.4 | Data FAQ berkelompok | `lib/marketing/faq.ts` | `pnpm test -- lib/marketing/faq.test.ts` |
| 1.5 | Data legal (privasi & syarat) | `lib/marketing/legal.ts` | `pnpm test -- lib/marketing/legal.test.ts` |
| 1.6 | Naskah copy landing & CTA | `lib/marketing/copy.ts` | `pnpm test -- lib/marketing/copy.test.ts` |
| 1.7 | Helper metadata + JSON-LD builder | `lib/seo.ts` | `pnpm test -- lib/seo.test.ts` |
| 1.8 | `JsonLd`, `Breadcrumbs`, `SectionHeading` | `components/marketing/*` | `pnpm build` |
| 1.9 | `MarketingHeader` (client) + `MarketingFooter` | `components/marketing/*` | `pnpm test -- components/marketing/MarketingHeader.dom.test.tsx` |
| 1.10 | Tes data konten (fitur/FAQ/copy) | `lib/marketing/*.test.ts` | `pnpm test` |
| 1.11 | Token visual neobrutalism (aksen kuning asam/merah, token motion, kelas `.neo-*`) + font display heading | `frontend/app/globals.css`, `frontend/app/layout.tsx` | `pnpm build`; `--font-display` terhubung ke `--font-heading` |

**Kriteria:** header/footer merender; data lengkap & tervalidasi; tidak ada `href="#"`.

---

## 4. Fase 2 — Landing Page

| ID | Aksi | Berkas | Verifikasi |
|----|------|--------|-----------|
| 2.1 | Pindahkan landing ke group marketing; `git rm frontend/app/page.tsx` | `app/(marketing)/page.tsx` | `pnpm build` |
| 2.2 | `Hero` + `InteractiveDemo` (tab Chat/Suara/Struk, mock data; copy dari `lib/marketing/copy.ts`) | `components/marketing/Hero.tsx`, `InteractiveDemo.tsx` | DOM test |
| 2.3 | Elemen khas: `ReceiptTapeMarquee`, `TickerBand`, `StatCounters`, `Reveal` (hormat `prefers-reduced-motion`) | `components/marketing/*` | DOM + a11y test |
| 2.4 | `TrustBar` (4 klaim faktual) | `components/marketing/TrustBar.tsx` | DOM test |
| 2.5 | `ProblemSolution` + `FeatureGrid` | `components/marketing/ProblemSolution.tsx`, `FeatureGrid.tsx` | DOM test |
| 2.6 | `HowItWorks` (3 langkah baru) | `components/marketing/HowItWorks.tsx` | DOM test |
| 2.7 | `SpkHighlight` | `components/marketing/SpkHighlight.tsx` | DOM test |
| 2.8 | `PricingTeaser` (props dari `fetchPublicPlans`) | `components/marketing/PricingTeaser.tsx` | DOM test (data kosong → tombol) |
| 2.9 | `PersonaUseCases` (pengganti testimoni) | `components/marketing/PersonaUseCases.tsx` | DOM test |
| 2.10 | `FaqAccordion` (`<details>`) + `CtaSection` | `components/marketing/*` | DOM + a11y test |
| 2.11 | Rakit landing + anchor `#pricing`/`#how-it-works` + metadata | `app/(marketing)/page.tsx` | `pnpm build`; cek anchor |
| 2.12 | DOM + a11y test landing | `components/marketing/*.dom.test.tsx`, `*.a11y.test.tsx` | `pnpm test` |

**Assertion wajib:** tidak ada string "API Key"/"API key" pada konten landing; tidak ada nama "Budi Santoso"/"Siti Aminah"/"Andi Wijaya"; tidak ada "ribuan pengguna"; marquee/ticker ber-`aria-hidden` dan berhenti pada `prefers-reduced-motion`; aksen kuning asam tidak pernah dipasangkan dengan teks putih.

**Kriteria:** tidak ada anchor mati; klaim akurat; a11y hijau.

---

## 5. Fase 3–5 — Halaman Publik

### Fase 3 — `/pricing`

| ID | Aksi | Berkas | Verifikasi |
|----|------|--------|-----------|
| 3.1 | `PricingCards` (client, toggle siklus, `-1` → "Tanpa batas") | `components/marketing/PricingCards.tsx` | DOM test |
| 3.2 | `ComparisonTable` (fitur + limit enforced) | `components/marketing/ComparisonTable.tsx` | DOM test; assert `max_categories`/`max_wallets` absen |
| 3.3 | Halaman `/pricing` + FAQ + schema `Product`/`Offer` | `app/(marketing)/pricing/page.tsx` | `pnpm build` |
| 3.4 | Fallback katalog gagal (pesan + `router.refresh()`) | `app/(marketing)/pricing/page.tsx` | DOM test |
| 3.5 | A11y test | `components/marketing/*.a11y.test.tsx` | `pnpm test` |

**Assertion wajib:** teks "Bank" / "bank_sync" tidak muncul; pajak hanya muncul bila `tax_enabled`.

### Fase 4 — `/features` & sub-halaman

| ID | Aksi | Berkas | Verifikasi |
|----|------|--------|-----------|
| 4.1 | Hub `/features` (6 kartu) | `app/(marketing)/features/page.tsx` | `pnpm build` |
| 4.2 | Sub-halaman dinamis + `generateStaticParams` + `dynamicParams=false` | `app/(marketing)/features/[slug]/page.tsx` | `pnpm build`; slug ngawur → 404 |
| 4.3 | Isi 6 sub-halaman (≥300 kata unik) | `lib/marketing/features.ts` | review manual + tes kelengkapan |
| 4.4 | "Fitur terkait" + badge paket | `app/(marketing)/features/[slug]/page.tsx` | DOM test |
| 4.5 | DOM test hub & sub-halaman | `app/(marketing)/features/*.dom.test.tsx` | `pnpm test` |

### Fase 5 — Halaman pendukung

| ID | Aksi | Berkas | Verifikasi |
|----|------|--------|-----------|
| 5.1 | `/about` | `app/(marketing)/about/page.tsx` | `pnpm build` |
| 5.2 | `/contact` (mailto/wa.me dari `site.ts`) | `app/(marketing)/contact/page.tsx` | DOM test (href benar) |
| 5.3 | `/faq` | `app/(marketing)/faq/page.tsx` | DOM test |
| 5.4 | `LegalPage` + `/privacy` + `/terms` | `app/(marketing)/privacy/page.tsx`, `terms/page.tsx` | `pnpm build` |
| 5.5 | A11y test halaman pendukung | `app/(marketing)/**/*.a11y.test.tsx` | `pnpm test` |

---

## 6. Fase 6 — Infrastruktur SEO

| ID | Aksi | Berkas | Verifikasi |
|----|------|--------|-----------|
| 6.1 | `metadataBase` + template judul + OG/Twitter default | `app/layout.tsx` | `pnpm build`; cek canonical absolut |
| 6.2 | `robots.ts` | `app/robots.ts` | `curl /robots.txt` |
| 6.3 | `sitemap.ts` (statis + 6 fitur) | `app/sitemap.ts` | `curl /sitemap.xml`; unit test |
| 6.4 | OG image dinamis | `app/opengraph-image.tsx` | `pnpm build`; cek meta `og:image` |
| 6.5 | JSON-LD `Organization`/`WebSite` di layout marketing | `app/(marketing)/layout.tsx` | View Source |
| 6.6 | Unit test `sitemap`/`robots`/builder JSON-LD | `app/sitemap.test.ts`, `app/robots.test.ts`, `lib/seo.test.ts` | `pnpm test` |
| 6.7 | Aset logo `Organization` | `frontend/public/logo.png` | cek URL 200 |

**Assertion wajib:** tidak ada `FAQPage`, `Review`, atau `AggregateRating` di JSON-LD mana pun.

---

## 7. Fase 7 — Verifikasi & Pembersihan

```bash
cd backend && go vet ./... && go test ./... && go build ./...
cd frontend && pnpm lint
cd frontend && pnpm test
cd frontend && pnpm build
```

Lalu jalankan checklist smoke dokumen 09 §2.5 dan checklist pra-rilis dokumen 10 §7.

Terakhir: perbarui `second_brain/1_Digest/Tugas-Rutin/log_aktivitas.md`.

---

## 8. Cadangan Komit yang Disarankan

| # | Cakupan | Pesan contoh |
|---|---------|--------------|
| 1 | Fase 0 | `fix(auth): izinkan rute marketing publik + endpoint publik paket` |
| 2 | Fase 1 | `feat(marketing): struktur route group, design system neobrutalism, dan data konten` |
| 3 | Fase 2 | `feat(marketing): rombak landing page (SEO, copy akurat, CTA)` |
| 4 | Fase 3 | `feat(marketing): halaman harga + tabel komparasi paket` |
| 5 | Fase 4 | `feat(marketing): halaman fitur + 6 sub-halaman` |
| 6 | Fase 5 | `feat(marketing): halaman tentang, kontak, FAQ, legal` |
| 7 | Fase 6 | `feat(seo): metadata, robots, sitemap, JSON-LD, OG dinamis` |

Catatan: repo saat ini memiliki perubahan tertunda yang tidak terkait Fase 1 (backend/frontend/dump). **Jangan** mencampurnya ke komit Fase 1; stage berkas per-fase secara eksplisit.

---

## 9. Checklist Eksekusi Global

**Fase 0**
- [ ] 0.1 `isPublicPath` + `decideRoute` diperbaiki
- [ ] 0.2 tes guard auth ditambah & lulus
- [ ] 0.3 rute publik `GET /public/plans`
- [ ] 0.4 rate limit eksplisit
- [ ] 0.5 tes rute backend
- [ ] 0.6 `lib/plans-server.ts`
- [ ] 0.7 tes `plans-server`
- [ ] 0.8 tipe `PublicPlan` disatukan

**Fase 1–5**
- [ ] 1.1–1.11 struktur, design system & data konten
- [ ] 2.1–2.12 landing page
- [ ] 3.1–3.5 halaman `/pricing`
- [ ] 4.1–4.5 halaman `/features` + sub-halaman
- [ ] 5.1–5.5 halaman pendukung

**Fase 6–7**
- [ ] 6.1–6.7 infrastruktur SEO
- [ ] 7.x verifikasi kanonik + smoke + log aktivitas

**Definition of Done:** lihat dokumen 09 §4 dan dokumen 10 §7. Jangan menyatakan selesai sebelum seluruh perintah kanonik dijalankan dan hijau.

---

## 10. Guardrails (Ringkas)

- Tidak menyentuh `frontend/proxy.ts`, `next.config.ts`, `app/(auth)/*`, `app/(dashboard)/*`.
- Tidak memakai gradien, blur, atau `shadow-lg`/`shadow-xl`; hanya hard shadow 3/4/6px.
- Tidak memakai emoji; hanya ikon `lucide-react`.
- Sudut tetap membulat (`rounded-md`/`lg`) — tidak dilancipkan.
- Tidak mengubah `/api/v1/plans` yang terproteksi.
- Tidak menambah dependensi runtime baru.
- Tidak memakai `FAQPage`/`Review`/`AggregateRating`.
- Tidak menampilkan `bank_sync` atau limit tak enforced.
- Tidak menampilkan email/WhatsApp placeholder di produksi (konfirmasi sebelum rilis).
