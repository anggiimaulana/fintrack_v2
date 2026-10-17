# 01 — Audit Kondisi Awal (Baseline)

Dokumen ini merekam kondisi nyata landing page FinTrack v2 sebelum perombakan, beserta bukti `file:line`, agar setiap perubahan dapat divertifikasi terhadap baseline. Semua temuan di bawah berasal dari pembacaan kode langsung, bukan asumsi.

---

## 1. Inventaris Berkas Relevan

| Berkas | Peran | Ukuran |
|--------|-------|--------|
| `frontend/app/page.tsx` | Landing page tunggal | ±258 baris, satu komponen |
| `frontend/app/layout.tsx` | Root layout, font, metadata global | 53 baris |
| `frontend/app/globals.css` | Token warna, kelas neo-brutalist | 267 baris |
| `frontend/lib/roles.ts` | Keputusan routing (dipakai edge guard + tes) | 77 baris |
| `frontend/proxy.ts` | Edge auth guard (pengganti `middleware.ts`) | 59 baris |
| `frontend/components/ui/button.tsx` | Tombol neo-brutalist | 60 baris |
| `frontend/lib/format.ts` | `formatRupiah`, `formatTanggal` | 24 baris |

Tidak ditemukan berkas: `app/sitemap.ts`, `app/robots.ts`, `app/opengraph-image.*`, `app/(marketing)/*`, `components/marketing/*`, `lib/marketing/*`.

---

## 2. Temuan Cacat (dengan Bukti)

### F-01 — Navigasi ke section yang tidak ada (dead anchor)

`frontend/app/page.tsx:22`

```tsx
<a href="#harga" ...>Harga</a>
```

Tidak ada elemen dengan `id="harga"` di seluruh berkas. Section yang ada hanya `id="fitur"` (baris 101) dan `id="cara-kerja"` (baris 128). **Dampak:** pengunjung mengklik "Harga" dan tidak terjadi apa-apa; kepercayaan turun; sinyal UX buruk.

### F-02 — Footer dengan tiga dead anchor

`frontend/app/page.tsx:246-249`

```tsx
<a href="#fitur">Fitur</a>
<a href="#tentang">Tentang</a>
<a href="#kontak">Kontak</a>
<a href="#privasi">Kebijakan Privasi</a>
```

`#tentang`, `#kontak`, `#privasi` tidak memiliki target. Tidak ada halaman `/tentang`, `/kontak`, `/privasi`. **Dampak:** hilangnya sinyal trust (halaman legal & kontak adalah faktor E-E-A-T), dan pengalaman rusak.

### F-03 — Tombol "Lihat Demo" non-fungsional

`frontend/app/page.tsx:58-60`

```tsx
<Button size="lg" variant="outline" ...>
  Lihat Demo
</Button>
```

Tidak memakai `asChild` maupun `onClick`, sehingga benar-benar tidak melakukan apa pun. **Dampak:** CTA sekunder utama hilang.

### F-04 — Klaim BYOK/API key yang sudah tidak berlaku

`frontend/app/page.tsx:136`

```tsx
{ step: "1", title: "Tambahkan API Key", desc: "Gunakan API key dari berbagai provider AI untuk privasi penuh dan akses tak terbatas." },
```

`frontend/app/page.tsx:114`

```tsx
{ icon: Shield, title: "Privasi Total", desc: "Data tersimpan aman. AI key milik Anda sendiri, kami tidak membaca data Anda." },
```

Per `CLAUDE.md`: "Jalur BYOK per-user sudah **dihapus total** (tabel `gemini_api_keys` di-drop oleh migrasi `0014` …)". Gateway AI terpusat adalah satu-satunya transport. **Dampak:** klaim menyesatkan; merusak trust dan akurasi konten (faktor helpful-content).

### F-05 — Testimoni fiktif + klaim tak terverifikasi

`frontend/app/page.tsx:196-199` menampilkan tiga testimoni bernama **Budi Santoso, Siti Aminah, Andi Wijaya** — persis nama akun seed demo (`second_brain/0_Inbox/2026-10-14_kredensial-login-demo.md`: Budi Santoso, Siti Rahayu, Andi Wijaya). Baris 226-228 mengklaim: "Bergabunglah dengan **ribuan pengguna lain** …". **Dampak:** sinyal Experience/Trustworthiness negatif; risiko dianggap konten tidak membantu.

### F-06 — Nol infrastruktur SEO

