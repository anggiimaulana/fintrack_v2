# 11 — Roadmap Fase 2: Blog CMS

Dokumen ini adalah **garis besar** untuk Fase 2 (Blog berbasis database + editor admin). Fase 2 **belum** dikerjakan dan akan dibrainstorm ulang (pertanyaan → desain → spec → rencana) saat tiba waktunya. Tujuan dokumen ini hanya menjaga kontinuitas keputusan dan memastikan Fase 1 tidak menutup pintu bagi Fase 2.

---

## 1. Keputusan yang Sudah Ditetapkan

| # | Keputusan | Pilihan |
|---|-----------|---------|
| 1 | Manajemen konten | CMS di database + editor admin (bukan MDX) |
| 2 | Di mana editor | Panel admin (perlu dikonfirmasi: `admin` atau `super_admin`) |
| 3 | Bahasa | Indonesia |
| 4 | Rendering publik | Server Component + ISR; konten dari DB |
| 5 | Pemisahan fase | Fase 2 terpisah dari marketing/SEO Fase 1 |

## 2. Ruang Lingkup yang Diusulkan (untuk dibrainstorm)

### 2.1 Backend (Go)

- **Migrasi baru bernomor** (mis. `database/migrations/00NN_blog.sql`) dengan tabel:
  - `blog_posts` — `id`, `slug` (unik), `title`, `excerpt`, `body` (format perlu diputuskan: markdown/HTML/JSON blok), `cover_image_url`, `status` (draft/published/archived), `author_id`, `published_at`, `created_at`, `updated_at`, `seo_title`, `seo_description`.
  - `blog_tags` / `blog_post_tags` (opsional) untuk kategori/tag.
  - (Opsional) `blog_revisions` untuk riwayat.
- **API:**
  - Publik: `GET /api/v1/public/blog/posts` (daftar, paginasi), `GET /api/v1/public/blog/posts/:slug`.
  - Admin: `POST/PUT/DELETE /api/v1/admin/blog/posts` (di balik RBAC).
- Validasi: slug unik, sanitasi HTML bila body HTML, rate limit endpoint publik.
- Regenerasi `database/dump/fintrack_v2.sql` setelah migrasi (aturan repo).

### 2.2 Frontend (Next.js)

- Halaman publik: `/blog` (daftar + paginasi), `/blog/[slug]` (artikel).
- Rendering: Server Component + ISR; markdown dirender aman (repo sudah punya `react-markdown` + `remark-gfm`).
- SEO: `BlogPosting`/`Article` JSON-LD, `BreadcrumbList`, canonical, OG per artikel, penulis (`Person` schema) untuk E-E-A-T.
- Tambahkan rute blog ke `app/sitemap.ts` (sudah dirancang untuk diperluas di Fase 1).
- `robots.ts`: pastikan `/blog` publik dapat diindeks.
- Editor admin: form tulis (judul, slug, excerpt, body, cover, status, SEO), pratinjau, simpan draf/publish.
- Internal linking: artikel menautkan ke `/fitur/*` dan `/harga`.

### 2.3 E-E-A-T

- Penulis harus nyata (bio/`Person` schema); hindari artikel tanpa atribusi.
- Tanggal terbit & tanggal pembaruan terlihat.
- Referensi sumber bila mengutip data.

## 3. Pertanyaan Terbuka untuk Brainstorm Fase 2

1. Siapa yang boleh menulis: `admin`, `super_admin`, atau keduanya?
2. Format body: Markdown, HTML tersanitasi, atau editor blok (JSON)?
3. Perlu gambar cover? Bila ya: penyimpanan berkas (mengikuti pola `ProofStore`/avatar)?
4. Perlu kategori/tag, atau cukup daftar kronologis?
5. Perlu penjadwalan terbit (`published_at` di masa depan)?
6. Perlu komentar pengunjung? (Rekomendasi: tidak, untuk menghindari moderasi/spam.)
7. Perlu pencarian di `/blog`?
8. Kebijakan SEO: `noindex` saat draf; canonical saat artikel dipindah.

## 4. Keterkaitan dengan Fase 1

- Fase 1 **tidak** membuat `/blog`, tetapi:
  - Footer sudah menyediakan slot "Blog" (ditandai "Segera hadir" sampai Fase 2).
  - `sitemap.ts` sudah disusun agar penambahan rute blog mudah.
  - Struktur `app/(marketing)/` sudah siap menampung `blog/`.
  - Data yang dapat dipakai ulang (site, SEO helpers, komponen marketing) sudah ada.
- Fase 1 **tidak** menambahkan dependensi yang menghalangi Fase 2.

## 5. Prasyarat Fase 2

- Fase 1 selesai dan hijau (build/lint/test).
- Kebutuhan penulisan & alur editorial disepakati pemilik produk.
- Keputusan format body & penyimpanan gambar final.
- Migrasi DB + regenerasi dump mengikuti prosedur repo (`database/README.md`).

## 6. Estimasi Kasar

| Bagian | Kompleksitas |
|--------|--------------|
| Migrasi + repository | Sedang |
| API publik & admin | Sedang |
| Halaman publik `/blog` & `/blog/[slug]` | Sedang |
| Editor admin | Sedang–Tinggi |
| SEO artikel + schema | Rendah |
| Tes (unit/DOM/a11y/backend) | Sedang |

> Fase 2 adalah subsistem mandiri: ia layak mendapat spec dan rencana implementasi sendiri, bukan digabung ke Fase 1.
