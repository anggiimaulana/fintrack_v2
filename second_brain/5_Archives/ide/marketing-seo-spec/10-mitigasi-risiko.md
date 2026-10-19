# 10 — Mitigasi Risiko

Dokumen ini memetakan risiko teknis, keamanan, SEO, konten, dan operasional untuk Fase 1, beserta strategi mitigasi dan rencana rollback.

---

## 1. Matriks Risiko

| ID | Area | Deskripsi Risiko | Keparahan | Probabilitas | Mitigasi |
|----|------|------------------|-----------|--------------|----------|
| **R-01** | Routing/Keamanan | Membuka prefix publik di `decideRoute` membuat rute privat ikut terbuka (mis. prefix `/features` ikut meloloskan `/features-internal`). | Tinggi | Sedang | Bandingkan sebagai segmen penuh (`=== p` atau `startsWith(p + '/')`), bukan `startsWith(p)`. Tes eksplisit `/pricingx`, `/featuresx`. |
| **R-02** | Keamanan | Endpoint publik `GET /public/plans` disalahgunakan (scraping/DoS). | Sedang | Sedang | Rate limit eksplisit per menit; hanya mengembalikan data publik; ISR frontend menekan frekuensi panggilan; tidak ada operasi tulis. |
| **R-03** | Data/Konsistensi | Harga di halaman publik berbeda dari harga checkout (karena ISR basi). | Sedang | Sedang | Checkout selalu memakai DB (otoritatif). Tampilkan disclaimer kecil; `revalidate: 3600` dapat diturunkan bila perlu. Signal "harga dapat berubah" di halaman harga. |
| **R-04** | SEO | Konten sub-halaman fitur menjadi tipis/duplikat satu sama lain. | Sedang | Sedang | Wajib konten unik ≥300 kata; larangan menyalin; tes memverifikasi kelengkapan data; review manual sebelum rilis. |
| **R-05** | SEO | Cannibalization antar halaman (mis. `/` vs `/features` vs `/pricing` bersaing kata kunci sama). | Sedang | Sedang | Tetapkan intent: `/` brand+umum, `/features/*` fitur spesifik, `/pricing` komersial. `title`/`H1` berbeda; canonical tepat. |
| **R-06** | SEO | Klaim palsu / schema rating palsu → penalti. | Tinggi | Rendah | Testimoni fiktif dihapus; tidak memakai `Review`/`AggregateRating`; copy diverifikasi terhadap `catalog.go` & seed. |
| **R-07** | SEO | Halaman legal di-`noindex` tanpa sengaja. | Sedang | Rendah | Tetapkan `robots: { index: true }`; jangan masukkan ke `disallow`. |
| **R-08** | Konten/Hukum | Kebijakan Privasi/Syarat terbit tanpa tinjauan dan memuat klaim kepatuhan berlebihan. | Tinggi | Sedang | Tandai isi sebagai draf internal di `legal.ts`; pemilik produk wajib meninjau; hindari klaim sertifikasi; cantumkan tanggal berlaku. |
| **R-09** | Konten | Email/WhatsApp placeholder terbit ke produksi. | Sedang | Sedang | `site.ts` memuat placeholder berkomentar; checklist rilis wajib mengonfirmasi sebelum deploy. |
| **R-10** | Aksesibilitas | Menu mobile / tabel / akordeon gagal aksesibilitas. | Sedang | Sedang | Tes `axe`; `<details>` native; `aria-expanded`/`aria-controls`; uji keyboard manual. |
| **R-11** | Performa | JS klien membengkak karena komponen baru. | Rendah | Sedang | Server Component default; `"use client"` hanya di daun; tanpa dependensi baru; tanpa animasi berat. |
| **R-12** | Build/Deploy | `BACKEND_ORIGIN` salah di produksi → harga kosong. | Sedang | Sedang | Fallback UI ramah; fallback localhost untuk dev; dokumentasikan env; smoke test pasca-deploy. |
| **R-13** | Regresi | Memindahkan `app/page.tsx` memutus tautan yang mengarah ke `/`. | Sedang | Rendah | Route group tidak mengubah URL `/`; jalankan smoke test seluruh rute; cek redirect lama di `next.config.ts` tetap utuh. |
| **R-14** | Data | Domain/URL dasar (`NEXT_PUBLIC_SITE_URL`) belum diset → canonical relatif/kosong. | Sedang | Sedang | `metadataBase` + fallback `http://localhost:3001`; set env di produksi sebelum rilis. |
| **R-15** | Kompatibilitas | Angka pajak salah tampil (mis. menampilkan "+0%" atau mencampur pajak ke harga). | Sedang | Sedang | Tampilkan baris pajak hanya bila `tax_enabled`; putuskan sekali apakah harga termasuk pajak; uji dengan data tanpa & dengan pajak. |
| **R-16** | Aksesibilitas/Fungsional | Akordeon berbasis JS gagal tanpa JS. | Rendah | Sedang | Gunakan `<details>`/`<summary>` native; konten ada di HTML server. |
| **R-17** | SEO/GEO | Memblokir crawler AI secara tak sengaja. | Rendah | Rendah | `robots.ts` hanya memblokir rute privat; jangan blokir GPTBot/PerplexityBot pada publik. |
| **R-18** | Operasional | Perubahan guard auth mengganggu alur login/redirect pengguna lama. | Sedang | Rendah | Tes regresi perilaku `/login`, `/user`, `/admin`, `/super-admin`; logika tetap terpusat di `roles.ts`. |
| **R-19** | Aksesibilitas | Animasi baru (receipt tape, ticker band, stat counter) memicu ketidaknyamanan atau tidak berhenti saat pengguna memilih reduced-motion. | Sedang | Rendah | Hentikan total pada `prefers-reduced-motion`; jeda saat hover/focus; `aria-hidden` pada track yang bergerak; sediakan versi statis; maksimum 2 elemen beranimasi per viewport. |
| **R-20** | Performa/Tipografi | Font display baru menambah bobot unduhan atau menggeser tata letak (CLS). | Rendah | Sedang | `next/font` (self-hosted, subset `latin`); fallback metrik; H1 hero sebagai elemen LCP tetap teks; uji CLS < 0.1. |