- Tidak ada `metadataBase`, `canonical`, `openGraph`, atau `twitter` di `frontend/app/layout.tsx:17-20` (hanya `title` dan `description`).
- Metadata login generik: `frontend/app/(auth)/layout.tsx:4` → `"Authentication | FinTrack"` untuk semua rute auth.
- Tidak ada `app/sitemap.ts` / `app/robots.ts`.
- Tidak ada JSON-LD.
- Tidak ada OG image (tidak ada aset di `frontend/public/` selain SVG template bawaan Next).
- Landing tidak punya H1 keyword yang menargetkan pencarian ("Kelola Keuangan Cerdas dengan Perintah Natural" — bagus untuk brand, lemah untuk intent pencarian).

### F-07 — Rute publik baru akan terblokir (KRITIS)

`frontend/lib/roles.ts:51`

```ts
const publicPath = pathname === '/login' || pathname === '/' || pathname === '/register';
```

`frontend/lib/roles.ts:53-56`

```ts
if (!hasSession) {
  if (publicPath) return 'pass';
  return hasRefresh ? 'pass' : 'redirect-login';
}
```

**Dampak:** pengunjung anonim yang membuka `/harga`, `/fitur`, `/tentang`, `/kontak`, `/faq`, `/privasi`, atau `/syarat` akan **dipantulkan ke `/login`**. Ini mematikan seluruh tujuan Fase 1. Wajib diperbaiki sebelum halaman apa pun dibuat.

### F-08 — Menu mobile tidak berfungsi

`frontend/app/page.tsx:32-36` — tombol hamburger tanpa state, tanpa `aria-expanded`, tanpa panel menu. Navigasi di layar kecil hanya menyisakan logo dan tanpa akses ke Fitur/Harga. **Dampak:** UX mobile rusak, aksesibilitas gagal.

### F-09 — Klaim privasi vs fitur nyata tidak sinkron

Aplikasi sebenarnya memiliki fitur ekspor data (`GET /api/v1/user/export-data`) dan hapus akun (`DELETE /api/v1/me`) sebagai wujud kepatuhan UU PDP (`CLAUDE.md`). Fitur-fitur ini **tidak** disebut di landing, padahal merupakan bukti trust yang kuat. Sebaliknya, yang tampil justru klaim BYOK yang salah (F-04).

### F-10 — CTA dan bagian konversi minim

Landing hanya punya 3 titik CTA (`Mulai Gratis` di hero, `Daftar Sekarang` di penutup, serta satu CTA non-fungsional). Tidak ada: bukti sosial yang jujur, bagian FAQ pembalik keberatan (objection handling), gambaran harga, maupun penjelasan "untuk siapa produk ini".

---

## 3. Peta Bagian Landing Saat Ini

| # | Section | Baris | Masalah |
|---|---------|-------|---------|
| 1 | Header + nav | 11-38 | Nav `#harga` mati; menu mobile mati |
| 2 | Hero | 41-99 | H1 bukan keyword pencarian; mockup chat OK; CTA demo mati |
| 3 | Fitur Unggulan | 101-126 | Dua kartu memuat klaim BYOK salah (F-04) |
| 4 | Cara Kerja | 128-150 | Step 1 = "Tambahkan API Key" (salah) |
| 5 | SPK highlight | 152-190 | Akurat; dapat dipertahankan |
| 6 | Testimoni | 192-221 | Fiktif (F-05) |
| 7 | CTA penutup | 223-233 | Klaim "ribuan pengguna" (F-05) |
| 8 | Footer | 236-255 | 3 dead anchor (F-02) |

---

## 4. Aset Data Paket (Sumber Kebenaran)

Dari `database/seeds/0002_subscription_token_seeds.sql`:

| Field | Free | Pro | Business |
|-------|------|-----|----------|
| `code` | `free` | `pro` | `business` |
| `price_monthly_idr` | 0 | 39.000 | 129.000 |
| `price_yearly_idr` | 0 | 390.000 | 1.290.000 |
| `billing_period_days` | 30 | 30 | 30 |
| `fallback_gateway` | `manual` | `midtrans` | `midtrans` |
| `sort_order` | 1 | 2 | 3 |

Dari `backend/internal/entitlement/catalog.go` (label resmi) dan seed `plan_entitlements` / `plan_limits`:

**Fitur** (`features` map, label resmi):

