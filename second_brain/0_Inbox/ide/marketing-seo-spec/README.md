# Master Spec: Situs Marketing & SEO FinTrack v2 — Fase 1

Paket spesifikasi komprehensif untuk merombak **landing page** dan membangun **situs marketing publik** FinTrack v2 agar: SEO kuat, mudah dipahami dan digunakan, menjual, dan mendorong pendaftaran/langganan.

Fase 1 berfokus pada **permukaan marketing publik + infrastruktur SEO**. **Fase 2** (Blog CMS berbasis database + editor admin) dirinci terpisah di [`11-roadmap-fase2-blog-cms.md`](./11-roadmap-fase2-blog-cms.md) dan dibrainstorm ulang saat akan dikerjakan.

Tanggal spec: **2026-10-17**
Status: **Disetujui untuk implementasi (menunggu rencana eksekusi)**

---

## 1. Masalah yang Diselesaikan

Baseline `frontend/app/page.tsx` adalah landing satu-file tanpa struktur, dengan cacat yang langsung merugikan konversi, kepercayaan, dan SEO:

1. **Navigasi mati.** Nav menautkan `#harga` padahal tidak ada section `id="harga"`. Footer menautkan `#tentang`, `#kontak`, `#privasi` yang juga tidak ada.
2. **Tombol "Lihat Demo" tanpa `href`.** Tombol utama non-fungsional.
3. **Klaim tidak akurat (melanggar trust / E-E-A-T).** Step "Tambahkan API Key" dan fitur "AI key milik Anda sendiri" — padahal BYOK sudah dihapus total dari produk (`CLAUDE.md`, migrasi `0014`).
4. **Testimoni fiktif.** Nama "Budi Santoso / Siti Aminah / Andi Wijaya" sama dengan akun seed demo; klaim "ribuan pengguna" tidak terverifikasi.
5. **Nol infrastruktur SEO.** Tidak ada `robots.ts`, `sitemap.ts`, `metadataBase`, canonical, Open Graph, JSON-LD, atau OG image.
6. **Halaman publik akan terblokir.** `lib/roles.ts:51` hanya mengenali `/`, `/login`, `/register` sebagai publik; rute marketing baru akan dipantulkan ke `/login` bagi pengunjung anonim.
7. **Tidak ada halaman harga/komparasi.** Pengunjung tidak punya tempat untuk memahami dan membandingkan paket sebelum mendaftar.

## 2. Keputusan Desain Kunci

| # | Keputusan | Pilihan |
|---|-----------|---------|
| 1 | Manajemen konten blog | **CMS di database + editor admin** (Fase 2) |
| 2 | Pembagian kerja | **2 fase**: Fase 1 marketing + SEO, Fase 2 Blog CMS |
| 3 | Halaman Fase 1 | `/harga`, `/tentang`, `/kontak`, `/fitur` (+ sub-halaman), `/privasi`, `/syarat`, `/faq` |
| 4 | Form kontak | **Tanpa backend**: `mailto:` + `wa.me` |
| 5 | Sumber data harga | **Endpoint publik baru** di backend (bukan hardcode) |
| 6 | Tombol "Lihat Demo" | **Anchor** ke section produk (`#cara-kerja`) |
| 7 | Domain/URL dasar | `NEXT_PUBLIC_SITE_URL` + fallback `http://localhost:3001` |
| 8 | Testimoni fiktif | **Diganti** menjadi bagian "Cocok untuk siapa" (persona use-case) |
| 9 | Pendekatan struktur | **Route group `app/(marketing)/`** + layout bersama |
| 10 | Rendering harga | Server Component + ISR (`revalidate: 3600`) |
| 11 | FAQ rich result | **Tidak** memakai schema `FAQPage` (Google tak lagi menampilkannya); konten Q&A tetap ada untuk GEO |
| 12 | Bahasa situs | Bahasa Indonesia (`lang="id"` sudah diset di root layout) |

## 3. Daftar Dokumen

| # | Dokumen | Isi |
|---|---------|-----|
| 01 | [Audit Kondisi Awal](01-audit-kondisi-awal.md) | Temuan baseline dengan bukti `file:line`, celah SEO, cacat konversi, dan masalah auth guard. |
| 02 | [Arsitektur Informasi & Routing](02-arsitektur-informasi-routing.md) | Peta situs, hierarki URL, struktur route group, perbaikan `decideRoute`, nav/footer, strategi internal linking. |
| 03 | [Desain Landing Page](03-desain-landing-page.md) | Susunan section baru, copy final, kata kunci, alur konversi, koreksi klaim, penanganan aksesibilitas. |
| 04 | [Halaman Harga & Fitur](04-halaman-publik-harga-fitur.md) | `/harga` (kartu paket, toggle bulanan/tahunan, tabel komparasi), `/fitur` + 6 sub-halaman (konten, schema, metadata). |
| 05 | [Halaman Pendukung](05-halaman-publik-pendukung.md) | `/tentang`, `/kontak`, `/faq`, `/privasi`, `/syarat` — konten, struktur, schema, metadata. |
| 06 | [Backend: Endpoint Publik Paket](06-backend-endpoint-publik-paket.md) | Rute publik `GET /api/v1/public/plans` (reuse catalog), kontrak respons, klien server ISR, penanganan kegagalan, tes. |
| 07 | [SEO Teknis & Schema](07-seo-teknis-metadata-schema.md) | Metadata, canonical, Open Graph/Twitter, OG image dinamis, `robots.ts`, `sitemap.ts`, JSON-LD, Core Web Vitals, kesiapan GEO/AI. |
| 08 | [Rencana Implementasi](08-implementation-plan.md) | 7 fase eksekusi, inventaris berkas dibuat/diubah, matriks dependensi, kriteria penerimaan per fase. |
| 09 | [Strategi Pengujian & QA](09-strategi-pengujian-qa.md) | Matriks uji L1–L5 (unit, DOM, a11y, backend, manual), perintah kanonik, Definition of Done. |
| 10 | [Mitigasi Risiko](10-mitigasi-risiko.md) | Matriks risiko teknis, keamanan/SEO-black-hat, zero-downtime, rollback, observabilitas. |
| 11 | [Roadmap Fase 2: Blog CMS](11-roadmap-fase2-blog-cms.md) | Garis besar subsistem blog (skema DB, API, editor admin, rendering publik) untuk brainstorm terpisah. |