---

## 2. Keamanan

### 2.1 Permukaan Serangan

| Permukaan | Risiko | Kontrol |
|-----------|--------|---------|
| `GET /public/plans` | Scraping, DoS | Rate limit; data publik saja; tanpa tulis; tanpa data pengguna |
| Rute marketing anonim | Eksposur tidak sengaja | Hanya konten statis + data publik; tidak ada query ber-scope pengguna |
| Guard auth | Bypass ke dashboard | Prefix dicocokkan per-segmen; tes regresi rute privat |

### 2.2 Prinsip

- **Least privilege:** rute publik hanya membaca katalog publik.
- **No new secrets:** tidak ada kredensial baru; `NEXT_PUBLIC_SITE_URL` tidak sensitif.
- **Fail-safe:** bila data gagal dimuat, halaman tetap tampil tanpa data sensitif.
- **Tanpa markup berbahaya:** JSON-LD dirender dengan `JSON.stringify` (meloloskan karakter), bukan injeksi HTML mentah.

### 2.3 Dilarang

- Membuka `/api/v1/plans` (versi terproteksi) ke publik.
- Mencatat atau menampilkan data pengguna mana pun di halaman publik.
- Menambahkan endpoint publik yang memicu operasi tulis.

---

## 3. SEO & Konten

| Risiko | Mitigasi |
|--------|----------|
| Thin/duplicate content | Konten unik per sub-halaman; larangan salin-tempel; review manual |
| Keyword cannibalization | Intent per halaman; title/H1 berbeda; canonical tepat |
| Klaim tidak akurat | Semua klaim fitur diverifikasi ke `backend/internal/entitlement/catalog.go` dan seed |
| Markup tidak sesuai | Tanpa `FAQPage`/`Review`; hanya schema yang benar-benar cocok |
| Halaman legal tak terindeks | `index: true`; tidak di-`disallow` |
| Fokus berlebihan pada keyword | Prioritaskan kejelasan & kebergunaan (pedoman helpful content) |

