# 00 — Ringkasan Ide

- **Tanggal:** 2026-10-11
- **Status:** Ide mentah → sudah dianalisis, menunggu BRD/PRD final
- **Penulis:** Analisis brainstorming bersama pemilik produk (FinTrack v2)
- **Konteks repo:** `D:/project/finance/fintrack_v2` (Go + PostgreSQL + Next.js, P01 selesai 100%)

## Ringkasan satu paragraf

FinTrack v2 saat ini adalah aplikasi keuangan pribadi yang sudah berjalan (auth Go-native, RAG pgvector,
dashboard user/admin, AI chat berbasis API key **per-user**). Pemilik produk ingin mengubahnya menjadi
**produk berlangganan (SaaS)**: kunci AI dipusatkan di super admin lewat gateway **9router**, ditambah
**sistem plan/subscription dengan kuota token dinamis**, **pembayaran Midtrans + manual** yang bisa
di-switch realtime, **dashboard & manajemen super admin**, **RBAC berbasis entitlement plan**, serta
**standar data-table, caching realtime, dan zero-downtime** yang berlaku di semua halaman.

## Lima pilar permintaan

1. **API key terpusat** — hapus model per-user; semua trafik AI lewat pool kunci 9router yang dikelola
   super admin (URL & key dinamis; dapat memakai kuota/key/combo spesifik; dapat dipisah untuk embedding,
   vision, dan kebutuhan lain).
2. **Subscription & sistem token** — paket/plan dinamis, tiap plan punya maksimum token, kondisi
   upgrade/downgrade, siklus per-hari, reset kuota, serta harga per 1.000 token yang dapat di-markup
   (kali & persen) sepenuhnya dari super admin.
3. **Dashboard & manajemen super admin** — belum ada; perlu dibangun lengkap (user, plan, subscription,
   pembayaran, token, gateway, audit, pengaturan).
4. **RBAC** — hak akses fitur/limit mengikuti plan (free/upgrade/downgrade), diatur dengan "centang"
   secara realtime, cepat, zero downtime, tanpa reload/refresh, dan tanpa login/logout.
5. **Standar wajib semua halaman & role** — pagination 20/halaman, sorting, searching, filtering,
   penomoran, semuanya dengan parameter URL; caching realtime (fetch hanya data yang berubah); TanStack
   Query; chart interaktif; cursor pointer; data realtime & zero downtime.

## Keputusan kunci (hasil dialog)

| # | Topik | Keputusan |
|---|---|---|
| 1 | Tenancy | **Per-user (B2C)** — plan menempel ke user, tanpa `tenant_id` |
| 2 | Role | **Hanya 2: `super_admin` + `user`** — admin lama dipromosikan jadi super_admin |
| 3 | Cakupan token | **Semua fitur AI** (chat, vision, voice, embedding), harga dinamis (base + markup kali/persen) |
| 4 | Siklus | **Per-hari** (bukan bulan kalender), reset pada tanggal langganan, TZ Asia/Jakarta |
| 5 | Renewal | **Hybrid** — auto jika metode mendukung, manual + reminder jika tidak |
| 6 | Upgrade/downgrade | **Upgrade instan prorated**, **downgrade akhir siklus** |
| 7 | Switch gateway | **Global default + override per-plan**, berlaku untuk checkout baru |
| 8 | Invoice | **Dasar, siap PPN** |
| 9 | Entitlement | **Fitur (bool) + limit numerik**, diatur super admin |
| 10 | Penegakan | **Backend otoritatif**, frontend hanya cermin UX |
| 11 | Realtime RBAC | **Lookup per-request + cache berversi** (bukan di JWT) |
| 12 | Efek ubah plan | **Efektif langsung**, tanpa migrasi data user |
| 13 | BYOK | **Dihapus total**, semua terpusat |
| 14 | Topologi 9router | **Pool multi-endpoint + failover + circuit breaker** |
| 15 | Routing | **Tabel route per-use-case**; embedding dikunci 768d |
| 16 | Sumber token | **Usage provider + fallback estimasi** |
| 17 | Transport realtime | **Surgical cache update + SSE** untuk perubahan eksternal |
| 18 | Infra | **Tambah Redis** (cache + pub/sub) |
| 19 | DataTable | **TanStack Table v8 + state di URL** |
| 20 | Zero downtime | **Migrasi expand/contract + rolling deploy + feature flag** |
| 21 | Pendekatan | **Hibrida A+B** — incremental (Strangler Fig) + modul baru bersih ber-interface |
| 22 | Fase | **7 fase P02–P08** |
| 23 | Struktur | **4_Resources/Dokumen-Spesifikasi/{BRD,PRD,Diagram}**, 1 folder ide, 3_Areas +2 area baru |

## Kenapa ini penting

- **Monetisasi**: FinTrack berubah dari aplikasi gratis/self-host menjadi produk berlangganan.
- **Kontrol biaya**: kunci AI terpusat + metering token = biaya LLM terkendali & terukur.
- **Skalabilitas**: Redis + zero-downtime + feature flag menyiapkan jalur produksi multi-instance.
- **Kepercayaan**: audit log, invoice, dan verifikasi webhook menjadikannya layak untuk pembayaran nyata.

## Dokumen terkait di folder ini

- `01-analisis-mendalam.md` — analisis teknis & bisnis menyeluruh
- `02-kritik-saran-best-practice.md` — kritik jujur + rujukan praktik industri
- `03-keputusan-terkunci.md` — daftar keputusan lengkap dengan alasan
- `04-pertanyaan-terbuka.md` — daftar pertanyaan asli (sudah dijawab di 06)
- `05-peta-jalan-awal.md` — pembagian fase P02–P08
- `06-jawaban-rekomendasi-pertanyaan-terbuka.md` — jawaban + rekomendasi + alasan + rujukan
- `07-konfigurasi-seed-awal.md` — nilai konkret siap seed (plan, limit, pricing, settings, gateway)

## Rujukan

- Brief & keputusan migrasi P01: `../../brief_awal.md`
- Roadmap P01: `../../../2_Projects/P01_FinTrack_v2/roadmap.md`