## 4. Ruang Lingkup

**Termasuk (Fase 1):**

- Perbaikan `lib/roles.ts` agar rute marketing publik dapat diakses tanpa login (+ pembaruan tes).
- Route group `app/(marketing)/` dengan layout, header, dan footer bersama.
- Landing page (`/`) dirombak total: struktur, copy akurat, CTA, aksesibilitas.
- Halaman publik: `/harga`, `/fitur`, `/fitur/[slug]`, `/tentang`, `/kontak`, `/faq`, `/privasi`, `/syarat`.
- Endpoint backend publik `GET /api/v1/public/plans` + klien server ISR.
- Infrastruktur SEO: metadata, canonical, OG/Twitter, OG image dinamis, `robots.ts`, `sitemap.ts`, JSON-LD.
- Tes unit/DOM/a11y backend & frontend.

**Tidak termasuk (Fase 1):**

- Blog publik dan CMS blog (Fase 2).
- Form kontak dengan penyimpanan pesan di backend.
- Multi-bahasa (i18n) — situs berbahasa Indonesia.
- Perubahan alur checkout, harga, atau entitlement.
- Aset merek baru (logo desain) — memakai ikon `lucide-react` yang sudah dipakai.

## 5. Prinsip Rekayasa (Non-Negotiable)

1. **Akurasi di atas persuasi.** Tidak ada klaim yang tidak bisa dibuktikan. BYOK, "ribuan pengguna", dan testimoni fiktif dihapus/diganti.
2. **Server-first.** Halaman marketing adalah Server Component; JS klien hanya untuk interaksi (menu mobile, akordeon, toggle harga).
3. **Anggaran JS minimal.** Tidak menambah dependensi runtime baru (tanpa `@next/mdx`, tanpa library UI baru).
4. **Data nyata, satu sumber.** Harga & fitur berasal dari katalog backend; frontend tidak menyimpan salinan harga.
5. **Aksesibilitas wajib.** Semua komponen baru lulus `axe-core`; navigasi keyboard penuh.
6. **Gratis emoji.** Nol karakter emotikon di UI; 100% ikon `lucide-react`.
7. **Konsisten dengan desain yang ada.** Neo-brutalist: `border-2`, `shadow-[Npx_Npx_0px_0px]`, token warna dari `globals.css`.
8. **Verifikasi jujur.** `pnpm build`, `pnpm lint`, `pnpm test`, dan `go test ./...` harus hijau sebelum klaim selesai.

## 6. Referensi Kode & Data (Sumber Kebenaran)

| Aspek | Sumber |
|-------|--------|
| Harga & fitur paket | `database/seeds/0002_subscription_token_seeds.sql` |
| Label fitur resmi | `backend/internal/entitlement/catalog.go:29-36` |
| Label limit + `enforced` | `backend/internal/entitlement/catalog.go:41-49` |
| Katalog paket (service) | `backend/internal/payment/catalog.go:66-160` |
| Handler katalog | `backend/internal/delivery/http/handler/payment_user_handler.go:48-64` |
| Registrasi rute | `backend/internal/delivery/http/router.go:256-379` |
| Guard auth edge | `frontend/proxy.ts`, `frontend/lib/roles.ts` |
| Landing saat ini | `frontend/app/page.tsx` |
| Root layout & metadata | `frontend/app/layout.tsx` |
| Format Rupiah | `frontend/lib/format.ts:12-16` |
| Token desain & kelas neo | `frontend/app/globals.css` |
| Komponen tombol | `frontend/components/ui/button.tsx` |
| Panduan repo (zona, perintah) | `CLAUDE.md` |

## 7. Ringkasan Kriteria Keberhasilan

| Metrik | Baseline | Target setelah Fase 1 |
|--------|----------|------------------------|
| Halaman publik terindeks | 1 (`/`) | 13+ (`/`, `/harga`, `/fitur`, 6 sub-fitur, `/tentang`, `/kontak`, `/faq`, `/privasi`, `/syarat`) |
| Link navigasi mati | 5 (`#harga`, `#tentang`, `#kontak`, `#privasi`, tombol demo) | 0 |
| Klaim produk tidak akurat | 2 (BYOK/API key) | 0 |
| Sitemap & robots | Tidak ada | Ada, valid, menunjuk ke seluruh rute publik |
| Structured data | Tidak ada | `Organization`, `WebSite`, `Product`+`Offer`, `BreadcrumbList`, `AboutPage`, `ContactPage` |
| Halaman publik dapat diakses anonim | Hanya `/` | Seluruh rute marketing |
| Aksesibilitas | Tidak teruji untuk marketing | Semua komponen baru lulus `axe-core` |
| Build/lint/test | Hijau (baseline) | Tetap hijau |

> Angka di atas adalah target operasional internal, bukan jaminan peringkat. Volume pencarian dan peringkat tidak dapat dipastikan tanpa data pihak pertama (Search Console) atau DataForSEO; kata kunci di dokumen 03/07 bersifat heuristik.