---

## 4. Deployment & Zero Downtime

| Langkah | Detail |
|---------|--------|
| Perubahan backend | Murni penambahan rute (tidak mengubah yang ada) → kompatibel mundur, tanpa migrasi. |
| Perubahan frontend | Penambahan rute + pemindahan `/` dalam route group (URL sama). |
| Urutan deploy | Deploy **backend lebih dulu** (agar `/public/plans` sudah ada), lalu frontend. |
| Kompatibilitas | Frontend lama tetap berfungsi bila backend baru lebih dulu (tidak ada rute yang dihapus). |
| Env baru | `NEXT_PUBLIC_SITE_URL` (frontend). `BACKEND_ORIGIN` sudah ada. |
| Verifikasi pasca-deploy | `curl /api/v1/public/plans`; buka `/pricing`; cek `robots.txt`/`sitemap.xml`; cek canonical di HTML. |

### 4.1 Rollback

- **Frontend:** revert deploy; route group tidak mengubah URL, sehingga tidak ada redirect yang perlu dibersihkan. `app/page.tsx` lama dapat dikembalikan bila perlu.
- **Backend:** rute publik baru bersifat aditif; menghapusnya tidak merusak apa pun (perilaku kembali ke baseline).
- **Data:** tidak ada migrasi → tidak ada rollback DB.
- Pemicu rollback: harga salah tampil masif, halaman publik 5xx, atau regresi aksesibilitas serius.

---

## 5. Observabilitas

| Sinyal | Sumber | Aksi |
|--------|--------|------|
| 5xx pada `/public/plans` | Log backend / metrik | Investasi kegagalan katalog |
| Halaman `/pricing` kosong | Laporan pengguna / smoke test | Cek `BACKEND_ORIGIN` & ISR |
| Lonjakan trafik endpoint publik | Rate-limit hit | Evaluasi batas |
| Penurunan peringkat/trafik | Search Console | Periksa cannibalization/markup |

---

## 6. Risiko yang Diterima (Accepted)

| Risiko | Alasan penerimaan |
|--------|-------------------|
| Harga di halaman publik dapat basi hingga 1 jam | Checkout otoritatif; dampak hanya tampilan |
| Tanpa form kontak tersimpan | Keputusan produk; mailto/wa.me mencukupi untuk Fase 1 |
| Tanpa blog di Fase 1 | Dijadwalkan sebagai Fase 2 terpisah |
| Legal belum ditinjau ahli | Ditandai draf; pemilik produk meninjau sebelum rilis |
| Tanpa i18n | Situs berbahasa Indonesia |

---

## 7. Checklist Pra-Rilis

- [ ] `NEXT_PUBLIC_SITE_URL` diset ke domain produksi.
- [ ] `BACKEND_ORIGIN` benar di lingkungan produksi.
- [ ] Email & WhatsApp nyata di `site.ts` (bukan placeholder).
- [ ] Aset logo (`public/logo.png`) ada dan `Organization.logo` menunjuk URL valid.
- [ ] Kebijakan Privasi & Syarat ditinjau dan diberi tanggal berlaku.
- [ ] Smoke test seluruh rute publik tanpa login.
- [ ] `robots.txt` & `sitemap.xml` benar.
- [ ] Tidak ada `href="#"` atau anchor mati.
- [ ] Tidak ada klaim BYOK / testimoni fiktif.
- [ ] Animasi berhenti pada `prefers-reduced-motion`; marquee/ticker `aria-hidden` dan punya versi statis.
- [ ] Font display termuat tanpa pergeseran tata letak (CLS < 0.1).
- [ ] Tidak ada gradien/blur/emoji; aksen kuning asam selalu berpasangan teks gelap.
- [ ] `go test ./...`, `pnpm lint`, `pnpm test`, `pnpm build` hijau.
