# 04 — Halaman Harga & Fitur

Dokumen ini merinci dua permukaan SEO paling penting setelah beranda: `/pricing` (pembalik keberatan + konversi) dan `/features` beserta enam sub-halamannya (penangkapan long-tail intent).

> **Gaya & nada (mengikuti dokumen 03):** visual neobrutalism — border 3px, hard shadow 6px, sudut **membulat**, aksen **kuning asam** untuk CTA/highlight, **merah** hanya untuk masalah/peringatan. Copy **kalem & profesional**: kalimat jelas, tanpa hiperbola, tanpa klaim yang tidak dapat dibuktikan. Semua kartu, tabel, dan badge memakai token dari `globals.css`.

---

## Bagian A — Halaman `/pricing`

### A.1 Tujuan

- Membuat pengunjung memahami nilai tiap paket dan memilih dengan percaya diri.
- Menjawab keberatan harga secara langsung (FAQ harga).
- Mengarahkan ke `/register` (untuk Free) atau alur berlangganan (setelah daftar).

### A.2 Sumber Data

- **Server-side** memuat `/api/v1/public/plans` (dokumen 06) via `lib/plans-server.ts` dengan ISR `revalidate: 3600`.
- Bentuk data mengikuti `PublicPlan` (`backend/internal/payment/catalog.go:66-95`): `code, name, description, price_monthly_idr, price_yearly_idr, billing_period_days, sort_order, features, limits, limit_meta, cycles, tax_enabled, tax_label, tax_rate_percent, tax_idr, total_idr`.
- **Urutan** paket mengikuti `sort_order` (Free → Pro → Business).
- **Kebijakan pajak (eksplisit):** harga yang ditampilkan di kartu adalah **subtotal (belum termasuk pajak)**. Pajak ditampilkan sebagai **baris terpisah** dan total = subtotal + pajak — konsisten dengan makna `subtotal_idr`/`tax_idr`/`total_idr` di backend (`catalog.go:145-156`, `subtotal = PriceMonthlyIDR`). Tampilkan baris pajak hanya bila `tax_enabled === true` (mis. "PPN 11%: Rp X · Total: Rp Y"). Jangan pernah menampilkan "+ 0%" atau mencampur pajak ke dalam harga kartu.

### A.3 Struktur Halaman

| # | Blok | Isi |
|---|------|-----|
| 1 | Hero ringkas | H1 "Harga FinTrack", subhead "Mulai gratis, naikkan paket saat butuh lebih banyak kuota AI." |
| 2 | Toggle siklus | Bulanan / Tahunan (mengubah angka tampil; tahunan memakai `price_yearly_idr`) |
| 3 | Kartu paket | 3 kartu (lihat A.4) |
| 4 | Tabel komparasi fitur | Fitur × 3 paket (lihat A.5) |
| 5 | Tabel komparasi limit | Limit yang **enforced** × 3 paket (lihat A.6) |
| 6 | Cara pembayaran | Midtrans (otomatis) & transfer manual (verifikasi admin) — jujur soal alur verifikasi |
| 7 | FAQ harga | 6–8 Q&A (lihat A.7) |
| 8 | CTA | "Mulai Gratis" / "Pilih Paket" → `/register` |
| 9 | Footer | `MarketingFooter` |

### A.4 Kartu Paket (`PricingCards`, komponen client)

Untuk setiap paket:

- Nama paket + `description`.
- Harga besar (`formatRupiah`) untuk siklus terpilih; jika tahunan, tampilkan harga tahunan **dan** harga efektif per bulan (harga tahunan ÷ 12) untuk kejujuran.
- Label **"Paling Populer"** hanya pada paket Pro (disengaja secara produk; bukan klaim statistik).
- Daftar fitur unggulan (maks 4 poin, hanya fitur yang `features[key] === true`):
  - Free: Asisten AI (chat), catat via chat.
  - Pro: + Pindai struk (OCR), Input suara, Pencarian cerdas (RAG), Anggaran lanjutan, Ekspor laporan.
  - Business: + Akses API.
- CTA: `Mulai Gratis` (Free) / `Pilih {Nama}` → `/register?plan={code}` (parameter boleh diabaikan bila belum didukung; lihat catatan A.9).
- Aksesibilitas: toggle memakai `role="group"` + dua `<button>` dengan `aria-pressed`; harga memakai `aria-live="polite"` agar pembaca layar mengumumkan perubahan.

### A.5 Tabel Komparasi Fitur