| Key | Label | Free | Pro | Business |
|-----|-------|------|-----|----------|
| `ai_chat` | Asisten AI (chat) | ✅ | ✅ | ✅ |
| `ocr_receipt` | Pindai struk (OCR) | ❌ | ✅ | ✅ |
| `voice_input` | Input suara | ❌ | ✅ | ✅ |
| `rag_search` | Pencarian cerdas (RAG) | ❌ | ✅ | ✅ |
| `reports_export` | Ekspor laporan | ❌ | ✅ | ✅ |
| `advanced_budgeting` | Anggaran lanjutan | ❌ | ✅ | ✅ |
| `api_access` | Akses API | ❌ | ❌ | ✅ |
| `bank_sync` | Sinkronisasi bank | ❌ | ❌ | ❌ |

> `bank_sync` **tidak boleh** diiklankan: katalog menandainya "belum tersedia".

**Limit** (`limits` + `limit_meta`, dengan flag `enforced`):

| Key | Label resmi | Satuan | Enforced | Free | Pro | Business |
|-----|-------------|--------|----------|------|-----|----------|
| `ai_tokens_per_cycle` | Token AI / siklus | token | ✅ | 5.000 | 150.000 | 1.000.000 |
| `ai_requests_per_cycle` | Permintaan AI / siklus | permintaan | ✅ | 50 | 2.000 | -1 (tanpa batas) |
| `max_transactions` | Transaksi | transaksi | ✅ | 100 | 5.000 | -1 |
| `ocr_scans_per_cycle` | Pindai struk / siklus | scan | ✅ | 0 | 100 | -1 |
| `voice_requests_per_day` | Permintaan suara / hari | permintaan | ✅ | 1 | 50 | -1 |
| `max_categories` | Kategori | kategori | ❌ | 5 | 30 | -1 |
| `max_wallets` | Dompet | dompet | ❌ | 1 | 5 | -1 |

> Aturan tampilan: layar publik **hanya** menampilkan limit dengan `limit_meta[key].enforced !== false` (mengikuti `frontend/lib/plan-limits.ts:25`). Artinya `max_categories` dan `max_wallets` **tidak** diiklankan sebagai benefit.

---

## 5. Endpoint Katalog Saat Ini

- Rute: `userRoutes.GET("/plans", paymentH.ListPublicPlans)` — `backend/internal/delivery/http/router.go:369`.
- `userRoutes` berada di bawah `protected` (butuh JWT) + `RequireUser()`. **Konsekuensi:** pengunjung anonim tidak bisa memuat harga.
- Handler: `payment_user_handler.go:48-64`, membungkus `CatalogService.ListPublicPlans` (`catalog.go:99-160`).
- Service mengembalikan `PublicPlan` dengan `features`, `limits`, `limit_meta`, `cycles`, dan field pajak (`tax_enabled`, `tax_label`, `tax_rate_percent`, `tax_idr`, `total_idr`), plus `price_monthly_idr` / `price_yearly_idr`.
- Middleware `OriginCheck` hanya menjaga metode mutasi (`POST/PUT/PATCH/DELETE`) — lihat `origin_check.go:31-37` — sehingga rute **GET publik aman** dari sisi CSRF.

---

## 6. Ringkasan Defect

| ID | Kategori | Keparahan | Rencana Perbaikan | Dokumen |
|----|----------|-----------|-------------------|---------|
| F-01 | Navigasi | Tinggi | Section `#harga` nyata + tautan ke `/harga` | 02, 03 |
| F-02 | Navigasi/Trust | Tinggi | Halaman nyata + footer fungsional | 02, 05 |
| F-03 | Konversi | Tinggi | Anchor ke `#cara-kerja` | 03 |
| F-04 | Akurasi/Trust | Kritis | Hapus klaim BYOK; ganti pernyataan akurat | 03 |
| F-05 | Trust/E-E-A-T | Tinggi | Ganti testimoni → persona use-case | 03 |
| F-06 | SEO | Tinggi | Metadata, canonical, OG, sitemap, robots, JSON-LD | 07 |
| F-07 | Routing (KRITIS) | Kritis | Perbaiki `decideRoute` jadi allowlist prefix | 02 |
| F-08 | Aksesibilitas/UX | Tinggi | Header/menu mobile responsif | 02, 03 |
| F-09 | Trust | Sedang | Tonjolkan ekspor/hapus data sebagai bukti | 03, 05 |
| F-10 | Konversi | Sedang | Tambah FAQ, teaser harga, persona | 03, 04 |