Baris = fitur dari katalog resmi (`backend/internal/entitlement/catalog.go:29-36`). Kolom = Free / Pro / Business. Sel bertanda centang (ikon `Check`) atau strip (ikon `Minus`), dengan `aria-label` deskriptif ("tersedia"/"tidak tersedia").

| Fitur (label resmi) | Free | Pro | Business |
|---------------------|:----:|:---:|:--------:|
| Asisten AI (chat) | ✅ | ✅ | ✅ |
| Pindai struk (OCR) | — | ✅ | ✅ |
| Input suara | — | ✅ | ✅ |
| Pencarian cerdas (RAG) | — | ✅ | ✅ |
| Anggaran lanjutan | — | ✅ | ✅ |
| Ekspor laporan | — | ✅ | ✅ |
| Akses API | — | — | ✅ |

**Dilarang menampilkan** `bank_sync` (katalog menyatakan "belum tersedia"). Setiap baris fitur menautkan ke sub-halaman `/features/{slug}` yang relevan (internal linking, dokumen 02 §6).

### A.6 Tabel Komparasi Limit

Hanya limit dengan `limit_meta[key].enforced !== false` (mengikuti `frontend/lib/plan-limits.ts:25`):

| Limit (label resmi + satuan) | Free | Pro | Business |
|------------------------------|:----:|:---:|:--------:|
| Token AI / siklus | 5.000 token | 150.000 token | 1.000.000 token |
| Permintaan AI / siklus | 50 | 2.000 | Tanpa batas |
| Transaksi | 100 | 5.000 | Tanpa batas |
| Pindai struk / siklus | — | 100 | Tanpa batas |
| Permintaan suara / hari | 1 | 50 | Tanpa batas |

Aturan tampilan:

- Nilai `-1` dirender **"Tanpa batas"**, bukan "-1".
- Label & satuan diambil dari `limit_meta[key]`, bukan kamus lokal — agar tetap sinkron bila admin mengubahnya.
- `max_categories` dan `max_wallets` **tidak ditampilkan** karena tidak di-enforce backend.
- Angka di atas adalah nilai seed saat ini; halaman selalu mengikuti data backend.

### A.7 FAQ Harga (8 item, ringkas)

1. Apakah ada paket gratis? — Ya, Free selamanya untuk mulai.
2. Apa bedanya Pro dan Business? — kuota AI lebih besar + akses API (Business).
3. Bagaimana penagihannya? — per siklus (billing_period_days), pajak bila berlaku.
4. Metode pembayaran? — Midtrans dan transfer manual (verifikasi admin).
5. Bisa beralih paket? — ya; upgrade/downgrade terjadwal (lihat perilaku backend: downgrade berlaku di akhir periode).
6. Bisa berhenti langganan? — ya, akses berlanjut sampai akhir periode yang sudah dibayar.
7. Bagaimana kalau kuota habis? — jelaskan gambaran; tidak menjanjikan yang tidak ada.
8. Apakah ada biaya tersembunyi? — tidak; pajak ditampilkan transparan di checkout.

> Jawaban detail finansial mengikuti perilaku nyata backend (downgrade di akhir periode, verifikasi manual). Jangan menjanjikan fitur yang tidak ada.

### A.8 Metadata & Schema

- `title`: "Harga FinTrack — Paket Gratis, Pro, dan Business"
- `description`: "Bandingkan paket FinTrack: kuota AI, fitur OCR struk, input suara, dan laporan. Mulai gratis, upgrade kapan saja."
- `canonical`: `/pricing`
- JSON-LD: `Product` dengan `offers` (satu `Offer` per paket, `price` dan `priceCurrency: "IDR"`) + `BreadcrumbList` (Beranda › Harga). Lihat dokumen 07.

### A.9 Catatan Teknis

- Toggle siklus hanya mengubah **tampilan** harga; tidak ada transaksi di halaman ini.
- `?plan=` pada tautan CTA bersifat opsional. Bila halaman `/register` belum membaca parameter ini, tautan tetap berfungsi (parameter diabaikan). Tidak ada perubahan alur auth di Fase 1.
- Fallback kegagalan data: bila katalog gagal dimuat, tampilkan pesan ramah + tombol "Coba lagi" (memanggil ulang rute Next via `router.refresh()`), dan **tetap** render FAQ + CTA agar halaman tidak kosong.

---

## Bagian B — Halaman `/features` & Sub-halaman

### B.1 Strategi Konten

- `/features` adalah **hub** dengan ringkasan 6 fitur (kartu) + tautan ke sub-halaman.
- Tiap sub-halaman adalah **halaman mendalam** (≥ 300 kata unik) untuk menangkap pencarian spesifik.
- Melarang konten tipis/duplikat: setiap sub-halaman punya masalah, alur, contoh nyata, dan CTA berbeda.
- Data fitur terpusat di `lib/marketing/features.ts` (satu sumber untuk hub, footer, landing, tabel harga, dan sitemap).

### B.2 Model Data `lib/marketing/features.ts`

```ts
export interface FeaturePage {
  slug: string;              // "receipt-scan"
  icon: LucideIcon;
  title: string;             // "Foto Struk"
  navLabel: string;          // untuk tabel harga & footer
  tagline: string;           // satu kalimat
  keywords: string[];
  problem: string;           // masalah yang dipecahkan
  howItWorks: string[];      // langkah-langkah
  benefits: string[];        // 3-4 poin
  example: { prompt: string; result: string }; // contoh interaksi
  relatedSlugs: string[];    // 2 tautan silang
  enabledInPlans: Array<"free" | "pro" | "business">; // untuk badge
}

export const FEATURES: FeaturePage[] = [ /* 6 item */ ];
```

### B.3 Daftar Sub-halaman

| Slug | Judul | Keyword target | Fitur katalog terkait |
|------|-------|----------------|-----------------------|
| `ai-chat` | Catat & Tanya Lewat Chat | catat pengeluaran via chat, asisten keuangan AI | `ai_chat` |
| `voice-input` | Catat dengan Suara | catat transaksi suara, input suara Indonesia | `voice_input` |
| `receipt-scan` | Pindai Struk Otomatis | scan struk otomatis, OCR struk belanja | `ocr_receipt` |
| `dashboard` | Dasbor Keuangan Interaktif | dashboard keuangan, grafik pengeluaran | — |
| `financial-health` | Analisis Kesehatan Finansial | analisis keuangan pribadi, skor keuangan | — |
| `reports` | Laporan & Ekspor Data | laporan keuangan otomatis, ekspor data keuangan | `reports_export` |

### B.4 Struktur Tiap Sub-halaman

1. **Breadcrumb** (Beranda › Fitur › {Judul}) + JSON-LD `BreadcrumbList`.
2. **Hero fitur:** `H1` = judul; tagline; CTA `Mulai Gratis` + `Lihat Harga`.
3. **Masalah** — satu paragraf tentang pain.
4. **Cara kerjanya** — daftar langkah bernomor.
5. **Manfaat** — 3–4 poin (memakai label katalog resmi bila menyebut fitur).
6. **Contoh nyata** — blok contoh percakapan/hasil (bukan testimoni).
7. **Ketersediaan paket** — badge "Tersedia di: Pro, Business" berdasarkan `enabledInPlans`.
8. **Fitur terkait** — 2 tautan silang.
9. **CTA penutup.**

### B.5 Contoh Konten (pola yang harus diikuti)

`/features/receipt-scan`:

- Keywords: scan struk otomatis, OCR struk belanja, catat belanja dari foto.
- Problem: "Mengetik ulang deretan belanjaan dari struk itu melelahkan dan rawan salah."
- How it works: (1) Buka menu AI dan pilih kamera; (2) Ambil/foto struk yang jelas; (3) AI membaca item & total; (4) Periksa lalu simpan.
- Manfaat: hemat waktu, akurasi angka, kategori otomatis.
- Contoh: prompt "foto struk ini" → hasil "Belanja bulanan Rp 312.500 tercatat, 8 item".
- `enabledInPlans: ["pro", "business"]` (katalog: `ocr_receipt` Free = false).

`/features/voice-input`:

- `enabledInPlans: ["pro", "business"]` (`voice_input` Free = false).
- Sebut batas wajar memakai label resmi "Permintaan suara / hari".

`/features/ai-chat`:

- `enabledInPlans: ["free", "pro", "business"]` (`ai_chat` semua plan).
- Tegaskan **tidak perlu API key** (koreksi F-04).

### B.6 Metadata & Schema per Sub-halaman

- `title`: "{{Judul}} — FinTrack"
- `description`: ringkas, mengandung keyword target.
- `canonical`: `/features/{slug}`
- `generateStaticParams` dari `FEATURES`; `dynamicParams = false` agar slug tak dikenal → 404.
- JSON-LD: `BreadcrumbList` + `WebPage` (dengan `about`). Tidak memakai `FAQPage`.

### B.7 Hub `/features`

- H1: "Fitur FinTrack".
- Grid kartu 6 fitur (ikon, judul, tagline, tautan "Pelajari lebih lanjut").
- Section "Bagaimana semuanya terhubung" singkat.
- CTA → `/register` dan `/pricing`.
- JSON-LD `BreadcrumbList` (Beranda › Fitur).
